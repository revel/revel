package models

import (
	"github.com/robfig/revel"
)

type Hotel struct {
	HotelId          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

func (h *Hotel) Validate(v *rev.Validation) {
	v.Check(h.Name,
		rev.Required{},
		rev.MaxSize{50},
	).Key("hotel.Name")

	v.MaxSize(h.Address, 100).Key("hotel.Address")

	v.Check(h.City,
		rev.Required{},
		rev.MaxSize{40},
	).Key("hotel.City")

	v.Check(h.State,
		rev.Required{},
		rev.MaxSize{6},
		rev.MinSize{2},
	).Key("hotel.State")

	v.Check(h.Zip,
		rev.Required{},
		rev.MaxSize{6},
		rev.MinSize{5},
	).Key("hotel.Zip")

	v.Check(h.Country,
		rev.Required{},
		rev.MaxSize{40},
		rev.MinSize{2},
	).Key("hotel.Country")
}
