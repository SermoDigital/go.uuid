// +build protobuf

package uuid

import "bytes"

func (u UUID) Size() int {
	return len(u)
}

func (u UUID) MarshalTo(data []byte) (int, error) {
	return copy(data, u[:]), nil
}

func (u *UUID) Unmarshal(data []byte) error {
	return u.UnmarshalBinary(data)
}

func (u UUID) Marshal() ([]byte, error) {
	return u.MarshalBinary()
}

func (u UUID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.MarshalText() + `"`)
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	return u.UnmarshalText(data)
}

func (u UUID) Compare(u2 UUID) int {
	return bytes.Compare(u[:], u2[:])
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
