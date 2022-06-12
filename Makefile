.PHONY: build
build: github metadata
	hugo -D
	./util/publish.sh

.PHONY: metadata
metadata:
	bash -c "exiv2 rm **/*.{jpg,jpeg,png}"

.PHONY: github
github:
	go run util/fetch_github.go > data/github.json
