package glsl

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

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

	page := 0
	if p, ok := r.URL.Query()["page"]; ok && len(p) == 1 {
		page, err = strconv.Atoi(p[0])
		if err != nil {
			log.Errorf(err, "invalid page: %v", p)
			page = 0
		}
	}

	gallery := Gallery{
		Page: page,
	}
	db := s.db.Order("modified desc").Limit(perPage).Offset(page * perPage).
		Preload("Versions").Find(&gallery.Effects)

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
