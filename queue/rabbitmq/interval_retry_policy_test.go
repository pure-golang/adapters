package rabbitmq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMaxInterval(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	correctBaseInterval := DefaultRetryInterval
	incorrectBaseInterval := time.Duration(0)
	correctMaxInterval := DefaultMaxInterval
	incorrectMaxInterval := time.Duration(0)
	correctMultiplicator := DefaultConnIntervalMultiplicator
	incorrectMultiplicator := 0

	testCases := []struct {
		baseInterval       time.Duration
		maxInterval        time.Duration
		multiplicator      int
		expectedValueIsNil bool
		expectedError      string
	}{
		{
			correctBaseInterval,
			correctMaxInterval,
			correctMultiplicator,
			false,
			"",
		},
		{
			incorrectBaseInterval,
			correctMaxInterval,
			correctMultiplicator,
			true,
			"interval should not be 0",
		},
		{
			correctBaseInterval,
			incorrectMaxInterval,
			correctMultiplicator,
			true,
			"max interval should not be 0",
		},
		{
			correctBaseInterval,
			correctMaxInterval,
			incorrectMultiplicator,
			true,
			"multiplicator should not be 0",
		},
	}

	for _, testCase := range testCases {
		f := func() {
			expectedInterval := NewMaxInterval(
				testCase.baseInterval,
				testCase.maxInterval,
				testCase.multiplicator,
			)

			assert.Equal(t, testCase.expectedValueIsNil, expectedInterval == nil)
		}

		if testCase.expectedError == "" {
			assert.NotPanics(t, f)
		} else {
			assert.PanicsWithValue(t, testCase.expectedError, f)
		}
	}
}

func TestMaxInterval_TryNum(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	testCases := []struct {
		baseInterval  time.Duration
		maxInterval   time.Duration
		multiplicator int
		tryNum        int
		expectedValue time.Duration
		isStop        bool
	}{
		{
			baseInterval:  time.Second,
			maxInterval:   time.Second * 3,
			multiplicator: 2,
			tryNum:        1,
			expectedValue: time.Second * 2,
			isStop:        false,
		},
		{
			baseInterval:  time.Second,
			maxInterval:   time.Second * 5,
			multiplicator: 2,
			tryNum:        2,
			expectedValue: time.Second * 4,
			isStop:        false,
		},
		{
			baseInterval:  time.Second,
			maxInterval:   time.Second * 2,
			multiplicator: 2,
			tryNum:        3,
			expectedValue: 0,
			isStop:        true,
		},
	}

	for idx, testCase := range testCases {
		interval := NewMaxInterval(
			testCase.baseInterval,
			testCase.maxInterval,
			testCase.multiplicator,
		)

		expected, stop := interval.TryNum(testCase.tryNum)

		assert.Equal(t, testCase.expectedValue, expected, "Case %d failed", idx)
		assert.Equal(t, testCase.isStop, stop)
	}
}
