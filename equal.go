// +build !unsafe

package uuid

// Equal returns true if u1 and u2 equals, otherwise returns false.
func Equal(u1 UUID, u2 UUID) bool {
	return u1 == u2
}
