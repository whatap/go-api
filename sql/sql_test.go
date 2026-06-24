package sql

import (
	"database/sql/driver"
	"strings"
	"testing"
)

// §48: paramsToString — SQL 파라미터 문자열 변환 테스트
func TestParamsToString_Basic(t *testing.T) {
	result := paramsToString("value1", 123, "value2")
	if !strings.Contains(result, "'value1'") {
		t.Errorf("§48: string param should be quoted, got %q", result)
	}
	if !strings.Contains(result, "123") {
		t.Errorf("§48: int param should be present, got %q", result)
	}
}

// §48: paramsToString — driver.NamedValue 처리
func TestParamsToString_NamedValue(t *testing.T) {
	nv := driver.NamedValue{
		Name:    "id",
		Ordinal: 1,
		Value:   "test_value",
	}
	result := paramsToString(nv)
	if !strings.Contains(result, "'test_value'") {
		t.Errorf("§48: NamedValue should extract .Value, got %q", result)
	}
}

// §48: paramsToString — 빈 파라미터
func TestParamsToString_Empty(t *testing.T) {
	result := paramsToString()
	if result != "" {
		t.Errorf("§48: empty params should return empty string, got %q", result)
	}
}

// §48: paramsToString — 파라미터 개수 제한 (SQL_PARAM_MAX_COUNT=20)
func TestParamsToString_MaxCount(t *testing.T) {
	params := make([]interface{}, 25)
	for i := range params {
		params[i] = i
	}
	result := paramsToString(params...)
	// Should not contain params beyond index 19 (0-indexed, 20 max)
	if strings.Contains(result, "20,") || strings.Contains(result, ",20") {
		// param at index 20 should be cut off
		parts := strings.Split(result, ",")
		if len(parts) > SQL_PARAM_MAX_COUNT {
			t.Errorf("§48: should limit to %d params, got %d parts", SQL_PARAM_MAX_COUNT, len(parts))
		}
	}
}

// §48: paramsToString — nil 파라미터 처리
func TestParamsToString_NilParam(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("§48: paramsToString(nil) panicked: %v", r)
		}
	}()
	result := paramsToString(nil, "value")
	if result == "" {
		t.Error("§48: nil + string param should produce non-empty result")
	}
}
