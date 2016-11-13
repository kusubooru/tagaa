package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const (
	uploadFormFileName = "uploadfile"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		serveUpload(w, r)
	case "POST":
		handleUpload(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func serveUpload(w http.ResponseWriter, r *http.Request) {
	m, err := loadFromCSVFile(globalModel.WorkingDir, globalModel.CSVFilename)
	if err != nil {
		globalModel.Err = fmt.Errorf("Error: could not load from CSV File: %v", err)
	} else {
		globalModel = m
	}
	render(w, uploadTmpl, globalModel)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	m, err := loadFromCSVFile(globalModel.WorkingDir, globalModel.CSVFilename)
	if err != nil {
		globalModel.Err = fmt.Errorf("Error: could not load from CSV File: %v", err)
	} else {
		globalModel = m
	}

	uploadFiles, err := readUploadFiles(globalModel)
	if err != nil {
		//http.Error(w, fmt.Sprintf("Failed to read upload files: %v", err), http.StatusInternalServerError)
		globalModel.Err = fmt.Errorf("Failed to read upload files: %v", err)
		render(w, uploadTmpl, globalModel)
		return
	}

	workingDirBase := filepath.Base(globalModel.WorkingDir)
	zipFilename := filepath.Join(globalModel.WorkingDir, workingDirBase+".zip")
	if err := zipFiles(uploadFiles, zipFilename, workingDirBase); err != nil {
		//http.Error(w, fmt.Sprintf("Failed to zip files: %v", err), http.StatusInternalServerError)
		globalModel.Err = fmt.Errorf("Failed to zip files: %v", err)
		render(w, uploadTmpl, globalModel)
		return
	}

	if err := postFile(zipFilename, *uploadURL, uploadFormFileName); err != nil {
		//http.Error(w, fmt.Sprintf("Failed to upload zip file: %v", err), http.StatusInternalServerError)
		globalModel.Err = fmt.Errorf("Failed to upload zip file: %v", err)
		render(w, uploadTmpl, globalModel)
		return
	}
	globalModel.Success = "Upload was successful!"
	render(w, uploadTmpl, globalModel)
}

type uploadFile struct {
	Name string
	Body []byte
	Info os.FileInfo
}

func readUploadFiles(model *model) ([]*uploadFile, error) {
	var uploadFiles []*uploadFile
	csvFile := filepath.Join(model.WorkingDir, model.CSVFilename)
	csvBody, err := ioutil.ReadFile(csvFile)
	if err != nil {
		return nil, fmt.Errorf("read csv file: %v", err)
	}
	info, err := os.Stat(csvFile)
	if err != nil {
		return nil, fmt.Errorf("stat csv file: %v", err)
	}
	uploadFiles = append(uploadFiles, &uploadFile{Name: model.CSVFilename, Body: csvBody, Info: info})

	for _, img := range model.Images {
		imgFile := filepath.Join(model.WorkingDir, img.Name)
		imgBody, err := ioutil.ReadFile(imgFile)
		if err != nil {
			return nil, fmt.Errorf("read img file: %v", err)
		}
		info, err := os.Stat(imgFile)
		if err != nil {
			return nil, fmt.Errorf("stat img file: %v", err)
		}
		uploadFiles = append(uploadFiles, &uploadFile{Name: img.Name, Body: imgBody, Info: info})
	}
	return uploadFiles, nil
}

func zipFiles(uploadFiles []*uploadFile, zipFilename, dirName string) error {
	zipFile, err := os.Create(zipFilename)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := zipFile.Close(); err == nil {
			err = cerr
		}
	}()

	zw := zip.NewWriter(zipFile)

	for _, file := range uploadFiles {
		header, ierr := zip.FileInfoHeader(file.Info)
		if ierr != nil {
			return fmt.Errorf("file info header: %v", ierr)
		}
		// Putting the files under a directory.
		header.Name = filepath.Join(dirName, header.Name)

		hw, herr := zw.CreateHeader(header)
		if herr != nil {
			return fmt.Errorf("create header: %v", herr)
		}

		_, err = hw.Write(file.Body)
		if err != nil {
			return fmt.Errorf("write zip file: %v", err)
		}
	}

	if err = zw.Close(); err != nil {
		return fmt.Errorf("close zip archive: %v", err)
	}

	return err
}

func postFile(filename, targetURL, formName string) error {
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

	contentType := mw.FormDataContentType()
	if err = mw.Close(); err != nil {
		return fmt.Errorf("error closing multipart writer: %v", err)
	}

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
