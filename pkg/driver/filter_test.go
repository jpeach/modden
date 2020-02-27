package driver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestMetaInjectionFilter(t *testing.T) {
	rn := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: httpbin
  template:
    metadata:
      labels:
        app.kubernetes.io/name: httpbin
    spec:
      containers:
      - image: docker.io/kennethreitz/httpbin
`)

	i := &MetaInjectionFilter{
		RunID:     "test-run-id",
		ManagedBy: "modden",
	}

	_, err := rn.Pipe(i)
	require.NoError(t, err)

	wanted := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  labels:
    app.kubernetes.io/managed-by: modden
  annotations:
    modden/run-id: test-run-id
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: httpbin
  template:
    metadata:
      labels:
        app.kubernetes.io/name: httpbin
      annotations:
        modden/run-id: test-run-id
    spec:
      containers:
      - image: docker.io/kennethreitz/httpbin
`)

	assert.Equal(t, rn.MustString(), wanted.MustString())
}
