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

var DbFilter dbFilter

type dbFilter struct{}

func (p dbFilter) OnAppStart() {
	// Read configuration.
	var found bool
	if Driver, found = revel.Config.String("db.driver"); !found {
		revel.ERROR.Fatal("No db.driver found.")
	}
	if Spec, found = revel.Config.String("db.spec"); !found {
		revel.ERROR.Fatal("No db.spec found.")
	}

	// Open a connection.
	var err error
	Db, err = sql.Open(Driver, Spec)
	if err != nil {
		revel.ERROR.Fatal(err)
	}
}

func (p dbFilter) Call(c *revel.Controller, fc revel.FilterChain) {
	// Begin transaction
	txn, err := Db.Begin()
	if err != nil {
		panic(err)
	}
	c.Txn = txn

	// Catch panics and roll back.
	defer func() {
		if err := c.Txn.Rollback(); err != nil {
			if err != sql.ErrTxDone {
				panic(err)
			}
		}
	}()

	fc[0].Call(c, fc[1:])

	// Commit
	if err := c.Txn.Commit(); err != nil {
		if err != sql.ErrTxDone {
			panic(err)
		}
	}
	c.Txn = nil
}
