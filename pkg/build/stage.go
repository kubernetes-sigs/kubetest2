package build

type Stager interface {
	// Stage determines how kubernetes artifacts will be staged (e.g. to say a GCS bucket)
	// for the specified version
	Stage(version string) error
}

type NoopStager struct{}

var _ Stager = &NoopStager{}

func (n *NoopStager) Stage(string) error {
	return nil
}
