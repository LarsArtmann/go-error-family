package errorfamily

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

func TestRegisterStdlibDefaults(t *testing.T) {
	reg := NewRegistry()
	RegisterStdlibDefaults(reg)

	tests := []struct {
		name string
		err  error
		want Family
	}{
		{"deadline exceeded is transient", context.DeadlineExceeded, Transient},
		{"canceled is rejection", context.Canceled, Rejection},
		{"sql no rows is rejection", sql.ErrNoRows, Rejection},
		{"sql conn done is transient", sql.ErrConnDone, Transient},
		{"os not exist is rejection", os.ErrNotExist, Rejection},
		{"os permission is rejection", os.ErrPermission, Rejection},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reg.Classify(tt.err); got != tt.want {
				t.Errorf("Classify(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
