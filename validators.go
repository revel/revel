package revel

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Validator interface {
	IsSatisfied(interface{}) bool
	DefaultMessage() string
}

type Required struct{}

func ValidRequired() Required {
	return Required{}
}

func (r Required) IsSatisfied(obj interface{}) bool {
	if obj == nil {
		return false
	}

	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) > 0
	}
	if b, ok := obj.(bool); ok {
		return b
	}
	if i, ok := obj.(int); ok {
		return i != 0
	}
	if t, ok := obj.(time.Time); ok {
		return !t.IsZero()
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() > 0
	}
	return true
}

func (r Required) DefaultMessage() string {
	return "Required"
}

type Min struct {
	Min int
}

func ValidMin(min int) Min {
	return Min{min}
}

func (m Min) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num >= m.Min
	}
	return false
}

func (m Min) DefaultMessage() string {
	return fmt.Sprintln("Minimum is", m.Min)
}

type Max struct {
	Max int
}

func ValidMax(max int) Max {
	return Max{max}
}

func (m Max) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num <= m.Max
	}
	return false
}

func (m Max) DefaultMessage() string {
	return fmt.Sprintln("Maximum is", m.Max)
}

// Requires an integer to be within Min, Max inclusive.
type Range struct {
	Min
	Max
}

func ValidRange(min, max int) Range {
	return Range{Min{min}, Max{max}}
}

func (r Range) IsSatisfied(obj interface{}) bool {
	return r.Min.IsSatisfied(obj) && r.Max.IsSatisfied(obj)
}

func (r Range) DefaultMessage() string {
	return fmt.Sprintln("Range is", r.Min.Min, "to", r.Max.Max)
}

// Requires an array or string to be at least a given length.
type MinSize struct {
	Min int
}

func ValidMinSize(min int) MinSize {
	return MinSize{min}
}

func (m MinSize) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) >= m.Min
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() >= m.Min
	}
	return false
}

func (m MinSize) DefaultMessage() string {
	return fmt.Sprintln("Minimum size is", m.Min)
}

// Requires an array or string to be at most a given length.
type MaxSize struct {
	Max int
}

func ValidMaxSize(max int) MaxSize {
	return MaxSize{max}
}

func (m MaxSize) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) <= m.Max
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() <= m.Max
	}
	return false
}

func (m MaxSize) DefaultMessage() string {
	return fmt.Sprintln("Maximum size is", m.Max)
}

// Requires an array or string to be exactly a given length.
type Length struct {
	N int
}

func ValidLength(n int) Length {
	return Length{n}
}

func (s Length) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) == s.N
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() == s.N
	}
	return false
}

func (s Length) DefaultMessage() string {
	return fmt.Sprintln("Required length is", s.N)
}

// Requires a string to match a given regex.
type Match struct {
	Regexp *regexp.Regexp
}

func ValidMatch(regex *regexp.Regexp) Match {
	return Match{regex}
}

func (m Match) IsSatisfied(obj interface{}) bool {
	str := obj.(string)
	return m.Regexp.MatchString(str)
}

func (m Match) DefaultMessage() string {
	return fmt.Sprintln("Must match", m.Regexp)
}

var emailPattern = regexp.MustCompile("^[\\w!#$%&'*+/=?^_`{|}~-]+(?:\\.[\\w!#$%&'*+/=?^_`{|}~-]+)*@(?:[\\w](?:[\\w-]*[\\w])?\\.)+[a-zA-Z0-9](?:[\\w-]*[\\w])?$")

type Email struct {
	Match
}

func ValidEmail() Email {
	return Email{Match{emailPattern}}
}

func (e Email) DefaultMessage() string {
	return fmt.Sprintln("Must be a valid email address")
}

const (
	None               = 0
	IPAll              = 1
	IPv4               = 32 // IPv4 (32 chars)
	IPv6               = 39 // IPv6(39 chars)
	IPv4MappedIPv6     = 45 // IP4-mapped IPv6 (45 chars) , Ex) ::FFFF:129.144.52.38
	IPv4CIDR           = IPv4 + 3
	IPv6CIDR           = IPv6 + 3
	IPv4MappedIPv6CIDR = IPv4MappedIPv6 + 3
)

type IPAddr struct {
	vaildtypes []int
}

func ValidIPAddr(cktypes ...int) IPAddr {

	for _, cktype := range cktypes {

		if cktype != IPAll && cktype != IPv4 && cktype != IPv6 && cktype != IPv4MappedIPv6 && cktype != IPv4CIDR && cktype != IPv6CIDR && cktype != IPv4MappedIPv6CIDR {
			return IPAddr{vaildtypes: []int{None}}
		}
	}

	return IPAddr{vaildtypes: cktypes}
}

func isWithCIDR(str string, l int) bool {

	if str[l-3] == '/' || str[l-2] == '/' {

		cidr_bit := strings.Split(str, "/")
		if 2 == len(cidr_bit) {
			bit, err := strconv.Atoi(cidr_bit[1])
			//IPv4 : 0~32, IPv6 : 0 ~ 128
			if err == nil && bit >= 0 && bit <= 128 {
				return true
			}
		}
	}

	return false
}

func getIPType(str string, l int) int {

	if l < 3 { //least 3 chars (::F)
		return None
	}

	has_dot := strings.Index(str[2:], ".")
	has_colon := strings.Index(str[2:], ":")

	switch {
	case has_dot > -1 && has_colon == -1 && l >= 7 && l <= IPv4CIDR:
		if isWithCIDR(str, l) == true {
			return IPv4CIDR
		} else {
			return IPv4
		}
	case has_dot == -1 && has_colon > -1 && l >= 6 && l <= IPv6CIDR:
		if isWithCIDR(str, l) == true {
			return IPv6CIDR
		} else {
			return IPv6
		}

	case has_dot > -1 && has_colon > -1 && l >= 14 && l <= IPv4MappedIPv6:
		if isWithCIDR(str, l) == true {
			return IPv4MappedIPv6CIDR
		} else {
			return IPv4MappedIPv6
		}
	}

	return None
}

func (i IPAddr) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {

		l := len(str)
		ret := getIPType(str, l)

		for _, ck := range i.vaildtypes {

			if ret != None && (ck == ret || ck == IPAll) {

				switch ret {
				case IPv4, IPv6, IPv4MappedIPv6:
					ip := net.ParseIP(str)

					if ip != nil {
						return true
					}

				case IPv4CIDR, IPv6CIDR, IPv4MappedIPv6CIDR:
					_, _, err := net.ParseCIDR(str)
					if err == nil {
						return true
					}
				}
			}
		}
	}

	return false
}

func (i IPAddr) DefaultMessage() string {
	return fmt.Sprintln("Must be a vaild IP address")
}

type MacAddr struct{}

func ValidMacAddr() MacAddr {

	return MacAddr{}
}

func (m MacAddr) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {
		if _, err := net.ParseMAC(str); err == nil {
			return true
		}
	}

	return false
}

func (m MacAddr) DefaultMessage() string {
	return fmt.Sprintln("Must be a vaild MAC address")
}

var domainPattern = regexp.MustCompile(`^(([a-zA-Z0-9-\p{L}]{1,63}\.)?(xn--)?[a-zA-Z0-9\p{L}]+(-[a-zA-Z0-9\p{L}]+)*\.)+[a-zA-Z\p{L}]{2,63}$`)

type Domain struct {
	Regexp *regexp.Regexp
}

func ValidDomain() Domain {
	return Domain{domainPattern}
}

func (d Domain) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {

		l := len(str)
		//can't exceed 253 chars.
		if l > 253 {
			return false
		}

		//first and last char must be alphanumeric
		if str[l-1] == 46 || str[0] == 46 {
			return false
		}

		return domainPattern.MatchString(str)
	}

	return false
}

func (d Domain) DefaultMessage() string {
	return fmt.Sprintln("Must be a vaild domain address")
}

type URL struct {
	Domain
}

func ValidURL() URL {
	return URL{Domain: ValidDomain()}
}

func (u URL) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {
		if url, err := url.Parse(str); err == nil {

			if url.Scheme != "" && url.Host != "" && u.Domain.IsSatisfied(url.Host) == true {
				return true
			}
		}
	}

	return false
}

func (u URL) DefaultMessage() string {
	return fmt.Sprintln("Must be a vaild URL address")
}

type PureText struct{}

func ValidPureText() PureText {
	return PureText{}
}

func isPureText(str string) (bool, error) {

	l := len(str)

	for i := 0; i < l; i++ {

		c := str[i]

		// deny : control char
		if c >= 0 && c <= 31 && c != 9 && c != 10 && c != 13 {
			return false, errors.New("detect control character")
		}

		//deny : CRLF
		if c == 13 && i+2 < l && str[i+1] == 10 {
			return false, errors.New("detect <CR><LF>")
		}

		//deny : html tag (< ~ >)
		if c == 60 {

			for n := i; n < l; n++ {

				if str[n] == 60 && n+1 <= l && str[n+1] == 47 {
					return false, errors.New("detect tag (<~>)")
				}
			}
		}

		//deby : html encoded tag (&xxx;)
		if c == 38 && i+1 <= l && str[i+1] != 35 {

			for n := i; n < (n + 10); n++ {

				if str[n] == 59 {
					return false, errors.New("detect html encoded ta (&XXX;)")
				}
			}
		}
	}

	return true, nil
}

func (p PureText) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {

		ret, _ := isPureText(str)
		return ret
	}

	return false
}

func (p PureText) DefaultMessage() string {
	return fmt.Sprintln("Must be a vaild Text")
}
