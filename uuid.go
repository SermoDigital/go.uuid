// Copyright (C) 2013-2015 by Maxim Bublis <b@codemonkey.ru>
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package uuid provides implementation of Universally Unique Identifier (UUID).
// Supported versions are 1, 3, 4 and 5 (as specified in RFC 4122) and
// version 2 (as specified in DCE 1.1).
package uuid

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// UUID layout variants.
const (
	VariantNCS = iota
	VariantRFC4122
	VariantMicrosoft
	VariantFuture
)

// UUID DCE domains.
const (
	DomainPerson = iota
	DomainGroup
	DomainOrg
)

// Difference in 100-nanosecond intervals between
// UUID epoch (October 15, 1582) and Unix epoch (January 1, 1970).
const epochStart = 122192928000000000

// Used in string method conversion
const dash byte = '-'

// UUID v1/v2 storage.
var (
	storageMutex  sync.Mutex
	storageOnce   sync.Once
	epochFunc     = unixTimeFunc
	clockSequence uint16
	lastTime      uint64
	hardwareAddr  [6]byte
	posixUID      = uint32(os.Getuid())
	posixGID      = uint32(os.Getgid())
)

// String parse helpers.
var (
	urnPrefix  = []byte("urn:uuid:")
	byteGroups = [...]int{8, 4, 4, 4, 12}
)

func initClockSequence() {
	var buf [2]byte
	safeRandom(buf[:])
	clockSequence = binary.BigEndian.Uint16(buf[:])
}

func initHardwareAddr() {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if len(iface.HardwareAddr) >= 6 {
				copy(hardwareAddr[:], iface.HardwareAddr)
				return
			}
		}
	}

	// Initialize hardwareAddr randomly in case
	// of real network interfaces absence
	safeRandom(hardwareAddr[:])

	// Set multicast bit as recommended in RFC 4122
	hardwareAddr[0] |= 0x01
}

func initStorage() {
	initClockSequence()
	initHardwareAddr()
}

func safeRandom(dest []byte) {
	if _, err := rand.Read(dest); err != nil {
		panic(err)
	}
}

// Returns difference in 100-nanosecond intervals between
// UUID epoch (October 15, 1582) and current time.
// This is default epoch calculation function.
func unixTimeFunc() uint64 {
	return epochStart + uint64(time.Now().UnixNano()/100)
}

// UUID representation compliant with specification described in RFC 4122.
type UUID [16]byte

// NullUUID can be used with the standard sql package to represent a UUID value
// that can be NULL in the database
type NullUUID struct {
	UUID  UUID
	Valid bool
}

// The nil UUID is special form of UUID that is specified to have all 128 bits
// set to zero.
var Nil UUID

// Predefined namespace UUIDs.
var (
	NamespaceDNS, _  = FromString("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	NamespaceURL, _  = FromString("6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	NamespaceOID, _  = FromString("6ba7b812-9dad-11d1-80b4-00c04fd430c8")
	NamespaceX500, _ = FromString("6ba7b814-9dad-11d1-80b4-00c04fd430c8")
)

// And returns result of binary AND of two UUIDs.
func And(u1 UUID, u2 UUID) UUID {
	var u UUID
	for i := 0; i < 16; i++ {
		u[i] = u1[i] & u2[i]
	}
	return u
}

// Or returns result of binary OR of two UUIDs.
func Or(u1 UUID, u2 UUID) UUID {
	var u UUID
	for i := 0; i < 16; i++ {
		u[i] = u1[i] | u2[i]
	}
	return u
}

// Version returns algorithm version used to generate UUID.
func (u UUID) Version() uint {
	return uint(u[6] >> 4)
}

// Variant returns UUID layout variant.
func (u UUID) Variant() uint {
	if (u[8] & 0x80) == 0x00 {
		return VariantNCS
	}
	if (u[8]&0xc0)|0x80 == 0x80 {
		return VariantRFC4122
	}
	if (u[8]&0xe0)|0xc0 == 0xc0 {
		return VariantMicrosoft
	}
	return VariantFuture
}

// IsNil returns true if u is a nil UUID.
func (u UUID) IsNil() bool {
	return Equal(u, Nil)
}

// Equals returns true if Equal(u, u2).
func (u UUID) Equals(u2 UUID) bool {
	return Equal(u, u2)
}

// Equal returns true if u1 and u2 equals, otherwise false.
func Equal(u1, u2 UUID) bool {
	if runtime.GOOS == "amd64" {
		return *(*uint64)(unsafe.Pointer(&u1[0])) == *(*uint64)(unsafe.Pointer(&u2[0])) &&
			*(*uint64)(unsafe.Pointer(&u1[8])) == *(*uint64)(unsafe.Pointer(&u2[8]))
	}
	if runtime.GOOS == "386" {
		return *(*uint32)(unsafe.Pointer(&u1[0])) == *(*uint32)(unsafe.Pointer(&u2[0])) &&
			*(*uint32)(unsafe.Pointer(&u1[4])) == *(*uint32)(unsafe.Pointer(&u2[4])) &&
			*(*uint32)(unsafe.Pointer(&u1[8])) == *(*uint32)(unsafe.Pointer(&u2[8])) &&
			*(*uint32)(unsafe.Pointer(&u1[12])) == *(*uint32)(unsafe.Pointer(&u2[12]))
	}
	return u1 == u2
}

// Bytes returns the canonical representation of a UUID as byte slice:
// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (u UUID) Bytes() []byte {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = dash
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = dash
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = dash
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = dash
	hex.Encode(buf[24:], u[10:])
	return buf[:]
}

// String returns the canonical representation of UUID as a string:
// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (u UUID) String() string {
	return string(u.Bytes())
}

// SetVersion sets version bits.
func (u *UUID) SetVersion(v byte) {
	u[6] = (u[6] & 0x0f) | (v << 4)
}

// SetVariant sets variant bits as described in RFC 4122.
func (u *UUID) SetVariant() {
	u[8] = (u[8] & 0xbf) | 0x80
}

// MarshalText implements the encoding.TextMarshaler interface. The encoding is
// the same as returned by String.
func (u UUID) MarshalText() (text []byte, err error) {
	return u.Bytes(), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface. The following
// formats are supported:
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
// "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
// "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"
func (u *UUID) UnmarshalText(text []byte) (err error) {
	// Plain UUID.
	if len(text) < 32 {
		return fmt.Errorf("uuid: UUID string too short: %q", text)
	}
	// urn prefix.
	if len(text) > 45 {
		return fmt.Errorf("uuid: UUID string too long: %q", text)
	}

	t := text
	braced := false

	if bytes.Equal(t[:9], urnPrefix) {
		t = t[9:]
	} else if t[0] == '{' {
		braced = true
		t = t[1:]
	}

	b := u[:]
	for i, byteGroup := range byteGroups {
		if i > 0 {
			if t[0] != '-' {
				return fmt.Errorf("uuid: invalid string format")
			}
			t = t[1:]
		}

		if len(t) < byteGroup {
			return fmt.Errorf("uuid: UUID string too short: %s", text)
		}

		if i == 4 && len(t) > byteGroup &&
			((braced && t[byteGroup] != '}') || len(t[byteGroup:]) > 1 || !braced) {
			return fmt.Errorf("uuid: UUID string too long: %s", text)
		}

		_, err = hex.Decode(b[:byteGroup/2], t[:byteGroup])
		if err != nil {
			return err
		}

		t = t[byteGroup:]
		b = b[byteGroup/2:]
	}
	return nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (u UUID) MarshalBinary() (data []byte, err error) {
	return u[:], nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
// It will return error if the slice isn't 16 bytes long.
func (u *UUID) UnmarshalBinary(data []byte) (err error) {
	if len(data) != 16 {
		return fmt.Errorf("uuid: UUID must be exactly 16 bytes long, got %d bytes", len(data))
	}
	copy(u[:], data)
	return nil
}

// Value implements the driver.Valuer interface.
func (u UUID) Value() (driver.Value, error) {
	return u.Bytes(), nil
}

// Scan implements the sql.Scanner interface. A 16-byte slice is handled by
// UnmarshalBinary, while a longer byte slice or a string is handled by
// UnmarshalText.
func (u *UUID) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		if len(src) == 16 {
			return u.UnmarshalBinary(src)
		}
		return u.UnmarshalText(src)

	case string:
		return u.UnmarshalText([]byte(src))
	}
	return fmt.Errorf("uuid: cannot convert %T to UUID", src)
}

// Value implements the driver.Valuer interface.
func (u NullUUID) Value() (driver.Value, error) {
	if !u.Valid {
		return nil, nil
	}
	// Delegate to UUID Value function.
	return u.UUID.Value()
}

// Scan implements the sql.Scanner interface.
func (u *NullUUID) Scan(src interface{}) error {
	if src == nil {
		u.UUID, u.Valid = Nil, false
		return nil
	}
	// Delegate to UUID Scan function.
	u.Valid = true
	return u.UUID.Scan(src)
}

// MarshalText implements the encoding.TextMarshaler interface. The encoding is
// the same as returned by String.
func (u NullUUID) MarshalText() ([]byte, error) {
	return u.UUID.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface. The following
// formats are supported:
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
// "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
// "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"
func (u *NullUUID) UnmarshalText(data []byte) error {
	return u.UUID.UnmarshalText(data)
}

// FromBytes returns UUID converted from raw byte slice input.
// It will return error if the slice isn't 16 bytes long.
func FromBytes(input []byte) (u UUID, err error) {
	err = u.UnmarshalBinary(input)
	return u, err
}

// FromBytesOrNil returns UUID converted from raw byte slice input.
// Same behavior as FromBytes, but returns a Nil UUID on error.
func FromBytesOrNil(input []byte) (u UUID) {
	// FromBytes returns Nil on error.
	u, _ = FromBytes(input)
	return u
}

// FromString returns UUID parsed from string input.
// Input is expected in a form accepted by UnmarshalText.
func FromString(input string) (u UUID, err error) {
	err = u.UnmarshalText([]byte(input))
	return
}

// FromStringOrNil returns UUID parsed from string input.
// Same behavior as FromString, but returns a Nil UUID on error.
func FromStringOrNil(input string) UUID {
	uuid, err := FromString(input)
	if err != nil {
		return Nil
	}
	return uuid
}

// Returns UUID v1/v2 storage state.
// Returns epoch timestamp, clock sequence, and hardware address.
func getStorage() (now uint64, seq uint16, addr [6]byte) {
	storageOnce.Do(initStorage)

	storageMutex.Lock()

	now = epochFunc()
	// Clock changed backwards since last UUID generation.
	// Should increase clock sequence.
	if now <= lastTime {
		clockSequence++
	}
	lastTime = now

	seq = clockSequence
	addr = hardwareAddr

	storageMutex.Unlock()
	return now, seq, addr
}

// NewV1 returns UUID based on current timestamp and MAC address.
func NewV1() UUID {
	var u UUID

	timeNow, clockSeq, hardwareAddr := getStorage()

	binary.BigEndian.PutUint32(u[0:], uint32(timeNow))
	binary.BigEndian.PutUint16(u[4:], uint16(timeNow>>32))
	binary.BigEndian.PutUint16(u[6:], uint16(timeNow>>48))
	binary.BigEndian.PutUint16(u[8:], clockSeq)

	copy(u[10:], hardwareAddr[:])

	u.SetVersion(1)
	u.SetVariant()

	return u
}

// NewV2 returns DCE Security UUID based on POSIX UID/GID.
func NewV2(domain byte) UUID {
	var u UUID

	timeNow, clockSeq, hardwareAddr := getStorage()

	if domain == DomainPerson {
		binary.BigEndian.PutUint32(u[0:], posixUID)
	} else if domain == DomainGroup {
		binary.BigEndian.PutUint32(u[0:], posixGID)
	}

	binary.BigEndian.PutUint16(u[4:], uint16(timeNow>>32))
	binary.BigEndian.PutUint16(u[6:], uint16(timeNow>>48))
	binary.BigEndian.PutUint16(u[8:], clockSeq)
	u[9] = domain

	copy(u[10:], hardwareAddr[:])

	u.SetVersion(2)
	u.SetVariant()
	return u
}

// NewV3 returns UUID based on MD5 hash of namespace UUID and name.
func NewV3(ns UUID, name string) UUID {
	u := newFromHash(md5.New(), ns, name)
	u.SetVersion(3)
	u.SetVariant()
	return u
}

// NewV4 returns random generated UUID.
func NewV4() UUID {
	var u UUID
	safeRandom(u[:])
	u.SetVersion(4)
	u.SetVariant()
	return u
}

// NewV5 returns UUID based on SHA-1 hash of namespace UUID and name.
func NewV5(ns UUID, name string) UUID {
	u := newFromHash(sha1.New(), ns, name)
	u.SetVersion(5)
	u.SetVariant()
	return u
}

// Returns UUID based on hashing of namespace UUID and name.
func newFromHash(h hash.Hash, ns UUID, name string) UUID {
	var u UUID
	h.Write(ns[:])
	h.Write([]byte(name))
	copy(u[:], h.Sum(nil))
	return u
}

// NewTime returns a time-based UUID. The first 40 bits are a unix timestamp, in
// network order. The last 86 are random bytes from the OS' CSPRNG. (Two other
// bits are the version, 'T', and variant.) 40 bits allows for a maximum
// timestamp of 274877906944, which is August of 10680.
func NewTime(t time.Time) UUID {
	var u UUID
	binary.BigEndian.PutUint64(u[:], uint64(t.Unix()<<24))

	safeRandom(u[ /* 5 */ 40/8:])
	u.SetVersion(6)
	u.SetVariant()
	return u
}

// Time returns the date encoded in the UUID, if any. Only applicable to UUIDs
// version one and those created with NewTime. The returned boolean will be true
// iff the UUID contains an encoded date.
func (u UUID) Time() (t time.Time, ok bool) {
	switch u.Version() {
	case 1:
		ts := int64(binary.BigEndian.Uint32(u[:])) |
			int64(binary.BigEndian.Uint16(u[4:]))<<32 |
			int64(binary.BigEndian.Uint16(u[6:])&0xFFF)<<48
		return time.Unix(ts, 0), true
	case 6:
		ts := int64(binary.BigEndian.Uint64(u[:])) >> 24
		return time.Unix(ts, 0), true
	default:
		return t, false
	}
}
