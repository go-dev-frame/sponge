package query

import "strings"

var defaultMaxSize = 1000

// SetMaxSize change the default maximum number of pages per page
func SetMaxSize(maxValue int) {
	if maxValue < 10 {
		maxValue = 10
	}
	defaultMaxSize = maxValue
}

// Page info
type Page struct {
	page  int    // page number, starting from page 0
	limit int    // number per page
	sort  string // sort fields, default is id backwards, you can add - sign before the field to indicate reverse order, no - sign to indicate ascending order, multiple fields separated by comma
}

// Page get page value
func (p *Page) Page() int {
	return p.page
}

// Limit number per page
func (p *Page) Limit() int {
	return p.limit
}

// Size number per page
// Deprecated: use Limit instead
func (p *Page) Size() int {
	return p.limit
}

// Sort get sort field
func (p *Page) Sort() string {
	return p.sort
}

// Offset get offset value
func (p *Page) Offset() int {
	return p.page * p.limit
}

// DefaultPage default page, number 20 per page, sorted by id backwards
func DefaultPage(page int) *Page {
	if page < 0 {
		page = 0
	}
	return &Page{
		page:  page,
		limit: 20,
		sort:  "id DESC",
	}
}

// NewPage custom page, starting from page 0.
// the parameter columnNames indicates a sort field, if empty means id descending,
// if there are multiple column names, separated by a comma,
// a '-' sign in front of each column name indicates descending order, otherwise ascending order.
func NewPage(page int, limit int, columnNames string) *Page {
	if page < 0 {
		page = 0
	}
	if limit > defaultMaxSize || limit < 1 {
		limit = defaultMaxSize
	}

	return &Page{
		page:  page,
		limit: limit,
		sort:  getSort(columnNames),
	}
}

// convert to mysql sort, each column name preceded by a '-' sign, indicating descending order, otherwise ascending order, example:
//
//	columnNames="name" means sort by name in ascending order,
//	columnNames="-name" means sort by name descending,
//	columnNames="name,age" means sort by name in ascending order, otherwise sort by age in ascending order,
//	columnNames="-name,-age" means sort by name descending before sorting by age descending.
func getSort(columnNames string) string {
	columnNames = strings.Replace(columnNames, " ", "", -1)
	if columnNames == "" || columnNames == "ignorecount" {
		return "id DESC"
	}

	names := strings.Split(columnNames, ",")
	strs := make([]string, 0, len(names))
	for _, name := range names {
		if name[0] == '-' && len(name) > 1 {
			strs = append(strs, name[1:]+" DESC")
		} else {
			strs = append(strs, name+" ASC")
		}
	}

	return strings.Join(strs, ", ")
}
