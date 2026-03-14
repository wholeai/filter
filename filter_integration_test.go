package filter

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Test FilterByQuery and parsing functions that still have 0% coverage
func TestFilterByQuery_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with basic query parameters
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set query parameters
	c.Request = httptest.NewRequest("GET", "/products?page=2&page_size=20&search=test&sort_by=name:desc&include=Category&filter[price]=gt:100", nil)

	opts := NewOptions(
		WithAllowedFilters([]FilterDefine{
			{Field: "price", FieldType: FieldTypeInt, Operators: []Operator{GreaterThan, LessThan}},
		}),
		WithAllowedIncludes([]string{"Category", "Tags"}),
		WithSearchFields([]string{"name", "description"}),
	)

	cfg, err := FilterByQuery(c, opts)
	if err != nil {
		t.Fatalf("FilterByQuery failed: %v", err)
	}

	// Verify parsed values
	if cfg.Page != 2 {
		t.Errorf("Expected page 2, got %d", cfg.Page)
	}
	if cfg.PageSize != 20 {
		t.Errorf("Expected page size 20, got %d", cfg.PageSize)
	}
	if cfg.Search != "test" {
		t.Errorf("Expected search 'test', got %s", cfg.Search)
	}
	if len(cfg.Sorters) == 0 || cfg.Sorters[0].Field != "name" || cfg.Sorters[0].Order != Desc {
		t.Errorf("Expected name:desc sorter, got %v", cfg.Sorters)
	}
}

func TestFilterByQuery_ParseFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		queryParams     string
		expectedFilters []FilterClause
		expectError     bool
	}{
		{
			name:        "Valid single filter",
			queryParams: "filter[name]=equals:test",
			expectedFilters: []FilterClause{
				{Field: "name", Operator: Equals, Value: "test", Raw: "equals:test"},
			},
			expectError: false,
		},
		{
			name:        "Valid operator filter",
			queryParams: "filter[price]=gt:100",
			expectedFilters: []FilterClause{
				{Field: "price", Operator: GreaterThan, Value: 100, Raw: "gt:100"},
			},
			expectError: false,
		},
		{
			name:        "IN filter with multiple values",
			queryParams: "filter[status]=in:active,inactive",
			expectedFilters: []FilterClause{
				{Field: "status", Operator: In, Value: []interface{}{"active", "inactive"}, Raw: "in:active,inactive"},
			},
			expectError: false,
		},
		{
			name:        "Range filter",
			queryParams: "filter[created_at]=range:2023-01-01,2023-12-31",
			expectedFilters: []FilterClause{
				{Field: "created_at", Operator: Range, Value: []interface{}{"2023-01-01", "2023-12-31"}, Raw: "range:2023-01-01,2023-12-31"},
			},
			expectError: false,
		},
		{
			name:        "Null filter",
			queryParams: "filter[deleted_at]=is_null:",
			expectedFilters: []FilterClause{
				{Field: "deleted_at", Operator: IsNull, Value: nil, Raw: "is_null:"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test?"+tt.queryParams, nil)

			opts := NewOptions()
			cfg, err := FilterByQuery(c, opts)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(cfg.Filters) != len(tt.expectedFilters) {
					t.Errorf("Expected %d filters, got %d", len(tt.expectedFilters), len(cfg.Filters))
					return
				}
				for i, expected := range tt.expectedFilters {
					if cfg.Filters[i].Field != expected.Field {
						t.Errorf("Filter %d field: expected %s, got %s", i, expected.Field, cfg.Filters[i].Field)
					}
					if cfg.Filters[i].Operator != expected.Operator {
						t.Errorf("Filter %d operator: expected %v, got %v", i, expected.Operator, cfg.Filters[i].Operator)
					}
				}
			}
		})
	}
}

func TestFilterByQuery_ParseFilters_ErrorCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryParams string
		errorMsg    string
	}{
		{
			name:        "Invalid operator",
			queryParams: "filter[price]=invalid:100",
			errorMsg:    "invalid: INVALID_OPERATOR",
		},
		{
			name:        "Invalid range format",
			queryParams: "filter[price]=range:100",
			errorMsg:    "range: INVALID_VALUE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test?"+tt.queryParams, nil)

			opts := NewOptions()
			cfg, err := FilterByQuery(c, opts)

			if err == nil {
				t.Error("Expected error but got none")
			} else if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
			}
			if cfg != nil && len(cfg.Filters) > 0 {
				t.Error("Expected no filters parsed due to error")
			}
		})
	}
}

func TestFilterByQuery_PageSizeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		queryParams  string
		expectedSize int
	}{
		{
			name:         "Within limits",
			queryParams:  "page_size=50",
			expectedSize: 50,
		},
		{
			name:         "Above max limit",
			queryParams:  "page_size=200",
			expectedSize: 100, // Default max
		},
		{
			name:         "Zero page size",
			queryParams:  "page_size=0",
			expectedSize: 10, // Default
		},
		{
			name:         "Negative page size",
			queryParams:  "page_size=-5",
			expectedSize: 10, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test?"+tt.queryParams, nil)

			opts := NewOptions(
				WithPageSize(10, 100), // Default 10, Max 100
			)
			cfg, err := FilterByQuery(c, opts)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if cfg.PageSize != tt.expectedSize {
				t.Errorf("Expected page size %d, got %d", tt.expectedSize, cfg.PageSize)
			}
		})
	}
}

func TestFilterByQuery_DefaultSortApplied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil) // No sort parameters

	opts := NewOptions(
		WithDefaultSort("created_at", Desc),
		WithOrderBy(true),
	)

	cfg, err := FilterByQuery(c, opts)
	if err != nil {
		t.Fatalf("FilterByQuery failed: %v", err)
	}

	// Should have default sort
	if len(cfg.Sorters) != 1 {
		t.Fatalf("Expected 1 default sorter, got %d", len(cfg.Sorters))
	}

	sorter := cfg.Sorters[0]
	if sorter.Field != "created_at" {
		t.Errorf("Expected default sort field 'created_at', got %s", sorter.Field)
	}
	if sorter.Order != Desc {
		t.Errorf("Expected default sort order 'desc', got %s", sorter.Order)
	}
}

func TestFilterByQuery_IncludeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test?include=Category,Secret,Tags", nil)

	opts := NewOptions(
		WithAllowedIncludes([]string{"Category", "Tags"}), // Secret is not allowed
	)

	cfg, err := FilterByQuery(c, opts)
	if err != nil {
		t.Fatalf("FilterByQuery failed: %v", err)
	}

	// Should only have allowed includes
	if len(cfg.Includes) != 2 {
		t.Errorf("Expected 2 includes, got %d", len(cfg.Includes))
	}

	includeNames := make([]string, len(cfg.Includes))
	for i, inc := range cfg.Includes {
		includeNames[i] = inc.Name
	}

	expectedIncludes := []string{"Category", "Tags"}
	for i, expected := range expectedIncludes {
		if includeNames[i] != expected {
			t.Errorf("Include %d: expected %s, got %s", i, expected, includeNames[i])
		}
	}
}

func TestParseFilterValue_ComplexCases(t *testing.T) {
	tests := []struct {
		name        string
		valueStr    string
		filterDef   *FilterDefine
		expectOp    Operator
		expectVal   interface{}
		expectError bool
	}{
		{
			name:     "String with enum validation",
			valueStr: "active",
			filterDef: &FilterDefine{
				FieldType:  FieldTypeEnum,
				EnumValues: []string{"active", "inactive"},
			},
			expectOp:  Equals,
			expectVal: "active",
		},
		{
			name:     "String with max length validation",
			valueStr: "hello",
			filterDef: &FilterDefine{
				FieldType: FieldTypeString,
				MaxLength: &[]int{10}[0],
			},
			expectOp:  Equals,
			expectVal: "hello",
		},
		{
			name:     "Comma-separated defaults to IN",
			valueStr: "a,b,c",
			filterDef: &FilterDefine{
				FieldType: FieldTypeString,
			},
			expectOp:  In,
			expectVal: []interface{}{"a", "b", "c"},
		},
		{
			name:     "Empty operator defaults to Equals",
			valueStr: "test",
			filterDef: &FilterDefine{
				FieldType: FieldTypeString,
			},
			expectOp:  Equals,
			expectVal: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, val, err := parseFilterValue(tt.valueStr, tt.filterDef, 100)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if op != tt.expectOp {
					t.Errorf("Expected operator %v, got %v", tt.expectOp, op)
				}
				if !compareValues(val, tt.expectVal) {
					t.Errorf("Expected value %v, got %v", tt.expectVal, val)
				}
			}
		})
	}

	// Test error case
	t.Run("Invalid enum value", func(t *testing.T) {
		op, val, err := parseFilterValue("deleted", &FilterDefine{
			FieldType:  FieldTypeEnum,
			EnumValues: []string{"active", "inactive"},
		}, 100)
		if err == nil {
			t.Error("Expected error for invalid enum value")
		}
		if op != "" || val != nil {
			t.Error("Expected empty op and nil val on error")
		}
	})
}

// Helper function to compare interface{} values
func compareValues(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle slices
	if aSlice, ok := a.([]interface{}); ok {
		if bSlice, ok := b.([]interface{}); ok {
			if len(aSlice) != len(bSlice) {
				return false
			}
			for i := range aSlice {
				if aSlice[i] != bSlice[i] {
					return false
				}
			}
			return true
		}
	}

	return a == b
}
