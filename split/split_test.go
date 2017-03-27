package split_test

import (
	"reflect"
	"testing"

	"github.com/kusubooru/tagaa/split"
)

var bytesTests = []struct {
	in    []byte
	incol int
	out   [][]byte
}{
	{
		[]byte{}, 0,
		[][]byte{},
	},
	{
		[]byte{}, 2,
		[][]byte{},
	},
	{
		[]byte{1, 2, 3}, 3,
		[][]byte{
			{1, 2, 3},
		},
	},
	{
		[]byte{1, 2, 3}, 2,
		[][]byte{
			{1, 2},
			{3},
		},
	},
	{
		[]byte{1, 2, 3}, 1,
		[][]byte{
			{1},
			{2},
			{3},
		},
	},
	{
		[]byte{1, 2, 3}, 0,
		[][]byte{
			{1},
			{2},
			{3},
		},
	},
}

func TestBytes(t *testing.T) {
	for _, tt := range bytesTests {
		got := split.Bytes(tt.in, tt.incol)
		if want := tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("bytesSplit(%q, %q) => %#v, want %#v", tt.in, tt.incol, got, want)
		}
	}
}
