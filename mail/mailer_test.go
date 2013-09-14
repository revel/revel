package mail

import (
	"fmt"
	"github.com/robfig/revel"
	"os"
	"testing"
)

func TestSend(t *testing.T) {
	mailer := &Mailer{Address: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxx"}

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
	mailer := &Mailer{Address: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxx"}

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

func TestSendFromTemplate(t *testing.T) {
	mailer := &Mailer{Address: "smtp.gmail.com", Port: 587, UserName: "fangzhou@arkxu.com", Password: "xxx"}
	mailer.DefaultSender = &DefaultSender{From: "fangzhou@arkxu.com"}

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

	err := mailer.SendFromTemplate("testTemplate.html", []string{"fangzhou@arkxu.com"}, "from template 3", true, args)
	if err != nil {
		fmt.Println(err)
	}
}
