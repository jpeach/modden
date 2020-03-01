package driver

// Make sure we import the client-go auth provider plugins.

import (
	// Import client authentication methods.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)
