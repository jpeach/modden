package builtin.check.update

# Default check for updating Kubernetes object updates.

fatal[msg] {
  input.latest != data.resources.applied.last

  fmt := `internal data inconsistency; input.latest is not the last object applied
    input.latest: %s
    applied.last: %s`

  msg := sprintf(fmt, [
      input.latest,
      data.resources.applied.last,
  ])
}

error[msg] {
  # If the Error field is present, the update failed.
  input.error

  msg := sprintf("failed to update %s '%s/%s': %s", [
    input.target.meta.kind,
    input.target.namespace,
    input.target.name,
    input.error.message,
  ])
}

# vim: ts=2 sts=2 sw=2 et:
