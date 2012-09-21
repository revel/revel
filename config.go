package rev

import (
	"github.com/kless/goconfig/config"
)

// This handles the parsing of application.conf
// It has a "preferred" section that is checked first for option queries.
// If the preferred section does not have the option, the DEFAULT section is
// checked fallback.
type MergedConfig struct {
	*config.Config
	section string // Check this section first, then fall back to DEFAULT
}

func LoadConfig(fname string) (*MergedConfig, error) {
	conf, err := config.ReadDefault(fname)
	if err != nil {
		return nil, err
	}
	return &MergedConfig{conf, ""}, nil
}

// The top-level keys are in a section called "DEFAULT".  The DEFAULT is used as
// a fall-back if the key is not found in the section that has been set.
const defaultSection = "DEFAULT"

func (c *MergedConfig) SetSection(section string) {
	c.section = section
}

func (c *MergedConfig) Int(option string) (result int, found bool) {
	if result, found = getInt(c.Config, c.section, option); found {
		return
	}
	if result, found = getInt(c.Config, defaultSection, option); found {
		return
	}
	return 0, false
}

func (c *MergedConfig) IntDefault(option string, dfault int) int {
	if r, found := c.Int(option); found {
		return r
	}
	return dfault
}

func (c *MergedConfig) Bool(option string) (result, found bool) {
	if result, found = getBool(c.Config, c.section, option); found {
		return
	}
	if result, found = getBool(c.Config, defaultSection, option); found {
		return
	}
	return false, false
}

func (c *MergedConfig) BoolDefault(option string, dfault bool) bool {
	if r, found := c.Bool(option); found {
		return r
	}
	return dfault
}

func (c *MergedConfig) String(option string) (result string, found bool) {
	if r, err := c.Config.String(c.section, option); err == nil {
		return stripQuotes(r), true
	}
	r, err := c.Config.String(defaultSection, option)
	return stripQuotes(r), err == nil
}

func (c *MergedConfig) StringDefault(option, dfault string) string {
	if r, found := c.String(option); found {
		return r
	}
	return dfault
}

// Helpers

func stripQuotes(s string) string {
	if s == "" {
		return s
	}

	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}

	return s
}

func getInt(config *config.Config, section, option string) (result int, found bool) {
	// TODO: config.HasOption checks the DEFAULT section, while config.Int does not.
	// File an issue to make them agree, and use HasOption instead of RawString.
	if _, err := config.RawString(section, option); err == nil {
		r, err := config.Int(section, option)
		if err == nil {
			return r, true
		}
		ERROR.Println("Failed to parse config option", option, "as int:", err)
	}
	return 0, false
}

func getBool(config *config.Config, section, option string) (result, found bool) {
	if config.HasOption(section, option) {
		r, err := config.Bool(section, option)
		if err == nil {
			return r, true
		}
		ERROR.Println("Failed to parse config option", option, "as bool:", err)
	}
	return false, false
}
