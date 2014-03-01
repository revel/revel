package revel

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	errorsMessage   = "validation should not be satisfied with %s\n"
	noErrorsMessage = "validation should be satisfied with %s\n"
)

func TestRequired(t *testing.T) {
	for _, required := range []Required{Required{}, ValidRequired()} {
		// nil
		if required.IsSatisfied(nil) {
			t.Errorf(errorsMessage, "nil data")
		}

		// string
		if !required.IsSatisfied("Testing") {
			t.Errorf(noErrorsMessage, "non-empty string")
		}
		if required.IsSatisfied("") {
			t.Errorf(errorsMessage, "empty string")
		}

		// bool
		if !required.IsSatisfied(true) {
			t.Errorf(noErrorsMessage, "true boolean")
		}
		if required.IsSatisfied(false) {
			t.Errorf(errorsMessage, "false boolean")
		}

		// int
		if !required.IsSatisfied(1) {
			t.Errorf(noErrorsMessage, "positive integer")
		}
		if !required.IsSatisfied(-1) {
			t.Errorf(noErrorsMessage, "negative integer")
		}
		if required.IsSatisfied(0) {
			t.Errorf(errorsMessage, "0 integer")
		}

		// time
		if !required.IsSatisfied(time.Now()) {
			t.Errorf(noErrorsMessage, "current time")
		}
		if required.IsSatisfied(time.Time{}) {
			t.Errorf(errorsMessage, "a zero time")
		}

		// slice
		if !required.IsSatisfied([]string{"Test"}) {
			t.Errorf(noErrorsMessage, "len > 0")
		}
		if required.IsSatisfied([]string{}) {
			t.Errorf(errorsMessage, "a slice len < 1")
		}

		// some other random data type
		if !required.IsSatisfied(func() {}) {
			t.Errorf(noErrorsMessage, "other non-nil data types")
		}
	}
}

func TestMin(t *testing.T) {
	for _, min := range []Min{Min{10}, ValidMin(10)} {
		if !min.IsSatisfied(11) {
			t.Errorf(noErrorsMessage, "val > min")
		}

		if !min.IsSatisfied(10) {
			t.Errorf(noErrorsMessage, "val == min")
		}

		if min.IsSatisfied(9) {
			t.Errorf(noErrorsMessage, "val < min")
		}

		if min.IsSatisfied(true) {
			t.Errorf(errorsMessage, "TypeOf(val) != int")
		}
	}
}

func TestMax(t *testing.T) {
	for _, max := range []Max{Max{10}, ValidMax(10)} {
		if !max.IsSatisfied(9) {
			t.Errorf(noErrorsMessage, "val < max")
		}

		if !max.IsSatisfied(10) {
			t.Errorf(noErrorsMessage, "val == max")
		}

		if max.IsSatisfied(11) {
			t.Errorf(errorsMessage, "val > max")
		}

		if max.IsSatisfied(true) {
			t.Errorf(errorsMessage, "TypeOf(val) != int")
		}
	}
}

func TestRange(t *testing.T) {
	goodValidators := []Range{
		Range{Min{10}, Max{100}},
		ValidRange(10, 100),
	}
	for _, rangeValidator := range goodValidators {
		if !rangeValidator.IsSatisfied(50) {
			t.Errorf(noErrorsMessage, "min <= val <= max")
		}
		if !rangeValidator.IsSatisfied(10) {
			t.Errorf(noErrorsMessage, "val == min")
		}
		if !rangeValidator.IsSatisfied(100) {
			t.Errorf(noErrorsMessage, "val == max")
		}

		if rangeValidator.IsSatisfied(9) {
			t.Errorf(errorsMessage, "val < min")
		}
		if rangeValidator.IsSatisfied(101) {
			t.Errorf(errorsMessage, "val > max")
		}
	}

	goodValidators = []Range{
		Range{Min{10}, Max{10}},
		ValidRange(10, 10),
	}
	for _, rangeValidator := range goodValidators {
		if !rangeValidator.IsSatisfied(10) {
			t.Errorf(noErrorsMessage, "min == val == max")
		}

		if rangeValidator.IsSatisfied(9) {
			t.Errorf(noErrorsMessage, "val < min && val < max && min == max")
		}

		if rangeValidator.IsSatisfied(11) {
			t.Errorf(noErrorsMessage, "val > min && val > max && min == max")
		}
	}

	// these are min/max with values swapped, so the min is the high
	// and max is the low. rangeValidator.IsSatisfied() should ALWAYS
	// result in false since val can never be greater than min and less
	// than max when min > max
	badValidators := []Range{
		Range{Min{100}, Max{10}},
		ValidRange(100, 10),
	}
	for _, rangeValidator := range badValidators {
		for _, i := range []int{50, 100, 10, 9, 101, 0, -1} {
			if rangeValidator.IsSatisfied(i) {
				t.Errorf(noErrorsMessage, "min > val < max")
			}
		}
	}
}

func TestMinSize(t *testing.T) {
	for _, minSize := range []MinSize{MinSize{1}, ValidMinSize(1)} {
		// string
		if !minSize.IsSatisfied("1") || !minSize.IsSatisfied("12") {
			t.Errorf(noErrorsMessage, "len(val) >= min")
		}

		// slice
		if !minSize.IsSatisfied([]int{1}) || !minSize.IsSatisfied([]int{1, 2}) {
			t.Errorf(noErrorsMessage, "len(val) >= min")
		}

		// string/slice
		if minSize.IsSatisfied("") || minSize.IsSatisfied([]int{}) {
			t.Errorf(errorsMessage, "len(val) <= min")
		}

		// non-string/slice type
		if minSize.IsSatisfied(nil) {
			t.Errorf(errorsMessage, "TypeOf(val) != string && TypeOf(val) != slice")
		}
	}
}

func TestMaxSize(t *testing.T) {
	for _, maxSize := range []MaxSize{MaxSize{2}, ValidMaxSize(2)} {
		// string
		if !maxSize.IsSatisfied("") || !maxSize.IsSatisfied("12") {
			t.Errorf(noErrorsMessage, "len(val) <= max")
		}

		// slice
		if !maxSize.IsSatisfied([]int{}) || !maxSize.IsSatisfied([]int{1, 2}) {
			t.Errorf(noErrorsMessage, "len(val) <= max")
		}

		// string/slice with len > max
		if maxSize.IsSatisfied("123") || maxSize.IsSatisfied([]int{1, 2, 3}) {
			t.Errorf(errorsMessage, "len(val) >= max")
		}

		// non-string/slice type
		if maxSize.IsSatisfied(nil) {
			t.Errorf(errorsMessage, "TypeOf(val) != string && TypeOf(val) != slice")
		}
	}
}

func TestLength(t *testing.T) {
	for _, length := range []Length{Length{2}, ValidLength(2)} {
		// string/slice
		if !length.IsSatisfied("12") || !length.IsSatisfied([]int{1, 2}) {
			t.Errorf(noErrorsMessage, "len(val) == length")
		}

		// string/slice with len > length
		if length.IsSatisfied("123") || length.IsSatisfied([]int{1, 2, 3}) {
			t.Errorf(errorsMessage, "len(val) > length")
		}

		// string/slice with len < length
		if length.IsSatisfied("1") || length.IsSatisfied([]int{1}) {
			t.Errorf(errorsMessage, "len(val) < length")
		}

		// non-string/slice type
		if length.IsSatisfied(nil) {
			t.Errorf(errorsMessage, "TypeOf(val) != string && TypeOf(val) != slice")
		}
	}
}

func TestMatch(t *testing.T) {
	regex := regexp.MustCompile(`[abc]{3}\d*`)
	for _, match := range []Match{Match{regex}, ValidMatch(regex)} {
		if !match.IsSatisfied("bca123") {
			t.Errorf(noErrorsMessage, `"[abc]{3}\d*" matches "bca123"`)
		}

		if match.IsSatisfied("bc123") {
			t.Errorf(errorsMessage, `"[abc]{3}\d*" does not match "ca123"`)
		}
		if match.IsSatisfied("") {
			t.Errorf(errorsMessage, `"[abc]{3}\d*" does not match "c"`)
		}
	}
}

func TestEmail(t *testing.T) {
	// unicode char included
	validStartingCharacters := strings.Split("!#$%^&*_+1234567890abcdefghijklmnopqrstuvwxyzñ", "")
	invalidCharacters := strings.Split(" ()", "")
	definiteInvalidDomains := []string{
		"",                  // any empty string (x@)
		".com",              // only the TLD (x@.com)
		".",                 // only the . (x@.)
		".*",                // TLD containing symbol (x@.*)
		"asdf",              // no TLD
		"a!@#$%^&*()+_.com", // characters which are not ASCII/0-9/dash(-) in a domain
		"-a.com",            // host starting with any symbol
		"a-.com",            // host ending with any symbol
		"aå.com",            // domain containing unicode (however, unicode domains do exist in the state of xn--<POINT>.com e.g. å.com = xn--5ca.com)
	}

	for _, email := range []Email{Email{Match{emailPattern}}, ValidEmail()} {
		var currentEmail string

		// test invalid starting chars
		for _, startingChar := range validStartingCharacters {
			currentEmail = fmt.Sprintf("%sñbc+123@do-main.com", startingChar)
			if email.IsSatisfied(currentEmail) {
				t.Errorf(noErrorsMessage, fmt.Sprintf("email = %s", currentEmail))
			}

			// validation should fail because of multiple @ symbols
			currentEmail = fmt.Sprintf("%s@ñbc+123@do-main.com", startingChar)
			if email.IsSatisfied(currentEmail) {
				t.Errorf(errorsMessage, fmt.Sprintf("email = %s", currentEmail))
			}

			// should fail simply because of the invalid char
			for _, invalidChar := range invalidCharacters {
				currentEmail = fmt.Sprintf("%sñbc%s+123@do-main.com", startingChar, invalidChar)
				if email.IsSatisfied(currentEmail) {
					t.Errorf(errorsMessage, fmt.Sprintf("email = %s", currentEmail))
				}
			}
		}

		// test invalid domains
		for _, invalidDomain := range definiteInvalidDomains {
			currentEmail = fmt.Sprintf("a@%s", invalidDomain)
			if email.IsSatisfied(currentEmail) {
				t.Errorf(errorsMessage, fmt.Sprintf("email = %s", currentEmail))
			}
		}

		// should always be satisfied
		if !email.IsSatisfied("t0.est+email123@1abc0-def.com") {
			t.Errorf(noErrorsMessage, fmt.Sprintf("email = %s", "t0.est+email123@1abc0-def.com"))
		}

		// should never be satisfied (this is redundant given the loops above)
		if email.IsSatisfied("a@xcom") {
			t.Errorf(noErrorsMessage, fmt.Sprintf("email = %s", "a@xcom"))
		}
		if email.IsSatisfied("a@@x.com") {
			t.Errorf(noErrorsMessage, fmt.Sprintf("email = %s", "a@@x.com"))
		}
	}
}
