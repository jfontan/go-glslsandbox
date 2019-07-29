package glsl

import (
	"encoding/json"
	"time"
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
