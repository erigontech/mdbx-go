package mdbx

/*
#include "mdbxgo.h"
*/
import "C"

func (env *Env) Path() (string, error) {
	var cpath *C.wchar_t
	ret := C.mdbx_env_get_pathW(env._env, &cpath)
	if ret != success {
		return "", operrno("mdbx_env_get_path", ret)
	}
	if cpath == nil {
		return "", errNotOpen
	}
	return C.GoString(cpath), nil
}
