package mail

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/robfig/revel"
)

func TestSend(t *testing.T) {
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxx"}

	message1 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第1封from message1, single connection", PlainBody: bytes.NewBufferString("<h2>你好 from message1, should show in plain text</h2>")}

	message2 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第2封from message2, single connection", HtmlBody: bytes.NewBufferString("<h2>您好 from message2</h2>")}

	err := mailer.SendMessage(message1, message2)
	if err != nil {
		fmt.Println(err)
	}

}

func TestWithDefaultSender(t *testing.T) {
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxx", Host: "arkxu.com"}

	mailer.Sender = &Sender{From: "fangzhou@arkxu.com", ReplyTo: "fangzhou@arkxu.com"}

	message1 := NewTextMessage([]string{"fangzhou@arkxu.com"}, "我的第3个", "这个不是html的")
	message2 := NewHtmlMessage([]string{"fangzhou@arkxu.com"}, "我的第4个", "<h1>这个是html的</h1>")
	message3 := NewTextAndHtmlMessage([]string{"fangzhou@arkxu.com"}, "我的第5个", "这个不是html的", "<h1>这个是html的, 同时也有plain text 版本</h1>")
	err := mailer.SendMessage([]*Message{message1, message2, message3}...)
	if err != nil {
		fmt.Println(err)
	}
}

func TestSendFromTemplate(t *testing.T) {
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxx"}
	mailer.Sender = &Sender{From: "fangzhou@arkxu.com"}

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

	message := &Message{To: []string{"fangzhou@arkxu.com"}, Subject: "from template 6", Cc: []string{"ark.work@gmail.com"}}
	err := message.RenderTemplate("testdata/testTemplate", args)
	if err != nil {
		fmt.Println(err)
	}

	mailer.SendMessage(message)
	if err != nil {
		fmt.Println(err)
	}
}
