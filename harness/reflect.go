package harness

// This file handles the app code introspection.
// It catalogs the controllers, their methods, and their arguments.

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"log"
	"os"
	"play"
	"strings"
	"unicode"
)

type ControllerSpec struct {
	PackageName string
	StructName  string
	ImportPath  string
	MethodSpecs []*MethodSpec

	// Used internally to identify controllers that indirectly embed *play.Controller.
	embeddedTypes []*embeddedTypeName
}

// This is a description of a call to c.Render(..)
// It documents the argument names used, in order to propagate them to RenderArgs.
type renderCall struct {
	Line  int
	Names []string
}

type MethodSpec struct {
	Name        string        // Name of the method, e.g. "Index"
	Args        []*MethodArg  // Argument descriptors
	RenderCalls []*renderCall // Descriptions of Render() invocations from this Method.
}

type MethodArg struct {
	Name       string // Name of the argument.
	TypeName   string // The name of the type, e.g. "int", "*pkg.UserType"
	ImportPath string // If the arg is of an imported type, this is the import path.
}

type embeddedTypeName struct {
	PackageName, StructName string
}

// Maps a controller simple name (e.g. "Login") to the methods for which it is a
// receiver.
type methodMap map[string][]*MethodSpec

// Parse the app directory and return a list of the controller types found.
// Returns a CompileError if the parsing fails.
func ScanControllers(path string) (specs []*ControllerSpec, compileError *play.CompileError) {
	// Parse files within the path.
	var pkgs map[string]*ast.Package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(f os.FileInfo) bool { return !f.IsDir() }, 0)
	if err != nil {
		if errList, ok := err.(scanner.ErrorList); ok {
			var pos token.Position = errList[0].Pos
			return nil, &play.CompileError{
				SourceType:  ".go source",
				Title:       "Go Compilation Error",
				Path:        pos.Filename,
				Description: errList[0].Msg,
				Line:        pos.Line,
				Column:      pos.Column,
				SourceLines: play.MustReadLines(pos.Filename),
			}
		}
		ast.Print(nil, err)
		log.Fatalf("Failed to parse dir: %s", err)
	}

	// For each package... (often only "controllers")
	for _, pkg := range pkgs {
		var structSpecs []*ControllerSpec
		methodSpecs := make(methodMap)

		// For each source file in the package...
		for _, file := range pkg.Files {

			// Imports maps the package key to the full import path.
			// e.g. import "sample/app/models" => "models": "sample/app/models"
			imports := map[string]string{}

			// For each declaration in the source file...
			for _, decl := range file.Decls {

				// Match and add both structs and methods
				addImports(imports, decl)
				structSpecs = appendStruct(structSpecs, pkg, decl)
				appendMethod(fset, methodSpecs, decl, pkg.Name, imports)
			}
		}

		// Filter the struct specs to just the ones that embed play.Controller.
		structSpecs = filterControllers(structSpecs)

		// Add the method specs to them.
		for _, spec := range structSpecs {
			spec.MethodSpecs = methodSpecs[spec.StructName]
		}

		// Add the prepared ControllerSpecs to the list.
		specs = append(specs, structSpecs...)
	}
	return
}

func addImports(imports map[string]string, decl ast.Decl) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.IMPORT {
		return
	}

	for _, spec := range genDecl.Specs {
		importSpec := spec.(*ast.ImportSpec)
		quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
		fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes
		key := fullPath
		if lastSlash := strings.LastIndex(fullPath, "/"); lastSlash != -1 {
			key = fullPath[lastSlash+1:]
		}
		imports[key] = fullPath
	}
}

// If this Decl is a struct type definition, it is summarized and added to specs.
// Else, specs is returned unchanged.
func appendStruct(specs []*ControllerSpec, pkg *ast.Package, decl ast.Decl) []*ControllerSpec {
	// Filter out non-Struct type declarations.
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return specs
	}

	if genDecl.Tok != token.TYPE {
		return specs
	}

	if len(genDecl.Specs) != 1 {
		play.LOG.Printf("Surprising: Decl does not have 1 Spec: %v", genDecl)
		return specs
	}

	spec := genDecl.Specs[0].(*ast.TypeSpec)
	structType, ok := spec.Type.(*ast.StructType)
	if !ok {
		return specs
	}

	// At this point we know it's a type declaration for a struct.
	// Fill in the rest of the info by diving into the fields.
	// Add it provisionally to the Controller list -- it's later filtered using field info.
	controllerSpec := &ControllerSpec{
		PackageName: pkg.Name,
		StructName:  spec.Name.Name,
		ImportPath:  play.ImportPath + "/app/" + pkg.Name,
	}

	for _, field := range structType.Fields.List {
		// If field.Names is set, it's not an embedded type.
		if field.Names != nil {
			continue
		}

		// A direct "sub-type" has an ast.Field as either:
		//   Ident { "AppController" }
		//   SelectorExpr { "play", "Controller" }
		// Additionally, that can be wrapped by StarExprs.
		fieldType := field.Type
		pkgName, typeName := func() (string, string) {
			// Drill through any StarExprs.
			for {
				if starExpr, ok := fieldType.(*ast.StarExpr); ok {
					fieldType = starExpr.X
					continue
				}
				break
			}

			// If the embedded type is in the same package, it's an Ident.
			if ident, ok := fieldType.(*ast.Ident); ok {
				return pkg.Name, ident.Name
			}

			if selectorExpr, ok := fieldType.(*ast.SelectorExpr); ok {
				if pkgIdent, ok := selectorExpr.X.(*ast.Ident); ok {
					return pkgIdent.Name, selectorExpr.Sel.Name
				}
			}

			return "", ""
		}()

		// If a typename wasn't found, skip it.
		if typeName == "" {
			continue
		}

		controllerSpec.embeddedTypes = append(controllerSpec.embeddedTypes, &embeddedTypeName{
			PackageName: pkgName,
			StructName:  typeName,
		})
	}

	return append(specs, controllerSpec)
}

// If decl is a Method declaration, it is summarized and added to the array
// underneath its receiver type.
// e.g. "Login" => {MethodSpec, MethodSpec, ..}
func appendMethod(fset *token.FileSet, mm methodMap, decl ast.Decl, pkgName string, imports map[string]string) {
	// Func declaration?
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		return
	}

	// Have a receiver?
	if funcDecl.Recv == nil {
		return
	}

	// Is it public?
	if !unicode.IsUpper([]rune(funcDecl.Name.Name)[0]) {
		return
	}

	// Does it return a play.Result?
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return
	}
	selExpr, ok := funcDecl.Type.Results.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if pkgIdent, ok := selExpr.X.(*ast.Ident); !ok || pkgIdent.Name != "play" {
		return
	}
	if selExpr.Sel.Name != "Result" {
		return
	}

	// Get the receiver type, "dereferencing" it if necessary
	var recvTypeName string
	var recvType ast.Expr = funcDecl.Recv.List[0].Type
	if recvStarType, ok := recvType.(*ast.StarExpr); ok {
		recvTypeName = recvStarType.X.(*ast.Ident).Name
	} else {
		recvTypeName = recvType.(*ast.Ident).Name
	}

	method := &MethodSpec{
		Name: funcDecl.Name.Name,
	}

	// Add a description of the arguments to the method.
	for _, field := range funcDecl.Type.Params.List {
		for _, name := range field.Names {
			typeName := ExprName(field.Type)
			importPath := ""
			dotIndex := strings.Index(typeName, ".")
			isExported := unicode.IsUpper([]rune(typeName)[0])
			if dotIndex == -1 && isExported {
				typeName = pkgName + "." + typeName
			} else if dotIndex != -1 {
				// The type comes from may come from an imported package.
				argPkgName := typeName[:dotIndex]
				if importPath, ok = imports[argPkgName]; !ok {
					log.Println("Failed to find import for arg of type:", typeName)
				}
			}

			method.Args = append(method.Args, &MethodArg{
				Name:       name.Name,
				TypeName:   typeName,
				ImportPath: importPath,
			})
		}
	}

	// Add a description of the calls to Render from the method.
	// Inspect every node (e.g. always return true).
	method.RenderCalls = []*renderCall{}
	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		// Is it a function call?
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Is it calling (*Controller).Render?
		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// The type of the receiver is not easily available, so just store every
		// call to any method called Render.
		if selExpr.Sel.Name != "Render" {
			return true
		}

		// Add this call's args to the renderArgs.
		pos := fset.Position(callExpr.Rparen)
		renderCall := &renderCall{
			Line:  pos.Line,
			Names: []string{},
		}
		for _, arg := range callExpr.Args {
			argIdent, ok := arg.(*ast.Ident)
			if !ok {
				log.Println("Unnamed argument to Render call:", pos)
				continue
			}
			renderCall.Names = append(renderCall.Names, argIdent.Name)
		}
		method.RenderCalls = append(method.RenderCalls, renderCall)
		return true
	})

	mm[recvTypeName] = append(mm[recvTypeName], method)
}

func (s *ControllerSpec) SimpleName() string {
	return s.PackageName + "." + s.StructName
}

func (s *embeddedTypeName) SimpleName() string {
	return s.PackageName + "." + s.StructName
}

// Remove any types that do not (directly or indirectly) embed *play.Controller.
func filterControllers(specs []*ControllerSpec) (filtered []*ControllerSpec) {
	// Do a search in the "embedded type graph", starting with play.Controller.
	nodeQueue := []string{"play.Controller"}
	for len(nodeQueue) > 0 {
		controllerSimpleName := nodeQueue[0]
		nodeQueue = nodeQueue[1:]
		for _, spec := range specs {
			if play.ContainsString(nodeQueue, spec.SimpleName()) {
				continue // Already added
			}

			// Look through the embedded types to see if the current type is among them.
			for _, embeddedType := range spec.embeddedTypes {

				// If so, add this type's simple name to the nodeQueue, and its spec to
				// the filtered list.
				if controllerSimpleName == embeddedType.SimpleName() {
					nodeQueue = append(nodeQueue, spec.SimpleName())
					filtered = append(filtered, spec)
					break
				}
			}
		}
	}
	return
}

// This returns the syntactic expression for referencing this type in Go.
// One complexity is that package-local types have to be fully-qualified.
// For example, if the type is "Hello", then it really means "pkg.Hello".
func ExprName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return ExprName(t.X) + "." + ExprName(t.Sel)
	case *ast.StarExpr:
		return "*" + ExprName(t.X)
	default:
		ast.Print(nil, expr)
		panic("Failed to generate name for field.")
	}
	return ""
}
