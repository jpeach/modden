package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/must"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

func NewGetCommand() *cobra.Command {
	get := &cobra.Command{
		Use:          "get",
		Short:        "Gets one of [objects, tests]",
		Long:         "Gets one of [objects, tests]",
		SilenceUsage: true,
	}

	objects := &cobra.Command{
		Use:   "objects",
		Short: "Gets one Kubernetes objects",
		Long: fmt.Sprintf(
			`Gets Kubernetes objects managed by tests

This command lists Kubernetes API objects that are labeled as managed
by modden. modden labels objects created or modified by test documents
with the %s%s%s label.`,
			"`", driver.LabelManagedBy, "`"),
		RunE: func(cmd *cobra.Command, args []string) error {
			kube, err := driver.NewKubeClient()
			if err != nil {
				return fmt.Errorf("failed to initialize Kubernetes context: %s", err)
			}

			results, err := kube.ListManagedObjects()
			if err != nil {
				log.Printf("%s", err)
				return err
			}

			if len(results) == 0 {
				return nil
			}

			now := metav1.Now()
			table := uitable.New()
			table.AddRow("NAMESPACE", "NAME", "RUN ID", "AGE")

			for _, r := range results {
				gk := r.GetObjectKind().GroupVersionKind().GroupKind()
				name := fmt.Sprintf("%s/%s", strings.ToLower(gk.String()), r.GetName())
				age := now.Sub(r.GetCreationTimestamp().UTC())

				table.AddRow(
					r.GetNamespace(),
					name,
					must.String(kube.RunIDFor(r)),
					duration.HumanDuration(age),
				)
			}

			fmt.Println(table)
			return nil
		},
	}

	get.AddCommand(CommandWithDefaults(objects))
	return CommandWithDefaults(get)
}
