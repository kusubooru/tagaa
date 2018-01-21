package boltdb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/kusubooru/tagaa/tagaa"
)

func setup() (tagaa.Store, string) {
	f, err := ioutil.TempFile("", "tagaa_boltdb_tmpfile_")
	if err != nil {
		log.Fatal("could not create boltdb temp file for tests:", err)
	}
	store, err := NewStore(f.Name())
	if err != nil {
		log.Fatal("NewStore for temp file error:", err)
	}
	return store, f.Name()
}

func teardown(store tagaa.Store, tmpfile string) {
	store.Close()
	if err := os.Remove(tmpfile); err != nil {
		log.Println("could not remove boltdb temp file:", err)
	}
}

func TestCreateGroup(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groupA := "group A"
	groupB := "group B"

	if err := store.CreateGroup(groupA); err != nil {
		t.Fatalf("store.CreateGroup(%q) failed:", groupA, err)
	}

	if err := store.CreateGroup(groupB); err != nil {
		t.Fatalf("store.CreateGroup(%q) failed:", groupB, err)
	}
	out, err := store.GetAllGroups()
	if err != nil {
		t.Fatal("store.GetAllGroups failed:", err)
	}
	if got, want := len(out), 2; got != want {
		t.Fatalf("store.GetAllGroups returned %d results, expected %d instead", got, want)
	}
	got := out
	want := []*tagaa.Group{
		{Name: groupA},
		{Name: groupB},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.GetAllGroups() \nhave: %#v \nwant: %#v", got, want)
		data, _ := json.Marshal(got)
		fmt.Printf("have: %v\n", string(data))
		data, _ = json.Marshal(want)
		fmt.Printf("want: %v\n", string(data))
	}
}

func TestCreateGroup_sameGroupName(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groupName := "same group name"

	if err := store.CreateGroup(groupName); err != nil {
		t.Fatalf("store.CreateGroup(%q) failed: %v", groupName, err)
	}

	if err := store.CreateGroup(groupName); err == nil {
		t.Fatalf("store.CreateGroup(%q) expected to return err", err)
	}
}

func TestDeleteGroup(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groupName := "delete me"

	want := &tagaa.Group{Name: groupName}
	testCreateGroupDeepEqual(t, store, groupName, want)

	if err := store.DeleteGroup(groupName); err != nil {
		t.Fatalf("delete group %q failed: %v", groupName, err)
	}

	_, err := store.GetGroup(groupName)
	switch err {
	case tagaa.ErrNotFound:
	case nil:
		t.Fatalf("store.GetGroup(%q) expected to return item not found error", groupName)
	default:
		t.Fatalf("store.GetGroup(%q) failed: %v", groupName, err)
	}
}

func TestDeleteGroup_notEmpty(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groupName := "delete me"

	want := &tagaa.Group{Name: groupName}
	testCreateGroupDeepEqual(t, store, groupName, want)
	if err := store.AddImage(groupName, &tagaa.Image{ID: uint64(1)}); err != nil {
		t.Fatal("add image to group failed:", err)
	}

	err := store.DeleteGroup(groupName)
	switch err {
	case tagaa.ErrGroupNotEmpty:
	case nil:
		t.Fatalf("delete not empty group expected to return error %q", tagaa.ErrGroupNotEmpty)
	default:
		t.Fatalf("delete group %q failed: %v", groupName, err)
	}
}

func TestDeleteGroup_notFound(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groupName := "non existent group"
	err := store.DeleteGroup(groupName)
	switch err {
	case tagaa.ErrNotFound:
	case nil:
		t.Fatalf("delete non existent group expected to return error %q", tagaa.ErrNotFound)
	default:
		t.Fatalf("delete group %q failed: %v", groupName, err)
	}
}

func testGetGroupDeepEqual(t *testing.T, store tagaa.Store, groupName string, want *tagaa.Group) *tagaa.Group {
	t.Helper()
	got, err := store.GetGroup(groupName)
	if err != nil {
		t.Fatalf("store.GetGroup(%q) failed: %v", groupName, err)
	}

	deepEqual(t, got, want)
	return got
}

func testCreateGroupDeepEqual(t *testing.T, store tagaa.Store, groupName string, want *tagaa.Group) *tagaa.Group {
	t.Helper()
	if err := store.CreateGroup(groupName); err != nil {
		t.Fatalf("store.CreateGroup(%q) failed: %v", groupName, err)
	}

	return testGetGroupDeepEqual(t, store, groupName, want)
}

func deepEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("\nhave: %#v \nwant: %#v", got, want)
		data, _ := json.Marshal(got)
		fmt.Printf("have: %v\n", string(data))
		data, _ = json.Marshal(want)
		fmt.Printf("want: %v\n", string(data))
	}
}

func TestAddImage(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groupA := "group A"
	if err := store.CreateGroup(groupA); err != nil {
		t.Fatalf("store.CreateGroup(%q) failed:", groupA, err)
	}

	imgName := "img.jpg"
	img := &tagaa.Image{Name: imgName, Size: 5}
	if err := store.AddImage(groupA, img); err != nil {
		t.Fatalf("store.AddImage(%q, %#v) failed:", groupA, img, err)
	}
	// add another image
	if err := store.AddImage(groupA, img); err != nil {
		t.Fatalf("store.AddImage(%q, %#v) failed:", groupA, img, err)
	}

	got, err := store.GetAllGroups()
	if err != nil {
		t.Fatal("store.GetAllGroups failed:", err)
	}
	want := []*tagaa.Group{
		{Name: groupA, Size: 10, Images: []uint64{1, 2}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.GetAllGroups() \nhave: %#v \nwant: %#v", got, want)
		data, _ := json.Marshal(got)
		fmt.Printf("have: %v\n", string(data))
		data, _ = json.Marshal(want)
		fmt.Printf("want: %v\n", string(data))
	}

	images, err := store.GetGroupImages(groupA)
	if err != nil {
		t.Fatal("store.GetGroupImages(%q) failed:", groupA, err)
		//t.Errorf("store.GetGroupImages(%q) \nhave: %#v \nwant: %#v", groupA, got, want)
	}
	if got, want := len(images), 2; got != want {
		t.Fatalf("store.GetGroupImages returned %d results, expected %d instead", got, want)
	}
}

func TestAddImage_nonExistentGroup(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	// Try to put an image to a non existent group.
	groupName := "unknown group"
	imgName := "img.jpg"
	img := &tagaa.Image{Name: imgName, Size: 5}
	if err := store.AddImage(groupName, img); err != nil {
		t.Fatalf("store.AddImage(%q, %#v) failed: %v", groupName, img, err)
	}

	// Test that the new group has been created.
	got, err := store.GetAllGroups()
	if err != nil {
		t.Fatal("store.GetAllGroups failed:", err)
	}
	want := []*tagaa.Group{
		{Name: groupName, Size: 5, Images: []uint64{1}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.GetAllGroups() \nhave: %#v \nwant: %#v", got, want)
		data, _ := json.Marshal(got)
		fmt.Printf("have: %v\n", string(data))
		data, _ = json.Marshal(want)
		fmt.Printf("want: %v\n", string(data))
	}

}
