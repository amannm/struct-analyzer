package core

import (
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/mod/modfile"
	"log"

	"os"
	"path/filepath"
	"strings"
)

type File struct {
	Module        string             `json:"module"`
	Package       string             `json:"package"`
	Location      string             `json:"location"`
	Imports       []*Import          `json:"imports,omitempty"`
	Aliases       map[string]string  `json:"aliases,omitempty"`
	Structs       map[string]*Struct `json:"structs,omitempty"`
	Documentation string             `json:"documentation,omitempty"`
}
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}
type Struct struct {
	Embeds        []string          `json:"embeds,omitempty"`
	Fields        map[string]*Field `json:"fields"`
	Documentation string            `json:"documentation,omitempty"`
	Comments      string            `json:"comments,omitempty"`
}
type Field struct {
	GoType        string `json:"goType"`
	Tags          []*Tag `json:"tags"`
	Documentation string `json:"documentation,omitempty"`
	Comments      string `json:"comments,omitempty"`
}
type Tag struct {
	Type     string   `json:"type"`
	Argument string   `json:"argument,omitempty"`
	Options  []string `json:"options,omitempty"`
}

// AnalyzeRepositories analyzes a local git repository and extracts all types with struct tags usually associated with config files
func AnalyzeRepositories(gitUris []string, destinationPath string) error {
	analyses := make([]*File, 0)
	tempDirs := make([]string, 0)
	defer func() {
		for _, tempDir := range tempDirs {
			_ = os.RemoveAll(tempDir)
		}
	}()
	for _, gitUri := range gitUris {
		tmp, err := os.MkdirTemp("", "repo-*")
		if err != nil {
			return err
		}
		tempDirs = append(tempDirs, tmp)
		_, err = git.PlainClone(tmp, false, &git.CloneOptions{URL: gitUri, Depth: 1})
		if err != nil {
			return err
		}
		roots := locateModuleRoots(tmp)
		result := doAnalyze(gitUri, tmp, roots)
		analyses = append(analyses, result...)
	}
	content, err := json.MarshalIndent(analyses, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destinationPath, content, 0644)
}

func locateModuleRoots(gitRoot string) map[string]string {
	results := map[string]string{}
	err := filepath.WalkDir(gitRoot, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			if filepath.Base(path) == "go.mod" {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				modulePath := modfile.ModulePath(content)
				results[filepath.Dir(path)] = modulePath
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return results
}

func resolveModule(path string, modules map[string]string) string {
	for {
		dir, _ := filepath.Split(path)
		dir = strings.TrimSuffix(dir, "/")
		if dir == "" {
			return ""
		}
		module, ok := modules[dir]
		if ok {
			return module
		}
		path = dir
	}
}

func doAnalyze(remoteGitUri string, localGitRoot string, moduleRoots map[string]string) []*File {
	var analyses []*File
	err := filepath.WalkDir(localGitRoot, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		ext := filepath.Ext(path)
		if !info.IsDir() {
			if ext == ".go" && !strings.HasSuffix(path, "_test.go") {
				currentModulePath := resolveModule(path, moduleRoots)
				if currentModulePath == "" {
					currentModulePath, err = calculateModule(remoteGitUri, localGitRoot, path)
					if err != nil {
						return err
					}
				}
				analysis, err := analyzeModule(localGitRoot, path, currentModulePath)
				if err != nil {
					return err
				}
				if analysis != nil {
					analyses = append(analyses, analysis)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return analyses
}

func calculateModule(remoteGitUri string, localGitRoot string, path string) (string, error) {
	internalPath, err := filepath.Rel(localGitRoot, filepath.Dir(path))
	if err != nil {
		return "", err
	}
	remoteGitUrlBase := strings.TrimPrefix(strings.TrimSuffix(remoteGitUri, ".git"), "https://")
	return remoteGitUrlBase + "/" + internalPath, nil
}

func analyzeModule(gitRoot string, path string, currentModulePath string) (*File, error) {
	fs := token.NewFileSet()
	goSourceFile, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s\n", path)
	analysis := analyze(goSourceFile)
	if analysis != nil {
		relPath, err := filepath.Rel(gitRoot, path)
		if err != nil {
			return nil, err
		}
		relPath = alignPaths(currentModulePath, relPath)
		analysis.Location = relPath
		analysis.Module = currentModulePath
		return analysis, nil
	}
	return nil, nil
}

func analyze(goFile *ast.File) *File {
	packageName := goFile.Name.Name
	imports := make([]*Import, 0)
	structs := map[string]*Struct{}
	aliases := map[string]string{}
	fileDoc := ""
	if goFile.Doc != nil {
		fileDoc = strings.TrimSuffix(goFile.Doc.Text(), "\n")
	}
	for _, x := range goFile.Imports {
		imp := &Import{
			Path: strings.Trim(x.Path.Value, "\""),
		}
		if x.Name != nil {
			imp.Alias = x.Name.Name
		}
		imports = append(imports, imp)
	}
	ast.Inspect(goFile, func(n ast.Node) bool {
		switch typedNode := n.(type) {
		case *ast.TypeSpec:
			doc := ""
			if typedNode.Doc != nil {
				doc = strings.TrimSuffix(typedNode.Doc.Text(), "\n")
			}
			comm := ""
			if typedNode.Comment != nil {
				comm = strings.TrimSuffix(typedNode.Comment.Text(), "\n")
			}
			typeIdentifier := typedNode.Name.Name
			//varParams := typeExpr.TypeParams
			if typedNode.Assign.IsValid() {
				aliases[typeIdentifier] = renderType(typedNode.Type)
			} else {
				switch x := typedNode.Type.(type) {
				default:
					renderedType := renderType(x)
					if renderedType != "" {
						aliases[typeIdentifier] = renderedType
					}
				case *ast.StructType:
					st := handleStruct(x)
					if st != nil {
						st.Documentation = doc
						st.Comments = comm
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
		Package:       packageName,
		Imports:       imports,
		Aliases:       aliases,
		Structs:       structs,
		Documentation: fileDoc,
	}
}

func handleStruct(st *ast.StructType) *Struct {
	fieldAnalyses := map[string]*Field{}
	embeds := make([]string, 0)
	fields := st.Fields
	if fields != nil {
		for _, field := range fields.List {
			typ := renderType(field.Type)
			if typ != "" {
				if len(field.Names) == 0 {
					embeds = append(embeds, typ)
				} else {
					tag := field.Tag
					tagResults := make([]*Tag, 0)
					if tag != nil {
						tagResults = parseTags(tag)
					}
					for _, name := range field.Names {
						var doc string
						if field.Doc != nil {
							doc = field.Doc.Text()
						}
						var comment string
						if field.Comment != nil {
							comment = field.Comment.Text()
						}
						fieldAnalyses[name.Name] = &Field{
							Tags:          tagResults,
							GoType:        typ,
							Documentation: strings.TrimSuffix(doc, "\n"),
							Comments:      strings.TrimSuffix(comment, "\n"),
						}
					}
				}
			}
		}
	}
	if len(fieldAnalyses) == 0 && len(embeds) == 0 {
		return nil
	}
	return &Struct{
		Fields: fieldAnalyses,
		Embeds: embeds,
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
	case *ast.Ellipsis:
		return ""
	case *ast.FuncType:
		innerTypes := make([]string, 0)
		params := n.Params
		if params != nil {
			for _, ft := range params.List {
				innerTypes = append(innerTypes, renderType(ft.Type))
			}
		}
		returnTypes := make([]string, 0)
		results := n.Results
		if results != nil {
			for _, ft := range results.List {
				returnTypes = append(returnTypes, renderType(ft.Type))
			}
		}
		return "func(" + strings.Join(innerTypes, ", ") + ") (" + strings.Join(returnTypes, ", ") + ")"
	default:
		panic(n)
	}
}

func alignPaths(base string, other string) string {
	otherParts := strings.Split(other, "/")
	suffix := ""
	for i := 0; i < len(otherParts); i++ {
		suffix += "/" + otherParts[i]
		if strings.HasSuffix(base, suffix) {
			return strings.TrimPrefix(other, suffix[1:])[1:]
		}
	}
	return other
}
