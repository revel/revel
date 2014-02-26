package models

import (
	"github.com/revel/revel"
)

type Hotel struct {
	HotelId          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

func (hotel *Hotel) Validate(v *revel.Validation) {
	v.Check(hotel.Name,
		revel.Required{},
		revel.MaxSize{50},
	)

	v.MaxSize(hotel.Address, 100)

	v.Check(hotel.City,
		revel.Required{},
		revel.MaxSize{40},
	)

	v.Check(hotel.State,
		revel.Required{},
		revel.MaxSize{6},
		revel.MinSize{2},
	)

	v.Check(hotel.Zip,
		revel.Required{},
		revel.MaxSize{6},
		revel.MinSize{5},
	)

	v.Check(hotel.Country,
		revel.Required{},
		revel.MaxSize{40},
		revel.MinSize{2},
	)
}
