.PHONY: build
build: github metadata
	hugo -D
	./util/publish.sh

.PHONY: metadata
metadata:
	exiv2 rm **/*.{jpg,png}

.PHONY: github
github:
	go run util/fetch_github.go > data/github.json
