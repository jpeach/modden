package test

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jpeach/modden/pkg/must"
)

// Severity indicated the seriousness of a test failure.
type Severity string

// SeverityNone ...
const SeverityNone Severity = "None"

// SeverityWarn ...
const SeverityWarn Severity = "Warn"

// SeverityError ...
const SeverityError Severity = "Error"

// SeverityFatal ...
const SeverityFatal Severity = "Fatal"

// MessageSink collects Message entries
type MessageSink struct {
	Messages []Message
}

// Document records the execution of a test document.
type Document struct {
	MessageSink

	Description string
	Properties  map[string]interface{}
	Steps       []*Step
}

// EachError walks the test document and applies the function to
// each error.
func (d *Document) EachError(f func(*Step, *Error)) {
	for _, s := range d.Steps {
		for _, e := range s.Errors {
			f(s, &e)
		}
	}
}

// Step describes a stage in a test document that can generate onr
// or more related errors.
type Step struct {
	MessageSink

	Description string
	Start       time.Time
	End         time.Time
	Errors      []Error
	Diagnostics map[string]interface{}
}

// Error describes a specific test failure.
type Error struct {
	Severity Severity
	Message  Message
}

// Message is a timestamped log entry.
type Message struct {
	Message   string
	Timestamp time.Time
}

// Messagef formats the arguments into a new Message.
func Messagef(format string, args ...interface{}) Message {
	return Message{
		Message:   fmt.Sprintf(format, args...),
		Timestamp: time.Now(),
	}
}

// Closer is an interface that closes an implicit test tracking entity.
type Closer interface {
	Close()
}

// CloserFunc is a Closer adaptor.
type CloserFunc func()

// Close implements Closer.
func (c CloserFunc) Close() {
	c()
}

// Recorder records test execution actions.
type Recorder struct {
	docs []*Document

	sink        []*MessageSink
	currentDoc  *Document
	currentStep *Step
}

func push(s *MessageSink, stack []*MessageSink) []*MessageSink {
	return append([]*MessageSink{s}, stack...)
}

func pop(stack []*MessageSink) []*MessageSink {
	return stack[1:]
}

// ShouldContinue returns false if any fatal errors have been recorded.
func (r *Recorder) ShouldContinue() bool {
	count := 0

	// Make the check context-dependent. If we are in the middle
	// of a doc, this asks whether we should keep going on the
	// doc, otherwise it asks whether we should keep going at all.
	which := r.docs
	if r.currentDoc != nil {
		which = []*Document{r.currentDoc}
	}

	for _, d := range which {
		d.EachError(func(s *Step, e *Error) {
			if e.Severity == SeverityFatal {
				count++
			}
		})
	}

	return count == 0
}

// Failed returns true if any errors have been recorded.
func (r *Recorder) Failed() bool {
	failed := false

	for _, d := range r.docs {
		d.EachError(func(s *Step, e *Error) {
			switch e.Severity {
			case SeverityFatal, SeverityError:
				failed = true
			}
		})
	}

	return failed
}

// NewDocument creates a new Document and makes it current.
func (r *Recorder) NewDocument(desc string) Closer {
	must.Check(r.currentStep == nil,
		fmt.Errorf("can't create a new doc with an open step"))

	doc := &Document{}

	r.currentDoc = doc
	r.docs = append(r.docs, doc)
	r.sink = push(&doc.MessageSink, r.sink)

	log.Printf("Document Start: %s", desc)
	return CloserFunc(func() {
		must.Check(r.currentDoc == doc,
			fmt.Errorf("overlapping docs"))
		must.Check(r.currentStep == nil,
			fmt.Errorf("closing doc with open step"))

		r.sink = pop(r.sink)
		r.currentDoc = nil
		log.Printf("Document End: %s", desc)
	})
}

// NewStep creates a new Step within the current Document and makes
// that the current Step.
func (r *Recorder) NewStep(desc string) Closer {
	must.Check(r.currentDoc != nil,
		fmt.Errorf("no open document"))

	step := &Step{
		Description: desc,
		Start:       time.Now(),
	}

	r.currentStep = step
	r.currentDoc.Steps = append(r.currentDoc.Steps, step)
	r.sink = push(&step.MessageSink, r.sink)

	log.Printf("Step Start: %s", desc)
	return CloserFunc(func() {
		must.Check(r.currentStep == step,
			fmt.Errorf("overlapping steps"))

		step.End = time.Now()

		r.sink = pop(r.sink)
		r.currentStep = nil

		log.Printf("Step End: %s", desc)
	})
}

// Messagef records a message to the current message sink (i.e. Step or Document).
func (r *Recorder) Messagef(format string, args ...interface{}) {
	log.Printf(format, args...)
	r.sink[0].Messages = append(r.sink[0].Messages, Messagef(format, args...))
}

// Errorf records a test error to the current Step.
func (r *Recorder) Errorf(severity Severity, format string, args ...interface{}) {
	must.Check(r.currentStep != nil,
		fmt.Errorf("no open step"))

	log.Printf(strings.ToUpper(string(severity))+": "+format, args...)

	r.currentStep.Errors = append(r.currentStep.Errors, Error{
		Severity: severity,
		Message:  Messagef(format, args...),
	})
}
