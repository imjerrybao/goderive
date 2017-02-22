//  Copyright 2017 Walter Schulze
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
)

type Printer interface {
	P(format string, a ...interface{})
	In()
	Out()
	WriteTo(w io.Writer) error

	NewImport(path string) Import
	HasContent() bool
}

type printer struct {
	pkgName    string
	w          *bytes.Buffer
	indent     string
	imports    map[string]string
	hasContent bool
}

func newPrinter(pkgName string) Printer {
	return &printer{pkgName, bytes.NewBuffer(nil), "", make(map[string]string), false}
}

func badToUnderscore(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return r
	}
	return '_'
}

type Import func() string

func (p *printer) NewImport(path string) Import {
	return func() string {
		fullpath := strings.Map(badToUnderscore, path)
		fullpaths := strings.Split(fullpath, "_")
		shortpath := fullpaths[len(fullpaths)-1]
		if _, ok := p.imports[shortpath]; !ok {
			p.imports[shortpath] = path
			return shortpath
		}
		if p.imports[shortpath] == path {
			return shortpath
		}
		if path2, ok := p.imports[fullpath]; ok {
			if path2 != path {
				panic("non unique fullpath: " + path2 + " != " + path)
			}
		}
		p.imports[fullpath] = path
		return fullpath
	}
}

func (p *printer) P(format string, a ...interface{}) {
	p.hasContent = true
	fmt.Fprintf(p.w, p.indent+format+"\n", a...)
}

func (p *printer) In() {
	p.indent += "\t"
}

func (p *printer) Out() {
	if len(p.indent) > 0 {
		p.indent = p.indent[1:]
	} else {
		panic("bug in code generator: unindenting more than has been indented")
	}
}

func (p *printer) HasContent() bool {
	return p.hasContent
}

func (p *printer) WriteTo(file io.Writer) error {
	top := bytes.NewBuffer(nil)
	top.WriteString("//DO NOT EDIT generated by goderive\n")
	top.WriteString("\n")
	top.WriteString("package " + p.pkgName + "\n")
	if len(p.imports) > 0 {
		top.WriteString("\n")
		top.WriteString("import (\n")
		paths := make([]string, 0, len(p.imports))
		pathToQual := make(map[string]string, len(p.imports))
		for qual, path := range p.imports {
			pathToQual[path] = qual
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			qual := pathToQual[path]
			if qual == path {
				top.WriteString("\t\"" + path + "\"\n")
			} else {
				top.WriteString("\t" + qual + " \"" + path + "\"\n")
			}
		}
		top.WriteString(")\n")
	}
	_, err := top.WriteTo(file)
	if err != nil {
		return err
	}
	_, err = p.w.WriteTo(file)
	return err
}
