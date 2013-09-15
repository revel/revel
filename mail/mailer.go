package mail

import (
	"bytes"
	"github.com/robfig/revel"
	"net/smtp"
)

type Mailer struct {
	Address       string
	Port          int
	UserName      string
	Password      string
	Auth          smtp.Auth      // This is optional, only used if Authentication is not plain
	DefaultSender *DefaultSender // This is optional, only used if the From/ReplyTo does not specified in the message
}

type DefaultSender struct {
	From    string
	ReplyTo string
}

// This is convinient method to send single email with either plain text or html body
func (m *Mailer) SendMail(to []string, subject string, body string, html bool) error {
	message := &Message{To: to, Subject: subject}

	if html {
		message.HtmlBody = body
	} else {
		message.PlainBody = body
	}

	return m.SendMails([]*Message{message})
}

// This is the convinient method to send single email rendered from a view template with dynamic data
func (m *Mailer) SendFromTemplate(templatePath string, to []string, subject string, args map[string]interface{}) error {
	message := &Message{To: to, Subject: subject}

	htmlTempateFile := templatePath + ".html"
	txtTempateFile := templatePath + ".txt"

	message.HtmlBody = m.renderViewTemplate(htmlTempateFile, args)
	message.PlainBody = m.renderViewTemplate(txtTempateFile, args)

	return m.SendMails([]*Message{message})
}

// send multiple emails in a single connection
func (m *Mailer) SendMails(messages []*Message) (err error) {

	if m.Auth == nil {
		m.Auth = smtp.PlainAuth(m.UserName, m.UserName, m.Password, m.Address)
	}

	c, err := Transport(m.Address, m.Port, m.Auth)
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
	if m.DefaultSender == nil {
		return
	}
	if message.From == "" {
		message.From = m.DefaultSender.From
	}

	if message.ReplyTo == "" {
		message.ReplyTo = m.DefaultSender.ReplyTo
	}
}
