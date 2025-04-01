package clusterfuc

import "errors"

var (
	ErrExceededMaxToolCount = errors.New("exceeded max tool count")
	ErrToolAlreadyExists    = errors.New("invalid args were passed")
)
