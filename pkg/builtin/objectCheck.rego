# Default check for applying Kubernetes object updates.

fatal[msg] {
  input == data.resources.applied.last
  msg = "internal data inconsistency"
}

error[msg] {
  # If the Error field is present, the last applied resource generated an
  # error.
  data.resources.applied.last.error

  msg = sprintf("failed to update %s '%s/%s': %s", [
    data.resources.applied.last.target.meta.kind,
    data.resources.applied.last.target.namespace,
    data.resources.applied.last.target.name,
    data.resources.applied.last.error.message,
  ])
}

# vim: ts=2 sts=2 sw=2 et:
