package gmcore_uid

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
)

var ErrRandomRead = errors.New("failed to read random bytes")

type UID interface {
	String() string
	Bytes() []byte
}

type UUID [16]byte

func NewUUID() (UUID, error) {
	var u UUID
	_, err := rand.Read(u[:])
	if err != nil {
		return u, err
	}
	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80
	return u, nil
}

func (u UUID) String() string {
	return hex.EncodeToString(u[:])
}

func (u UUID) Bytes() []byte {
	return u[:]
}

type ULID [16]byte

func NewULID() (ULID, error) {
	var u ULID
	timestamp := uint64(time.Now().UnixNano() / 1000000)
	for i := 0; i < 6; i++ {
		u[i] = byte(timestamp >> (5 * uint(i)))
	}
	_, err := rand.Read(u[6:])
	if err != nil {
		return u, err
	}
	return u, nil
}

func (u ULID) String() string {
	return base64.URLEncoding.EncodeToString(u[:])
}

func (u ULID) Bytes() []byte {
	return u[:]
}

type NanoID struct {
	value []byte
}

func NewNanoID(size int) (string, error) {
	if size <= 0 {
		return "", errors.New("size must be positive")
	}
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	alphabetLen := len(alphabet)
	mask := nextPowerOfTwo(alphabetLen)*2 - 1
	b := make([]byte, size)
	for i := range b {
		for {
			randByte := make([]byte, 1)
			if _, err := rand.Read(randByte); err != nil {
				return "", ErrRandomRead
			}
			index := int(randByte[0]) & mask
			if index < alphabetLen {
				b[i] = alphabet[index]
				break
			}
		}
	}
	return string(b), nil
}

func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

type Factory struct {
	encoding string
	size     int
}

func NewFactory(size int) *Factory {
	return &Factory{size: size}
}

func (f *Factory) Make() string {
	id, err := NewNanoID(f.size)
	if err != nil {
		return ""
	}
	return id
}

func (f *Factory) MakeUUID() string {
	u, err := NewUUID()
	if err != nil {
		return ""
	}
	return u.String()
}

func (f *Factory) MakeULID() string {
	u, err := NewULID()
	if err != nil {
		return ""
	}
	return u.String()
}
