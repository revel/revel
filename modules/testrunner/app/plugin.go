package app

import (
	"github.com/revel/revel"
)

func init() {
	revel.OnAppStart(func() {
		revel.INFO.Print("Go to /@tests to run the tests.")
	})
}
