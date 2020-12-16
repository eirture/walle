BUILD_FILES = $(shell go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}}\
{{end}}' ./...)

.PHONY: build
bin/walle: $(BUILD_FILES)
	go build -o "$@" ./pkg/cmd/walle


.PHONY: clean
clean:
	@rm -rf ./bin


.PHONY: lint
lint:
	@golangci-lint run
