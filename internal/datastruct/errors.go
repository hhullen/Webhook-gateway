package datastruct

import "errors"

var (
	ErrInvalidAuthToken  = errors.New("invalid auth token")
	ErrIncorrectTunnelId = errors.New("incorrect tunnel id")
)
