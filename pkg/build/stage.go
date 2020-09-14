package build

type Stager interface {
	// Stage determines how kubernetes artifacts will be staged (e.g. to say a GCS bucket)
	Stage() error
}

type NoopStager struct{}

var _ Stager = &NoopStager{}

func (n *NoopStager) Stage() error {
	return nil
}
