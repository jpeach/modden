package driver

import (
	"sync"

	"github.com/jpeach/modden/pkg/doc"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FixtureSet is a collection of fixture objects.
type FixtureSet interface {
	Insert(u *unstructured.Unstructured)
	Match(u *unstructured.Unstructured) *unstructured.Unstructured
}

type fixtureKey struct {
	apiVersion string
	kind       string
	name       string
	namespace  string
}

type defaultFixtureSet struct {
	lock     sync.Mutex
	fixtures map[fixtureKey]*unstructured.Unstructured
}

var _ FixtureSet = &defaultFixtureSet{}

func (f *defaultFixtureSet) Insert(u *unstructured.Unstructured) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.fixtures[keyFor(u)] = u
}

func (f *defaultFixtureSet) Match(u *unstructured.Unstructured) *unstructured.Unstructured {
	f.lock.Lock()
	defer f.lock.Unlock()

	wanted := keyFor(u)
	return f.fixtures[wanted]
}

func keyFor(u *unstructured.Unstructured) fixtureKey {
	return fixtureKey{
		apiVersion: u.GetAPIVersion(),
		kind:       u.GetKind(),
		name:       u.GetName(),
		namespace:  u.GetNamespace(),
	}
}

// AddFixtures decodes the given doc.Document and adds all the
// object fragments to the default FixtureSet.
func AddFixtures(f *doc.Document) error {
	for _, p := range f.Parts {
		ftype, err := p.Decode()
		if err != nil {
			return err
		}

		if ftype == doc.FragmentTypeObject {
			DefaultFixtures.Insert(p.Object())
		}
	}

	return nil
}

// DefaultFixtures is a default FixtureSet.
var DefaultFixtures = &defaultFixtureSet{
	fixtures: map[fixtureKey]*unstructured.Unstructured{},
}
