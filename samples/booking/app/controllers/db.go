package controllers

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/revel"
)

// This plugin manages transaction-per-request for any controllers that embed
// GorpController.
type GorpController struct {
	*rev.Controller
}

var (
	db *sql.DB
)

type DbPlugin struct {
	rev.EmptyPlugin
}

func (p DbPlugin) OnAppStart() {
	fmt.Println("Open DB")
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	// Create tables
	_, err = db.Exec(`
create table User (
  UserId   integer primary key autoincrement,
  Username varchar(20),
  Password varchar(20),
  Name varchar(100))`)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("insert into User (Username, Password, Name)" +
		" values ('demo', 'demo', 'Demo User')")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
create table Booking (
  BookingId    integer primary key autoincrement,
  UserId       int,
  HotelId      int,
  CheckInDate  datetime,
  CheckOutDate datetime,
  CardNumber   varchar(16),
  NameOnCard   varchar(50),
  CardExpMonth int,
  CardExpYear  int,
  Smoking      boolean,
  Beds         int
)`)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
create table Hotel (
  HotelId integer primary key autoincrement,
  Name    varchar(50),
  Address varchar(100),
  City    varchar(40),
  State   varchar(6),
  Zip     varchar(6),
  Country varchar(40),
  Price   int
)`)
	if err != nil {
		panic(err)
	}

	hotels := []string{
		"('Marriott Courtyard', 'Tower Pl, Buckhead', 'Atlanta', 'GA', '30305', 'USA', 120)",
		"('W Hotel', 'Union Square, Manhattan', 'New York', 'NY', '10011', 'USA', 450)",
		"('Hotel Rouge', '1315 16th St NW', 'Washington', 'DC', '20036', 'USA', 250)",
	}

	for _, h := range hotels {
		_, err = db.Exec(`insert into Hotel
(Name, Address, City, State, Zip, Country, Price)
 values ` + h)
		if err != nil {
			panic(err)
		}
	}
}

func (p DbPlugin) BeforeRequest(c *rev.Controller) {
	txn, err := db.Begin()
	if err != nil {
		panic(err)
	}
	c.Txn = txn
}

func (p DbPlugin) AfterRequest(c *rev.Controller) {
	if err := c.Txn.Commit(); err != nil {
		if err != sql.ErrTxDone {
			panic(err)
		}
	}
	c.Txn = nil
}

func (p DbPlugin) OnException(c *rev.Controller, err interface{}) {
	if err := c.Txn.Rollback(); err != nil {
		if err != sql.ErrTxDone {
			panic(err)
		}
	}
}

func init() {
	rev.RegisterPlugin(DbPlugin{})
}
