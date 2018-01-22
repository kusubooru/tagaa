package tagaa

import (
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("item not found")
	ErrGroupNotEmpty = errors.New("group not empty")
)

type Rating int

const (
	Unknown Rating = iota
	Safe
	Questionable
	Explicit
)

type Image struct {
	ID      uint64
	Name    string
	Tags    string
	Source  string
	Rating  Rating
	Added   time.Time
	Updated time.Time
	Size    int
	Width   int
	Height  int
	Hash    string
	Ext     string
}

type Group struct {
	Name   string
	Images []uint64
	Size   int
}

type Store interface {
	CreateGroup(name string) error
	DeleteGroup(name string) error
	GetGroup(name string) (Group, error)
	GetAllGroups() ([]Group, error)
	GetGroupImages(name string) ([]*Image, error)
	AddImage(group string, img *Image) error
	UpdateImage(group string, img *Image) error
	DeleteImage(group string, id uint64) error
	GetImage(group string, id uint64) (*Image, error)
	GetImageData(hash string) ([]byte, error)
	Close() error
}
