package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

			fragType, err := f.Decode()

			assert.Equal(t, tc.Want, fragType)

			// Errors must mart the fragment invalid.
			switch fragType {
			case FragmentTypeInvalid:
				assert.Error(t, err)
			default:
				assert.NoError(t, err)
			}

			switch fragType {
			case FragmentTypeUnknown, FragmentTypeInvalid:
				if f.Object() != nil {
					t.Errorf("non-nil object for unknown/invalid fragment")
				}
				if f.Rego() != nil {
					t.Errorf("non-nil module for unknown/invalid fragment")
				}
			case FragmentTypeObject:
				if f.Object() == nil {
					t.Errorf("nil object for object fragment")
				}
				if f.Rego() != nil {
					t.Errorf("non-nil module for object fragment")
				}
			case FragmentTypeModule:
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
		Want: FragmentTypeInvalid,
	})

	run(t, "non-object YAML", testcase{
		Data: `foo: "bar"`,
		Want: FragmentTypeInvalid,
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
		Want: FragmentTypeModule,
	})

	run(t, "Rego rule", testcase{
		Data: `t { x := 42; y := 41; x > y }`,
		Want: FragmentTypeModule,
	})
}
