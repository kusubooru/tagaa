package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/kusubooru/shimmie2-tools/bulk"
)

var fns = template.FuncMap{
	"last": func(s []string) string {
		if len(s) == 0 {
			return ""
		}
		return s[len(s)-1]
	},
}

var templates = template.Must(template.New("").Funcs(fns).ParseGlob("web/*.tmpl"))

var (
	directory   = flag.String("dir", ".", "the directory that contains the images")
	csvFilename = flag.String("csv", "bulk.csv", "the name of the CSV file")
	pathPrefix  = flag.String("prefix", "", "the path that should be prefixed before the directory and the image name on the CSV file")
	port        = flag.String("port", "8080", "server port")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: bulk-add-csv-ui [Options...]\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

type Model struct {
	Err         error
	Prefix      string
	Dir         string
	CSVFilename string
	Images      []bulk.Image
}

var model *Model

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run() error {
	flag.Usage = usage
	flag.Parse()

	d, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	*directory = d

	m, err := loadFromCSVFile(*directory, *csvFilename)
	if err != nil {
		return err
	}
	model = m

	if *pathPrefix != "" {
		model.Prefix = *pathPrefix
	}

	http.Handle("/", http.HandlerFunc(indexHandler))
	http.Handle("/load", http.HandlerFunc(loadHandler))
	http.Handle("/update", http.HandlerFunc(updateHandler))
	http.Handle("/img/", http.HandlerFunc(serveImage))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	go func() {
		localURL := fmt.Sprintf("http://localhost:%v", *port)
		if err := browserOpen(localURL); err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not open browser, please visit %v manually.\n", localURL)
		}
	}()

	fmt.Println("Starting server at :" + *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		return err
	}

	return nil
}

func loadFromCSVFile(dir, csvFilename string) (*Model, error) {
	m := &Model{Dir: dir, CSVFilename: csvFilename}

	// Loading images from folder
	images, err := bulk.LoadImages(dir)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filepath.Join(dir, csvFilename), os.O_RDONLY|os.O_CREATE, 0644)
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
	r.ParseMultipartForm(32 << 20)
	f, h, err := r.FormFile("csvFilename")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	img, err := bulk.LoadCSV(f)
	if err != nil {
		model.Err = fmt.Errorf("Error: could not load image info from CSV File: %v", err)
		render(w, "index", model)
	} else {
		model.CSVFilename = h.Filename
		model.Images = bulk.Combine(model.Images, img)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	m, err := loadFromCSVFile(model.Dir, model.CSVFilename)
	if err != nil {
		model.Err = fmt.Errorf("Error: could not load from CSV File: %v", err)
	} else {
		model = m
	}

	render(w, "index", model)
}

func render(w http.ResponseWriter, tmpl string, model interface{}) {
	if err := templates.ExecuteTemplate(w, tmpl+".tmpl", model); err != nil {
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
	model.Prefix = r.PostForm["prefix"][0]
	// csvFilename
	model.CSVFilename = r.PostForm["csvFilename"][0]
	for _, img := range model.Images {
		// tags
		areaTags := r.PostForm[fmt.Sprintf("image[%d].tags", img.ID)]
		model.Images[img.ID].Tags = strings.Fields(areaTags[0])
		// source
		model.Images[img.ID].Source = r.PostForm[fmt.Sprintf("image[%d].source", img.ID)][0]
		// rating
		rating := r.PostForm[fmt.Sprintf("image[%d].rating", img.ID)]
		if len(rating) != 0 {
			model.Images[img.ID].Rating = rating[0]
		}
	}

	if err := saveToCSVFile(model); err != nil {
		model.Err = fmt.Errorf("Error: could not save to CSV file: %v", err)
		render(w, "index", model)
	} else {
		model.Err = nil
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func saveToCSVFile(m *Model) error {
	return bulk.Save(m.Images, m.Dir, m.CSVFilename, m.Prefix)
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v is not a valid image ID", idStr), http.StatusBadRequest)
		return
	}
	img := bulk.FindByID(model.Images, id)
	if img == nil {
		http.Error(w, fmt.Sprintf("no image found with ID: %v", id), http.StatusNotFound)
		return
	}
	p := filepath.Join(*directory, img.Name)

	f, err := os.Open(p)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not open image %v", p), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read image %v", p), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func browserOpen(input string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", input).Start()
	case "windows":
		err = exec.Command("cmd", "/C", "start", "", input).Start()
	case "darwin":
		err = exec.Command("open", input).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		return err
	}
	return nil
}
