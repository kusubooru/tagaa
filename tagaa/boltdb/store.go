package boltdb

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/tagaa/tagaa"
)

func (db *store) CreateGroup(groupName string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(groupName))
		switch err {
		case bolt.ErrBucketExists:
			return tagaa.ErrGroupExists
		case nil:
			return nil
		default:
			return err
		}
	})
	return err
}

func (db *store) DeleteGroup(groupName string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupName))
		if b == nil {
			return tagaa.ErrGroupNotFound
		}
		// If the bucket contains a key, then it is not empty and thus not safe
		// for deletion.
		key, _ := b.Cursor().First()
		if key != nil {
			return tagaa.ErrGroupNotEmpty
		}
		return tx.DeleteBucket([]byte(groupName))
	})
	return err
}

func (db *store) GetAllGroups() ([]string, error) {
	var groups = make([]string, 0)
	err := db.View(func(tx *bolt.Tx) error {
		err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			groups = append(groups, string(name))
			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (db *store) GetGroupImages(groupName string) ([]*tagaa.Image, error) {
	var images []*tagaa.Image
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupName))
		if b == nil {
			return tagaa.ErrGroupNotFound
		}
		err := b.ForEach(func(k []byte, v []byte) error {
			img := new(tagaa.Image)
			if err := decode(v, img); err != nil {
				return err
			}
			images = append(images, img)
			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (db *store) AddImage(groupName string, img *tagaa.Image) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupName))
		if b == nil {
			newBucket, err := tx.CreateBucket([]byte(groupName))
			if err != nil {
				return err
			}
			b = newBucket
		}

		// Create image ID.
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		img.ID = id
		img.Added = time.Now()
		return put(b, uitob(id), img)
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
	data := b.Get(key)
	if data == nil {
		return tagaa.ErrImageNotFound
	}
	return decode(data, v)
}

func decode(data []byte, v interface{}) error {
	buf := bytes.Buffer{}
	if _, err := buf.Write(data); err != nil {
		return err
	}
	return json.NewDecoder(&buf).Decode(v)
}

func put(b *bolt.Bucket, key []byte, v interface{}) error {
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}
	return b.Put(key, buf.Bytes())
}

func (db *store) DeleteImage(groupName string, id uint64) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupName))
		if b == nil {
			return tagaa.ErrGroupNotFound
		}
		return b.Delete(uitob(id))
	})
	return err
}

func (db *store) GetImage(groupName string, id uint64) (*tagaa.Image, error) {
	var img = new(tagaa.Image)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(groupName))
		if b == nil {
			return tagaa.ErrGroupNotFound
		}
		return get(b, uitob(id), img)
	})
	return img, err
}

func (db *store) GetImageData(hash string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
