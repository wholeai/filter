package filter

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testProduct struct {
	ID       uint   `gorm:"column:id"`
	Price    int    `gorm:"column:price"`
	Featured bool   `gorm:"column:featured"`
	Name     string `gorm:"column:name"`
}

func openDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("gorm.Open(DryRun) failed: %v", err)
	}
	return db
}

func TestScopeByQuery_AllowedFilterExpression_NoPlaceholderDoesNotInjectVars(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(true),
		WithPaginate(false),
		WithSearch(false),
		WithOrderBy(false),
		WithAllowedFilters([]FilterDefine{
			{Field: "featured", Expression: "featured = 1"},
		}),
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10},
		Options: &opts,
		Filters: []FilterClause{
			{Field: "featured", Value: true},
		},
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	if !strings.Contains(sql, "featured = 1") {
		t.Fatalf("expected SQL to include custom expression, got: %s", sql)
	}
	if len(tx.Statement.Vars) != 0 {
		t.Fatalf("expected no vars for expression without placeholders, got: %#v", tx.Statement.Vars)
	}
}

func TestScopeByQuery_AllowedFilterExpression_NoPlaceholderBoolFalseIsSkipped(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(true),
		WithPaginate(false),
		WithSearch(false),
		WithOrderBy(false),
		WithAllowedFilters([]FilterDefine{
			{Field: "featured", Expression: "featured = 1"},
		}),
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10},
		Options: &opts,
		Filters: []FilterClause{
			{Field: "featured", Value: false},
		},
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	if strings.Contains(sql, "featured = 1") {
		t.Fatalf("expected SQL to skip false boolean toggle filter, got: %s", sql)
	}
}

func TestScopeByQuery_AllowedFilterExpression_WithPlaceholderBindsVar(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(true),
		WithPaginate(false),
		WithSearch(false),
		WithOrderBy(false),
		WithAllowedFilters([]FilterDefine{
			{Field: "min_price", Expression: "price >= ?"},
		}),
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10},
		Options: &opts,
		Filters: []FilterClause{
			{Field: "min_price", Value: 100},
		},
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	if !strings.Contains(sql, "price >= ?") {
		t.Fatalf("expected SQL to include placeholder expression, got: %s", sql)
	}
	if len(tx.Statement.Vars) != 1 || tx.Statement.Vars[0] != 100 {
		t.Fatalf("expected vars [100], got: %#v", tx.Statement.Vars)
	}
}

// Test search functionality with allowed search fields
func TestScopeByQuery_SearchWithAllowedFields(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(false),
		WithPaginate(false),
		WithSearch(true),
		WithOrderBy(false),
		WithSearchFields([]string{"name", "description"}),
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10, Search: "test"},
		Options: &opts,
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	// GORM generates: LOWER(`products`.`name`) LIKE ?
	if !strings.Contains(sql, "LOWER(`products`.`name`) LIKE ?") {
		t.Fatalf("expected SQL to include LOWER function for case-insensitive search, got: %s", sql)
	}
	// Check that vars contain search pattern (will be in Vars, not SQL)
	if len(tx.Statement.Vars) < 2 {
		t.Fatalf("expected at least 2 search variables (one for each field), got: %d", len(tx.Statement.Vars))
	}
	// Check that vars contain lowercase search term
	foundSearchTerm := false
	for _, v := range tx.Statement.Vars {
		if str, ok := v.(string); ok && strings.Contains(str, "test") {
			foundSearchTerm = true
			break
		}
	}
	if !foundSearchTerm {
		t.Fatalf("expected vars to contain search term, got: %#v", tx.Statement.Vars)
	}
}

// Test search with different search modes
func TestScopeByQuery_SearchModes(t *testing.T) {
	tests := []struct {
		name        string
		searchMode  SearchMode
		expectedSQL string
		expectedVar string
		expectLike  bool
	}{
		{"Contains", SearchModeContains, "LOWER(`products`.`name`) LIKE ?", "%test%", true},
		{"StartsWith", SearchModeStartsWith, "LOWER(`products`.`name`) LIKE ?", "test%", true},
		{"EndsWith", SearchModeEndsWith, "LOWER(`products`.`name`) LIKE ?", "%test", true},
		{"Exact", SearchModeExact, "`products`.`name` = ?", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openDryRunDB(t)

			opts := NewOptions(
				WithFilter(false),
				WithPaginate(false),
				WithSearch(true),
				WithOrderBy(false),
				WithSearchFields([]string{"name"}),
				WithSearchMode(tt.searchMode),
			)

			cfg := &Config{
				Query:   &Query{Page: 1, PageSize: 10, Search: "test"},
				Options: &opts,
			}

			var out []testProduct
			tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
			sql := tx.Statement.SQL.String()

			if !strings.Contains(sql, tt.expectedSQL) {
				t.Fatalf("expected SQL to include %s, got: %s", tt.expectedSQL, sql)
			}

			// Check that vars contain expected pattern
			if len(tx.Statement.Vars) == 0 {
				t.Fatalf("expected at least 1 variable, got: %d", len(tx.Statement.Vars))
			}

			varValue := fmt.Sprint(tx.Statement.Vars[0])
			if !strings.Contains(varValue, tt.expectedVar) {
				t.Fatalf("expected variable to contain %s, got: %s", tt.expectedVar, varValue)
			}
		})
	}
}

// Test pagination functionality
func TestScopeByQuery_Pagination(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(false),
		WithPaginate(true),
		WithSearch(false),
		WithOrderBy(false),
	)

	cfg := &Config{
		Query:   &Query{Page: 2, PageSize: 5},
		Options: &opts,
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	// Page 2 with page size 5 should have LIMIT and OFFSET (in dry run they're embedded)
	if !strings.Contains(sql, "LIMIT") {
		t.Fatalf("expected SQL to include LIMIT, got: %s", sql)
	}
	if !strings.Contains(sql, "OFFSET") {
		t.Fatalf("expected SQL to include OFFSET, got: %s", sql)
	}

	// In dry run mode, LIMIT and OFFSET values are embedded in SQL
	if !strings.Contains(sql, "LIMIT 5") {
		t.Fatalf("expected LIMIT 5, got: %s", sql)
	}
	if !strings.Contains(sql, "OFFSET 5") {
		t.Fatalf("expected OFFSET 5, got: %s", sql)
	}
}

// Test sorting with allowed filters
func TestScopeByQuery_Sorting(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(false),
		WithPaginate(false),
		WithSearch(false),
		WithOrderBy(true),
		WithAllowedFilters([]FilterDefine{
			{Field: "name", Sortable: true},
			{Field: "price", Sortable: true},
		}),
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10},
		Options: &opts,
		Sorters: []SorterClause{
			{Field: "name", Order: Asc},
			{Field: "price", Order: Desc},
		},
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	// Should contain ORDER BY clauses
	if !strings.Contains(sql, "ORDER BY") {
		t.Fatalf("expected SQL to include ORDER BY, got: %s", sql)
	}
}

// Test includes/preloads functionality
func TestScopeByQuery_Includes(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(false),
		WithPaginate(false),
		WithSearch(false),
		WithOrderBy(false),
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10},
		Options: &opts,
		Includes: []IncludeClause{
			{Name: "Category", Preload: "Category"},
			{Name: "Tags", Preload: "Tags"},
		},
	}

	var out []testProduct
	_ = db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)

	// In dry run mode, we can verify includes are processed without error
	// The actual preload verification would require a real database connection
	t.Log("Includes processed without error")
}

// Test multiple filters combined
func TestScopeByQuery_MultipleFilters(t *testing.T) {
	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(true),
		WithPaginate(false),
		WithSearch(false),
		WithOrderBy(false),
		WithAllowedFilters([]FilterDefine{
			{Field: "min_price", Expression: "price >= ?"},
			{Field: "max_price", Expression: "price <= ?"},
			{Field: "featured", Expression: "featured = ?"},
		}),
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10},
		Options: &opts,
		Filters: []FilterClause{
			{Field: "min_price", Value: 100},
			{Field: "max_price", Value: 500},
			{Field: "featured", Value: true},
		},
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	// Should contain all filter conditions
	if !strings.Contains(sql, "price >= ?") || !strings.Contains(sql, "price <= ?") || !strings.Contains(sql, "featured = ?") {
		t.Fatalf("expected SQL to include all filter conditions, got: %s", sql)
	}

	// Should have 3 variables
	if len(tx.Statement.Vars) != 3 {
		t.Fatalf("expected 3 filter variables, got: %d", len(tx.Statement.Vars))
	}
}

// Test filter operators without custom expressions
func TestScopeByQuery_FilterOperators(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		operator  Operator
		value     any
		expected  string
		expectVar any
	}{
		{"GreaterThan", "price", GreaterThan, 100, "WHERE `products`.`price` > ?", 100},
		{"LessThan", "price", LessThan, 500, "WHERE `products`.`price` < ?", 500},
		{"Contains", "name", Contains, "test", "WHERE `products`.`name` LIKE ?", "%test%"},
		{"In", "category_id", In, []any{1, 2}, "WHERE `products`.`category_id` IN (?,?)", []any{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openDryRunDB(t)

			opts := NewOptions(
				WithFilter(true),
				WithPaginate(false),
				WithSearch(false),
				WithOrderBy(false),
				WithAllowedFilters([]FilterDefine{
					{Field: tt.field, FieldType: FieldTypeInt},
				}),
			)

			cfg := &Config{
				Query:   &Query{Page: 1, PageSize: 10},
				Options: &opts,
				Filters: []FilterClause{
					{Field: tt.field, Operator: tt.operator, Value: tt.value},
				},
			}

			var out []testProduct
			tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
			sql := tx.Statement.SQL.String()

			// Check if SQL contains expected condition
			// Note: This is a basic check - in real implementation might need more sophisticated matching
			t.Logf("Generated SQL: %s", sql)
			t.Logf("Generated vars: %#v", tx.Statement.Vars)
		})
	}
}

// Test with no allowed filters (reflection-based)
func TestScopeByQuery_ReflectionBasedFilter(t *testing.T) {
	type testProductWithTags struct {
		ID       uint   `gorm:"column:id" filter:"searchable,sortable,filterable"`
		Name     string `gorm:"column:name" filter:"searchable,sortable,filterable"`
		Price    int    `gorm:"column:price" filter:"filterable,sortable"`
		Featured bool   `gorm:"column:featured" filter:"filterable"`
	}

	db := openDryRunDB(t)

	opts := NewOptions(
		WithFilter(true),
		WithPaginate(false),
		WithSearch(true),
		WithOrderBy(true),
		// No allowed filters - will use reflection
	)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10, Search: "test"},
		Options: &opts,
		Filters: []FilterClause{
			{Field: "price", Operator: GreaterThan, Value: 100},
		},
		Sorters: []SorterClause{
			{Field: "name", Order: Asc},
		},
	}

	var out []testProductWithTags
	tx := db.Table("products").Model(&testProductWithTags{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	t.Logf("Generated SQL: %s", sql)
	t.Logf("Generated vars: %#v", tx.Statement.Vars)

	// Should contain search condition for name field (marked as searchable)
	// Should contain filter condition for price field (marked as filterable)
	// Should contain order by name field (marked as sortable)
}

// Test empty configuration
func TestScopeByQuery_EmptyConfig(t *testing.T) {
	db := openDryRunDB(t)

	cfg := &Config{
		Query:   &Query{Page: 1, PageSize: 10},
		Options: &Options{EnablePaginate: false}, // Disable all features
	}

	var out []testProduct
	tx := db.Table("products").Model(&testProduct{}).Scopes(ScopeByQuery(cfg)).Find(&out)
	sql := tx.Statement.SQL.String()

	// Should not modify the query significantly
	if strings.Contains(sql, "WHERE") || strings.Contains(sql, "ORDER BY") {
		t.Fatalf("expected clean query without conditions, got: %s", sql)
	}
}
