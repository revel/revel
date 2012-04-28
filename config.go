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

func (c *MergedConfig) SetSection(section string) {
	c.section = section
}

func (c *MergedConfig) Int(option string) (int, error) {
	if r, err := c.Config.Int(c.section, option); err == nil {
		return r, nil
	}
	return c.Config.Int("DEFAULT", option)
}

func (c *MergedConfig) String(option string) (string, error) {
	if r, err := c.Config.String(c.section, option); err == nil {
		return r, nil
	}
	return c.Config.String("DEFAULT", option)
}
