// +build protobuf

package uuid

import (
	"bytes"
	"fmt"
)

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
	b := make([]byte, 0, len(u)+2)
	b = append(b, '"')
	m, err := u.MarshalText()
	if err != nil {
		return nil, err
	}
	b = append(b, m...)
	return append(b, '"'), nil
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	if len(data) < 32 {
		return fmt.Errorf("uuid: UUID string too short: %q", data)
	}
	if data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("uuid: invalid string format")
	}
	return u.UnmarshalText(data[1 : len(data)-1])
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
