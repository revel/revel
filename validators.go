// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"errors"
	"fmt"
	"html"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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
	switch v := reflect.ValueOf(obj); v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String, reflect.Chan:
		if v.Len() == 0 {
			return false
		}
	case reflect.Ptr:
		return r.IsSatisfied(reflect.Indirect(v).Interface())
	}
	return !reflect.DeepEqual(obj, reflect.Zero(reflect.TypeOf(obj)).Interface())
}

func (r Required) DefaultMessage() string {
	return fmt.Sprintln("Required")
}

type Min struct {
	Min float64
}

func ValidMin(min int) Min {
	return ValidMinFloat(float64(min))
}

func ValidMinFloat(min float64) Min {
	return Min{min}
}

func (m Min) IsSatisfied(obj interface{}) bool {
	var (
		num float64
		ok  bool
	)
	switch reflect.TypeOf(obj).Kind() {
	case reflect.Float64:
		num, ok = obj.(float64)
	case reflect.Float32:
		ok = true
		num = float64(obj.(float32))
	case reflect.Int:
		ok = true
		num = float64(obj.(int))
	}

	if ok {
		return num >= m.Min
	}
	return false
}

func (m Min) DefaultMessage() string {
	return fmt.Sprintln("Minimum is", m.Min)
}

type Max struct {
	Max float64
}

func ValidMax(max int) Max {
	return ValidMaxFloat(float64(max))
}

func ValidMaxFloat(max float64) Max {
	return Max{max}
}

func (m Max) IsSatisfied(obj interface{}) bool {
	var (
		num float64
		ok  bool
	)
	switch reflect.TypeOf(obj).Kind() {
	case reflect.Float64:
		num, ok = obj.(float64)
	case reflect.Float32:
		ok = true
		num = float64(obj.(float32))
	case reflect.Int:
		ok = true
		num = float64(obj.(int))
	}

	if ok {
		return num <= m.Max
	}
	return false
}

func (m Max) DefaultMessage() string {
	return fmt.Sprintln("Maximum is", m.Max)
}

// Range requires an integer to be within Min, Max inclusive.
type Range struct {
	Min
	Max
}

func ValidRange(min, max int) Range {
	return ValidRangeFloat(float64(min), float64(max))
}

func ValidRangeFloat(min, max float64) Range {
	return Range{Min{min}, Max{max}}
}

func (r Range) IsSatisfied(obj interface{}) bool {
	return r.Min.IsSatisfied(obj) && r.Max.IsSatisfied(obj)
}

func (r Range) DefaultMessage() string {
	return fmt.Sprintln("Range is", r.Min.Min, "to", r.Max.Max)
}

// MinSize requires an array or string to be at least a given length.
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

// MaxSize requires an array or string to be at most a given length.
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

// Length requires an array or string to be exactly a given length.
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

// Match requires a string to match a given regex.
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
	IPAny              = 1
	IPv4               = 32 // IPv4 (32 chars)
	IPv6               = 39 // IPv6(39 chars)
	IPv4MappedIPv6     = 45 // IP4-mapped IPv6 (45 chars) , Ex) ::FFFF:129.144.52.38
	IPv4CIDR           = IPv4 + 3
	IPv6CIDR           = IPv6 + 3
	IPv4MappedIPv6CIDR = IPv4MappedIPv6 + 3
)

// Requires a string(IP Address) to be within IP Pattern type inclusive.
type IPAddr struct {
	Vaildtypes []int
}

// Requires an IP Address string to be exactly a given  validation type (IPv4, IPv6, IPv4MappedIPv6, IPv4CIDR, IPv6CIDR, IPv4MappedIPv6CIDR OR IPAny)
func ValidIPAddr(cktypes ...int) IPAddr {

	for _, cktype := range cktypes {

		if cktype != IPAny && cktype != IPv4 && cktype != IPv6 && cktype != IPv4MappedIPv6 && cktype != IPv4CIDR && cktype != IPv6CIDR && cktype != IPv4MappedIPv6CIDR {
			return IPAddr{Vaildtypes: []int{None}}
		}
	}

	return IPAddr{Vaildtypes: cktypes}
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

		for _, ck := range i.Vaildtypes {

			if ret != None && (ck == ret || ck == IPAny) {

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

// Requires a MAC Address string to be exactly
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

// Requires a Domain string to be exactly
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

var urlPattern = regexp.MustCompile(`^((((https?|ftps?|gopher|telnet|nntp)://)|(mailto:|news:))(%[0-9A-Fa-f]{2}|[-()_.!~*';/?:@#&=+$,A-Za-z0-9\p{L}])+)([).!';/?:,][[:blank:]])?$`)

type URL struct {
	Domain
}

func ValidURL() URL {
	return URL{Domain: ValidDomain()}
}

func (u URL) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {

		// TODO : Required lot of testing
		return urlPattern.MatchString(str)
	}

	return false
}

func (u URL) DefaultMessage() string {
	return fmt.Sprintln("Must be a vaild URL address")
}

/*
NORMAL BenchmarkRegex-8   	2000000000	         0.24 ns/op
STRICT BenchmarkLoop-8    	2000000000	         0.01 ns/op
*/
const (
	NORMAL = 0
	STRICT = 4
)

// Requires a string to be without invisible characters
type PureText struct {
	Mode int
}

func ValidPureText(m int) PureText {
	if m != NORMAL && m != STRICT { // Q:required fatal error
		m = STRICT
	}
	return PureText{m}
}

func isPureTextStrict(str string) (bool, error) {

	l := len(str)

	for i := 0; i < l; i++ {

		c := str[i]

		// deny : control char (00-31 without 9(TAB) and Single 10(LF),13(CR)
		if c >= 0 && c <= 31 && c != 9 && c != 10 && c != 13 {
			return false, errors.New("detect control character")
		}

		// deny : control char (DEL)
		if c == 127 {
			return false, errors.New("detect control character (DEL)")
		}

		//deny : html tag (< ~ >)
		if c == 60 {

			ds := 0
			for n := i; n < l; n++ {

				// 60 (<) , 47(/) | 33(!) | 63(?)
				if str[n] == 60 && n+1 <= l && (str[n+1] == 47 || str[n+1] == 33 || str[n+1] == 63) {
					ds = 1
					n += 3 //jump to next char
				}

				// 62 (>)
				if ds == 1 && str[n] == 62 {
					return false, errors.New("detect tag (<[!|?]~>)")
				}
			}
		}

		//deby : html encoded tag (&xxx;)
		if c == 38 && i+1 <= l && str[i+1] != 35 {

			max := i + 64
			if max > l {
				max = l
			}
			for n := i; n < max; n++ {
				if str[n] == 59 {
					return false, errors.New("detect html encoded ta (&XXX;)")
				}
			}
		}
	}

	return true, nil
}

// Requires a string to match a given html tag elements regex pattern
// referrer : http://www.w3schools.com/Tags/
var elementPattern = regexp.MustCompile(`(?im)<(?P<tag>(/*\s*|\?*|\!*)(figcaption|expression|blockquote|plaintext|textarea|progress|optgroup|noscript|noframes|menuitem|frameset|fieldset|!DOCTYPE|datalist|colgroup|behavior|basefont|summary|section|isindex|details|caption|bgsound|article|address|acronym|strong|strike|source|select|script|output|option|object|legend|keygen|ilayer|iframe|header|footer|figure|dialog|center|canvas|button|applet|video|track|title|thead|tfoot|tbody|table|style|small|param|meter|layer|label|input|frame|embed|blink|audio|aside|alert|time|span|samp|ruby|meta|menu|mark|main|link|html|head|form|font|code|cite|body|base|area|abbr|xss|xml|wbr|var|svg|sup|sub|pre|nav|map|kbd|ins|img|div|dir|dfn|del|col|big|bdo|bdi|!--|ul|tt|tr|th|td|rt|rp|ol|li|hr|em|dt|dl|dd|br|u|s|q|p|i|b|a|(h[0-9]+)))([^><]*)([><]*)`)

// Requires a string to match a given urlencoded regex pattern
var urlencodedPattern = regexp.MustCompile(`(?im)(\%[0-9a-fA-F]{1,})`)

// Requires a string to match a given control characters regex pattern (ASCII : 00-08, 11, 12, 14, 15-31)
var controlcharPattern = regexp.MustCompile(`(?im)([\x00-\x08\x0B\x0C\x0E-\x1F\x7F]+)`)

func isPureTextNormal(str string) (bool, error) {

	decoded_str := html.UnescapeString(str)

	matched_urlencoded := urlencodedPattern.MatchString(decoded_str)
	if matched_urlencoded == true {
		temp_buf, err := url.QueryUnescape(decoded_str)
		if err == nil {
			decoded_str = temp_buf
		}
	}

	matched_element := elementPattern.MatchString(decoded_str)
	if matched_element == true {
		return false, errors.New("detect html element")
	}

	matched_cc := controlcharPattern.MatchString(decoded_str)
	if matched_cc == true {
		return false, errors.New("detect control character")
	}

	return true, nil
}

func (p PureText) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {

		var ret bool
		switch p.Mode {
		case STRICT:
			ret, _ = isPureTextStrict(str)
		case NORMAL:
			ret, _ = isPureTextStrict(str)
		}
		return ret
	}

	return false
}

func (p PureText) DefaultMessage() string {
	return fmt.Sprintln("Must be a vaild Text")
}

const (
	ONLY_FILENAME       = 0
	ALLOW_RELATIVE_PATH = 1
)

const regexDenyFileNameCharList = `[\x00-\x1f|\x21-\x2c|\x3b-\x40|\x5b-\x5e|\x60|\x7b-\x7f]+`
const regexDenyFileName = `|\x2e\x2e\x2f+`

var checkAllowRelativePath = regexp.MustCompile(`(?m)(` + regexDenyFileNameCharList + `)`)
var checkDenyRelativePath = regexp.MustCompile(`(?m)(` + regexDenyFileNameCharList + regexDenyFileName + `)`)

// Requires an string to be sanitary file path
type FilePath struct {
	Mode int
}

func ValidFilePath(m int) FilePath {

	if m != ONLY_FILENAME && m != ALLOW_RELATIVE_PATH {
		m = ONLY_FILENAME
	}
	return FilePath{m}
}

func (f FilePath) IsSatisfied(obj interface{}) bool {

	if str, ok := obj.(string); ok {

		var ret bool
		switch f.Mode {

		case ALLOW_RELATIVE_PATH:
			ret = checkAllowRelativePath.MatchString(str)
			if ret == false {
				return true
			}
		default: //ONLY_FILENAME
			ret = checkDenyRelativePath.MatchString(str)
			if ret == false {
				return true
			}
		}
	}

	return false
}

func (f FilePath) DefaultMessage() string {
	return fmt.Sprintln("Must be a unsanitary string")
}
