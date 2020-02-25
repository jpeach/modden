package driver

import (
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSpecialOpsFilter(t *testing.T) {
	specialOps := SpecialOpsFilter{}
	rn, err := yaml.MustParse(`
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: httpbin
$special: special value
`).Pipe(&specialOps)

	require.NoError(t, err)
	assert.Equal(t, specialOps.Ops, map[string]string{
		"$special": "special value",
	})

	// Verify that we removed the special node.
	assert.Equal(t,
		strings.TrimSpace(rn.MustString()),
		strings.TrimSpace(`apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: httpbin`))
}
