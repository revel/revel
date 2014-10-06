package revel

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	errorsMessage   = "validation for %s should not be satisfied with %s\n"
	noErrorsMessage = "validation for %s should be satisfied with %s\n"
)

type Expect struct {
	input          interface{}
	expectedResult bool
	errorMessage   string
}

func performTests(validator Validator, tests []Expect, t *testing.T) {
	for _, test := range tests {
		if validator.IsSatisfied(test.input) != test.expectedResult {
			if test.expectedResult == false {
				t.Errorf(errorsMessage, reflect.TypeOf(validator), test.errorMessage)
			} else {
				t.Errorf(noErrorsMessage, reflect.TypeOf(validator), test.errorMessage)
			}
		}
	}
}

func TestRequired(t *testing.T) {

	tests := []Expect{
		Expect{nil, false, "nil data"},
		Expect{"Testing", true, "non-empty string"},
		Expect{"", false, "empty string"},
		Expect{true, true, "true boolean"},
		Expect{false, false, "false boolean"},
		Expect{1, true, "positive integer"},
		Expect{-1, true, "negative integer"},
		Expect{0, false, "0 integer"},
		Expect{time.Now(), true, "current time"},
		Expect{time.Time{}, false, "a zero time"},
		Expect{func() {}, true, "other non-nil data types"},
	}

	// testing both the struct and the helper method
	for _, required := range []Required{Required{}, ValidRequired()} {
		performTests(required, tests, t)
	}
}

func TestMin(t *testing.T) {
	tests := []Expect{
		Expect{11, true, "val > min"},
		Expect{10, true, "val == min"},
		Expect{9, false, "val < min"},
		Expect{true, false, "TypeOf(val) != int"},
	}
	for _, min := range []Min{Min{10}, ValidMin(10)} {
		performTests(min, tests, t)
	}
}

func TestMax(t *testing.T) {
	tests := []Expect{
		Expect{9, true, "val < max"},
		Expect{10, true, "val == max"},
		Expect{11, false, "val > max"},
		Expect{true, false, "TypeOf(val) != int"},
	}
	for _, max := range []Max{Max{10}, ValidMax(10)} {
		performTests(max, tests, t)
	}
}

func TestRange(t *testing.T) {
	tests := []Expect{
		Expect{50, true, "min <= val <= max"},
		Expect{10, true, "val == min"},
		Expect{100, true, "val == max"},
		Expect{9, false, "val < min"},
		Expect{101, false, "val > max"},
	}

	goodValidators := []Range{
		Range{Min{10}, Max{100}},
		ValidRange(10, 100),
	}
	for _, rangeValidator := range goodValidators {
		performTests(rangeValidator, tests, t)
	}

	tests = []Expect{
		Expect{10, true, "min == val == max"},
		Expect{9, false, "val < min && val < max && min == max"},
		Expect{11, false, "val > min && val > max && min == max"},
	}

	goodValidators = []Range{
		Range{Min{10}, Max{10}},
		ValidRange(10, 10),
	}
	for _, rangeValidator := range goodValidators {
		performTests(rangeValidator, tests, t)
	}

	tests = make([]Expect, 7)
	for i, num := range []int{50, 100, 10, 9, 101, 0, -1} {
		tests[i] = Expect{
			num,
			false,
			"min > val < max",
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
		performTests(rangeValidator, tests, t)
	}
}

func TestMinSize(t *testing.T) {
	greaterThanMessage := "len(val) >= min"
	tests := []Expect{
		Expect{"1", true, greaterThanMessage},
		Expect{"12", true, greaterThanMessage},
		Expect{[]int{1}, true, greaterThanMessage},
		Expect{[]int{1, 2}, true, greaterThanMessage},
		Expect{"", false, "len(val) <= min"},
		Expect{[]int{}, false, "len(val) <= min"},
		Expect{nil, false, "TypeOf(val) != string && TypeOf(val) != slice"},
	}

	for _, minSize := range []MinSize{MinSize{1}, ValidMinSize(1)} {
		performTests(minSize, tests, t)
	}
}

func TestMaxSize(t *testing.T) {
	lessThanMessage := "len(val) <= max"
	tests := []Expect{
		Expect{"", true, lessThanMessage},
		Expect{"12", true, lessThanMessage},
		Expect{[]int{}, true, lessThanMessage},
		Expect{[]int{1, 2}, true, lessThanMessage},
		Expect{"123", false, "len(val) >= max"},
		Expect{[]int{1, 2, 3}, false, "len(val) >= max"},
	}
	for _, maxSize := range []MaxSize{MaxSize{2}, ValidMaxSize(2)} {
		performTests(maxSize, tests, t)
	}
}

func TestLength(t *testing.T) {
	tests := []Expect{
		Expect{"12", true, "len(val) == length"},
		Expect{[]int{1, 2}, true, "len(val) == length"},
		Expect{"123", false, "len(val) > length"},
		Expect{[]int{1, 2, 3}, false, "len(val) > length"},
		Expect{"1", false, "len(val) < length"},
		Expect{[]int{1}, false, "len(val) < length"},
		Expect{nil, false, "TypeOf(val) != string && TypeOf(val) != slice"},
	}
	for _, length := range []Length{Length{2}, ValidLength(2)} {
		performTests(length, tests, t)
	}
}

func TestMatch(t *testing.T) {
	tests := []Expect{
		Expect{"bca123", true, `"[abc]{3}\d*" matches "bca123"`},
		Expect{"bc123", false, `"[abc]{3}\d*" does not match "bc123"`},
		Expect{"", false, `"[abc]{3}\d*" does not match ""`},
	}
	regex := regexp.MustCompile(`[abc]{3}\d*`)
	for _, match := range []Match{Match{regex}, ValidMatch(regex)} {
		performTests(match, tests, t)
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
				t.Errorf(noErrorsMessage, "starting characters", fmt.Sprintf("email = %s", currentEmail))
			}

			// validation should fail because of multiple @ symbols
			currentEmail = fmt.Sprintf("%s@ñbc+123@do-main.com", startingChar)
			if email.IsSatisfied(currentEmail) {
				t.Errorf(errorsMessage, "starting characters with multiple @ symbols", fmt.Sprintf("email = %s", currentEmail))
			}

			// should fail simply because of the invalid char
			for _, invalidChar := range invalidCharacters {
				currentEmail = fmt.Sprintf("%sñbc%s+123@do-main.com", startingChar, invalidChar)
				if email.IsSatisfied(currentEmail) {
					t.Errorf(errorsMessage, "invalid starting characters", fmt.Sprintf("email = %s", currentEmail))
				}
			}
		}

		// test invalid domains
		for _, invalidDomain := range definiteInvalidDomains {
			currentEmail = fmt.Sprintf("a@%s", invalidDomain)
			if email.IsSatisfied(currentEmail) {
				t.Errorf(errorsMessage, "invalid domain", fmt.Sprintf("email = %s", currentEmail))
			}
		}

		// should always be satisfied
		if !email.IsSatisfied("t0.est+email123@1abc0-def.com") {
			t.Errorf(noErrorsMessage, "guaranteed valid email", fmt.Sprintf("email = %s", "t0.est+email123@1abc0-def.com"))
		}

		// should never be satisfied (this is redundant given the loops above)
		if email.IsSatisfied("a@xcom") {
			t.Errorf(noErrorsMessage, "guaranteed invalid email", fmt.Sprintf("email = %s", "a@xcom"))
		}
		if email.IsSatisfied("a@@x.com") {
			t.Errorf(noErrorsMessage, "guaranteed invaild email", fmt.Sprintf("email = %s", "a@@x.com"))
		}
	}
}
