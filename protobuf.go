// +build protobuf

package uuid

func (u UUID) Size() int {
	return len(u)
}

func (u UUID) MarshalTo(data []byte) (int, error) {
	copy(data, u[:])
	return 16, nil
}

type randy interface {
	Intn(n int) int
}

func NewPopulatedUUID(r randy) *UUID {
	var u UUID
	for i := 0; i < len(u); i++ {
		u[i] = byte(r.Intn(255))
	}
	return &u
}
