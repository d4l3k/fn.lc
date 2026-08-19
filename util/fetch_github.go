package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/gernest/front"
	"github.com/tomnomnom/linkheader"
)

const userReposURL = "https://api.github.com/users/d4l3k/repos?type=all&per_page=100"

type project struct {
	path  string
	front map[string]interface{}
	body  string
}

// get fetches url as JSON into out and returns the response headers. Set
// GITHUB_TOKEN to raise the 60 requests/hour unauthenticated rate limit.
func get(url string, out interface{}) (http.Header, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GET %s: %s: %s", url, resp.Status, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, err
	}
	return resp.Header, nil
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.SetOutput(os.Stderr)

	final := map[string]map[string]interface{}{}

	for nextURL := userReposURL; nextURL != ""; {
		log.Printf("Fetching: %s", nextURL)
		var repos []map[string]interface{}
		header, err := get(nextURL, &repos)
		if err != nil {
			log.Fatal(err)
		}
		for _, r := range repos {
			final[strings.ToLower(r["full_name"].(string))] = r
		}

		nextURL = ""
		for _, link := range linkheader.Parse(header.Get("Link")) {
			if link.Rel == "next" {
				nextURL = link.URL
				break
			}
		}
	}

	files, err := filepath.Glob("content/project/*.md")
	if err != nil {
		log.Fatal(err)
	}
	m := front.NewMatter()
	m.Handle("---", front.YAMLHandler)

	var projects []project
	for _, f := range files {
		fo, err := os.Open(f)
		if err != nil {
			log.Fatal(err)
		}
		matter, body, err := m.Parse(fo)
		fo.Close()
		if err != nil {
			log.Fatalf("%s: %v", f, err)
		}
		projects = append(projects, project{path: f, front: matter, body: body})
	}

	// Org owned repos (pytorch/*, meta-pytorch/*, nwplus/*, ...) aren't all
	// returned by the user repos endpoint. Without this they'd be dropped from
	// github.json and their project pages would lose their description.
	for _, p := range projects {
		name, ok := p.front["github"].(string)
		if !ok {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := final[key]; ok {
			continue
		}
		url := "https://api.github.com/repos/" + name
		log.Printf("Fetching: %s", url)
		var repo map[string]interface{}
		if _, err := get(url, &repo); err != nil {
			log.Fatal(err)
		}
		final[key] = repo
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(final); err != nil {
		log.Fatal(err)
	}

	for _, p := range projects {
		if name, ok := p.front["github"].(string); ok {
			details, ok := final[strings.ToLower(name)]
			if !ok {
				log.Printf("can't find github repo %q", name)
				continue
			}
			p.front["date"] = details["pushed_at"].(string)
			p.front["stars"] = details["stargazers_count"].(float64)
			p.front["weight"] = details["stargazers_count"].(float64) + 1
		}

		var buf bytes.Buffer
		buf.WriteString("---\n")
		yamlBytes, err := yaml.Marshal(p.front)
		if err != nil {
			log.Fatal(err)
		}
		buf.Write(yamlBytes)
		buf.WriteString("---\n")
		buf.WriteString(p.body)

		if err := os.WriteFile(p.path, buf.Bytes(), 0644); err != nil {
			log.Fatal(err)
		}
	}
}
