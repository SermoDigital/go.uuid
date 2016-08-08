// +build protobuf

package uuid

func (u UUID) Size() int {
	return len(u)
}

func (u UUID) MarshalTo(data []byte) (int, error) {
	copy(data, u[:])
	return 16, nil
}
