package clusterfuc

import "errors"

var (
	ErrExceededMaxToolCount = errors.New("exceeded max tool count")
	ErrToolAlreadyExists    = errors.New("invalid args were passed")
	ErrAgentClientInvalid   = errors.New("invalid client was passed")
	ErrAgentOptInvalid      = errors.New("invalid agent option was passed")
)
