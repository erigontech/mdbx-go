
.PHONY: deps all test race bin

MASTER_COMMIT=`git rev-parse --short origin/master`

deps: lint-deps
	go get ./...

all: deps

test:
	go test ./mdbx ./exp/mdbxpool

race:
	go test -race ./mdbx ./exp/mdbxpool

lint:
	./build/bin/golangci-lint run ./...

lint-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v2.1.5

clean:
	cd libmdbx && make clean

tools: clean
	cd libmdbx && MDBX_BUILD_TIMESTAMP=unknown CFLAGS="${CFLAGS} -Wno-unknown-warning-option -Wno-enum-int-mismatch -Wno-strict-prototypes -Wno-unused-but-set-variable" make tools

cp:
	#cd ../libmdbx && make dist
	cp -R ./../libmdbx/* ./libmdbx/
