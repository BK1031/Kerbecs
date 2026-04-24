package utils

import (
	"testing"
)

func TestInitializeLogger(t *testing.T) {
	t.Run("DEV", func(t *testing.T) {
		t.Setenv("ENV", "DEV")
		InitializeLogger()
		if Logger == nil {
			t.Error("Expected Logger to not be nil")
		}
		if SugarLogger == nil {
			t.Error("Expected SugarLogger to not be nil")
		}
	})
	t.Run("PROD", func(t *testing.T) {
		t.Setenv("ENV", "PROD")
		InitializeLogger()
		if Logger == nil {
			t.Error("Expected Logger to not be nil")
		}
		if SugarLogger == nil {
			t.Error("Expected SugarLogger to not be nil")
		}
	})
}
