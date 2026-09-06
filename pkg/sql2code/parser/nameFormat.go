package parser

import (
	"strings"

	"github.com/huandu/xstrings"
)

var peculiarNouns = map[string]string{
	"ID":    "Id",
	"UID":   "Uid",
	"UUID":  "Uuid",
	"GUID":  "Guid",
	"URI":   "Uri",
	"URL":   "Url",
	"IP":    "Ip",
	"QPS":   "Qps",
	"API":   "Api",
	"ASCII": "Ascii",
	"CPU":   "Cpu",
	"CSS":   "Css",
	"DNS":   "Dns",
	"EOF":   "Eof",
	"HTML":  "Html",
	"HTTP":  "Http",
	"HTTPS": "Https",
	"JSON":  "Json",
	"LHS":   "Lhs",
	"RAM":   "Ram",
	"RHS":   "Rhs",
	"RPC":   "Rpc",
	"SLA":   "Sla",
	"SMTP":  "Smtp",
	"SSH":   "Ssh",
	"TLS":   "Tls",
	"TTL":   "Ttl",
	"UI":    "Ui",
	"UTF8":  "Utf8",
	"VM":    "Vm",
	"XML":   "Xml",
	"XSRF":  "Xsrf",
	"XSS":   "Xss",
}

func toCamel(s string) string {
	str := xstrings.ToCamelCase(s)

	name := strings.ToUpper(str)
	if _, ok := peculiarNouns[name]; ok {
		return name
	}

	l := len(str)
	for k, v := range peculiarNouns {
		nl := len(v)
		if l > nl {
			if str[l-nl:] == v {
				str = str[:l-nl] + k
				break
			}
		}
	}

	if str == "_ID" { // special case for table column ID
		str = "ID"
	}

	return str
}

func firstLetterToLower(str string) string {
	if len(str) == 0 {
		return str
	}

	if (str[0] >= 'A' && str[0] <= 'Z') || (str[0] >= 'a' && str[0] <= 'z') {
		return strings.ToLower(str[:1]) + str[1:]
	}

	return str
}

func customToCamel(str string) string {
	str = toCamel(str)

	if _, ok := peculiarNouns[str]; ok {
		str = strings.ToLower(str)
	} else {
		str = firstLetterToLower(str)
	}

	return str
}

func customToSnake(str string) string {
	str = toCamel(str)
	l := len(str)
	for k := range peculiarNouns {
		if str == k {
			str = strings.ToLower(str)
			break
		}

		nl := len(k)
		if l > nl {
			if str[l-nl:] == k {
				str = str[:l-nl] + "_" + strings.ToLower(k)
				break
			}
		}
	}

	str = xstrings.ToSnakeCase(str)
	if strings.HasPrefix(str, "__") {
		str = str[1:]
	}

	return str
}
