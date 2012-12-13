// This plugin provides a database transaction to the application.
// A transaction is begun before each request and committed on success.
// If a panic occurred during the request, the transaction is rolled back.
// (The application may also roll the transaction back itself.)
package db

import (
	"database/sql"
	"github.com/robfig/revel"
)

var (
	Db     *sql.DB
	Driver string
	Spec   string
)

type DbPlugin struct {
	rev.EmptyPlugin
}

func (p DbPlugin) OnAppStart() {
	// Read configuration.
	var found bool
	if Driver, found = rev.Config.String("db.driver"); !found {
		rev.ERROR.Fatal("No db.driver found.")
	}
	if Spec, found = rev.Config.String("db.spec"); !found {
		rev.ERROR.Fatal("No db.spec found.")
	}

	// Open a connection.
	var err error
	Db, err = sql.Open(Driver, Spec)
	if err != nil {
		rev.ERROR.Fatal(err)
	}
}

// Begin a transaction.
func (p DbPlugin) BeforeRequest(c *rev.Controller) {
	txn, err := Db.Begin()
	if err != nil {
		panic(err)
	}
	c.Txn = txn
}

// Commit the active transaction.
func (p DbPlugin) AfterRequest(c *rev.Controller) {
	if err := c.Txn.Commit(); err != nil {
		if err != sql.ErrTxDone {
			panic(err)
		}
	}
	c.Txn = nil
}

// Rollback the active transaction, if any.
func (p DbPlugin) OnException(c *rev.Controller, err interface{}) {
	if c.Txn == nil {
		return
	}
	if err := c.Txn.Rollback(); err != nil {
		if err != sql.ErrTxDone {
			panic(err)
		}
	}
}
