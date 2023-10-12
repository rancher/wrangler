package data

type errorCode int

const (
	errorInvalidData errorCode = iota + 1
	errorInvalidField
	errorFieldValueNotFound
)

type DataError struct {
	message string
	code    errorCode
}

// Error returns the underlying error message, satisfying the error interface
func (d *DataError) Error() string {
	return d.message
}

func newDataError(message string, code errorCode) *DataError {
	return &DataError{
		message: message,
		code:    code,
	}
}

// IsInvalidDataError checks if a given error indicates that the provided data field was invalid.
func IsInvalidDataError(err error) bool {
	return checkErrTypeAndCode(err, errorInvalidData)
}

// IsInvalidFieldError checks if a given error indicates that the provided field was invalid.
func IsInvalidFieldError(err error) bool {
	return checkErrTypeAndCode(err, errorInvalidField)
}

// IsFieldValueNotFoundError checks if a given error indicates that the provided field was not found in the provided
// data.
func IsFieldValueNotFoundError(err error) bool {
	return checkErrTypeAndCode(err, errorFieldValueNotFound)
}

func checkErrTypeAndCode(err error, code errorCode) bool {
	dataErr, ok := err.(*DataError)
	if !ok {
		return false
	}
	return dataErr.code == code
}
