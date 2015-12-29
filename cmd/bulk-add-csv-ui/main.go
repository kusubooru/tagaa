package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/kusubooru/shimmie2-tools/bulk"
)

var templates = template.Must(template.ParseGlob("web/*.tmpl"))

var (
	directory   = flag.String("d", "", "the directory that contains the images")
	csvFilename = flag.String("csv", "bulk.csv", "the name of the CSV file")
	pathPrefix  = flag.String("prefix", "", "the path that should be prefixed before the directory and the image name on the CSV file.")
	port        = flag.String("port", "8080", "server port")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: bulk-add-csv-ui -d <directory with images> [options...]\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

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
	images, err := bulk.Load(*directory, *csvFilename)
	if err != nil {
		return err
	}
	fmt.Fprintf(ioutil.Discard, "", images)

	prefix, err := bulk.CurrentPrefix(*directory, *csvFilename)
	if err != nil {
		return err
	}
	fmt.Fprintf(ioutil.Discard, prefix)

	http.Handle("/", http.HandlerFunc(index))
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
	render(w, "index", nil)
	//w.Write([]byte(staticHtml))
}

func render(w http.ResponseWriter, tmpl string, model interface{}) {
	if err := templates.ExecuteTemplate(w, tmpl+".tmpl", model); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
