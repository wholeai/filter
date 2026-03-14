package filter

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// paramTypeRegext extracts type information from struct tags
// Example: `filter:"filterable,type:int"` -> extracts "int"
var paramTypeRegexp = regexp.MustCompile(`(?m)type:(\w{1,}).*`)

// dangerousSQLKeywords contains SQL keywords that should not appear in custom expressions
// These are SQL commands that could modify the database schema or data
var dangerousSQLKeywords = []string{
	"DROP", "DELETE", "INSERT", "UPDATE", "TRUNCATE", "ALTER", "CREATE",
	"EXEC", "EXECUTE", "SCRIPT", "JAVASCRIPT", "--", "/*", "*/",
	"xp_", "sp_", "GRANT", "REVOKE", "SHUTDOWN",
}

// validateSQLExpression checks if a custom SQL expression contains dangerous patterns
// This prevents SQL injection through FilterDefine.Expression
// Returns an error if dangerous keywords or multiple statements are detected
func validateSQLExpression(expr string) error {
	upperExpr := strings.ToUpper(expr)
	for _, keyword := range dangerousSQLKeywords {
		if strings.Contains(upperExpr, keyword) {
			return fmt.Errorf("dangerous SQL keyword '%s' detected in expression", keyword)
		}
	}
	// Check for multiple statements (semicolon)
	if strings.Contains(expr, ";") {
		return fmt.Errorf("multiple SQL statements detected")
	}
	return nil
}

func orderBy(db *gorm.DB, sorters []*SorterClause) *gorm.DB {
	for _, sorter := range sorters {
		db = db.Order(clause.OrderByColumn{
			Column: clause.Column{Name: sorter.Field},
			Desc:   sorter.Order == Desc,
		})
	}
	return db
}

func paginate(db *gorm.DB, query *Query) *gorm.DB {
	offset := (query.Page - 1) * query.PageSize
	return db.Offset(offset).Limit(query.PageSize)
}

func buildSearchExpression(searchMode SearchMode, columnName string, phrase string) clause.Expression {
	switch searchMode {
	case SearchModeContains:
		return clause.Like{
			Column: clause.Expr{SQL: "LOWER(?)", Vars: []interface{}{clause.Column{Table: clause.CurrentTable, Name: columnName}}},
			Value:  "%" + strings.ToLower(phrase) + "%",
		}
	case SearchModeStartsWith:
		return clause.Like{
			Column: clause.Expr{SQL: "LOWER(?)", Vars: []interface{}{clause.Column{Table: clause.CurrentTable, Name: columnName}}},
			Value:  strings.ToLower(phrase) + "%",
		}
	case SearchModeEndsWith:
		return clause.Like{
			Column: clause.Expr{SQL: "LOWER(?)", Vars: []interface{}{clause.Column{Table: clause.CurrentTable, Name: columnName}}},
			Value:  "%" + strings.ToLower(phrase),
		}
	case SearchModeExact:
		return clause.Eq{
			Column: clause.Column{Table: clause.CurrentTable, Name: columnName},
			Value:  phrase,
		}
	}
	return nil
}

func searchField(columnName string, field reflect.StructField, phrase string, searchMode SearchMode) clause.Expression {
	filterTag := field.Tag.Get(tagKey)

	if strings.Contains(filterTag, "searchable") {
		return buildSearchExpression(searchMode, columnName, phrase)
	}
	return nil
}

func sortField(columnName string, field reflect.StructField, sorter SorterClause) *SorterClause {
	allowSortFields := []string{"id"}
	if !slices.Contains(allowSortFields, columnName) && !strings.Contains(field.Tag.Get(tagKey), "sortable") {
		return nil
	}
	if columnName != sorter.Field {
		return nil
	}
	return &SorterClause{Field: columnName, Order: sorter.Order}
}

func filterField(columnName string, field reflect.StructField, filter FilterClause) []clause.Expression {
	if !strings.Contains(field.Tag.Get(tagKey), "filterable") {
		return nil
	}

	if columnName != filter.Field {
		return nil
	}
	paramType := ""
	paramMatch := paramTypeRegexp.FindStringSubmatch(field.Tag.Get(tagKey))
	if len(paramMatch) == 2 {
		paramType = paramMatch[1]
	}
	val, err := convertToType(filter.Value, paramType)
	if err != nil {
		return nil
	}
	return buildExpressions(filter.Operator, filter.Field, val)
}

func expressionByField(
	db *gorm.DB, phrases []string,
	operator func(string, reflect.StructField, string, SearchMode) clause.Expression,
	predicate func(...clause.Expression) clause.Expression,
	searchMode SearchMode,
) *gorm.DB {
	modelType := reflect.TypeOf(db.Statement.Model).Elem()
	numFields := modelType.NumField()
	modelSchema, err := schema.Parse(db.Statement.Model, &sync.Map{}, db.NamingStrategy)
	if err != nil {
		return db
	}
	var allExpressions []clause.Expression

	for _, phrase := range phrases {
		expressions := make([]clause.Expression, 0, numFields)
		for i := 0; i < numFields; i++ {
			field := modelType.Field(i)
			sf := modelSchema.LookUpField(field.Name)
			if sf == nil {
				continue
			}
			expression := operator(sf.DBName, field, phrase, searchMode)
			if expression != nil {
				expressions = append(expressions, expression)
			}
		}
		if len(expressions) > 0 {
			allExpressions = append(allExpressions, predicate(expressions...))
		}
	}
	if len(allExpressions) == 1 {
		db = db.Where(allExpressions[0])
	} else if len(allExpressions) > 1 {
		db = db.Where(predicate(allExpressions...))
	}
	return db
}

func expressionByFilter(
	db *gorm.DB, filters []FilterClause,
	operator func(string, reflect.StructField, FilterClause) []clause.Expression,
	predicate func(...clause.Expression) clause.Expression,
) *gorm.DB {
	modelType := reflect.TypeOf(db.Statement.Model).Elem()
	numFields := modelType.NumField()
	modelSchema, err := schema.Parse(db.Statement.Model, &sync.Map{}, db.NamingStrategy)
	if err != nil {
		return db
	}
	var allExpressions []clause.Expression

	for _, phrase := range filters {
		expressions := make([]clause.Expression, 0, numFields)
		for i := 0; i < numFields; i++ {
			field := modelType.Field(i)
			sf := modelSchema.LookUpField(field.Name)
			if sf == nil {
				continue
			}
			expression := operator(sf.DBName, field, phrase)
			if expression != nil {
				expressions = append(expressions, expression...)
			}
		}
		if len(expressions) > 0 {
			allExpressions = append(allExpressions, predicate(expressions...))
		}
	}
	if len(allExpressions) == 1 {
		db = db.Where(allExpressions[0])
	} else if len(allExpressions) > 1 {
		db = db.Where(predicate(allExpressions...))
	}
	return db
}

func expressionBySort(
	db *gorm.DB, sorters []SorterClause,
	operator func(string, reflect.StructField, SorterClause) *SorterClause,
) *gorm.DB {
	modelType := reflect.TypeOf(db.Statement.Model).Elem()
	numFields := modelType.NumField()
	modelSchema, err := schema.Parse(db.Statement.Model, &sync.Map{}, db.NamingStrategy)
	if err != nil {
		return db
	}
	allSorters := []*SorterClause{}
	for _, sorter := range sorters {
		for i := 0; i < numFields; i++ {
			field := modelType.Field(i)
			sf := modelSchema.LookUpField(field.Name)
			if sf == nil {
				continue
			}
			sortClause := operator(sf.DBName, field, sorter)
			if sortClause != nil {
				allSorters = append(allSorters, sortClause)
			}
		}
	}
	if len(allSorters) > 0 {
		db = orderBy(db, allSorters)
	}
	return db
}

func expressionByFieldAllowed(
	db *gorm.DB, phrases []string,
	allowedSearchFields []string,
	predicate func(...clause.Expression) clause.Expression,
	searchMode SearchMode,
) *gorm.DB {
	var allExpressions []clause.Expression
	for _, phrase := range phrases {
		expressions := make([]clause.Expression, 0, len(allowedSearchFields))
		for _, columnName := range allowedSearchFields {
			expressions = append(expressions, buildSearchExpression(searchMode, columnName, phrase))
		}
		if len(expressions) > 0 {
			allExpressions = append(allExpressions, predicate(expressions...))
		}
	}
	if len(allExpressions) == 1 {
		return db.Where(allExpressions[0])
	}
	if len(allExpressions) > 1 {
		return db.Where(predicate(allExpressions...))
	}
	return db
}

func expressionByFilterAllowed(
	db *gorm.DB, filters []FilterClause,
	allowed map[string]FilterDefine,
	predicate func(...clause.Expression) clause.Expression,
) *gorm.DB {
	var allExpressions []clause.Expression
	for _, filter := range filters {
		filterDef, ok := allowed[filter.Field]
		if !ok {
			continue
		}

		var expressions []clause.Expression

		// Use custom expression if defined
		if filterDef.Expression != "" {
			sqlExpr := filterDef.Expression
			// Security: validate custom SQL expression
			if err := validateSQLExpression(sqlExpr); err != nil {
				continue // Skip invalid expressions rather than crashing
			}
			usesPlaceholder := strings.Contains(sqlExpr, "?")

			// Boolean toggle filters (e.g. featured / ignore_featured) should only apply when true.
			if b, ok := filter.Value.(bool); ok && !usesPlaceholder && !b {
				continue
			}

			var vars []interface{}
			switch {
			case usesPlaceholder && filter.Value != nil:
				vars = []interface{}{filter.Value}
			default:
				// When SQL contains no placeholders, ignore filter.Value to avoid leaking extra vars into gorm.
				vars = filterDef.Params
			}

			expressions = append(expressions, clause.Expr{SQL: sqlExpr, Vars: vars})
		} else {
			expressions = buildExpressions(filter.Operator, filter.Field, filter.Value)
		}

		if len(expressions) > 0 {
			allExpressions = append(allExpressions, predicate(expressions...))
		}
	}

	if len(allExpressions) == 1 {
		return db.Where(allExpressions[0])
	}
	if len(allExpressions) > 1 {
		return db.Where(predicate(allExpressions...))
	}
	return db
}

func expressionBySortAllowed(db *gorm.DB, sorters []SorterClause, allowed map[string]FilterDefine) *gorm.DB {
	allSorters := []*SorterClause{}
	for _, sorter := range sorters {
		def, ok := allowed[sorter.Field]
		if !ok || !def.Sortable {
			continue
		}
		allSorters = append(allSorters, &SorterClause{Field: sorter.Field, Order: sorter.Order})
	}
	if len(allSorters) > 0 {
		db = orderBy(db, allSorters)
	}
	return db
}

func buildExpressions(op Operator, field string, val interface{}) []clause.Expression {
	switch op {
	case GreaterThanOrEqual:
		return []clause.Expression{clause.Gte{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: val}}
	case LessThanOrEqual:
		return []clause.Expression{clause.Lte{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: val}}
	case NotEquals:
		return []clause.Expression{clause.Neq{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: val}}
	case GreaterThan:
		return []clause.Expression{clause.Gt{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: val}}
	case LessThan:
		return []clause.Expression{clause.Lt{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: val}}
	case Contains:
		return []clause.Expression{clause.Like{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: "%" + fmt.Sprint(val) + "%"}}
	case StartsWith:
		return []clause.Expression{clause.Like{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: fmt.Sprint(val) + "%"}}
	case EndsWith:
		return []clause.Expression{clause.Like{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: "%" + fmt.Sprint(val)}}
	case In:
		if values, ok := val.([]interface{}); ok && len(values) > 0 {
			return []clause.Expression{clause.IN{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Values: values}}
		}
	case NotIn:
		if values, ok := val.([]interface{}); ok && len(values) > 0 {
			return []clause.Expression{clause.Not(clause.IN{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Values: values})}
		}
	case Range:
		if values, ok := val.([]interface{}); ok && len(values) > 1 {
			return []clause.Expression{
				clause.Gte{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: values[0]},
				clause.Lte{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: values[1]},
			}
		}
	case IsNull:
		return []clause.Expression{clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: nil}}
	case IsNotNull:
		return []clause.Expression{clause.Neq{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: nil}}
	default:
		return []clause.Expression{clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: field}, Value: val}}
	}
	return nil
}

func ScopeByQuery(cfg *Config) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		model := db.Statement.Model
		modelType := reflect.TypeOf(model)
		if model != nil && modelType.Kind() == reflect.Ptr && modelType.Elem().Kind() == reflect.Struct {
			if len(cfg.Options.AllowedFilters) > 0 {
				allowed := make(map[string]FilterDefine)
				for _, def := range cfg.Options.AllowedFilters {
					if _, exists := allowed[def.Field]; !exists {
						allowed[def.Field] = def
					}
				}
				if cfg.Options.EnableFilter && len(cfg.Filters) > 0 {
					db = expressionByFilterAllowed(db, cfg.Filters, allowed, clause.And)
				}
				if cfg.Options.EnableOrderBy && len(cfg.Sorters) > 0 {
					db = expressionBySortAllowed(db, cfg.Sorters, allowed)
				}
			} else {
				if cfg.Options.EnableFilter && len(cfg.Filters) > 0 {
					db = expressionByFilter(db, cfg.Filters, filterField, clause.And)
				}
				if cfg.Options.EnableOrderBy && len(cfg.Sorters) > 0 {
					db = expressionBySort(db, cfg.Sorters, sortField)
				}
			}
			if cfg.Options.EnableSearch && cfg.Search != "" {
				if len(cfg.Options.AllowedSearchFields) > 0 {
					db = expressionByFieldAllowed(db, []string{cfg.Search}, cfg.Options.AllowedSearchFields, clause.Or, cfg.Options.SearchMode)
				} else {
					db = expressionByField(db, []string{cfg.Search}, searchField, clause.Or, cfg.Options.SearchMode)
				}
			}
		}
		if cfg.Options.EnablePaginate {
			db = paginate(db, cfg.Query)
		}
		if len(cfg.Includes) > 0 {
			for _, include := range cfg.Includes {
				db = db.Preload(include.Preload)
			}
		}
		return db
	}
}
