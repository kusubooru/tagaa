package bulk_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kusubooru/local-tagger/bulk"
)

var errInvalidFormat = errors.New("invalid csv file format")
var loadCSVTests = []struct {
	in   string
	out  []bulk.Image
	oerr error
}{
	{"", []bulk.Image{}, nil},
	{",,,,", []bulk.Image{}, nil},
	{",,,,,,,,,,,,,,,,,,,", []bulk.Image(nil), errInvalidFormat},
	{"invalid format", []bulk.Image(nil), errInvalidFormat},
	{",,,,\n,,,,\n,,,,", []bulk.Image{}, nil},
	{
		"/server/path/img1,tag1 tag2,source,s,",
		[]bulk.Image{
			{Name: "img1", Tags: []string{"tag1", "tag2"}, Source: "source", Rating: "s"},
		},
		nil,
	},
	{
		"/server/path/img1,tag1 tag2,source1,s,\n" +
			",,,,\n" +
			"/server/path/img2,tag1 tag2,source2,q,\n",
		[]bulk.Image{
			{Name: "img1", Tags: []string{"tag1", "tag2"}, Source: "source1", Rating: "s"},
			{Name: "img2", Tags: []string{"tag1", "tag2"}, Source: "source2", Rating: "q"},
		},
		nil,
	},
}

func TestLoadCSV(t *testing.T) {

	for _, tt := range loadCSVTests {
		got, err := bulk.LoadCSV(strings.NewReader(tt.in))
		if want := tt.oerr; !reflect.DeepEqual(err, want) {
			t.Errorf("LoadCSV(%q) returned err %q, want %q", tt.in, err, want)
		}
		if want := tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("LoadCSV(%q) => %q, want %q", tt.in, got, want)
		}
	}
}

func TestLoadCSV_readFail(t *testing.T) {
	r := strings.NewReader("")
	in := ErrReader(r, fmt.Errorf("read fail"))
	got, err := bulk.LoadCSV(in)
	if err == nil {
		t.Errorf("LoadCSV with read failure must return err but returned %q, %q", got, err)
	}
}

var currentPrefixTests = []struct {
	indir  string
	infile string
	out    string
	oerr   error
}{
	{"/local/path/dir", ",,,,", "/", nil},
	{"/local/path/dir", "/server/path/dir,,,,", "/server/path", nil},
	{"/local/path/dir", "", "", nil},
}

func TestCurrentPrefix(t *testing.T) {

	for _, tt := range currentPrefixTests {
		got, err := bulk.CurrentPrefix(tt.indir, strings.NewReader(tt.infile))
		if want := tt.oerr; !reflect.DeepEqual(err, want) {
			t.Errorf("CurrentPrefix(%q, %q) returned err %q, want %q", tt.indir, tt.infile, err, want)
		}
		if want := tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("CurrentPrefix(%q, %q) => %q, want %q", tt.indir, tt.infile, got, want)
		}
	}
}

var findByIDTests = []struct {
	images []bulk.Image
	id     int
	out    *bulk.Image
}{
	{nil, 0, nil},
	{[]bulk.Image{}, 0, nil},
	{[]bulk.Image{{ID: 1}}, 0, nil},
	{[]bulk.Image{{ID: 1}}, 1, &bulk.Image{ID: 1}},
	{[]bulk.Image{{ID: 2}, {ID: 1}}, 1, &bulk.Image{ID: 1}},
	{[]bulk.Image{{ID: 2}, {ID: 3}, {ID: 1}}, 3, &bulk.Image{ID: 3}},
	{[]bulk.Image{{ID: 2}, {ID: 3}, {ID: 1}}, 5, nil},
}

func TestFindByID(t *testing.T) {

	for _, tt := range findByIDTests {
		got := bulk.FindByID(tt.images, tt.id)
		if want := tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("FindByID(%q, %q) => %q, want %q", tt.images, tt.id, got, want)
		}
	}
}

var combineTests = []struct {
	images   []bulk.Image
	metadata []bulk.Image
	out      []bulk.Image
}{
	{
		[]bulk.Image{{ID: 0}},
		[]bulk.Image{{ID: 0, Source: "source"}},
		[]bulk.Image{{ID: 0}},
	},
	{
		nil,
		[]bulk.Image{{ID: 0, Source: "source"}},
		nil,
	},
	{
		[]bulk.Image{{ID: 0}},
		nil,
		[]bulk.Image{{ID: 0}},
	},
	{
		[]bulk.Image{{Name: "img1"}},
		[]bulk.Image{{Name: "img1", Source: "source"}},
		[]bulk.Image{{Name: "img1", Source: "source"}},
	},
	{
		[]bulk.Image{{Name: "img1"}, {Name: "img1"}},
		[]bulk.Image{{Name: "img1", Source: "source"}},
		[]bulk.Image{{Name: "img1", Source: "source"}, {Name: "img1"}},
	},
	{
		[]bulk.Image{{Name: "img1"}, {Name: "img1"}},
		[]bulk.Image{{Name: "img1", Source: "source"}, {Name: "img1", Rating: "q"}},
		[]bulk.Image{{Name: "img1", Rating: "q"}, {Name: "img1"}},
	},
}

func TestCombine(t *testing.T) {

	for _, tt := range combineTests {
		got := bulk.Combine(tt.images, tt.metadata)
		if want := tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("Combine(%q, %q) => %q, want %q", tt.images, tt.metadata, got, want)
		}
	}
}

func TestLoadImages_emptyDir(t *testing.T) {
	dirname := ""
	_, err := bulk.LoadImages(dirname)
	if err == nil {
		t.Errorf("LoadImages(%q) must return err", dirname)
	}
}

func TestLoadImages(t *testing.T) {
	const prefix = "local-tagger-test"

	dirname, err := ioutil.TempDir("", prefix)
	if err != nil {
		t.Error("could not create temp dir")
	}
	fname, err := ioutil.TempFile(dirname, prefix)
	if err != nil {
		t.Error("could not create temp file")
	}
	jpgFilepath := fname.Name() + ".jpg"
	err = os.Rename(fname.Name(), jpgFilepath)
	if err != nil {
		t.Error("could not rename temp file")
	}

	defer func() {
		err := os.Remove(jpgFilepath)
		if err != nil {
			t.Error("could not clean up temp file")
		}
		err = os.Remove(dirname)
		if err != nil {
			t.Error("could not clean up temp dir")
		}
	}()

	want := []bulk.Image{{ID: 0, Name: filepath.Base(jpgFilepath)}}
	got, err := bulk.LoadImages(dirname)
	if err != nil {
		t.Errorf("LoadImages(%q) returned err %v", dirname, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadImages(%q) => %q, want %q", dirname, got, want)
	}
}

var saveTests = []struct {
	images []bulk.Image
	dir    string
	prefix string
	out    string
}{
	{
		[]bulk.Image{{ID: 0, Name: "img1", Source: "source1", Rating: "s"}},
		"/local/path/dir",
		"/server/path",
		"/server/path/dir/img1,,source1,s,\n",
	},
	{
		[]bulk.Image{
			{ID: 0, Name: "img1", Source: "source1", Rating: "s"},
			{ID: 1, Name: "img2", Source: "source2", Rating: "q"},
		},
		"/local/path/dir",
		"/server/path",
		"/server/path/dir/img1,,source1,s,\n/server/path/dir/img2,,source2,q,\n",
	},
}

func TestSave(t *testing.T) {
	for _, tt := range saveTests {
		var b bytes.Buffer
		err := bulk.Save(&b, tt.images, tt.dir, tt.prefix)
		if err != nil {
			t.Errorf("Save(%q, %q, %q) returned err %q", tt.images, tt.dir, tt.prefix)
		}
		if got, want := b.String(), tt.out; got != want {
			t.Errorf("Save(%q, %q, %q) => %q, want %q", tt.images, tt.dir, tt.prefix, got, want)
		}
	}
}

func TestSave_writeFail(t *testing.T) {
	images := []bulk.Image{{ID: 0}, {ID: 1}}
	var b bytes.Buffer
	in := ErrWriter(&b, fmt.Errorf("write fail"))
	err := bulk.Save(in, images, "", "")
	if err == nil {
		t.Errorf("Save with write failure must return err but returned %q", err)
	}
}

// Failure case Writer helper.
func ErrWriter(w io.Writer, err error) io.Writer {
	return &errWriter{w, err}
}

type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) Write(p []byte) (n int, err error) {
	err = e.err
	return
}

// Failure case Reader helpers.
func ErrReader(r io.Reader, err error) io.Reader {
	return &errReader{r, err}
}

type errReader struct {
	r   io.Reader
	err error
}

func (e *errReader) Read(p []byte) (n int, err error) {
	err = e.err
	return
}
