package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/mod/modfile"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

// AnalyzeRepositories analyzes go types used in one or more remote git repositories
func AnalyzeRepositories(gitUris []string, destinationPath string) error {
	var taskResults [][]*File
	taskResults, err := ExecuteAll(gitUris, func(gitUri string) ([]*File, error) {
		return analyzeRepository(gitUri)
	})
	if err != nil {
		return err
	}
	flattenedResults := slices.Concat(taskResults...)
	content, err := json.MarshalIndent(flattenedResults, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destinationPath, content, 0644)
}

func ExecuteAll[T any, U any](args []T, mapper func(T) (U, error)) ([]U, error) {
	numArgs := len(args)
	taskResults := make([]U, numArgs)
	errs := make([]error, numArgs)
	var wg sync.WaitGroup
	wg.Add(numArgs)
	for i, arg := range args {
		go func() {
			defer wg.Done()
			result, err := mapper(arg)
			if err != nil {
				errs[i] = err
				return
			}
			taskResults[i] = result
		}()
	}
	wg.Wait()
	err := errors.Join(errs...)
	if err != nil {
		return nil, err
	}
	return taskResults, nil
}

func analyzeRepository(gitUri string) ([]*File, error) {
	tmp, err := os.MkdirTemp("", "repo-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	_, err = git.PlainClone(tmp, false, &git.CloneOptions{URL: gitUri, Depth: 1})
	if err != nil {
		return nil, err
	}
	roots := locateModuleRoots(tmp)
	result := analyzeModules(gitUri, tmp, roots)
	return result, nil
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

func analyzeModules(remoteGitUri string, localGitRoot string, moduleRoots map[string]string) []*File {
	var results []*File
	err := filepath.WalkDir(localGitRoot, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		ext := filepath.Ext(path)
		if !info.IsDir() {
			if ext == ".go" && !strings.HasSuffix(path, "_test.go") {
				moduleIdentifier := resolveModuleIdentifier(path, moduleRoots)
				if moduleIdentifier == "" {
					moduleIdentifier, err = calculateModuleIdentifier(remoteGitUri, localGitRoot, path)
					if err != nil {
						return err
					}
				}
				result, err := parseModule(localGitRoot, path, moduleIdentifier)
				if err != nil {
					return err
				}
				if result != nil {
					results = append(results, result)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return results
}

func resolveModuleIdentifier(path string, modules map[string]string) string {
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

func calculateModuleIdentifier(remoteGitUri string, localGitRoot string, path string) (string, error) {
	internalPath, err := filepath.Rel(localGitRoot, filepath.Dir(path))
	if err != nil {
		return "", err
	}
	remoteGitUrlBase := strings.TrimPrefix(strings.TrimSuffix(remoteGitUri, ".git"), "https://")
	return remoteGitUrlBase + "/" + internalPath, nil
}

func parseModule(sourceRootPath string, sourceFilePath string, moduleIdentifier string) (*File, error) {
	fs := token.NewFileSet()
	goSourceFile, err := parser.ParseFile(fs, sourceFilePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s\n", sourceFilePath)
	analysis := analyze(goSourceFile)
	if analysis != nil {
		relPath, err := filepath.Rel(sourceRootPath, sourceFilePath)
		if err != nil {
			return nil, err
		}
		relPath = alignPaths(moduleIdentifier, relPath)
		analysis.Location = relPath
		analysis.Module = moduleIdentifier
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
