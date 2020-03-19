package utils

import (
	"errors"
	"io/ioutil"

	"github.com/open-policy-agent/opa/ast"
)

// ParseModuleFile parses the Rego module in the given file path.
func ParseModuleFile(filePath string) (*ast.Module, error) {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileModule, err := ast.ParseModule(filePath, string(fileData))
	if err != nil {
		return nil, err
	}

	return fileModule, nil
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
