// Package query is a library of custom condition queries, support for complex conditional paging queries.
package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// Eq equal
	Eq       = "eq"
	eqSymbol = "="
	// Neq not equal
	Neq       = "neq"
	neqSymbol = "!="
	// Gt greater than
	Gt       = "gt"
	gtSymbol = ">"
	// Gte greater than or equal
	Gte       = "gte"
	gteSymbol = ">="
	// Lt less than
	Lt       = "lt"
	ltSymbol = "<"
	// Lte less than or equal
	Lte       = "lte"
	lteSymbol = "<="
	// Like fuzzy lookup
	Like = "like"
	// In include
	In = "in"
	// NotIn exclude
	NotIn = "nin"
	// IsNull is null
	IsNull = "isnull"
	// IsNotNull is not null
	IsNotNull = "isnotnull"

	// AND logic and
	AND        string = "and" //nolint
	andSymbol1        = "&"
	andSymbol2        = "&&"
	// OR logic or
	OR        string = "or" //nolint
	orSymbol1        = "|"
	orSymbol2        = "||"
)

var expMap = map[string]string{
	Eq:            eqSymbol,
	eqSymbol:      eqSymbol,
	Neq:           neqSymbol,
	neqSymbol:     neqSymbol,
	Gt:            gtSymbol,
	gtSymbol:      gtSymbol,
	Gte:           gteSymbol,
	gteSymbol:     gteSymbol,
	Lt:            ltSymbol,
	ltSymbol:      ltSymbol,
	Lte:           lteSymbol,
	lteSymbol:     lteSymbol,
	Like:          Like,
	In:            In,
	NotIn:         NotIn,
	"notin":       NotIn,
	"not in":      NotIn,
	IsNull:        IsNull,
	IsNotNull:     IsNotNull,
	"is null":     IsNull,
	"is not null": IsNotNull,
}

var logicMap = map[string]string{
	AND:        AND,
	"AND":      AND,
	andSymbol1: AND,
	andSymbol2: AND,

	OR:        OR,
	"OR":      OR,
	orSymbol1: OR,
	orSymbol2: OR,

	"and:(": AND,
	"and:)": AND,
	"or:(":  OR,
	"or:)":  OR,
}

// ---------------------------------------------------------------------------

type rulerOptions struct {
	whitelistNames map[string]bool
	validateFn     func(columns []Column) error
}

// RulerOption set the parameters of ruler options
type RulerOption func(*rulerOptions)

func (o *rulerOptions) apply(opts ...RulerOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithWhitelistNames set white list names of columns
func WithWhitelistNames(whitelistNames map[string]bool) RulerOption {
	return func(o *rulerOptions) {
		o.whitelistNames = whitelistNames
	}
}

// WithValidateFn set validate function of columns
func WithValidateFn(fn func(columns []Column) error) RulerOption {
	return func(o *rulerOptions) {
		o.validateFn = fn
	}
}

// -----------------------------------------------------------------------------

// Params query parameters
type Params struct {
	Page  int    `json:"page" form:"page" binding:"gte=0"`
	Limit int    `json:"limit" form:"limit" binding:"gte=1"`
	Sort  string `json:"sort,omitempty" form:"sort" binding:""`

	Columns []Column `json:"columns,omitempty" form:"columns"` // not required

	// Deprecated: use Limit instead in sponge version v1.8.6, will remove in the future
	Size int `json:"size" form:"size"`
}

// Column query info
type Column struct {
	Name  string      `json:"name" form:"name"`   // column name
	Exp   string      `json:"exp" form:"exp"`     // expressions, default value is "=", support =, !=, >, >=, <, <=, like, in
	Value interface{} `json:"value" form:"value"` // column value
	Logic string      `json:"logic" form:"logic"` // logical type, defaults to and when the value is null, with &(and), ||(or)
}

func (c *Column) checkName(whitelists map[string]bool) error {
	if c.Name == "" || (whitelists != nil && !whitelists[c.Name]) {
		return fmt.Errorf("field name '%s' is not allowed", c.Name)
	}
	return nil
}

func (c *Column) checkValid() error {
	if c.Name == "" {
		return fmt.Errorf("field 'name' cannot be empty")
	}
	if c.Value == nil {
		return fmt.Errorf("field 'value' cannot be nil")
	}
	return nil
}

func (c *Column) convertLogic() error {
	if c.Logic == "" {
		c.Logic = AND
	}
	if v, ok := logicMap[strings.ToLower(c.Logic)]; ok { //nolint
		c.Logic = v
		return nil
	}
	return fmt.Errorf("convertLogic error: unknown logic type '%s'", c.Logic)
}

func (c *Column) checkLogic() error {
	if c.Logic == "" {
		c.Logic = AND
	}
	if _, ok := logicMap[strings.ToLower(c.Logic)]; ok { //nolint
		return nil
	}
	return fmt.Errorf("checkLogic error: unknown logic type '%s'", c.Logic)
}

// converting ExpType to sql expressions and LogicType to sql using characters
func (c *Column) convert() error {
	err := c.convertValue()
	if err != nil {
		return err
	}
	return c.convertLogic()
}

// nolint
func (c *Column) convertValue() error {
	if err := c.checkValid(); err != nil {
		return err
	}

	if oid, ok := isObjectID(c.Value); ok {
		c.Value = oid

		if c.Name == "id" {
			c.Name = "_id" // force to "_id"
		} else if strings.HasSuffix(c.Name, ":oid") {
			c.Name = strings.TrimSuffix(c.Name, ":oid")
		}
	} else {
		c.Value = convertValue(c.Value)
	}

	if c.Exp == "" {
		c.Exp = Eq
	}
	if v, ok := expMap[strings.ToLower(c.Exp)]; ok {
		c.Exp = v
		switch c.Exp {
		// case eqSymbol:
		case neqSymbol:
			c.Value = bson.M{"$ne": c.Value}
		case gtSymbol:
			c.Value = bson.M{"$gt": c.Value}
		case gteSymbol:
			c.Value = bson.M{"$gte": c.Value}
		case ltSymbol:
			c.Value = bson.M{"$lt": c.Value}
		case lteSymbol:
			c.Value = bson.M{"$lte": c.Value}
		case IsNull:
			c.Value = bson.M{"$exist": false}
		case IsNotNull:
			c.Value = bson.M{"$exist": true}
		case Like:
			escapedValue := regexp.QuoteMeta(fmt.Sprintf("%v", c.Value))
			c.Value = bson.M{"$regex": escapedValue, "$options": "i"}
		case In, NotIn:
			val, ok2 := c.Value.(string)
			if ok2 {
				values := []interface{}{}
				ss := strings.Split(val, ",")
				for _, s := range ss {
					s = strings.TrimSpace(s)
					if strings.HasPrefix(s, "\"") {
						values = append(values, strings.Trim(s, "\""))
						continue
					} else if strings.HasPrefix(s, "'") {
						values = append(values, strings.Trim(s, "'"))
						continue
					}
					value, err := strconv.Atoi(s)
					if err == nil {
						values = append(values, value)
					} else {
						values = append(values, s)
					}
				}
				c.Value = bson.M{"$" + c.Exp: values}
			} else {
				c.Value = bson.M{"$" + c.Exp: c.Value}
			}
		}
	} else {
		return fmt.Errorf("unsupported exp type '%s'", c.Exp)
	}
	return nil
}

// ConvertToPage converted to page
func (p *Params) ConvertToPage() (sort bson.D, limit int, skip int) { //nolint
	page := NewPage(p.Page, p.Limit, p.Sort)
	sort = page.sort
	limit = page.limit
	skip = page.page * page.limit
	return //nolint
}

// ConvertToMongoFilter conversion to mongo-compliant parameters based on the Columns parameter
// ignore the logical type of the last column, whether it is a one-column or multi-column query
func (p *Params) ConvertToMongoFilter(opts ...RulerOption) (bson.M, error) {
	o := rulerOptions{}
	o.apply(opts...)
	if o.validateFn != nil {
		err := o.validateFn(p.Columns)
		if err != nil {
			return nil, err
		}
	}

	filter := bson.M{}
	l := len(p.Columns)
	switch l {
	case 0:
		return bson.M{}, nil

	case 1: // l == 1
		err := p.Columns[0].checkName(o.whitelistNames)
		if err != nil {
			return nil, err
		}
		err = p.Columns[0].convert()
		if err != nil {
			return nil, err
		}
		filter[p.Columns[0].Name] = p.Columns[0].Value
		return filter, nil

	case 2: // l == 2
		err := p.Columns[0].checkName(o.whitelistNames)
		if err != nil {
			return nil, err
		}
		err = p.Columns[1].checkName(o.whitelistNames)
		if err != nil {
			return nil, err
		}
		err = p.Columns[0].convert()
		if err != nil {
			return nil, err
		}
		err = p.Columns[1].convert()
		if err != nil {
			return nil, err
		}
		if p.Columns[0].Logic == AND {
			filter = bson.M{"$and": []bson.M{
				{p.Columns[0].Name: p.Columns[0].Value},
				{p.Columns[1].Name: p.Columns[1].Value},
			}}
		} else {
			filter = bson.M{"$or": []bson.M{
				{p.Columns[0].Name: p.Columns[0].Value},
				{p.Columns[1].Name: p.Columns[1].Value},
			}}
		}
		return filter, nil

	default: // l >=3
		return p.convertMultiColumns(o.whitelistNames)
	}
}

func (p *Params) convertMultiColumns(whitelistNames map[string]bool) (bson.M, error) {
	if len(p.Columns) == 0 {
		return bson.M{"filter": bson.M{}}, nil
	}

	hasParentheses := false
	countLeftParentheses := 0
	countRightParentheses := 0
	for _, col := range p.Columns {
		err := col.checkName(whitelistNames)
		if err != nil {
			return nil, err
		}
		if strings.Contains(col.Logic, "(") {
			hasParentheses = true
			countLeftParentheses++
		}
		if strings.Contains(col.Logic, ")") {
			countRightParentheses++
			hasParentheses = true
		}
	}
	if countLeftParentheses != countRightParentheses {
		return nil, fmt.Errorf("mismatched parentheses in logic")
	}

	var finalFilter bson.M
	var err error

	if hasParentheses {
		finalFilter, err = buildFilterWithStack(p.Columns)
	} else {
		finalFilter, err = buildFilterWithPrecedence(p.Columns)
	}

	return finalFilter, err
}

func isObjectID(v interface{}) (primitive.ObjectID, bool) {
	if str, ok := v.(string); ok && len(str) == 24 {
		value, err := primitive.ObjectIDFromHex(str)
		if err == nil {
			return value, true
		}
	}
	return [12]byte{}, false
}

type filterGroup struct {
	operator string   // "$and", "$or"
	filters  []bson.M // list of filters within this group
}

// use stack to handle explicit grouping
func buildFilterWithStack(columns []Column) (bson.M, error) {
	stack := []*filterGroup{
		{operator: "$and", filters: []bson.M{}},
	}

	for _, col := range columns {
		if err := col.checkLogic(); err != nil {
			return nil, err
		}
		singleFilter, err := col.createSingleCondition()
		if err != nil {
			return nil, fmt.Errorf("failed to create condition for column '%s': %w", col.Name, err)
		}

		logic := strings.ToLower(col.Logic)
		if logic == "" {
			logic = "and"
		}
		op := "$and"
		if strings.HasPrefix(logic, "or") {
			op = "$or"
		}

		if strings.HasSuffix(logic, ":(") {
			newGroup := &filterGroup{
				operator: op,
				filters:  []bson.M{singleFilter},
			}
			stack = append(stack, newGroup)
		} else if strings.HasSuffix(logic, ":)") {
			if len(stack) < 2 {
				return nil, fmt.Errorf("mismatched parentheses in logic: '%s'", logic)
			}
			currentGroup := stack[len(stack)-1]
			currentGroup.filters = append(currentGroup.filters, singleFilter)
			stack = stack[:len(stack)-1]

			var combined bson.M
			if currentGroup.operator == "$and" {
				merged := bson.M{}
				for _, f := range currentGroup.filters {
					for k, v := range f {
						merged[k] = v
					}
				}
				combined = merged
			} else {
				combined = bson.M{currentGroup.operator: currentGroup.filters}
			}

			parentGroup := stack[len(stack)-1]
			parentGroup.filters = append(parentGroup.filters, combined)
			if op == "$or" {
				parentGroup.operator = "$or"
			}
		} else {
			topGroup := stack[len(stack)-1]
			topGroup.filters = append(topGroup.filters, singleFilter)
			if op == "$or" {
				topGroup.operator = "$or"
			}
		}
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("unclosed parentheses at the end of query")
	}

	rootGroup := stack[0]
	var finalFilter bson.M
	if len(rootGroup.filters) == 1 && rootGroup.operator == "$and" {
		finalFilter = rootGroup.filters[0]
	} else {
		finalFilter = bson.M{rootGroup.operator: rootGroup.filters}
	}

	return finalFilter, nil
}

// use precedence rules to handle flat lists (AND has higher precedence than OR)
func buildFilterWithPrecedence(columns []Column) (bson.M, error) {
	orGroups := [][]*Column{}
	currentAndGroup := []*Column{}

	for i := range columns {
		col := &columns[i]
		if err := col.convertLogic(); err != nil {
			return nil, err
		}
		currentAndGroup = append(currentAndGroup, col)
		if strings.ToLower(col.Logic) == "or" {
			orGroups = append(orGroups, currentAndGroup)
			currentAndGroup = []*Column{}
		}
	}

	if len(currentAndGroup) > 0 {
		orGroups = append(orGroups, currentAndGroup)
	}

	orParts := []bson.M{}
	for _, group := range orGroups {
		andParts := []bson.M{}
		for _, col := range group {
			condition, err := col.createSingleCondition()
			if err != nil {
				return nil, err
			}
			andParts = append(andParts, condition)
		}

		if len(andParts) == 0 {
			continue
		} else if len(andParts) == 1 {
			orParts = append(orParts, andParts[0])
		} else {
			orParts = append(orParts, bson.M{"$and": andParts})
		}
	}

	if len(orParts) == 0 {
		return bson.M{}, nil
	}
	if len(orParts) == 1 {
		return orParts[0], nil
	}

	return bson.M{"$or": orParts}, nil
}

// convert a single Column to a BSON condition (no change)
func (c *Column) createSingleCondition() (bson.M, error) {
	err := c.convertValue()
	if err != nil {
		return nil, fmt.Errorf("convertValue error: %v", err)
	}
	return bson.M{c.Name: c.Value}, nil
}

// if the value is a string or an integer, if true means it is a string, otherwise it is an integer
func convertValue(v interface{}) interface{} {
	s, ok := v.(string)
	if !ok {
		return v
	}

	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "\"") {
		s2 := strings.Trim(s, "\"")
		if _, err := strconv.Atoi(s2); err == nil {
			return s2
		}
		return s
	}
	intVal, err := strconv.Atoi(s)
	if err == nil {
		return intVal
	}
	boolVal, err := strconv.ParseBool(s)
	if err == nil {
		return boolVal
	}
	floatVal, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return floatVal
	}

	// try to parse as RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	// support other formats
	layouts := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05.999999999Z0700",
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05 -07:00",
		"2006-01-02 15:04:05.999999999 -07:00",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t
		}
	}
	return v
}

// -------------------------------------------------------------------------------------------

// Conditions query conditions
type Conditions struct {
	Columns []Column `json:"columns" form:"columns" binding:"min=1"` // columns info
}

// CheckValid check valid
func (c *Conditions) CheckValid() error {
	if len(c.Columns) == 0 {
		return fmt.Errorf("field 'columns' cannot be empty")
	}

	for _, column := range c.Columns {
		err := column.checkValid()
		if err != nil {
			return err
		}
		if column.Exp != "" {
			if _, ok := expMap[column.Exp]; !ok {
				return fmt.Errorf("unknown exp type '%s'", column.Exp)
			}
		}
		if column.Logic != "" {
			if _, ok := logicMap[column.Logic]; !ok {
				return fmt.Errorf("unknown logic type '%s'", column.Logic)
			}
		}
	}

	return nil
}

// ConvertToMongo conversion to mongo-compliant parameters based on the Columns parameter
// ignore the logical type of the last column, whether it is a one-column or multi-column query
func (c *Conditions) ConvertToMongo(opts ...RulerOption) (bson.M, error) {
	p := &Params{Columns: c.Columns}
	return p.ConvertToMongoFilter(opts...)
}
