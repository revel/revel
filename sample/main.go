package main

import (
	"play"
	"sample/app/controllers"
)

func main() {
	play.RegisterController((*controllers.Application)(nil))
	play.Run()
}
