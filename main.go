package main

import (
	"bytes"
	"flag"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type Endpoint struct {
	TmplPath string
}

type Config struct {
	Http struct {
		Tls      bool
		Addr     string
		CertFile string
		KeyFile  string
	}
	endpoints map[string]Endpoint
	tmpl      *template.Template
}

var config = Config{
	endpoints: map[string]Endpoint{
		"/":        {"index.go.html"},
		"/contact": {"contact.go.html"},
	},
}

var startTime = time.Now()

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM, meta.Meta),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
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
		return func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}
	}

	var buf bytes.Buffer
	context := parser.NewContext()
	if err := md.Convert(source, &buf, parser.WithContext(context)); err != nil {
		panic(err)
	}
	metaData := meta.Get(context)

	var html = template.HTML(string(buf.Bytes()))

	return func(w http.ResponseWriter, r *http.Request) {
		config.tmpl.ExecuteTemplate(w, "markdown.go.html", map[string]any{
			"markdown": html,
			"uptime":   math.Round(time.Now().Sub(startTime).Seconds()),
			"cmd":      os.Args[0],
			"title":    metaData["Title"],
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

func mkServeArticle(sourcePath string) func(w http.ResponseWriter, r *http.Request) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}
	}

	var buf bytes.Buffer
	context := parser.NewContext()
	if err := md.Convert(source, &buf, parser.WithContext(context)); err != nil {
		panic(err)
	}
	metaData := meta.Get(context)

	var html = template.HTML(string(buf.Bytes()))

	return func(w http.ResponseWriter, r *http.Request) {
		config.tmpl.ExecuteTemplate(w, "article.go.html", map[string]any{
			"markdown": html,
			"uptime":   math.Round(time.Now().Sub(startTime).Seconds()),
			"cmd":      os.Args[0],
			"title":    metaData["Title"],
		})
	}
}

func serveBlog(w http.ResponseWriter, r *http.Request) {
	type BlogEntry struct {
		Pattern string
		Title   string
		Date    string
	}

	if r.URL.Path != "/blog/" {
		mkServeArticle("."+r.URL.Path+".md")(w, r)
		//http.NotFound(w, r)
		return
	}

	var blogEntries = []BlogEntry{}
	dir, err := os.ReadDir("blog/")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range dir {
		if file.IsDir() || filepath.Ext(file.Name()) != ".md" {
			continue
		}

		data, err := os.ReadFile("blog/" + file.Name())
		if err != nil {
			log.Fatal(err)
		}

		var buf bytes.Buffer
		context := parser.NewContext()
		if err := md.Convert(data, &buf, parser.WithContext(context)); err != nil {
			panic(err)
		}
		metaData := meta.Get(context)

		entry := BlogEntry{}
		entry.Pattern = strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		if v, ok := metaData["Title"]; ok == true {
			entry.Title = v.(string)
		}
		if v, ok := metaData["Date"]; ok == true {
			layoutIn := "02/01/2006"
			layoutOut := "02 Jan 2006"
			date, err := time.Parse(layoutIn, v.(string))
			if err != nil {
				panic(err)
			}
			entry.Date = date.Format(layoutOut)
		}

		blogEntries = append(blogEntries, entry)
	}
	sort.Slice(blogEntries, func(i, j int) bool {
		return blogEntries[i].Date < blogEntries[j].Date
	})
	config.tmpl.ExecuteTemplate(w, "blog.go.html", map[string]any{
		"uptime":      math.Round(time.Now().Sub(startTime).Seconds()),
		"cmd":         os.Args[0],
		"blogEntries": blogEntries,
	})
}

func setupConfig() {
	configPath := flag.String("config", "config.toml", "Path for the configuration.")
	flag.BoolVar(&config.Http.Tls, "tls", false, "Wether to enable TLS.")
	flag.StringVar(&config.Http.Addr, "addr", ":8080", "Address to bind.")
	flag.StringVar(&config.Http.CertFile, "certKey", "", "Path for the TLS certificate key.")
	flag.StringVar(&config.Http.KeyFile, "keyFile", "", "Path for the TLS private key file.")
	flag.Parse()

	source, err := os.ReadFile(*configPath)
	if err != nil {
		panic(err)
	}
	tomlData := string(source)

	_, err = toml.Decode(tomlData, &config)
	if err != nil {
		log.Fatal(err)
	}

	flag.Parse()
}

func main() {

	setupConfig()

	mux := http.NewServeMux()

	config.tmpl = template.Must(template.ParseGlob("templates/*.go.html"))

	mux.HandleFunc("/static/", serveStatic)

	for pattern, endpoint := range config.endpoints {
		mux.HandleFunc(pattern, mkServeEndpoint(pattern, endpoint))
		log.Printf("Added template '%s' for pattern '%s'", endpoint.TmplPath, pattern)
	}

	mux.HandleFunc("/resume", mkServeMarkdown("markdown/resume.md"))
	//mux.HandleFunc("/test", mkServeMarkdown("test.md"))
	mux.HandleFunc("/blog/", serveBlog)

	switch config.Http.Tls {
	case false:
		log.Printf("Listening http on %s", config.Http.Addr)
		log.Fatal(http.ListenAndServe(config.Http.Addr, mux))
	case true:
		log.Printf("Listening https on %s", config.Http.Addr)
		log.Fatal(http.ListenAndServeTLS(config.Http.Addr, config.Http.CertFile, config.Http.KeyFile, mux))
	}
}
