package main

import (
	"bufio"
	"os"

	glsl "github.com/jfontan/go-glslsandbox"
	"github.com/src-d/go-cli"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func init() {
	app.AddCommand(&importCommand{})
}

type importCommand struct {
	cli.Command `name:"import" short-description:"imports data from old mongodb dump"`

	Args struct {
		File string `positional-arg-name:"file"`
	} `positional-args:"true" required:"yes"`
}

func (i *importCommand) Execute(args []string) error {
	db, err := prepareDB()
	if err != nil {
		return err
	}
	defer db.Close()

	err = importEffects(db, i.Args.File)
	if err != nil {
		return err
	}

	return nil
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
