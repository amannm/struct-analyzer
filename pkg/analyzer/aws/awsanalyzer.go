package aws

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type DocFile struct {
	Structs map[string]map[string]string `json:"structs,omitempty"`
}

// ExtractSDKTypes reads a file from
func ExtractSDKTypes(path string) map[string]map[string]string {
	fs := token.NewFileSet()
	goSourceFile, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	fmt.Printf("%s\n", path)
	return analyzeStructs(goSourceFile)
}

func analyzeStructs(goFile *ast.File) map[string]map[string]string {
	structs := map[string]map[string]string{}
	ast.Inspect(goFile, func(n ast.Node) bool {
		switch typedNode := n.(type) {
		case *ast.TypeSpec:
			typeIdentifier := typedNode.Name.Name
			switch x := typedNode.Type.(type) {
			case *ast.StructType:
				structs[typeIdentifier] = analyzeStructFields(x)
			}
		}
		return true
	})
	return structs
}
func analyzeStructFields(st *ast.StructType) map[string]string {
	fieldAnalyses := map[string]string{}
	fields := st.Fields
	if fields != nil {
		for _, field := range fields.List {
			for _, name := range field.Names {
				fieldAnalyses[name.Name] = strings.TrimSuffix(field.Doc.Text(), "\n")
			}
		}
	}
	return fieldAnalyses
}
