package migration

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
)

type importManager struct {
	addPaths    map[string]bool
	removePaths map[string]bool
}

func newImportManager() *importManager {
	return &importManager{
		addPaths:    map[string]bool{},
		removePaths: map[string]bool{},
	}
}

func (m *importManager) Add(path string) {
	m.addPaths[path] = true
}

func (m *importManager) RemoveIfAliasUnused(path, alias string) {
	m.removePaths[path] = true
}

func (m *importManager) Apply(filename string, src []byte) ([]byte, error) {
	if len(m.addPaths) == 0 && len(m.removePaths) == 0 {
		return src, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	add := map[string]bool{}
	for p := range m.addPaths {
		add[p] = true
	}
	for _, spec := range file.Imports {
		p, _ := strconv.Unquote(spec.Path.Value)
		delete(add, p)
	}
	remove := map[*ast.ImportSpec]bool{}
	for _, spec := range file.Imports {
		p, _ := strconv.Unquote(spec.Path.Value)
		if !m.removePaths[p] {
			continue
		}
		alias := importSpecAlias(spec, p)
		if alias == "." || alias == "_" {
			continue
		}
		if !selectorAliasUsed(file, alias) {
			remove[spec] = true
		}
	}
	if len(add) == 0 && len(remove) == 0 {
		return src, nil
	}
	decls := importDecls(file)
	if len(decls) == 0 {
		paths := make([]string, 0, len(add))
		for p := range add {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		var b strings.Builder
		if len(paths) == 1 {
			fmt.Fprintf(&b, "import %q\n\n", paths[0])
		} else {
			b.WriteString("import (\n")
			for _, p := range paths {
				fmt.Fprintf(&b, "\t%q\n", p)
			}
			b.WriteString(")\n\n")
		}
		insert := fset.Position(file.Name.End()).Offset
		next := append([]byte{}, src[:insert]...)
		next = append(next, '\n', '\n')
		next = append(next, b.String()...)
		next = append(next, src[insert:]...)
		return next, nil
	}
	first := decls[0]
	last := decls[len(decls)-1]
	start := fset.Position(first.Pos()).Offset
	end := absorbImportTrailingNewlines(src, fset.Position(last.End()).Offset)
	specs := make([]*ast.ImportSpec, 0, len(file.Imports)+len(add))
	for _, spec := range file.Imports {
		if !remove[spec] {
			specs = append(specs, spec)
		}
	}
	addPaths := make([]string, 0, len(add))
	for p := range add {
		addPaths = append(addPaths, p)
	}
	sort.Strings(addPaths)
	for _, p := range addPaths {
		specs = append(specs, &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(p)}})
	}
	sort.SliceStable(specs, func(i, j int) bool {
		ip, _ := strconv.Unquote(specs[i].Path.Value)
		jp, _ := strconv.Unquote(specs[j].Path.Value)
		return ip < jp
	})
	newDecl := renderImportDecl(specs)
	next := append([]byte{}, src[:start]...)
	if newDecl != "" {
		next = append(next, newDecl...)
	}
	next = append(next, src[end:]...)
	return next, nil
}

func absorbImportTrailingNewlines(src []byte, end int) int {
	if end >= len(src) || src[end] != '\n' {
		return end
	}
	end++
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	if end < len(src) && src[end] == '\n' {
		end++
	}
	return end
}

func importDecls(file *ast.File) []*ast.GenDecl {
	var decls []*ast.GenDecl
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			decls = append(decls, gen)
		}
	}
	return decls
}

func renderImportDecl(specs []*ast.ImportSpec) string {
	if len(specs) == 0 {
		return ""
	}
	var node ast.Node
	if len(specs) == 1 {
		node = &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{specs[0]}}
	} else {
		gen := &ast.GenDecl{Tok: token.IMPORT, Lparen: token.Pos(1)}
		for _, spec := range specs {
			gen.Specs = append(gen.Specs, spec)
		}
		node = gen
	}
	var b bytes.Buffer
	_ = printer.Fprint(&b, token.NewFileSet(), node)
	b.WriteString("\n\n")
	return b.String()
}

func selectorAliasUsed(file *ast.File, alias string) bool {
	used := false
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == alias {
			used = true
			return false
		}
		return true
	})
	return used
}

func importAliases(file *ast.File) map[string][]string {
	aliases := make(map[string][]string)
	for _, spec := range file.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := importSpecAlias(spec, p)
		aliases[p] = append(aliases[p], name)
	}
	return aliases
}

func importSpecAlias(spec *ast.ImportSpec, importPath string) string {
	name := path.Base(importPath)
	if spec.Name != nil {
		name = spec.Name.Name
	}
	return name
}
