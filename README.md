# modden
mashup of kubectl, conftest, OPA and E2E

## Libraries

- kustomize
[kyaml](https://github.com/kubernetes-sigs/kustomize/tree/master/kyaml)
and
[kstatus](https://github.com/kubernetes-sigs/kustomize/tree/master/kstatus)
for dealing with YAML and generic Kubernetes objects

## Required features

- YAML
    - Apply and reconcile uninterpreted YAML objects.
    - Interpolate object names to ensure uniqueness for a test run.
    - Apply objects as a patch, shorthand or template to reduce test verbosity.
    - Patch existing objects.
    - Delete objects (by name or labels).

# References

- https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md
