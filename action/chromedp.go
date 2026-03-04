package action

import (
	"context"
	"fmt"
	"time"

	"github.com/s-ajensen/browse-mcp/selector"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

func ChromedpEval(browserCtx context.Context, expression string) (any, error) {
	var result any
	err := chromedp.Run(browserCtx, chromedp.Evaluate(expression, &result))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	return result, nil
}

func chromedpNavigateAndReadPage(browserCtx context.Context, action chromedp.Action) (NavigateResult, error) {
	err := chromedp.Run(browserCtx, action)
	if err != nil {
		return NavigateResult{}, fmt.Errorf("navigation failed: %w", err)
	}
	var currentURL, title string
	err = chromedp.Run(browserCtx,
		chromedp.Location(&currentURL),
		chromedp.Title(&title),
	)
	if err != nil {
		return NavigateResult{}, fmt.Errorf("failed to get page info after navigation: %w", err)
	}
	return NavigateResult{URL: currentURL, Title: title}, nil
}

func ChromedpNavigate(browserCtx context.Context, url string, waitUntil string) (NavigateResult, error) {
	return chromedpNavigateAndReadPage(browserCtx, chromedp.Navigate(url))
}

func ChromedpBack(browserCtx context.Context) (NavigateResult, error) {
	return chromedpNavigateAndReadPage(browserCtx, chromedp.NavigateBack())
}

func ChromedpForward(browserCtx context.Context) (NavigateResult, error) {
	return chromedpNavigateAndReadPage(browserCtx, chromedp.NavigateForward())
}

func ChromedpReload(browserCtx context.Context) (NavigateResult, error) {
	return chromedpNavigateAndReadPage(browserCtx, chromedp.Reload())
}

func ChromedpGetURL(browserCtx context.Context) (GetURLResult, error) {
	var url, title string
	err := chromedp.Run(browserCtx,
		chromedp.Location(&url),
		chromedp.Title(&title),
	)
	if err != nil {
		return GetURLResult{}, fmt.Errorf("failed to get page URL: %w", err)
	}
	return GetURLResult{URL: url, Title: title}, nil
}

func ChromedpGetText(browserCtx context.Context, selector string) (GetTextResult, error) {
	var text string
	err := chromedp.Run(browserCtx,
		chromedp.Text(selector, &text, chromedp.ByQuery),
	)
	if err != nil {
		return GetTextResult{}, fmt.Errorf("failed to get text from %s: %w", selector, err)
	}
	return GetTextResult{Text: text}, nil
}

func ChromedpGetHTML(browserCtx context.Context, params GetHTMLParams) (GetHTMLResult, error) {
	sel := params.Selector
	if sel == "" {
		sel = "html"
	}
	var html string
	var action chromedp.Action
	if params.Outer {
		action = chromedp.OuterHTML(sel, &html, chromedp.ByQuery)
	} else {
		action = chromedp.InnerHTML(sel, &html, chromedp.ByQuery)
	}
	err := chromedp.Run(browserCtx, action)
	if err != nil {
		return GetHTMLResult{}, fmt.Errorf("failed to get HTML from %s: %w", sel, err)
	}
	return GetHTMLResult{HTML: html}, nil
}

func ChromedpScreenshot(browserCtx context.Context, params ScreenshotParams) ([]byte, error) {
	var buf []byte
	var action chromedp.Action
	if params.Selector != "" {
		action = chromedp.Screenshot(params.Selector, &buf, chromedp.ByQuery)
	} else if params.FullPage {
		action = chromedp.FullScreenshot(&buf, 90)
	} else {
		action = chromedp.CaptureScreenshot(&buf)
	}
	err := chromedp.Run(browserCtx, action)
	return buf, err
}

var clickQueryOption = map[selector.Kind]chromedp.QueryOption{
	selector.KindCSS:   chromedp.ByQuery,
	selector.KindXPath: chromedp.BySearch,
	selector.KindText:  chromedp.BySearch,
}

func ChromedpClick(browserCtx context.Context, params ClickParams) error {
	if params.Target.Kind == selector.KindCoordinates {
		return chromedp.Run(browserCtx, chromedp.MouseClickXY(params.Target.X, params.Target.Y))
	}
	option, ok := clickQueryOption[params.Target.Kind]
	if !ok {
		return fmt.Errorf("unsupported target kind: %s", params.Target.Kind)
	}
	return chromedp.Run(browserCtx, chromedp.Click(params.Target.Selector, option))
}

func hoverSelectorJS(target selector.Resolved) string {
	if target.Kind == selector.KindXPath || target.Kind == selector.KindText {
		return fmt.Sprintf(
			`document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue.dispatchEvent(new MouseEvent('mouseover', {bubbles: true}))`,
			target.Selector,
		)
	}
	return fmt.Sprintf(
		`document.querySelector(%q).dispatchEvent(new MouseEvent('mouseover', {bubbles: true}))`,
		target.Selector,
	)
}

func ChromedpHover(browserCtx context.Context, target selector.Resolved) error {
	if target.Kind == selector.KindCoordinates {
		return chromedp.Run(browserCtx,
			chromedp.MouseEvent(input.MouseMoved, target.X, target.Y),
		)
	}
	return chromedp.Run(browserCtx, chromedp.Evaluate(hoverSelectorJS(target), nil))
}

func typeInputActions(params TypeParams) []chromedp.Action {
	if params.Selector == "" {
		return []chromedp.Action{chromedp.KeyEvent(params.Text)}
	}
	var actions []chromedp.Action
	if params.Clear {
		actions = append(actions, chromedp.Clear(params.Selector))
	}
	return append(actions, chromedp.SendKeys(params.Selector, params.Text, chromedp.ByQuery))
}

func submitAction(params TypeParams) chromedp.Action {
	if params.Selector != "" {
		return chromedp.SendKeys(params.Selector, "\r", chromedp.ByQuery)
	}
	return chromedp.KeyEvent("\r")
}

func ChromedpType(browserCtx context.Context, params TypeParams) error {
	actions := typeInputActions(params)
	if params.Submit {
		actions = append(actions, submitAction(params))
	}
	return chromedp.Run(browserCtx, actions...)
}

func ChromedpKey(browserCtx context.Context, key string) error {
	return chromedp.Run(browserCtx, chromedp.KeyEvent(key))
}

func selectByValueJS(sel string, value string) string {
	return fmt.Sprintf(
		`(() => { const el = document.querySelector(%q); el.value = %q; el.dispatchEvent(new Event('change', {bubbles: true})); const opt = el.querySelector('option[value=' + JSON.stringify(%q) + ']'); return {selected_value: el.value, selected_label: opt ? opt.textContent : ''}; })()`,
		sel, value, value,
	)
}

func selectByLabelJS(sel string, label string) string {
	return fmt.Sprintf(
		`(() => { const el = document.querySelector(%q); const opts = Array.from(el.options); const opt = opts.find(o => o.textContent === %q); if (opt) { el.value = opt.value; el.dispatchEvent(new Event('change', {bubbles: true})); } return {selected_value: el.value, selected_label: opt ? opt.textContent : ''}; })()`,
		sel, label,
	)
}

func selectJS(params SelectParams) string {
	if params.Value != "" {
		return selectByValueJS(params.Selector, params.Value)
	}
	return selectByLabelJS(params.Selector, params.Label)
}

func ChromedpSelect(browserCtx context.Context, params SelectParams) (SelectResult, error) {
	var raw map[string]interface{}
	err := chromedp.Run(browserCtx, chromedp.Evaluate(selectJS(params), &raw))
	if err != nil {
		return SelectResult{}, fmt.Errorf("failed to select option: %w", err)
	}
	return SelectResult{
		SelectedValue: fmt.Sprintf("%v", raw["selected_value"]),
		SelectedLabel: fmt.Sprintf("%v", raw["selected_label"]),
	}, nil
}

var directionScrollDeltas = map[string][2]int{
	"up":    {0, -1},
	"down":  {0, 1},
	"left":  {-1, 0},
	"right": {1, 0},
}

func buildDirectionScrollJS(direction string, amount float64) string {
	deltas := directionScrollDeltas[direction]
	if amount == 0 {
		amount = 1
	}
	deltaX := float64(deltas[0]) * amount
	deltaY := float64(deltas[1]) * amount
	return fmt.Sprintf(`window.scrollBy(%f, %f)`, deltaX, deltaY)
}

func scrollJS(params ScrollParams) string {
	if params.Direction != "" {
		return buildDirectionScrollJS(params.Direction, params.Amount)
	}
	if params.Selector != "" {
		return fmt.Sprintf(`document.querySelector(%q).scrollIntoView({behavior:"smooth"})`, params.Selector)
	}
	return fmt.Sprintf(`window.scrollBy(%f, %f)`, params.DeltaX, params.DeltaY)
}

func readScrollPosition(browserCtx context.Context) (ScrollResult, error) {
	var scrollX, scrollY float64
	err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`window.scrollX`, &scrollX),
		chromedp.Evaluate(`window.scrollY`, &scrollY),
	)
	if err != nil {
		return ScrollResult{}, fmt.Errorf("failed to get scroll position: %w", err)
	}
	return ScrollResult{ScrollX: scrollX, ScrollY: scrollY}, nil
}

func ChromedpScroll(browserCtx context.Context, params ScrollParams) (ScrollResult, error) {
	err := chromedp.Run(browserCtx, chromedp.Evaluate(scrollJS(params), nil))
	if err != nil {
		return ScrollResult{}, fmt.Errorf("failed to scroll: %w", err)
	}
	return readScrollPosition(browserCtx)
}

var waitStateAction = map[string]func(string) chromedp.QueryAction{
	"visible":  func(sel string) chromedp.QueryAction { return chromedp.WaitVisible(sel, chromedp.ByQuery) },
	"hidden":   func(sel string) chromedp.QueryAction { return chromedp.WaitNotVisible(sel, chromedp.ByQuery) },
	"attached": func(sel string) chromedp.QueryAction { return chromedp.WaitReady(sel, chromedp.ByQuery) },
	"detached": func(sel string) chromedp.QueryAction { return chromedp.WaitNotPresent(sel, chromedp.ByQuery) },
}

func chromedpWaitSelector(timeoutCtx context.Context, params WaitParams) error {
	factory, ok := waitStateAction[params.State]
	if !ok {
		return fmt.Errorf("unsupported wait state: %s", params.State)
	}
	return chromedp.Run(timeoutCtx, factory(params.Selector))
}

func chromedpWaitXPath(timeoutCtx context.Context, params WaitParams) error {
	return chromedp.Run(timeoutCtx, chromedp.WaitReady(params.XPath, chromedp.BySearch))
}

func chromedpWaitPoll(timeoutCtx context.Context, expression string) error {
	for {
		var result bool
		err := chromedp.Run(timeoutCtx, chromedp.Evaluate(expression, &result))
		if err != nil {
			return err
		}
		if result {
			return nil
		}
		select {
		case <-timeoutCtx.Done():
			return timeoutCtx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func chromedpWaitText(timeoutCtx context.Context, params WaitParams) error {
	expression := fmt.Sprintf(`document.body.innerText.includes(%q)`, params.Text)
	return chromedpWaitPoll(timeoutCtx, expression)
}

func chromedpWaitURL(timeoutCtx context.Context, params WaitParams) error {
	expression := fmt.Sprintf(`window.location.href.includes(%q)`, params.URLContains)
	return chromedpWaitPoll(timeoutCtx, expression)
}

func chromedpWaitFunction(timeoutCtx context.Context, params WaitParams) error {
	expression := fmt.Sprintf(`!!(%s)`, params.Function)
	return chromedpWaitPoll(timeoutCtx, expression)
}

func dispatchWait(timeoutCtx context.Context, params WaitParams) error {
	if params.Selector != "" {
		return chromedpWaitSelector(timeoutCtx, params)
	}
	if params.XPath != "" {
		return chromedpWaitXPath(timeoutCtx, params)
	}
	if params.Text != "" {
		return chromedpWaitText(timeoutCtx, params)
	}
	if params.URLContains != "" {
		return chromedpWaitURL(timeoutCtx, params)
	}
	return chromedpWaitFunction(timeoutCtx, params)
}

func ChromedpWait(browserCtx context.Context, params WaitParams) (WaitResult, error) {
	start := time.Now()
	timeout := time.Duration(params.TimeoutMs) * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(browserCtx, timeout)
	defer cancel()
	err := dispatchWait(timeoutCtx, params)
	if err != nil {
		return WaitResult{}, fmt.Errorf("wait failed: %w", err)
	}
	elapsed := time.Since(start).Milliseconds()
	return WaitResult{ElapsedMs: elapsed}, nil
}
