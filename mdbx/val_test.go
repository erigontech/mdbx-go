package mdbx

import (
	"bytes"
	"reflect"
	"testing"
)

func TestMultiVal(t *testing.T) {
	data := []byte("abcdef")
	m := WrapMulti(data, 2)
	vals := m.Vals()
	if !reflect.DeepEqual(vals, [][]byte{{'a', 'b'}, {'c', 'd'}, {'e', 'f'}}) {
		t.Errorf("unexpected vals: %q", vals)
	}
	size := m.Size()
	if size != 6 {
		t.Errorf("unexpected size: %v (!= %v)", size, 6)
	}
	length := m.Len()
	if length != 3 {
		t.Errorf("unexpected length: %v (!= %v)", length, 3)
	}
	stride := m.Stride()
	if stride != 2 {
		t.Errorf("unexpected stride: %v (!= %v)", stride, 2)
	}
	page := m.Page()
	if !bytes.Equal(page, data) {
		t.Errorf("unexpected page: %v (!= %v)", page, data)
	}
}

func TestMultiVal_panic(t *testing.T) {
	var p bool
	defer func() {
		if e := recover(); e != nil {
			p = true
		}
		if !p {
			t.Errorf("expected a panic")
		}
	}()
	WrapMulti([]byte("123"), 2)
}

func TestVal(t *testing.T) {
	orig := []byte("hey hey")
	val := wrapVal(orig)

	p := castToBytes(val)
	if !bytes.Equal(p, orig) {
		t.Errorf("castToBytes() not the same as original data: %q", p)
	}
	if &p[0] != &orig[0] {
		t.Errorf("castToBytes() is not the same slice as original")
	}
}
