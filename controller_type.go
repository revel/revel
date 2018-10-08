package revel

import (
	"reflect"
	"strings"
)

// Controller registry and types.
type ControllerType struct {
	Namespace         string  // The namespace of the controller
	ModuleSource      *Module // The module for the controller
	Type              reflect.Type
	Methods           []*MethodType
	ControllerIndexes [][]int // FieldByIndex to all embedded *Controllers
	ControllerEvents  *ControllerTypeEvents
}
type ControllerTypeEvents struct {
	Before, After, Finally, Panic []*ControllerFieldPath
}

// The controller field path provides the caller the ability to invoke the call
// directly
type ControllerFieldPath struct {
	IsPointer      bool
	FieldIndexPath []int
	FunctionCall   reflect.Value
}

type MethodType struct {
	Name           string
	Args           []*MethodArg
	RenderArgNames map[int][]string
	lowerName      string
	Index          int
}

type MethodArg struct {
	Name string
	Type reflect.Type
}

// Adds the controller to the controllers map using its namespace, also adds it to the module list of controllers.
// If the controller is in the main application it is added without its namespace as well.
func AddControllerType(moduleSource *Module, controllerType reflect.Type, methods []*MethodType) (newControllerType *ControllerType) {
	if moduleSource == nil {
		moduleSource = appModule
	}

	newControllerType = &ControllerType{ModuleSource: moduleSource, Type: controllerType, Methods: methods, ControllerIndexes: findControllers(controllerType)}
	newControllerType.ControllerEvents = NewControllerTypeEvents(newControllerType)
	newControllerType.Namespace = moduleSource.Namespace()
	controllerName := newControllerType.Name()

	// Store the first controller only in the controllers map with the unmapped namespace.
	if _, found := controllers[controllerName]; !found {
		controllers[controllerName] = newControllerType
		newControllerType.ModuleSource.AddController(newControllerType)
		if newControllerType.ModuleSource == appModule {
			// Add the controller mapping into the global namespace
			controllers[newControllerType.ShortName()] = newControllerType
		}
	} else {
		controllerLog.Errorf("Error, attempt to register duplicate controller as %s", controllerName)
	}
	controllerLog.Debugf("Registered controller: %s", controllerName)

	return
}

// Method searches for a given exported method (case insensitive)
func (ct *ControllerType) Method(name string) *MethodType {
	lowerName := strings.ToLower(name)
	for _, method := range ct.Methods {
		if method.lowerName == lowerName {
			return method
		}
	}
	return nil
}

// The controller name with the namespace
func (ct *ControllerType) Name() string {
	return ct.Namespace + ct.ShortName()
}

// The controller name without the namespace
func (ct *ControllerType) ShortName() string {
	return strings.ToLower(ct.Type.Name())
}

func NewControllerTypeEvents(c *ControllerType) (ce *ControllerTypeEvents) {
	ce = &ControllerTypeEvents{}
	// Parse the methods for the controller type, assign any control methods
	checkType := c.Type
	ce.check(checkType, []int{})
	return
}

// Add in before after panic and finally, recursive call
// Befores are ordered in revers, everything else is in order of first encountered
func (cte *ControllerTypeEvents) check(theType reflect.Type, fieldPath []int) {
	typeChecker := func(checkType reflect.Type) {
		for index := 0; index < checkType.NumMethod(); index++ {
			m := checkType.Method(index)
			// Must be two arguments, the second returns the controller type
			// Go cannot differentiate between promoted methods and
			// embedded methods, this allows the embedded method to be run
			// https://github.com/golang/go/issues/21162
			if m.Type.NumOut() == 2 && m.Type.Out(1) == checkType {
				if checkType.Kind() == reflect.Ptr {
					controllerLog.Debug("Found controller type event method pointer", "name", checkType.Elem().Name(), "methodname", m.Name)
				} else {
					controllerLog.Debug("Found controller type event method", "name", checkType.Name(), "methodname", m.Name)
				}
				controllerFieldPath := newFieldPath(checkType.Kind() == reflect.Ptr, m.Func, fieldPath)
				switch strings.ToLower(m.Name) {
				case "before":
					cte.Before = append([]*ControllerFieldPath{controllerFieldPath}, cte.Before...)
				case "after":
					cte.After = append(cte.After, controllerFieldPath)
				case "panic":
					cte.Panic = append(cte.Panic, controllerFieldPath)
				case "finally":
					cte.Finally = append(cte.Finally, controllerFieldPath)
				}
			}
		}
	}

	// Check methods of both types
	typeChecker(theType)
	typeChecker(reflect.PtrTo(theType))

	// Check for any sub controllers, ignore any pointers to controllers revel.Controller
	for i := 0; i < theType.NumField(); i++ {
		v := theType.Field(i)

		switch v.Type.Kind() {
		case reflect.Struct:
			cte.check(v.Type, append(fieldPath, i))
		}
	}
}
func newFieldPath(isPointer bool, value reflect.Value, fieldPath []int) *ControllerFieldPath {
	return &ControllerFieldPath{IsPointer: isPointer, FunctionCall: value, FieldIndexPath: fieldPath}
}

func (fieldPath *ControllerFieldPath) Invoke(value reflect.Value, input []reflect.Value) (result []reflect.Value) {
	for _, index := range fieldPath.FieldIndexPath {
		// You can only fetch fields from non pointers
		if value.Type().Kind() == reflect.Ptr {
			value = value.Elem().Field(index)
		} else {
			value = value.Field(index)
		}
	}
	if fieldPath.IsPointer && value.Type().Kind() != reflect.Ptr {
		value = value.Addr()
	} else if !fieldPath.IsPointer && value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}

	return fieldPath.FunctionCall.Call(append([]reflect.Value{value}, input...))
}
