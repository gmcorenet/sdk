package gmcore_uuid

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	UUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func IsValid(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return UUIDPattern.MatchString(value)
}

func IsValidV4(value string) bool {
	if !IsValid(value) {
		return false
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Version() == 4
}

func New() string {
	return uuid.New().String()
}

func NewV4() string {
	return uuid.New().String()
}

func Parse(value string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(value))
}

func MustParse(value string) uuid.UUID {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		panic(fmt.Sprintf("invalid UUID: %s", value))
	}
	return parsed
}
