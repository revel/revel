package mail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/revel/revel"
)

var NewLine string = "\r\n"

type Message struct {
	From      string
	ReplyTo   string
	To        []string
	Cc        []string
	Bcc       []string
	Subject   string
	PlainBody *bytes.Buffer
	HtmlBody  *bytes.Buffer

	Date      time.Time
	MessageId string
}

// NewTextMessage create a plain text message
func NewTextMessage(to []string, subject string, body string) *Message {
	return &Message{To: to, Subject: subject, PlainBody: bytes.NewBufferString(body)}
}

// NewHtmlMessage create a html message
func NewHtmlMessage(to []string, subject string, body string) *Message {
	return &Message{To: to, Subject: subject, HtmlBody: bytes.NewBufferString(body)}
}

// NewTextAndHtmlMessage create a message contains both plain text and html message
func NewTextAndHtmlMessage(to []string, subject string, plainBody string, htmlBody string) *Message {
	return &Message{To: to, Subject: subject, PlainBody: bytes.NewBufferString(plainBody), HtmlBody: bytes.NewBufferString(htmlBody)}
}

// RenderData render the whole email body
func (m *Message) RenderData() (data []byte, err error) {
	if m.HtmlBody == nil && m.PlainBody == nil {
		err = errors.New("HtmlBody and PlainBody can not both be blank")
		return
	}

	switch {
	case m.HtmlBody == nil, m.HtmlBody.Len() == 0:
		data, err = m.renderSingleContentType("text/plain", m.PlainBody)
	case m.PlainBody == nil, m.PlainBody.Len() == 0:
		data, err = m.renderSingleContentType("text/html", m.HtmlBody)
	default:
		data, err = m.renderPlainAndHtmlText()
	}

	return
}

// RenderTemplate renders the message body from the template and input parameters
func (m *Message) RenderTemplate(templatePath string, args map[string]interface{}) error {
	m.HtmlBody = m.renderViewTemplate(templatePath+".html", args)
	m.PlainBody = m.renderViewTemplate(templatePath+".txt", args)

	if m.HtmlBody == nil && m.PlainBody == nil {
		return errors.New("Both HTML body and Plain body are blank.")
	}
	return nil
}

func (m *Message) renderViewTemplate(templateFilePath string, args map[string]interface{}) *bytes.Buffer {
	// Get the Template.
	template, err := revel.MainTemplateLoader.Template(templateFilePath)
	if err != nil {
		revel.ERROR.Print(err)
		return nil
	}

	var b bytes.Buffer

	err = template.Render(&b, args)
	if err != nil {
		revel.ERROR.Print(err)
		return nil
	}

	return &b
}

func (m *Message) renderSingleContentType(contentType string, bodyText *bytes.Buffer) ([]byte, error) {
	var b bytes.Buffer
	m.writeRecipient(&b)

	if err := m.writeMessageId(&b); err != nil {
		return nil, err
	}

	if err := m.writeDate(&b); err != nil {
		return nil, err
	}

	if err := m.writeMIME(&b); err != nil {
		return nil, err
	}

	if err := m.writeContentType(&b, contentType); err != nil {
		return nil, err
	}

	if _, err := b.Write(bodyText.Bytes()); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (m *Message) renderPlainAndHtmlText() ([]byte, error) {
	writer := multipart.NewWriter(bytes.NewBufferString(""))
	var b bytes.Buffer
	m.writeRecipient(&b)

	if err := m.writeMessageId(&b); err != nil {
		return nil, err
	}

	if err := m.writeDate(&b); err != nil {
		return nil, err
	}

	if err := m.writeMIME(&b); err != nil {
		return nil, err
	}

	if err := m.writeMultipartStart(&b, writer); err != nil {
		return nil, err
	}

	if err := m.writeContentTypeWithBoundary(&b, writer, "text/plain"); err != nil {
		return nil, err
	}

	if _, err := b.Write(m.PlainBody.Bytes()); err != nil {
		return nil, err
	}

	if err := m.writeContentTypeWithBoundary(&b, writer, "text/html"); err != nil {
		return nil, err
	}

	if _, err := b.Write(m.HtmlBody.Bytes()); err != nil {
		return nil, err
	}

	if err := m.writeMultipartEnd(&b, writer); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (m *Message) writeRecipient(b *bytes.Buffer) {
	if m.From != "" {
		fmt.Fprintf(b, "From: %s %s", m.From, NewLine)
	}

	if m.ReplyTo != "" {
		fmt.Fprintf(b, "Reply-To: %s %s", m.ReplyTo, NewLine)
	}

	if len(m.To) > 0 {
		fmt.Fprintf(b, "To: %s %s", strings.Join(m.To, ", "), NewLine)
	}

	if len(m.Cc) > 0 {
		fmt.Fprintf(b, "Cc: %s %s", strings.Join(m.Cc, ", "), NewLine)
	}

	if m.Subject != "" {
		subjectBytes := []byte(m.Subject)
		if len(subjectBytes) == utf8.RuneCount(subjectBytes) {
			fmt.Fprintf(b, "Subject: %s %s", m.Subject, NewLine)
		} else {
			fmt.Fprintf(b, "Subject: =?UTF-8?B?%s?= %s", base64.StdEncoding.EncodeToString(subjectBytes), NewLine)
		}
	}
}

func (m *Message) writeDate(b *bytes.Buffer) error {
	if m.Date.IsZero() {
		m.Date = time.Now()
	}

	_, err := b.WriteString("Date: " + m.Date.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT") + "\n")
	return err
}

func (m *Message) writeMessageId(b *bytes.Buffer) error {
	if m.MessageId == "" {
		return nil
	}

	_, err := b.WriteString("Message-Id: <" + m.MessageId + ">\n")
	return err
}

func (m *Message) writeMIME(b *bytes.Buffer) error {
	_, err := b.WriteString("MIME-Version: 1.0")
	return err
}

func (m *Message) writeContentType(b *bytes.Buffer, contentType string) error {
	contentTypeFormat := `
Content-Type: %s; charset="UTF-8";
Content-Transfer-Encoding: 8bit

`
	_, err := b.WriteString(fmt.Sprintf(contentTypeFormat, contentType))
	return err
}

func (m *Message) writeMultipartStart(b *bytes.Buffer, writer *multipart.Writer) error {
	multipart := `
Content-Type: multipart/alternative; charset="UTF-8"; boundary="%s"
`
	_, err := b.WriteString(fmt.Sprintf(multipart, writer.Boundary()))
	return err
}

func (m *Message) writeMultipartEnd(b *bytes.Buffer, writer *multipart.Writer) error {
	multipart := `
--%s--

`
	_, err := b.WriteString(fmt.Sprintf(multipart, writer.Boundary()))
	return err
}

func (m *Message) writeContentTypeWithBoundary(b *bytes.Buffer, writer *multipart.Writer, contentType string) error {
	contentTypeFormat := `

--%s
Content-Type: %s; charset=UTF-8;
Content-Transfer-Encoding: 8bit

`
	_, err := b.WriteString(fmt.Sprintf(contentTypeFormat, writer.Boundary(), contentType))
	return err
}
