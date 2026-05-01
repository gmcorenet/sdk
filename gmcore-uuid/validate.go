package gmcoreuuid

import (
	"fmt"
	"strconv"
	"strings"
)

type PrimaryKeyType string

const (
	PrimaryKeyUUID PrimaryKeyType = "uuid"
	PrimaryKeyInt  PrimaryKeyType = "int"
)

func IsValidPrimaryKey(key string, pkType PrimaryKeyType) error {
	switch pkType {
	case PrimaryKeyUUID:
		if !IsValid(key) {
			return fmt.Errorf("invalid UUID format: %s", key)
		}
	case PrimaryKeyInt:
		if !IsValidInt(key) {
			return fmt.Errorf("invalid integer: %s", key)
		}
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
