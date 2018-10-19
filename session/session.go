// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/twinj/uuid"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	// The key for the identity of the session
	SessionIDKey = "_ID"
	// The expiration date of the session
	TimestampKey = "_TS"
	// The value name indicating how long the session should persist - ie should it persist after the browser closes
	// this is set under the TimestampKey if the session data should expire immediately
	SessionValueName = "session"
	// The key container for the json objects of the data, any non strings found in the map will be placed in here
	// serialized by key using JSON
	SessionObjectKeyName = "_object_"
	// The mapped session object
	SessionMapKeyName = "_map_"
	// The suffix of the session cookie
	SessionCookieSuffix = "_SESSION"
)

// Session data, can be any data, there are reserved keywords used by the storage data
// SessionIDKey Is the key name for the session
// TimestampKey Is the time that the session should expire
//
type Session map[string]interface{}

func NewSession() Session {
	return Session{}
}

// ID retrieves from the cookie or creates a time-based UUID identifying this
// session.
func (s Session) ID() string {
	if sessionIDStr, ok := s[SessionIDKey]; ok {
		return sessionIDStr.(string)
	}

	buffer := uuid.NewV4()

	s[SessionIDKey] = hex.EncodeToString(buffer.Bytes())
	return s[SessionIDKey].(string)
}

// getExpiration return a time.Time with the session's expiration date.
// It uses the passed in expireAfterDuration to add with the current time if the timeout is not
// browser dependent (ie session). If previous session has set to "session", the time returned is time.IsZero()
func (s Session) GetExpiration(expireAfterDuration time.Duration) time.Time {
	if expireAfterDuration == 0 || s[TimestampKey] == SessionValueName {
		// Expire after closing browser
		return time.Time{}
	}
	return time.Now().Add(expireAfterDuration)
}

// SetNoExpiration sets session to expire when browser session ends
func (s Session) SetNoExpiration() {
	s[TimestampKey] = SessionValueName
}

// SetDefaultExpiration sets session to expire after default duration
func (s Session) SetDefaultExpiration() {
	delete(s, TimestampKey)
}

// sessionTimeoutExpiredOrMissing returns a boolean of whether the session
// cookie is either not present or present but beyond its time to live; i.e.,
// whether there is not a valid session.
func (s Session) SessionTimeoutExpiredOrMissing() bool {
	if exp, present := s[TimestampKey]; !present {
		return true
	} else if exp == SessionValueName {
		return false
	} else if expInt, _ := strconv.Atoi(exp.(string)); int64(expInt) < time.Now().Unix() {
		return true
	}
	return false
}

// Constant error if session value is not found
var SESSION_VALUE_NOT_FOUND = errors.New("Session value not found")

// Get an object or property from the session
// it may be embedded inside the session.
func (s Session) Get(key string) (newValue interface{}, err error) {
	// First check to see if it is in the session
	if v, found := s[key]; found {
		return v, nil
	}
	return s.GetInto(key, nil, false)
}

// Get into the specified value.
// If value exists in the session it will just return the value
func (s Session) GetInto(key string, target interface{}, force bool) (result interface{}, err error) {
	if v, found := s[key]; found && !force {
		return v, nil
	}
	splitKey := strings.Split(key, ".")
	rootKey := splitKey[0]

	// Force always recreates the object from the session data map
	if force {
		if target == nil {
			if result, err = s.sessionDataFromMap(key); err != nil {
				return
			}
		} else if result, err = s.sessionDataFromObject(rootKey, target); err != nil {
			return
		}

		return s.getNestedProperty(splitKey, result)
	}

	// Attempt to find the key in the session, this is the most generalized form
	v, found := s[rootKey]
	if !found {
		if target == nil {
			// Try to fetch it from the session

			if v, err = s.sessionDataFromMap(rootKey); err != nil {
				return
			}
		} else if v, err = s.sessionDataFromObject(rootKey, target); err != nil {
			return
		}
	}

	return s.getNestedProperty(splitKey, v)
}

// Returns the default value if the key is not found
func (s Session) GetDefault(key string, value interface{}, defaultValue interface{}) interface{} {
	v, e := s.GetInto(key, value, false)
	if e != nil {
		v = defaultValue
	}
	return v
}

// Extract the values from the session
func (s Session) GetProperty(key string, value interface{}) (interface{}, error) {
	// Capitalize the first letter
	key = strings.Title(key)

	sessionLog.Info("getProperty", "key", key, "value", value)

	// For a map it is easy
	if reflect.TypeOf(value).Kind() == reflect.Map {
		val := reflect.ValueOf(value)
		valueOf := val.MapIndex(reflect.ValueOf(key))
		if valueOf == reflect.Zero(reflect.ValueOf(value).Type()) {
			return nil, nil
		}
		//idx := val.MapIndex(reflect.ValueOf(key))
		if !valueOf.IsValid() {
			return nil, nil
		}

		return valueOf.Interface(), nil
	}

	objValue := s.reflectValue(value)
	field := objValue.FieldByName(key)
	if !field.IsValid() {
		return nil, SESSION_VALUE_NOT_FOUND
	}

	return field.Interface(), nil
}

// Places the object into the session, a nil value will cause remove the key from the session
// (or you can use the Session.Del(key) function
func (s Session) Set(key string, value interface{}) error {
	if value == nil {
		s.Del(key)
		return nil
	}

	s[key] = value
	return nil
}

// Delete the key from the sessionObjects and Session
func (s Session) Del(key string) {
	sessionJsonMap := s.getSessionJsonMap()
	delete(sessionJsonMap, key)
	delete(s, key)
}

// Extracts the session as a map of [string keys] and json values
func (s Session) getSessionJsonMap() map[string]string {
	if sessionJson, found := s[SessionObjectKeyName]; found {
		if _, valid := sessionJson.(map[string]string); !valid {
			sessionLog.Error("Session object key corrupted, reset", "was", sessionJson)
			s[SessionObjectKeyName] = map[string]string{}
		}
		// serialized data inside the session _objects
	} else {
		s[SessionObjectKeyName] = map[string]string{}
	}

	return s[SessionObjectKeyName].(map[string]string)
}

// Convert the map to a simple map[string]string map
// this will marshal any non string objects encountered and store them the the jsonMap
// The expiration time will also be assigned
func (s Session) Serialize() map[string]string {
	sessionJsonMap := s.getSessionJsonMap()
	newMap := map[string]string{}
	newObjectMap := map[string]string{}
	for key, value := range sessionJsonMap {
		newObjectMap[key] = value
	}
	for key, value := range s {
		if key == SessionObjectKeyName || key == SessionMapKeyName {
			continue
		}
		if reflect.ValueOf(value).Kind() == reflect.String {
			newMap[key] = value.(string)
			continue
		}
		println("Serialize the data for", key)
		if data, err := json.Marshal(value); err != nil {
			sessionLog.Error("Unable to marshal session ", "key", key, "error", err)
			continue
		} else {
			newObjectMap[key] = string(data)
		}
	}
	if len(newObjectMap) > 0 {
		if data, err := json.Marshal(newObjectMap); err != nil {
			sessionLog.Error("Unable to marshal session ", "key", SessionObjectKeyName, "error", err)

		} else {
			newMap[SessionObjectKeyName] = string(data)
		}
	}

	return newMap
}

// Set the session object from the loaded data
func (s Session) Load(data map[string]string) {
	for key, value := range data {
		if key == SessionObjectKeyName {
			target := map[string]string{}
			if err := json.Unmarshal([]byte(value), &target); err != nil {
				sessionLog.Error("Unable to unmarshal session ", "key", SessionObjectKeyName, "error", err)
			} else {
				s[key] = target
			}
		} else {
			s[key] = value
		}

	}
}

// Checks to see if the session is empty
func (s Session) Empty() bool {
	i := 0
	for k := range s {
		i++
		if k == SessionObjectKeyName || k == SessionMapKeyName {
			continue
		}
	}
	return i == 0
}

func (s *Session) reflectValue(obj interface{}) reflect.Value {
	var val reflect.Value

	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		val = reflect.ValueOf(obj).Elem()
	} else {
		val = reflect.ValueOf(obj)
	}

	return val
}

// Starting at position 1 drill into the object
func (s Session) getNestedProperty(keys []string, newValue interface{}) (result interface{}, err error) {
	for x := 1; x < len(keys); x++ {
		newValue, err = s.GetProperty(keys[x], newValue)
		if err != nil || newValue == nil {
			return newValue, err
		}
	}
	return newValue, nil
}

// Always converts the data from the session mapped objects into the target,
// it will store the results under the session key name SessionMapKeyName
func (s Session) sessionDataFromMap(key string) (result interface{}, err error) {
	var mapValue map[string]interface{}
	uncastMapValue, found := s[SessionMapKeyName]
	if !found {
		mapValue = map[string]interface{}{}
		s[SessionMapKeyName] = mapValue
	} else if mapValue, found = uncastMapValue.(map[string]interface{}); !found {
		// Unusual means that the value in the session was not expected
		sessionLog.Errorf("Unusual means that the value in the session was not expected", "session", uncastMapValue)
		mapValue = map[string]interface{}{}
		s[SessionMapKeyName] = mapValue
	}

	// Try to extract the key from the map
	result, found = mapValue[key]
	if !found {
		result, err = s.convertSessionData(key, nil)
		if err == nil {
			mapValue[key] = result
		}
	}
	return
}

// Unpack the object from the session map and store it in the session when done, if no error occurs
func (s Session) sessionDataFromObject(key string, newValue interface{}) (result interface{}, err error) {
	result, err = s.convertSessionData(key, newValue)
	if err != nil {
		return
	}
	s[key] = result
	return
}

// Converts from the session json map into the target,
func (s Session) convertSessionData(key string, target interface{}) (result interface{}, err error) {
	sessionJsonMap := s.getSessionJsonMap()
	v, found := sessionJsonMap[key]
	if !found {
		return target, SESSION_VALUE_NOT_FOUND
	}

	// Create a target if needed
	if target == nil {
		target = map[string]interface{}{}
		if err := json.Unmarshal([]byte(v), &target); err != nil {
			return target, err
		}
	} else if err := json.Unmarshal([]byte(v), target); err != nil {
		return target, err
	}
	result = target
	return
}
