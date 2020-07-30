package exec

import (
	"fmt"
	"os"

	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/pflag"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/process"
)

type Tester struct {
	argv []string
}

const usage = `kubetest2 --test=exec --  [TestCommand] [TestArgs]
  TestCommand: the command to invoke for testing
  TestArgs:    arguments passed to test command
`

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize tester: %v", err)
	}

	fs.Usage = func() {
		fmt.Print(usage)
	}

	if len(os.Args) < 2 {
		fs.Usage()
		return nil
	}

	// gracefully handle -h or --help if it is the only argument
	if len(os.Args) == 2 {
		// check for -h, --help
		fs.Init("", pflag.ContinueOnError)
		help := fs.BoolP("help", "h", false, "")
		// we don't care about errors, only if -h / --help was set
		if err := fs.Parse(os.Args); err != nil {
			return fmt.Errorf("failed to parse flags: %v", err)
		}
		if *help {
			fs.Usage()
			return nil
		}
	}

	t.argv = os.Args[1:]
	return t.Test()
}

func (t *Tester) Test() error {
	return process.ExecJUnit(t.argv[0], t.argv[1:], os.Environ())
}

func NewDefaultTester() *Tester {
	return &Tester{}
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run exec tester: %v", err)
	}
}
