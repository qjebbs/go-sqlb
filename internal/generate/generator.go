package generate

import (
	"bytes"
	_ "embed"
	"go/ast"
	"go/format"
	"go/types"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/qjebbs/go-sqlb/internal/tag/syntax"
	"golang.org/x/tools/go/packages"
)

// Generator is responsible for generating SQL builder code based on struct definitions in Go source files.
type Generator struct {
	// Unifile indicates whether to generate a single file for the entire package
	// or separate files for each file.
	Unifile bool
}

// NewGenerator creates a new instance of Generator with the specified unifile option.
func NewGenerator(unifile bool) *Generator {
	return &Generator{
		Unifile: unifile,
	}
}

// Generate processes the provided patterns to find Go packages, extract struct information, and generate SQL builder code.
func (g *Generator) Generate(patterns []string) {
	pkgs := g.findPackages(patterns)

	if g.Unifile {
		for _, pkg := range pkgs {
			var allStructs []StructInfo
			for _, fileNode := range pkg.Syntax {
				structs := g.processFile(pkg, fileNode)
				allStructs = append(allStructs, structs...)
			}
			if len(allStructs) > 0 {
				dir := filepath.Dir(pkg.GoFiles[0])
				g.write(
					pkg, allStructs,
					filepath.Join(dir, pkg.Name),
				)
			}
		}
		return
	}

	for _, pkg := range pkgs {
		// file path -> structs
		fileStructs := make(map[string][]StructInfo)
		for i, fileNode := range pkg.Syntax {
			filePath := pkg.GoFiles[i]
			structs := g.processFile(pkg, fileNode)
			if len(structs) > 0 {
				fileStructs[filePath] = append(fileStructs[filePath], structs...)
			}
		}

		for filePath, structs := range fileStructs {
			if len(structs) > 0 {
				g.write(pkg, structs, filePath)
			}
		}
	}
}

func (g *Generator) findPackages(patterns []string) []*packages.Package {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		log.Fatalf("failed to load packages: %v", err)
	}
	if len(pkgs) == 0 {
		log.Fatalf("no packages found for patterns: %v", patterns)
	}
	return pkgs
}

func (g *Generator) processFile(pkg *packages.Package, node *ast.File) []StructInfo {
	var structs []StructInfo
	ast.Inspect(node, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		s, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}

		var columns []ColumnInfo

		type context struct {
			table [2]string
		}

		var findFields func(fields []FieldInfo, basePath []int, ctx context)
		findFields = func(fields []FieldInfo, basePath []int, ctx context) {
			curTable := ctx.table
			for i, field := range fields {
				currentPath := append(basePath, i)

				var info *syntax.Info
				if field.Tag != "" {
					tagVal := reflect.StructTag(strings.Trim(field.Tag, "`")).Get("sqlb")
					if tagVal == "-" {
						continue
					}
					if tagVal != "" {
						parsed, err := syntax.Parse(tagVal)
						if err != nil {
							log.Fatalf("failed to parse tag: %v", err)
						}
						if parsed.Table[0] != "" {
							curTable = parsed.Table
						} else {
							parsed.Table = curTable
						}
						info = parsed
					}
				}

				isAnonymous := field.IsAnonymous
				if isAnonymous {
					var typeObj types.Object
					var ok bool = true

					switch t := field.Type.(type) {
					case *ast.Ident:
						typeObj, ok = pkg.TypesInfo.Uses[t]
						if !ok || typeObj == nil {
							typeObj, ok = pkg.TypesInfo.Defs[t]
						}
					case *ast.SelectorExpr:
						typeObj, ok = pkg.TypesInfo.Uses[t.Sel]
					case *types.Named:
						typeObj = t.Obj()
					case *types.Alias:
						typeObj = t.Obj()
					}
					if !ok || typeObj == nil {
						continue
					}

					underlyingStruct := findUnderlyingStruct(typeObj.Type())

					if underlyingStruct != nil {
						var embeddedFields []FieldInfo
						for i := 0; i < underlyingStruct.NumFields(); i++ {
							f := underlyingStruct.Field(i)
							embeddedFields = append(embeddedFields, FieldInfo{
								Name:        f.Name(),
								Tag:         underlyingStruct.Tag(i),
								IsAnonymous: f.Anonymous(),
								IsExported:  f.Exported(),
								Type:        f.Type(),
							})
						}
						ctx.table = curTable
						findFields(embeddedFields, currentPath, ctx)
					}
					continue
				}

				if !field.IsExported {
					continue
				}

				if info != nil {
					if info.Table[0] == "" || info.Column == "" {
						continue
					}

					columns = append(columns, ColumnInfo{
						FieldName:  field.Name,
						ColumnName: info.Column,
						TableName:  info.Table[0],
						TableAlias: info.Table[1],
					})
				}
			}
		}

		var initialFields []FieldInfo
		for _, f := range s.Fields.List {
			var name string
			if len(f.Names) > 0 {
				name = f.Names[0].Name
			}
			var tag string
			if f.Tag != nil {
				tag = f.Tag.Value
			}
			initialFields = append(initialFields, FieldInfo{
				Name:        name,
				Tag:         tag,
				IsAnonymous: f.Names == nil,
				IsExported:  f.Names == nil || f.Names[0].IsExported(),
				Type:        f.Type,
			})
		}

		findFields(initialFields, []int{}, context{})

		if len(columns) > 0 {
			var onlyTable [2]string
			for _, col := range columns {
				if onlyTable[0] == "" {
					onlyTable = [2]string{col.TableName, col.TableAlias}
				} else if onlyTable[0] != col.TableName || onlyTable[1] != col.TableAlias {
					onlyTable = [2]string{}
					break
				}
			}
			structs = append(structs, StructInfo{
				Name:       ts.Name.Name,
				Columns:    columns,
				TableName:  onlyTable[0],
				TableAlias: onlyTable[1],
			})
		}
		return false
	})
	return structs
}

func (g *Generator) write(pkg *packages.Package, structs []StructInfo, filePath string) {
	if len(structs) == 0 {
		return
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, &PackageInfo{
		Name:    pkg.Name,
		Structs: structs,
	})
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("failed to format generated code: %v", err)
	}

	var outputName string

	if strings.HasSuffix(pkg.Name, "_test") {
		outputName = strings.TrimSuffix(filePath, "_test.go") + "_sqlb_gen_test.go"
	} else {
		outputName = strings.TrimSuffix(filePath, ".go") + "_sqlb_gen.go"
	}
	err = os.WriteFile(outputName, formatted, 0644)
	if err != nil {
		log.Fatalf("failed to write output file: %v", err)
	}
}

//go:embed code.tmpl
var codeTemplate string

var tmpl = template.Must(template.New("").Parse(codeTemplate))
