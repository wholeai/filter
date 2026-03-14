# Go Filter Package 🚀

A **powerful, simple, and production-ready** filtering, searching, sorting, and pagination library for Go with GORM integration. Built with modern Go practices, this library provides comprehensive database query capabilities with built-in security and performance optimizations.

## ✨ Key Features

- 🔍 **Smart Search**: Automatic search across multiple fields with configurable search modes
- 🗂️ **Advanced Filtering**: Dynamic filters with custom operators, expressions, and validation
- 📄 **Pagination**: Built-in pagination with customizable page sizes
- 🔄 **Flexible Sorting**: Multi-field sorting with ascending/descending support
- 🔗 **Relationship Preloading**: Automatic preloading of related data with security validation
- 🛡️ **Security First**: SQL injection protection and input validation
- ⚡ **High Performance**: Optimized query generation with GORM integration
- 🧪 **Production Ready**: Comprehensive test coverage
- 🔧 **Configurable**: Extensive configuration options with sensible defaults
- 🌐 **Type Safe**: Strong typing with interface-based design

## 📦 Installation

```bash
go get github.com/wholeai/filter
```

## 📋 Table of Contents

- [Quick Start](#-quick-start)
- [Struct Tag Configuration](#-struct-tag-configuration)
- [Configuration Options](#-configuration-options)
- [Filter Types](#-filter-types)
- [Search Modes](#-search-modes)
- [Sorting and Pagination](#-sorting-and-pagination)
- [Relationship Preloading](#-relationship-preloading)
- [Security Features](#-security-features)
- [Advanced Examples](#-advanced-examples)
- [Testing](#-testing)

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    "github.com/wholeai/filter"
)

type Product struct {
    ID       uint   `json:"id" gorm:"primaryKey"`
    Name     string `json:"name" gorm:"not null"`
    Price    int    `json:"price"`
    Featured bool   `json:"featured"`
}

func GetProducts(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Configure filter options
        opts := filter.NewOptions(
            filter.WithAllowedFilters([]filter.FilterDefine{
                {Field: "price", FieldType: filter.FieldTypeInt, Operators: []filter.Operator{filter.GreaterThan, filter.LessThan}},
                {Field: "featured", FieldType: filter.FieldTypeBool},
                {Field: "name", FieldType: filter.FieldTypeString, Operators: []filter.Operator{filter.Contains}},
            }),
            filter.WithSearchFields([]string{"name"}),
            filter.WithDefaultSort("id", filter.Desc),
        )

        // Parse request
        cfg, err := filter.FilterByQuery(c, opts)
        if err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        // Apply filter scope
        var products []Product
        query := db.Model(&Product{}).Scopes(filter.ScopeByQuery(cfg))

        // Get total count for pagination
        var total int64
        query.Count(&total)

        // Get paginated results
        if err := query.Find(&products).Error; err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        // Build response
        totalPages := cfg.GetTotalPages(total)
        c.JSON(200, gin.H{
            "data":       products,
            "total":      total,
            "page":       cfg.Page,
            "per_page":   cfg.PageSize,
            "total_pages": totalPages,
        })
    }
}
```

**Try these URLs:**
```bash
# Basic pagination
curl "http://localhost:8080/products?page=1&page_size=10"

# Search products
curl "http://localhost:8080/products?search=laptop"

# Filter by price
curl "http://localhost:8080/products?filter[price]=gt:100"

# Multiple filters
curl "http://localhost:8080/products?filter[price]=gt:50&filter[featured]=true"

# Sorting
curl "http://localhost:8080/products?sort_by=price:desc"

# Combined search and filters
curl "http://localhost:8080/products?search=phone&filter[price]=lt:1000&sort_by=name:asc"
```

## 🏷️ Struct Tag Configuration

### Default Filter Configuration with Struct Tags

Instead of manually defining `FilterDefine` configurations, you can use struct tags for automatic filter detection. This provides a cleaner, more maintainable approach:

```go
package main

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    "github.com/wholeai/filter"
)

type Product struct {
    ID       uint   `json:"id" gorm:"primaryKey" filter:"searchable,sortable"`
    Name     string `json:"name" gorm:"not null" filter:"searchable,sortable,filterable"`
    Price    int    `json:"price" filter:"filterable,sortable,type:int"`
    Featured bool   `json:"featured" filter:"filterable"`
    Stock    int    `json:"stock" filter:"filterable"`
    CreatedAt string `json:"created_at" filter:"sortable"`
}

func GetProductsWithTags(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // No explicit FilterDefine needed - uses struct tags!
        opts := filter.NewOptions(
            filter.WithSearch(true),       // Enable search on searchable fields
            filter.WithFilter(true),       // Enable filtering on filterable fields
            filter.WithOrderBy(true),      // Enable sorting on sortable fields
            filter.WithPaginate(true),     // Enable pagination
            filter.WithDefaultSort("id", filter.Desc),
        )

        cfg, err := filter.FilterByQuery(c, opts)
        if err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        var products []Product
        query := db.Model(&Product{}).Scopes(filter.ScopeByQuery(cfg))

        var total int64
        query.Count(&total)
        query.Find(&products)

        totalPages := cfg.GetTotalPages(total)
        c.JSON(200, gin.H{
            "data":       products,
            "total":      total,
            "page":       cfg.Page,
            "per_page":   cfg.PageSize,
            "total_pages": totalPages,
        })
    }
}
```

### Available Struct Tag Options

| Tag Option | Description | Example |
|------------|-------------|---------|
| `searchable` | Field can be searched using global search parameter | `filter:"searchable"` |
| `filterable` | Field can be filtered with operators | `filter:"filterable"` |
| `sortable` | Field can be used for sorting | `filter:"sortable"` |
| `type:xxx` | Override field type for validation | `filter:"type:int"` |

**Combined Usage Examples:**
```go
type User struct {
    ID       uint   `filter:"searchable,sortable,filterable"`
    Username string `filter:"searchable,filterable"`
    Email    string `filter:"searchable,filterable"`
    Age      int    `filter:"filterable,sortable,type:int"`
    Active   bool   `filter:"filterable"`
    Role     string `filter:"filterable,type:enum"`
}

// This enables:
// ?search=john      // searches ID, Username, Email
// ?filter[age]=gt:25  // filters Age > 25
// ?sort_by=age:desc,username:asc   // sorts by Age descending and Username Ascending
// ?filter[active]=true // filters Active = true
```

**Try these URLs with struct tags:**
```bash
# Search across searchable fields (ID, Username, Email)
curl "http://localhost:8080/users?search=john"

# Filter filterable fields
curl "http://localhost:8080/users?filter[age]=gt:25&filter[active]=true"

# Sort by sortable fields
curl "http://localhost:8080/users?sort_by=age:desc"

# Combined search and filter
curl "http://localhost:8080/users?search=admin&filter[active]=true&sort_by=username:asc"
```

### When to Use Struct Tags vs Manual Configuration

**Use Struct Tags when:**
- You want convention over configuration
- Your model structure maps directly to filterable fields
- You prefer minimal code and automatic field detection
- Building simple CRUD APIs

**Use Manual Configuration when:**
- You need complex validation rules (MinValue, MaxValue, EnumValues)
- You want to restrict certain operators per field
- You need custom SQL expressions
- Your filtering logic differs from your model structure
- You need field descriptions for API documentation

## 🔧 Configuration Options

### Basic Options

```go
opts := filter.NewOptions(
    filter.WithSearch(true),                    // Enable/disable search
    filter.WithFilter(true),                   // Enable/disable filtering
    filter.WithPaginate(true),                 // Enable/disable pagination
    filter.WithOrderBy(true),                  // Enable/disable sorting
    filter.WithPageSize(10, 100),             // Default and max page size
    filter.WithDefaultSort("created_at", filter.Desc), // Default sort
    filter.WithSearchMode(filter.SearchModeContains),   // Search mode
)

// Custom search fields
opts = filter.NewOptions(
    filter.WithSearchFields([]string{"name", "description", "brand"}),
)

// Custom filter definitions
opts = filter.NewOptions(
    filter.WithAllowedFilters([]filter.FilterDefine{
        {
            Field:       "price",
            FieldType:   filter.FieldTypeFloat,
            Operators:   []filter.Operator{filter.GreaterThan, filter.LessThan, filter.Range},
            Description: "Price range filter",
            MinValue:    &[]float64{0}[0],
            MaxValue:    &[]float64{10000}[0],
        },
    }),
)
```

### Security and Strict Mode Options

```go
// DoS Protection - Configure limits to prevent abuse
opts = filter.NewOptions(
    filter.WithSecurityLimits(
        20,  // MaxFilters: maximum number of filter conditions
        100, // MaxFilterValues: maximum values in IN/NotIn operators
        5,   // MaxSortFields: maximum number of sort fields
        3,   // MaxIncludeDepth: maximum nesting for includes (e.g., "A.B.C" = depth 3)
        200, // MaxSearchLength: maximum characters in search query
    ),
)

// Strict Mode - Error on unknown fields instead of silently ignoring
opts = filter.NewOptions(
    filter.WithAllowedFilters([]filter.FilterDefine{...}),
    filter.WithStrictMode(true), // Returns error for unknown filters/sorts/includes
)
```

**Why use Strict Mode?**
- Catches typos in query parameters early
- Prevents silent failures in production
- Makes debugging easier
- Recommended for public APIs

### Filter Definitions

```go
filters := []filter.FilterDefine{
    // String field with length validation
    {
        Field:       "name",
        FieldType:   filter.FieldTypeString,
        Operators:   []filter.Operator{filter.Equals, filter.Contains, filter.StartsWith},
        MaxLength:   &[]int{255}[0],
        Description: "Product name",
    },
    
    // Numeric field with range validation
    {
        Field:       "price",
        FieldType:   filter.FieldTypeFloat,
        Operators:   []filter.Operator{filter.Equals, filter.GreaterThan, filter.LessThan, filter.Range},
        MinValue:    &[]float64{0}[0],
        MaxValue:    &[]float64{999999}[0],
        Description: "Product price",
    },
    
    // Enum field with allowed values
    {
        Field:       "status",
        FieldType:   filter.FieldTypeEnum,
        Operators:   []filter.Operator{filter.Equals, filter.In},
        EnumValues:  []string{"active", "inactive", "pending"},
        Description: "Product status",
    },
    
    // Custom expression filter
    {
        Field:       "featured",
        FieldType:   filter.FieldTypeBool,
        Expression:  "featured = 1",
        Description: "Featured products only",
    },
    
    // Sortable field
    {
        Field:       "created_at",
        FieldType:   filter.FieldTypeTime,
        Sortable:    true,
        Description: "Creation date",
    },
}
```

## 🗂️ Filter Types

### Basic Operators

| Operator | Symbol | Example | Description |
|----------|--------|---------|-------------|
| Equals | `eq:` | `filter[age]=eq:25` | Exact match |
| NotEquals | `ne:` | `filter[status]=ne:inactive` | Not equal to |
| GreaterThan | `gt:` | `filter[price]=gt:100` | Greater than |
| GreaterThanOrEqual | `gte:` | `filter[rating]=gte:4` | Greater or equal |
| LessThan | `lt:` | `filter[age]=lt:65` | Less than |
| LessThanOrEqual | `lte:` | `filter[price]=lte:500` | Less or equal |
| Contains | `contains:` | `filter[name]=contains:laptop` | Contains substring |
| StartsWith | `starts:` | `filter[name]=starts:Mac` | Starts with |
| EndsWith | `ends:` | `filter[email]=ends:@gmail.com` | Ends with |

### Advanced Operators

| Operator | Symbol | Example | Description |
|----------|--------|---------|-------------|
| In | `in:` | `filter[category]=in:1,2,3` | In list of values |
| NotIn | `nin:` | `filter[status]=nin:banned,deleted` | Not in list |
| Range | `range:` | `filter[price]=range:100,500` | Between values (inclusive) |
| IsNull | `null:` | `filter[description]=null:` | Is null |
| IsNotNull | `notnull:` | `filter[deleted_at]=notnull:` | Is not null |

## 🔍 Search Modes

```go
// Search modes
opts := filter.NewOptions(
    filter.WithSearchMode(filter.SearchModeContains),    // %term%
    filter.WithSearchMode(filter.SearchModeStartsWith),  // term%
    filter.WithSearchMode(filter.SearchModeEndsWith),    // %term
    filter.WithSearchMode(filter.SearchModeExact),       // term
)
```

## 🔄 Sorting and Pagination

### Sorting Configuration

```go
// Default sort
opts := filter.NewOptions(
    filter.WithDefaultSort("created_at", filter.Desc),
)

// Allowed sortable fields
opts = filter.NewOptions(
    filter.WithAllowedFilters([]filter.FilterDefine{
        {Field: "name", Sortable: true},
        {Field: "price", Sortable: true},
        {Field: "created_at", Sortable: true},
    }),
)
```

### Pagination

```go
// Custom page sizes
opts := filter.NewOptions(
    filter.WithPageSize(20, 100), // Default 20, Max 100
)

// Automatic pagination info
totalPages := cfg.GetTotalPages(totalCount)
```

## 🔗 Relationship Preloading

```go
type Product struct {
    ID        uint       `json:"id" gorm:"primaryKey"`
    Name      string     `json:"name"`
    Category  Category   `json:"category" gorm:"foreignKey:CategoryID"`
    Tags      []Tag      `json:"tags" gorm:"many2many:product_tags;"`
    Reviews   []Review   `json:"reviews,omitempty" gorm:"foreignKey:ProductID"`
}

func GetProductsWithRelations(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        opts := filter.NewOptions(
            filter.WithAllowedIncludes([]string{"Category", "Tags", "Reviews"}),
        )

        cfg, _ := filter.FilterByQuery(c, opts)
        
        var products []Product
        query := db.Model(&Product{}).Scopes(filter.ScopeByQuery(cfg))
        query.Find(&products)

        c.JSON(200, gin.H{"data": products})
    }
}

// Load specific relationships
curl "http://localhost:8080/products?include=Category,Tags"
```

## 🛡️ Security Features

### Built-in DoS Protection

The package includes built-in protections against denial-of-service attacks:

```go
// Default security limits (automatically applied)
opts := filter.NewOptions(
    filter.WithSecurityLimits(
        20,  // MaxFilters: maximum number of filter conditions
        100, // MaxFilterValues: maximum values in IN/NotIn operators
        5,   // MaxSortFields: maximum number of sort fields
        3,   // MaxIncludeDepth: maximum nesting for includes
        200, // MaxSearchLength: maximum characters in search query
    ),
)
```

**Why this matters:**
- Prevents attackers from creating overly complex queries
- Limits memory and CPU consumption
- Protects against large IN clauses
- Controls search query length

### SQL Injection Protection

```go
// Automatic type validation
{
    Field: "price",
    FieldType: filter.FieldTypeFloat,
    MinValue: &[]float64{0}[0],     // Reject negative prices
    MaxValue: &[]float64{10000}[0], // Reject excessive prices
}

// String length validation
{
    Field: "name",
    FieldType: filter.FieldTypeString,
    MaxLength: &[]int{255}[0], // Maximum length
}

// Enum validation
{
    Field: "status",
    FieldType: filter.FieldTypeEnum,
    EnumValues: []string{"active", "inactive"}, // Only allowed values
}
```

### SQL Injection Protection

- All user input is parameterized
- Custom expressions are validated
- Only allowed fields can be filtered/sorted
- Preloads are restricted to whitelist

## 📚 Advanced Examples

### E-commerce Product Filter

```go
func GetProductFilterOptions() filter.Options {
    return filter.NewOptions(
        filter.WithAllowedFilters([]filter.FilterDefine{
            {
                Field:       "price",
                FieldType:   filter.FieldTypeFloat,
                Operators:   []filter.Operator{filter.GreaterThan, filter.LessThan, filter.Range},
                Description: "Price range filter",
            },
            {
                Field:       "rating",
                FieldType:   filter.FieldTypeInt,
                Operators:   []filter.Operator{filter.GreaterThanOrEqual},
                Description: "Minimum rating",
                MinValue:    &[]float64{1}[0],
                MaxValue:    &[]float64{5}[0],
            },
            {
                Field:       "category",
                FieldType:   filter.FieldTypeString,
                Operators:   []filter.Operator{filter.Equals, filter.Contains},
                Description: "Product category",
            },
            {
                Field:       "in_stock",
                FieldType:   filter.FieldTypeBool,
                Expression:  "stock_quantity > 0",
                Description: "In stock products only",
            },
        }),
        filter.WithSearchFields([]string{"name", "description", "brand"}),
        filter.WithAllowedIncludes([]string{"Category", "Reviews", "Tags"}),
        filter.WithDefaultSort("created_at", filter.Desc),
    )
}

func AdvancedProductSearch(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        opts := GetProductFilterOptions()
        cfg, err := filter.FilterByQuery(c, opts)
        if err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        var products []Product
        query := db.Model(&Product{}).Scopes(filter.ScopeByQuery(cfg))
        
        // Get stats
        var total int64
        query.Count(&total)
        
        // Execute query
        query.Find(&products)

        c.JSON(200, gin.H{
            "data":        products,
            "total":       total,
            "page":        cfg.Page,
            "per_page":    cfg.PageSize,
            "total_pages": cfg.GetTotalPages(total),
        })
    }
}
```

### API Response Example

```json
{
  "data": [
    {
      "id": 1,
      "name": "MacBook Pro 14",
      "price": 1999.99,
      "featured": true,
      "category": {
        "id": 1,
        "name": "Laptops"
      },
      "tags": [
        {"id": 1, "name": "apple"},
        {"id": 2, "name": "premium"}
      ]
    }
  ],
  "total": 150,
  "page": 1,
  "per_page": 10,
  "total_pages": 15
}
```

## 🧪 Testing

The package includes comprehensive tests. Run them with:

```bash
go test ./...
go test -v ./...  # Verbose output
go test -race ./... # Race detection
go test -bench=. ./... # Benchmarks
```

### Running Specific Tests

```bash
# Run only GORM tests
go test -run TestScopeByQuery ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🤝 Contributing

We welcome contributions! Here's how to help:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Add tests for new functionality
4. Run tests: `go test ./...`
5. Commit changes: `git commit -m 'feat: add amazing feature'`
6. Push to branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **[GORM](https://gorm.io/)** - The fantastic Go ORM library
- **[Gin](https://gin-gonic.com/)** - High-performance HTTP web framework
- **Go Community** - For inspiration and feedback

## 📞 Support

- 📖 **Documentation**: Check this README and test files
- 🐛 **Issues**: [GitHub Issues](https://github.com/wholeai/filter/issues)
- ⭐ **Star the repo** if you find it useful!