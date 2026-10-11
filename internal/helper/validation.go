package helper

import (
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

func ValidUUID(id string) bool {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	return err == nil
}

func MaxLength(s string, max int) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= max
}

func OptionalMaxLength(s *string, max int) bool { return s == nil || MaxLength(*s, max) }

func ValidUUIDs(ids []string) bool {
	for _, id := range ids {
		if !ValidUUID(id) {
			return false
		}
	}
	return true
}
