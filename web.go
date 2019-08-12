package glsl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"gopkg.in/src-d/go-log.v1"
)

const (
	galleryPath = "assets/gallery.html"
	perPage     = 40
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{
		db: db,
	}
}

func (s *Server) Start() {
	r := chi.NewRouter()

	r.Get("/", s.gallery)
	r.Get("/e", s.editor)
	r.Get("/diff", s.diff)
	r.Get("/images/{id:[0-9]+}.png", s.image)
	r.Get("/js/{name:[a-z]+\\.js}", s.js)
	r.Get("/css/{name:[a-z]+\\.(css|png)}", s.css)
	r.Get("/item/{effect:[0-9]+}", s.item)
	r.Get("/item/{effect:[0-9]+}.{version:[0-9]+}", s.item)

	http.ListenAndServe(":3000", r)
}

func loadTemplate(name string) (*template.Template, error) {
	f, err := os.Open(name)
	if err != nil {
		log.Errorf(err, "cannot find asset %v", name)
		return nil, err
	}
	defer f.Close()

	tmplText, err := ioutil.ReadAll(f)
	if err != nil {
		log.Errorf(err, "cannot load asset %v", name)
		return nil, err
	}

	tmpl, err := template.New("template").Parse(string(tmplText))
	if err != nil {
		log.Errorf(err, "cannot parse template %v", name)
		return nil, err
	}

	return tmpl, nil
}

func (s *Server) gallery(w http.ResponseWriter, r *http.Request) {
	tmpl, err := loadTemplate(galleryPath)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	page := 0
	if p, ok := r.URL.Query()["page"]; ok && len(p) == 1 {
		page, err = strconv.Atoi(p[0])
		if err != nil {
			log.Errorf(err, "invalid page: %v", p)
			page = 0
		}
	}

	start := time.Now()

	gallery := Gallery{
		Page: page,
	}
	db := s.db.Order("modified desc").Limit(perPage).Offset(page * perPage).
		Preload("Versions").Find(&gallery.Effects)

	log.With(log.Fields{
		"duration": time.Since(start),
	}).Infof("database queried")

	errors := db.GetErrors()
	for _, err = range errors {
		log.Errorf(err, "error querying database")
	}
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, gallery)
	if err != nil {
		log.Errorf(err, "cannot render template %v", galleryPath)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	_, err = w.Write(buf.Bytes())
	if err != nil {
		log.Errorf(err, "cannot write page")
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	println("id", id)

	name := fmt.Sprintf("images/%v.png", id)
	f, err := os.Open(name)
	if err != nil {
		log.Errorf(err, "cannot load image %v", galleryPath)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "image/png")
	io.Copy(w, f)
}

func (s *Server) css(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	println("css", name)

	path := fmt.Sprintf("assets/css/%v", name)
	f, err := os.Open(path)
	if err != nil {
		log.Errorf(err, "cannot load asset %v", path)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	if strings.HasSuffix(name, ".png") {
		w.Header().Set("Content-Type", "application/png")
	} else {
		w.Header().Set("Content-Type", "text/css")
	}
	io.Copy(w, f)
}

func (s *Server) js(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	println("js", name)

	path := fmt.Sprintf("assets/js/%v", name)
	f, err := os.Open(path)
	if err != nil {
		log.Errorf(err, "cannot load asset %v", path)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/javascript")
	io.Copy(w, f)
}

func (s *Server) editor(w http.ResponseWriter, r *http.Request) {
	path := "assets/editor.html"
	f, err := os.Open(path)
	if err != nil {
		log.Errorf(err, "cannot load asset %v", path)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/html")
	io.Copy(w, f)
}

func (s *Server) diff(w http.ResponseWriter, r *http.Request) {
	path := "assets/diff.html"
	f, err := os.Open(path)
	if err != nil {
		log.Errorf(err, "cannot load asset %v", path)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/html")
	io.Copy(w, f)
}

type item struct {
	Code   string `json:"code"`
	User   string `json:"user"`
	Parent string `json:"parent"`
}

func (s *Server) item(w http.ResponseWriter, r *http.Request) {
	effectText := chi.URLParam(r, "effect")
	effectID, _ := strconv.Atoi(effectText)

	versionID := -1
	versionText := chi.URLParam(r, "version")
	if versionText != "" {
		versionID, _ = strconv.Atoi(versionText)
	}

	println("item", effectID, versionID)

	var effect Effect
	db := s.db.Preload("Versions").Find(&effect, effectID)

	var err error
	errs := db.GetErrors()
	for _, err = range errs {
		log.Errorf(err, "cannot load item %v.%v", effectID, versionID)
	}
	if err != nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	if versionID >= len(effect.Versions) {
		http.Error(w, http.StatusText(404), 404)
		return
	}
	if versionID < 0 {
		versionID = len(effect.Versions) - 1
	}

	parent := fmt.Sprintf("%v.%v", effect.ParentID, effect.ParentVersion)
	// ParentID can be null but uint cannot. Use the default values to detect
	// orphan effects.
	if parent == "0.0" {
		parent = ""
	}

	i := item{
		Code:   effect.Versions[versionID].Code,
		User:   effect.User,
		Parent: parent,
	}

	m, err := json.Marshal(i)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	println("item", string(m))
	w.Write(m)
}

type Gallery struct {
	Page    int
	Effects []Effect
}

func (g Gallery) HasPreviousPage() bool {
	return g.Page > 0
}
func (g Gallery) PreviousPage() int {
	return g.Page - 1
}

func (g Gallery) NextPage() int {
	return g.Page + 1
}
