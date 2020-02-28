# modden
mashup of kubectl, conftest, OPA and E2E

## Libraries

- kustomize
[kyaml](https://github.com/kubernetes-sigs/kustomize/tree/master/kyaml)
and
[kstatus](https://github.com/kubernetes-sigs/kustomize/tree/master/kstatus)
for dealing with YAML and generic Kubernetes objects
- [OPA](https://pkg.go.dev/github.com/open-policy-agent/opa) for writing checks

## Required features

- YAML
    - [X] Apply uninterpreted YAML objects.
    - [ ] Reconcile (i.e. report status) for uninterpreted YAML objects.
    - [ ] Interpolate object names to ensure uniqueness for a test run.
    - [ ] Apply objects as a patch, shorthand or template to reduce test verbosity.
    - [X] Patch existing objects.
    - [ ] Delete objects (by name or labels).
    - [ ] Take a path to a directory of YAML objects that can be used as defaults.

- Logging
    - [ ] Global logging wrappers package.
    - [ ] Log messages implicitly attached to current test step.
    - [ ] Adjustable log level.

- HTTP queries
    - [ ] Specify sequences of HTTP requests to send.
    - [ ] Inspect responses with Rego expressions.
    - [ ] Use Rego query watcher API to deal with timing of responses

- Quality of implementation
    - [ ] Document fragment line numbers for error reporting
    - [ ] CLI errors bubbled up from cobra should be "$PROGNAME: blah"
    - [ ] colorize errors and so forth

# Notes

**Query watching.** We can often observe the results of changing a
Kubernetes object by issuing HTTP requests. However, since Kubernetes
controllers are eventually consistent, this observation is vulnerable
to timing issues. We might be able to address this with the OPA
[watch](https://pkg.go.dev/github.com/open-policy-agent/opa@v0.17.1/watch)
API, which lets us set a watch on a query and be notified when its
result changes. Note that there is still an ordering problem to solve,
since to avoid race conditions, we should start the watch before making
a configuration change.

**Document parsing.** It's pretty common that Rego doesn't parse or
compile the first time. If that happens, we often report an unknown
fragment type rather than an invalid Rego fragment. This has to be
made more deterministic, because it makes the user experience suck.

# References

- https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md
