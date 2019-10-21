package cache

// OpError :
type OpError struct {
	Op  string
	err error
}

func (e *OpError) Error() string {
	if e == nil {
		return "<nil>"
	}
	s := e.Op
	if e.err != nil {
		s += ": " + e.err.Error()
	} else {
		s = "error on " + s
	}
	return s
}

// Unwrap :
func (e *OpError) Unwrap() error {
	return e.err
}
