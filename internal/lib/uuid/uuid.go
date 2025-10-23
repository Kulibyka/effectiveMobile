package uuid

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var ErrInvalidUUID = errors.New("invalid uuid format")

type UUID string

func Parse(s string) (UUID, error) {
	if len(s) != 36 {
		return "", ErrInvalidUUID
	}

	lower := strings.ToLower(s)
	for i, r := range lower {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return "", ErrInvalidUUID
			}
		default:
			if !isHex(r) {
				return "", ErrInvalidUUID
			}
		}
	}

	return UUID(lower), nil
}

func (u UUID) String() string {
	return string(u)
}

func (u UUID) Value() (driver.Value, error) {
	if u == "" {
		return nil, nil
	}

	return string(u), nil
}

func (u *UUID) Scan(src any) error {
	if src == nil {
		*u = ""
		return nil
	}

	switch v := src.(type) {
	case string:
		parsed, err := Parse(v)
		if err != nil {
			return err
		}
		*u = parsed
		return nil
	case []byte:
		parsed, err := Parse(string(v))
		if err != nil {
			return err
		}
		*u = parsed
		return nil
	default:
		return fmt.Errorf("%w: unexpected type %T", ErrInvalidUUID, src)
	}
}

func isHex(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'a' && r <= 'f')
}
