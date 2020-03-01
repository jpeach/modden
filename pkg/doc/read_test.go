package doc

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

			if diff := cmp.Diff(&tc.Want, got, cmpopts.IgnoreUnexported(Fragment{})); diff != "" {
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
				{
					Bytes: []byte{'o', 'n', 'e'},
				},
			},
		},
	})

	run(t, "three empty", testcase{
		Data: `---
---
---`,
		Want: Document{
			Parts: []Fragment{
				{Bytes: []byte{}},
				{Bytes: []byte{}},
				{Bytes: []byte{}},
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
				{Bytes: []byte{'a', '\n'}},
				{Bytes: []byte{'b', '\n'}},
				{Bytes: []byte{'c'}},
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
				{Bytes: []byte{'a', '\n'}},
				{Bytes: []byte{'b', '\n'}},
				{Bytes: []byte{'c', '\n'}},
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
				{Bytes: []byte{'f', ' ', '-', '-', '-', '\n', 'a', '\n'}},
				{Bytes: []byte{'b'}},
			},
		},
	})

}
