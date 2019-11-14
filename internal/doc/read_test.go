package doc

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadDocument(t *testing.T) {
	type testcase struct {
		Data string
		Want Document
	}

	run := func(t *testing.T, name string, tc testcase) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()

			got, err := ReadDocument(bytes.NewBufferString(tc.Data))
			if err != nil {
				t.Fatalf("read error: %s", err)
			}

			if diff := cmp.Diff(&tc.Want, got); diff != "" {
				t.Fatalf(diff)
			}
		})
	}

	run(t, "empty", testcase{
		Data: "",
		Want: Document{},
	})

	run(t, "one", testcase{
		Data: "one",
		Want: Document{
			Parts: []Fragment{
				[]byte{'o', 'n', 'e'},
			},
		},
	})

	run(t, "three empty", testcase{
		Data: `---
---
---`,
		Want: Document{
			Parts: []Fragment{
				[]byte{},
				[]byte{},
				[]byte{},
			},
		},
	})

	run(t, "three frags", testcase{
		Data: `a
---
b
---
c`,
		Want: Document{
			Parts: []Fragment{
				[]byte{'a', '\n'},
				[]byte{'b', '\n'},
				[]byte{'c'},
			},
		},
	})

	run(t, "three frags with trailer", testcase{
		Data: `a
---
b
---
c
---`,
		Want: Document{
			Parts: []Fragment{
				[]byte{'a', '\n'},
				[]byte{'b', '\n'},
				[]byte{'c', '\n'},
			},
		},
	})

	run(t, "leading junk", testcase{
		Data: `f ---
a
---
b`,
		Want: Document{
			Parts: []Fragment{
				[]byte{'f', ' ', '-', '-', '-', '\n', 'a', '\n'},
				[]byte{'b'},
			},
		},
	})

}
