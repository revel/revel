package mail

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
)

type Message struct {
	From      string
	ReplyTo   string //implement later
	To        []string
	BCC       string //implement later
	CC        string //implement later
	Subject   string
	PlainBody string
	HtmlBody  string
}

// RenderData render the whole email body
func (m *Message) RenderData() (data string, err error) {
	if m.HtmlBody == "" && m.PlainBody == "" {
		err = errors.New("HtmlBody and PlainBody can not both be blank")
		return
	}
	if m.HtmlBody == "" {
		data = m.renderSingleContentType("text/plain", m.PlainBody)
	} else if m.PlainBody == "" {
		data = m.renderSingleContentType("text/html", m.HtmlBody)
	} else {
		data = m.renderPlainAndHtmlText()
	}

	return
}

func (m *Message) renderSingleContentType(contentType string, bodyText string) string {
	body :=
		`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: %s; charset="UTF-8";
Content-Transfer-Encoding: 8bit

%s

`

	data := fmt.Sprintf(body, m.From, strings.Join(m.To, ","), m.Subject, contentType, bodyText)
	return data
}

func (m *Message) renderPlainAndHtmlText() string {
	writer := multipart.NewWriter(bytes.NewBufferString(""))
	body :=
		`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: multipart/alternative; charset="UTF-8"; boundary="%s"

--%s
Content-Type: text/plain; charset=UTF-8
Content-Transfer-Encoding: 8bit

%s

--%s
Content-Type: text/html; charset=UTF-8
Content-Transfer-Encoding: 8bit

%s

--%s--

`
	data := fmt.Sprintf(body, m.From, strings.Join(m.To, ","), m.Subject, writer.Boundary(),
		writer.Boundary(), m.PlainBody, writer.Boundary(), m.HtmlBody, writer.Boundary())
	return data
}
