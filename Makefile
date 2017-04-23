.PHONY: build
build: github
	hugo


.PHONY: github
github:
	go run util/fetch_github.go > data/github.json
