package mail

import (
	"fmt"
	"testing"
)

func TestSend(t *testing.T) {
	mailer := &Mailer{Address: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxxxx"}

	message1 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第11封from message1, single connection", PlainBody: "<h2>你好 from message1, should show in plain text</h2>"}

	message2 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第12封from message2, single connection", HtmlBody: "<h2>您好 from message2</h2>"}

	err := mailer.SendMails([]*Message{message1, message2})
	if err != nil {
		fmt.Println(err)
	}

}

func TestWithDefaultSender(t *testing.T) {
	mailer := &Mailer{Address: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxxxx"}

	mailer.DefaultSender = &DefaultSender{From: "fangzhou@arkxu.com"}

	err := mailer.SendMail([]string{"fangzhou@arkxu.com"}, "我的第13个", "这个不是html的", false)
	if err != nil {
		fmt.Println(err)
	}

	err = mailer.SendMail([]string{"fangzhou@arkxu.com"}, "我的第14个", "<h1>这个是html的</h1>", true)
	if err != nil {
		fmt.Println(err)
	}
}
