
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
	cp mdbx/dist/*.h mdbx/
	cp mdbx/dist/*.c mdbx/
	cp C:\WINDOWS\SYSTEM32\ntdll.dll .
	CGO_LDFLAGS_ALLOW=".*"	go test ./mdbx
