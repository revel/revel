// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import "testing"

type FakeController struct{}

func (c FakeController) Foo()  {}
func (c *FakeController) Bar() {}

func TestFilterConfiguratorKey(t *testing.T) {
	conf := FilterController(FakeController{})
	if conf.key != "FakeController" {
		t.Errorf("Expected key 'FakeController', was %s", conf.key)
	}

	conf = FilterController(&FakeController{})
	if conf.key != "FakeController" {
		t.Errorf("Expected key 'FakeController', was %s", conf.key)
	}

	conf = FilterAction(FakeController.Foo)
	if conf.key != "FakeController.Foo" {
		t.Errorf("Expected key 'FakeController.Foo', was %s", conf.key)
	}

	conf = FilterAction((*FakeController).Bar)
	if conf.key != "FakeController.Bar" {
		t.Errorf("Expected key 'FakeController.Bar', was %s", conf.key)
	}
}

func TestFilterConfigurator(t *testing.T) {
	// Filters is global state.  Restore it after this test.
	oldFilters := make([]Filter, len(Filters))
	copy(oldFilters, Filters)
	defer func() {
		Filters = oldFilters
	}()

	Filters = []Filter{
		RouterFilter,
		FilterConfiguringFilter,
		SessionFilter,
		FlashFilter,
		ActionInvoker,
	}

	// Do one of each operation.
	conf := FilterAction(FakeController.Foo).
		Add(NilFilter).
		Remove(FlashFilter).
		Insert(ValidationFilter, BEFORE, NilFilter).
		Insert(I18nFilter, AFTER, NilFilter)
	expected := []Filter{
		SessionFilter,
		ValidationFilter,
		NilFilter,
		I18nFilter,
		ActionInvoker,
	}
	actual := getOverride("Foo")
	if len(actual) != len(expected) || !filterSliceEqual(actual, expected) {
		t.Errorf("Ops failed.\nActual: %#v\nExpect: %#v\nConf:%v", actual, expected, conf)
	}

	// Action2 should be unchanged
	if getOverride("Bar") != nil {
		t.Errorf("Filtering Action should not affect Action2.")
	}

	// Test that combining overrides on both the Controller and Action works.
	FilterController(FakeController{}).
		Add(PanicFilter)
	expected = []Filter{
		SessionFilter,
		ValidationFilter,
		NilFilter,
		I18nFilter,
		PanicFilter,
		ActionInvoker,
	}
	actual = getOverride("Foo")
	if len(actual) != len(expected) || !filterSliceEqual(actual, expected) {
		t.Errorf("Expected PanicFilter added to Foo.\nActual: %#v\nExpect: %#v", actual, expected)
	}

	expected = []Filter{
		SessionFilter,
		FlashFilter,
		PanicFilter,
		ActionInvoker,
	}
	actual = getOverride("Bar")
	if len(actual) != len(expected) || !filterSliceEqual(actual, expected) {
		t.Errorf("Expected PanicFilter added to Bar.\nActual: %#v\nExpect: %#v", actual, expected)
	}

	FilterAction((*FakeController).Bar).
		Add(NilFilter)
	expected = []Filter{
		SessionFilter,
		ValidationFilter,
		NilFilter,
		I18nFilter,
		PanicFilter,
		ActionInvoker,
	}
	actual = getOverride("Foo")
	if len(actual) != len(expected) || !filterSliceEqual(actual, expected) {
		t.Errorf("Expected no change to Foo.\nActual: %#v\nExpect: %#v", actual, expected)
	}

	expected = []Filter{
		SessionFilter,
		FlashFilter,
		PanicFilter,
		NilFilter,
		ActionInvoker,
	}
	actual = getOverride("Bar")
	if len(actual) != len(expected) || !filterSliceEqual(actual, expected) {
		t.Errorf("Expected NilFilter added to Bar.\nActual: %#v\nExpect: %#v", actual, expected)
	}
}

func filterSliceEqual(a, e []Filter) bool {
	for i, f := range a {
		if !FilterEq(f, e[i]) {
			return false
		}
	}
	return true
}

func getOverride(methodName string) []Filter {
	return getOverrideChain("FakeController", "FakeController."+methodName)
}
