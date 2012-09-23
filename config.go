package rev

import (
	"errors"
	"github.com/robfig/goconfig/config"
	"path"
)

// This handles the parsing of application.conf
// It has a "preferred" section that is checked first for option queries.
// If the preferred section does not have the option, the DEFAULT section is
// checked fallback.
type MergedConfig struct {
	*config.Config
	section string // Check this section first, then fall back to DEFAULT
}

func LoadConfig(confName string) (*MergedConfig, error) {
	var err error
	for _, confPath := range ConfPaths {
		conf, err := config.ReadDefault(path.Join(confPath, confName))
		if err == nil {
			return &MergedConfig{conf, ""}, nil
		}
	}
	if err == nil {
		err = errors.New("not found")
	}
	return nil, err
}

func (c *MergedConfig) SetSection(section string) {
	c.section = section
}

func (c *MergedConfig) Int(option string) (result int, found bool) {
	result, err := c.Config.Int(c.section, option)
	if err == nil {
		return result, true
	}
	if _, ok := err.(config.OptionError); ok {
		return 0, false
	}

	// If it wasn't an OptionError, it must have failed to parse.
	ERROR.Println("Failed to parse config option", option, "as int:", err)
	return 0, false
}

func (c *MergedConfig) IntDefault(option string, dfault int) int {
	if r, found := c.Int(option); found {
		return r
	}
	return dfault
}

func (c *MergedConfig) Bool(option string) (result, found bool) {
	result, err := c.Config.Bool(c.section, option)
	if err == nil {
		return result, true
	}
	if _, ok := err.(config.OptionError); ok {
		return false, false
	}

	// If it wasn't an OptionError, it must have failed to parse.
	ERROR.Println("Failed to parse config option", option, "as bool:", err)
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
	return "", false
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
