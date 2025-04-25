package mdbx

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestErrno_Error(t *testing.T) {
	operr := &OpError{fmt.Errorf("testmsg"), "testop"}
	msg := operr.Error()
	if msg != "testop: testmsg" {
		t.Errorf("message: %q", msg)
	}
}

func BenchmarkErrno_Error(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, errno := range []error{
			syscall.EINVAL,
			NotFound,
			MapFull,
		} {
			operr := &OpError{errno, "mdb_testop"}
			msg := operr.Error()
			if msg == "" {
				b.Fatal("empty message")
			}
		}

	}
}
func TestErrno(t *testing.T) {
	zeroerr := operrno("testop", 0)
	if zeroerr != nil {
		t.Errorf("errno(0) != nil: %#v", zeroerr)
	}
	syserr := _operrno("testop", int(syscall.EINVAL))
	if !errors.Is(syserr, syscall.EINVAL) { // fails if error is Errno(syscall.EINVAL)
		t.Errorf("errno(syscall.EINVAL) != syscall.EINVAL: %#v", syserr)
	}
	mdberr := _operrno("testop", int(KeyExist))
	if !errors.Is(mdberr, KeyExist) {
		t.Errorf("errno(ErrKeyExist) != ErrKeyExist: %#v", syserr)
	}
}

func TestIsErrno(t *testing.T) {
	err := NotFound
	if !IsErrno(err, err) {
		t.Errorf("expected match: %v", err)
	}

	operr := &OpError{
		Op:    "testop",
		Errno: err,
	}
	if !IsErrno(operr, err) {
		t.Errorf("expected match: %v", operr)
	}
}
