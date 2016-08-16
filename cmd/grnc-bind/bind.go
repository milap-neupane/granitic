package main

import (
	"flag"
	"github.com/graniticio/granitic/config"
	"strings"
	"fmt"
	"os"
	"github.com/graniticio/granitic/facility/jsonmerger"
	"github.com/graniticio/granitic/logging"
	"encoding/json"
	"io/ioutil"
	"path"
	"bufio"
)

const (

	packagesField = "packages"
	componentsField = "components"
	templatesField = "templates"
	templateField = "compTemplate"
	templateFieldAlias = "ct"
	typeField = "type"
	typeFieldAlias = "t"

	protoSuffix = "Proto"

	bindingsPackage = "bindings"
	iocImport = "github.com/graniticio/granitic/ioc"
	entryFuncSignature = "func Components() []*ioc.ProtoComponent {"
	protoArrayVar = "pc"
	confLocationFlag string = "c"
	confLocationDefault string = "resource/components"
	confLocationHelp string = "A comma separated list of component definition files or directories containing component definition files"

	bindingsFileFlag string = "o"
	bindingsFileDefault string = "bindings/bindings.go"
	bindingsFileHelp string = "Path to the Go source file that will be generated"

	mergeLocationFlag string = "m"
	mergeLocationDefault string = ""
	mergeLocationHelp string = "The path of a file where the merged component defintion file should be written to. Execution will halt after writing."

	newline = "\n"

	refPrefix      = "ref:"
	refAlias       = "r:"
	confPrefix     = "conf:"
	confAlias      = "c:"

)


func main() {

	var confLocation = flag.String(confLocationFlag, confLocationDefault, confLocationHelp)
	var bindingsFile = flag.String(bindingsFileFlag, bindingsFileDefault, bindingsFileHelp)
	var mergedComponentsFile = flag.String(mergeLocationFlag, mergeLocationDefault, mergeLocationHelp)

	flag.Parse()

	ca := loadConfig(*confLocation)

	if (*mergedComponentsFile != "") {
		writeMergedAndExit(ca, *mergedComponentsFile)
	}

	f := openOutputFile(*bindingsFile)
	defer f.Close()

	w := bufio.NewWriter(f)
	writeBindings(w, ca)
}

func writeBindings(w *bufio.Writer, ca *config.ConfigAccessor) {
	writePackage(w)
	writeImports(w, ca)

	c := ca.ObjectVal(componentsField)
	t := parseTemplates(ca)

	writeEntryFunctionOpen(w, len(c))

	var i = 0

	for name, v := range c {

		writeComponent(w, name, v.((map[string]interface{})), t, i)
		i++
	}

	writeEntryFunctionClose(w)
	w.Flush()
}




func writePackage(w *bufio.Writer) {

	l := fmt.Sprintf("package %s\n\n", bindingsPackage)
	w.WriteString(l)
}


func writeImports(w *bufio.Writer, configAccessor *config.ConfigAccessor) {
	packages := configAccessor.Array(packagesField)

	w.WriteString("import (\n")

	iocImp := tabIndent(quoteString(iocImport), 1)
	w.WriteString(iocImp + newline)

	for _, packageName := range packages {

		i := quoteString(packageName.(string))
		i = tabIndent(i, 1)
		w.WriteString(i + newline)
	}

	w.WriteString(")\n\n")
}

func writeEntryFunctionOpen(w *bufio.Writer, t int) {
	w.WriteString(entryFuncSignature + newline)

	a := fmt.Sprintf("%s := make([]*ioc.ProtoComponent, %d)\n\n", protoArrayVar, t)
	w.WriteString(tabIndent(a, 1))
}

func writeComponent(w *bufio.Writer, name string, component map[string]interface{}, templates map[string]interface{}, index int) {
	baseIdent := 1

	values := make(map[string]interface{})
	refs := make(map[string]interface{})
	confPromises := make(map[string]interface{})

	mergeValueSources(component, templates)
	validateHasTypeField(component, name)

	writeComponentNameComment(w, name, baseIdent)
	writeInstanceVar(w, name, component[typeField].(string), baseIdent)
	writeProto(w, name, index, baseIdent)

	for field, value := range component {

		if isPromise(value) {
			confPromises[field] = value

		} else if isRef(value) {
			refs[field] = value

		} else {
			values[field] = value
		}

	}

	writeValues(w, name, values, baseIdent)
	writeDeferred(w, name, confPromises, baseIdent, "AddConfigPromise")
	writeDeferred(w, name, refs, baseIdent, "AddDependency")

	w.WriteString(newline)
	w.WriteString(newline)

}

func writeValues(w *bufio.Writer, cName string, values map[string]interface{}, tabs int) {

	if len(values) > 0 {
		w.WriteString(newline)
	}


	for k, v := range values {

		if reservedFieldName(k) {
			continue
		}

		s := fmt.Sprintf("%s.%s = %s\n", cName, k, asGoInit(v))
		w.WriteString(tabIndent(s, tabs))
	}

}

func writeDeferred(w *bufio.Writer, cName string, promises map[string]interface{}, tabs int, funcName string) {

	p := protoName(cName)

	if len(promises) > 0 {
		w.WriteString(newline)
	}

	for k, v := range promises {

		fc := strings.SplitN(v.(string), ":", 2)[1]

		s := fmt.Sprintf("%s.%s(%s, %s)\n", p, funcName, quoteString(k), quoteString(fc))
		w.WriteString(tabIndent(s, tabs))

	}

}

func asGoInit(v interface{}) string {
	return ""
}


func writeComponentNameComment(w *bufio.Writer, n string, i int) {
	s := fmt.Sprintf("//%s\n", n)
	w.WriteString(tabIndent(s, i))
}

func writeInstanceVar(w *bufio.Writer, n string, ct string, tabs int) {
	s := fmt.Sprintf("%s := new(%s)\n",n, ct)
	w.WriteString(tabIndent(s, tabs))
}

func writeProto(w *bufio.Writer, n string, index int, tabs int) {

	p := protoName(n)

	s := fmt.Sprintf("%s := ioc.CreateProtoComponent(%s, %s)\n", p, n, quoteString(n))
	w.WriteString(tabIndent(s, tabs))
	s = fmt.Sprintf("%s[%d] = %s\n", protoArrayVar, index, p)
	w.WriteString(tabIndent(s, tabs))
}

func writeEntryFunctionClose(w *bufio.Writer) {
	a := fmt.Sprintf("\treturn %s\n}\n", protoArrayVar)
	w.WriteString(a)
}


func protoName(n string) string{
	return n + protoSuffix
}


func isPromise(v interface{}) bool{

	s, found := v.(string)

	if !found {
		return false
	}

	return strings.HasPrefix(s, confPrefix) || strings.HasPrefix(s, confAlias)
}

func isRef(v interface{}) bool{
	s, found := v.(string)

	if !found {
		return false
	}

	return strings.HasPrefix(s, refPrefix) || strings.HasPrefix(s, refAlias)

}

func reservedFieldName(f string) bool {
	return f == templateField || f == templateFieldAlias || f == typeField || f == typeFieldAlias
}

func validateHasTypeField(v map[string]interface{}, name string) {

	t := v[typeField]

	if t == nil {
		m := fmt.Sprintf("Component %s does not have a 'type' defined in its component defintion (or any parent templates).\n", name)
		fatal(m)
	}

	_, found := t.(string)

	if !found {
		m := fmt.Sprintf("Component %s has a 'type' field defined but the value of the field is not a string.\n", name)
		fatal(m)
	}

}

func mergeValueSources(c map[string]interface{}, t map[string]interface{}){

	replaceAliases(c)


	if c[templateField] != nil {
		flatten(c, t, c[templateField].(string))
	}
}

func quoteString(s string) string{
	return fmt.Sprintf("\"%s\"", s)
}

func tabIndent(s string, t int) string{

	for i := 0; i < t; i++ {
		s = "\t" + s
	}

	return s
}

func writeMergedAndExit(ca *config.ConfigAccessor, f string) {

	b, err := json.MarshalIndent(ca.JsonData, "", "\t")

	if err != nil {
		fatal(err.Error())
	}

	err = ioutil.WriteFile(f, b, 0644)

	if err != nil {
		fatal(err.Error())
	}

	os.Exit(0)
}

func openOutputFile(p string) *os.File {
	os.MkdirAll(path.Dir(p), 0777)
	f, err := os.Create(p)

	if err != nil {
		m := fmt.Sprintf(err.Error() + "\n")
		fatal(m)
	}

	return f
}

func parseTemplates(ca *config.ConfigAccessor) map[string]interface{} {

	flattened := make(map[string]interface{})

	if !ca.PathExists(templatesField) {
		return flattened
	}

	templates := ca.ObjectVal(templatesField)

	for _, template := range templates {
		replaceAliases(template.(map[string]interface{}))
	}


	for n, template := range templates {

		t := template.(map[string]interface{})

		checkForTemplateLoop(t, templates, []string{n})

		ft := make(map[string]interface{})
		flatten(ft, templates, n)

		flattened[n] = ft

	}

	return flattened

}

func replaceAliases(vs map[string]interface{}){
	tma := vs[templateFieldAlias]

	if tma != nil {
		delete(vs, templateFieldAlias)
		vs[templateField] = tma
	}

	tya := vs[typeFieldAlias]

	if tya != nil {

		delete(vs, typeFieldAlias)
		vs[typeField] = tya

	}
}


func flatten(target map[string]interface{}, templates map[string]interface{}, tname string) {

	if templates[tname] == nil{
		fmt.Printf("No template %s\n", tname)
		return
	}

	parent := templates[tname].(map[string]interface{})

	for k, v := range parent {

		if target[k] == nil && k != templateField{
			target[k] = v
		}

	}

	if parent[templateField] != nil {
		flatten(target, templates, parent[templateField].(string))
	}


}

func checkForTemplateLoop(template map[string]interface{}, templates map[string]interface{}, chain []string) {

	if template[templateField] == nil {
		return
	}

	p := template[templateField].(string)

	if contains(chain, p) {
		message := fmt.Sprintf("Invalid template inheritance %v\n", append(chain, p))
		fatal(message)
	}

	if templates[p] ==  nil{
		message := fmt.Sprintf("No template exists with name %s\n", p)
		fatal(message)
	}

	checkForTemplateLoop(templates[p].(map[string]interface{}), templates, append(chain, p))


}

func contains(a []string, c string) bool{
	for _, s := range a {
		if s == c {
			return true
		}
	}

	return false
}


func fatal(m string) {
	fmt.Printf(m)
	os.Exit(-1)
}

func loadConfig(l string) *config.ConfigAccessor{

	s := strings.Split(l, ",")
	fl, err := config.ExpandToFiles(s)

	if err != nil {
		m := fmt.Sprintf("Problem loading config from %s %s", l, err.Error())
		fatal(m)
	}

	jm := new(jsonmerger.JsonMerger)
	jm.Logger = new(logging.ConsoleErrorLogger)

	mc := jm.LoadAndMergeConfig(fl)

	ca := new(config.ConfigAccessor)
	ca.JsonData = mc
	ca.FrameworkLogger = new(logging.ConsoleErrorLogger)

	if !ca.PathExists(packagesField) || !ca.PathExists(componentsField){
		m := fmt.Sprintf("The merged component definition file must contain a %s and a %s section.\n", packagesField, componentsField)
		fatal(m)

	}

	return ca
}