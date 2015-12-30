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
	directory   = flag.String("d", "", "the directory that contains the images")
	csvFilename = flag.String("csv", "bulk.csv", "the name of the CSV file")
	pathPrefix  = flag.String("prefix", "", "the path that should be prefixed before the directory and the image name on the CSV file")
	port        = flag.String("port", "8080", "server port")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: bulk-add-csv-ui -d <directory with images> [Options...]\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

type Model struct {
	Prefix string
	Dir    string
	Images []bulk.Image
}

var model = Model{}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run() error {
	flag.Usage = usage
	flag.Parse()

	if *directory == "" {
		usage()
		return fmt.Errorf("argument -d is required")
	}
	model.Dir = *directory

	images, err := bulk.Load(*directory, *csvFilename)
	if err != nil {
		return err
	}
	model.Images = images

	currentPrefix, err := bulk.CurrentPrefix(*directory, *csvFilename)
	if err != nil {
		return err
	}
	if *pathPrefix != "" && *pathPrefix != currentPrefix {
		return fmt.Errorf("path prefix conflict: -prefix is %v, while csv file is %v", *pathPrefix, currentPrefix)
	}
	model.Prefix = currentPrefix

	http.Handle("/", http.HandlerFunc(index))
	http.Handle("/update", http.HandlerFunc(update))
	http.Handle("/img/", http.HandlerFunc(serveImage))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	fmt.Println("Starting server at :" + *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		return err
	}
	//if err := bulk.Save(images, *directory, *csvFilename, prefix); err != nil {
	//	return err
	//}
	return nil
}

func index(w http.ResponseWriter, r *http.Request) {
	render(w, "index", model)
}

func render(w http.ResponseWriter, tmpl string, model interface{}) {
	if err := templates.ExecuteTemplate(w, tmpl+".tmpl", model); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func update(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not parse form", http.StatusInternalServerError)
	}

	// prefix
	model.Prefix = r.PostForm["prefix"][0]
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

	http.Redirect(w, r, "/", http.StatusFound)
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	img := bulk.FindByID(model.Images, id)
	p := filepath.Join(*directory, img.Name)

	f, err := os.Open(p)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not open image %v", p), http.StatusInternalServerError)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read image %v", p), http.StatusInternalServerError)
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
