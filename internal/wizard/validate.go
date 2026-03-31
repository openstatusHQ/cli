package wizard

import (
	"fmt"
	"time"
)

func ValidRFC3339(fieldName string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s cannot be empty", fieldName)
		}
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			return fmt.Errorf("%s must be valid RFC 3339 format (e.g. 2006-01-02T15:04:05Z)", fieldName)
		}
		return nil
	}
}
