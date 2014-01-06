package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
)

// Transport initialize the smtp client
func Transport(address string, port int, host string, a smtp.Auth) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", address, port)

	var conn net.Conn
	conn, err := tls.Dial("tcp", addr, nil) // some smtp servers require TLS handshake
	if err != nil {
		conn, err = net.Dial("tcp", addr) // fall back
		if err != nil {
			return nil, err
		}
	}

	c, err := smtp.NewClient(conn, address)
	if err != nil {
		return nil, err
	}

	if host != "" {
		if err := c.Hello(host); err != nil {
			return nil, err
		}
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{
			InsecureSkipVerify: true,
		}

		if err = c.StartTLS(config); err != nil {
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

	if err = addRcpt(c, message.To); err != nil {
		return
	}

	if err = addRcpt(c, message.Cc); err != nil {
		return
	}

	if err = addRcpt(c, message.Bcc); err != nil {
		return
	}

	w, err := c.Data()
	if err != nil {
		return
	}
	defer w.Close()

	_, err = w.Write(data)
	return
}

func addRcpt(c *smtp.Client, address []string) error {
	for _, addr := range address {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	return nil
}
