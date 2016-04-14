// +build amd64

package uuid

import "unsafe"

// Equal returns true if u1 and u2 equals, otherwise returns false.
func Equal(u1 UUID, u2 UUID) bool {
	return *(*uint64)(unsafe.Pointer(&u1[0])) == *(*uint64)(unsafe.Pointer(&u2[0])) &&
		*(*uint64)(unsafe.Pointer(&u1[8])) == *(*uint64)(unsafe.Pointer(&u2[8]))
}
