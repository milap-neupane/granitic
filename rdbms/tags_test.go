// Copyright 2016 Granitic. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file at the root of this project.

package rdbms

import (
	"fmt"
	"github.com/graniticio/granitic/test"
	"testing"
)

func TestTagReading(t *testing.T) {

	tt := new(TagTest)

	tt.NoTag = "none"
	tt.ExplicitTag = "exp"

	p, err := ParamsFromTags(tt)

	fmt.Printf("%v\n", p)

	test.ExpectNil(t, err)
	test.ExpectInt(t, len(p), 1)
	test.ExpectString(t, p["explicit"].(string), "exp")

}

func TestNonStructTags(t *testing.T) {
	_, err := ParamsFromTags(1)
	test.ExpectNotNil(t, err)
}

type TagTest struct {
	NoTag       string
	ExplicitTag string `dbparam:"explicit"`
}
