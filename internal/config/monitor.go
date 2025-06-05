package config

type Monitor struct {
	// Whether the monitor is active
	Active bool `json:"active,omitempty"`
	// Assertions to run on the response
	Assertions []Assertion `json:"assertions,omitempty"`
	// Time in milliseconds to wait before marking the request as degraded
	DegradedAfter int64          `json:"degradedAfter,omitempty"`
	Description   string         `json:"description,omitempty"`
	Frequency     Frequency      `json:"frequency"`
	Kind          CoordinateKind `json:"kind"`
	// Name of the monitor
	Name string `json:"name"`
	// Whether the monitor is public
	Public bool `json:"public,omitempty"`
	// Regions to run the request in
	Regions []Region `json:"regions"`
	// The HTTP Request we are sending
	Request Request `json:"request"`
	// Number of retries to attempt
	Retry int64 `json:"retry,omitempty"`
	// Time in milliseconds to wait before marking the request as timed out
	Timeout int64 `json:"timeout,omitempty"`
}

type Assertion struct {
	// Comparison operator
	Compare Compare       `json:"compare"`
	Kind    AssertionKind `json:"kind"`
	// Status code to assert
	//
	// Header value to assert
	//
	// Text body to assert
	Target any `json:"target"`
	// Header key to assert
	Key string `json:"key,omitempty"`
}

// The HTTP Request we are sending
type Request struct {
	// Body to send with the request
	Body    string            `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Method  Method            `json:"method,omitempty"`
	// URL to request
	URL string `json:"url,omitempty"`
	// Host to connect to
	Host string `json:"host,omitempty"`
	// Port to connect to
	Port float64 `json:"port,omitempty"`
}

// Comparison operator
type Compare string

const (
	Contains    Compare = "contains"
	Empty       Compare = "empty"
	Eq          Compare = "eq"
	Gt          Compare = "gt"
	Gte         Compare = "gte"
	LTE         Compare = "lte"
	Lt          Compare = "lt"
	NotContains Compare = "not_contains"
	NotEmpty    Compare = "not_empty"
	NotEq       Compare = "not_eq"
)

type AssertionKind string

const (
	Header     AssertionKind = "header"
	StatusCode AssertionKind = "statusCode"
	TextBody   AssertionKind = "textBody"
)

type Frequency string

const (
	The10M Frequency = "10m"
	The1H  Frequency = "1h"
	The1M  Frequency = "1m"
	The30M Frequency = "30m"
	The30S Frequency = "30s"
	The5M  Frequency = "5m"
)

type CoordinateKind string

const (
	HTTP CoordinateKind = "http"
	TCP  CoordinateKind = "tcp"
)

type Region string

const (
	Ams     Region = "ams"
	Arn     Region = "arn"
	Atl     Region = "atl"
	BOM     Region = "bom"
	Bog     Region = "bog"
	Bos     Region = "bos"
	Cdg     Region = "cdg"
	Den     Region = "den"
	Dfw     Region = "dfw"
	Ewr     Region = "ewr"
	Eze     Region = "eze"
	Fra     Region = "fra"
	Gdl     Region = "gdl"
	Gig     Region = "gig"
	Gru     Region = "gru"
	Hkg     Region = "hkg"
	Iad     Region = "iad"
	Jnb     Region = "jnb"
	Lax     Region = "lax"
	Lhr     Region = "lhr"
	Mad     Region = "mad"
	Mia     Region = "mia"
	Nrt     Region = "nrt"
	Ord     Region = "ord"
	Otp     Region = "otp"
	Phx     Region = "phx"
	Private Region = "private"
	Qro     Region = "qro"
	Scl     Region = "scl"
	Sea     Region = "sea"
	Sin     Region = "sin"
	Sjc     Region = "sjc"
	Syd     Region = "syd"
	Waw     Region = "waw"
	Yul     Region = "yul"
	Yyz     Region = "yyz"
)

type Method string

const (
	Delete  Method = "DELETE"
	Get     Method = "GET"
	Head    Method = "HEAD"
	Options Method = "OPTIONS"
	Patch   Method = "PATCH"
	Post    Method = "POST"
	Put     Method = "PUT"
)

type Target struct {
	Int    *int64
	String *string
}
