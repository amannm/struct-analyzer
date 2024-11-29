package analyzer

import (
	"encoding/json"
	"fmt"
	"slices"
)
import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Package struct {
	Files []*File `json:"files"`
}
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}
type File struct {
	Source  string             `json:"source"`
	Package string             `json:"package"`
	Imports []*Import          `json:"imports,omitempty"`
	Aliases map[string]string  `json:"aliases,omitempty"`
	Structs map[string]*Struct `json:"structs,omitempty"`
}
type Struct struct {
	Extends []string          `json:"extends,omitempty"`
	Fields  map[string]*Field `json:"fields"`
}
type Field struct {
	GoType string `json:"goType"`
	Tags   []*Tag `json:"tags"`
}
type Tag struct {
	Type     string   `json:"type"`
	Argument string   `json:"argument,omitempty"`
	Options  []string `json:"options,omitempty"`
}

func AnalyzeSourceRoot(sourcePaths []string, destinationPath string) error {
	analyses := make([]*File, 0)
	for _, sourcePath := range sourcePaths {
		result := doAnalyze(sourcePath)
		analyses = append(analyses, result...)
	}
	content, err := json.Marshal(analyses)
	if err != nil {
		return err
	}
	return os.WriteFile(destinationPath, content, 0644)
}

var skippedDirectoryNames = []string{"vendor"}

func doAnalyze(root string) []*File {
	analyses := make([]*File, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		ext := filepath.Ext(path)
		if !info.IsDir() {
			if ext == ".go" && !strings.HasSuffix(path, "_test.go") {
				fs := token.NewFileSet()
				goSourceFile, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
				if err != nil {
					return err
				}
				rootParent := filepath.Dir(root)
				relPath, err := filepath.Rel(rootParent, path)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", relPath)
				analysis := analyze(goSourceFile)
				if analysis != nil {
					analysis.Source = relPath
					analyses = append(analyses, analysis)
				}
			}
		} else {
			if slices.Contains(skippedDirectoryNames, info.Name()) {
				return filepath.SkipDir
			}
		}
		if err != nil {
			panic(err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return analyses
}

func analyze(goFile *ast.File) *File {
	packageName := goFile.Name.Name
	imports := make([]*Import, 0)
	structs := map[string]*Struct{}
	aliases := map[string]string{}
	for _, x := range goFile.Imports {
		imp := &Import{
			Path: x.Path.Value,
		}
		if x.Name != nil {
			imp.Alias = x.Name.Name
		}
		imports = append(imports)
	}
	ast.Inspect(goFile, func(n ast.Node) bool {
		switch typedNode := n.(type) {
		case *ast.TypeSpec:
			typeIdentifier := typedNode.Name.Name
			//varParams := typeExpr.TypeParams
			if typedNode.Assign.IsValid() {
				aliases[typeIdentifier] = renderType(typedNode.Type)
			} else {
				switch x := typedNode.Type.(type) {
				case *ast.StructType:
					st := handleStruct(x)
					if st != nil {
						structs[typeIdentifier] = st
					}
				}
			}
		}
		return true
	})
	if len(structs) == 0 && len(aliases) == 0 {
		return nil
	}
	return &File{
		Package: packageName,
		Imports: imports,
		Aliases: aliases,
		Structs: structs,
	}
}

func handleStruct(st *ast.StructType) *Struct {
	fieldAnalyses := map[string]*Field{}
	extensions := make([]string, 0)
	fields := st.Fields
	if fields != nil {
		for _, field := range fields.List {
			typ := renderType(field.Type)
			if typ != "" {
				if len(field.Names) == 0 {
					extensions = append(extensions, typ)
				} else {
					tag := field.Tag
					if tag != nil {
						tagResults := parseTags(tag)
						for _, name := range field.Names {
							fieldAnalyses[name.Name] = &Field{
								Tags:   tagResults,
								GoType: typ,
							}
						}
					}
				}
			}
		}
	}
	if len(fieldAnalyses) == 0 && len(extensions) == 0 {
		return nil
	}
	return &Struct{
		Fields:  fieldAnalyses,
		Extends: extensions,
	}
}

func parseTags(tag *ast.BasicLit) []*Tag {
	unwrapped := strings.Trim(tag.Value, "`")
	rawTags := strings.Split(unwrapped, " ")
	tagResults := make([]*Tag, 0)
	for _, rawTag := range rawTags {
		before, after, ok := strings.Cut(rawTag, ":")
		after = strings.Trim(after, "\"")
		if !ok || before == "" || after == "" {
			continue
		}
		afterParts := strings.Split(after, ",")
		tagResults = append(tagResults, &Tag{
			Type:     before,
			Argument: afterParts[0],
			Options:  afterParts[1:],
		})
	}
	return tagResults
}

func renderType(typeExpression ast.Expr) string {
	switch n := typeExpression.(type) {
	case *ast.Ident:
		// type
		return n.Name
	case *ast.SelectorExpr:
		// package.type
		return renderType(n.X) + "." + n.Sel.Name
	case *ast.StarExpr:
		// *type
		return "*" + renderType(n.X)
	case *ast.ArrayType:
		// []itemType
		return "[]" + renderType(n.Elt)
	case *ast.MapType:
		// map[keyType]valueType
		return "map[" + renderType(n.Key) + "]" + renderType(n.Value)
	case *ast.IndexExpr:
		// type[type]
		return renderType(n.X) + "[" + renderType(n.Index) + "]"
	case *ast.IndexListExpr:
		// type[type, type, type]
		innerTypes := make([]string, 0)
		fields := n.Indices
		if fields != nil {
			for _, ft := range fields {
				innerTypes = append(innerTypes, renderType(ft))
			}
		}
		return renderType(n.X) + "[" + strings.Join(innerTypes, ", ") + "]"
	case *ast.StructType:
		// struct { field, field, field... }
		innerTypes := make([]string, 0)
		fields := n.Fields
		if fields != nil {
			for _, ft := range fields.List {
				innerTypes = append(innerTypes, renderType(ft.Type))
			}
		}
		return "struct{" + strings.Join(innerTypes, ", ") + "}"
	case *ast.InterfaceType:
		return ""
	case *ast.ChanType:
		return ""
	case *ast.FuncType:
		return ""
	default:
		panic(n)
	}
}
