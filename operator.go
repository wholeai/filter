package filter

// Operator represents a filter operator type
type Operator string

const (
	// Comparison operators
	Equals             Operator = "equals"
	NotEquals          Operator = "not_equals"
	GreaterThan        Operator = "gt"
	GreaterThanOrEqual Operator = "gte"
	LessThan           Operator = "lt"
	LessThanOrEqual    Operator = "lte"

	// String operators
	Contains   Operator = "contains"
	StartsWith Operator = "starts_with"
	EndsWith   Operator = "ends_with"

	// Collection operators
	In    Operator = "in"
	NotIn Operator = "not_in"

	// Range operator
	Range Operator = "range"

	// Null operators
	IsNull    Operator = "is_null"
	IsNotNull Operator = "is_not_null"
)

func NewOperator(op string) Operator {
	switch op {
	case "eq", "=", string(Equals):
		return Equals
	case "neq", "!=", "ne", string(NotEquals):
		return NotEquals
	case "like", "likes", string(Contains):
		return Contains
	case "startswith", "starts", string(StartsWith):
		return StartsWith
	case "endswith", "ends", string(EndsWith):
		return EndsWith
	case "nin", string(NotIn):
		return NotIn
	}
	return Operator(op)
}

// IsValid checks if the operator is valid
func (o Operator) IsValid() bool {
	switch o {
	case Equals, NotEquals, GreaterThan, GreaterThanOrEqual, LessThan, LessThanOrEqual,
		Contains, StartsWith, EndsWith,
		In, NotIn, Range,
		IsNull, IsNotNull:
		return true
	default:
		return false
	}
}

// SortDirection represents the sort direction
type SortDirection string

const (
	Asc  SortDirection = "asc"
	Desc SortDirection = "desc"
)

type SearchMode string

const (
	SearchModeContains   SearchMode = "contains"
	SearchModeExact      SearchMode = "exact"
	SearchModeStartsWith SearchMode = "start"
	SearchModeEndsWith   SearchMode = "end"
)
