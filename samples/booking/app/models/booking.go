package models

import (
	"fmt"
	"play"
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
func (b Booking) Validate(v *play.Validation) {
	v.Required(b.User).Key("booking.User")
	v.Required(b.Hotel).Key("booking.Hotel")
	v.Required(b.CheckInDate).Key("booking.CheckInDate")
	v.Required(b.CheckOutDate).Key("booking.CheckOutDate")

	v.Match(b.CardNumber, regexp.MustCompile(`\d{16}`)).
		Key("booking.CardNumber").
		Message("Credit card number must be numeric and 16 digits")

	v.Required(b.NameOnCard).Key("booking.NameOnCard")
	v.MinSize(b.NameOnCard, 3).Key("booking.NameOnCard")
	v.MaxSize(b.NameOnCard, 70).Key("booking.NameOnCard")
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
