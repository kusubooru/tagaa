// +build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultMaxMemory   = 32 << 20 // 32 MB
	maxFileSize        = 50 << 20 // 50 MB
	uploadFormFileName = "uploadfile"
)

var (
	httpAddr  = flag.String("http", ":8081", "http address the server listens")
	uploadDir = flag.String("updir", "tagaa_uploads", "upload directory")
)

type UploadsLimiter interface {
	Reset() error
	CanUpload(int64) (bool, error)
	//AddSpace()
}

type limiter struct {
	maxSpace    int64
	addAmount   int64
	addInterval time.Duration
	addTicker   *time.Ticker
	mu          sync.Mutex
	spaceLeft   int64
	lastAdd     time.Time
}

func NewUploadsLimiter(maxSpace, addAmount int64, addInterval time.Duration) UploadsLimiter {
	lim := &limiter{
		maxSpace: maxSpace,
		//spaceLeft: maxSpace,
		addAmount: addAmount,
		addTicker: time.NewTicker(addInterval),
	}

	go func() {
		for range lim.addTicker.C {
			lim.addSpace()
		}
	}()
	return lim
}

func (lim *limiter) addSpace() {
	lim.mu.Lock()
	defer lim.mu.Unlock()

	lim.lastAdd = time.Now()
	if lim.spaceLeft+lim.addAmount > lim.maxSpace {
		lim.spaceLeft = lim.maxSpace
	} else {
		prevSpace := lim.spaceLeft
		lim.spaceLeft += lim.addAmount
		fmt.Printf("space: %d, add: %d, new space: %d\n", prevSpace, lim.addAmount, lim.spaceLeft)
	}
}

//func (lim *limiter) tryAgainIn(n int64) time.Duration {
//timeSinceLastAdd := time.Now().Sub(lim.lastAdd)
//time.Now().Sub(lim.addInterval).Sub(lim.lastAdd)
//lim.addInterval.Hours()
//intervals := n / lim.addAmount
//return intervals * lim.addInverval

//   n := 100
//   addAmount := 5
//   intervals := n / addAmount
//   //againTime := time.Now()
//   //for i := 0; i < intervals; i++ {
//   //	againTime = againTime.Add(addInterval)
//   //}
//   againIn := time.Second * intervals
//   //fmt.Println("try again in", againTime.Sub(time.Now()))
//   fmt.Println(againIn)
//}

func (lim *limiter) CanUpload(n int64) (bool, error) {
	lim.mu.Lock()
	defer lim.mu.Unlock()

	if lim.spaceLeft == 0 || lim.spaceLeft < n {
		return false, nil
	}

	lim.spaceLeft -= n
	if lim.spaceLeft < 0 {
		lim.spaceLeft = 0
	}

	return true, nil
}

func (lim *limiter) Reset() error {
	lim.mu.Lock()
	defer lim.mu.Unlock()

	lim.spaceLeft = lim.maxSpace
	return nil
}

type app struct {
	limiter UploadsLimiter
}

func main() {
	flag.Parse()

	lim := NewUploadsLimiter(100<<20, 10<<20, time.Second*1)
	fmt.Println(lim)
	//addTicker := time.NewTicker(time.Second * 1)

	//go func() {
	//	for range addTicker.C {
	//		lim.AddSpace()
	//	}
	//}()

	app := app{lim}

	http.HandleFunc("/upload", app.upload)

	mkDirIfNotExist(*uploadDir, 0700)

	fmt.Println("Upload server listening on", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

func mkDirIfNotExist(name string, perm os.FileMode) error {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(name, perm); err != nil {
				return fmt.Errorf("could not make dir: %v", err)
			}
			log.Printf("Created directory %s %v", name, perm)
		}
	}
	return nil
}

func (app *app) upload(w http.ResponseWriter, r *http.Request) {
	log.Println("received upload")

	// Limit max upload.
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	username := r.PostFormValue("username")
	//_ := r.PostFormValue("password")
	// handle authentication

	file, handler, err := r.FormFile(uploadFormFileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	uploadLocation := filepath.Join(*uploadDir, username)
	if err = os.MkdirAll(uploadLocation, 0700); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := time.Now().Format("2006-01-02_15.04.05.000_") + filepath.Base(handler.Filename)
	f, err := os.Create(filepath.Join(uploadLocation, filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	n, err := io.Copy(f, file)
	if err != nil {
		http.Error(w, "file copy failed", http.StatusInternalServerError)
		log.Println("file copy failed:", err)
		return
	}
	fmt.Println("wrote:", n)

	// TODO: ask limiter service for space
	ok, err := app.limiter.CanUpload(n)
	if err != nil || !ok {
		log.Println("add to limiter failed")
		if rerr := os.Remove(f.Name()); rerr != nil {
			log.Println("file cleanup failed:", rerr)
			http.Error(w, "file cleanup failed", http.StatusInternalServerError)
			return
		}
		http.Error(w, "add to limiter failed", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%v", handler.Header)
	// TODO: send mail
}
