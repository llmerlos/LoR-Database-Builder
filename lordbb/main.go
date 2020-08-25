// Demo code for the Form primitive.
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	var locales = []string{"en_us"}
	versionPtr := flag.String("v", "latest", "version: X.X.X or latest")
	flag.Parse()

	version := *versionPtr
	tail := flag.Args()

	if version != "latest" {
		r, _ := http.Head(DDUrl + version + "/core-en_us.zip")
		if r.StatusCode != 200 {
			log.Fatal("Version " + version + " not found")
		}
	}

	if len(tail) > 0 {
		for _, locale := range tail {
			if locale != "en_us" {
				r, _ := http.Head(DDUrl + version + "/core-" + locale + ".zip")
				if r.StatusCode == 200 {
					locales = append(locales, locale)
				} else {
					log.Println("Locale: " + locale + " not found")
				}
			}
		}
	} else {
		locales = Locales
	}

	process(version, locales)

}