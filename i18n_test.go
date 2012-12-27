package rev

import (
	"io/ioutil"
	"log"
	"testing"
)

const (
	testDataPath string = "testdata/i18n"
)

func TestI18nLoadMessages(t *testing.T) {
	loadMessages(testDataPath)

	// Assert that we have the expected number of languages
	if len(MessageLanguages()) != 2 {
		t.Fatalf("Expected messages to contain no more or less than 2 languages, instead there are %d languages", len(MessageLanguages()))
	}
}

func TestI18nMessage(t *testing.T) {
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
	if message := Message("en-AU", "greeting"); message != "G'day" {
		t.Errorf("Message 'greeting' for locale 'en-AU' (%s) does not have the expected value", message)
	}
	if message := Message("en-AU", "only_exists_in_default"); message != "Default" {
		t.Errorf("Message 'only_exists_in_default' for locale 'en-AU' (%s) does not have the expected value", message)
	}

	// Assert that message merging works
	if message := Message("en", "greeting2"); message != "Yo!" {
		t.Errorf("Message 'greeting2' for locale 'en' (%s) does not have the expected value", message)
	}

	// Assert that we get the expected return value for a locale that doesn't exist
	if message := Message("unknown locale", "message"); message != "??? message ???" {
		t.Error("Locale 'unknown locale' is not supposed to exist")
	}
	// Assert that we get the expected return value for a message that doesn't exist
	if message := Message("nl", "unknown message"); message != "??? unknown message ???" {
		t.Error("Message 'unknown message' is not supposed to exist")
	}
}

func TestI18nMessageWithDefaultLocale(t *testing.T) {
	loadMessages(testDataPath)
	stubI18nConfig(t)

	if message := Message("doesn't exist", "greeting"); message != "Hello" {
		t.Errorf("Expected message '%s' for unknown locale to be default '%s' but was '%s'", "greeting", "Hello", message)
	}
	if message := Message("doesn't exist", "unknown message"); message != "??? unknown message ???" {
		t.Error("Message 'unknown message' is not supposed to exist in the default language")
	}
}

func BenchmarkI18nLoadMessages(b *testing.B) {
	excludeFromTimer(b, func() { TRACE = log.New(ioutil.Discard, "", 0) })

	for i := 0; i < b.N; i++ {
		loadMessages(testDataPath)
	}
}

func BenchmarkI18nMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Message("nl", "greeting")
	}
}

func BenchmarkI18nMessageWithArguments(b *testing.B) {
	excludeFromTimer(b, func() { TRACE = log.New(ioutil.Discard, "", 0) })

	for i := 0; i < b.N; i++ {
		Message("en", "arguments.string", "Vincent Hanna")
	}
}

func BenchmarkI18nMessageWithFoldingAndArguments(b *testing.B) {
	excludeFromTimer(b, func() { TRACE = log.New(ioutil.Discard, "", 0) })

	for i := 0; i < b.N; i++ {
		Message("en", "folded.arguments", 12345)
	}
}

// Exclude whatever operations the given function performs from the benchmark timer.
func excludeFromTimer(b *testing.B, f func()) {
	b.StopTimer()
	f()
	b.StartTimer()
}
