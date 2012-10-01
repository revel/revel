package controllers

import (
	"code.google.com/p/go.crypto/bcrypt"
	"database/sql"
	"fmt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/samples/booking/app/models"
	"strings"
	"time"
)

func checkUser(c *rev.Controller) rev.Result {
	if user := connected(c); user == nil {
		c.Flash.Error("Please log in first")
		return c.Redirect(Application.Index)
	}
	return nil
}

type Hotels struct {
	*rev.Controller
}

func (c Hotels) Index() rev.Result {
	rows, err := c.Txn.Query(`
select BookingId, UserId, HotelId, CheckInDate, CheckOutDate,
       CardNumber, NameOnCard, CardExpMonth, CardExpYear, Smoking, Beds
  from Booking
 where UserId = ?`, connected(c.Controller).UserId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var bookings []*models.Booking
	var checkInDate, checkOutDate string
	for rows.Next() {
		b := &models.Booking{}
		err := rows.Scan(&b.BookingId, &b.UserId, &b.HotelId, &checkInDate, &checkOutDate,
			&b.CardNumber, &b.NameOnCard, &b.CardExpMonth, &b.CardExpYear, &b.Smoking, &b.Beds)
		if err != nil {
			panic(err)
		}
		b.CheckInDate, _ = time.Parse("2006-01-02", checkInDate)
		b.CheckOutDate, _ = time.Parse("2006-01-02", checkOutDate)
		bookings = append(bookings, b)
	}

	for _, b := range bookings {
		b.Hotel = c.loadHotelById(b.HotelId)
	}

	return c.Render(bookings)
}

func (c Hotels) List(search string, size, page int) rev.Result {
	if page == 0 {
		page = 1
	}
	nextPage := page + 1
	search = strings.TrimSpace(search)

	var hotels []*models.Hotel
	if search == "" {
		hotels = loadHotels(c.Txn.Query(`
select HotelId, Name, Address, City, State, Zip, Country, Price
  from Hotel
 limit ?, ?`, (page-1)*size, size))
	} else {
		search = strings.ToLower(search)
		hotels = loadHotels(c.Txn.Query(`
select HotelId, Name, Address, City, State, Zip, Country, Price
  from Hotel
 where lower(Name) like ? or lower(City) like ?
 limit ?, ?`, "%"+search+"%", "%"+search+"%", (page-1)*size, size))
	}

	return c.Render(hotels, search, size, page, nextPage)
}

func loadHotels(rows *sql.Rows, err error) []*models.Hotel {
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var hotels []*models.Hotel
	for rows.Next() {
		h := &models.Hotel{}
		err := rows.Scan(&h.HotelId, &h.Name,
			&h.Address, &h.City, &h.State, &h.Zip, &h.Country, &h.Price)
		if err != nil {
			panic(err)
		}
		hotels = append(hotels, h)
	}
	return hotels
}

func (c Hotels) loadHotelById(id int) *models.Hotel {
	hotels := loadHotels(c.Txn.Query(`
select HotelId, Name, Address, City, State, Zip, Country, Price
  from Hotel
 where HotelId = ?`, id))
	if len(hotels) == 0 {
		return nil
	}
	return hotels[0]
}

func (c Hotels) Show(id int) rev.Result {
	var title string
	hotel := c.loadHotelById(id)
	if hotel == nil {
		title = "Not found"
		// 	TODO: return c.NotFound("Hotel does not exist")
	} else {
		title = hotel.Name
	}
	return c.Render(title, hotel)
}

func (c Hotels) Settings() rev.Result {
	return c.Render()
}

func (c Hotels) SaveSettings(password, verifyPassword string) rev.Result {
	models.ValidatePassword(c.Validation, password).Key("password")
	c.Validation.Required(verifyPassword).
		Key("verifyPassword").
		Message("Please verify your password")
	c.Validation.Required(password == verifyPassword).
		Key("verifyPassword").
		Message("Your password doesn't match")
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		return c.Redirect(Hotels.Settings)
	}

	bcryptPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := c.Txn.Exec("update User set HashedPassword = ? where UserId = ?",
		bcryptPassword, connected(c.Controller).UserId)
	if err != nil {
		panic(err)
	}
	c.Flash.Success("Password updated")
	return c.Redirect(Hotels.Index)
}

func (c Hotels) ConfirmBooking(id int, booking models.Booking) rev.Result {
	hotel := c.loadHotelById(id)
	title := fmt.Sprintf("Confirm %s booking", hotel.Name)
	booking.Hotel = hotel
	booking.User = connected(c.Controller)
	booking.Validate(c.Validation)

	if c.Validation.HasErrors() || c.Params.Get("revise") != "" {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect("/hotels/%d/booking", id)
	}

	if c.Params.Get("confirm") != "" {
		result, err := c.Txn.Exec(`
insert into Booking (
  UserId, HotelId, CheckInDate, CheckOutDate,
  CardNumber, NameOnCard, CardExpMonth, CardExpYear,
  Smoking, Beds
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			booking.User.UserId,
			booking.Hotel.HotelId,
			booking.CheckInDate.Format("2006-01-02"),
			booking.CheckOutDate.Format("2006-01-02"),
			booking.CardNumber,
			booking.NameOnCard,
			booking.CardExpMonth,
			booking.CardExpYear,
			booking.Smoking,
			booking.Beds)
		if err != nil {
			panic(err)
		}
		bookingId, _ := result.LastInsertId()
		c.Flash.Success("Thank you, %s, your confirmation number for %s is %d",
			booking.User.Name, hotel.Name, bookingId)
		return c.Redirect(Hotels.Index)
	}

	return c.Render(title, hotel, booking)
}

func (c Hotels) CancelBooking(id int) rev.Result {
	_, err := c.Txn.Exec("delete from Booking where BookingId = ?", id)
	if err != nil {
		panic(err)
	}
	c.Flash.Success(fmt.Sprintln("Booking cancelled for confirmation number", id))
	return c.Redirect(Hotels.Index)
}

func (c Hotels) Book(id int) rev.Result {
	hotel := c.loadHotelById(id)
	title := "Book " + hotel.Name
	// if hotel == nil {
	// 	return c.NotFound("Hotel does not exist")
	// }
	return c.Render(title, hotel)
}

func init() {
	rev.InterceptFunc(checkUser, rev.BEFORE, &Hotels{})
}
