package selector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func floatPtr(val float64) *float64 {
	return &val
}

func TestResolve_CSSSelector_ReturnsKindCSSWithSelector(t *testing.T) {
	params := Params{Selector: "div.content > p"}

	resolved, err := Resolve(params)

	assert.NoError(t, err)
	assert.Equal(t, KindCSS, resolved.Kind)
	assert.Equal(t, "div.content > p", resolved.Selector)
}

func TestResolve_XPath_ReturnsKindXPathWithXPath(t *testing.T) {
	params := Params{XPath: "//div[@class='main']"}

	resolved, err := Resolve(params)

	assert.NoError(t, err)
	assert.Equal(t, KindXPath, resolved.Kind)
	assert.Equal(t, "//div[@class='main']", resolved.Selector)
}

func TestResolve_Text_ReturnsKindTextWithGeneratedXPath(t *testing.T) {
	params := Params{Text: "Click here"}

	resolved, err := Resolve(params)

	assert.NoError(t, err)
	assert.Equal(t, KindText, resolved.Kind)
	assert.Equal(t, `//*[contains(normalize-space(text()), "Click here")]`, resolved.Selector)
}

func TestResolve_Coordinates_ReturnsKindCoordinatesWithXY(t *testing.T) {
	params := Params{X: floatPtr(100.5), Y: floatPtr(200.75)}

	resolved, err := Resolve(params)

	assert.NoError(t, err)
	assert.Equal(t, KindCoordinates, resolved.Kind)
	assert.Equal(t, 100.5, resolved.X)
	assert.Equal(t, 200.75, resolved.Y)
}

func TestResolve_NoStrategy_ReturnsError(t *testing.T) {
	params := Params{}

	_, err := Resolve(params)

	assert.Error(t, err)
}

func TestResolve_MultipleStrategies_ReturnsError(t *testing.T) {
	params := Params{Selector: "div.content", XPath: "//div"}

	_, err := Resolve(params)

	assert.Error(t, err)
}

func TestResolve_CoordinatesOnlyX_ReturnsError(t *testing.T) {
	params := Params{X: floatPtr(100.0)}

	_, err := Resolve(params)

	assert.Error(t, err)
}

func TestResolve_CoordinatesOnlyY_ReturnsError(t *testing.T) {
	params := Params{Y: floatPtr(200.0)}

	_, err := Resolve(params)

	assert.Error(t, err)
}
