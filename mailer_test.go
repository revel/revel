package revel

import (
  "testing"
)

type testMailer struct {
  Mailer
}

func (m testMailer) Testmail() error {
  return m.Send(H{})
}
// Test that the booking app can be successfully run for a test.
func TestMailer(t *testing.T) {
	startFakeBookingApp()
  err := testMailer{}.Testmail()
  if err != nil {
    t.Errorf("Mailer err %s", err)
  }
}
