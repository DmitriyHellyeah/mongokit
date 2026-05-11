package mongokit

import "errors"

var (
	ErrNilFilter             = errors.New("filter cannot be nil")
	ErrNilUpdate             = errors.New("update cannot be nil")
	ErrNilPipeline           = errors.New("pipeline cannot be nil")
	ErrNilID                 = errors.New("id cannot be nil")
	ErrEmptySlice            = errors.New("documents slice is empty")
ErrUnsupportedUpdateType = errors.New("unsupported update type")
)
