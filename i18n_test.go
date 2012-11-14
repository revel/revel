package rev

import (
	"io/ioutil"
	"log"
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

func TestI8nMessage(t *testing.T) {
	loadMessages(testDataPath)

	// Assert that we can get a message and we get the expected return value
	if message := Message("nl", "greeting"); message != "Hallo" {
		t.Errorf("Message 'greeting' for locale 'nl' (%s) does not have the expected value", message)
	}
	if message := Message("en", "greeting"); message != "Hello" {
		t.Errorf("Message 'greeting' for locale 'en' (%s) does not have the expected value", message)
	}
	if message := Message("en", "folded"); message != "Greeting is 'Hello'" {
		t.Error("Unexpected unfolded message: ", message)
	}
	if message := Message("en", "arguments.string", "Vincent Hanna"); message != "My name is Vincent Hanna" {
		t.Errorf("Message 'arguments.string' for locale 'en' (%s) does not have the expected value", message)
	}
	if message := Message("en", "arguments.hex", 1234567, 1234567); message != "The number 1234567 in hexadecimal notation would be 12d687" {
		t.Errorf("Message 'arguments.hex' for locale 'en' (%s) does not have the expected value", message)
	}
	if message := Message("en", "folded.arguments", 12345); message != "Rob is 12345 years old" {
		t.Errorf("Message 'folded.arguments' for locale 'en' (%s) does not have the expected value", message)
	}

	// TODO: re-enable this test once files merging has been implemented
	/*if message := Message("en", "greeting2"); message != "Yo!" {
		t.Error("Message 'greeting2' for locale 'en' does not have the expected value")
	}*/

	// Assert that we get the expected return value for a locale that doesn't exist
	if message := Message("unknown locale", "message"); message != "??? unknown locale ???" {
		t.Error("Locale 'unknown locale' is not supposed to exist")
	}
	// Assert that we get the expected return value for a message that doesn't exist
	if message := Message("nl", "unknown message"); message != "??? unknown message ???" {
		t.Error("Message 'unknown message' is not supposed to exist")
	}
}

func BenchmarkI8nMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Message("nl", "greeting")
	}
}

func BenchmarkI8nMessageWithArguments(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TRACE = log.New(ioutil.Discard, "", 0)
		Message("en", "arguments.string", "Vincent Hanna")
	}
}

func BenchmarkI8nMessageWithFoldingAndArguments(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TRACE = log.New(ioutil.Discard, "", 0)
		Message("en", "folded.arguments", 12345)
	}
}
