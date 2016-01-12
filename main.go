package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kusubooru/local-tagger/bulk"
)

//go:generate go run generate/templates.go

const TheVersion = "1.0.0"

var fns = template.FuncMap{
	"last": func(s []string) string {
		if len(s) == 0 {
			return ""
		}
		return s[len(s)-1]
	},
	"join": strings.Join,
}

var (
	directory   = flag.String("dir", ".", "the directory that contains the images")
	csvFilename = flag.String("csv", "bulk.csv", "the name of the CSV file")
	pathPrefix  = flag.String("prefix", "", "the path that should be prefixed before the directory and the image name on the CSV file")
	port        = flag.String("port", "8080", "server port")
	openBrowser = flag.Bool("openbrowser", true, "open browser automatically")
	version     = flag.Bool("version", false, "print program version")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: bulk-add-csv-ui [Options...]\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

type model struct {
	Err         error
	Prefix      string
	Dir         string
	CSVFilename string
	Images      []bulk.Image
	Version     string
}

var globalModel *model

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Usage = usage
	flag.Parse()

	if *version {
		fmt.Printf("local-tagger%v\n", TheVersion)
		return nil
	}

	d, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	*directory = d

	m, err := loadFromCSVFile(*directory, *csvFilename)
	if err != nil {
		return err
	}
	globalModel = m

	if *pathPrefix != "" {
		globalModel.Prefix = *pathPrefix
	}

	http.Handle("/", http.HandlerFunc(indexHandler))
	http.Handle("/load", http.HandlerFunc(loadHandler))
	http.Handle("/update", http.HandlerFunc(updateHandler))
	http.Handle("/ok/", http.HandlerFunc(okHandler))
	http.Handle("/img/", http.HandlerFunc(serveImage))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	go func() {
		localURL := fmt.Sprintf("http://localhost:%v", *port)
		okURL := fmt.Sprintf("%v/ok", localURL)
		if waitServer(okURL) && *openBrowser && startBrowser(localURL) {
			log.Printf("A browser window should open. If not, please visit %s", localURL)
		} else {
			log.Printf("Please open your web browser and visit %s", localURL)
		}
	}()

	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		return err
	}

	return nil
}

func loadFromCSVFile(dir, csvFilename string) (*model, error) {

	m := &model{Dir: dir, CSVFilename: csvFilename, Version: TheVersion}

	// Loading images from folder
	images, err := bulk.LoadImages(dir)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filepath.Join(dir, csvFilename))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Loading CSV image data
	imagesWithInfo, err := bulk.LoadCSV(f)
	if err != nil {
		return nil, err
	}
	m.Images = bulk.Combine(images, imagesWithInfo)

	// Getting current prefix
	if _, err = f.Seek(0, 0); err != nil {
		return nil, err
	}
	cp, err := bulk.CurrentPrefix(dir, f)
	if err != nil {
		return nil, err
	}
	m.Prefix = cp

	return m, nil
}

func loadHandler(w http.ResponseWriter, r *http.Request) {
	f, h, err := r.FormFile("csvFilename")
	if err != nil {
		globalModel.Err = fmt.Errorf("Error: could not parse multipart file: %v", err)
		render(w, indexTmpl, globalModel)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("Error: could not close multipart file: %v\n", cerr)
		}
	}()

	if err := addFromMultipartFile(globalModel, f); err != nil {
		globalModel.Err = fmt.Errorf("Error: could not load image metadata from multipart CSV File: %v", err)
		render(w, indexTmpl, globalModel)
		return
	}
	globalModel.CSVFilename = h.Filename

	err = saveToCSVFile(globalModel)
	if err != nil {
		globalModel.Err = fmt.Errorf("Error: could not save file to disk: %v", err)
		render(w, indexTmpl, globalModel)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
func addFromMultipartFile(m *model, file multipart.File) error {
	imgMetadata, err := bulk.LoadCSV(file)
	if err != nil {
		return fmt.Errorf("could not load image info from CSV File: %v", err)
	}
	if _, err = file.Seek(0, 0); err != nil {
		return fmt.Errorf("could not seek multipart file: %v", err)
	}
	prefix, err := bulk.CurrentPrefix(m.Dir, file)
	if err != nil {
		return fmt.Errorf("could not read current prefix from multipart file: %v", err)
	}
	m.Prefix = prefix
	m.Images = bulk.Combine(m.Images, imgMetadata)

	return nil
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	m, err := loadFromCSVFile(globalModel.Dir, globalModel.CSVFilename)
	if err != nil {
		globalModel.Err = fmt.Errorf("Error: could not load from CSV File: %v", err)
	} else {
		globalModel = m
	}

	render(w, indexTmpl, globalModel)
}

func render(w http.ResponseWriter, t *template.Template, model interface{}) {
	if err := t.Execute(w, model); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not parse form", http.StatusInternalServerError)
		return
	}

	// prefix
	globalModel.Prefix = r.PostForm["prefix"][0]
	// csvFilename
	globalModel.CSVFilename = r.PostForm["csvFilename"][0]
	for _, img := range globalModel.Images {
		// tags
		areaTags := r.PostForm[fmt.Sprintf("image[%d].tags", img.ID)]
		globalModel.Images[img.ID].Tags = strings.Fields(areaTags[0])
		// source
		globalModel.Images[img.ID].Source = r.PostForm[fmt.Sprintf("image[%d].source", img.ID)][0]
		// rating
		rating := r.PostForm[fmt.Sprintf("image[%d].rating", img.ID)]
		if len(rating) != 0 {
			globalModel.Images[img.ID].Rating = rating[0]
		}
	}

	if err := saveToCSVFile(globalModel); err != nil {
		globalModel.Err = fmt.Errorf("Error: could not save to CSV file: %v", err)
		render(w, indexTmpl, globalModel)
	} else {
		globalModel.Err = nil
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func saveToCSVFile(m *model) error {
	csvFilepath := filepath.Join(m.Dir, m.CSVFilename)
	f, err := os.Create(csvFilepath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err != nil {
			err = cerr
		}
	}()

	if err := bulk.Save(f, m.Images, m.Dir, m.Prefix); err != nil {
		return err
	}
	return err
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v is not a valid image ID", idStr), http.StatusBadRequest)
		return
	}
	img := bulk.FindByID(globalModel.Images, id)
	if img == nil {
		http.Error(w, fmt.Sprintf("no image found with ID: %v", id), http.StatusNotFound)
		return
	}
	p := filepath.Join(*directory, img.Name)

	f, err := os.Open(p)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not open image: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("Error: could not close image file: %v\n", cerr)
		}
	}()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read image: %v", err), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not write image bytes: %v", err), http.StatusInternalServerError)
		return
	}
}

// startBrowser tries to open the URL in a browser, and returns
// whether it succeed.
func startBrowser(url string) bool {
	// try to start the browser
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open"}
	case "windows":
		args = []string{"cmd", "/c", "start"}
	default:
		args = []string{"xdg-open"}
	}
	cmd := exec.Command(args[0], append(args[1:], url)...)
	return cmd.Start() == nil
}

// waitServer waits some time for the http Server to start
// serving url. The return value reports whether it starts.
func waitServer(url string) bool {
	tries := 20
	for tries > 0 {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
		tries--
	}
	return false
}
