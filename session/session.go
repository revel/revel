// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"encoding/hex"
	"strconv"
	"time"
	"github.com/twinj/uuid"
	"github.com/revel/revel/logger"
	"reflect"
	"encoding/json"
	"errors"
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
	SessionObjectKeyName  = "_object_"
	// The suffix of the session cookie
	SessionCookieSuffix = "_SESSION"
	// The page session parameter
	PageSessionParam = "PAGE_SESSION_ID"
)

// Session data, can be any data, there are reserved keywords used by the storage data
// SessionIDKey
//
type Session map[string]interface{}

func NewSession() Session {
	return Session{}
}

// The logger for the session
var sessionLog logger.MultiLogger

// ID retrieves from the cookie or creates a time-based UUID identifying this
// session.
func (s Session) ID() string {
	if sessionIDStr, ok := s[SessionIDKey]; ok {
		return sessionIDStr.(string)
	}

	buffer := uuid.NewV4()

	s[SessionIDKey] = hex.EncodeToString(buffer)
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

// Unmarshal a session object, if same object unmarshalled more then once this will return
// the original object (the passed in pointer is NOT populated). It uses
// json.Unmarshal extract the object from the session
// and place it in the value.
func (s Session) Get(key string, value interface{}) (interface{}, error) {
	if v, found := s[key]; found {
		return v, nil
	}
	sessionJsonMap := s.getSessionJsonMap()
	v, found := sessionJsonMap[key]
	if found {
		// Attempt to decode the value
		if err := json.Unmarshal([]byte(v), value); err != nil {
			return nil, err
		}
		s[key] = v
		return v, nil
	}

	return nil, SESSION_VALUE_NOT_FOUND
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

func (s Session) getSessionJsonMap() map[string]string {
	if sessionJson, found := s[SessionObjectKeyName] ; found {
		if _, valid := sessionJson.(map[string]string); !valid {
			s[SessionObjectKeyName] = map[string]string{}
			sessionLog.Error("Session object key corrupted, reset")
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
	for key,value := range sessionJsonMap {
		newMap[key] = value
	}
	for key,value := range s {
		if key == SessionObjectKeyName {
			continue
		}
		if reflect.ValueOf(value).Kind() == reflect.String {
			newMap[key] = value.(string)
			continue
		}
		if data,err:=json.Marshal(value);err!=nil {
			sessionLog.Error("Unable to marshal session ","key",key,"error",err)
			continue
		} else {
			newMap[key] = string(data)
		}
	}

	return newMap
}

// Set the smartsession object from the loaded data
func (s Session) Load(data map[string]string)  {
	for key,value := range data {
		s[key] = value
	}
}
func (s Session) Empty() bool {
	return len(s)<2
}
