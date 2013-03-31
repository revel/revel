package revel

import (
	"reflect"
	"strings"
)

var (
	controllerType    = reflect.TypeOf(Controller{})
	controllerPtrType = reflect.TypeOf(&Controller{})
)

func NewAppController(req *Request, resp *Response, controllerName, methodName string) (*Controller, reflect.Value) {
	var appControllerType *ControllerType = LookupControllerType(controllerName)
	if appControllerType == nil {
		INFO.Printf("Controller %s not found: %s", controllerName, req.URL)
		return nil, reflect.ValueOf(nil)
	}

	controller := NewController(req, resp, appControllerType)
	appControllerPtr := initNewAppController(appControllerType.Type, controller)

	// Set the method being called.
	controller.Action = controllerName + "." + methodName
	controller.AppController = appControllerPtr.Interface()
	controller.MethodType = appControllerType.Method(methodName)
	if controller.MethodType == nil {
		INFO.Println("Failed to find method", methodName, "on Controller",
			controllerName)
		return nil, reflect.ValueOf(nil)
	}

	return controller, appControllerPtr
}

// This is a helper that initializes (zeros) a new app controller value.
// Generally, everything is set to its zero value, except:
// 1. Embedded controller pointers are newed up.
// 2. The revel.Controller embedded type is set to the value provided.
// Returns a value representing a pointer to the new app controller.
func initNewAppController(appControllerType reflect.Type, c *Controller) reflect.Value {
	// It might be a multi-level embedding, so we have to create new controllers
	// at every level of the hierarchy.  To find the controllers, we follow every
	// anonymous field, using breadth-first search.
	appControllerPtr := reflect.New(appControllerType)
	valueQueue := []reflect.Value{appControllerPtr}
	for len(valueQueue) > 0 {
		// Get the next value and de-reference it if necessary.
		var (
			value    = valueQueue[0]
			elem     = value
			elemType = value.Type()
		)
		if elemType.Kind() == reflect.Ptr {
			elem = value.Elem()
			elemType = elem.Type()
		}
		valueQueue = valueQueue[1:]

		// Look at all the struct fields.
		for i := 0; i < elem.NumField(); i++ {
			// If this is not an anonymous field, skip it.
			structField := elemType.Field(i)
			if !structField.Anonymous {
				continue
			}

			fieldValue := elem.Field(i)
			fieldType := structField.Type

			// If it's a Controller, set it to the new instance.
			if fieldType == controllerPtrType {
				fieldValue.Set(reflect.ValueOf(c))
				continue
			}

			// Else, add it to the valueQueue, after instantiating (if necessary).
			if fieldValue.Kind() == reflect.Ptr {
				fieldValue.Set(reflect.New(fieldType.Elem()))
			}
			valueQueue = append(valueQueue, fieldValue)
		}
	}
	return appControllerPtr
}

func RenderError(req *Request, resp *Response, err error) {
	stubController(req, resp).RenderError(err).Apply(req, resp)
}

// This function is useful if there is no relevant Controller available.
// It writes the 404 response immediately.
func NotFound(req *Request, resp *Response, msg string) {
	stubController(req, resp).NotFound(msg).Apply(req, resp)
}

func stubController(req *Request, resp *Response) *Controller {
	return &Controller{
		Response: resp,
		RenderArgs: map[string]interface{}{
			"RunMode": RunMode,
			"DevMode": DevMode,
		},
	}
}

// Write the header (for now, just the status code).
// The status may be set directly by the application (c.Response.Status = 501).
// if it isn't, then fall back to the provided status code.
func (resp *Response) WriteHeader(defaultStatusCode int, defaultContentType string) {
	if resp.Status == 0 {
		resp.Status = defaultStatusCode
	}
	if resp.ContentType == "" {
		resp.ContentType = defaultContentType
	}
	resp.Out.Header().Set("Content-Type", resp.ContentType)
	resp.Out.WriteHeader(resp.Status)
}

// Internal bookeeping

type ControllerType struct {
	Type    reflect.Type
	Methods []*MethodType
}

type MethodType struct {
	Name           string
	Args           []*MethodArg
	RenderArgNames map[int][]string
	lowerName      string
}

type MethodArg struct {
	Name string
	Type reflect.Type
}

// Searches for a given exported method (case insensitive)
func (ct *ControllerType) Method(name string) *MethodType {
	lowerName := strings.ToLower(name)
	for _, method := range ct.Methods {
		if method.lowerName == lowerName {
			return method
		}
	}
	return nil
}

var controllers = make(map[string]*ControllerType)

// Register a Controller and its Methods with Revel.
func RegisterController(c interface{}, methods []*MethodType) {
	// De-star the controller type
	// (e.g. given TypeOf((*Application)(nil)), want TypeOf(Application))
	var t reflect.Type = reflect.TypeOf(c)
	var elem reflect.Type = t.Elem()

	// De-star all of the method arg types too.
	for _, m := range methods {
		m.lowerName = strings.ToLower(m.Name)
		for _, arg := range m.Args {
			arg.Type = arg.Type.Elem()
		}
	}

	controllers[strings.ToLower(elem.Name())] = &ControllerType{Type: elem, Methods: methods}
	TRACE.Printf("Registered controller: %s", elem.Name())
}

func LookupControllerType(name string) *ControllerType {
	return controllers[strings.ToLower(name)]
}
