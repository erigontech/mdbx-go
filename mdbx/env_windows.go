package mdbx

/*
#include "mdbxgo.h"
*/
import "C"
import (
	"syscall"
	"unsafe"
)

// TODO: fix me please
func (env *Env) Path() (string, error) {
	var cpath *C.wchar_t
	ret := C.mdbx_env_get_pathW(env._env, &cpath)
	if ret != success {
		return "", operrno("mdbx_env_get_path", ret)
	}
	if cpath == nil {
		return "", errNotOpen
	}

	return syscall.UTF16PtrToString((*uint16)(unsafe.Pointer(cpath))), nil
}
