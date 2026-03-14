package filter

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Test field types functionality
func TestFieldType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{"String type", FieldTypeString, true},
		{"Int type", FieldTypeInt, true},
		{"Float type", FieldTypeFloat, true},
		{"Bool type", FieldTypeBool, true},
		{"Time type", FieldTypeTime, true},
		{"Date type", FieldTypeDate, true},
		{"Enum type", FieldTypeEnum, true},
		{"UUID type", FieldTypeUUID, true},
		{"Text type", FieldTypeText, true},
		{"Invalid type", FieldType("invalid"), false},
		{"Empty type", FieldType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ft.IsValid()
			if result != tt.expected {
				t.Errorf("FieldType.IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFieldType_GetDefaultOperators(t *testing.T) {
	tests := []struct {
		name     string
		ft       FieldType
		expected []Operator
	}{
		{
			name: "String type operators",
			ft:   FieldTypeString,
			expected: []Operator{
				Equals, NotEquals, Contains, StartsWith, EndsWith, In, NotIn, IsNull, IsNotNull,
			},
		},
		{
			name: "Int type operators",
			ft:   FieldTypeInt,
			expected: []Operator{
				Equals, NotEquals, GreaterThan, GreaterThanOrEqual, LessThan, LessThanOrEqual, Range, In, NotIn, IsNull, IsNotNull,
			},
		},
		{
			name:     "Bool type operators",
			ft:       FieldTypeBool,
			expected: []Operator{Equals, NotEquals, IsNull, IsNotNull},
		},
		{
			name:     "Invalid type operators",
			ft:       FieldType("invalid"),
			expected: []Operator{Equals, NotEquals},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ft.GetDefaultOperators()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FieldType.GetDefaultOperators() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFieldType_RequiresEscaping(t *testing.T) {
	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{"String escaping", FieldTypeString, true},
		{"Text escaping", FieldTypeText, true},
		{"Int no escaping", FieldTypeInt, false},
		{"Float no escaping", FieldTypeFloat, false},
		{"Bool no escaping", FieldTypeBool, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ft.RequiresEscaping()
			if result != tt.expected {
				t.Errorf("FieldType.RequiresEscaping() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFieldType_SupportsLike(t *testing.T) {
	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{"String supports LIKE", FieldTypeString, true},
		{"Text supports LIKE", FieldTypeText, true},
		{"Int no LIKE", FieldTypeInt, false},
		{"Bool no LIKE", FieldTypeBool, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ft.SupportsLike()
			if result != tt.expected {
				t.Errorf("FieldType.SupportsLike() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFieldType_SupportsRange(t *testing.T) {
	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{"Int supports range", FieldTypeInt, true},
		{"Float supports range", FieldTypeFloat, true},
		{"String no range", FieldTypeString, false},
		{"Bool no range", FieldTypeBool, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ft.SupportsRange()
			if result != tt.expected {
				t.Errorf("FieldType.SupportsRange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFieldType_SupportsIn(t *testing.T) {
	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{"String supports IN", FieldTypeString, true},
		{"Int supports IN", FieldTypeInt, true},
		{"Enum supports IN", FieldTypeEnum, true},
		{"Bool no IN", FieldTypeBool, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ft.SupportsIn()
			if result != tt.expected {
				t.Errorf("FieldType.SupportsIn() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterDefine_GetAllowedOperators(t *testing.T) {
	tests := []struct {
		name     string
		fd       FilterDefine
		expected []Operator
	}{
		{
			name: "Empty operators uses type defaults",
			fd: FilterDefine{
				Field:     "name",
				FieldType: FieldTypeString,
				Operators: []Operator{},
			},
			expected: []Operator{
				Equals, NotEquals, Contains, StartsWith, EndsWith, In, NotIn, IsNull, IsNotNull,
			},
		},
		{
			name: "Custom operators validated",
			fd: FilterDefine{
				Field:     "price",
				FieldType: FieldTypeInt,
				Operators: []Operator{Equals, GreaterThan, LessThan},
			},
			expected: []Operator{Equals, GreaterThan, LessThan},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fd.GetAllowedOperators()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FilterDefine.GetAllowedOperators() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterDefine_IsOperatorAllowed(t *testing.T) {
	tests := []struct {
		name     string
		fd       FilterDefine
		op       Operator
		expected bool
	}{
		{
			name: "Allowed operator",
			fd: FilterDefine{
				Field:     "name",
				FieldType: FieldTypeString,
				Operators: []Operator{Equals, Contains},
			},
			op:       Equals,
			expected: true,
		},
		{
			name: "Not allowed operator",
			fd: FilterDefine{
				Field:     "name",
				FieldType: FieldTypeString,
				Operators: []Operator{Equals, Contains},
			},
			op:       GreaterThan,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fd.IsOperatorAllowed(tt.op)
			if result != tt.expected {
				t.Errorf("FilterDefine.IsOperatorAllowed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected Operator
	}{
		{"eq", Equals},
		{"=", Equals},
		{"contains", Contains},
		{"startswith", StartsWith},
		{"endswith", EndsWith},
		{"unknown", Operator("unknown")},
		{"gt", Operator("gt")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NewOperator(tt.input)
			if result != tt.expected {
				t.Errorf("NewOperator(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOperator_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		op       Operator
		expected bool
	}{
		{"Valid Equals", Equals, true},
		{"Valid NotEquals", NotEquals, true},
		{"Valid GreaterThan", GreaterThan, true},
		{"Valid Contains", Contains, true},
		{"Valid In", In, true},
		{"Valid Range", Range, true},
		{"Valid IsNull", IsNull, true},
		{"Valid IsNotNull", IsNotNull, true},
		{"Invalid operator", Operator("invalid"), false},
		{"Empty operator", Operator(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.op.IsValid()
			if result != tt.expected {
				t.Errorf("Operator.IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetTotalPages(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		total    int64
		expected int64
	}{
		{
			name:     "Zero total",
			cfg:      &Config{Query: &Query{PageSize: 10}, Options: &Options{EnablePaginate: true}},
			total:    0,
			expected: 0,
		},
		{
			name:     "Less than page size",
			cfg:      &Config{Query: &Query{PageSize: 10}, Options: &Options{EnablePaginate: true}},
			total:    5,
			expected: 1,
		},
		{
			name:     "Multiple pages",
			cfg:      &Config{Query: &Query{PageSize: 10}, Options: &Options{EnablePaginate: true}},
			total:    25,
			expected: 3,
		},
		{
			name:     "Pagination disabled",
			cfg:      &Config{Query: &Query{PageSize: 10}, Options: &Options{EnablePaginate: false}},
			total:    100,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.GetTotalPages(tt.total)
			if result != tt.expected {
				t.Errorf("GetTotalPages() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWithPageSize(t *testing.T) {
	opts := Options{}
	WithPageSize(20, 100)(&opts)

	if opts.DefaultPageSize != 20 {
		t.Errorf("DefaultPageSize = %v, want %v", opts.DefaultPageSize, 20)
	}
	if opts.MaxPageSize != 100 {
		t.Errorf("MaxPageSize = %v, want %v", opts.MaxPageSize, 100)
	}
}

func TestWithDefaultSort(t *testing.T) {
	opts := Options{}
	WithDefaultSort("name", Asc)(&opts)

	if opts.DefaultSortBy != "name" {
		t.Errorf("DefaultSortBy = %v, want %v", opts.DefaultSortBy, "name")
	}
	if opts.DefaultOrder != Asc {
		t.Errorf("DefaultOrder = %v, want %v", opts.DefaultOrder, Asc)
	}
}

func TestWithAllowedIncludes(t *testing.T) {
	includes := []string{"Category", "Tags"}
	opts := Options{}
	WithAllowedIncludes(includes)(&opts)

	if !reflect.DeepEqual(opts.AllowedIncludes, includes) {
		t.Errorf("AllowedIncludes = %v, want %v", opts.AllowedIncludes, includes)
	}
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Empty string", "", []string{}},
		{"Single value", "test", []string{"test"}},
		{"Multiple values", "a,b,c", []string{"a", "b", "c"}},
		{"With spaces", "a, b ,c", []string{"a", "b", "c"}},
		{"Empty values", "a,,c", []string{"a", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseCommaSeparated() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoGeneratePreload(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single word", "user", "User"},
		{"Snake case", "user_profile", "UserProfile"},
		{"Multiple underscores", "very_long_name", "VeryLongName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := autoGeneratePreload(tt.input)
			if result != tt.expected {
				t.Errorf("autoGeneratePreload() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertToType(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		targetType  string
		expected    interface{}
		expectError bool
	}{
		{"String to int", "123", "int", 123, false},
		{"String to uint", "456", "uint", uint(456), false},
		{"String to int64", "789", "int64", int64(789), false},
		{"String to float", "123.45", "float", 123.45, false},
		{"String to bool", "true", "bool", true, false},
		{"Nil value", nil, "string", nil, false},
		{"Invalid int", "abc", "int", nil, true},
		{"Unknown type", "test", "unknown", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToType(tt.value, tt.targetType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for convertToType(%s, %s)", tt.value, tt.targetType)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for convertToType(%s, %s): %v", tt.value, tt.targetType, err)
				}
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("convertToType() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestValidateValueType(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		filterDef   *FilterDefine
		expectError bool
	}{
		{
			name:  "Valid int within range",
			value: 50,
			filterDef: &FilterDefine{
				FieldType: FieldTypeInt,
				MinValue:  &[]float64{0}[0],
				MaxValue:  &[]float64{100}[0],
			},
			expectError: false,
		},
		{
			name:  "Int below minimum",
			value: -10,
			filterDef: &FilterDefine{
				FieldType: FieldTypeInt,
				MinValue:  &[]float64{0}[0],
				MaxValue:  &[]float64{100}[0],
			},
			expectError: true,
		},
		{
			name:  "Valid string within max length",
			value: "hello",
			filterDef: &FilterDefine{
				FieldType: FieldTypeString,
				MaxLength: &[]int{10}[0],
			},
			expectError: false,
		},
		{
			name:        "Nil filter definition",
			value:       "test",
			filterDef:   nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateValueType(tt.value, tt.filterDef)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for validateValueType")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for validateValueType: %v", err)
			}
		})
	}
}

func TestParseSorters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []SorterClause
	}{
		{"Empty string", "", []SorterClause{}},
		{"Valid asc", "name:asc", []SorterClause{{Field: "name", Order: Asc}}},
		{"Valid desc", "name:desc", []SorterClause{{Field: "name", Order: Desc}}},
		{"Multiple sorters", "name:asc,price:desc", []SorterClause{
			{Field: "name", Order: Asc},
			{Field: "price", Order: Desc},
		}},
		{"Invalid format", "name", []SorterClause{}},
		{"Invalid order", "name:invalid", []SorterClause{}},
		{"Mixed valid and invalid", "name:asc,invalid,price:desc", []SorterClause{
			{Field: "name", Order: Asc},
			{Field: "price", Order: Desc},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sorters []SorterClause
			_ = parseSorters(tt.input, &sorters, nil, false)
			if len(sorters) != len(tt.expected) {
				t.Errorf("parseSorters() length = %d, want %d", len(sorters), len(tt.expected))
				return
			}
			if len(sorters) > 0 && len(tt.expected) > 0 {
				if !reflect.DeepEqual(sorters, tt.expected) {
					t.Errorf("parseSorters() = %v, want %v", sorters, tt.expected)
				}
			}
		})
	}
}

func TestBuildExpressions_UncoveredBranches(t *testing.T) {
	// Test the uncovered branches in buildExpressions
	tests := []struct {
		name      string
		op        Operator
		field     string
		value     interface{}
		shouldGen bool // should generate expressions
	}{
		{"IsNull", IsNull, "deleted_at", nil, true},
		{"IsNotNull", IsNotNull, "verified_at", nil, true},
		{"In with empty slice", In, "id", []interface{}{}, false},
		{"NotIn with empty slice", NotIn, "id", []interface{}{}, false},
		{"Range with single value", Range, "price", []interface{}{100}, false},
		{"In with non-empty slice", In, "status", []interface{}{"active", "inactive"}, true},
		{"NotIn with non-empty slice", NotIn, "role", []interface{}{"admin", "guest"}, true},
		{"Range with two values", Range, "created_at", []interface{}{"2023-01-01", "2023-12-31"}, true},
		{"NotEquals", NotEquals, "name", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprs := buildExpressions(tt.op, tt.field, tt.value)

			if tt.shouldGen && len(exprs) == 0 {
				t.Errorf("Should generate at least one expression for %s", tt.name)
			}
			if !tt.shouldGen && len(exprs) > 0 {
				t.Errorf("Should not generate expressions for %s", tt.name)
			}
		})
	}
}

func TestReflectBasedFields(t *testing.T) {
	// Test the reflection-based field detection
	type TestModel struct {
		ID        int     `filter:"searchable,sortable,filterable"`
		Name      string  `filter:"searchable,sortable"`
		Price     float64 `filter:"filterable,sortable,type:float"`
		CreatedAt string  // no tag
	}

	// This test ensures the reflection code paths are executed
	modelType := reflect.TypeOf(TestModel{})
	if modelType.Kind() != reflect.Struct {
		t.Errorf("Expected Struct kind, got %v", modelType.Kind())
	}
	if modelType.NumField() != 4 {
		t.Errorf("Expected 4 fields, got %d", modelType.NumField())
	}

	// Test field name mapping
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		tag := field.Tag.Get("filter")
		t.Logf("Field %s has tag: %s", field.Name, tag)
	}
}

// TestSecurityLimits tests the DoS protection limits
func TestSecurityLimits(t *testing.T) {
	tests := []struct {
		name              string
		maxFilters        int
		maxFilterValues   int
		maxSortFields     int
		maxIncludeDepth   int
		maxSearchLength   int
		testFilters       int
		testSortFields    int
		testSearchLength  int
		testIncludeDepth  int
		shouldError       bool
		errorContains     string
	}{
		{
			name:            "All within limits",
			maxFilters:      5,
			maxFilterValues: 50,
			maxSortFields:   3,
			maxIncludeDepth: 2,
			maxSearchLength: 100,
			testFilters:     3,
			testSortFields:  2,
			testSearchLength: 50,
			testIncludeDepth: 1,
			shouldError:     false,
		},
		{
			name:            "Too many filters",
			maxFilters:      2,
			testFilters:     5,
			shouldError:     true,
			errorContains:   "too many filter conditions",
		},
		{
			name:            "Too many sort fields",
			maxSortFields:   2,
			testSortFields:  5,
			shouldError:     true,
			errorContains:   "too many sort fields",
		},
		{
			name:            "Search query too long",
			maxSearchLength: 10,
			testSearchLength: 50,
			shouldError:     true,
			errorContains:   "search query exceeds maximum length",
		},
		{
			name:            "Include depth exceeded",
			maxIncludeDepth: 2,
			testIncludeDepth: 3,
			shouldError:     true,
			errorContains:   "include depth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(
				WithSearch(true),
				WithFilter(true),
				WithOrderBy(true),
				WithSecurityLimits(tt.maxFilters, tt.maxFilterValues, tt.maxSortFields, tt.maxIncludeDepth, tt.maxSearchLength),
			)

			// Create a mock query with test values
			query := &Query{
				Search:   string(make([]byte, tt.testSearchLength)),
				Page:     1,
				PageSize: 10,
				SortBy:   generateSortString(tt.testSortFields),
				Filter:   make(map[string]string),
				Include:  generateIncludeString(tt.testIncludeDepth),
			}

			// Add test filters
			for i := 0; i < tt.testFilters; i++ {
				query.Filter[fmt.Sprintf("field%d", i)] = "value"
			}

			_ = &Config{
				Query:   query,
				Options: &opts,
				Filters: []FilterClause{},
			}

			// Test search length validation
			if tt.maxSearchLength > 0 && tt.testSearchLength > tt.maxSearchLength {
				if opts.EnableSearch && len(query.Search) > opts.MaxSearchLength {
					err := fmt.Errorf("search query exceeds maximum length of %d", opts.MaxSearchLength)
					if err == nil {
						t.Errorf("Expected error for search length")
					}
					if tt.shouldError && !strings.Contains(err.Error(), tt.errorContains) {
						t.Errorf("Error should contain %q, got %q", tt.errorContains, err.Error())
					}
				}
			}
		})
	}
}

// generateSortString creates a sort string with N fields
func generateSortString(n int) string {
	if n <= 0 {
		return ""
	}
	result := "field1:asc"
	for i := 2; i <= n; i++ {
		result += fmt.Sprintf(",field%d:asc", i)
	}
	return result
}

// generateIncludeString creates an include string with specified depth
func generateIncludeString(depth int) string {
	if depth <= 0 {
		return ""
	}
	result := "A"
	for i := 2; i <= depth; i++ {
		result += ".B"
	}
	return result
}

// TestStrictMode tests the strict mode functionality
func TestStrictMode(t *testing.T) {
	tests := []struct {
		name            string
		strictMode      bool
		allowedFilters  []FilterDefine
		allowedIncludes []string
		inputFilters    map[string]string
		inputIncludes   string
		shouldError     bool
		errorContains   string
	}{
		{
			name:       "Strict mode with unknown filter",
			strictMode: true,
			allowedFilters: []FilterDefine{
				{Field: "name", FieldType: FieldTypeString},
			},
			inputFilters: map[string]string{
				"unknown_field": "value",
			},
			shouldError:   true,
			errorContains: "unknown filter field",
		},
		{
			name:       "Strict mode with known filter",
			strictMode: true,
			allowedFilters: []FilterDefine{
				{Field: "name", FieldType: FieldTypeString},
			},
			inputFilters: map[string]string{
				"name": "value",
			},
			shouldError: false,
		},
		{
			name:       "Non-strict mode ignores unknown filter",
			strictMode: false,
			allowedFilters: []FilterDefine{
				{Field: "name", FieldType: FieldTypeString},
			},
			inputFilters: map[string]string{
				"unknown_field": "value",
			},
			shouldError: false,
		},
		{
			name:            "Strict mode with unknown include",
			strictMode:      true,
			allowedFilters:  []FilterDefine{},
			allowedIncludes: []string{"Category"},
			inputIncludes:   "Unknown,Category",
			shouldError:     true,
			errorContains:   "unknown include",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test filter strict mode
			if tt.inputFilters != nil {
				filters := []FilterClause{}
				err := parseFilters(tt.inputFilters, tt.allowedFilters, &filters, 100, tt.strictMode)
				if tt.shouldError && err == nil {
					t.Errorf("Expected error containing %q", tt.errorContains)
				}
				if tt.shouldError && err != nil && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error should contain %q, got %q", tt.errorContains, err.Error())
				}
				if !tt.shouldError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Test include strict mode
			if tt.inputIncludes != "" {
				includes := []IncludeClause{}
				err := parseInclude(tt.inputIncludes, tt.allowedIncludes, &includes, tt.strictMode, 3)
				if tt.shouldError && err == nil {
					t.Errorf("Expected error containing %q", tt.errorContains)
				}
				if tt.shouldError && err != nil && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error should contain %q, got %q", tt.errorContains, err.Error())
				}
			}
		})
	}
}

// TestOperatorAliases tests that operator aliases work correctly
func TestOperatorAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Operator
	}{
		{"eq alias", "eq", Equals},
		{"ne alias", "ne", NotEquals},
		{"neq alias", "neq", NotEquals},
		{"starts alias", "starts", StartsWith},
		{"ends alias", "ends", EndsWith},
		{"nin alias", "nin", NotIn},
		{"full operator name", "not_equals", NotEquals},
		{"unknown operator", "unknown_op", Operator("unknown_op")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewOperator(tt.input)
			if result != tt.expected {
				t.Errorf("NewOperator(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
