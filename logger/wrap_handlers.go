package logger

// FuncHandler returns a Handler that logs records with the given
// function.
import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Function handler wraps the declared function and returns the handler for it
func FuncHandler(fn func(r *Record) error) LogHandler {
	return funcHandler(fn)
}

// The type decleration for the function
type funcHandler func(r *Record) error

// The implementation of the Log
func (h funcHandler) Log(r *Record) error {
	return h(r)
}

// This function allows you to do a full declaration for the log,
// it is recommended you use FuncHandler instead
func HandlerFunc(log func(message string, time time.Time, level LogLevel, call CallStack, context ContextMap) error) LogHandler {
	return remoteHandler(log)
}

// The type used for the HandlerFunc
type remoteHandler func(message string, time time.Time, level LogLevel, call CallStack, context ContextMap) error

// The Log implementation
func (c remoteHandler) Log(record *Record) error {
	return c(record.Message, record.Time, record.Level, record.Call, record.Context)
}

// SyncHandler can be wrapped around a handler to guarantee that
// only a single Log operation can proceed at a time. It's necessary
// for thread-safe concurrent writes.
func SyncHandler(h LogHandler) LogHandler {
	var mu sync.Mutex
	return FuncHandler(func(r *Record) error {
		defer mu.Unlock()
		mu.Lock()
		return h.Log(r)
	})
}

// LazyHandler writes all values to the wrapped handler after evaluating
// any lazy functions in the record's context. It is already wrapped
// around StreamHandler and SyslogHandler in this library, you'll only need
// it if you write your own Handler.
func LazyHandler(h LogHandler) LogHandler {
	return FuncHandler(func(r *Record) error {
		for k, v := range r.Context {
			if lz, ok := v.(Lazy); ok {
				value, err := evaluateLazy(lz)
				if err != nil {
					r.Context[errorKey] = "bad lazy " + k
				} else {
					v = value
				}
			}
		}

		return h.Log(r)
	})
}

func evaluateLazy(lz Lazy) (interface{}, error) {
	t := reflect.TypeOf(lz.Fn)

	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("INVALID_LAZY, not func: %+v", lz.Fn)
	}

	if t.NumIn() > 0 {
		return nil, fmt.Errorf("INVALID_LAZY, func takes args: %+v", lz.Fn)
	}

	if t.NumOut() == 0 {
		return nil, fmt.Errorf("INVALID_LAZY, no func return val: %+v", lz.Fn)
	}

	value := reflect.ValueOf(lz.Fn)
	results := value.Call([]reflect.Value{})
	if len(results) == 1 {
		return results[0].Interface(), nil
	} else {
		values := make([]interface{}, len(results))
		for i, v := range results {
			values[i] = v.Interface()
		}
		return values, nil
	}
}
