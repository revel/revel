package harness

// This file handles the app code introspection.
// It catalogs the controllers, their methods, and their arguments.

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"log"
	"reflect"
	"os"
	"play"
)

type ControllerSpec struct {
	PackageName string
	StructName  string
	ImportPath  string
	MethodSpecs []*MethodSpec

	// Used internally to identify controllers that indirectly embed *play.Controller.
	embeddedTypes []*embeddedTypeName
}

type MethodSpec struct {
	Name string    // Name of the method, e.g. "Index"
	Args []string  // Argument names, in the order that they are accepted.
	ArgTypes []reflect.Type  // Argument types, parallel to Args.
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
				SourceType: ".go source",
				Title: "Go Compilation Error",
				Path: pos.Filename,
				Description: errList[0].Msg,
				Line: pos.Line,
				Column: pos.Column,
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

			// For each declaration in the source file...
			for _, decl := range file.Decls {

				// Match and add both structs and methods
				structSpecs = appendStruct(structSpecs, pkg, decl)
				appendMethod(methodSpecs, decl)
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
		StructName: spec.Name.Name,
		ImportPath: play.ImportPath + "/app/" + pkg.Name,
		// MethodSpecs: make([]*MethodSpec),
		// embeddedTypes: make([]*embeddedTypeName),
	}

	for _, field := range structType.Fields.List {
		// If field.Names is set, it's not an embedded type.
		if field.Names != nil {
			continue
		}

		// A direct "sub-type" has an ast.Field like:
		// StarExpr { SelectorExpr { "play", "Controller" } }
		starExpr, ok := field.Type.(*ast.StarExpr)
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

		controllerSpec.embeddedTypes = append(controllerSpec.embeddedTypes, &embeddedTypeName{
			PackageName: pkgIdent.Name,
			StructName: selectorExpr.Sel.Name,
		})
	}

	return append(specs, controllerSpec)
}

// If decl is a Method declaration, it is summarized and added to the array
// underneath its receiver type.
// e.g. "Login" => {MethodSpec, MethodSpec, ..}
func appendMethod(mm methodMap, decl ast.Decl) {
	// Func declaration?
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		return
	}

	// Have a receiver?
	if funcDecl.Recv == nil {
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

	mm[recvTypeName] = append(mm[recvTypeName], &MethodSpec{
		Name: funcDecl.Name.Name,
	})

	// var args []string
	// funcType := funcDecl.Type.(*ast.FuncType)
	// for _, field := range funcType.Params.List {
	// 	Args = append(args,
	// }
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
	nodeQueue := []string {"play.Controller"}
	for _, controllerSimpleName := range nodeQueue {
		for _, spec := range specs {
			if play.ContainsString(nodeQueue, spec.SimpleName()) {
				continue  // Already added
			}

			// Look through the embedded types to see if the current type is among them.
			for _, embeddedType := range spec.embeddedTypes {

				// If so, add this type's simple name to the nodeQueue, and it's spec to
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
