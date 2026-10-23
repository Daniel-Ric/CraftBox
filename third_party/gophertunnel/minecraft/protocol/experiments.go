package protocol

type ExperimentData struct {
	Name string

	Enabled bool
}

func (x *ExperimentData) Marshal(r IO) {
	r.String(&x.Name)
	r.Bool(&x.Enabled)
}
