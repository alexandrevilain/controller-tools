package patch

import "reflect"

// isNil returns an error if the passed interface is equal to nil or if it has an interface value of nil.
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() { //nolint:exhaustive
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.Slice, reflect.Interface, reflect.UnsafePointer, reflect.Func:
		return reflect.ValueOf(i).IsValid() && reflect.ValueOf(i).IsNil()
	}
	return false
}
