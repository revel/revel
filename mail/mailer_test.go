package mail

import (
	"fmt"
	"testing"
)

func TestSend(t *testing.T) {
	mailer := &Mailer{Address: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxxxxxx", Authentication: "plain"}

	message1 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第1封from message1, single connection", PlainBody: "<h2>你好 from message1, should show in plain text</h2>"}

	message2 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第2封from message2, single connection", HtmlBody: "<h2>您好 from message2</h2>"}

	err := mailer.SendMails([]*Message{message1, message2})
	if err != nil {
		fmt.Println(err)
	}

	err = mailer.SendMail("fangzhou@arkxu.com", []string{"fangzhou@arkxu.com"}, "我的第3个", "这个不是html的", false)
	if err != nil {
		fmt.Println(err)
	}

	err = mailer.SendMail("fangzhou@arkxu.com", []string{"fangzhou@arkxu.com"}, "我的第4个", "<h1>这个是html的</h1>", true)
	if err != nil {
		fmt.Println(err)
	}
}
