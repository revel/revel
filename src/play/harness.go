// The Harness for a GoPlay! program.
//
// It has a couple responsibilities:
// 1. Parse the user program, generating source files:
//   _controllers.go: Calls to register all of the controller classes.
// 2. Build and run the user program.  Show compile errors.
// 3. Monitor the user source and re-build / restart the program when necessary.
//
// Source files are generated in the app/tmp directory.

package main

import (
	"text/template"
	"play"
	"log"
	"os"
	"path/filepath"
	"go/build"
	"go/token"
	"go/parser"
	"go/ast"
	"fmt"
	"os/exec"
)

const REGISTER_CONTROLLERS = `
package main

import (
	"play"
	{{range .controllers}}
  "{{.FullPackageName}}"
  {{end}}
)

func main() {
  {{range .controllers}}
	play.RegisterController((*{{.PackageName}}.{{.StructName}})(nil))
  {{end}}
  play.Run()
}
`

func main() {
	tmpl := template.New("RegisterControllers")
	tmpl = template.Must(tmpl.Parse(REGISTER_CONTROLLERS))

	var registerControllerSource string = play.ExecuteTemplate(tmpl, map[string]interface{} {
		"controllers": ListControllers(filepath.Join(play.AppPath, "controllers")),
	})

	// Create a fresh temp dir.
	tmpPath := filepath.Join(play.AppPath, "tmp")

	os.Remove(tmpPath)
	err := os.Mkdir(tmpPath, 0777)
	if err != nil {
		log.Fatalf("Failed to make tmp directory: %v", err)
	}

	// Create the new file
	controllersFile, err := os.Create(filepath.Join(tmpPath, "main.go"))
	if err != nil {
		log.Fatalf("Failed to create main.go: %v", err)
	}
	_, err = controllersFile.WriteString(registerControllerSource)
	if err != nil {
		log.Fatalf("Failed to write to main.go: %v", err)
	}

	// Build the user program (all code under app).

	// Find all subdirectories of /app/
	var appDirectories []string
	appDir, err := os.Open(play.AppPath)
	if err != nil {
		log.Fatalf("Failed to open directory: %s", err)
	}

	fileInfos, err := appDir.Readdir(-1)
	if err != nil {
		log.Fatalf("Failed to read directory %s: %s", play.AppPath, err)
	}
	for _, fileInfo := range(fileInfos) {
		if fileInfo.IsDir() {
			appDirectories = append(appDirectories, fileInfo.Name())
		}
	}

	// Scan each directory.
	for _, appDirectory := range(appDirectories) {
		fqDir := filepath.Join(play.AppPath, appDirectory)
		dir, e := build.ScanDir(fqDir)
		if e != nil {
			// TODO: Ignore just the "no Go source files" error.
			continue
		}

		tree, pkg, e := build.FindTree(fqDir)
		if e != nil {
			log.Fatal(e)
		}

		// If this tree has the main package, we're going to be running it.
		script, e := build.Build(tree, pkg, dir)
		if e != nil {
			log.Fatal(e)
		}

		e = script.Run()
		if e != nil {
			log.Fatal(e)
		}
	}

	// Run the user's server, via tmp/main.go.
	appTree, _, _ := build.FindTree(play.AppPath)
	cmd := exec.Command(filepath.Join(appTree.BinDir(), "tmp"))
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error running:", err)
	}
	fmt.Println("Exit.")
}

type typeName struct {
	FullPackageName, PackageName, StructName string
}

func ListControllers(path string) (controllerTypeNames []*typeName) {
	// Parse files within the path.
	var pkgs map[string]*ast.Package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(f os.FileInfo) bool { return !f.IsDir() }, 0)
	if err != nil {
		log.Fatalf("Failed to parse dir: %s", err)
	}

	// For each package... (often only "controllers")
	for _, pkg := range pkgs {

		// For each source file in the package...
		for _, file := range pkg.Files {

			// For each declaration in the source file...
			for _, decl := range file.Decls {
				// Find Type declarations that embed *play.Controller.
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				if genDecl.Tok != token.TYPE {
					continue
				}

				if len(genDecl.Specs) != 1 {
					play.LOG.Printf("Surprising: Decl does not have 1 Spec: %v", genDecl)
					continue
				}

				spec := genDecl.Specs[0].(*ast.TypeSpec)
				structType, ok := spec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				var fieldList []*ast.Field = structType.Fields.List
				if len(fieldList) == 0 {
					continue
				}

				// Look for ast.Field to have type StarExpr { SelectorExpr { "play", "Controller" } }
				starExpr, ok := fieldList[0].Type.(*ast.StarExpr)
				if !ok {
					continue
				}

				selectorExpr, ok := starExpr.X.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				pkgIdent, ok := selectorExpr.X.(*ast.Ident)
				if !ok {
					continue
				}

				// TODO: Support sub-types of play.Controller as well.
				if pkgIdent.Name == "play" && selectorExpr.Sel.Name == "Controller" {
					controllerTypeNames = append(controllerTypeNames,
						&typeName{play.AppName + "/app/" + pkg.Name, pkg.Name, spec.Name.Name})
				}
			}
		}
	}
	return
}
