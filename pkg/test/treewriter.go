package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/jpeach/modden/pkg/must"
)

const (
	// Fixed-width boxing characters.
	boxBranch   = "├─"
	boxVertical = "│ "
	boxLeft     = "└─"

	// tabPrintf leaders are boxing characters with a bit of
	// fixed breathing space.
	branchLeader = boxBranch + " "
	elbowLeader  = boxLeft + " "
)

func formatIndent(n int) string {
	b := strings.Builder{}
	b.Grow(n * len(boxVertical))

	for i := 0; i < n; i++ {
		must.Int(b.WriteString(boxVertical))
	}

	return b.String()
}

func formatFailCounters(fails map[Severity]int) string {
	b := strings.Builder{}

	pluralize := func(s Severity, n int) string {
		switch n {
		case 1:
			return map[Severity]string{
				SeverityWarn:  "warning",
				SeverityError: "error",
				SeverityFatal: "error",
			}[s]
		default:
			return map[Severity]string{
				SeverityWarn:  "warnings",
				SeverityError: "errors",
				SeverityFatal: "errors",
			}[s]
		}
	}

	if n := fails[SeverityError] + fails[SeverityFatal]; n > 0 {
		must.Int(b.WriteString(
			fmt.Sprintf("%d %s", n, pluralize(SeverityError, n))))
	}

	if n := fails[SeverityWarn]; n > 0 {
		if b.Len() > 0 {
			must.Int(b.WriteString(", "))
		}

		must.Int(b.WriteString(
			fmt.Sprintf("%d %s", n, pluralize(SeverityWarn, n))))
	}

	return b.String()
}

// TreeWriter is a Recorder that write test results to a standard
// output in a tree notation.
type TreeWriter struct {
	indent    int
	stepCount int

	stepErrors map[Severity]int
	allErrors  map[Severity]int
}

var _ Recorder = &TreeWriter{}

func tabPrintf(indent int, leader string, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05.0000")
	msg := fmt.Sprintf(format, args...)
	lines := strings.Split(msg, "\n")

	for n, line := range lines {
		// Format the leader only on the first output line,
		// replacing it with an extra indent on subsequent
		// lines. This makes branchLeader entries look better,
		// but will horrendously munge elbowLeader ones (the
		// logic needs to be reversed).
		if n == 0 {
			fmt.Printf("%s\t%s%s%s\n",
				timestamp, formatIndent(indent), leader, line)
		} else {
			fmt.Printf("%s\t%s %s\n",
				timestamp, formatIndent(indent+1), line)
		}
	}
}

// ShouldContinue ...
func (t *TreeWriter) ShouldContinue() bool {
	return true
}

// Failed ...
func (t *TreeWriter) Failed() bool {
	return false
}

// NewDocument ...
func (t *TreeWriter) NewDocument(desc string) Closer {
	tabPrintf(t.indent, "", "Running: %s", desc)

	t.stepCount = 0
	t.allErrors = map[Severity]int{}
	return CloserFunc(func() {
		nerr := t.allErrors[SeverityFatal] + t.allErrors[SeverityError]

		if nerr > 0 {
			tabPrintf(t.indent, elbowLeader, "Failed with %s ",
				formatFailCounters(t.allErrors))
		} else {
			tabPrintf(t.indent, elbowLeader, "Pass with %d steps OK", t.stepCount)
		}
	})
}

// NewStep ...
func (t *TreeWriter) NewStep(desc string) Closer {
	tabPrintf(t.indent, branchLeader, "Step %d: %s", t.stepCount, desc)

	t.indent++
	t.stepCount++
	t.stepErrors = map[Severity]int{}
	return CloserFunc(func() {
		nerr := t.stepErrors[SeverityFatal] + t.stepErrors[SeverityError]

		if nerr > 0 {
			tabPrintf(t.indent, elbowLeader, "Failed with %s ",
				formatFailCounters(t.stepErrors))
		} else {
			tabPrintf(t.indent, elbowLeader, "Pass")
		}

		t.indent--
		for k, v := range t.stepErrors {
			t.allErrors[k] = t.allErrors[k] + v
		}
	})
}

// Messagef ...
func (t *TreeWriter) Messagef(format string, args ...interface{}) {
	tabPrintf(t.indent, branchLeader, format, args...)
}

// Errorf ...
func (t *TreeWriter) Errorf(severity Severity, format string, args ...interface{}) {
	t.stepErrors[severity]++
	msg := fmt.Sprintf(format, args...)
	tabPrintf(t.indent, branchLeader, "%s: %s", strings.ToUpper(string(severity)), msg)
}
