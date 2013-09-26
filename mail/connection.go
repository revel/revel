package mail

import (
	"fmt"
	"net/smtp"
)

// Transport initialize the smtp client
func Transport(address string, port int, host string, a smtp.Auth) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", address, port)

	c, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	if host != "" {
		if err := c.Hello(host); err != nil {
			return nil, err
		}
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

// Send send message through the client
func Send(c *smtp.Client, message *Message) (err error) {

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
