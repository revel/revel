package play

import (
	"bytes"
	"errors"
	"reflect"
)

type Result interface {
	Apply(req *Request, resp *Response)
}

// Action methods return this result to request a template be rendered.
type RenderTemplateResult struct {
	Template   Template
	RenderArgs map[string]interface{}
	Response   *Response
}

func (r *RenderTemplateResult) Apply(req *Request, resp *Response) {
	// Render the template into a temporary buffer, to see if there was an error
	// rendering the template.  If not, then copy it into the response buffer.
	// TODO: It seems a shame to make a copy of everything, but if we don't,
	// template errors result in unpredictable HTML for error pages.
	var b bytes.Buffer
	err := r.Template.Render(&b, r.RenderArgs)
	if err != nil {
		line, description := parseTemplateError(err)
		compileError := CompileError{
			Title:       "Template Execution Error",
			Path:        r.Template.Name(),
			Description: description,
			Line:        line,
			SourceLines: r.Template.Content(),
			SourceType:  "template",
		}
		resp.out.Write([]byte(compileError.Html()))
		return
	}

	b.WriteTo(resp.out)
}
}

type RedirectResult struct {
	val interface{}
}

func (r *RedirectResult) Apply(req *Request, resp *Response) {
	url, err := getRedirectUrl(r.val)
	if err != nil {
		LOG.Println("Couldn't resolve redirect:", err.Error())
		resp.out.WriteHeader(500)
		return
	}
	resp.Headers.Set("Location", url)
	resp.out.WriteHeader(302)
}

func getRedirectUrl(item interface{}) (string, error) {
	// Handle strings
	if url, ok := item.(string); ok {
		return url, nil
	}

	// Handle funcs
	val := reflect.ValueOf(item)
	typ := reflect.TypeOf(item)
	if typ.Kind() == reflect.Func && typ.NumIn() > 0 {
		// Get the Controller Method
		recvType := typ.In(0)
		method := FindMethod(recvType, &val)
		if method == nil {
			return "", errors.New("couldn't find method")
		}

		// Construct the action string (e.g. "Controller.Method")
		if recvType.Kind() == reflect.Ptr {
			recvType = recvType.Elem()
		}
		action := recvType.Name() + "." + method.Name
		actionDef := router.Reverse(action, make(map[string]string))
		if actionDef == nil {
			return "", errors.New("no route for action " + action)
		}

		return actionDef.String(), nil
	}

	// Out of guesses
	return "", errors.New("didn't recognize type: " + typ.String())
}
