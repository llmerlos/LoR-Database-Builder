package main

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	if len(os.Args) != 2 {
		panic("Correct usage: <version>")
	}
	nsets := 2
	version := os.Args[1]
	dbname := "lor-" + version + ".db"

	os.Remove(dbname)

	log.Println("Creating" + dbname + "...")
	file, err := os.Create(dbname)
	if err != nil {
		log.Fatal(err.Error())
	}
	file.Close()
	log.Println(dbname + " created")

	sqlDb, _ := sql.Open("sqlite3", "./"+dbname)
	defer sqlDb.Close()
	CreateTables(sqlDb, SQLTableCreation)

	_, err = os.Stat("downloads")
	if os.IsNotExist(err) {
		errDir := os.MkdirAll("downloads", 0755)
		if errDir != nil {
			log.Fatal(err)
		}

	}
	_, err = os.Stat("downloads" + version)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll("downloads/"+version, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

	}

	for _, locale := range Locales {
		for set := 0; set <= nsets; set++ {
			var filepath, jsonpath string
			if set == 0 {
				filepath = version + "/" + "core-" + locale
				jsonpath = "downloads/" + filepath + "/" + locale + "/data/" + "globals-" + locale + ".json"
			} else {
				filepath = version + "/" + "set" + strconv.Itoa(set) + "-lite-" + locale
				jsonpath = "downloads/" + filepath + "/" + locale + "/data/" + "set" + strconv.Itoa(set) + "-" + locale + ".json"
			}

			if locale == Locales[0] {
				filepath = strings.ReplaceAll(filepath, "-lite", "")
			}
			jsonpath = strings.ReplaceAll(jsonpath, "-lite", "")
			DownloadFile(filepath + ".zip")
			Unzip("downloads/"+filepath+".zip", strings.ReplaceAll("downloads/"+filepath, "-lite", ""))
			ParseAndInsert(jsonpath, locale, sqlDb)
		}
	}
}

//DownloadFile downloads a file from the Data Dragon
func DownloadFile(filepath string) {
	_, err := os.Stat("downloads/" + filepath)
	if !os.IsNotExist(err) {
		return
	}

	out, err := os.Create("downloads/" + filepath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer out.Close()

	resp, err := http.Get(DDUrl + filepath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
}

//Unzip unzip a file
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

//CreateTables creates the tables in the DB
func CreateTables(db *sql.DB, dbQ []string) {
	for _, cquery := range dbQ {
		statement, err := db.Prepare(cquery)
		if err != nil {
			log.Fatal(err.Error())
		}
		statement.Exec()
	}
}

//ParseAndInsert does this thing
func ParseAndInsert(file string, locale string, db *sql.DB) {

	jsonStream, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}

	global := strings.Contains(file, "globals")

	dec := json.NewDecoder(jsonStream)

	if !global {
		_, err := dec.Token()
		if err != nil {
			log.Fatal(err)
		}

		for dec.More() {
			var c Card
			err := dec.Decode(&c)
			if err != nil {
				log.Fatal(err)
			}
			c.Set = file
			fmt.Println(c)
			insertCard(c, locale, db)
		}

		_, err = dec.Token()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		for dec.More() {
			var g Global
			err := dec.Decode(&g)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(g)
			insertGeneric(g, locale, db)
		}
	}
}

//insertCard inserts a card to the db
func insertCard(c Card, locale string, db *sql.DB) {
	if locale == "en_us" {
		statement, err := db.Prepare(InsertCardSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec(c.CardCode, NullS(c.Set), NullS(c.RegionRef), c.Attack, c.Health, c.Cost, NullS(c.ArtistName), NullS(c.SpellSpeedRef), NullS(c.RarityRef), NullS(c.Subtype), NullS(c.Supertype), NullS(c.Type), c.Collectible)
		if err != nil {
			log.Fatalln(err.Error())
		}

		for _, kw := range c.KeywordRefs {
			statement, err = db.Prepare(InsertKeywordsSQLQ)
			if err != nil {
				log.Fatalln(err.Error())
			}
			_, err = statement.Exec(c.CardCode, kw)
			if err != nil {
				log.Fatalln(err.Error())
			}
		}

		for _, as := range c.AssociatedCardRefs {
			statement, err = db.Prepare(InsertAssociationsSQLQ)
			if err != nil {
				log.Fatalln(err.Error())
			}
			_, err = statement.Exec(c.CardCode, as)
			if err != nil {
				log.Fatalln(err.Error())
			}
			_, notok := AssociationsExceptions[c.CardCode]
			if notok {
				break
			}
		}

	}

	statement, err := db.Prepare(InsertLocaleCardsSQLQ)
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = statement.Exec(c.CardCode, locale, c.Name, NullS(c.Description), NullS(c.DescriptionRaw), NullS(c.LevelupDescription), NullS(c.LevelupDescriptionRaw))
	if err != nil {
		log.Fatalln(err.Error())
	}

}

func insertGeneric(c Global, locale string, db *sql.DB) {
	for _, kw := range c.Keywords {
		statement, err := db.Prepare(InsertLocaleGenericSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec("keywords", locale, kw.NameRef, kw.Name, NullS(kw.Description))
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	for _, vt := range c.VocabTerms {
		statement, err := db.Prepare(InsertLocaleGenericSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec("vocabTerms", locale, vt.NameRef, vt.Name, NullS(vt.Description))
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	for _, ss := range c.SpellSpeeds {
		statement, err := db.Prepare(InsertLocaleGenericSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec("spellSpeeds", locale, ss.NameRef, ss.Name, sql.NullString{})
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	for _, ra := range c.Rarities {
		statement, err := db.Prepare(InsertLocaleGenericSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec("spellSpeeds", locale, ra.NameRef, ra.Name, sql.NullString{})
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	for _, re := range c.Regions {
		statement, err := db.Prepare(InsertLocaleGenericSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec("vocabTerms", locale, re.NameRef, re.Name, NullS(re.Description))
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

}

//NullS inserts null instead of empty string
func NullS(s string) sql.NullString {
	if len(s) == 0 || s == " " {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
