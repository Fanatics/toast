// +build darwin linux,386 windows,!cgo

//go:generate ./scripts/test.sh -f

// Package types is an example package
package types

import "github.com/Fanatics/toast/test/base"

var (
	// doc for simpleValue
	//go:generate do_something -else -helpful
	simpleValue  = "variableValue" // comment for simpleValue
	numeric      = 1233.99
	t            = thing{}
	known10Array = [10]int{}
	unknownArray = [...]string{}
	unary        = &thing{Name: "steve"}
	star         *thing
	binary       = 1 | 1 ^ 1&1&2
	function     = func() string { return "a string is returned" }
	aSimpleMap   = map[string]int{
		"one": 1,
		"two": 2,
	}
	aComplexMap = map[*base.Data]*Item{}
)

// IntFunc is an int func
func (m *MyInt) IntFunc() {
}

// OtherIntFunc is a func on MyInt
//go:generate something forOtherIntFunc
//go:noinline
func (m MyInt) OtherIntFunc() {
}

type AnotherType string

// MyInt is a type that is an int
//go:generate something forMyIntType
type MyInt int

type thing struct {
	Name string
}

const (
	simpleConstant  = "constantValue"
	numericConstant = 42
	// ExportedConstant doc string
	//go:noescape
	ExportedConstant = "EXPORTED" // comment about ExportedConstant
)

// Item is documented here and will grab other non-magic and non-generate
// comments as well.
//
// @decl:export --formats=json,csv --providers=s3
//go:generate make_item # bad example
//go:noinline
type Item struct {
	// field ignore -f
	base.Data
	ItemID   int32  `json:"item_id"`
	ItemName string `json:"item_name"`
	Size     string `json:"size"`
	// Dimensions is a field on an Item
	//go:generate dimensions 12,34,55
	// another comment on dimensions
	Dimensions      []string `json:"dimensions"` // dimensions side comment
	Weight          *float32 `json:"weight"`
	Color           string   `json:"color"`
	CountryOfOrigin string   `json:"country_of_origin"`
	Cost            int64    `json:"cost"`
	BasePrice       int64    `json:"base_price"`
	DynamicOptions  []map[string]map[string]interface{}
	DoneChan        <-chan bool
	SimpleChan      chan string
	SimpleMap       map[string]interface{}
} // item "Comment"

/*
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque euismod, mi vulputate imperdiet viverra, erat massa rutrum purus, quis hendrerit justo diam non ligula. Quisque fermentum tortor ac dui fringilla feugiat. Nam ultrices euismod viverra. Nullam et sem ut lacus facilisis tincidunt. Suspendisse eget ante at nibh congue placerat sed a metus. Aenean scelerisque ut dui sed posuere. Aenean in risus ipsum. Vivamus cursus ultrices massa ut cursus. Vestibulum sem erat, elementum in varius vitae, sagittis et elit. Donec a consectetur massa, vel posuere sapien. Phasellus accumsan tortor velit, non gravida sapien vulputate at. Nunc tempus, massa nec sagittis euismod, diam nunc commodo nulla, at vestibulum magna magna ut erat. Donec suscipit dictum est euismod placerat. Morbi at pulvinar ante. Ut feugiat diam et neque interdum sodales.

Aliquam erat volutpat. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Proin posuere convallis sapien, eget condimentum eros vestibulum et. Donec tristique purus eget ligula aliquam dictum. Nullam vulputate tincidunt ultrices. Integer vel porttitor velit. Nulla non tortor rutrum, placerat ligula eget, commodo nibh. Vivamus luctus suscipit nunc, faucibus lacinia arcu vulputate quis. Aliquam non urna id enim ullamcorper elementum ac ac nunc. Curabitur velit nibh, vulputate in orci sagittis, aliquet laoreet ex. Morbi commodo, arcu in varius viverra, odio arcu finibus ligula, vel aliquet metus nulla non ligula. Duis cursus eleifend mauris, quis volutpat nunc viverra in. In ornare tellus elit, bibendum fringilla magna blandit non. Morbi elementum lacinia mi sit amet mollis.

Etiam suscipit lacus at nisl facilisis, quis sagittis leo elementum. Aliquam erat volutpat. Nulla malesuada, ex quis pharetra egestas, ante erat malesuada nisl, nec viverra odio tellus eu dui. In id porttitor massa. Duis luctus justo id magna maximus dapibus molestie ac mi. Ut consequat varius metus non gravida. Duis eu dignissim ipsum. Suspendisse at urna id sem lobortis varius non sed enim. Aliquam non tincidunt nulla, non pretium erat.
*/

// This is a multiline comment as well, and the lines are logically collected
// as the whole group of comment lines without any new line breaks.
// That is pretty convenient!

// so alone :(

// These
//
// are
//
// connected
//
//
// together!

// RPCItem is an interface, and this is a doc comment.
type RPCItem interface {
	// ABOVE GetItem
	GetItem([]int64) ([]Item, error) // ASIDE GetItem
	CreateItem([]Item) ([]Item, error)
	UpdateItem([]Item) error
	DeleteItem([]Item) error
	base.EmbedMe
	RPCEmbed
}

type RPCEmbed interface {
	Embedded(int) error
}

// just a comment

// Export would implement an interface, and by returning false, we indicate that
// the override kicks in to prevent Items from being exported to S3, etc.
func (i *Item) Export() bool { return false }

// Try here is the documenation.
// this is a comment above the func
//go:generate something
//go:noinline
func Try(name string, id int64) error {
	// this is a comment inside the func
	return nil
}
