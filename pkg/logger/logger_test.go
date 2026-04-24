package logger

import "testing"

func TestInit(t *testing.T) {
	t.Run("development", func(t *testing.T) {
		Init(false)
		if Logger == nil {
			t.Error("Expected Logger to not be nil")
		}
		if SugarLogger == nil {
			t.Error("Expected SugarLogger to not be nil")
		}
	})
	t.Run("production", func(t *testing.T) {
		Init(true)
		if Logger == nil {
			t.Error("Expected Logger to not be nil")
		}
		if SugarLogger == nil {
			t.Error("Expected SugarLogger to not be nil")
		}
	})
}
