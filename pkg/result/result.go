package result

import (
	"fmt"
	"time"
)

// Severity indicates the seriousness of a Result.
type Severity string

// SeverityNone ...
const SeverityNone Severity = "None"

// SeverityError ...
const SeverityError Severity = "Error"

// SeverityFatal ...
const SeverityFatal Severity = "Fatal"

// SeveritySkip ...
const SeveritySkip Severity = "Skip"

// Result ...
type Result struct {
	Severity  Severity
	Message   string
	Timestamp time.Time
}

// IsTerminal returns true if this result should end the test.
func (c Result) IsTerminal() bool {
	switch c.Severity {
	case SeverityFatal, SeveritySkip:
		return true
	default:
		return false
	}
}

// IsFailed returns true if this result is a test failure.
func (c Result) IsFailed() bool {
	switch c.Severity {
	case SeverityFatal, SeverityError:
		return true
	default:
		return false
	}
}

func resultFrom(s Severity, format string, args ...interface{}) Result {
	return Result{
		Severity:  s,
		Message:   fmt.Sprintf(format, args...),
		Timestamp: time.Now(),
	}
}

// Infof formats a SeverityNone result.
func Infof(format string, args ...interface{}) Result {
	return resultFrom(SeverityNone, format, args...)
}

// Errorf formats a SeverityError result.
func Errorf(format string, args ...interface{}) Result {
	return resultFrom(SeverityError, format, args...)
}

// Fatalf formats a SeverityFatal result.
func Fatalf(format string, args ...interface{}) Result {
	return resultFrom(SeverityFatal, format, args...)
}

// Skipf formats a SeveritySkip result.
func Skipf(format string, args ...interface{}) Result {
	return resultFrom(SeveritySkip, format, args...)
}
