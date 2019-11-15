package doc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFragment(t *testing.T) {
	type testcase struct {
		Data string
		Want FragmentType
	}

	run := func(t *testing.T, name string, tc testcase) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()

			f := Fragment{
				Bytes: []byte(tc.Data),
			}

			fragType := f.Decode()

			if diff := cmp.Diff(tc.Want, fragType); diff != "" {
				t.Fatalf(diff)
			}
		})
	}

	run(t, "empty", testcase{
		Data: "",
		Want: FragmentTypeUnknown,
	})

	run(t, "non-object JSON", testcase{
		Data: `{ "foo": "bar"}`,
		Want: FragmentTypeUnknown,
	})

	run(t, "non-object YAML", testcase{
		Data: `foo: "bar"`,
		Want: FragmentTypeUnknown,
	})

	run(t, "YAML K8s object", testcase{
		Data: `
apiVersion: v1
kind: Namespace
metadata:
  name: projectcontour-monitoring
  labels:
    app: projectcontour-monitoring
    `,
		Want: FragmentTypeObject,
	})

	run(t, "JSON K8s object", testcase{
		Data: `
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "projectcontour-monitoring",
    "labels": {
      "app": "projectcontour-monitoring"
    }
  }
}
    `,
		Want: FragmentTypeObject,
	})

}
