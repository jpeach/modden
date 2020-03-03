package driver

import (
	"sync"

	"k8s.io/client-go/tools/cache"
)

// MuxingResourceEventHandler sends each event to every attached
// handler, in no particular order.
type MuxingResourceEventHandler struct {
	Handlers  map[int]cache.ResourceEventHandler
	nextIndex int
}

var _ cache.ResourceEventHandler = &MuxingResourceEventHandler{}

// Clear removes all the registered handlers.
func (m *MuxingResourceEventHandler) Clear() {
	m.Handlers = nil
}

// Add registers a new handler and returns a removal token.
func (m *MuxingResourceEventHandler) Add(c cache.ResourceEventHandler) int {
	if m.Handlers == nil {
		m.Handlers = make(map[int]cache.ResourceEventHandler)
	}

	i := m.nextIndex
	m.nextIndex++
	m.Handlers[i] = c
	return i
}

// Remove unregisters a handler using the removal token from a previous Add.
func (m *MuxingResourceEventHandler) Remove(which int) {
	if m.Handlers != nil {
		delete(m.Handlers, which)
	}
}

// OnAdd ...
func (m *MuxingResourceEventHandler) OnAdd(newObj interface{}) {
	for _, h := range m.Handlers {
		h.OnAdd(newObj)
	}
}

// OnUpdate ...
func (m *MuxingResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	for _, h := range m.Handlers {
		h.OnUpdate(oldObj, newObj)
	}
}

// OnDelete ...
func (m *MuxingResourceEventHandler) OnDelete(oldObj interface{}) {
	for _, h := range m.Handlers {
		h.OnDelete(oldObj)
	}
}

// LockingResourceEventHandler holds its lock, then invokes the next handler.
type LockingResourceEventHandler struct {
	Next cache.ResourceEventHandler
	Lock sync.Mutex
}

var _ cache.ResourceEventHandler = &LockingResourceEventHandler{}

// OnAdd ...
func (l *LockingResourceEventHandler) OnAdd(newObj interface{}) {
	l.Lock.Lock()
	defer l.Lock.Unlock()

	l.Next.OnAdd(newObj)
}

// OnUpdate ...
func (l *LockingResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	l.Lock.Lock()
	defer l.Lock.Unlock()

	l.Next.OnUpdate(oldObj, newObj)
}

// OnDelete ...
func (l *LockingResourceEventHandler) OnDelete(oldObj interface{}) {
	l.Lock.Lock()
	defer l.Lock.Unlock()

	l.Next.OnDelete(oldObj)
}

// WrappingResourceEventHandlerFuncs is the equivalent of
// cache.ResourceEventHandlerFuncs, except that after invoking the
// local handler. it also invoked the one pointer to by Next.
type WrappingResourceEventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
	DeleteFunc func(obj interface{})

	Next cache.ResourceEventHandler
}

var _ cache.ResourceEventHandler = &WrappingResourceEventHandlerFuncs{}

// OnAdd ...
func (r *WrappingResourceEventHandlerFuncs) OnAdd(newObj interface{}) {
	if r.AddFunc != nil {
		r.AddFunc(newObj)
	}

	r.Next.OnAdd(newObj)
}

// OnUpdate ...
func (r *WrappingResourceEventHandlerFuncs) OnUpdate(oldObj, newObj interface{}) {
	if r.UpdateFunc != nil {
		r.UpdateFunc(oldObj, newObj)
	}

	r.Next.OnUpdate(oldObj, newObj)
}

// OnDelete ...
func (r *WrappingResourceEventHandlerFuncs) OnDelete(oldObj interface{}) {
	if r.DeleteFunc != nil {
		r.DeleteFunc(oldObj)
	}

	r.Next.OnDelete(oldObj)
}
