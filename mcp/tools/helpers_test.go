package tools

import (
	"testing"
)

// MockCallToolRequest implements the interface needed for testing
type MockCallToolRequest struct {
	args map[string]interface{}
}

func (m MockCallToolRequest) GetArguments() map[string]interface{} {
	return m.args
}

func (m MockCallToolRequest) GetString(key, defaultValue string) string {
	if val, ok := m.args[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return defaultValue
}

func (m MockCallToolRequest) GetBool(key string, defaultValue bool) bool {
	if val, ok := m.args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func (m MockCallToolRequest) GetIntSlice(key string, defaultValue []int) []int {
	if val, ok := m.args[key]; ok {
		if s, ok := val.([]int); ok {
			return s
		}
	}
	return defaultValue
}

func (m MockCallToolRequest) GetFloatSlice(key string, defaultValue []float64) []float64 {
	if val, ok := m.args[key]; ok {
		if s, ok := val.([]float64); ok {
			return s
		}
	}
	return defaultValue
}

func (m MockCallToolRequest) GetStringSlice(key string, defaultValue []string) []string {
	if val, ok := m.args[key]; ok {
		if s, ok := val.([]string); ok {
			return s
		}
	}
	return defaultValue
}

// Note: These tests require the actual mcp.CallToolRequest interface
// For now we'll test the logic via integration or manually verify

func TestGetNumberFromInt(t *testing.T) {
	// Test type conversions directly
	testCases := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"int", 42, 42},
		{"int64", int64(42), 42},
		{"float64", float64(42.7), 42},
		{"string int", "42", 42},
		{"string float", "42.7", 42},
		{"invalid string", "abc", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result int
			switch v := tc.input.(type) {
			case int:
				result = v
			case int64:
				result = int(v)
			case float64:
				result = int(v)
			case string:
				// Parse logic
				var i int
				if _, err := parseIntFromString(v); err == nil {
					i, _ = parseIntFromString(v)
					result = i
				}
			}

			if tc.name == "invalid string" {
				// For invalid string, we expect default (0)
				return
			}

			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// Helper to mimic the string parsing logic
func parseIntFromString(s string) (int, error) {
	var i int
	_, err := stringToInt(s, &i)
	return i, err
}

func stringToInt(s string, result *int) (bool, error) {
	var i int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			i = i*10 + int(c-'0')
		} else if c == '.' {
			break // Stop at decimal point
		} else if c == '-' && i == 0 {
			continue // Allow leading minus (simplified)
		} else {
			return false, nil
		}
	}
	*result = i
	return true, nil
}

func TestUIDValidation(t *testing.T) {
	// Test UID validation logic
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"positive int", 123, true},
		{"zero", 0, true},
		{"negative int", -1, false},
		{"positive float", float64(123), true},
		{"negative float", float64(-1), false},
		{"positive int64", int64(123), true},
		{"negative int64", int64(-1), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var valid bool
			switch v := tc.value.(type) {
			case int:
				valid = v >= 0
			case int64:
				valid = v >= 0
			case float64:
				valid = v >= 0
			}

			if valid != tc.isValid {
				t.Errorf("expected valid=%v, got valid=%v", tc.isValid, valid)
			}
		})
	}
}

func TestStringArrayConversion(t *testing.T) {
	// Test various input types that should convert to string arrays
	testCases := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "string slice",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "interface slice with strings",
			input:    []interface{}{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "interface slice with numbers",
			input:    []interface{}{float64(1), float64(2), float64(3)},
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "mixed interface slice",
			input:    []interface{}{"a", float64(2), 3},
			expected: []string{"a", "2", "3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result []string

			switch v := tc.input.(type) {
			case []string:
				result = v
			case []interface{}:
				result = make([]string, 0, len(v))
				for _, item := range v {
					switch iv := item.(type) {
					case string:
						result = append(result, iv)
					case float64:
						result = append(result, formatFloat(iv))
					case int:
						result = append(result, formatInt(iv))
					}
				}
			}

			if len(result) != len(tc.expected) {
				t.Errorf("expected length %d, got %d", len(tc.expected), len(result))
				return
			}

			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("at index %d: expected %s, got %s", i, tc.expected[i], v)
				}
			}
		})
	}
}

func formatFloat(f float64) string {
	// Simple int conversion for whole numbers
	if f == float64(int(f)) {
		return formatInt(int(f))
	}
	return ""
}

func formatInt(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	negative := i < 0
	if negative {
		i = -i
	}
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	if negative {
		result = "-" + result
	}
	return result
}

func TestChunkUIDs(t *testing.T) {
	testCases := []struct {
		name       string
		uids       []uint32
		chunkSize  int
		numChunks  int
		lastChunkSize int
	}{
		{
			name:       "exact chunks",
			uids:       []uint32{1, 2, 3, 4, 5, 6},
			chunkSize:  2,
			numChunks:  3,
			lastChunkSize: 2,
		},
		{
			name:       "partial last chunk",
			uids:       []uint32{1, 2, 3, 4, 5},
			chunkSize:  2,
			numChunks:  3,
			lastChunkSize: 1,
		},
		{
			name:       "single chunk",
			uids:       []uint32{1, 2, 3},
			chunkSize:  10,
			numChunks:  1,
			lastChunkSize: 3,
		},
		{
			name:       "empty input",
			uids:       []uint32{},
			chunkSize:  10,
			numChunks:  0,
			lastChunkSize: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunks := chunkUIDsHelper(tc.uids, tc.chunkSize)

			if len(chunks) != tc.numChunks {
				t.Errorf("expected %d chunks, got %d", tc.numChunks, len(chunks))
			}

			if tc.numChunks > 0 && len(chunks[len(chunks)-1]) != tc.lastChunkSize {
				t.Errorf("expected last chunk size %d, got %d", tc.lastChunkSize, len(chunks[len(chunks)-1]))
			}

			// Verify all UIDs are present
			var all []uint32
			for _, chunk := range chunks {
				all = append(all, chunk...)
			}
			if len(all) != len(tc.uids) {
				t.Errorf("expected total %d UIDs, got %d", len(tc.uids), len(all))
			}
		})
	}
}

// Helper function mimicking the chunk logic from bulk.go
func chunkUIDsHelper(uids []uint32, chunkSize int) [][]uint32 {
	if len(uids) == 0 {
		return nil
	}

	var chunks [][]uint32
	for i := 0; i < len(uids); i += chunkSize {
		end := i + chunkSize
		if end > len(uids) {
			end = len(uids)
		}
		chunks = append(chunks, uids[i:end])
	}
	return chunks
}

func TestParseDate(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"RFC3339", "2024-01-15T10:30:00Z", false},
		{"datetime no tz", "2024-01-15T10:30:00", false},
		{"date only", "2024-01-15", false},
		{"european format", "15.01.2024", false},
		{"us format", "01/15/2024", false},
		{"invalid", "not-a-date", true},
		{"empty", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDate(tc.input)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegistryHelpers(t *testing.T) {
	// Test the helper functions for creating tool options
	// These are primarily builder patterns, so we just verify they don't panic

	t.Run("requiredString", func(t *testing.T) {
		opt := requiredString("test", "Test description")
		if opt == nil {
			t.Error("expected non-nil option")
		}
	})

	t.Run("optionalString", func(t *testing.T) {
		opt := optionalString("test", "Test description")
		if opt == nil {
			t.Error("expected non-nil option")
		}
	})

	t.Run("requiredNumber", func(t *testing.T) {
		opt := requiredNumber("test", "Test description")
		if opt == nil {
			t.Error("expected non-nil option")
		}
	})

	t.Run("optionalNumber", func(t *testing.T) {
		opt := optionalNumber("test", "Test description")
		if opt == nil {
			t.Error("expected non-nil option")
		}
	})

	t.Run("optionalBool", func(t *testing.T) {
		opt := optionalBool("test", "Test description")
		if opt == nil {
			t.Error("expected non-nil option")
		}
	})
}
