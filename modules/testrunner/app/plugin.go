package app

import (
	"fmt"
	"github.com/mcspring/revel"
)

func init() {
	revel.OnAppStart(func() {
		fmt.Println("Go to /@tests to run the tests.")
	})
}
