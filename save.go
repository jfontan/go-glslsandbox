package glsl

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"gopkg.in/src-d/go-log.v1"
)

type saveCode struct {
	CodeID        string `json:"code_id,omiempty"`
	Code          string `json:"code"`
	Image         string `json:"image"`
	User          string `json:"user"`
	Parent        string `json:"parent,omiempty"`
	ParentVersion string `json:"parent_version,omiempty"`
}

func (s *Server) save(w http.ResponseWriter, r *http.Request) {
	data := saveCode{}
	buffer, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf(err, "cannot read body")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err = json.Unmarshal(buffer, &data)
	if err != nil {
		log.Errorf(err, "cannot unmarshal json")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	effect, err := createOrUpdateEffect(s.db, data)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	version := &Version{
		EffectID: effect.ID,
		Number:   effect.NextVersion(),
		Created:  time.Now(),
		Code:     data.Code,
	}
	err = s.db.Create(version).Error
	if err != nil {
		log.Errorf(err, "could not create version %v", effect.ID)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err = saveImage(effect.ID, data)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	log.Debugf("saved effect %v", effect.ID)

	fmt.Fprintf(w, "%v.%v", effect.ID, version.Number)
}

func createOrUpdateEffect(db *Database, data saveCode) (*Effect, error) {
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
			return nil, err
		}

		effect, err = db.Effect(id)
		if err != nil && !gorm.IsRecordNotFoundError(err) {
			log.Errorf(err, "could not retrieve code %v", id)
			return nil, err
		}
	}

	// no code or effect not found
	if effect != nil {
		// check if the owner is saving
		if effect.User == data.User {
			err = db.UpdateTime(effect)
			if err != nil {
				log.Errorf(err, "could not update code %v", id)
				return nil, err
			}
		} else {
			parent = id
			parentVersion = effect.LastVersion()
			effect = nil
		}
	}

	// create new record for new effects
	if effect == nil {
		effect, err = db.NewEffect(parent, parentVersion, data.User)
		if err != nil {
			log.Errorf(err, "could not create code %v", id)
			return nil, err
		}
	}

	return effect, nil
}
func saveImage(id uint, data saveCode) error {
	imageName := fmt.Sprintf("%v.png", id)
	imagePath := filepath.Join("images", imageName)
	f, err := os.Create(imagePath)
	if err != nil {
		log.Errorf(err, "could not create image %v", id)
		return err
	}
	defer f.Close()

	base := strings.Split(data.Image, ",")
	if len(base) == 0 {
		log.Errorf(err, "invalid image %v", id)
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(base[len(base)-1])
	if err != nil {
		log.Errorf(err, "invalid image %v", id)
		return err
	}

	_, err = f.Write(decoded)
	if err != nil {
		log.Errorf(err, "could not save image %v", id)
		return err
	}

	return nil
}
