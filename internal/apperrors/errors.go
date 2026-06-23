package apperrors

import "errors"

var (
	// Numerator errors
	ErrNumeratorMismatch             = errors.New("numerator mismatch")
	ErrUnableToObtainUniqueNumerator = errors.New("unable to obtain unique numerator")
	ErrNumeratorService              = errors.New("numerator service returned error")

	// JSON server / repository errors
	ErrJSONServerService = errors.New("json-server service returned error")
	ErrNotFound          = errors.New("resource not found")
)
