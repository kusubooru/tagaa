package boltdb

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/tagaa/tagaa"
)

func (db *store) CreateGroup(groupName string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupBucket))

		v := b.Get([]byte(groupName))
		if len(v) != 0 {
			return fmt.Errorf("group exists")
		}

		g := &tagaa.Group{Name: groupName}
		return put(b, []byte(groupName), g)
	})
	return err
}

func (db *store) DeleteGroup(name string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupBucket))
		return b.Delete([]byte(name))
	})
	return err

}

func (db *store) GetAllGroups() ([]*tagaa.Group, error) {
	var groups []*tagaa.Group
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupBucket))

		return b.ForEach(func(k, v []byte) error {
			g := new(tagaa.Group)
			if err := decode(v, g); err != nil {
				return err
			}
			groups = append(groups, g)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (db *store) GetGroupImages(groupName string) ([]*tagaa.Image, error) {
	var images []*tagaa.Image
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupBucket))
		g := new(tagaa.Group)
		if err := get(b, []byte(groupName), g); err != nil {
			return err
		}
		b = tx.Bucket([]byte(imageBucket))
		for _, imgID := range g.Images {
			img := new(tagaa.Image)
			if err := get(b, uitob(imgID), img); err != nil {
				return err
			}
			images = append(images, img)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (db *store) AddImage(groupName string, img *tagaa.Image) error {
	err := db.Update(func(tx *bolt.Tx) error {
		// First we add the img to imageBucket.
		b := tx.Bucket([]byte(imageBucket))

		// Create image ID.
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		img.ID = id
		img.Added = time.Now()
		if err := put(b, uitob(id), img); err != nil {
			return err
		}

		// Then we get the image group.
		g := new(tagaa.Group)
		b = tx.Bucket([]byte(groupBucket))
		v := b.Get([]byte(groupName))
		// Group does not exist. Create a new one.
		if len(v) == 0 {
			newGroup := &tagaa.Group{Name: groupName}
			if err := put(b, []byte(groupName), newGroup); err != nil {
				return err
			}
			v = b.Get([]byte(groupName))
		}
		if err := decode(v, g); err != nil {
			return err
		}

		// Then we add the new image id to the group and update the size.
		g.Images = append(g.Images, id)
		g.Size += img.Size

		// Then we store the group again.
		return put(b, []byte(groupName), g)
	})
	return err
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func uitob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func (db *store) UpdateImage(group string, img *tagaa.Image) error {
	//err := db.Update(func(tx *bolt.Tx) error {
	//	// Make sure our image has an ID.
	//	if img.ID == 0 {
	//		return fmt.Errorf("trying to update image without ID")
	//	}
	//	// Take the group.
	//	buf := bytes.Buffer{}
	//	gb := tx.Bucket([]byte(groupBucket))
	//	g := new(tagaa.Group)
	//	v := b.Get([]byte(groupName))
	//	if len(v) == 0 {
	//		return fmt.Errorf("trying to update image of non-existent group")
	//	}
	//	if _, werr := buf.Write(v); werr != nil {
	//		return werr
	//	}
	//	if err := gob.NewDecoder(&buf).Decode(&g); err != nil {
	//		return err
	//	}
	//	// Make sure that img.ID exists in the group's image IDs.
	//	sort.Slice(g.Images, func(i, j int) bool { return g.Images[i] < g.Images[j] })
	//	i := sort.Search(len(g.Images), func(i int) bool { return g.Images[i] >= img.ID })
	//	if i < len(g.Images) && g.Images[i] == img.ID {
	//		// img.ID is present at g.Images[i]
	//	} else {
	//		// img.ID is not present in g.Images,
	//		// but i is the index where it would be inserted.
	//		return fmt.Errorf("group %q does not contain image ID: %d", group, img.ID)
	//	}

	//	// We can safely update the image.
	//	ib := tx.Bucket([]byte(imageBucket))

	//	// First get the old image.
	//	oldImage := new(tagaa.Image)
	//	v := ib.Get(img.ID)

	//	img.Updated = time.Now()
	//	if err := gob.NewEncoder(&buf).Encode(img); err != nil {
	//		return err
	//	}
	//	if err := b.Put(uitob(id), buf.Bytes()); err != nil {
	//		return err
	//	}

	//	// Then we get the image group.
	//	buf.Reset()

	//	// Then we add the new image id to the group and update the size.
	//	g.Images = append(g.Images, id)
	//	g.Size += img.Size

	//	// Then we store the group again.
	//	buf.Reset()
	//	if err := gob.NewEncoder(&buf).Encode(&g); err != nil {
	//		return err
	//	}
	//	return b.Put([]byte(groupName), buf.Bytes())
	//})
	//return err
	return fmt.Errorf("not implemented")
}

func get(b *bolt.Bucket, key []byte, v interface{}) error {
	return decode(b.Get(key), v)
}

func decode(data []byte, v interface{}) error {
	buf := bytes.Buffer{}
	if _, err := buf.Write(data); err != nil {
		return err
	}
	if err := gob.NewDecoder(&buf).Decode(v); err != nil {
		return err
	}
	return nil
}

func put(b *bolt.Bucket, key []byte, v interface{}) error {
	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}
	if err := b.Put(key, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *store) DeleteImage(group string, id uint64) error { return fmt.Errorf("not implemented") }
func (db *store) GetImage(group string, id uint64) (*tagaa.Image, error) {
	return nil, fmt.Errorf("not implemented")
}

func (db *store) GetImageData(hash string) ([]byte, error) {
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket([]byte(blobBucket)).Get([]byte(hash))
		_, werr := buf.Write(value)
		return werr
	})
	return buf.Bytes(), err
}
