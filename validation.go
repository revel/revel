package play

type ValidationError struct {
	Message, Key string
}

// Returns the Message.
func (e *ValidationError) String() string {
	return e.Message
}


// A Validation context manages data validation and error messages.
type Validation struct {
	Errors []*ValidationError
	keep bool
}

func (v *Validation) Keep() {
	v.keep = true
}

func (v *Validation) Clear() {
	v.Errors = []*ValidationError{}
}

func (v *Validation) HasErrors() bool {
	return len(v.Errors) > 0
}

// A ValidationResult is returned from every validation method.
// It provides an indication of success, and a pointer to the Error (if any).
type ValidationResult struct {
	Error *ValidationError
	Ok bool
}

func (r *ValidationResult) Key(key string) *ValidationResult {
	if r.Error != nil {
		r.Error.Key = key
	}
	return r
}

func (r *ValidationResult) Message(message string) *ValidationResult {
	if r.Error != nil {
		r.Error.Message = message
	}
	return r
}

type Check interface {
	IsSatisfied(interface{}) bool
}

type Required struct{}

func (r Required) IsSatisfied(obj interface{})  bool {
	if obj == nil {
		return false
	}

	if str, ok := obj.(string); ok {
		return len(str) > 0
	}
	if list, ok := obj.([]interface{}); ok {
		return len(list) > 0
	}
	if b, ok := obj.(bool); ok {
		return b
	}
	if i, ok := obj.(int); ok {
		return i > 0
	}
	return true
}

// Test that the argument is non-nil and non-empty (if string or list)
func (v *Validation) Required(obj interface{}) *ValidationResult {
	return v.check(Required{}, obj)
}

func (v *Validation) check(chk Check, obj interface{}) *ValidationResult {
	if chk.IsSatisfied(obj) {
		return &ValidationResult{Ok: true}
	}

	// Add the error to the validation context.
	err := &ValidationError{}
	v.Errors = append(v.Errors, err)

	// Also return it in the result.
	return &ValidationResult{
		Ok: false,
		Error: err,
	}
}
