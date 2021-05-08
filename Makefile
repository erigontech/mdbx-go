
.PHONY: deps all test race bin

BRANCH=`git rev-parse --abbrev-ref HEAD`
COMMIT=`git rev-parse --short HEAD`
MASTER_COMMIT=`git rev-parse --short origin/master`
GOLDFLAGS="-X main.branch $(BRANCH) -X main.commit $(COMMIT)"

deps: lintci-deps
	go get -d ./...

all: deps check race bin

test: mdbx-build
	go test ./mdbx ./exp/mdbxpool

race: mdbx-build
	go test -race ./mdbx ./exp/mdbxpool

lint: mdbx-build
	./build/bin/golangci-lint run --new-from-rev=$(MASTER_COMMIT) ./...

lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b ./build/bin v1.31.0

check:
	which goimports > /dev/null
	find . -name '*.go' | xargs goimports -d | tee /dev/stderr | wc -l | xargs test 0 -eq
	which golint > /dev/null
	golint ./... | tee /dev/stderr | wc -l | xargs test 0 -eq

clean:
	cd mdbx/dist/ && make clean

mdbx-build:
	echo "Building mdbx"
	cd mdbx/dist/ && make clean && make config.h && CFLAGS_EXTRA="-Wno-deprecated-declarations" make mdbx-static.o

win:
	go env
	ls
	echo ${CC}
	echo ${CXX_FOR_TARGET}
	echo ${CC_FOR_TARGET}
	cd libmdbx && ls && cmake  -DCMAKE_SYSTEM_NAME=Windows -DCMAKE_FIND_ROOT_PATH_MODE_PROGRAM=NEVER    -DCMAKE_FIND_ROOT_PATH_MODE_LIBRARY=ONLY  -DCMAKE_FIND_ROOT_PATH_MODE_INCLUDE=ONLY  -DCMAKE_C_COMPILER=x86_64-w64-mingw32-gcc  -DCMAKE_CXX_COMPILER=x86_64-w64-mingw32-g++  -DCMAKE_RC_COMPILER=x86_64-w64-mingw32-windres -DCXX_FOR_TARGET=x86_64-w64-mingw32-g++ -DCC_FOR_TARGET=x86_64-w64-mingw32-gcc -DCC=x86_64-w64-mingw32-gcc -DCMAKE_C_COMPILER=x86_64-w64-mingw32-gcc -DCMAKE_CXX_COMPILER_TARGET=Windows -DCMAKE_SYSTEM_NAME=Windows -DCMAKE_C_COMPILER_WORKS=1 .
	cd libmdbx && CXX_FOR_TARGET=x86_64-w64-mingw32-g++  CC_FOR_TARGET=x86_64-w64-mingw32-gcc CC=x86_64-w64-mingw32-gcc cmake --build .
	cp C:\WINDOWS\SYSTEM32\ntdll.dll mdbx/src/Debug
	cp libmdbx/mdbx.h mdbx/
	ls libmdbx/Debug
	CGO_CFLAGS='-DMDBX_BUILD_FLAGS_CONFIG="config.h"' go test ./mdbx
	#CGO_LDFLAGS_ALLOW=".*"	CGO_CFLAGS='-DMDBX_BUILD_FLAGS_CONFIG="config.h"' go test ./mdbx
