package models

import (
	"play"
)

type Hotel struct {
	HotelId          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

func (h *Hotel) Validate(v *play.Validation) {
	v.Required(h.Name).Key("hotel.Name")
	v.MaxSize(h.Name, 50).Key("hotel.Name")

	v.MaxSize(h.Address, 100).Key("hotel.Address")

	v.Required(h.City).Key("hotel.City")
	v.MaxSize(h.City, 40).Key("hotel.City")

	v.Required(h.State).Key("hotel.State")
	v.MaxSize(h.State, 6).Key("hotel.State")
	v.MinSize(h.State, 2).Key("hotel.State")

	v.Required(h.Zip).Key("hotel.Zip")
	v.MaxSize(h.Zip, 6).Key("hotel.Zip")
	v.MinSize(h.Zip, 5).Key("hotel.Zip")

	v.Required(h.Country).Key("hotel.Country")
	v.MaxSize(h.Country, 40).Key("hotel.Country")
	v.MinSize(h.Country, 2).Key("hotel.Country")
}
