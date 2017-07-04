package revel

import (
	"sort"
	"strings"
	"path/filepath"
	"go/build"
)

// Module specific functions
type Module struct {
	Name, ImportPath, Path string
	ControllerTypeList []*ControllerType
}

const namespaceSeperator = "|" // ., : are already used

var (
	anyModule = &Module{}
	appModule = &Module{Name:"App"}
)

// Returns the namespace for the module in the format `module_name|`
func (m *Module) Namespace() (namespace string) {
	namespace = m.Name + namespaceSeperator
	return
}

// Returns the named controller and action that is in this module
func (m *Module) ControllerByName(name,action string)(ctype *ControllerType) {
	comparision := name
	if strings.Index(name,namespaceSeperator)<0 {
		comparision =  m.Namespace() + name
	}
	for _,c := range m.ControllerTypeList {
		if strings.Index(c.Name(),comparision)>-1 {
			ctype = c
			break
		}
	}
	return
}
func (m *Module) AddController(ct *ControllerType) {
	m.ControllerTypeList = append(m.ControllerTypeList,ct)
}


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

// Based on the full path given return the relevant module
// Only be used on initialization
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

		if module!=nil {
			break
		}
	}
	return
}

