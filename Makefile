
.PHONY: deps all test race bin

deps: lintci-deps
	go get -d ./...

all: deps

test:
	go test ./...

race:
	go test -race ./...

lint:
	./build/bin/golangci-lint run --new-from-rev=$(MASTER_COMMIT) ./...

lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.46.2

clean:
	cd mdbxdist && make clean

tools: clean
	cd mdbxdist && make tools

cp:
	cp mdbxdist/mdbx.h mdbx/
	cp mdbxdist/mdbx.c mdbx/
