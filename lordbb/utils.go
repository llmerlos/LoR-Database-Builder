package main

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//Do the main work utside the UI thread
func process(version string, locales []string) {
	nsets := 1
	for true { //Checks for the existance of sets given a version
		r, _ := http.Head(DDUrl + version + "/set" + strconv.Itoa(nsets) + "-en_us.zip")
		if r.StatusCode != 200 {
			nsets--
			break
		} else {
			nsets++
		}
	}

	//Create empty database (erase if exists)
	dbname := "lor-" + version + ".db"
	log.Println("Creating " + dbname)
	os.Remove(dbname)
	file, err := os.Create(dbname)
	if err != nil {
		log.Fatal(err.Error())
	}
	file.Close()

	//Open database and create tables
	log.Println("Opening " + dbname)
	sqlDb, _ := sql.Open("sqlite3", "./"+dbname)
	defer sqlDb.Close()
	log.Println("Creating tables in " + dbname)
	createTables(sqlDb, SQLTableCreation)

	//Make download directory if it doesn't exist
	log.Println("Making downloads directory")
	_, err = os.Stat("downloads")
	if os.IsNotExist(err) {
		errDir := os.MkdirAll("downloads", 0755)
		if errDir != nil {
			log.Fatal(err)
		}

	}

	//Make version directory if it doesn't exist
	log.Println("Making version directory")
	_, err = os.Stat("downloads" + version)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll("downloads/"+version, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}

	//Iterate for every locale in core and each set
	for _, locale := range locales {
		for set := 0; set <= nsets; set++ {
			var filepath, jsonpath string
			if set == 0 { //There's no set0 instead its the core bundle
				filepath = version + "/" + "core-" + locale
				jsonpath = "downloads/" + filepath + "/" + locale + "/data/" + "globals-" + locale + ".json"
			} else { //Instead of downloading the full bundle the lite suffices
				filepath = version + "/" + "set" + strconv.Itoa(set) + "-lite-" + locale
				jsonpath = "downloads/" + filepath + "/" + locale + "/data/" + "set" + strconv.Itoa(set) + "-" + locale + ".json"
				jsonpath = strings.ReplaceAll(jsonpath, "-lite", "") //ease of constructing the string
			}

			if locale == "en_us" { //For the en_us locale we download the full bundle to have the full art
				filepath = strings.ReplaceAll(filepath, "-lite", "")
			}
			log.Println("Downloading " + filepath + ".zip")
			downloadFile(filepath + ".zip")
			log.Println("Unziping " + filepath + ".zip")
			unzip("downloads/"+filepath+".zip", strings.ReplaceAll("downloads/"+filepath, "-lite", ""))
			log.Println("Parsing and inserting " + jsonpath)
			parseAndInsert(jsonpath, set, locale, sqlDb)
		}
	}
}

//download a file given the filepath to it in the DD
func downloadFile(filepath string) {

	//if the file already exists don't download it again
	_, err := os.Stat("downloads/" + filepath)
	if !os.IsNotExist(err) {
		return
	}

	//create empty file
	out, err := os.Create("downloads/" + filepath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer out.Close()

	//make the request
	resp, err := http.Get(DDUrl + filepath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()

	//transfer the files from the request to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
}

//unzip from a source file to a destination path
func unzip(src, dest string) error {
	//open file
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	//create destination folder with appropiate perms
	os.MkdirAll(dest, 0755)

	//extract the file there
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

//execute an array of create table queries on a given database
func createTables(db *sql.DB, dbQ []string) {
	for _, cquery := range dbQ {
		statement, err := db.Prepare(cquery)
		if err != nil {
			log.Fatal(err.Error())
		}
		statement.Exec()
	}
}

//given a JSON file, a set, a locale and a databse parse the info and insert it
func parseAndInsert(file string, set int, locale string, db *sql.DB) {
	jsonStream, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	dec := json.NewDecoder(jsonStream)
	global := strings.Contains(file, "globals")
	//the global/core file has a different structure so it has the be parsed differently
	if !global {
		_, err := dec.Token() //takes out first token "["
		if err != nil {
			log.Fatal(err)
		}

		for dec.More() { //while it has more items keep parsing
			var c Card
			err := dec.Decode(&c)
			if err != nil {
				log.Fatal(err)
			}
			c.Set = "set" + strconv.Itoa(set) //Set has the be inserted manually because the json doesn't provide the info
			s, _ := json.MarshalIndent(c, "", "\t")
			log.Println(string(s))
			insertCard(c, locale, db) // insert on the database
		}

		_, err = dec.Token()
		if err != nil {
			log.Fatal(err)
		}
	} else { // does the same as above but doesnt remove the token
		for dec.More() {
			var g Global
			err := dec.Decode(&g)
			if err != nil {
				log.Fatal(err)
			}
			s, _ := json.MarshalIndent(g, "", "\t")
			log.Println(string(s))
			insertGeneric(g, locale, db)
		}
	}
}

//set of queries to perform for each card
func insertCard(c Card, locale string, db *sql.DB) {
	if locale == "en_us" { //the base stats are only inserted with "en_us" locale
		statement, err := db.Prepare(InsertCardSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		//cards table
		_, err = statement.Exec(c.CardCode, nullS(c.Set), nullS(c.RegionRef), c.Attack, c.Health, c.Cost, nullS(c.ArtistName), nullS(c.SpellSpeedRef), nullS(c.RarityRef), nullS(c.Subtype), nullS(c.Supertype), nullS(c.Type), c.Collectible)
		if err != nil {
			log.Fatalln(err.Error())
		}
		//keywords table
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
		//associations table
		for _, as := range c.AssociatedCardRefs {
			statement, err = db.Prepare(InsertAssociationsSQLQ)
			if err != nil {
				log.Fatalln(err.Error())
			}
			_, err = statement.Exec(c.CardCode, as)
			if err != nil {
				log.Fatalln(err.Error())
			}
			_, notok := AssociationsExceptions[c.CardCode] //Some cards have multiple times the same card associated triggering an error of uniqueness
			if notok {
				break
			}
		}

	}
	//insert the translations for the locale
	statement, err := db.Prepare(InsertLocaleCardsSQLQ)
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = statement.Exec(c.CardCode, locale, c.Name, nullS(c.Description), nullS(c.DescriptionRaw), nullS(c.LevelupDescription), nullS(c.LevelupDescriptionRaw))
	if err != nil {
		log.Fatalln(err.Error())
	}

}

//set of queries to perform for each core
func insertGeneric(c Global, locale string, db *sql.DB) {
	//localeGeneric table
	for _, kw := range c.Keywords {
		statement, err := db.Prepare(InsertLocaleGenericSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec("keywords", locale, kw.NameRef, kw.Name, nullS(kw.Description))
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	for _, vt := range c.VocabTerms {
		statement, err := db.Prepare(InsertLocaleGenericSQLQ)
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec("vocabTerms", locale, vt.NameRef, vt.Name, nullS(vt.Description))
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
		_, err = statement.Exec("vocabTerms", locale, re.NameRef, re.Name, nullS(re.Description))
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

}

//to insert a null string instead of an empty string when needed
func nullS(s string) sql.NullString {
	if len(s) == 0 || s == " " {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
