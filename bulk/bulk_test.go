package bulk_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kusubooru/tagaa/bulk"
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
	{
		filepath.Join("/", "local", "path", "dir"),
		",,,,",
		filepath.Join("/"),
		nil,
	},
	{
		filepath.Join("/", "local", "path", "dir"),
		filepath.Join("/", "server", "path", "dir") + ",,,,",
		filepath.Join("/", "server", "path"),
		nil,
	},
	{
		filepath.Join("/", "local", "path", "dir"),
		"",
		"",
		nil,
	},
	{
		filepath.Join("/", "local", "path", "dir"),
		filepath.Join("/", "server", "path", "dir", "somepic.jpg") + ",,,,",
		filepath.Join("/", "server", "path"),
		nil,
	},
}

func TestCurrentPrefix(t *testing.T) {
	for _, tt := range currentPrefixTests {
		got, err := bulk.CurrentPrefix(tt.indir, strings.NewReader(tt.infile))
		if want := tt.oerr; !reflect.DeepEqual(err, want) {
			t.Errorf("CurrentPrefix(%q, %q) returned err %q, want %q", tt.indir, tt.infile, err, want)
		}
		if want := tt.out; !reflect.DeepEqual(got, want) {
			t.Fatalf("CurrentPrefix(%q, %q) => %q, want %q", tt.indir, tt.infile, got, want)
		}
	}
}

func TestCurrentPrefix_readFail(t *testing.T) {
	r := strings.NewReader("")
	in := ErrReader(r, fmt.Errorf("read fail"))
	got, err := bulk.CurrentPrefix("", in)
	if err == nil {
		t.Errorf("CurrentPrefix with read failure must return err but returned %q, %q", got, err)
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
	{
		[]bulk.Image{{Name: "img1"}, {Name: "img2"}},
		[]bulk.Image{{Name: "img1", Source: "source1"}, {Name: "img2", Source: "source2"}, {Name: "img3", Source: "source3"}},
		[]bulk.Image{{Name: "img1", Source: "source1"}, {Name: "img2", Source: "source2"}},
	},
	{
		[]bulk.Image{{Name: "img2"}, {Name: "img1"}},
		[]bulk.Image{{Name: "img2", Source: "source2"}, {Name: "img1", Source: "source1"}},
		[]bulk.Image{{Name: "img1", Source: "source1"}, {Name: "img2", Source: "source2"}},
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

type FileInfoMock struct {
	name  string
	isDir bool
}

func (fim FileInfoMock) Name() string       { return fim.name }
func (fim FileInfoMock) Size() int64        { return 0 }
func (fim FileInfoMock) Mode() os.FileMode  { return 0 }
func (fim FileInfoMock) ModTime() time.Time { return time.Now() }
func (fim FileInfoMock) IsDir() bool        { return fim.isDir }
func (fim FileInfoMock) Sys() interface{}   { return nil }

var loadImagesTests = []struct {
	in  []os.FileInfo
	out []bulk.Image
}{
	// Empty case.
	{
		[]os.FileInfo{},
		[]bulk.Image{},
	},
	// Ignoring .csv and folders.
	{
		[]os.FileInfo{
			FileInfoMock{name: "bulk.csv"},
			FileInfoMock{name: "folder1", isDir: true},
			FileInfoMock{name: "folder2", isDir: true},
		},
		[]bulk.Image{},
	},
	// Ignoring .ico and folder.
	{
		[]os.FileInfo{
			FileInfoMock{name: "a.jpg"},
			FileInfoMock{name: "b.ico"},
			FileInfoMock{name: "folder", isDir: true},
		},
		[]bulk.Image{
			{ID: 0, Name: "a.jpg"},
		},
	},
	// All supported types.
	{
		[]os.FileInfo{
			FileInfoMock{name: "a.gif"},
			FileInfoMock{name: "b.jpeg"},
			FileInfoMock{name: "c.jpg"},
			FileInfoMock{name: "e.png"},
			FileInfoMock{name: "f.swf"},
		},
		[]bulk.Image{
			{ID: 0, Name: "a.gif"},
			{ID: 1, Name: "b.jpeg"},
			{ID: 2, Name: "c.jpg"},
			{ID: 3, Name: "e.png"},
			{ID: 4, Name: "f.swf"},
		},
	},
	// We  depend on iotuil.ReadDir to sort the dir entries.  If for some
	// reason we get unsorted dir entries we treat that order as the correct
	// one.
	{
		[]os.FileInfo{
			FileInfoMock{name: "zzz.jpg"},
			FileInfoMock{name: "bbb.jpg"},
			FileInfoMock{name: "aaa.jpg"},
		},
		[]bulk.Image{
			{ID: 0, Name: "zzz.jpg"},
			{ID: 1, Name: "bbb.jpg"},
			{ID: 2, Name: "aaa.jpg"},
		},
	},
}

func TestLoadImages(t *testing.T) {
	for _, tt := range loadImagesTests {
		got := bulk.LoadImages(tt.in)
		if want := tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("LoadImages(%q) => %q, want %q", tt.in, got, want)
		}
	}
}

var saveTests = []struct {
	images   []bulk.Image
	dir      string
	prefix   string
	useLinux bool
	out      string
}{
	{
		[]bulk.Image{{ID: 0, Name: "img1", Source: "source1", Rating: "s"}},
		filepath.Join("/", "local", "path", "dir"),
		filepath.Join("/", "server", "path"),
		false,
		filepath.Join("/", "server", "path", "dir", "img1") + ",,source1,s,\n",
	},
	{
		[]bulk.Image{{ID: 0, Name: "img1", Source: "source1", Rating: "s"}},
		filepath.Join("/", "local", "path", "dir"),
		filepath.Join("/", "server", "path"),
		true,
		"/server/path/dir/img1,,source1,s,\n",
	},
	{
		[]bulk.Image{
			{ID: 0, Name: "img1", Source: "source1", Rating: "s"},
			{ID: 1, Name: "img2", Source: "source2", Rating: "q"},
		},
		filepath.Join("/", "local", "path", "dir"),
		filepath.Join("/", "server", "path"),
		false,
		filepath.Join("/", "server", "path", "dir", "img1") + ",,source1,s,\n" + filepath.Join("/", "server", "path", "dir", "img2") + ",,source2,q,\n",
	},
}

func TestSave(t *testing.T) {
	for _, tt := range saveTests {
		var b bytes.Buffer
		err := bulk.Save(&b, tt.images, tt.dir, tt.prefix, tt.useLinux)
		if err != nil {
			t.Errorf("Save(%q, %q, %q) returned err %q", tt.images, tt.dir, tt.prefix, err)
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
	err := bulk.Save(in, images, "", "", false)
	if err == nil {
		t.Errorf("Save with write failure must return err but returned %q", err)
	}
}

// ErrWriter returns a writer that always fails with the provided error.
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

// ErrReader returns a reader that always fails with the provided error.
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
