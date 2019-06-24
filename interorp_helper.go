package why

import "github.com/d5/tengo/objects"

// ToError creates a tango error object from a error.
// This can be used inside of extension functions to
// quickly create a error that can be returned.
func ToError(err error) objects.Object {
	if err == nil {
		return nil
	}

	return &objects.Error{
		Value: &objects.String{
			Value: err.Error(),
		},
	}
}
