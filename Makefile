
.PHONY: deps all test race bin

MASTER_COMMIT=`git rev-parse --short origin/master`

deps: lintci-deps
	go get ./...

all: deps

test:
	go test ./mdbx ./exp/mdbxpool

race:
	go test -race ./mdbx ./exp/mdbxpool

lint:
	./build/bin/golangci-lint run ./...
#//--new-from-rev=$(MASTER_COMMIT) ./...

lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.62.2

clean:
	cd mdbxdist && make clean

tools: clean
	cd mdbxdist && MDBX_BUILD_TIMESTAMP=unknown CFLAGS="${CFLAGS} -Wno-unknown-warning-option -Wno-enum-int-mismatch -Wno-strict-prototypes -Wno-unused-but-set-variable" make tools

cp:
	#cd ../libmdbx && make dist
	cp -R ./../libmdbx/dist/* ./mdbxdist/
	cp mdbxdist/mdbx.h mdbx/
	cp mdbxdist/mdbx.c mdbx/
	#add 1 line to mdbx.h about build flags which we have in `mdbx.go`
	echo "$(echo '#define MDBX_BUILD_FLAGS "-std=gnu11 -fvisibility=hidden -ffast-math"'; cat mdbx.h)" > mdbx.h
