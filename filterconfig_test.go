package revel

import "testing"

type FakeController struct{}

func (c FakeController) FakeAction()   {}
func (c *FakeController) FakeAction2() {}

func TestFilterConfiguratorKey(t *testing.T) {
	conf := FilterController(FakeController{})
	if conf.key != "FakeController" {
		t.Errorf("Expected key 'FakeController', was %s", conf.key)
	}

	conf = FilterController(&FakeController{})
	if conf.key != "FakeController" {
		t.Errorf("Expected key 'FakeController', was %s", conf.key)
	}

	conf = FilterAction(FakeController.FakeAction)
	if conf.key != "FakeController.FakeAction" {
		t.Errorf("Expected key 'FakeController.FakeAction', was %s", conf.key)
	}

	conf = FilterAction((*FakeController).FakeAction2)
	if conf.key != "FakeController.FakeAction2" {
		t.Errorf("Expected key 'FakeController.FakeAction2', was %s", conf.key)
	}
}

func TestFilterConfiguratorOps(t *testing.T) {
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

	// First, verify getOverrideFilters returns just the filters after
	// FilterConfiguringFilter
	conf := FilterAction(FakeController.FakeAction)
	expected := []Filter{
		SessionFilter,
		FlashFilter,
		ActionInvoker,
	}
	actual := conf.getOverrideFilters()
	if len(actual) != len(expected) || !filterSliceEqual(actual, expected) {
		t.Errorf("getOverrideFilter failed.\nActual: %#v\nExpect: %#v", actual, expected)
	}

	// Now do one of each operation.
	conf.Add(NilFilter).
		Remove(FlashFilter).
		Insert(ValidationFilter, BEFORE, NilFilter)
	expected = []Filter{
		SessionFilter,
		ValidationFilter,
		NilFilter,
		ActionInvoker,
	}
	actual = filterOverrides[conf.key]
	if len(actual) != len(expected) || !filterSliceEqual(actual, expected) {
		t.Errorf("Ops failed.\nActual: %#v\nExpect: %#v", actual, expected)
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
