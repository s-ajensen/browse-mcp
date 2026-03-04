package selector

import (
	"errors"
	"fmt"
)

type Kind string

const (
	KindCSS         Kind = "css"
	KindXPath       Kind = "xpath"
	KindText        Kind = "text"
	KindCoordinates Kind = "coordinates"
)

type Params struct {
	Selector string
	XPath    string
	Text     string
	X        *float64
	Y        *float64
}

type Resolved struct {
	Kind     Kind
	Selector string
	X        float64
	Y        float64
}

func hasCoordinates(params Params) bool {
	return params.X != nil && params.Y != nil
}

func hasPartialCoordinates(params Params) bool {
	return (params.X != nil) != (params.Y != nil)
}

func countStrategies(params Params) int {
	count := 0
	if params.Selector != "" {
		count++
	}
	if params.XPath != "" {
		count++
	}
	if params.Text != "" {
		count++
	}
	if hasCoordinates(params) {
		count++
	}
	return count
}

func validateParams(params Params) error {
	if hasPartialCoordinates(params) {
		return errors.New("both X and Y coordinates are required")
	}
	strategies := countStrategies(params)
	if strategies == 0 {
		return errors.New("no selector strategy provided")
	}
	if strategies > 1 {
		return errors.New("multiple selector strategies provided")
	}
	return nil
}

func Resolve(params Params) (Resolved, error) {
	if err := validateParams(params); err != nil {
		return Resolved{}, err
	}
	if params.Selector != "" {
		return Resolved{Kind: KindCSS, Selector: params.Selector}, nil
	}
	if params.XPath != "" {
		return Resolved{Kind: KindXPath, Selector: params.XPath}, nil
	}
	if params.Text != "" {
		xpath := fmt.Sprintf(`//*[contains(normalize-space(text()), "%s")]`, params.Text)
		return Resolved{Kind: KindText, Selector: xpath}, nil
	}
	return Resolved{Kind: KindCoordinates, X: *params.X, Y: *params.Y}, nil
}
