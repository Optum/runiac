package terraform

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var tfOptions Options

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	os.Exit(retCode)
}

func setup() {
	logger := logrus.New()
	lvl, _ := logrus.ParseLevel("Panic") // Suppress noisy logs for testing
	logger.SetLevel(lvl)
	log := logrus.NewEntry(logger)
	tfOptions = Options{
		Logger: log,
	}
}

func TestGetFirstAndLastIndex(t *testing.T) {
	tests := []struct {
		S              string
		FirstSubstring string
		LastSubstring  string
		ExpectedFirst  int
		ExpectedLast   int
	}{
		{
			S:              "[\"a\", \"b\", \"c\"]",
			FirstSubstring: "[",
			LastSubstring:  "]",
			ExpectedFirst:  0,
			ExpectedLast:   14,
		},
		{
			S:              "[\"a\", \"b\", \"c\"",
			FirstSubstring: "[",
			LastSubstring:  "]",
			ExpectedFirst:  0,
			ExpectedLast:   -1,
		},
		{
			S:              "\"a\", \"b\", \"c\"]",
			FirstSubstring: "[",
			LastSubstring:  "]",
			ExpectedFirst:  -1,
			ExpectedLast:   13,
		},
	}

	for _, tc := range tests {
		first, last := getFirstAndLastIndex(tc.S, tc.FirstSubstring, tc.LastSubstring)
		assert.Equal(t, tc.ExpectedFirst, first)
		assert.Equal(t, tc.ExpectedLast, last)
	}
}

func TestOutputToString(t *testing.T) {
	tests := []struct {
		Value          interface{}
		ExpectedString string
	}{
		{
			Value:          "vpcid-123",
			ExpectedString: "vpcid-123",
		},
		{
			Value:          []interface{}{"subnet1", "subnet2", "subnet3"},
			ExpectedString: "[\"subnet1\",\"subnet2\",\"subnet3\"]",
		},
		{
			Value:          []interface{}{},
			ExpectedString: "[]",
		},
		{
			Value:          map[string]interface{}{"k1": "v1", "k2": "v2"},
			ExpectedString: "{\"k1\":\"v1\",\"k2\":\"v2\"}",
		},
		{
			Value:          map[string]map[string][]string{"AZU": {"UK": []string{"uk1", "uk2"}, "US": []string{"us1", "us2"}}, "AWS": {"UK": []string{"uk1"}}, "GCP": {"UK": []string{"uk1"}}},
			ExpectedString: "{\"AWS\":{\"UK\":[\"uk1\"]},\"AZU\":{\"UK\":[\"uk1\",\"uk2\"],\"US\":[\"us1\",\"us2\"]},\"GCP\":{\"UK\":[\"uk1\"]}}",
		},
	}

	for _, tc := range tests {
		result := OutputToString(tc.Value)
		assert.Equal(t, tc.ExpectedString, result)
	}
}
