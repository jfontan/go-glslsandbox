package glsl

import (
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
	"gopkg.in/src-d/go-log.v1"
)

type Effect struct {
	ID            uint `json:"_id" gorm:"primary_key"`
	Created       time.Time
	Modified      time.Time `gorm:"index:modified"`
	Parent        *Effect
	ParentID      uint   `json:"parent" sql:"default:null"`
	ParentVersion int    `json:"parent_version"`
	User          string `json:"user,omitempty"`
	Versions      []Version
}

func (e *Effect) LastVersion() int {
	if len(e.Versions) == 0 {
		println("no versions")
		return 0
	}

	return len(e.Versions) - 1
}

func (e *Effect) NextVersion() int {
	return len(e.Versions)
}

type effectJSON struct {
	Effect

	Versions   []versionJSON
	CreatedAt  Timestamp `json:"created_at" gorm:"-"`
	ModifiedAt Timestamp `json:"modified_at" gorm:"-"`
}

type Version struct {
	ID       uint
	EffectID uint `gorm:"index:effect_id"`

	Number  int
	Created time.Time
	Code    string `gorm:"type:text"`
}

type versionJSON struct {
	Version
	CreatedAt Timestamp `json:"created_at" gorm:"-"`
}

type Timestamp struct {
	Date int64 `json:"$date"`
}

func (t Timestamp) Time() time.Time {
	seconds := t.Date / 1000
	milis := t.Date % 1000
	return time.Unix(seconds, milis*1000000)
}

func (e *effectJSON) convert() {
	e.Created = e.CreatedAt.Time()
	e.Modified = e.ModifiedAt.Time()

	for i, ver := range e.Versions {
		ver.Number = i
		ver.Created = ver.CreatedAt.Time()
		e.Effect.Versions = append(e.Effect.Versions, ver.Version)
	}
}

func LoadEffect(text []byte) (*Effect, error) {
	var effect effectJSON
	err := json.Unmarshal(text, &effect)
	if err != nil {
		return nil, err
	}

	effect.convert()

	return &effect.Effect, nil
}

type Database struct {
	*gorm.DB
}

func NewDatabase(db *gorm.DB) *Database {
	return &Database{DB: db}
}

func (d *Database) Effect(id int) (*Effect, error) {
	var effect Effect
	db := d.Preload("Versions").Find(&effect, id)

	err := db.Error
	if err != nil {
		log.Errorf(err, "cannot retrieve effect%v", id)
		return nil, err
	}

	return &effect, nil
}

func (d *Database) Effects(page, size int) ([]Effect, error) {
	var effects []Effect
	db := d.Order("modified desc").Limit(perPage).Offset(page * perPage).
		Preload("Versions").Find(&effects)
	err := db.Error
	if err != nil {
		log.Errorf(err, "cannot retrieve effects")
		return nil, err
	}

	return effects, nil
}

func (d *Database) UpdateTime(e *Effect) error {
	err := d.DB.Model(e).Update("modified_at", time.Now()).Error
	if err != nil {
		log.Errorf(err, "cannot update effect time")
		return err
	}

	return nil
}

func (d *Database) NewEffect(
	parent, version int,
	user string,
) (*Effect, error) {
	effect := &Effect{
		Created:       time.Now(),
		Modified:      time.Now(),
		ParentID:      uint(parent),
		ParentVersion: version,
		User:          user,
	}

	err := d.Create(effect).Error
	if err != nil {
		log.Errorf(err, "cannot create effect")
		return nil, err
	}

	return effect, nil
}
