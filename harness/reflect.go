package harness

// This file handles the app code introspection.
// It catalogs the controllers, their methods, and their arguments.

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/revel/revel"
)

// SourceInfo is the top-level struct containing all extracted information
// about the app source code, used to generate main.go.
type SourceInfo struct {
	// StructSpecs lists type info for all structs found under the code paths.
	// They may be queried to determine which ones (transitively) embed certain types.
	StructSpecs []*TypeInfo
	// ValidationKeys provides a two-level lookup.  The keys are:
	// 1. The fully-qualified function name,
	//    e.g. "github.com/revel/revel/samples/chat/app/controllers.(*Application).Action"
	// 2. Within that func's file, the line number of the (overall) expression statement.
	//    e.g. the line returned from runtime.Caller()
	// The result of the lookup the name of variable being validated.
	ValidationKeys map[string]map[int]string
	// A list of import paths.
	// Revel notices files with an init() function and imports that package.
	InitImportPaths []string

	// controllerSpecs lists type info for all structs found under
	// app/controllers/... that embed (directly or indirectly) revel.Controller
	controllerSpecs []*TypeInfo
	// testSuites list the types that constitute the set of application tests.
	testSuites []*TypeInfo
}

// TypeInfo summarizes information about a struct type in the app source code.
type TypeInfo struct {
	StructName  string // e.g. "Application"
	ImportPath  string // e.g. "github.com/revel/revel/samples/chat/app/controllers"
	PackageName string // e.g. "controllers"
	MethodSpecs []*MethodSpec

	// Used internally to identify controllers that indirectly embed *revel.Controller.
	embeddedTypes []*embeddedTypeName
}

// methodCall describes a call to c.Render(..)
// It documents the argument names used, in order to propagate them to RenderArgs.
type methodCall struct {
	Path  string // e.g. "myapp/app/controllers.(*Application).Action"
	Line  int
	Names []string
}

type MethodSpec struct {
	Name        string        // Name of the method, e.g. "Index"
	Args        []*MethodArg  // Argument descriptors
	RenderCalls []*methodCall // Descriptions of Render() invocations from this Method.
}

type MethodArg struct {
	Name       string   // Name of the argument.
	TypeExpr   TypeExpr // The name of the type, e.g. "int", "*pkg.UserType"
	ImportPath string   // If the arg is of an imported type, this is the import path.
}

type embeddedTypeName struct {
	ImportPath, StructName string
}

// Maps a controller simple name (e.g. "Login") to the methods for which it is a
// receiver.
type methodMap map[string][]*MethodSpec

// Parse the app controllers directory and return a list of the controller types found.
// Returns a CompileError if the parsing fails.
func ProcessSource(roots []string) (*SourceInfo, *revel.Error) {
	var (
		srcInfo      *SourceInfo
		compileError *revel.Error
	)

	for _, root := range roots {
		rootImportPath := importPathFromPath(root)
		if rootImportPath == "" {
			revel.WARN.Println("Skipping code path", root)
			continue
		}

		// Start walking the directory tree.
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Println("Error scanning app source:", err)
				return nil
			}

			if !info.IsDir() || info.Name() == "tmp" {
				return nil
			}

			// Get the import path of the package.
			pkgImportPath := rootImportPath
			if root != path {
				pkgImportPath = rootImportPath + "/" + filepath.ToSlash(path[len(root)+1:])
			}

			// Parse files within the path.
			var pkgs map[string]*ast.Package
			fset := token.NewFileSet()
			pkgs, err = parser.ParseDir(fset, path, func(f os.FileInfo) bool {
				return !f.IsDir() && !strings.HasPrefix(f.Name(), ".") && strings.HasSuffix(f.Name(), ".go")
			}, 0)
			if err != nil {
				if errList, ok := err.(scanner.ErrorList); ok {
					var pos token.Position = errList[0].Pos
					compileError = &revel.Error{
						SourceType:  ".go source",
						Title:       "Go Compilation Error",
						Path:        pos.Filename,
						Description: errList[0].Msg,
						Line:        pos.Line,
						Column:      pos.Column,
						SourceLines: revel.MustReadLines(pos.Filename),
					}
					return compileError
				}
				ast.Print(nil, err)
				log.Fatalf("Failed to parse dir: %s", err)
			}

			// Skip "main" packages.
			delete(pkgs, "main")

			// If there is no code in this directory, skip it.
			if len(pkgs) == 0 {
				return nil
			}

			// There should be only one package in this directory.
			if len(pkgs) > 1 {
				log.Println("Most unexpected! Multiple packages in a single directory:", pkgs)
			}

			var pkg *ast.Package
			for _, v := range pkgs {
				pkg = v
			}

			srcInfo = appendSourceInfo(srcInfo, processPackage(fset, pkgImportPath, path, pkg))
			return nil
		})
	}

	return srcInfo, compileError
}

func appendSourceInfo(srcInfo1, srcInfo2 *SourceInfo) *SourceInfo {
	if srcInfo1 == nil {
		return srcInfo2
	}

	srcInfo1.StructSpecs = append(srcInfo1.StructSpecs, srcInfo2.StructSpecs...)
	srcInfo1.InitImportPaths = append(srcInfo1.InitImportPaths, srcInfo2.InitImportPaths...)
	for k, v := range srcInfo2.ValidationKeys {
		if _, ok := srcInfo1.ValidationKeys[k]; ok {
			log.Println("Key conflict when scanning validation calls:", k)
			continue
		}
		srcInfo1.ValidationKeys[k] = v
	}
	return srcInfo1
}

func processPackage(fset *token.FileSet, pkgImportPath, pkgPath string, pkg *ast.Package) *SourceInfo {
	var (
		structSpecs     []*TypeInfo
		initImportPaths []string

		methodSpecs     = make(methodMap)
		validationKeys  = make(map[string]map[int]string)
		scanControllers = strings.HasSuffix(pkgImportPath, "/controllers") ||
			strings.Contains(pkgImportPath, "/controllers/")
		scanTests = strings.HasSuffix(pkgImportPath, "/tests") ||
			strings.Contains(pkgImportPath, "/tests/")
	)

	// For each source file in the package...
	for _, file := range pkg.Files {

		// Imports maps the package key to the full import path.
		// e.g. import "sample/app/models" => "models": "sample/app/models"
		imports := map[string]string{}

		// For each declaration in the source file...
		for _, decl := range file.Decls {
			addImports(imports, decl, pkgPath)

			if scanControllers {
				// Match and add both structs and methods
				structSpecs = appendStruct(structSpecs, pkgImportPath, pkg, decl, imports)
				appendAction(fset, methodSpecs, decl, pkgImportPath, pkg.Name, imports)
			} else if scanTests {
				structSpecs = appendStruct(structSpecs, pkgImportPath, pkg, decl, imports)
			}

			// If this is a func...
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				// Scan it for validation calls
				lineKeys := getValidationKeys(fset, funcDecl, imports)
				if len(lineKeys) > 0 {
					validationKeys[pkgImportPath+"."+getFuncName(funcDecl)] = lineKeys
				}

				// Check if it's an init function.
				if funcDecl.Name.Name == "init" {
					initImportPaths = []string{pkgImportPath}
				}
			}
		}
	}

	// Add the method specs to the struct specs.
	for _, spec := range structSpecs {
		spec.MethodSpecs = methodSpecs[spec.StructName]
	}

	return &SourceInfo{
		StructSpecs:     structSpecs,
		ValidationKeys:  validationKeys,
		InitImportPaths: initImportPaths,
	}
}

// getFuncName returns a name for this func or method declaration.
// e.g. "(*Application).SayHello" for a method, "SayHello" for a func.
func getFuncName(funcDecl *ast.FuncDecl) string {
	prefix := ""
	if funcDecl.Recv != nil {
		recvType := funcDecl.Recv.List[0].Type
		if recvStarType, ok := recvType.(*ast.StarExpr); ok {
			prefix = "(*" + recvStarType.X.(*ast.Ident).Name + ")"
		} else {
			prefix = recvType.(*ast.Ident).Name
		}
		prefix += "."
	}
	return prefix + funcDecl.Name.Name
}

func addImports(imports map[string]string, decl ast.Decl, srcDir string) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.IMPORT {
		return
	}

	for _, spec := range genDecl.Specs {
		importSpec := spec.(*ast.ImportSpec)
		var pkgAlias string
		if importSpec.Name != nil {
			pkgAlias = importSpec.Name.Name
			if pkgAlias == "_" {
				continue
			}
		}
		quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
		fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes

		// If the package was not aliased (common case), we have to import it
		// to see what the package name is.
		// TODO: Can improve performance here a lot:
		// 1. Do not import everything over and over again.  Keep a cache.
		// 2. Exempt the standard library; their directories always match the package name.
		// 3. Can use build.FindOnly and then use parser.ParseDir with mode PackageClauseOnly
		if pkgAlias == "" {
			pkg, err := build.Import(fullPath, srcDir, 0)
			if err != nil {
				// We expect this to happen for apps using reverse routing (since we
				// have not yet generated the routes).  Don't log that.
				if !strings.HasSuffix(fullPath, "/app/routes") {
					revel.TRACE.Println("Could not find import:", fullPath)
				}
				continue
			}
			pkgAlias = pkg.Name
		}

		imports[pkgAlias] = fullPath
	}
}

// If this Decl is a struct type definition, it is summarized and added to specs.
// Else, specs is returned unchanged.
func appendStruct(specs []*TypeInfo, pkgImportPath string, pkg *ast.Package, decl ast.Decl, imports map[string]string) []*TypeInfo {
	// Filter out non-Struct type declarations.
	spec, found := getStructTypeDecl(decl)
	if !found {
		return specs
	}
	structType := spec.Type.(*ast.StructType)

	// At this point we know it's a type declaration for a struct.
	// Fill in the rest of the info by diving into the fields.
	// Add it provisionally to the Controller list -- it's later filtered using field info.
	controllerSpec := &TypeInfo{
		StructName:  spec.Name.Name,
		ImportPath:  pkgImportPath,
		PackageName: pkg.Name,
	}

	for _, field := range structType.Fields.List {
		// If field.Names is set, it's not an embedded type.
		if field.Names != nil {
			continue
		}

		// A direct "sub-type" has an ast.Field as either:
		//   Ident { "AppController" }
		//   SelectorExpr { "rev", "Controller" }
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
				return "", ident.Name
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

		// Find the import path for this type.
		// If it was referenced without a package name, use the current package import path.
		// Else, look up the package's import path by name.
		var importPath string
		if pkgName == "" {
			importPath = pkgImportPath
		} else {
			var ok bool
			if importPath, ok = imports[pkgName]; !ok {
				log.Print("Failed to find import path for ", pkgName, ".", typeName)
				continue
			}
		}

		controllerSpec.embeddedTypes = append(controllerSpec.embeddedTypes, &embeddedTypeName{
			ImportPath: importPath,
			StructName: typeName,
		})
	}

	return append(specs, controllerSpec)
}

// If decl is a Method declaration, it is summarized and added to the array
// underneath its receiver type.
// e.g. "Login" => {MethodSpec, MethodSpec, ..}
func appendAction(fset *token.FileSet, mm methodMap, decl ast.Decl, pkgImportPath, pkgName string, imports map[string]string) {
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
	if !funcDecl.Name.IsExported() {
		return
	}

	// Does it return a Result?
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return
	}
	selExpr, ok := funcDecl.Type.Results.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if selExpr.Sel.Name != "Result" {
		return
	}
	if pkgIdent, ok := selExpr.X.(*ast.Ident); !ok || imports[pkgIdent.Name] != revel.REVEL_IMPORT_PATH {
		return
	}

	method := &MethodSpec{
		Name: funcDecl.Name.Name,
	}

	// Add a description of the arguments to the method.
	for _, field := range funcDecl.Type.Params.List {
		for _, name := range field.Names {
			var importPath string
			typeExpr := NewTypeExpr(pkgName, field.Type)
			if !typeExpr.Valid {
				return // We didn't understand one of the args.  Ignore this action. (Already logged)
			}
			if typeExpr.PkgName != "" {
				var ok bool
				if importPath, ok = imports[typeExpr.PkgName]; !ok {
					log.Println("Failed to find import for arg of type:", typeExpr.TypeName(""))
				}
			}
			method.Args = append(method.Args, &MethodArg{
				Name:       name.Name,
				TypeExpr:   typeExpr,
				ImportPath: importPath,
			})
		}
	}

	// Add a description of the calls to Render from the method.
	// Inspect every node (e.g. always return true).
	method.RenderCalls = []*methodCall{}
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
		methodCall := &methodCall{
			Line:  pos.Line,
			Names: []string{},
		}
		for _, arg := range callExpr.Args {
			argIdent, ok := arg.(*ast.Ident)
			if !ok {
				continue
			}
			methodCall.Names = append(methodCall.Names, argIdent.Name)
		}
		method.RenderCalls = append(method.RenderCalls, methodCall)
		return true
	})

	var recvTypeName string
	var recvType ast.Expr = funcDecl.Recv.List[0].Type
	if recvStarType, ok := recvType.(*ast.StarExpr); ok {
		recvTypeName = recvStarType.X.(*ast.Ident).Name
	} else {
		recvTypeName = recvType.(*ast.Ident).Name
	}

	mm[recvTypeName] = append(mm[recvTypeName], method)
}

// Scan app source code for calls to X.Y(), where X is of type *Validation.
//
// Recognize these scenarios:
// - "Y" = "Validation" and is a member of the receiver.
//   (The common case for inline validation)
// - "X" is passed in to the func as a parameter.
//   (For structs implementing Validated)
//
// The line number to which a validation call is attributed is that of the
// surrounding ExprStmt.  This is so that it matches what runtime.Callers()
// reports.
//
// The end result is that we can set the default validation key for each call to
// be the same as the local variable.
func getValidationKeys(fset *token.FileSet, funcDecl *ast.FuncDecl, imports map[string]string) map[int]string {
	var (
		lineKeys = make(map[int]string)

		// Check the func parameters and the receiver's members for the *revel.Validation type.
		validationParam = getValidationParameter(funcDecl, imports)
	)

	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		// e.g. c.Validation.Required(arg) or v.Required(arg)
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		// e.g. c.Validation.Required or v.Required
		funcSelector, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		switch x := funcSelector.X.(type) {
		case *ast.SelectorExpr: // e.g. c.Validation
			if x.Sel.Name != "Validation" {
				return true
			}

		case *ast.Ident: // e.g. v
			if validationParam == nil || x.Obj != validationParam {
				return true
			}

		default:
			return true
		}

		if len(callExpr.Args) == 0 {
			return true
		}

		// Given the validation expression, extract the key.
		key := callExpr.Args[0]
		switch expr := key.(type) {
		case *ast.BinaryExpr:
			// If the argument is a binary expression, take the first expression.
			// (e.g. c.Validation.Required(myName != ""))
			key = expr.X
		case *ast.UnaryExpr:
			// If the argument is a unary expression, drill in.
			// (e.g. c.Validation.Required(!myBool)
			key = expr.X
		case *ast.BasicLit:
			// If it's a literal, skip it.
			return true
		}

		if typeExpr := NewTypeExpr("", key); typeExpr.Valid {
			lineKeys[fset.Position(callExpr.End()).Line] = typeExpr.TypeName("")
		}
		return true
	})

	return lineKeys
}

// Check to see if there is a *revel.Validation as an argument.
func getValidationParameter(funcDecl *ast.FuncDecl, imports map[string]string) *ast.Object {
	for _, field := range funcDecl.Type.Params.List {
		starExpr, ok := field.Type.(*ast.StarExpr) // e.g. *revel.Validation
		if !ok {
			continue
		}

		selExpr, ok := starExpr.X.(*ast.SelectorExpr) // e.g. revel.Validation
		if !ok {
			continue
		}

		xIdent, ok := selExpr.X.(*ast.Ident) // e.g. rev
		if !ok {
			continue
		}

		if selExpr.Sel.Name == "Validation" && imports[xIdent.Name] == revel.REVEL_IMPORT_PATH {
			return field.Names[0].Obj
		}
	}
	return nil
}

func (s *TypeInfo) String() string {
	return s.ImportPath + "." + s.StructName
}

func (s *embeddedTypeName) String() string {
	return s.ImportPath + "." + s.StructName
}

// getStructTypeDecl checks if the given decl is a type declaration for a
// struct.  If so, the TypeSpec is returned.
func getStructTypeDecl(decl ast.Decl) (spec *ast.TypeSpec, found bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.TYPE {
		return
	}

	if len(genDecl.Specs) != 1 {
		revel.TRACE.Printf("Surprising: Decl does not have 1 Spec: %v", genDecl)
		return
	}

	spec = genDecl.Specs[0].(*ast.TypeSpec)
	if _, ok := spec.Type.(*ast.StructType); ok {
		found = true
	}

	return
}

// TypesThatEmbed returns all types that (directly or indirectly) embed the
// target type, which must be a fully qualified type name,
// e.g. "github.com/revel/revel.Controller"
func (s *SourceInfo) TypesThatEmbed(targetType string) (filtered []*TypeInfo) {
	// Do a search in the "embedded type graph", starting with the target type.
	var (
		nodeQueue = []string{targetType}
		processed []string
	)
	for len(nodeQueue) > 0 {
		controllerSimpleName := nodeQueue[0]
		nodeQueue = nodeQueue[1:]
		processed = append(processed, controllerSimpleName)

		// Look through all known structs.
		for _, spec := range s.StructSpecs {
			// If this one has been processed or is already in nodeQueue, then skip it.
			if revel.ContainsString(processed, spec.String()) ||
				revel.ContainsString(nodeQueue, spec.String()) {
				continue
			}

			// Look through the embedded types to see if the current type is among them.
			for _, embeddedType := range spec.embeddedTypes {

				// If so, add this type's simple name to the nodeQueue, and its spec to
				// the filtered list.
				if controllerSimpleName == embeddedType.String() {
					nodeQueue = append(nodeQueue, spec.String())
					filtered = append(filtered, spec)
					break
				}
			}
		}
	}
	return
}

func (s *SourceInfo) ControllerSpecs() []*TypeInfo {
	if s.controllerSpecs == nil {
		s.controllerSpecs = s.TypesThatEmbed(revel.REVEL_IMPORT_PATH + ".Controller")
	}
	return s.controllerSpecs
}

func (s *SourceInfo) TestSuites() []*TypeInfo {
	if s.testSuites == nil {
		s.testSuites = s.TypesThatEmbed(revel.REVEL_IMPORT_PATH + ".TestSuite")
	}
	return s.testSuites
}

// TypeExpr provides a type name that may be rewritten to use a package name.
type TypeExpr struct {
	Expr     string // The unqualified type expression, e.g. "[]*MyType"
	PkgName  string // The default package idenifier
	pkgIndex int    // The index where the package identifier should be inserted.
	Valid    bool
}

// TypeName returns the fully-qualified type name for this expression.
// The caller may optionally specify a package name to override the default.
func (e TypeExpr) TypeName(pkgOverride string) string {
	pkgName := revel.FirstNonEmpty(pkgOverride, e.PkgName)
	if pkgName == "" {
		return e.Expr
	}
	return e.Expr[:e.pkgIndex] + pkgName + "." + e.Expr[e.pkgIndex:]
}

// This returns the syntactic expression for referencing this type in Go.
func NewTypeExpr(pkgName string, expr ast.Expr) TypeExpr {
	switch t := expr.(type) {
	case *ast.Ident:
		if IsBuiltinType(t.Name) {
			pkgName = ""
		}
		return TypeExpr{t.Name, pkgName, 0, true}
	case *ast.SelectorExpr:
		e := NewTypeExpr(pkgName, t.X)
		return TypeExpr{t.Sel.Name, e.Expr, 0, e.Valid}
	case *ast.StarExpr:
		e := NewTypeExpr(pkgName, t.X)
		return TypeExpr{"*" + e.Expr, e.PkgName, e.pkgIndex + 1, e.Valid}
	case *ast.ArrayType:
		e := NewTypeExpr(pkgName, t.Elt)
		return TypeExpr{"[]" + e.Expr, e.PkgName, e.pkgIndex + 2, e.Valid}
	case *ast.Ellipsis:
		e := NewTypeExpr(pkgName, t.Elt)
		return TypeExpr{"[]" + e.Expr, e.PkgName, e.pkgIndex + 2, e.Valid}
	default:
		log.Println("Failed to generate name for field. Make sure the field name is valid.")
	}
	return TypeExpr{Valid: false}
}

var _BUILTIN_TYPES = map[string]struct{}{
	"bool":       struct{}{},
	"byte":       struct{}{},
	"complex128": struct{}{},
	"complex64":  struct{}{},
	"error":      struct{}{},
	"float32":    struct{}{},
	"float64":    struct{}{},
	"int":        struct{}{},
	"int16":      struct{}{},
	"int32":      struct{}{},
	"int64":      struct{}{},
	"int8":       struct{}{},
	"rune":       struct{}{},
	"string":     struct{}{},
	"uint":       struct{}{},
	"uint16":     struct{}{},
	"uint32":     struct{}{},
	"uint64":     struct{}{},
	"uint8":      struct{}{},
	"uintptr":    struct{}{},
}

func IsBuiltinType(name string) bool {
	_, ok := _BUILTIN_TYPES[name]
	return ok
}

func importPathFromPath(root string) string {
	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		srcPath := filepath.Join(gopath, "src")
		if strings.HasPrefix(root, srcPath) {
			return filepath.ToSlash(root[len(srcPath)+1:])
		}
	}

	srcPath := filepath.Join(build.Default.GOROOT, "src", "pkg")
	if strings.HasPrefix(root, srcPath) {
		revel.WARN.Println("Code path should be in GOPATH, but is in GOROOT:", root)
		return filepath.ToSlash(root[len(srcPath)+1:])
	}

	revel.ERROR.Println("Unexpected! Code path is not in GOPATH:", root)
	return ""
}
