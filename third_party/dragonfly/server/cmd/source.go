package cmd

type Source interface {
	Target

	SendCommandOutput(o *Output)
}
