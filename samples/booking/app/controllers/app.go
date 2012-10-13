package controllers

import (
	"code.google.com/p/go.crypto/bcrypt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/samples/booking/app/models"
)

func addUser(c *rev.Controller) rev.Result {
	if user := connected(c); user != nil {
		c.RenderArgs["user"] = user
	}
	return nil
}

func connected(c *rev.Controller) *models.User {
	if c.RenderArgs["user"] != nil {
		return c.RenderArgs["user"].(*models.User)
	}
	if username, ok := c.Session["user"]; ok {
		return getUser(c, username)
	}
	return nil
}

func getUser(c *rev.Controller, username string) *models.User {
	rows, err := c.Txn.Query(`
select UserId, HashedPassword, Name
  from User where Username = :Username`, username)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil
	}

	user := &models.User{Username: username}
	err = rows.Scan(&user.UserId, &user.HashedPassword, &user.Name)
	if err != nil {
		panic(err)
	}
	return user
}

type Application struct {
	*rev.Controller
}

func (c Application) Index() rev.Result {
	if connected(c.Controller) != nil {
		return c.Redirect(Hotels.Index)
	}
	c.Flash.Error("Please log in first")
	return c.Render()
}

func (c Application) Register() rev.Result {
	return c.Render()
}

func (c Application) SaveUser(user models.User, verifyPassword string) rev.Result {
	c.Validation.Required(verifyPassword)
	c.Validation.Required(verifyPassword == user.Password).
		Message("Password does not match")
	user.Validate(c.Validation)

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Application.Register)
	}

	bcryptPassword, _ := bcrypt.GenerateFromPassword(
		[]byte(user.Password), bcrypt.DefaultCost)
	_, err := c.Txn.Exec(
		"insert into User (Username, HashedPassword, Name) values (?, ?, ?)",
		user.Username, bcryptPassword, user.Name)
	if err != nil {
		panic(err)
	}

	c.Session["user"] = user.Username
	c.Flash.Success("Welcome, " + user.Name)
	return c.Redirect(Hotels.Index)
}

func (c Application) Login(username, password string) rev.Result {
	user := getUser(c.Controller, username)
	err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
	if err == nil {
		c.Session["user"] = username
		c.Flash.Success("Welcome, " + username)
		return c.Redirect(Hotels.Index)
	}

	c.Flash.Out["username"] = username
	c.Flash.Error("Login failed")
	return c.Redirect(Application.Index)
}

func (c Application) Logout() rev.Result {
	for k := range c.Session {
		delete(c.Session, k)
	}
	return c.Redirect(Application.Index)
}

func init() {
	rev.InterceptFunc(addUser, rev.BEFORE, &Application{})
}
