package main

/*
#include "../../../libmdbx/mdbx.h"
#include <stdatomic.h>

static atomic_uint calls;

static void count_log(MDBX_log_level_t level, const char *function, int line,
                      const char *fmt, va_list args) {
    (void)level;
    (void)function;
    (void)line;
    (void)fmt;
    (void)args;
    atomic_fetch_add(&calls, 1);
}

typedef struct { MDBX_debug_func fn; } test_logger;
static test_logger callback(void) {
    test_logger logger = {count_log};
    return logger;
}
static unsigned log_count(void) { return atomic_load(&calls); }
*/
import "C"

import (
	"fmt"
	"os"

	"github.com/erigontech/mdbx-go/mdbx"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func openEnvironment() {
	path, err := os.MkdirTemp("", "mdbx-debug-logger-")
	check(err)
	defer os.RemoveAll(path)
	env, err := mdbx.NewEnv(mdbx.Default)
	check(err)
	check(env.Open(path, 0, 0600))
	check(env.Close())
}

func main() {
	env, err := mdbx.NewEnv(mdbx.Default)
	check(err)
	defer env.Close()

	// The unnamed pointer conversion bridges cgo's package-local typedefs.
	// The callback itself retains its C function-pointer type in the helper.
	logger := (*[0]byte)(C.callback().fn)
	check(env.SetDebug(mdbx.LogLvlVerbose, mdbx.DbgDoNotChange, logger))
	openEnvironment()
	first := C.log_count()
	if first == 0 {
		panic("custom logger was not called")
	}

	check(env.SetDebug(mdbx.LogLvlDoNotChange, mdbx.DbgDoNotChange, mdbx.LoggerDoNotChange))
	openEnvironment()
	second := C.log_count()
	if second <= first {
		panic("LoggerDoNotChange replaced the callback")
	}

	check(env.SetDebug(mdbx.LogLvlDoNotChange, mdbx.DbgDoNotChange, nil))
	openEnvironment()
	if C.log_count() != second {
		panic("nil did not remove the callback")
	}

	fmt.Println("custom callback, keep sentinel and nil logger verified")
}
