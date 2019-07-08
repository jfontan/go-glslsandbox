package main

import (
	"os"

	glsl "github.com/jfontan/go-glslsandbox"

	"bufio"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func main() {
	if len(os.Args) != 2 {
		panic("wrong number of parameters")
	}

	db, err := prepareDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = importEffects(db, os.Args[1])
	if err != nil {
		panic(err)
	}
}

func prepareDB() (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", "effects.db")
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&glsl.Effect{})
	db.AutoMigrate(&glsl.Version{})

	errs := db.GetErrors()
	if len(errs) != 0 {
		return nil, errs[0]
	}

	return db, nil
}

func importEffects(db *gorm.DB, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	bufSize := 64 * 1024 * 1024
	buf := make([]byte, bufSize)

	db.Begin()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(buf, len(buf))
	for scanner.Scan() {
		line := []byte(scanner.Text())

		effect, err := glsl.LoadEffect(line)
		if err != nil {
			println("cannot unmarshal", err.Error())
			continue
		}

		if effect.ID%100 == 0 {
			println("id", effect.ID, effect.User, len(line))
		}

		db.Create(effect)

		errs := db.GetErrors()
		if len(errs) != 0 {
			println("error", effect.ID, err.Error())
			return errs[0]
		}
	}

	db.Commit()

	return nil
}
