//go:build !windows

package mdbx

/*
#include "mdbxgo.h"
*/
import "C"

// Path returns the path argument passed to Open.  Path returns a non-nil error
// if env.Open() was not previously called.
//
// See mdbx_env_get_path.
//
//nolint:gocritic // reason: false positive on dupSubExpr
func (env *Env) Path() (string, error) {
	var cpath *C.char
	ret := C.mdbx_env_get_path(env._env, &cpath)
	if ret != success {
		return "", operrno("mdbx_env_get_path", ret)
	}
	if cpath == nil {
		return "", errNotOpen
	}
	return C.GoString(cpath), nil
}
