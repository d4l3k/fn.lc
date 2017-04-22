package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gernest/front"
	"github.com/tomnomnom/linkheader"
)

const baseURL = "https://api.github.com/users/d4l3k/repos?type=all"

func main() {
	final := map[string]map[string]interface{}{}

	log.SetOutput(os.Stderr)
	nextURL := baseURL
outer:
	for {
		log.Printf("Fetching: %s", nextURL)
		req, err := http.Get(nextURL)
		if err != nil {
			log.Fatal(err)
		}
		var repos []map[string]interface{}

		err = json.NewDecoder(req.Body).Decode(&repos)
		req.Body.Close()
		if err != nil {
			log.Fatal(err)
		}

		for _, r := range repos {
			fullName := strings.ToLower(r["full_name"].(string))
			final[fullName] = r
		}

		links := linkheader.Parse(req.Header.Get("Link"))
		for _, link := range links {
			if link.Rel == "next" {
				nextURL = link.URL
				continue outer
			}
		}
		break
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(final); err != nil {
		log.Fatal(err)
	}

	files, err := filepath.Glob("content/project/*.md")
	if err != nil {
		log.Fatal(err)
	}
	m := front.NewMatter()
	m.Handle("---", front.YAMLHandler)
	for _, f := range files {
		fo, err := os.Open(f)
		if err != nil {
			log.Fatal(err)
		}
		front, body, err := m.Parse(fo)
		if err != nil {
			log.Fatal(err)
		}
		github, ok := front["github"]
		if ok {
			details, ok := final[github.(string)]
			if !ok {
				log.Printf("can't find github repo %q", github)
			}
			front["date"] = details["updated_at"].(string)
			front["stars"] = details["stargazers_count"].(float64)
			log.Println("front", front)
		}
		_ = body
		fo.Close()
	}
}
