package revel

import (
	"go/build"
	"path/filepath"
	"sort"
	"strings"
)

// Module specific functions
type Module struct {
	Name, ImportPath, Path string
	ControllerTypeList     []*ControllerType
}

// The namespace separator constant
const namespaceSeperator = `\` // (note cannot be . or : as this is already used for routes)

var (
	Modules   []*Module              // The list of modules in use
	anyModule = &Module{}            // Wildcard search for controllers for a module (for backward compatible lookups)
	appModule = &Module{Name: "App"} // The app module
)

func init() {
	AddInitEventHandler(func(typeOf int, value interface{}) (responseOf int) {
		if typeOf == REVEL_BEFORE_MODULES_LOADED {
			Modules = []*Module{}
		}
		return
	})
}

// Returns the namespace for the module in the format `module_name|`
func (m *Module) Namespace() (namespace string) {
	namespace = m.Name + namespaceSeperator
	return
}

// Returns the named controller and action that is in this module
func (m *Module) ControllerByName(name, action string) (ctype *ControllerType) {
	comparision := name
	if strings.Index(name, namespaceSeperator) < 0 {
		comparision = m.Namespace() + name
	}
	for _, c := range m.ControllerTypeList {
		if c.Name() == comparision {
			ctype = c
			break
		}
	}
	return
}

// Adds the controller type to this module
func (m *Module) AddController(ct *ControllerType) {
	m.ControllerTypeList = append(m.ControllerTypeList, ct)
}

// Based on the full path given return the relevant module
// Only to be used on initialization
func ModuleFromPath(path string, addGopathToPath bool) (module *Module) {
	gopathList := filepath.SplitList(build.Default.GOPATH)

	// See if the path exists in the module based
	for i := range Modules {
		if addGopathToPath {
			for _, gopath := range gopathList {
				if strings.HasPrefix(gopath+"/src/"+path, Modules[i].Path) {
					module = Modules[i]
					break
				}
			}
		} else {
			if strings.HasPrefix(path, Modules[i].Path) {
				module = Modules[i]
				break
			}

		}

		if module != nil {
			break
		}
	}
	return
}

// ModuleByName returns the module of the given name, if loaded, case insensitive.
func ModuleByName(name string) (m *Module, found bool) {
	// If the name ends with the namespace separator remove it
	if name[len(name)-1] == []byte(namespaceSeperator)[0] {
		name = name[:len(name)-1]
	}
	name = strings.ToLower(name)
	if name == strings.ToLower(appModule.Name) {
		return appModule, true
	}
	for _, module := range Modules {
		if strings.ToLower(module.Name) == name {
			return module, true
		}
	}
	return nil, false
}

// Loads the modules specified in the config
func loadModules() {
	keys := []string{}
	for _, key := range Config.Options("module.") {
		keys = append(keys, key)
	}

	// Reorder module order by key name, a poor mans sort but at least it is consistent
	sort.Strings(keys)
	for _, key := range keys {
		INFO.Println("Sorted keys", key)

	}
	for _, key := range keys {
		moduleImportPath := Config.StringDefault(key, "")
		if moduleImportPath == "" {
			continue
		}

		modulePath, err := ResolveImportPath(moduleImportPath)
		if err != nil {
			ERROR.Fatalln("Failed to load module.  Import of", moduleImportPath, "failed:", err)
		}
		// Drop anything between module.???.<name of module>
		subKey := key[len("module."):]
		if index := strings.Index(subKey, "."); index > -1 {
			subKey = subKey[index+1:]
		}
		addModule(subKey, moduleImportPath, modulePath)
	}
}

//
func addModule(name, importPath, modulePath string) {
	if _, found := ModuleByName(name); found {
		ERROR.Panicf("Attempt to import duplicate module %s path %s aborting startup", name, modulePath)
	}
	Modules = append(Modules, &Module{Name: name, ImportPath: importPath, Path: modulePath})
	if codePath := filepath.Join(modulePath, "app"); DirExists(codePath) {
		CodePaths = append(CodePaths, codePath)
		if viewsPath := filepath.Join(modulePath, "app", "views"); DirExists(viewsPath) {
			TemplatePaths = append(TemplatePaths, viewsPath)
		}
	}

	INFO.Print("Loaded module ", filepath.Base(modulePath))

	// Hack: There is presently no way for the testrunner module to add the
	// "test" subdirectory to the CodePaths.  So this does it instead.
	if importPath == Config.StringDefault("module.testrunner", "github.com/revel/modules/testrunner") {
		INFO.Print("Found testrunner module, adding `tests` path ", filepath.Join(BasePath, "tests"))
		CodePaths = append(CodePaths, filepath.Join(BasePath, "tests"))
	}
	if testsPath := filepath.Join(modulePath, "tests"); DirExists(testsPath) {
		INFO.Print("Found tests path ", testsPath)
		CodePaths = append(CodePaths, testsPath)
	}
}
