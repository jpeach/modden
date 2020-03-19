package utils

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/open-policy-agent/opa/ast"
)

// ParseModuleFile parses the Rego module in the given file path.
func ParseModuleFile(filePath string) (*ast.Module, error) {
	fileData, err := ioutil.ReadFile(filePath) // nolint(gosec)
	if err != nil {
		return nil, err
	}

	fileModule, err := ast.ParseModule(filePath, string(fileData))
	if err != nil {
		return nil, err
	}

	return fileModule, nil
}

// ParseCheckFragment parses a Rego string into a *ast.Module. The
// Rego input is assumed to not have a package declaration so a random
// package name is prepended to make the parsed module globally unique.
// ParseCheckFragment can return nil with no error if the input is empty.
func ParseCheckFragment(input string) (*ast.Module, error) {
	// Rego requires a package name to generate any Rules.  Force
	// a package name that is unique to the fragment.  Note that
	// we also use this to generate a unique filename placeholder
	// since Rego internals will sometime use this as a map key.
	moduleName := RandomStringN(12)

	m, err := ast.ParseModule(
		fmt.Sprintf("internal/check/%s", moduleName),
		fmt.Sprintf("package check.%s\n%s", moduleName, input))
	if err != nil {
		return nil, err
	}

	// ParseModule can return nil with no error (empty module).
	if m == nil {
		return nil, io.EOF
	}

	return m, nil
}

// AsRegoCompilationErr attempts to convert this error into a Rego
// compilation error.
func AsRegoCompilationErr(err error) ast.Errors {
	var astErrors ast.Errors

	if errors.As(err, &astErrors) {
		return astErrors
	}

	return nil
}
