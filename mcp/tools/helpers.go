package tools

import (
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

// getNumber extracts a number from request - handles int, float64, and string
func getNumber(request mcp.CallToolRequest, key string, defaultValue int) int {
	args := request.GetArguments()
	val, ok := args[key]
	if !ok {
		return defaultValue
	}

	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		// Try parsing as float first
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return int(f)
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// getUint32 extracts a uint32 from request - handles int, float64, and string
func getUint32(request mcp.CallToolRequest, key string, defaultValue uint32) uint32 {
	args := request.GetArguments()
	val, ok := args[key]
	if !ok {
		return defaultValue
	}

	switch v := val.(type) {
	case int:
		if v < 0 {
			return defaultValue
		}
		return uint32(v)
	case int64:
		if v < 0 {
			return defaultValue
		}
		return uint32(v)
	case float64:
		if v < 0 {
			return defaultValue
		}
		return uint32(v)
	case string:
		if i, err := strconv.ParseUint(v, 10, 32); err == nil {
			return uint32(i)
		}
		// Try parsing as float first
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			if f >= 0 {
				return uint32(f)
			}
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// getUIDs extracts UIDs from request - handles both int and string arrays
func getUIDs(request mcp.CallToolRequest) ([]uint32, error) {
	// Try getting as int slice first (what AI typically sends)
	intUIDs := request.GetIntSlice("uids", nil)
	if len(intUIDs) > 0 {
		uids := make([]uint32, len(intUIDs))
		for i, v := range intUIDs {
			if v < 0 {
				return nil, fmt.Errorf("invalid UID (negative): %d", v)
			}
			uids[i] = uint32(v)
		}
		return uids, nil
	}

	// Try getting as float slice (JSON numbers are float64)
	floatUIDs := request.GetFloatSlice("uids", nil)
	if len(floatUIDs) > 0 {
		uids := make([]uint32, len(floatUIDs))
		for i, v := range floatUIDs {
			if v < 0 {
				return nil, fmt.Errorf("invalid UID (negative): %v", v)
			}
			uids[i] = uint32(v)
		}
		return uids, nil
	}

	// Fallback to string slice
	strUIDs := request.GetStringSlice("uids", nil)
	if len(strUIDs) > 0 {
		uids := make([]uint32, 0, len(strUIDs))
		for _, v := range strUIDs {
			var uid uint32
			if _, err := fmt.Sscanf(v, "%d", &uid); err != nil {
				return nil, fmt.Errorf("invalid UID: %v", v)
			}
			uids = append(uids, uid)
		}
		return uids, nil
	}

	// Try raw arguments as last resort
	args := request.GetArguments()
	if rawUIDs, ok := args["uids"]; ok {
		if arr, ok := rawUIDs.([]interface{}); ok {
			uids := make([]uint32, 0, len(arr))
			for _, item := range arr {
				switch v := item.(type) {
				case float64:
					if v < 0 {
						return nil, fmt.Errorf("invalid UID (negative): %v", v)
					}
					uids = append(uids, uint32(v))
				case int:
					if v < 0 {
						return nil, fmt.Errorf("invalid UID (negative): %d", v)
					}
					uids = append(uids, uint32(v))
				case int64:
					if v < 0 {
						return nil, fmt.Errorf("invalid UID (negative): %d", v)
					}
					uids = append(uids, uint32(v))
				case string:
					var uid uint32
					if _, err := fmt.Sscanf(v, "%d", &uid); err != nil {
						return nil, fmt.Errorf("invalid UID: %v", v)
					}
					uids = append(uids, uid)
				default:
					return nil, fmt.Errorf("invalid UID type: %T", item)
				}
			}
			if len(uids) > 0 {
				return uids, nil
			}
		}
	}

	return nil, nil
}

// getStringArray extracts string array - handles mixed input
func getStringArray(request mcp.CallToolRequest, key string) []string {
	// First try the standard method
	result := request.GetStringSlice(key, nil)
	if len(result) > 0 {
		return result
	}

	// Try raw arguments
	args := request.GetArguments()
	if raw, ok := args[key]; ok {
		if arr, ok := raw.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				switch v := item.(type) {
				case string:
					result = append(result, v)
				case float64:
					result = append(result, fmt.Sprintf("%v", v))
				case int:
					result = append(result, fmt.Sprintf("%d", v))
				default:
					result = append(result, fmt.Sprintf("%v", v))
				}
			}
			return result
		}
	}

	return nil
}
