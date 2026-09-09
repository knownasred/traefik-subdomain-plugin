.PHONY: lint test yaegi_test vendor clean

export GO111MODULE=on

default: lint test

lint:
	golangci-lint run

test:
	go test -v -cover ./...

yaegi_test:
	@set -eu; \
	module_path=$$(go list -m); \
	test_gopath=$$(mktemp -d); \
	trap 'rm -rf "$$test_gopath"' EXIT; \
	mkdir -p "$$test_gopath/src/$$(dirname "$$module_path")"; \
	ln -s "$$(pwd -P)" "$$test_gopath/src/$$module_path"; \
	GOPATH="$$test_gopath" yaegi test -v .

vendor:
	go mod vendor

clean:
	rm -rf ./vendor
