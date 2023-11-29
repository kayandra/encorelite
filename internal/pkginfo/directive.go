package pkginfo

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type directive string
type Directive interface{}
type ApiDirective struct {
	Raw    bool
	Method string
	Path   string
}

var (
	_ Directive = (*ApiDirective)(nil)

	ErrInvalidDirective = errors.New("invalid directive format")
	ErrInvalidOption    = errors.New("invalid option format")
	ErrUnknownDirective = errors.New("unknown directive")
)

const (
	tag                    = "do:"
	DirectiveApi directive = "api"
)

func ParseDirective(raw string) (Directive, error) {
	var t directive
	options := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		token := scanner.Text()
		if strings.HasPrefix(token, tag) {
			if t != "" {
				return new(Directive), ErrInvalidDirective
			}
			t = directive(token[len(tag):])
			continue
		}

		opts := strings.Split(token, "=")
		switch len(opts) {
		case 1:
			options[opts[0]] = "true"
		case 2:
			if opts[0] == "" || opts[1] == "" {
				return new(Directive), ErrInvalidOption
			}
			options[opts[0]] = opts[1]
		default:
			return new(Directive), ErrInvalidOption
		}
	}

	if err := scanner.Err(); err != nil {
		return new(Directive), fmt.Errorf("Invalid input: %s", err)
	}

	switch t {
	case DirectiveApi:
		return parseApiDirective(options)
	default:
		return new(Directive), ErrUnknownDirective
	}
}

func parseApiDirective(opts map[string]string) (ApiDirective, error) {
	d := ApiDirective{
		Raw:    true,
		Method: http.MethodGet,
		Path:   "",
	}

	if raw, ok := opts["raw"]; ok {
		switch raw {
		case "true":
			d.Raw = true
		case "false":
			d.Raw = false
		}
	}

	if method, ok := opts["method"]; ok {
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			d.Method = method
		}
	}

	if path, ok := opts["path"]; ok {
		d.Path = path
	}

	return d, nil
}
