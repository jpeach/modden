package test

import (
	"fmt"
	"strings"

	"github.com/jpeach/modden/pkg/must"

	"sigs.k8s.io/yaml"
)

type stepError struct {
	Severity Severity `yaml:"severity" json:"severity"`
	Message  string   `yaml:"message" json:"message"`
}

// TapWriter writes test records in TAP format.
// See https://testanything.org/tap-version-13-specification.html
type TapWriter struct {
	docCount  int
	stepCount int

	stepErrors []stepError
}

var _ Recorder = &TapWriter{}

// indentf prints a (possibly multi-line) message, prefixed by the indent.
func indentf(indent string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	for _, line := range strings.Split(msg, "\n") {
		fmt.Printf("%s%s\n", indent, line)
	}
}

// ShouldContinue ...
func (t *TapWriter) ShouldContinue() bool {
	return true
}

// Failed ...
func (t *TapWriter) Failed() bool {
	return false
}

// NewDocument ...
func (t *TapWriter) NewDocument(desc string) Closer {
	// It's not obvious how TAP separates test runs into suites
	// (maybe it doesn't?). Let's stuff a newline in there so at
	// least it's visually distinguished.
	if t.docCount == 0 {
		fmt.Printf("TAP version 13\n")
	} else {
		fmt.Printf("\nTAP version 13\n")
	}

	t.docCount++
	t.stepCount = 1

	return CloserFunc(func() {
		fmt.Printf("1..%d\n", t.stepCount+1)
	})
}

// NewStep ...
func (t *TapWriter) NewStep(desc string) Closer {
	n := t.stepCount
	t.stepCount++

	return CloserFunc(func() {
		if len(t.stepErrors) > 0 {
			fmt.Printf("not ok %d - %s\n", n, desc)
		} else {
			fmt.Printf("ok %d - %s\n", n, desc)
		}

		if len(t.stepErrors) > 0 {
			indent := "  "
			indentf(indent, "---")
			indentf(indent, string(must.Bytes(yaml.Marshal(t.stepErrors))))
			indentf(indent, "...")
		}

		t.stepErrors = nil
	})
}

// Messagef ...
func (t *TapWriter) Messagef(format string, args ...interface{}) {
	indentf("# ", format, args...)
}

// Errorf ...
func (t *TapWriter) Errorf(severity Severity, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	indentf(fmt.Sprintf("# %s -", string(severity)), msg)

	t.stepErrors = append(t.stepErrors, stepError{
		Severity: severity,
		Message:  msg,
	})
}
