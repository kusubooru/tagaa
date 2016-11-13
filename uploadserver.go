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
	"strings"
	"time"
)

const (
	defaultMaxMemory   = 32 << 20 // 32 MB
	maxFileSize        = 50 << 20 // 50 MB
	uploadFormFileName = "uploadfile"
)

var (
	httpAddr  = flag.String("http", ":8081", "http address the server listens")
	uploadDir = flag.String("upload-dir", "ltuploads", "http address the server listens")
)

func main() {
	http.HandleFunc("/upload", upload)

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

func upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	file, handler, err := r.FormFile(uploadFormFileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	ip := GetOriginalIP(r)
	uploadLocation := filepath.Join(*uploadDir, time.Now().Format("2006-01-02"), ip)
	if err = os.MkdirAll(uploadLocation, 0700); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f, err := os.Create(filepath.Join(uploadLocation, filepath.Base(handler.Filename)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	fmt.Fprintf(w, "%v", handler.Header)
	io.Copy(f, file)
}

// GetOriginalIP gets the original IP of the HTTP for the case of being behind
// a proxy. It searches for the X-Forwarded-For header.
func GetOriginalIP(r *http.Request) string {
	x := r.Header.Get("X-Forwarded-For")
	if x != "" && strings.Contains(r.RemoteAddr, "127.0.0.1") {
		// format is comma separated
		return strings.Split(x, ",")[0]
	}
	// it also contains the port
	return strings.Split(r.RemoteAddr, ":")[0]
}
