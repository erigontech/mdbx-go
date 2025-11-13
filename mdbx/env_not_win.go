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

// FD returns the open file descriptor (or Windows file handle) for the given
// environment.  An error is returned if the environment has not been
// successfully Opened (where C API just retruns an invalid handle).
//
// See mdbx_env_get_fd.
func (env *Env) FD() (uintptr, error) {
	// fdInvalid is the value -1 as a uintptr, which is used by MDBX in the
	// case that env has not been opened yet.  the strange construction is done
	// to avoid constant value overflow errors at compile time.
	const fdInvalid = ^uintptr(0)

	mf := new(C.mdbx_filehandle_t)
	ret := C.mdbx_env_get_fd(env._env, mf)
	err := operrno("mdbx_env_get_fd", ret)
	if err != nil {
		return 0, err
	}
	fd := uintptr(*mf)

	if fd == fdInvalid {
		return 0, errNotOpen
	}
	return fd, nil
}
