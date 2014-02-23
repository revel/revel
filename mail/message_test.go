package mail

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderRecipient(t *testing.T) {
	message := &Message{From: "foo@bar.com", To: []string{"bar@foo.com", "abc@test.com"}, ReplyTo: "none@arkxu.com",
		Subject: "from message1, single connection", PlainBody: bytes.NewBufferString("<h2>你好 from message1, should show in plain text</h2>")}

	var b bytes.Buffer
	message.writeRecipient(&b)
	recipient := b.String()

	if !strings.Contains(recipient, "From: foo@bar.com") {
		t.Error("Recipient should contain From")
	}

	if !strings.Contains(recipient, "Reply-To: none@arkxu.com") {
		t.Error("Recipient should contain Reply-To")
	}

	if !strings.Contains(recipient, "To: bar@foo.com, abc@test.com") {
		t.Error("Recipient should contain To")
	}

	if !strings.Contains(recipient, "Subject: from message1") {
		t.Error("Recipient should contain Subject")
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

	if strings.Contains(recipient, "这个是第11封from message1, single connection") {
		t.Error("Subject should be encoded")
	}
}

func TestRenderPlainAndHtmlText(t *testing.T) {
	plainBody := "你好 from message1, should show in plain text"
	htmlBody := "<h2>你好 from message1, should show in html text</h2>"
	testDate, _ := time.Parse("2006-Jan-02", "2014-Feb-23")
	message := &Message{
		From:      "foo@bar.com",
		To:        []string{"bar@foo.com", "abc@test.com"},
		Subject:   "这个是第11封from message1, single connection",
		PlainBody: bytes.NewBufferString(plainBody),
		HtmlBody:  bytes.NewBufferString(htmlBody),
		Date:      testDate,
		MessageId: "message-id@bar.com",
	}

	b, _ := message.RenderData()
	recipient := string(b)

	if !strings.Contains(recipient, plainBody) {
		t.Errorf("should have plain body: %s \n", plainBody)
	}

	if !strings.Contains(recipient, htmlBody) {
		t.Errorf("should have html body: %s \n", htmlBody)
	}

	if !strings.Contains(recipient, "Date: Sun, 23 Feb 2014 00:00:00 GMT") {
		t.Error("Message should have the Date header set")
	}

	if !strings.Contains(recipient, "Message-Id: <message-id@bar.com>") {
		t.Error("Message should have the Message-Id header set")
	}
}
