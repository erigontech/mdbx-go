
.PHONY: deps all test race bin

BRANCH=`git rev-parse --abbrev-ref HEAD`
COMMIT=`git rev-parse --short HEAD`
MASTER_COMMIT=`git rev-parse --short origin/master`
GOLDFLAGS="-X main.branch $(BRANCH) -X main.commit $(COMMIT)"

deps: lintci-deps
	go get -d ./...

all: deps check

test:
	go test ./mdbx ./exp/mdbxpool

race:
	go test -race ./mdbx ./exp/mdbxpool

lint:
	./build/bin/golangci-lint run --new-from-rev=$(MASTER_COMMIT) ./...

lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.46.2

check:
	which goimports > /dev/null
	find . -name '*.go' | xargs goimports -d | tee /dev/stderr | wc -l | xargs test 0 -eq
	which golint > /dev/null
	golint ./... | tee /dev/stderr | wc -l | xargs test 0 -eq

clean:
	cd mdbx/dist/ && make clean
