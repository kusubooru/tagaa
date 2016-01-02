package bulk

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Image struct {
	ID     int
	Name   string
	Tags   []string
	Source string
	Rating string
}

var supportedExt = []string{"gif", "jpeg", "jpg", "png", "swf"}

func isSupportedType(name string) bool {
	fname := strings.ToLower(name)
	for _, ext := range supportedExt {
		// The only possible returned error is ErrBadPattern, when pattern is
		// malformed. Patterns like *.jpg are never malformed so we ignore the
		// error.
		matches, _ := filepath.Match("*."+ext, fname)
		if matches {
			return true
		}
	}
	return false
}

func loadImages(dir string) ([]Image, error) {
	var images []Image

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	id := 0
	for _, f := range files {
		if !f.IsDir() {
			if isSupportedType(f.Name()) {
				img := Image{ID: id, Name: f.Name()}
				images = append(images, img)
				id++
			}
		}
	}
	return images, nil
}

func loadCSV(path string) ([]Image, error) {
	var images []Image

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		img := Image{
			Name:   filepath.Base(record[0]),
			Tags:   strings.Split(record[1], " "),
			Source: record[2],
			Rating: record[3],
		}
		images = append(images, img)
	}
	return images, nil
}

func Load(dir, csvFilename string) ([]Image, error) {
	images, err := loadImages(dir)
	if err != nil {
		return nil, err
	}

	csvFile := filepath.Join(dir, csvFilename)
	// if csv file doesn't exist
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		return images, nil
	}

	info, err := loadCSV(csvFile)
	if err != nil {
		return nil, err
	}
	imagesWithInfo := combine(images, info)

	return imagesWithInfo, nil
}

func combine(images, imagesWithInfo []Image) []Image {
	for _, info := range imagesWithInfo {
		img := findByName(images, info.Name)
		if img != nil {
			img.Source = info.Source
			img.Rating = info.Rating
			for _, t := range info.Tags {
				img.Tags = append(img.Tags, t)
			}
		}
	}
	return images
}

func findByName(image []Image, name string) *Image {
	i := sort.Search(len(image), func(i int) bool { return image[i].Name >= name })
	if i < len(image) && image[i].Name == name {
		return &image[i]
	}
	return nil
}

type byName []Image

func (img byName) Len() int           { return len(img) }
func (img byName) Swap(i, j int)      { img[i], img[j] = img[j], img[i] }
func (img byName) Less(i, j int) bool { return img[i].Name < img[j].Name }

func FindByID(image []Image, id string) *Image {
	i := sort.Search(len(image), func(i int) bool { return image[i].Name >= id })
	if i < len(image) && image[i].Name == id {
		return &image[i]
	}
	return nil
}

type ByID []Image

func (img ByID) Len() int           { return len(img) }
func (img ByID) Swap(i, j int)      { img[i], img[j] = img[j], img[i] }
func (img ByID) Less(i, j int) bool { return img[i].ID < img[j].ID }

func CurrentPrefix(dir, csvFilename string) (string, error) {
	csvFile := filepath.Join(dir, csvFilename)
	// if csv file doesn't exist
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		return "", fmt.Errorf("%v does not exist", csvFile)
	}

	f, err := os.Open(csvFile)
	if err != nil {
		return "", err
	}

	r := csv.NewReader(f)
	record, err := r.Read()
	if err == io.EOF {
		return "", fmt.Errorf("%v appears to be empty", csvFile)
	}

	folder := filepath.Base(dir)
	sep := fmt.Sprintf("%c", filepath.Separator)
	prefix := sep
	parts := strings.Split(record[0], sep)
	for _, p := range parts {
		if p == folder {
			break
		}
		prefix = filepath.Join(prefix, p)
	}
	return prefix, nil
}

func Save(images []Image, dir, csvFilename, prefix string) error {
	csvFile := filepath.Join(dir, csvFilename)
	f, err := os.Create(csvFile)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	w.WriteAll(toRecords(images, dir, prefix))

	if err := w.Error(); err != nil {
		return fmt.Errorf("error writing csv: %v", err)
	}
	return nil
}

func toRecords(images []Image, dir, prefix string) [][]string {
	var records [][]string
	for _, img := range images {
		record := toRecord(img, dir, prefix)
		records = append(records, record)
	}
	return records
}

func toRecord(img Image, dir, prefix string) []string {
	var record []string
	record = append(record, filepath.Join(prefix, filepath.Base(dir), img.Name))
	record = append(record, strings.Join(img.Tags, " "))
	record = append(record, img.Source)
	record = append(record, img.Rating)
	record = append(record, "")
	return record
}
