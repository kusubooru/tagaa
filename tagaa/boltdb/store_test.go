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
	want := []string{groupA, groupB}
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

	if err := store.CreateGroup(groupName); err != nil {
		t.Fatalf("store.CreateGroup(%q) failed: %v", groupName, err)
	}
	testGroupsLength(t, store, 1, "after creating a group")

	if err := store.DeleteGroup(groupName); err != nil {
		t.Fatalf("delete group %q failed: %v", groupName, err)
	}

	testGroupsLength(t, store, 0, "after deleting created group")
}

func testGroupsLength(t *testing.T, store tagaa.Store, length int, context string) {
	t.Helper()
	groups, err := store.GetAllGroups()
	if err != nil {
		t.Fatal("getting all groups failed:", err)
	}
	if got, want := len(groups), length; got != want {
		t.Errorf("%s, length of all groups is %d, want %d", context, got, want)
	}
}

func TestDeleteGroup_notEmpty(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groupName := "delete me"

	if err := store.CreateGroup(groupName); err != nil {
		t.Fatalf("store.CreateGroup(%q) failed: %v", groupName, err)
	}
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

func deepEqual(t *testing.T, got interface{}, want interface{}) {
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
	want := []string{groupA}
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
	want := []string{groupName}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.GetAllGroups() \nhave: %#v \nwant: %#v", got, want)
		data, _ := json.Marshal(got)
		fmt.Printf("have: %v\n", string(data))
		data, _ = json.Marshal(want)
		fmt.Printf("want: %v\n", string(data))
	}

}

func TestGetAllGroups(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	groups, err := store.GetAllGroups()
	if err != nil {
		t.Fatal("getting all groups failed:", err)
	}
	if groups == nil {
		t.Error("getting all groups should not return a nil slice")
	}
	if got, want := len(groups), 0; got != want {
		t.Errorf("length of all groups is %d, want %d, groups = %v", got, want, groups)
	}

}