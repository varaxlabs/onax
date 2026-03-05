package watcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSchedule_Valid(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"standard", "*/5 * * * *"},
		{"hourly", "@hourly"},
		{"daily", "@daily"},
		{"complex", "0 2 * * 1-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sched, err := ParseSchedule(tt.expr)
			require.NoError(t, err)
			assert.NotNil(t, sched)
		})
	}
}

func TestParseSchedule_Invalid(t *testing.T) {
	_, err := ParseSchedule("not-valid")
	assert.Error(t, err)
}
