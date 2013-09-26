package mail

import (
	"bytes"
	"errors"
	"github.com/robfig/revel"
	"net/smtp"
)

type Mailer struct {
	Server   string
	Port     int
	UserName string
	Password string
	Host     string    // This is optional, only used if you want to tell smtp server your hostname
	Auth     smtp.Auth // This is optional, only used if Authentication is not plain
	Default  *Default  // This is optional, only used if the From/ReplyTo does not specified in the message
}

type Default struct {
	From    string
	ReplyTo string
	To      []string
	BCC     string
	CC      string
}

// Send is convinient way to send several messages which can be listed, or you can choose to use SendMessages
func (m *Mailer) Send(messages ...*Message) error {
	return m.SendMessages(messages)
}

// SendMessages send multiple email in a single connection
func (m *Mailer) SendMessages(messages []*Message) (err error) {

	if m.Auth == nil {
		m.Auth = smtp.PlainAuth(m.UserName, m.UserName, m.Password, m.Server)
	}

	c, err := Transport(m.Server, m.Port, m.Host, m.Auth)
	if err != nil {
		return
	}
	defer func() {
		err = c.Quit()
	}()

	for _, message := range messages {
		m.fillDefault(message)
		if err = Send(c, message); err != nil {
			return
		}
	}

	return
}

// RenderTemplate renders the message body from the template and input parameters, the change is inline the message
func (m *Mailer) RenderTemplate(message *Message, templatePath string, args map[string]interface{}) error {
	htmlTempateFile := templatePath + ".html"
	txtTempateFile := templatePath + ".txt"

	message.HtmlBody = m.renderViewTemplate(htmlTempateFile, args)
	message.PlainBody = m.renderViewTemplate(txtTempateFile, args)

	if message.HtmlBody == "" && message.PlainBody == "" {
		return errors.New("Both HTML body and Plain body are blank.")
	}
	return nil
}

func (m *Mailer) renderViewTemplate(templateFilePath string, args map[string]interface{}) string {
	// Get the Template.
	template, err := revel.MainTemplateLoader.Template(templateFilePath)
	if err != nil {
		return ""
	}

	var b bytes.Buffer

	err = template.Render(&b, args)
	if err != nil {
		return ""
	}

	return b.String()
}

func (m *Mailer) fillDefault(message *Message) {
	if m.Default == nil {
		return
	}
	if message.From == "" {
		message.From = m.Default.From
	}

	if message.ReplyTo == "" {
		message.ReplyTo = m.Default.ReplyTo
	}

	if len(message.To) == 0 {
		message.To = m.Default.To
	}

	if message.BCC == "" {
		message.BCC = m.Default.BCC
	}

	if message.CC == "" {
		message.CC = m.Default.CC
	}
}
