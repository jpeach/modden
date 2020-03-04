package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/must"
	"github.com/jpeach/modden/pkg/test"
	"github.com/jpeach/modden/pkg/utils"

	"github.com/spf13/cobra"
)

// NewRunCommand returns a command ro run a test case.
func NewRunCommand() *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run a set of test documents",
		Long: `Execute a set of test documents given as arguments.

Test documents are ordered fragments of YAML object and Rego checks,
separated by the YAML document separator, '---'. The fragments in
the test document are executed sequentially.

If a Kubernetes object specifies a target namespace in its metadata,
modden will implicitly create and manage that namespace. This reduces
test verbosity be not requiring namespace YAML fragments.

When modden creates Kubernetes objects, it uses the current default
Kubernetes client context. Each Kubernetes object it creates is labeled
with the 'app.kubernetes.io/managed-by=modden' label. Objects are also
annotated with a unique test run ID under the key 'modden/run-id'

Unless the '--preserve' flag is specified, modden will automatically
delete all the Kubernetes objects it created at the end of each test.

Since both Kubernetes and the services in a cluster are eventually
consistent, checks are executed repeatedly until they succeed or
until the timeout given by the '--check-timeout' flag expires.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			traceFlags := strings.Split(must.String(cmd.Flags().GetString("trace")), ",")

			kube, err := driver.NewKubeClient()
			if err != nil {
				return fmt.Errorf("failed to initialize Kubernetes context: %s", err)
			}

			opts := []test.RunOpt{
				test.KubeClientOpt(kube),
				test.CheckTimeoutOpt(must.Duration(cmd.Flags().GetDuration("check-timeout"))),
			}

			if must.Bool(cmd.Flags().GetBool("preserve")) {
				opts = append(opts, test.PreserveObjectsOpt())
			}

			if must.Bool(cmd.Flags().GetBool("dry-run")) {
				opts = append(opts, test.DryRunOpt())
			}

			if utils.ContainsString(traceFlags, "rego") {
				opts = append(opts, test.TraceRegoOpt())
			}

			// TODO(jpeach): set user agent from program version.
			kube.SetUserAgent("modden/TODO")

			for _, a := range args {
				testDoc, err := doc.ReadFile(a)
				if err != nil {
					return err
				}

				log.Printf("reading document with %d parts from %s",
					len(testDoc.Parts), a)

				// Before executing anything, verify that we can decode all the
				// fragments and raise any syntax errors.
				for i := range testDoc.Parts {
					p := &testDoc.Parts[i]

					fragType, err := p.Decode()
					if err == nil {
						log.Printf("decoded fragment %d as %s", i, fragType)
						continue
					}

					log.Printf("error on %s fragment %d: %s", fragType, i, err)

					// If we have a compile error, puke it.
					if err := doc.AsRegoCompilationErr(err); err != nil {
						// TODO(jpeach): rewrite the location
						// of the Rego error. The line number
						// will be relative to the start of the
						// fragment, and we should make it
						// relative to the start of the document.
						return err
					}

					return err
				}

				if err := test.Run(testDoc, opts...); err != nil {
					return fmt.Errorf("test run failed: %s", err)
				}
			}

			return nil
		},
	}

	run.Flags().String("trace", "", "Set execution tracing flags")
	run.Flags().Bool("preserve", false, "Don't automatically delete Kubernetes objects")
	run.Flags().Bool("dry-run", false, "Don't actually create Kubernetes objects")
	run.Flags().Duration("check-timeout", time.Second*30, "Timeout for evaluating check steps")

	return CommandWithDefaults(run)
}
