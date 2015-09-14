package revel

import (
	"github.com/robfig/config"
	"reflect"
	"testing"
)

func TestArray(t *testing.T) {
	c := config.NewDefault()

	if !c.AddOption(config.DEFAULT_SECTION, "connections.mysql.driver", "mysql_driver") {
		t.Fatal("Unable to add option.")
	}
	if !c.AddOption(config.DEFAULT_SECTION, "connections.pgsql.driver", "pgsql_driver") {
		t.Fatal("Unable to add option.")
	}

	cfg := &MergedConfig{c, ""}

	cfg.SetSection(config.DEFAULT_SECTION)

	result := cfg.Array("connections")

	expected := map[string]interface{}{
		"mysql": map[string]interface{}{"driver": "mysql_driver"},
		"pgsql": map[string]interface{}{"driver": "pgsql_driver"},
	}

	if !reflect.DeepEqual(expected, result) {
		t.Fatal("Expected: ", expected, "Got: ", result)
	}
}
