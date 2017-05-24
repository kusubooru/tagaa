// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/kusubooru/tklid"
)

const (
	uploadFormFileName = "uploadfile"
	seed               = 42
	dummySize          = 5e7
	uploadsNumber      = 10
)

var uploadURL = flag.String("uploadurl", "http://localhost:8081/upload", "URL to upload zip file")

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	// create dummy file
	f, err := os.Create("dummy.zip")
	if err != nil {
		return fmt.Errorf("could not create dummy.zip: %v", err)
	}
	defer f.Close()

	if err := f.Truncate(dummySize); err != nil {
		return fmt.Errorf("could not truncate dummy data: %v", err)
	}

	randomID := tklid.New(seed)
	fmt.Println("id:", randomID)
	errors := make(chan error)
	defer close(errors)
	done := make(chan struct{})
	defer close(done)

	for i := 0; i < uploadsNumber; i++ {
		go func(num int) {
			fmt.Printf("starting upload #%d\n", num)
			if err := postFile(f.Name(), *uploadURL, uploadFormFileName, randomID); err != nil {
				errors <- fmt.Errorf("failed to upload dummy zip file: %v", err)
				return
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < uploadsNumber; i++ {
		select {
		case err := <-errors:
			fmt.Printf("error: %v\n", err)
		case <-done:
			fmt.Println("done")
		}
	}
	return nil
}

func postFile(filename, targetURL, formName, randomID string) error {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)

	// this step is very important
	formFile, err := mw.CreateFormFile(formName, filename)
	if err != nil {
		return fmt.Errorf("error creating form file: %v", err)
	}

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening upload file: %v", err)
	}

	_, err = io.Copy(formFile, f)
	if err != nil {
		return fmt.Errorf("error copying to form file: %v", err)
	}

	// Add the other fields
	if formFile, err = mw.CreateFormField("randomID"); err != nil {
		return fmt.Errorf("error creating form field randomID: %v", err)
	}
	if _, err = formFile.Write([]byte(randomID)); err != nil {
		return fmt.Errorf("error writing value for form field randomID: %v", err)
	}

	if err = mw.Close(); err != nil {
		return fmt.Errorf("error closing multipart writer: %v", err)
	}

	contentType := mw.FormDataContentType()
	resp, err := http.Post(targetURL, contentType, buf)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("%v %v: %d %v", resp.Request.Method, resp.Request.URL, resp.StatusCode, err)
		}
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr == nil {
			err = cerr
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		data, rerr := ioutil.ReadAll(resp.Body)
		if rerr != nil {
			return fmt.Errorf("error reading response body: %v", rerr)
		}
		return fmt.Errorf("%v %v: %d %s", resp.Request.Method, resp.Request.URL, resp.StatusCode, string(data))
	}
	return err
}
