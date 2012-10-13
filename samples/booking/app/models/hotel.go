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

func (hotel *Hotel) Validate(v *rev.Validation) {
	v.Check(hotel.Name,
		rev.Required{},
		rev.MaxSize{50},
	)

	v.MaxSize(hotel.Address, 100)

	v.Check(hotel.City,
		rev.Required{},
		rev.MaxSize{40},
	)

	v.Check(hotel.State,
		rev.Required{},
		rev.MaxSize{6},
		rev.MinSize{2},
	)

	v.Check(hotel.Zip,
		rev.Required{},
		rev.MaxSize{6},
		rev.MinSize{5},
	)

	v.Check(hotel.Country,
		rev.Required{},
		rev.MaxSize{40},
		rev.MinSize{2},
	)
}
