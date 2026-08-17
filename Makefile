
.PHONY: deps all test race bin cp

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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v2.12.2/install.sh | sh -s -- -b ./build/bin v2.12.2

clean:
	cd libmdbx && make clean

tools: clean
	cd libmdbx && MDBX_BUILD_TIMESTAMP=unknown CFLAGS="${CFLAGS} -Wno-unknown-warning-option -Wno-enum-int-mismatch -Wno-strict-prototypes -Wno-unused-but-set-variable" make tools

# Re-vendor libmdbx from a local upstream clone. Exports the tracked files of
# LIBMDBX_REF -- build artifacts left in the clone's working tree are never
# copied -- into a staging dir, drops upstream CI configs that cannot run from a
# vendored subdirectory, restores our keep.go, and swaps the result in. Stale
# files removed upstream disappear, which a plain copy would leave behind.
#
#   make cp                              # ../libmdbx at HEAD
#   make cp LIBMDBX_REF=v0.14.3          # a tag
#   make cp LIBMDBX_SRC=/path/to/libmdbx # a clone elsewhere
LIBMDBX_SRC ?= ../libmdbx
LIBMDBX_REF ?= HEAD

cp:
	@git -C $(LIBMDBX_SRC) rev-parse --verify --quiet $(LIBMDBX_REF)^{commit} >/dev/null || \
		{ echo "make cp: $(LIBMDBX_SRC) is not a git clone, or $(LIBMDBX_REF) is unknown there"; exit 1; }
	@echo "vendoring `git -C $(LIBMDBX_SRC) describe --tags --always $(LIBMDBX_REF)` from $(LIBMDBX_SRC)"
	rm -rf ./libmdbx.staging && mkdir ./libmdbx.staging
	git -C $(LIBMDBX_SRC) archive --format=tar $(LIBMDBX_REF) | tar -x -C ./libmdbx.staging
	rm -rf ./libmdbx.staging/.github ./libmdbx.staging/.sourcecraft
	cp ./libmdbx/keep.go ./libmdbx.staging/keep.go
	rm -rf ./libmdbx && mv ./libmdbx.staging ./libmdbx
