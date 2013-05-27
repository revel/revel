// This module configures a database connection for the application.
//
// Developers use this module by importing and calling db.Init().
// A "Transactional" controller type is provided as a way to import interceptors
// that manage the transaction
//
// In particular, a transaction is begun before each request and committed on
// success.  If a panic occurred during the request, the transaction is rolled
// back.  (The application may also roll the transaction back itself.)
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

func Init() {
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

type Transactional struct {
	*revel.Controller
	Txn *sql.Tx
}

// Begin a transaction
func (c *Transactional) Begin() revel.Result {
	txn, err := Db.Begin()
	if err != nil {
		panic(err)
	}
	c.Txn = txn
	return nil
}

// Rollback if it's still going (must have panicked).
func (c *Transactional) Rollback() revel.Result {
	if c.Txn != nil {
		if err := c.Txn.Rollback(); err != nil {
			if err != sql.ErrTxDone {
				panic(err)
			}
		}
		c.Txn = nil
	}
	return nil
}

// Commit the transaction.
func (c *Transactional) Commit() revel.Result {
	if c.Txn != nil {
		if err := c.Txn.Commit(); err != nil {
			if err != sql.ErrTxDone {
				panic(err)
			}
		}
		c.Txn = nil
	}
	return nil
}

func init() {
	revel.InterceptMethod((*Transactional).Begin, revel.BEFORE)
	revel.InterceptMethod((*Transactional).Commit, revel.AFTER)
	revel.InterceptMethod((*Transactional).Rollback, revel.FINALLY)
}
