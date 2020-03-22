package test

import "github.com/jpeach/modden/pkg/result"

// StackRecorders returns a new Recorder that stacks top and next.
// For each method in the Recorder interface, methods from top will
// be called first, followed by the ones from next.
func StackRecorders(top Recorder, next Recorder) Recorder {
	return &wrapRecorder{top, next}
}

func wrappedCloser(closers []Closer) Closer {
	return CloserFunc(func() {
		for _, c := range closers {
			c.Close()
		}
	})
}

type wrapRecorder struct {
	top  Recorder
	next Recorder
}

var _ Recorder = &wrapRecorder{}

func (w wrapRecorder) ShouldContinue() bool {
	return w.top.ShouldContinue() &&
		w.next.ShouldContinue()
}

func (w wrapRecorder) Failed() bool {
	return w.top.Failed() ||
		w.next.Failed()
}

func (w wrapRecorder) NewDocument(desc string) Closer {
	closers := []Closer{
		w.top.NewDocument(desc),
		w.next.NewDocument(desc),
	}

	return wrappedCloser(closers)
}

func (w wrapRecorder) NewStep(desc string) Closer {
	closers := []Closer{
		w.top.NewStep(desc),
		w.next.NewStep(desc),
	}

	return wrappedCloser(closers)
}

func (w wrapRecorder) Update(results ...result.Result) {
	w.top.Update(results...)
	w.next.Update(results...)
}
