package mail

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/revel/revel"
)

type faker struct {
	io.ReadWriter
}

func (f faker) Close() error                     { return nil }
func (f faker) LocalAddr() net.Addr              { return nil }
func (f faker) RemoteAddr() net.Addr             { return nil }
func (f faker) SetDeadline(time.Time) error      { return nil }
func (f faker) SetReadDeadline(time.Time) error  { return nil }
func (f faker) SetWriteDeadline(time.Time) error { return nil }

// Send the given email messages using this Mailer.
func (m *Mailer) SendTestMessage(basicServer string, messages ...*Message) (actualcmds string, err error) {
	if m.Auth == nil {
		m.Auth = smtp.PlainAuth(m.UserName, m.UserName, m.Password, m.Server)
	}

	server := strings.Join(strings.Split(basicServer, "\n"), "\r\n")
	var cmdbuf bytes.Buffer
	bcmdbuf := bufio.NewWriter(&cmdbuf)
	var fake faker
	fake.ReadWriter = bufio.NewReadWriter(bufio.NewReader(strings.NewReader(server)), bcmdbuf)

	defer func() {
		bcmdbuf.Flush()
		actualcmds = cmdbuf.String()
	}()

	c, err := smtp.NewClient(fake, "fake.host")
	if err != nil {
		return
	}
	defer c.Quit()

	for _, message := range messages {
		m.fillDefault(message)
		if err = Send(c, message); err != nil {
			return
		}
	}

	return
}

func TestSend(t *testing.T) {

	server := `220 hello world
502 EH?
250 mx.google.com at your service
250 Ok resetting state
250 Sender ok
250 Receiver ok
250 Receiver ok
250 Receiver ok
354 Go ahead
250 Data ok
250 Ok resetting state
250 Sender ok
250 Receiver ok
250 Receiver ok
250 Receiver ok
250 Receiver ok
354 Go ahead
250 Data ok
221 Goodbye
`

	basicClient := `EHLO localhost
HELO localhost
RSET
MAIL FROM:<foo@bar.com>
RCPT TO:<bar@foo.com>
RCPT TO:<cc1@test.com>
RCPT TO:<cc2@test.com>
DATA
From: foo@bar.com 
To: bar@foo.com 
Cc: cc1@test.com, cc2@test.com 
Subject: from message1, single connection 
Message-Id: <message-id1@bar.com>
Date: Sun, 23 Feb 2014 00:00:00 GMT
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8";
Content-Transfer-Encoding: 8bit

<h2>你好 from message1, should show in plain text</h2>
.
RSET
MAIL FROM:<abc@test.com>
RCPT TO:<def@test.com>
RCPT TO:<nonoo@test.com>
RCPT TO:<bcc1@test.com>
RCPT TO:<bcc2@test.com>
DATA
From: abc@test.com 
To: def@test.com, nonoo@test.com 
Subject: =?UTF-8?B?6L+Z5Liq5piv56ysMuWwgWZyb20gbWVzc2FnZTIsIHNpbmdsZSBjb25uZWN0aW9u?= 
Message-Id: <message-id2@test.com>
Date: Sun, 23 Feb 2014 00:00:00 GMT
MIME-Version: 1.0
Content-Type: text/html; charset="UTF-8";
Content-Transfer-Encoding: 8bit

<h2>您好 from message2</h2>
.
QUIT
`
	testDate, _ := time.Parse("2006-Jan-02", "2014-Feb-23")
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "foo@bar.com", Password: "xxx"}

	message1 := &Message{From: "foo@bar.com", To: []string{"bar@foo.com"}, Cc: []string{"cc1@test.com", "cc2@test.com"},
		Subject: "from message1, single connection", PlainBody: bytes.NewBufferString("<h2>你好 from message1, should show in plain text</h2>"),
		Date: testDate, MessageId: "message-id1@bar.com"}

	message2 := &Message{From: "abc@test.com", To: []string{"def@test.com", "nonoo@test.com"}, Bcc: []string{"bcc1@test.com", "bcc2@test.com"},
		Subject: "这个是第2封from message2, single connection", HtmlBody: bytes.NewBufferString("<h2>您好 from message2</h2>"),
		Date: testDate, MessageId: "message-id2@test.com"}

	actualcmds, err := mailer.SendTestMessage(server, message1, message2)
	if err != nil {
		fmt.Println(err)
	}

	client := strings.Join(strings.Split(basicClient, "\n"), "\r\n")
	if client != actualcmds {
		t.Errorf("Got:\n%s\nExpected:\n%s", actualcmds, client)
	}
}

func TestWithDefaultSender(t *testing.T) {
	server := `220 hello world
502 EH?
250 mx.google.com at your service
250 Ok resetting state
250 Sender ok
250 Receiver ok
354 Go ahead
250 Data ok
250 Ok resetting state
250 Sender ok
250 Receiver ok
354 Go ahead
250 Data ok
250 Ok resetting state
250 Sender ok
250 Receiver ok
354 Go ahead
250 Data ok
221 Goodbye
`

	basicClient := `EHLO localhost
HELO localhost
RSET
MAIL FROM:<sender@test.com>
RCPT TO:<to1@test.com>
DATA
From: sender@test.com 
Reply-To: reply@test.com 
To: to1@test.com 
Subject: =?UTF-8?B?5oiR55qE56ysM+S4qg==?= 
Message-Id: <message-id1@test.com>
Date: Sun, 23 Feb 2014 00:00:00 GMT
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8";
Content-Transfer-Encoding: 8bit

这个不是html的
.
RSET
MAIL FROM:<sender@test.com>
RCPT TO:<to2@test.com>
DATA
From: sender@test.com 
Reply-To: reply@test.com 
To: to2@test.com 
Subject: =?UTF-8?B?5oiR55qE56ysNOS4qg==?= 
Message-Id: <message-id2@test.com>
Date: Sun, 23 Feb 2014 00:00:00 GMT
MIME-Version: 1.0
Content-Type: text/html; charset="UTF-8";
Content-Transfer-Encoding: 8bit

<h1>这个是html的</h1>
.
RSET
MAIL FROM:<sender@test.com>
RCPT TO:<to3@test.com>
DATA
From: sender@test.com 
Reply-To: reply@test.com 
To: to3@test.com 
Subject: =?UTF-8?B?5oiR55qE56ysNeS4qg==?= 
Message-Id: <message-id3@test.com>
Date: Sun, 23 Feb 2014 00:00:00 GMT
MIME-Version: 1.0
`
	testDate, _ := time.Parse("2006-Jan-02", "2014-Feb-23")
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "foobar@test.com", Password: "xxx", Host: "arkxu.com"}

	mailer.Sender = &Sender{From: "sender@test.com", ReplyTo: "reply@test.com"}

	message1 := NewTextMessage([]string{"to1@test.com"}, "我的第3个", "这个不是html的")
	message1.Date = testDate
	message1.MessageId = "message-id1@test.com"

	message2 := NewHtmlMessage([]string{"to2@test.com"}, "我的第4个", "<h1>这个是html的</h1>")
	message2.Date = testDate
	message2.MessageId = "message-id2@test.com"

	message3 := NewTextAndHtmlMessage([]string{"to3@test.com"}, "我的第5个", "这个不是html的", "<h1>这个是html的, 同时也有plain text 版本</h1>")
	message3.Date = testDate
	message3.MessageId = "message-id3@test.com"

	actualcmds, err := mailer.SendTestMessage(server, []*Message{message1, message2, message3}...)
	if err != nil {
		fmt.Println(err)
	}

	client := strings.Join(strings.Split(basicClient, "\n"), "\r\n")
	if !strings.Contains(actualcmds, client) {
		t.Errorf("Got:\n%s\nSubstring:\n%s", actualcmds, client)
	}
}

func TestSendFromTemplate(t *testing.T) {
	server := `220 hello world
502 EH?
250 mx.google.com at your service
250 Ok resetting state
250 Sender ok
250 Receiver ok
250 Receiver ok
354 Go ahead
250 Data ok
221 Goodbye
`

	snippet1 := `Hello 世界

Welcome Ark, please click the link:
http://www.arkxu.com
`

	snippet2 := `<h1>Hello 世界</h1>

<p>Welcome Ark, please click the link:</p>
<a href="http://www.arkxu.com">http://www.arkxu.com</a>
`
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "foo@bar.com", Password: "xxx"}
	mailer.Sender = &Sender{From: "sender@foo.com"}

	// reset the revel template loader for testing purpose
	viewPath, _ := os.Getwd()
	revel.MainTemplateLoader = revel.NewTemplateLoader([]string{viewPath})
	revel.MainTemplateLoader.Refresh()

	// arguments used for template rendering
	var args = make(map[string]interface{})
	args["world"] = "世界"
	args["user"] = struct {
		Name string
		Link string
	}{
		"Ark",
		"http://www.arkxu.com",
	}

	message := &Message{To: []string{"to1@test1.com"}, Subject: "from template 6", Cc: []string{"to2@gmail.com"}}
	err := message.RenderTemplate("testdata/testTemplate", args)
	if err != nil {
		fmt.Println(err)
	}

	actualcmds, err := mailer.SendTestMessage(server, message)
	if err != nil {
		fmt.Println(err)
	}

	if strings.Contains(actualcmds, snippet1) {
		t.Errorf("it should contains %s\n", snippet1)
	}

	if strings.Contains(actualcmds, snippet2) {
		t.Errorf("it should contains %s\n", snippet2)
	}

}
