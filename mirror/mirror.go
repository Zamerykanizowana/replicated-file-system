package mirror

type Mirror struct{}

func (m *Mirror) Mirror(content []byte) error {
	return nil
}

func (m *Mirror) Consult() bool {
	return true
}
