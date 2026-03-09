package generate

import (
	"bytes"
	"fmt"

	// embed is used to embed the code template into the binary,
	// allowing for easy distribution without external template files.
	_ "embed"
	"go/format"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

//go:embed code.tmpl
var codeTemplate string

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"mkmap": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("invalid number of arguments to mkmap")
		}
		m := make(map[string]interface{})
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("map key must be a string")
			}
			m[key] = values[i+1]
		}
		return m, nil
	},
}).Parse(codeTemplate))

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
func (g *Generator) Generate(patterns []string) error {
	pkgs, err := g.findPackages(patterns)
	if err != nil {
		return fmt.Errorf("failed to find packages: %v", err)
	}

	if g.Unifile {
		for _, pkg := range pkgs {
			var allStructs []StructInfo
			for i, fileNode := range pkg.Syntax {
				filePath := pkg.GoFiles[i]
				structs, err := g.processFile(pkg, fileNode)
				if err != nil {
					return fmt.Errorf("failed to process file %s: %v", filePath, err)
				}
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
		return nil
	}

	for _, pkg := range pkgs {
		// file path -> structs
		fileStructs := make(map[string][]StructInfo)
		for i, fileNode := range pkg.Syntax {
			filePath := pkg.GoFiles[i]
			structs, err := g.processFile(pkg, fileNode)
			if err != nil {
				return fmt.Errorf("failed to process file %s: %v", filePath, err)
			}
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
	return nil
}

func (g *Generator) findPackages(patterns []string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %v", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for patterns: %v", patterns)
	}
	return pkgs, nil
}

// Info holds information about the package and structs to be used in the code generation template.
type Info struct {
	Name    string
	Structs []StructInfo
	Imports []string
}

func (g *Generator) write(pkg *packages.Package, structs []StructInfo, filePath string) {
	if len(structs) == 0 {
		return
	}
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, &Info{
		Name:    pkg.Name,
		Structs: structs,
		Imports: g.collectImports(structs),
	})
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Println("Generated code:")
		fmt.Println(buf.String())
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

func (g *Generator) collectImports(structs []StructInfo) []string {
	importSet := make(map[string]struct{})
	for _, s := range structs {
		if s.Select != nil {
			for _, imp := range s.Select.Imports {
				importSet[imp] = struct{}{}
			}
		}
	}
	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	return imports
}
