package fixture

import (
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FixtureSet is a collection of fixture objects.
// nolint(golint)
type FixtureSet interface {
	Insert(Key, Fixture)
	Match(u *unstructured.Unstructured) Fixture
}

// Key is the indexing fixture set key.
type Key struct {
	apiVersion string
	kind       string
	name       string
	namespace  string
}

// KeyFor returns the key for indexing the given object.
func KeyFor(u *unstructured.Unstructured) Key {
	return Key{
		apiVersion: u.GetAPIVersion(),
		kind:       u.GetKind(),
		name:       u.GetName(),
		namespace:  u.GetNamespace(),
	}
}

type defaultFixtureSet struct {
	lock     sync.Mutex
	fixtures map[Key]Fixture
}

var _ FixtureSet = &defaultFixtureSet{}

// Insert a fixture with the given key.
func (s *defaultFixtureSet) Insert(k Key, f Fixture) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.fixtures[k] = f
}

// Match the given object to an existing Fixture.
func (s *defaultFixtureSet) Match(u *unstructured.Unstructured) Fixture {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Assume that the caller will not modify the result.
	return s.fixtures[KeyFor(u)]
}

// Set is the default FixtureSet.
var Set = &defaultFixtureSet{
	fixtures: map[Key]Fixture{},
}
