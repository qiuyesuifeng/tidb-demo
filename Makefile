ARCH      := "`uname -s`"
LINUX     := "Linux"
MAC       := "Darwin"
PACKAGES  := $$(go list ./...| grep -vE 'vendor')
FILES     := $$(find . -name '*.go' -type f | grep -vE 'vendor')

.PHONY: build master minion counter

default: build

all: build

build: master minion counter

master:
	go build -o bin/tidemo-master cmd/demo-master/main.go

minion:
	go build -o bin/tidemo-minion cmd/demo-minion/main.go

counter:
	go build -o bin/tidemo-counter cmd/demo-counter/main.go

fmt:
	go fmt ./...
	@goimports -w $(FILES)

clean:
	go clean ./...

deps:
	go list -f '{{range .Deps}}{{printf "%s\n" .}}{{end}}{{range .TestImports}}{{printf "%s\n" .}}{{end}}' ./... | \
		sort | uniq | grep -E '[^/]+\.[^/]+/' | \
		awk 'BEGIN{print "#!/bin/bash"}{ printf("go get -u %s\n", $$1) }' > deps.sh
	chmod +x deps.sh
