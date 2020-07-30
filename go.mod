module sigs.k8s.io/kubetest2

go 1.14

replace k8s.io/client-go => k8s.io/client-go v0.17.3

require (
	github.com/aws/aws-k8s-tester v1.0.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/octago/sflags v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	k8s.io/client-go v11.0.0+incompatible //indirect
	k8s.io/klog v1.0.0
	k8s.io/test-infra v0.0.0-20200617221206-ea73eaeab7ff
	sigs.k8s.io/boskos v0.0.0-20200710214748-f5935686c7fc
)
