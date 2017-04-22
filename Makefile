.PHONY: build
build: github


.PHONY: github
github:
	go run util/fetch_github.go > data/github.json
