// Package validator is gin request parameter check library.
package validator

import (
	"reflect"
	"sync"

	valid "github.com/go-playground/validator/v10"
)

// Init validator instance, used to gin request parameter check
func Init() *CustomValidator {
	v := NewCustomValidator()
	v.Engine()
	return v
}

// CustomValidator Custom valid objects
type CustomValidator struct {
	once     sync.Once
	Validate *valid.Validate
}

// NewCustomValidator Instantiate
func NewCustomValidator() *CustomValidator {
	return &CustomValidator{}
}

// ValidateStruct validates a struct or slice/array
func (v *CustomValidator) ValidateStruct(obj interface{}) error {
	if obj == nil {
		return nil
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		if err := v.Validate.Struct(obj); err != nil {
			return err
		}

	case reflect.Pointer:
		// pointer type: if nil, no validation required; otherwise recursive validation after dereference
		if val.IsNil() {
			return nil
		}
		return v.ValidateStruct(val.Elem().Interface())

	case reflect.Slice, reflect.Array:
		// slice or array type: iterates over each element, recursively validating one by one
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			if err := v.ValidateStruct(elem.Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}

// Engine set tag name "binding", which is implementing the validator interface of the gin framework
func (v *CustomValidator) Engine() interface{} {
	v.lazyInit()
	return v.Validate
}

func (v *CustomValidator) lazyInit() {
	v.once.Do(func() {
		v.Validate = valid.New()
		v.Validate.SetTagName("binding")
	})
}
