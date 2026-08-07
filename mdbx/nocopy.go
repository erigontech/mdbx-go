package mdbx

// noCopy makes `go vet -copylocks` reject copies of the type embedding it.
// The C-handle types here own a raw libmdbx object, so a copy would give two
// values one underlying handle: closing either frees it while the other still
// points at it.
//
// Zero-sized, so it costs no space when placed first in a struct. Held as a
// named field rather than embedded, which would promote Lock/Unlock onto the
// enclosing type's method set.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
