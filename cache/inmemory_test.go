// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"testing"
	"time"
)

var newInMemoryCache = func(_ *testing.T, defaultExpiration time.Duration) Cache {
	return NewInMemoryCache(defaultExpiration)
}

// Test typical cache interactions
func TestInMemoryCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newInMemoryCache)
}

// Test the increment-decrement cases
func TestInMemoryCache_IncrDecr(t *testing.T) {
	incrDecr(t, newInMemoryCache)
}

func TestInMemoryCache_Expiration(t *testing.T) {
	expiration(t, newInMemoryCache)
}

func TestInMemoryCache_EmptyCache(t *testing.T) {
	emptyCache(t, newInMemoryCache)
}

func TestInMemoryCache_Replace(t *testing.T) {
	testReplace(t, newInMemoryCache)
}

func TestInMemoryCache_Add(t *testing.T) {
	testAdd(t, newInMemoryCache)
}

func TestInMemoryCache_GetMulti(t *testing.T) {
	testGetMulti(t, newInMemoryCache)
}
