package schema

import (
	"errors"
	"fmt"
)

func Validate(s *Schema) error {
	if s == nil {
		return errors.New("schema is nil")
	}
	if s.ID == "" {
		return errors.New("schema id is required")
	}
	if s.Standard == "" {
		return errors.New("schema standard is required")
	}
	if s.Transaction == "" && s.Message == "" {
		return errors.New("schema transaction or message is required")
	}
	if len(s.Maps) == 0 && len(s.Mapping) == 0 {
		return fmt.Errorf("schema %q must define maps or mapping", s.ID)
	}
	return nil
}
