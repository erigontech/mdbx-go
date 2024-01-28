
.PHONY: deps all test race bin

MASTER_COMMIT=`git rev-parse --short origin/master`

deps: lintci-deps
	go get -d ./...

all: deps

test:
	go test ./mdbx ./exp/mdbxpool

race:
	go test -race ./mdbx ./exp/mdbxpool

lint:
	./build/bin/golangci-lint run --new-from-rev=$(MASTER_COMMIT) ./...

lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.55.2

clean:
	cd mdbxdist && make clean

tools: clean
	cd mdbxdist && MDBX_BUILD_TIMESTAMP=unknown CFLAGS="${CFLAGS} -Wno-unknown-warning-option -Wno-enum-int-mismatch -Wno-strict-prototypes -Wno-unused-but-set-variable" make tools

cp:
	cp mdbxdist/mdbx.h mdbx/
	cp mdbxdist/mdbx.c mdbx/
