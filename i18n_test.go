package rev

import (
	"testing"
)

const (
	testDataPath string = "testdata/i18n"
)

func TestI8nLoadMessages(t *testing.T) {
	loadMessages(testDataPath)

	// Assert that we have the expected number of locales
	if len(MessageLocales()) != 2 {
		t.Fatalf("Expected messages to contain no more or less than 2 locales, instead there are %d locales", len(MessageLocales()))
	}
}

func TestI8nGetMessage(t *testing.T) {
	loadMessages(testDataPath)

	// Assert that we can get a message and we get the expected return value
	if message := GetMessage("nl", "greeting"); message != "Hallo" {
		t.Error("Message 'greeting' for locale 'nl' does not have the expected value")
	}
	if message := GetMessage("en", "greeting"); message != "Hello" {
		t.Error("Message 'greeting' for locale 'en' does not have the expected value")
	}
	if message := GetMessage("en", "folded"); message != "Greeting is 'Hello'" {
		t.Error("Unexpected unfolded message: ", message)
	}

	// TODO: re-enable this test once files merging has been implemented
	/*if message := GetMessage("en", "greeting2"); message != "Yo!" {
		t.Error("Message 'greeting2' for locale 'en' does not have the expected value")
	}*/

	// Assert that we get the expected return value for a locale that doesn't exist
	if message := GetMessage("unknown locale", "message"); message != "??? unknown locale ???" {
		t.Error("Locale 'doesn't exist' is not supposed to exist")
	}

	// Assert that we get the expected return value for a message that doesn't exist
	if message := GetMessage("nl", "unknown message"); message != "??? unknown message ???" {
		t.Error("Message 'doesn't exist' is not supposed to exist")
	}
}

func BenchmarkI8nGetMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetMessage("nl", "greeting")
	}
}
