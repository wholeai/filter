package filter

// FieldType represents the data type of a field
type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeInt    FieldType = "int"
	FieldTypeFloat  FieldType = "float"
	FieldTypeBool   FieldType = "bool"
	FieldTypeTime   FieldType = "time"
	FieldTypeDate   FieldType = "date"
	FieldTypeEnum   FieldType = "enum"
	FieldTypeUUID   FieldType = "uuid"
	FieldTypeText   FieldType = "text"
)

// IsValid checks if the field type is valid
func (ft FieldType) IsValid() bool {
	switch ft {
	case FieldTypeString, FieldTypeInt, FieldTypeFloat, FieldTypeBool,
		FieldTypeTime, FieldTypeDate, FieldTypeEnum,
		FieldTypeUUID, FieldTypeText:
		return true
	default:
		return false
	}
}

// GetDefaultOperators returns the default allowed operators for a field type
func (ft FieldType) GetDefaultOperators() []Operator {
	switch ft {
	case FieldTypeString, FieldTypeText:
		return []Operator{Equals, NotEquals, Contains, StartsWith, EndsWith, In, NotIn, IsNull, IsNotNull}
	case FieldTypeInt, FieldTypeFloat:
		return []Operator{Equals, NotEquals, GreaterThan, GreaterThanOrEqual, LessThan, LessThanOrEqual, Range, In, NotIn, IsNull, IsNotNull}
	case FieldTypeBool:
		return []Operator{Equals, NotEquals, IsNull, IsNotNull}
	case FieldTypeTime, FieldTypeDate:
		return []Operator{Equals, NotEquals, GreaterThan, GreaterThanOrEqual, LessThan, LessThanOrEqual, Range, In, NotIn}
	case FieldTypeEnum:
		return []Operator{Equals, NotEquals, In, NotIn, IsNull, IsNotNull}
	case FieldTypeUUID:
		return []Operator{Equals, NotEquals, In, NotIn, IsNull, IsNotNull}
	default:
		return []Operator{Equals, NotEquals}
	}
}

// RequiresEscaping checks if the field type requires SQL escaping
func (ft FieldType) RequiresEscaping() bool {
	switch ft {
	case FieldTypeString, FieldTypeText, FieldTypeDate, FieldTypeTime:
		return true
	default:
		return false
	}
}

// SupportsLike checks if the field type supports LIKE operations
func (ft FieldType) SupportsLike() bool {
	switch ft {
	case FieldTypeString, FieldTypeText:
		return true
	default:
		return false
	}
}

// SupportsRange checks if the field type supports range operations
func (ft FieldType) SupportsRange() bool {
	switch ft {
	case FieldTypeInt, FieldTypeFloat, FieldTypeTime, FieldTypeDate:
		return true
	default:
		return false
	}
}

// SupportsIn checks if the field type supports IN operations
func (ft FieldType) SupportsIn() bool {
	switch ft {
	case FieldTypeString, FieldTypeInt, FieldTypeFloat, FieldTypeEnum, FieldTypeUUID:
		return true
	default:
		return false
	}
}

// GetAllowedOperators returns the allowed operators for this field type
// If userOperators is provided, it will be filtered and validated
// Otherwise, default operators for this type will be used
func (ft FieldType) GetAllowedOperators(userOperators []Operator) []Operator {
	var allowedOperators []Operator

	if len(userOperators) > 0 {
		// filter and validate userOperators
		defaultOperators := ft.GetDefaultOperators()
		for _, userOp := range userOperators {
			if userOp.IsValid() {
				// check if userOp is in defaultOperators
				for _, defaultOp := range defaultOperators {
					if userOp == defaultOp {
						allowedOperators = append(allowedOperators, userOp)
						break
					}
				}
			}
		}
	} else {
		// use default operators
		allowedOperators = ft.GetDefaultOperators()
	}

	return allowedOperators
}

// FilterDefine methods

// GetAllowedOperators get allowed operators for the filter definition
func (fd FilterDefine) GetAllowedOperators() []Operator {
	return fd.FieldType.GetAllowedOperators(fd.Operators)
}

// IsOperatorAllowed checks if the given operator is allowed for the filter definition
func (fd FilterDefine) IsOperatorAllowed(op Operator) bool {
	allowedOperators := fd.GetAllowedOperators()
	for _, allowedOp := range allowedOperators {
		if allowedOp == op {
			return true
		}
	}
	return false
}
