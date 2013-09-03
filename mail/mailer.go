package mail

import (
	"fmt"
	"net/smtp"
)

type Mailer struct {
	Address        string
	Port           int
	UserName       string
	Password       string
	Authentication string
	Auth           smtp.Auth // This is optional, only used if Authentication is not plain
}

// This is convinient method to send single email with either plain text or html body
func (m *Mailer) SendMail(from string, to []string, subject string, body string, html bool) error {
	message := &Message{From: from, To: to, Subject: subject}

	if html {
		message.HtmlBody = body
	} else {
		message.PlainBody = body
	}

	return m.SendMails([]*Message{message})
}

// This is the convinient method to send single email rendered from a view template with dynamic data
func (m *Mailer) SendFromTemplate(from string, to []string, subject string, template, string, html bool, args ...interface{}) error {
	// going to implement
	return nil
}

// send multiple emails in a single connection
func (m *Mailer) SendMails(messages []*Message) (err error) {
	addr := fmt.Sprintf("%s:%d", m.Address, m.Port)

	if m.Authentication == "plain" {
		m.Auth = smtp.PlainAuth(m.UserName, m.UserName, m.Password, m.Address)
	}

	c, err := m.Transport(addr, m.Auth)
	if err != nil {
		return
	}
	defer func() {
		err = c.Quit()
	}()

	for _, message := range messages {
		if err = m.Send(c, message); err != nil {
			return
		}
	}

	return
}

// initialize the smtp client
func (m *Mailer) Transport(addr string, a smtp.Auth) (*smtp.Client, error) {
	c, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}
	if err := c.Hello(m.Address); err != nil {
		return nil, err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err = c.StartTLS(nil); err != nil {
			return nil, err
		}
	}
	if a != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(a); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

// send message through the client
func (m *Mailer) Send(c *smtp.Client, message *Message) (err error) {

	data, err := message.RenderData()
	if err != nil {
		return
	}

	if err = c.Reset(); err != nil {
		return
	}

	if err = c.Mail(message.From); err != nil {
		return
	}

	for _, addr := range message.To {
		if err = c.Rcpt(addr); err != nil {
			return
		}
	}
	w, err := c.Data()
	if err != nil {
		return
	}
	defer func() {
		err = w.Close()
	}()

	_, err = w.Write([]byte(data))
	if err != nil {
		return
	}
	return
}
