package revel

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-stack/stack"
	"github.com/revel/revel/logger"
)

// Module specific functions
type Module struct {
	Name, ImportPath, Path string
	ControllerTypeList     []*ControllerType
	Log                    logger.MultiLogger
	initializedModules     map[string]ModuleCallbackInterface
}

// Modules can be called back after they are loaded in revel by using this interface.
type ModuleCallbackInterface func(*Module)

// The namespace separator constant
const namespaceSeperator = `\` // (note cannot be . or : as this is already used for routes)

var (
	Modules   []*Module                                                                                     // The list of modules in use
	anyModule = &Module{}                                                                                   // Wildcard search for controllers for a module (for backward compatible lookups)
	appModule = &Module{Name: "App", initializedModules: map[string]ModuleCallbackInterface{}, Log: AppLog} // The app module
	moduleLog = RevelLog.New("section", "module")
)

// Called by a module init() function, caller will receive the *Module object created for that module
// This would be useful for assigning a logger for logging information in the module (since the module context would be correct)
func RegisterModuleInit(callback ModuleCallbackInterface) {
	// Store the module that called this so we can do a callback when the app is initialized
	// The format %+k is from go-stack/Call.Format and returns the package path
	key := fmt.Sprintf("%+k", stack.Caller(1))
	appModule.initializedModules[key] = callback
	if Initialized {
		RevelLog.Error("Application already initialized, initializing using app module", "key", key)
		callback(appModule)
	}
}

// Called on startup to make a callback so that modules can be initialized through the `RegisterModuleInit` function
func init() {
	AddInitEventHandler(func(typeOf Event, value interface{}) (responseOf EventResponse) {
		if typeOf == REVEL_BEFORE_MODULES_LOADED {
			Modules = []*Module{appModule}
			appModule.Path = filepath.ToSlash(AppPath)
			appModule.ImportPath = filepath.ToSlash(ImportPath)
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
	comparison := name
	if strings.Index(name, namespaceSeperator) < 0 {
		comparison = m.Namespace() + name
	}
	for _, c := range m.ControllerTypeList {
		if c.Name() == comparison {
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
func ModuleFromPath(packagePath string, addGopathToPath bool) (module *Module) {
	packagePath = filepath.ToSlash(packagePath)
	// The module paths will match the configuration module paths, so we will use those to determine them
	// Since the revel.Init is called first, then revel.Config exists and can be used to determine the module path

	// See if the path exists in the module based
	for i := range Modules {
		if strings.Index(packagePath, Modules[i].ImportPath)==0 {
			// This is a prefix, so the module is this module
			module = Modules[i]
			break
		}
		if module != nil {
			break
		}
	}
	// Default to the app module if not found
	if module == nil {
		module = appModule
	}
	return
}

// ModuleByName returns the module of the given name, if loaded, case insensitive.
func ModuleByName(name string) (*Module, bool) {
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
		moduleLog.Debug("Sorted keys", "keys", key)

	}
	for _, key := range keys {
		moduleImportPath := Config.StringDefault(key, "")
		if moduleImportPath == "" {
			continue
		}

		modulePath, err := ResolveImportPath(moduleImportPath)
		if err != nil {
			moduleLog.Error("Failed to load module.  Import of path failed", "modulePath", moduleImportPath, "error", err)
		}
		// Drop anything between module.???.<name of module>
		subKey := key[len("module."):]
		if index := strings.Index(subKey, "."); index > -1 {
			subKey = subKey[index+1:]
		}
		addModule(subKey, moduleImportPath, modulePath)
	}

	// Modules loaded, now show module path
	for key, callback := range appModule.initializedModules {
		if m := ModuleFromPath(key, false); m != nil {
			callback(m)
		} else {
			RevelLog.Error("Callback for non registered module initializing with application module", "modulePath", key)
			callback(appModule)
		}
	}
}

// called by `loadModules`, creates a new `Module` instance and appends it to the `Modules` list
func addModule(name, importPath, modulePath string) {
	if _, found := ModuleByName(name); found {
		moduleLog.Panic("Attempt to import duplicate module %s path %s aborting startup", "name", name, "path", modulePath)
	}
	Modules = append(Modules, &Module{Name: name,
		ImportPath: filepath.ToSlash(importPath),
		Path:       filepath.ToSlash(modulePath),
		Log:        RootLog.New("module", name)})
	if codePath := filepath.Join(modulePath, "app"); DirExists(codePath) {
		CodePaths = append(CodePaths, codePath)
		if viewsPath := filepath.Join(modulePath, "app", "views"); DirExists(viewsPath) {
			TemplatePaths = append(TemplatePaths, viewsPath)
		}
	}

	moduleLog.Debug("Loaded module ", "module", filepath.Base(modulePath))

	// Hack: There is presently no way for the testrunner module to add the
	// "test" subdirectory to the CodePaths.  So this does it instead.
	if importPath == Config.StringDefault("module.testrunner", "github.com/revel/modules/testrunner") {
		joinedPath := filepath.Join(BasePath, "tests")
		moduleLog.Debug("Found testrunner module, adding `tests` path ", "path", joinedPath)
		CodePaths = append(CodePaths, joinedPath)
	}
	if testsPath := filepath.Join(modulePath, "tests"); DirExists(testsPath) {
		moduleLog.Debug("Found tests path ", "path", testsPath)
		CodePaths = append(CodePaths, testsPath)
	}
}
