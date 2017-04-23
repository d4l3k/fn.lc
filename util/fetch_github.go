package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/gernest/front"
	"github.com/tomnomnom/linkheader"
)

const baseURL = "https://api.github.com/users/d4l3k/repos?type=all"

func main() {
	token := os.Getenv("GITHUB_TOKEN")

	log.SetFlags(log.Flags() | log.Lshortfile)

	final := map[string]map[string]interface{}{}

	log.SetOutput(os.Stderr)
	nextURL := baseURL
outer:
	for {
		if len(token) > 0 {
			nextURL += "&access_token=" + token
		}
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
		fo.Close()
		github, ok := front["github"]
		if ok {
			details, ok := final[github.(string)]
			if !ok {
				log.Printf("can't find github repo %q", github)
			}
			front["date"] = details["pushed_at"].(string)
			front["stars"] = details["stargazers_count"].(float64)
			front["weight"] = details["stargazers_count"].(float64) + 1
		}
		var buf bytes.Buffer
		buf.WriteString("---\n")
		yamlBytes, err := yaml.Marshal(front)
		if err != nil {
			log.Fatal(err)
		}
		buf.Write(yamlBytes)
		buf.WriteString("---\n")
		buf.WriteString(body)

		fo, err = os.OpenFile(f, os.O_WRONLY, 0755)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := buf.WriteTo(fo); err != nil {
			log.Fatal(err)
		}

		fo.Close()
	}
}
