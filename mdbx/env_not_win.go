//go:build !windows

package mdbx

/*
#include "mdbxgo.h"
*/
import "C"

//TODO: fix me please

// Path returns the path argument passed to Open.  Path returns a non-nil error
// if env.Open() was not previously called.
//
// See mdbx_env_get_path.
func (env *Env) Path() (string, error) {
	var cpath *C.char
	//nolint:dupSubExpr // false positive from Cgo (C.mdbx_env_get_path)
	ret := C.mdbx_env_get_path(env._env, &cpath)
	if ret != success {
		return "", operrno("mdbx_env_get_path", ret)
	}
	if cpath == nil {
		return "", errNotOpen
	}
	return C.GoString(cpath), nil
}
