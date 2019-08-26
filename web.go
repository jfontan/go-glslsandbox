package glsl

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"gopkg.in/src-d/go-log.v1"
)

const (
	imagesDir   = "images"
	galleryPath = "assets/gallery.html"
	perPage     = 40
)

type Server struct {
	db *Database
}

func NewServer(db *gorm.DB) *Server {
	return &Server{
		db: NewDatabase(db),
	}
}

func (s *Server) Start() {
	r := chi.NewRouter()

	r.Get("/", s.gallery)
	r.Get("/e", s.editor)
	r.Post("/e", s.save)
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

	gallery.Effects, err = s.db.Effects(page, perPage)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	log.With(log.Fields{
		"duration": time.Since(start),
	}).Infof("database queried")

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

	name := filepath.Join(imagesDir, fmt.Sprintf("%v.png", id))
	f, err := os.Open(name)
	if err != nil {
		log.Errorf(err, "cannot load image %v", name)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "image/png")
	_, err = io.Copy(w, f)
	if err != nil {
		log.Errorf(err, "cannot write image %v", name)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func (s *Server) css(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

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

	_, err = io.Copy(w, f)
	if err != nil {
		log.Errorf(err, "cannot write asset %v", path)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func (s *Server) js(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

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

type saveCode struct {
	CodeID        string `json:"code_id,omiempty"`
	Code          string `json:"code"`
	Image         string `json:"image"`
	User          string `json:"user"`
	Parent        string `json:"parent,omiempty"`
	ParentVersion string `json:"parent_version,omiempty"`
}

func (s *Server) save(w http.ResponseWriter, r *http.Request) {
	fmt.Println("saving effect %P", r)

	data := &saveCode{}
	buffer, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf(err, "cannot read body")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err = json.Unmarshal(buffer, data)
	if err != nil {
		log.Errorf(err, "cannot unmarshal json")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	parent, err := strconv.Atoi(data.Parent)
	if err != nil {
		parent = 0
	}

	parentVersion, err := strconv.Atoi(data.ParentVersion)
	if err != nil {
		parentVersion = 0
	}

	var effect *Effect
	var id int
	if data.CodeID != "" {
		split := strings.Split(data.CodeID, ".")
		id, err := strconv.Atoi(split[0])
		if err != nil {
			log.Errorf(err, "malformed code idetifier %s", data.CodeID)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		effect, err = s.db.Effect(id)
		if err != nil && !gorm.IsRecordNotFoundError(err) {
			log.Errorf(err, "could not retrieve code %v", id)
			http.Error(w, http.StatusText(500), 500)
			return
		}
	}

	if effect != nil {
		// check if the owner is saving
		if effect.User == data.User {
			err = s.db.UpdateTime(effect)
			if err != nil {
				log.Errorf(err, "could not update code %v", id)
				http.Error(w, http.StatusText(500), 500)
				return
			}
		} else {
			parent = id
			parentVersion = effect.LastVersion()
			effect = nil
		}
	}

	// create new record for new effects
	if effect == nil {
		effect, err = s.db.NewEffect(parent, parentVersion, data.User)
		if err != nil {
			log.Errorf(err, "could not create code %v", id)
			http.Error(w, http.StatusText(500), 500)
			return
		}
	}

	id = int(effect.ID)

	log.Debugf("saved effect %v", effect.ID)

	// TODO: add version

	version := &Version{
		EffectID: effect.ID,
		Number:   effect.NextVersion(),
		Created:  time.Now(),
		Code:     data.Code,
	}
	err = s.db.Create(version).Error
	if err != nil {
		log.Errorf(err, "could not create version %v", id)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	imageName := fmt.Sprintf("%v.png", id)
	imagePath := filepath.Join("images", imageName)
	f, err := os.Create(imagePath)
	if err != nil {
		log.Errorf(err, "could not create image %v", id)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	base := strings.Split(data.Image, ",")
	if len(base) == 0 {
		log.Errorf(err, "invalid image %v", id)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(base[len(base)-1])
	if err != nil {
		log.Errorf(err, "invalid image %v", id)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	_, err = f.Write(decoded)
	if err != nil {
		log.Errorf(err, "could not save image %v", id)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	fmt.Fprintf(w, "%v.%v", effect.ID, version.Number)
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

	effect, err := s.db.Effect(effectID)
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

	_, err = w.Write(m)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
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
