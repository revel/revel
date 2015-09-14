package connections

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/robfig/config"
	"github.com/robfig/revel"
	"testing"
)

func TestConnection(t *testing.T) {
	revel.ConfPaths = append(revel.ConfPaths, "../conf")
	cfg, err := revel.LoadConfig("sample.conf")
	if err != nil {
		t.Fatal("LoadConfig() error:", err)
	}

	revel.Config = cfg
	revel.Config.SetSection(config.DEFAULT_SECTION)

	Init()
}
