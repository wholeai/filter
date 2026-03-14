package filter

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

type Query struct {
	Search   string            `form:"search"`
	Page     int               `form:"page,default=1"`
	PageSize int               `form:"page_size,default=10"`
	SortBy   string            `form:"sort_by"`
	Include  string            `form:"include"`
	Filter   map[string]string `form:"-"`
}

const (
	tagKey = "filter"
)

type Config struct {
	*Query
	Options  *Options
	Filters  []FilterClause
	Sorters  []SorterClause
	Includes []IncludeClause
}

type FilterClause struct {
	Field    string      // Field name from request
	Operator Operator    // Filter operator
	Raw      string      // Raw filter value string
	Value    interface{} // Filter value (can be single value, slice, or range)
}

type SorterClause struct {
	Field string        // Field name from request
	Order SortDirection // Sort order
}

type IncludeClause struct {
	Name    string // Include name
	Preload string // GORM preload
}

type FilterDefine struct {
	Field       string
	FieldType   FieldType  `json:"data_type"`
	Operators   []Operator // allowed operators, empty means default operators of the type
	Expression  string
	Params      []interface{}
	Description string
	Sortable    bool
	MinValue    *float64 `json:"min_value,omitempty"`
	MaxValue    *float64 `json:"max_value,omitempty"`
	MaxLength   *int     `json:"max_length,omitempty"`
	EnumValues  []string `json:"enum_values,omitempty"`
}

type Options struct {
	EnableSearch        bool
	EnableFilter        bool
	EnablePaginate      bool
	EnableOrderBy       bool
	DefaultPageSize     int
	MaxPageSize         int
	DefaultSortBy       string
	DefaultOrder        SortDirection
	SearchMode          SearchMode
	AllowedSearchFields []string

	// Includes related configuration
	AllowedIncludes []string // allowed include names
	// Filters related configuration
	AllowedFilters []FilterDefine // allowed filters

	// Security limits to prevent DoS
	MaxFilters      int // maximum number of filter conditions
	MaxFilterValues int // maximum number of values in IN/NotIn operators
	MaxSortFields   int // maximum number of sort fields
	MaxIncludeDepth int // maximum nesting depth for includes
	MaxSearchLength int // maximum length of search query

	// Strict mode for better error handling
	StrictMode bool // error on unknown filters/sorts/includes instead of silently ignoring
}

func NewOptions(withOptions ...func(*Options)) Options {
	options := Options{
		EnableSearch:        true,
		EnableFilter:        true,
		EnablePaginate:      true,
		EnableOrderBy:       true,
		MaxPageSize:         100,
		DefaultPageSize:     10,
		DefaultSortBy:       "id", // default sort by ID
		DefaultOrder:        Desc,
		SearchMode:          SearchModeContains,
		AllowedSearchFields: []string{},
		AllowedIncludes:     []string{},

		// Security defaults to prevent DoS
		MaxFilters:      20,  // reasonable limit for API queries
		MaxFilterValues: 100, // for IN/NotIn operators
		MaxSortFields:   5,   // multi-sort limit
		MaxIncludeDepth: 3,   // nested preload limit
		MaxSearchLength: 200, // character limit for search
	}

	for _, option := range withOptions {
		option(&options)
	}

	return options
}

func (cfg Config) GetTotalPages(total int64) int64 {
	if total <= 0 {
		return 0
	}
	if cfg.Options.EnablePaginate {
		if cfg.PageSize <= 0 {
			return 0
		}
		return int64(math.Ceil(float64(total) / float64(cfg.PageSize)))
	}
	return 1
}

func WithSearchFields(fields []string) func(*Options) {
	return func(options *Options) {
		options.AllowedSearchFields = fields
	}
}

func WithAllowedIncludes(allowedIncludes []string) func(*Options) {
	return func(options *Options) {
		options.AllowedIncludes = allowedIncludes
	}
}

func WithAllowedFilters(allowedFilters []FilterDefine) func(*Options) {
	return func(options *Options) {
		options.AllowedFilters = allowedFilters
	}
}

func WithSearchMode(mode SearchMode) func(*Options) {
	return func(options *Options) {
		options.SearchMode = mode
	}
}

func WithSearch(enable bool) func(*Options) {
	return func(options *Options) {
		options.EnableSearch = enable
	}
}

func WithFilter(enable bool) func(*Options) {
	return func(options *Options) {
		options.EnableFilter = enable
	}
}

func WithPaginate(enable bool) func(*Options) {
	return func(options *Options) {
		options.EnablePaginate = enable
	}
}

func WithOrderBy(enable bool) func(*Options) {
	return func(options *Options) {
		options.EnableOrderBy = enable
	}
}

func WithPageSize(defaultSize int, maxSize int) func(*Options) {
	return func(options *Options) {
		options.DefaultPageSize = defaultSize
		options.MaxPageSize = maxSize
	}
}

func WithDefaultSort(sortBy string, order SortDirection) func(*Options) {
	return func(options *Options) {
		options.DefaultSortBy = sortBy
		options.DefaultOrder = order
	}
}

// WithSecurityLimits sets DoS protection limits
func WithSecurityLimits(maxFilters, maxFilterValues, maxSortFields, maxIncludeDepth, maxSearchLength int) func(*Options) {
	return func(options *Options) {
		options.MaxFilters = maxFilters
		options.MaxFilterValues = maxFilterValues
		options.MaxSortFields = maxSortFields
		options.MaxIncludeDepth = maxIncludeDepth
		options.MaxSearchLength = maxSearchLength
	}
}

// WithStrictMode enables strict mode which errors on unknown filters/sorts/includes
func WithStrictMode(strict bool) func(*Options) {
	return func(options *Options) {
		options.StrictMode = strict
	}
}

func FilterByQuery(c *gin.Context, options Options) (cfg *Config, err error) {
	query := &Query{}
	err = c.BindQuery(query)
	if err != nil {
		return
	}
	query.Filter = c.QueryMap("filter")

	cfg = &Config{
		Query:   query,
		Options: &options,
		Filters: []FilterClause{},
	}

	// Security: validate search length
	if options.EnableSearch && len(query.Search) > options.MaxSearchLength {
		return nil, fmt.Errorf("search query exceeds maximum length of %d", options.MaxSearchLength)
	}

	if options.EnablePaginate {
		if query.PageSize > options.MaxPageSize {
			query.PageSize = options.MaxPageSize
		}

		if query.PageSize <= 0 {
			query.PageSize = options.DefaultPageSize
		}

		if query.Page <= 0 {
			query.Page = 1
		}
	}

	if options.EnableOrderBy {
		if err := parseSorters(query.SortBy, &cfg.Sorters, options.AllowedFilters, options.StrictMode); err != nil {
			return nil, err
		}
		if len(cfg.Sorters) == 0 {
			cfg.Sorters = []SorterClause{{
				Field: options.DefaultSortBy,
				Order: SortDirection(options.DefaultOrder),
			}}
		}
		// Security: enforce max sort fields
		if len(cfg.Sorters) > options.MaxSortFields {
			return nil, fmt.Errorf("too many sort fields, maximum is %d", options.MaxSortFields)
		}
	}

	// handle includes
	if len(query.Include) > 0 {
		if err := parseInclude(query.Include, options.AllowedIncludes, &cfg.Includes, options.StrictMode, options.MaxIncludeDepth); err != nil {
			return nil, err
		}
	}
	if options.EnableFilter {
		// Security: enforce max filters
		if len(query.Filter) > options.MaxFilters {
			return nil, fmt.Errorf("too many filter conditions, maximum is %d", options.MaxFilters)
		}
		// Parse filters
		if err := parseFilters(query.Filter, options.AllowedFilters, &cfg.Filters, options.MaxFilterValues, options.StrictMode); err != nil {
			return nil, err
		}
	}

	return
}

func parseInclude(includeStr string, allowedIncludes []string, includes *[]IncludeClause, strictMode bool, maxIncludeDepth int) error {
	// Iterate through all include names
	includeNames := parseCommaSeparated(includeStr)
	for _, includeName := range includeNames {
		if includeName == "" {
			continue
		}

		// Security: validate include depth (nesting level)
		depth := strings.Count(includeName, ".") + 1
		if maxIncludeDepth > 0 && depth > maxIncludeDepth {
			return fmt.Errorf("include depth %d exceeds maximum of %d for: %s", depth, maxIncludeDepth, includeName)
		}

		// Check if include is allowed
		allowed := false
		for _, allowedInclude := range allowedIncludes {
			if allowedInclude == includeName {
				allowed = true
				break
			}
		}
		if !allowed {
			if strictMode && len(allowedIncludes) > 0 {
				return fmt.Errorf("unknown include: %s (allowed: %v)", includeName, allowedIncludes)
			}
			continue
		}
		preload := autoGeneratePreload(includeName)
		// Add to includes
		*includes = append(*includes, IncludeClause{
			Name:    includeName,
			Preload: preload,
		})
	}
	return nil
}

// parseSorters parses sort parameters
func parseSorters(sort string, sorters *[]SorterClause, allowedFilters []FilterDefine, strictMode bool) error {
	// Iterate through all query parameters
	splitSorts := parseCommaSeparated(sort)
	for _, s := range splitSorts {
		// Check if it's a sort parameter
		if s == "" {
			continue
		}
		sortParts := strings.SplitN(s, ":", 2)
		if len(sortParts) != 2 {
			continue
		}
		field := sortParts[0]
		order := strings.ToLower(sortParts[1])
		if order != "asc" && order != "desc" {
			continue
		}

		// Strict mode: validate sort field is allowed
		if strictMode && len(allowedFilters) > 0 {
			allowed := false
			for _, def := range allowedFilters {
				if def.Field == field && def.Sortable {
					allowed = true
					break
				}
			}
			// Default sort field is always allowed
			if !allowed {
				return fmt.Errorf("unknown sort field: %s", field)
			}
		}

		if order == "asc" {
			*sorters = append(*sorters, SorterClause{
				Field: field,
				Order: Asc,
			})
		} else {
			*sorters = append(*sorters, SorterClause{
				Field: field,
				Order: Desc,
			})
		}
	}
	return nil
}

// parseFilters parses filter parameters
func parseFilters(filterMap map[string]string, filterDefs []FilterDefine, filters *[]FilterClause, maxFilterValues int, strictMode bool) error {
	// Iterate through all query parameters
	for field, valueStr := range filterMap {
		// Find filter definition
		var filterDef *FilterDefine
		for i := range filterDefs {
			if filterDefs[i].Field == field {
				filterDef = &filterDefs[i]
				break
			}
		}

		// Strict mode: error on unknown filters
		if strictMode && len(filterDefs) > 0 && filterDef == nil {
			return fmt.Errorf("unknown filter field: %s", field)
		}

		// Parse operator and value
		operator, value, err := parseFilterValue(valueStr, filterDef, maxFilterValues)
		if err != nil {
			return fmt.Errorf("invalid filter value for field %s: %w", field, err)
		}

		// Check if operator is allowed
		if filterDef != nil && !filterDef.IsOperatorAllowed(operator) {
			return fmt.Errorf("%s: %s for field %s", ErrCodeInvalidOperator, operator, field)
		}

		*filters = append(*filters, FilterClause{
			Field:    field,
			Operator: operator,
			Value:    value,
			Raw:      valueStr,
		})
	}

	return nil
}

// parseFilterValue parses a filter value string into operator and value
// Format: "operator:value" or just "value" (defaults to equals)
func parseFilterValue(valueStr string, filterDef *FilterDefine, maxFilterValues int) (Operator, interface{}, error) {
	if filterDef == nil {
		filterDef = &FilterDefine{}
	}
	// Check if value contains operator prefix
	parts := strings.SplitN(valueStr, ":", 2)

	var operator Operator
	var rawValue string

	if len(parts) == 2 {
		// Explicit operator - use NewOperator to handle aliases
		operator = NewOperator(strings.ToLower(parts[0]))
		rawValue = parts[1]

		if !operator.IsValid() {
			return "", nil, fmt.Errorf("%s: %s", operator, ErrCodeInvalidOperator)
		}
	} else {
		// No operator specified, default to equals
		if strings.Contains(valueStr, ",") {
			operator = In
		} else {
			operator = Equals
		}
		rawValue = valueStr
	}

	// Parse value based on operator
	var value interface{}

	switch operator {
	case IsNull, IsNotNull:
		// No value needed
		value = nil

	case In, NotIn:
		// Parse comma-separated values
		values := parseCommaSeparated(rawValue)
		// Security: enforce max filter values
		if maxFilterValues > 0 && len(values) > maxFilterValues {
			return "", nil, fmt.Errorf("too many values (got %d, maximum is %d)", len(values), maxFilterValues)
		}
		typedValues := make([]interface{}, len(values))
		for i, v := range values {
			converted, err := convertToType(v, string(filterDef.FieldType))
			if err != nil {
				return "", nil, fmt.Errorf("invalid value at index %d: %w", i, err)
			}
			typedValues[i] = converted
		}
		err := validateValueType(typedValues, filterDef)
		if err != nil {
			return "", nil, err
		}
		value = typedValues

	case Range:
		// Parse range values
		values := parseCommaSeparated(rawValue)
		if len(values) != 2 {
			return "", nil, fmt.Errorf("%s: %s", operator, ErrCodeInvalidValue)
		}
		fromVal, err := convertToType(values[0], string(filterDef.FieldType))
		if err != nil {
			return "", nil, fmt.Errorf("invalid range start value: %w", err)
		}
		toVal, err := convertToType(values[1], string(filterDef.FieldType))
		if err != nil {
			return "", nil, fmt.Errorf("invalid range end value: %w", err)
		}
		typedValues := make([]interface{}, 2)
		typedValues[0] = fromVal
		typedValues[1] = toVal
		err = validateValueType(typedValues, filterDef)
		if err != nil {
			return "", nil, err
		}
		value = typedValues
	default:
		var err error
		value, err = convertToType(rawValue, string(filterDef.FieldType))
		if err != nil {
			return "", nil, err
		}
		err = validateValueType(value, filterDef)
		if err != nil {
			return "", nil, err
		}
	}

	return operator, value, nil
}

func parseCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	parts := strings.Split(s, ",")
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// autoGeneratePreload generates GORM preload string from include name
// Converts snake_case include names to PascalCase for GORM
// Example: "user_profile" -> "UserProfile"
func autoGeneratePreload(includeName string) string {
	if len(includeName) == 0 {
		return includeName
	}

	// user_profile -> UserProfile
	words := strings.Split(includeName, "_")
	var result string
	for _, word := range words {
		if len(word) > 0 {
			result += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return result
}

// convertToType attempts to convert a string value to the appropriate type
func convertToType(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if values, ok := value.([]interface{}); ok {
		typedValues := make([]interface{}, len(values))
		for i, value := range values {
			val, err := convertToType(value, targetType)
			if err != nil {
				return nil, err
			}
			typedValues[i] = val
		}
		return typedValues, nil
	}
	switch targetType {
	case "int":
		n, err := cast.ToIntE(value)
		if err != nil {
			return nil, fmt.Errorf("cannot convert '%s' to int", value)
		}
		return n, nil

	case "uint":
		n, err := cast.ToUintE(value)
		if err != nil {
			return nil, fmt.Errorf("cannot convert '%s' to uint", value)
		}
		return n, nil

	case "int64":
		n, err := cast.ToInt64E(value)
		if err != nil {
			return nil, fmt.Errorf("cannot convert '%s' to int64", value)
		}
		return n, nil

	case "float":
		f, err := cast.ToFloat64E(value)
		if err != nil {
			return nil, fmt.Errorf("cannot convert '%s' to float", value)
		}
		return f, nil

	case "bool":
		b := cast.ToBool(value)
		return b, nil

	case "date":
		dt, err := cast.ToTimeE(value)
		if err != nil {
			return nil, fmt.Errorf("cannot convert '%s' to date: %w", value, err)
		}
		return dt.Format(time.DateOnly), nil

	case "time":
		dt, err := cast.ToTimeE(value)
		if err != nil {
			return nil, fmt.Errorf("cannot convert '%s' to time: %w", value, err)
		}
		return dt.Format(time.DateTime), nil
	case "string":
		s := cast.ToString(value)
		return s, nil
	default:
		return value, nil
	}
}

// validateValue checks if the value conforms to the constraints defined in FilterDefine
func validateValueType(value interface{}, filterDef *FilterDefine) error {
	if filterDef == nil {
		return nil
	}
	if values, ok := value.([]interface{}); ok {
		for _, value := range values {
			err := validateValueType(value, filterDef)
			if err != nil {
				return err
			}
		}
		return nil
	}
	switch filterDef.FieldType {
	case FieldTypeInt:
		num, err := cast.ToIntE(value)
		if err != nil {
			return fmt.Errorf("value is not an integer")
		}
		if filterDef.MinValue != nil && float64(num) < *filterDef.MinValue {
			return fmt.Errorf("value %d is less than minimum %f", num, *filterDef.MinValue)
		}
		if filterDef.MaxValue != nil && float64(num) > *filterDef.MaxValue {
			return fmt.Errorf("value %d is greater than maximum %f", num, *filterDef.MaxValue)
		}
	case FieldTypeFloat:
		num, err := cast.ToFloat64E(value)
		if err != nil {
			return fmt.Errorf("value is not a number")
		}
		if filterDef.MinValue != nil && num < *filterDef.MinValue {
			return fmt.Errorf("value %f is less than minimum %f", num, *filterDef.MinValue)
		}
		if filterDef.MaxValue != nil && num > *filterDef.MaxValue {
			return fmt.Errorf("value %f is greater than maximum %f", num, *filterDef.MaxValue)
		}
	case FieldTypeString, FieldTypeEnum, FieldTypeText, FieldTypeUUID:
		str := cast.ToString(value)
		if filterDef.MaxLength != nil && len(str) > *filterDef.MaxLength {
			return fmt.Errorf("string length %d exceeds maximum %d", len(str), *filterDef.MaxLength)
		}
		if len(filterDef.EnumValues) > 0 {
			valid := false
			for _, enumVal := range filterDef.EnumValues {
				if str == enumVal {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("value '%s' is not in allowed enum values", str)
			}
		}
	}

	return nil
}
