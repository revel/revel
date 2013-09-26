package mail

import (
	"fmt"
	"github.com/robfig/revel"
	"os"
	"testing"
)

func TestSend(t *testing.T) {
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xx"}

	message1 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第11封from message1, single connection", PlainBody: "<h2>你好 from message1, should show in plain text</h2>"}

	message2 := &Message{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"},
		Subject: "这个是第12封from message2, single connection", HtmlBody: "<h2>您好 from message2</h2>"}

	err := mailer.Send(message1, message2)
	if err != nil {
		fmt.Println(err)
	}

}

func TestWithDefaultSender(t *testing.T) {
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xx", Host: "arkxu.com"}

	mailer.Default = &Default{From: "fangzhou@arkxu.com", To: []string{"fangzhou@arkxu.com"}}

	message1 := &Message{Subject: "我的第13个", PlainBody: "这个不是html的"}
	message2 := &Message{Subject: "我的第14个", HtmlBody: "<h1>这个是html的</h1>"}
	err := mailer.SendMessages([]*Message{message1, message2})
	if err != nil {
		fmt.Println(err)
	}
}

func TestSendFromTemplate(t *testing.T) {
	mailer := &Mailer{Server: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xx"}
	mailer.Default = &Default{From: "fangzhou@arkxu.com"}

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

	message := &Message{To: []string{"fangzhou@arkxu.com"}, Subject: "from template 4"}
	err := mailer.RenderTemplate(message, "testdata/testTemplate", args)
	if err != nil {
		fmt.Println(err)
	}

	mailer.Send(message)
	if err != nil {
		fmt.Println(err)
	}
}
