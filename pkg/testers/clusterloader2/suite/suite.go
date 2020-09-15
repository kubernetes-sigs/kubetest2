package suite

type Suite struct {
	TestConfigs   []string
	TestOverrides []string
}

// GetSuite returns the default configurations for well-known testing setups.
func GetSuite(suite string) *Suite {
	const (
		load           = "load"
		density        = "density"
		nodeThroughput = "node-throughput"
	)

	var supportedSuites = map[string]*Suite{
		load: {
			TestConfigs: []string{
				"testing/load/config.yaml",
			},
		},

		density: {
			TestConfigs: []string{
				"testing/density/config.yaml",
			},
		},

		nodeThroughput: {
			TestConfigs: []string{
				"testing/node-throughput/config.yaml",
			},
		},
	}
	return supportedSuites[suite]
}
