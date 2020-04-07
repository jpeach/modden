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
					Bytes:    []byte("one"),
					Location: Location{Start: 1, End: 1},
				},
			},
		},
	})

	// Empty fragments don't create anything.
	run(t, "three empty", testcase{
		Data: `---
---
---`,
		Want: Document{
			Parts: nil,
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
				{Bytes: []byte("a\n"), Location: Location{Start: 1, End: 1}},
				{Bytes: []byte("b\n"), Location: Location{Start: 3, End: 3}},
				{Bytes: []byte("c"), Location: Location{Start: 5, End: 5}},
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
				{Bytes: []byte("a\n"), Location: Location{Start: 1, End: 1}},
				{Bytes: []byte("b\n"), Location: Location{Start: 3, End: 3}},
				{Bytes: []byte("c\n"), Location: Location{Start: 5, End: 5}},
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
				{Bytes: []byte("f ---\na\n"), Location: Location{Start: 1, End: 2}},
				{Bytes: []byte("b"), Location: Location{Start: 4, End: 4}},
			},
		},
	})

}
