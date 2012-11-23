package models

import (
	"fmt"
	"github.com/coopernurse/gorp"
	"github.com/robfig/revel"
	"regexp"
	"time"
)

type Booking struct {
	BookingId    int
	UserId       int
	HotelId      int
	CheckInDate  time.Time
	CheckOutDate time.Time
	CardNumber   string
	NameOnCard   string
	CardExpMonth int
	CardExpYear  int
	Smoking      bool
	Beds         int

	User  *User
	Hotel *Hotel
}

// TODO: Make an interface for Validate() and then validation can pass in the
// key prefix ("booking.")
func (booking Booking) Validate(v *rev.Validation) {
	v.Required(booking.User)
	v.Required(booking.Hotel)
	v.Required(booking.CheckInDate)
	v.Required(booking.CheckOutDate)

	v.Match(booking.CardNumber, regexp.MustCompile(`\d{16}`)).
		Message("Credit card number must be numeric and 16 digits")

	v.Check(booking.NameOnCard,
		rev.Required{},
		rev.MinSize{3},
		rev.MaxSize{70},
	)
}

func (b Booking) Total() int {
	return b.Hotel.Price * b.Nights()
}

func (b Booking) Nights() int {
	return int((b.CheckOutDate.Unix() - b.CheckInDate.Unix()) / 60 / 60 / 24)
}

const DATE_FORMAT = "Jan _2, 2006"

func (b Booking) Description() string {
	if b.Hotel == nil {
		return ""
	}

	return fmt.Sprintf("%s, %s to %s",
		b.Hotel.Name,
		b.CheckInDate.Format(DATE_FORMAT),
		b.CheckOutDate.Format(DATE_FORMAT))
}

func (b Booking) String() string {
	return fmt.Sprintf("Booking(%s,%s)", b.User, b.Hotel)
}

func (b *Booking) PreInsert(_ gorp.SqlExecutor) error {
	if b.User != nil {
		b.UserId = b.User.UserId
	}
	if b.Hotel != nil {
		b.HotelId = b.Hotel.HotelId
	}
	return nil
}
