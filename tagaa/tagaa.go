package tagaa

import (
	"errors"
	"time"
)

var (
	ErrImageNotFound = errors.New("image not found")
	ErrGroupNotEmpty = errors.New("group not empty")
	ErrGroupExists   = errors.New("group already exists")
	ErrGroupNotFound = errors.New("group not found")
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

type Store interface {
	CreateGroup(name string) error
	DeleteGroup(name string) error
	GetAllGroups() ([]string, error)
	GetGroupImages(name string) ([]*Image, error)
	AddImage(group string, img *Image) error
	UpdateImage(group string, img *Image) error
	DeleteImage(group string, id uint64) error
	GetImage(group string, id uint64) (*Image, error)
	GetImageData(hash string) ([]byte, error)
	Close() error
}
