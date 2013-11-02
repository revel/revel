package mail

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRenderRecipient(t *testing.T) {
	message := &Message{From: "foo@bar.com", To: []string{"bar@foo.com", "abc@test.com"}, ReplyTo: "none@arkxu.com",
		Subject: "这个是第11封from message1, single connection", PlainBody: bytes.NewBufferString("<h2>你好 from message1, should show in plain text</h2>")}

	var b bytes.Buffer
	message.writeRecipient(&b)
	recipient := b.String()
	fmt.Println(recipient)
	if !strings.Contains(recipient, "From: foo@bar.com") {
		t.Error("Recipient should contains From")
	}

	if !strings.Contains(recipient, "Reply-To: none@arkxu.com") {
		t.Error("Recipient should contains Reply-To")
	}

	if !strings.Contains(recipient, "To: bar@foo.com, abc@test.com") {
		t.Error("Recipient should contains To")
	}

	if !strings.Contains(recipient, "Subject: 这个是第11封from message1") {
		t.Error("Recipient should contains Subject")
	}
}

func TestRenderRecipientNoReply(t *testing.T) {
	message := &Message{From: "foo@bar.com", To: []string{"bar@foo.com", "abc@test.com"},
		Subject: "这个是第11封from message1, single connection", PlainBody: bytes.NewBufferString("<h2>你好 from message1, should show in plain text</h2>")}

	var b bytes.Buffer
	message.writeRecipient(&b)
	recipient := b.String()

	if strings.Contains(recipient, "Reply-To") {
		t.Error("Recipient should not contains Reply-To")
	}
}

func TestRenderPlainAndHtmlText(t *testing.T) {
	message := &Message{From: "foo@bar.com", To: []string{"bar@foo.com", "abc@test.com"},
		Subject: "这个是第11封from message1, single connection", PlainBody: bytes.NewBufferString("<h2>你好 from message1, should show in plain text</h2>")}

	b, _ := message.RenderData()
	fmt.Println(string(b))
}
