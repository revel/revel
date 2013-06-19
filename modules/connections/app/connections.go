package connections

// Multiple named connections configuration module.
// It support the following config format:

// connection.mysql.drive = mysql
// connection.mysql.spec = user:pass@localhost:3306
// connection.pgsql.drive = pgsql
// connection.pgsql.spec = postgres:pass@dbhost:5432
//
// Usage:
// connections.Init()
// Connections[name].Db

import (
	"database/sql"
	"github.com/3d0c/revel"
)

type Connection struct {
	Db     *sql.DB
	Driver string
	Spec   string
}

var Connections = map[string]Connection{}

func Init() {
	connections := revel.Config.Array("connections").(map[string]interface{})

	if len(connections) == 0 {
		revel.ERROR.Fatal("No `connections` entry found.")
	}

	for name, options := range connections {
		driver := options.(map[string]interface{})["driver"].(string)
		spec := options.(map[string]interface{})["spec"].(string)

		db, err := sql.Open(driver, spec)
		if err != nil {
			revel.ERROR.Fatal(err)
		}

		Connections[name] = Connection{db, driver, spec}
	}
}
