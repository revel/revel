package controllers

import (
	"code.google.com/p/go.crypto/bcrypt"
	"fmt"
	"github.com/revel/revel"
	"github.com/revel/revel/samples/booking/app/models"
	"github.com/revel/revel/samples/booking/app/routes"
	"strings"
)

type Hotels struct {
	Application
}

func (c Hotels) checkUser() revel.Result {
	if user := c.connected(); user == nil {
		c.Flash.Error("Please log in first")
		return c.Redirect(routes.Application.Index())
	}
	return nil
}

func (c Hotels) Index() revel.Result {
	results, err := c.Txn.Select(models.Booking{},
		`select * from Booking where UserId = ?`, c.connected().UserId)
	if err != nil {
		panic(err)
	}

	var bookings []*models.Booking
	for _, r := range results {
		b := r.(*models.Booking)
		bookings = append(bookings, b)
	}

	return c.Render(bookings)
}

func (c Hotels) List(search string, size, page int) revel.Result {
	if page == 0 {
		page = 1
	}
	nextPage := page + 1
	search = strings.TrimSpace(search)

	var hotels []*models.Hotel
	if search == "" {
		hotels = loadHotels(c.Txn.Select(models.Hotel{},
			`select * from Hotel limit ?, ?`, (page-1)*size, size))
	} else {
		search = strings.ToLower(search)
		hotels = loadHotels(c.Txn.Select(models.Hotel{},
			`select * from Hotel where lower(Name) like ? or lower(City) like ?
 limit ?, ?`, "%"+search+"%", "%"+search+"%", (page-1)*size, size))
	}

	return c.Render(hotels, search, size, page, nextPage)
}

func loadHotels(results []interface{}, err error) []*models.Hotel {
	if err != nil {
		panic(err)
	}
	var hotels []*models.Hotel
	for _, r := range results {
		hotels = append(hotels, r.(*models.Hotel))
	}
	return hotels
}

func (c Hotels) loadHotelById(id int) *models.Hotel {
	h, err := c.Txn.Get(models.Hotel{}, id)
	if err != nil {
		panic(err)
	}
	if h == nil {
		return nil
	}
	return h.(*models.Hotel)
}

func (c Hotels) Show(id int) revel.Result {
	hotel := c.loadHotelById(id)
	if hotel == nil {
		return c.NotFound("Hotel %d does not exist", id)
	}
	title := hotel.Name
	return c.Render(title, hotel)
}

func (c Hotels) Settings() revel.Result {
	return c.Render()
}

func (c Hotels) SaveSettings(password, verifyPassword string) revel.Result {
	models.ValidatePassword(c.Validation, password)
	c.Validation.Required(verifyPassword).
		Message("Please verify your password")
	c.Validation.Required(verifyPassword == password).
		Message("Your password doesn't match")
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		return c.Redirect(routes.Hotels.Settings())
	}

	bcryptPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := c.Txn.Exec("update User set HashedPassword = ? where UserId = ?",
		bcryptPassword, c.connected().UserId)
	if err != nil {
		panic(err)
	}
	c.Flash.Success("Password updated")
	return c.Redirect(routes.Hotels.Index())
}

func (c Hotels) ConfirmBooking(id int, booking models.Booking) revel.Result {
	hotel := c.loadHotelById(id)
	if hotel == nil {
		return c.NotFound("Hotel %d does not exist", id)
	}

	title := fmt.Sprintf("Confirm %s booking", hotel.Name)
	booking.Hotel = hotel
	booking.User = c.connected()
	booking.Validate(c.Validation)

	if c.Validation.HasErrors() || c.Params.Get("revise") != "" {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(routes.Hotels.Book(id))
	}

	if c.Params.Get("confirm") != "" {
		err := c.Txn.Insert(&booking)
		if err != nil {
			panic(err)
		}
		c.Flash.Success("Thank you, %s, your confirmation number for %s is %d",
			booking.User.Name, hotel.Name, booking.BookingId)
		return c.Redirect(routes.Hotels.Index())
	}

	return c.Render(title, hotel, booking)
}

func (c Hotels) CancelBooking(id int) revel.Result {
	_, err := c.Txn.Delete(&models.Booking{BookingId: id})
	if err != nil {
		panic(err)
	}
	c.Flash.Success(fmt.Sprintln("Booking cancelled for confirmation number", id))
	return c.Redirect(routes.Hotels.Index())
}

func (c Hotels) Book(id int) revel.Result {
	hotel := c.loadHotelById(id)
	if hotel == nil {
		return c.NotFound("Hotel %d does not exist", id)
	}

	title := "Book " + hotel.Name
	return c.Render(title, hotel)
}
