// Copyright 2018 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"bytes"
	"testing"
)

var parserTestCases = []struct {
	name string
	in   string
	out  []Node
}{
	{
		name: "Escaped $",
		in:   `a$$ b: c`,
		out: []Node{
			&Rule{
				Target:        SimpleMakeString("a$ b", NoPos),
				Prerequisites: SimpleMakeString("c", NoPos),
			},
		},
	},
	{
		name: "Simple warning",
		in:   `$(warning A warning)`,
		out: []Node{
			&Variable{
				Name: SimpleMakeString("warning A warning", NoPos),
			},
		},
	},
	{
		name: "Warning with #",
		in:   `$(warning # A warning)`,
		out: []Node{
			&Variable{
				Name: SimpleMakeString("warning # A warning", NoPos),
			},
		},
	},
	{
		name: "Findstring with #",
		in:   `$(findstring x,x a #)`,
		out: []Node{
			&Variable{
				Name: SimpleMakeString("findstring x,x a #", NoPos),
			},
		},
	},
	{
		name: "If statement",
		in: `ifeq (a,b) # comment
endif`,
		out: []Node{
			&Directive{
				NamePos: NoPos,
				Name:    "ifeq",
				Args:    SimpleMakeString("(a,b) ", NoPos),
				EndPos:  NoPos,
			},
			&Comment{
				CommentPos: NoPos,
				Comment:    " comment",
			},
			&Directive{
				NamePos: NoPos,
				Name:    "endif",
				Args:    SimpleMakeString("", NoPos),
				EndPos:  NoPos,
			},
		},
	},
	{
		name: "Blank line in rule's command",
		in: `all:
	echo first line

	echo second line`,
		out: []Node{
			&Rule{
				Target:        SimpleMakeString("all", NoPos),
				RecipePos:     NoPos,
				Recipe:        "echo first line\necho second line",
				Prerequisites: SimpleMakeString("", NoPos),
			},
		},
	},
}

func TestParse(t *testing.T) {
	for _, test := range parserTestCases {
		t.Run(test.name, func(t *testing.T) {
			p := NewParser(test.name, bytes.NewBufferString(test.in))
			got, errs := p.Parse()

			if len(errs) != 0 {
				t.Fatalf("Unexpected errors while parsing: %v", errs)
			}

			if len(got) != len(test.out) {
				t.Fatalf("length mismatch, expected %d nodes, got %d", len(test.out), len(got))
			}

			for i := range got {
				if got[i].Dump() != test.out[i].Dump() {
					t.Errorf("incorrect node %d:\nexpected: %#v (%s)\n     got: %#v (%s)",
						i, test.out[i], test.out[i].Dump(), got[i], got[i].Dump())
				}
			}
		})
	}
}

func TestRuleEnd(t *testing.T) {
	name := "ruleEndTest"
	in := `all:
ifeq (A, A)
	echo foo
	echo foo
	echo foo
	echo foo
endif
	echo bar
`
	p := NewParser(name, bytes.NewBufferString(in))
	got, errs := p.Parse()
	if len(errs) != 0 {
		t.Fatalf("Unexpected errors while parsing: %v", errs)
	}

	if got[0].End() < got[len(got) -1].Pos() {
		t.Errorf("Rule's end (%d) is smaller than directive that inside of rule's start (%v)\n", got[0].End(), got[len(got) -1].Pos())
	}
}
