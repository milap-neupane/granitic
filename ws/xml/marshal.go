// Copyright 2016 Granitic. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file at the root of this project.

/*
	Package xml defines types that are specific to handling web service requests and responses as XML. Components
	implementing this type will be created when you enable the XmlWs facility. For more information on XML web services
	in Granitic, see http://granitic.io/1.0/ref/xml.

	Unmarshalling

	HTTP request bodies sent to a Granitic application with the XmlWs facility enabled are unmarshalled from XML into a
	Go struct using Go's built-in XML unmarshalling techniques. See https://golang.org/pkg/encoding/xml/#Unmarshal

	Templated or marshalling mode

	The data generated by your web service endpoints can be serialised to XML in two ways by setting either:

		{
		  "XmlWs": {
		    "ResponseMode": "MARSHAL"
		  }
		}

	or

		{
		  "XmlWs": {
		    "ResponseMode": "TEMPLATE"
		  }
		}

	in your application configuration file.

	In MARSHAL mode the data and errors in your endpoint's WsResponse objects are serialised using Go's built-in XML
	marshalling techniques. See https://golang.org/pkg/encoding/xml/#Marshal. In TEMPLATE mode each endpoint is
	associated with the name of a template file which is populated with the data and errors in your response. See
	http://granitic.io/1.0/ref/xml#templates for more details.

	Response wrapping

	In MARSHAL mode, any data serialised to XML will first be wrapped with a containing data structure by an instance of GraniticXmlResponseWrapper. This
	means that all responses share a common top level structure for finding the body of the response or errors if they exist.
	For more information on this behaviour (and how to override it) see: http://granitic.io/1.0/ref/xml#wrapping

	Error formatting

	Any service errors found in a response are formatted by an instance of GraniticXmlErrorFormatter before being serialised to XML.
	For more information on this behaviour (and how to override it) see: http://granitic.io/1.0/ref/xml#errors

*/
package xml

import (
	"encoding/xml"
	"github.com/graniticio/granitic/ws"
	"net/http"
)

// Component wrapper over Go's xml.Marshalxx functions. Serialises a struct to XML and writes it to the HTTP response
// output stream.
type XmlMarshalingWriter struct {
	// Format generated XML in a human readable form.
	PrettyPrint bool

	// The characters (generally tabs or spaces) to indent child elements in pretty-printed XML.
	IndentString string

	// A prefix for each line of generated XML.
	PrefixString string
}

// MarshalAndWrite serialises the supplied interface to XML and writes it to the HTTP response output stream.
func (mw *XmlMarshalingWriter) MarshalAndWrite(data interface{}, w http.ResponseWriter) error {

	var b []byte
	var err error

	if mw.PrettyPrint {
		b, err = xml.MarshalIndent(data, mw.PrefixString, mw.IndentString)
	} else {
		b, err = xml.Marshal(data)
	}

	if err != nil {
		return err
	}

	_, err = w.Write(b)

	return err

}

// Component for wrapping response data in a common strcuture before it is serialised.
type GraniticXmlResponseWrapper struct {
}

// WrapResponse wraps the supplied data and errors with an XmlWrapper
func (rw *GraniticXmlResponseWrapper) WrapResponse(body interface{}, errors interface{}) interface{} {

	w := new(GraniticXmlWrapper)

	w.XMLName = xml.Name{"", "response"}
	w.Body = body
	w.Errors = errors

	return w

}

// Wrapper for web service data and errors giving a consistent structure across all XML endpoints.
type GraniticXmlWrapper struct {
	XMLName xml.Name
	Errors  interface{}
	Body    interface{} `xml:"body"`
}

// Converts service errors into a data structure for consistent serialisation to XML.
type GraniticXmlErrorFormatter struct{}

// FormatErrors converts all of the errors present in the supplied objects into a structure suitable for serialisation.
func (ef *GraniticXmlErrorFormatter) FormatErrors(errors *ws.ServiceErrors) interface{} {

	if errors == nil || !errors.HasErrors() {
		return nil
	}

	es := new(XmlErrors)
	es.XMLName = xml.Name{"", "errors"}

	fe := make([]*GraniticXmlError, len(errors.Errors))

	for i, se := range errors.Errors {

		e := new(GraniticXmlError)
		e.XMLName = xml.Name{"", "error"}

		fe[i] = e
		e.Error = se.Message
		e.Field = se.Field
		e.Category = ws.CategoryToName(se.Category)
		e.Code = se.Code

	}

	es.Errors = fe

	return es
}

//Wrapper to create an errors element in generated XML
type XmlErrors struct {
	XMLName xml.Name
	Errors  interface{}
}

//Default XML representation of a service error. See ws.CategorisedError
type GraniticXmlError struct {
	XMLName  xml.Name
	Error    string `xml:",chardata"`
	Field    string `xml:"field,attr,omitempty"`
	Code     string `xml:"code,attr,omitempty"`
	Category string `xml:"category,attr,omitempty"`
}
