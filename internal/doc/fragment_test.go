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
				t.Errorf(diff)
			}

			switch fragType {
			case FragmentTypeUnknown:
				if f.Object() != nil {
					t.Errorf("non-nil object for unknown fragment")
				}
				if f.Rego() != nil {
					t.Errorf("non-nil module for unknown fragment")
				}
			case FragmentTypeObject:
				if f.Object() == nil {
					t.Errorf("nil object for object fragment")
				}
				if f.Rego() != nil {
					t.Errorf("non-nil module for object fragment")
				}
			case FragmentTypeRego:
				if f.Object() != nil {
					t.Errorf("non-nil object for rego fragment")
				}
				if f.Rego() == nil {
					t.Errorf("nil module for rego fragment")
				}
			default:
				t.Errorf("invalid fragment type %d", fragType)
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

	run(t, "Rego composite value", testcase{
		Data: `
		rect := {"width": 2, "height": 4}`,
		Want: FragmentTypeRego,
	})

	run(t, "Rego rule", testcase{
		Data: `t { x := 42; y := 41; x > y }`,
		Want: FragmentTypeRego,
	})

	run(t, "Rego module", testcase{
		Data: ` t { x := 42; y := 41; x > y } `,
		Want: FragmentTypeRego,
	})

}
