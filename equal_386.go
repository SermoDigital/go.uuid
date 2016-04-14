// +build 386

package uuid

import "unsafe"

// Equal returns true if u1 and u2 equals, otherwise returns false.
func Equal(u1 UUID, u2 UUID) bool {
	return *(*uint32)(unsafe.Pointer(&u1[0])) == *(*uint32)(unsafe.Pointer(&u2[0])) &&
		*(*uint32)(unsafe.Pointer(&u1[4])) == *(*uint32)(unsafe.Pointer(&u2[4])) &&
		*(*uint32)(unsafe.Pointer(&u1[8])) == *(*uint32)(unsafe.Pointer(&u2[8])) &&
		*(*uint32)(unsafe.Pointer(&u1[12])) == *(*uint32)(unsafe.Pointer(&u2[12]))
}
