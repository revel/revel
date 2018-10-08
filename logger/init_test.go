// Copyright (c) 2012-2018 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package logger_test

import (
	"github.com/revel/config"
	"github.com/revel/revel/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

type (
	// A counter for the tester
	testCounter struct {
		debug, info, warn, error, critical int
	}
	// The data to tes
	testData struct {
		config []string
		result testResult
		tc     *testCounter
	}
	// The test result
	testResult struct {
		debug, info, warn, error, critical int
	}
)

// Single test cases
var singleCases = []testData{
	{config: []string{"log.crit.output"},
		result: testResult{0, 0, 0, 0, 1}},
	{config: []string{"log.error.output"},
		result: testResult{0, 0, 0, 1, 1}},
	{config: []string{"log.warn.output"},
		result: testResult{0, 0, 1, 0, 0}},
	{config: []string{"log.info.output"},
		result: testResult{0, 1, 0, 0, 0}},
	{config: []string{"log.debug.output"},
		result: testResult{1, 0, 0, 0, 0}},
}

// Test singles
func TestSingleCases(t *testing.T) {
	rootLog := logger.New()
	for _, testCase := range singleCases {
		testCase.logTest(rootLog, t)
		testCase.validate(t)
	}
}

// Filter test cases
var filterCases = []testData{
	{config: []string{"log.crit.filter.module.app"},
		result: testResult{0, 0, 0, 0, 1}},
	{config: []string{"log.crit.filter.module.appa"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.error.filter.module.app"},
		result: testResult{0, 0, 0, 1, 1}},
	{config: []string{"log.error.filter.module.appa"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.warn.filter.module.app"},
		result: testResult{0, 0, 1, 0, 0}},
	{config: []string{"log.warn.filter.module.appa"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.info.filter.module.app"},
		result: testResult{0, 1, 0, 0, 0}},
	{config: []string{"log.info.filter.module.appa"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.debug.filter.module.app"},
		result: testResult{1, 0, 0, 0, 0}},
	{config: []string{"log.debug.filter.module.appa"},
		result: testResult{0, 0, 0, 0, 0}},
}

// Filter test
func TestFilterCases(t *testing.T) {
	rootLog := logger.New("module", "app")
	for _, testCase := range filterCases {
		testCase.logTest(rootLog, t)
		testCase.validate(t)
	}
}

// Inverse test cases
var nfilterCases = []testData{
	{config: []string{"log.crit.nfilter.module.appa"},
		result: testResult{0, 0, 0, 0, 1}},
	{config: []string{"log.crit.nfilter.modules.appa"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.crit.nfilter.module.app"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.error.nfilter.module.appa"}, // Special case, when error is not nill critical inherits from error
		result: testResult{0, 0, 0, 1, 1}},
	{config: []string{"log.error.nfilter.module.app"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.warn.nfilter.module.appa"},
		result: testResult{0, 0, 1, 0, 0}},
	{config: []string{"log.warn.nfilter.module.app"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.info.nfilter.module.appa"},
		result: testResult{0, 1, 0, 0, 0}},
	{config: []string{"log.info.nfilter.module.app"},
		result: testResult{0, 0, 0, 0, 0}},
	{config: []string{"log.debug.nfilter.module.appa"},
		result: testResult{1, 0, 0, 0, 0}},
	{config: []string{"log.debug.nfilter.module.app"},
		result: testResult{0, 0, 0, 0, 0}},
}

// Inverse test
func TestNotFilterCases(t *testing.T) {
	rootLog := logger.New("module", "app")
	for _, testCase := range nfilterCases {
		testCase.logTest(rootLog, t)
		testCase.validate(t)
	}
}

// off test cases
var offCases = []testData{
	{config: []string{"log.all.output", "log.error.output=off"},
		result: testResult{1, 1, 1, 0, 1}},
}

// Off test
func TestOffCases(t *testing.T) {
	rootLog := logger.New("module", "app")
	for _, testCase := range offCases {
		testCase.logTest(rootLog, t)
		testCase.validate(t)
	}
}

// Duplicate test cases
var duplicateCases = []testData{
	{config: []string{"log.all.output", "log.error.output", "log.error.filter.module.app"},
		result: testResult{1, 1, 1, 2, 1}},
}

// test duplicate cases
func TestDuplicateCases(t *testing.T) {
	rootLog := logger.New("module", "app")
	for _, testCase := range duplicateCases {
		testCase.logTest(rootLog, t)
		testCase.validate(t)
	}
}

// Contradicting cases
var contradictCases = []testData{
	{config: []string{"log.all.output", "log.error.output=off", "log.all.output"},
		result: testResult{1, 1, 1, 0, 1}},
	{config: []string{"log.all.output", "log.error.output=off", "log.debug.filter.module.app"},
		result: testResult{2, 1, 1, 0, 1}},
	{config: []string{"log.all.filter.module.app", "log.info.output=off", "log.info.filter.module.app"},
		result: testResult{1, 2, 1, 1, 1}},
	{config: []string{"log.all.output", "log.info.output=off", "log.info.filter.module.app"},
		result: testResult{1, 1, 1, 1, 1}},
}

// Contradiction test
func TestContradictCases(t *testing.T) {
	rootLog := logger.New("module", "app")
	for _, testCase := range contradictCases {
		testCase.logTest(rootLog, t)
		testCase.validate(t)
	}
}

// All test cases
var allCases = []testData{
	{config: []string{"log.all.filter.module.app"},
		result: testResult{1, 1, 1, 1, 1}},
	{config: []string{"log.all.output"},
		result: testResult{2, 2, 2, 2, 2}},
}

// All tests
func TestAllCases(t *testing.T) {
	rootLog := logger.New("module", "app")
	for i, testCase := range allCases {
		testCase.logTest(rootLog, t)
		allCases[i] = testCase
	}
	rootLog = logger.New()
	for i, testCase := range allCases {
		testCase.logTest(rootLog, t)
		allCases[i] = testCase
	}
	for _, testCase := range allCases {
		testCase.validate(t)
	}

}

func (c *testCounter) Log(r *logger.Record) error {
	switch r.Level {
	case logger.LvlDebug:
		c.debug++
	case logger.LvlInfo:
		c.info++
	case logger.LvlWarn:
		c.warn++
	case logger.LvlError:
		c.error++
	case logger.LvlCrit:
		c.critical++
	default:
		panic("Unknown log level")
	}
	return nil
}
func (td *testData) logTest(rootLog logger.MultiLogger, t *testing.T) {
	if td.tc == nil {
		td.tc = &testCounter{}
		counterInit(td.tc)
	}
	newContext := config.NewContext()
	for _, i := range td.config {
		iout := strings.Split(i, "=")
		if len(iout) > 1 {
			newContext.SetOption(iout[0], iout[1])
		} else {
			newContext.SetOption(i, "test")
		}
	}

	newContext.SetOption("specialUseFlag", "true")

	handler := logger.InitializeFromConfig("test", newContext)

	rootLog.SetHandler(handler)

	td.runLogTest(rootLog)
}

func (td *testData) runLogTest(log logger.MultiLogger) {
	log.Debug("test")
	log.Info("test")
	log.Warn("test")
	log.Error("test")
	log.Crit("test")
}

func (td *testData) validate(t *testing.T) {
	t.Logf("Test %#v expected %#v", td.tc, td.result)
	assert.Equal(t, td.result.debug, td.tc.debug, "Debug failed "+strings.Join(td.config, " "))
	assert.Equal(t, td.result.info, td.tc.info, "Info failed "+strings.Join(td.config, " "))
	assert.Equal(t, td.result.warn, td.tc.warn, "Warn failed "+strings.Join(td.config, " "))
	assert.Equal(t, td.result.error, td.tc.error, "Error failed "+strings.Join(td.config, " "))
	assert.Equal(t, td.result.critical, td.tc.critical, "Critical failed "+strings.Join(td.config, " "))
}

// Add test to the function map
func counterInit(tc *testCounter) {
	logger.LogFunctionMap["test"] = func(c *logger.CompositeMultiHandler, logOptions *logger.LogOptions) {
		// Output to the test log and the stdout
		outHandler := logger.LogHandler(
			logger.NewListLogHandler(tc,
				logger.StreamHandler(os.Stdout, logger.TerminalFormatHandler(false, true))),
		)
		if logOptions.HandlerWrap != nil {
			outHandler = logOptions.HandlerWrap.SetChild(outHandler)
		}

		c.SetHandlers(outHandler, logOptions)
	}
}
