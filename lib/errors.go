package lib

type ParsedError struct {
	Message error
	Parsed  string
}

func (m ParsedError) Error() string {
	return m.Message.Error()
}
