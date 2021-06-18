module sigs.k8s.io/kubetest2

go 1.14

require (
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.1.4
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/octago/sflags v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/math v0.0.0-20141027224758-f2ed9e40e245
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog v1.0.0
	k8s.io/release v0.7.1-0.20210204090829-09fb5e3883b8
	k8s.io/test-infra v0.0.0-20210618033204-5173d0d741f3
	sigs.k8s.io/boskos v0.0.0-20200710214748-f5935686c7fc
)

// Required for importing k8s.io/test-infra.
replace k8s.io/client-go => k8s.io/client-go v0.21.1
