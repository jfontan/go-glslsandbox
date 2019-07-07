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
	db, err := gorm.Open("sqlite3", "test.db")
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

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := []byte(scanner.Text())

		effect, err := glsl.LoadEffect(line)
		if err != nil {
			return err
		}

		db.Save(effect)

		errs := db.GetErrors()
		if len(errs) != 0 {
			return errs[0]
		}
	}

	return nil
}
