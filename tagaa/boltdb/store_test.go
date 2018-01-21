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

func setup() (tagaa.Store, *os.File) {
	f, err := ioutil.TempFile("", "tagaa_boltdb_tmpfile_")
	if err != nil {
		log.Fatal("could not create boltdb temp file for tests:", err)
	}
	return NewStore(f.Name()), f
}

func teardown(store tagaa.Store, tmpfile *os.File) {
	store.Close()
	if err := os.Remove(tmpfile.Name()); err != nil {
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
