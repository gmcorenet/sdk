package gmcore_uid

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidUUIDFormat = errors.New("invalid UUID format")
	ErrInvalidUUID       = errors.New("invalid UUID")
)

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func IsValidUUID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return uuidPattern.MatchString(value)
}

func IsValidV4(value string) bool {
	if !IsValidUUID(value) {
		return false
	}

	parts := strings.Split(value, "-")
	if len(parts) != 5 {
		return false
	}

	version := parts[2][0]
	if version != '4' {
		return false
	}

	return true
}

func ParseUUID(value string) (UUID, error) {
	value = strings.TrimSpace(value)
	if !IsValidUUID(value) {
		return UUID{}, ErrInvalidUUIDFormat
	}

	parts := strings.Split(value, "-")
	if len(parts) != 5 {
		return UUID{}, ErrInvalidUUIDFormat
	}

	hexStr := strings.Join(parts, "")
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return UUID{}, ErrInvalidUUIDFormat
	}

	var u UUID
	copy(u[:], bytes)
	return u, nil
}

func MustParseUUID(value string) UUID {
	u, err := ParseUUID(value)
	if err != nil {
		panic(fmt.Sprintf("invalid UUID: %s", value))
	}
	return u
}

type PrimaryKeyType string

const (
	PrimaryKeyUUID PrimaryKeyType = "uuid"
	PrimaryKeyInt  PrimaryKeyType = "int"
)

func IsValidPrimaryKey(key string, pkType PrimaryKeyType) error {
	switch pkType {
	case PrimaryKeyUUID:
		if !IsValidUUID(key) {
			return fmt.Errorf("invalid UUID format: %s", key)
		}
	case PrimaryKeyInt:
		if !IsValidInt(key) {
			return fmt.Errorf("invalid integer: %s", key)
		}
	default:
		return fmt.Errorf("unsupported primary key type: %s", pkType)
	}
	return nil
}

func IsValidInt(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	_, err := strconv.ParseInt(value, 10, 64)
	return err == nil
}
