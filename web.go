package glsl

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"gopkg.in/src-d/go-log.v1"
)

const (
	galleryPath = "assets/gallery.html"
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
	// r.Get("/", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("welcome"))
	// })
	r.Get("/", s.gallery)
	r.Get("/images/{id:[0-9]+}.png", s.image)

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

	var effects []Effect
	db := s.db.Order("modified desc").Limit(40).Preload("Versions").Find(&effects)

	errors := db.GetErrors()
	for _, err = range errors {
		log.Errorf(err, "error querying database")
	}
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, effects)
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
