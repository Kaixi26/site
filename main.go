package main

import (
	"bytes"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type Endpoint struct {
	TmplPath string
}

type Config struct {
	Endpoints map[string]Endpoint
	tmpl      *template.Template
}

var config = Config{
	Endpoints: map[string]Endpoint{
		"/":        {"index.go.html"},
		"/contact": {"contact.go.html"},
	},
}

var startTime = time.Now()

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
	),
)

func serveStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
	log.Println(r.URL.Path)
}

func mkServeMarkdown(sourcePath string) func(w http.ResponseWriter, r *http.Request) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		panic(err)
	}

	var html = template.HTML(string(buf.Bytes()))

	return func(w http.ResponseWriter, r *http.Request) {
		config.tmpl.ExecuteTemplate(w, "markdown.go.html", map[string]any{
			"markdown": html,
			"uptime":   math.Round(time.Now().Sub(startTime).Seconds()),
			"cmd":      os.Args[0],
		})
	}
}

func mkServeEndpoint(pattern string, endpoint Endpoint) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pattern {
			http.NotFound(w, r)
			return
		}
		config.tmpl.ExecuteTemplate(w, endpoint.TmplPath, map[string]any{
			"uptime": math.Round(time.Now().Sub(startTime).Seconds()),
			"cmd":    os.Args[0],
		})
	}
}

func main() {

	mux := http.NewServeMux()

	config.tmpl = template.Must(template.ParseGlob("templates/*.go.html"))

	mux.HandleFunc("/static/", serveStatic)

	for pattern, endpoint := range config.Endpoints {
		mux.HandleFunc(pattern, mkServeEndpoint(pattern, endpoint))
		log.Printf("Added handler on pattern '%s' based on template '%s'.\n", pattern, endpoint.TmplPath)
	}

	mux.HandleFunc("/resume", mkServeMarkdown("markdown/resume.md"))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
