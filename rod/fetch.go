package rod

import (
	"context"
	_ "embed"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	readability "github.com/go-shiori/go-readability"
)

//go:embed embed/stealth.js
var defaultStealthJS string

//go:embed embed/listener.js
var defaultListenerJS string

type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}

type FetchOption struct {
	Timeout   time.Duration
	IdleWait  time.Duration
	MaxLength int
	UserAgent string
	KeepLinks bool
	StealthJS string
	SettleJS  string
	Viewport  *Viewport
}

type FetchResult struct {
	Href        string
	FinalURL    string
	Markdown    string
	Title       string
	Author      string
	PublishedAt string
	Excerpt     string
	Status      int
}

type FetchError struct {
	Status int
	Href   string
}

func (e *FetchError) Error() string {
	return fmt.Sprintf("http %d: %s", e.Status, e.Href)
}

const (
	defaultTimeout   = 30 * time.Second
	defaultIdleWait  = 5 * time.Second
	defaultMaxLength = 1 << 20
)

func prepareOpt(opt *FetchOption) *FetchOption {
	o := FetchOption{}
	if opt != nil {
		o = *opt
	}
	if o.Timeout == 0 {
		o.Timeout = defaultTimeout
	}
	if o.IdleWait == 0 {
		o.IdleWait = defaultIdleWait
	}
	if o.MaxLength == 0 {
		o.MaxLength = defaultMaxLength
	}
	if o.StealthJS == "" {
		o.StealthJS = defaultStealthJS
	}
	if o.SettleJS == "" {
		o.SettleJS = defaultListenerJS
	}
	if o.Viewport == nil {
		o.Viewport = &Viewport{Width: 1280, Height: 960}
	}
	return &o
}

func parseHref(href string) (*url.URL, error) {
	parsed, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("url.Parse: %w", err)
	}
	if parsed.Scheme == "" || !strings.Contains(parsed.Hostname(), ".") {
		return nil, fmt.Errorf("invalid url: %s", href)
	}
	return parsed, nil
}

func Fetch(ctx context.Context, href string, opt *FetchOption) (*FetchResult, error) {
	o := prepareOpt(opt)
	parsed, err := parseHref(href)
	if err != nil {
		return nil, err
	}
	b, err := ensureBrowser(o.UserAgent)
	if err != nil {
		return nil, err
	}
	return load(ctx, b, href, parsed, o)
}

func FetchWS(ctx context.Context, controlURL, href string, opt *FetchOption) (*FetchResult, error) {
	o := prepareOpt(opt)
	parsed, err := parseHref(href)
	if err != nil {
		return nil, err
	}
	b, err := ensureBrowserWS(controlURL)
	if err != nil {
		return nil, err
	}
	r, err := load(ctx, b, href, parsed, o)
	if err != nil && strings.Contains(err.Error(), "browser.Page") {
		resetBrowserWS(controlURL)
	}
	return r, err
}

func load(ctx context.Context, b *rod.Browser, href string, parsed *url.URL, opt *FetchOption) (*FetchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()

	release, err := acquireSem(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquireSem: %w", err)
	}
	defer release()

	page, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("browser.Page: %w", err)
	}
	defer func() { _ = page.Close() }()

	page = page.Context(ctx)

	if opt.Viewport != nil {
		scale := opt.Viewport.DeviceScaleFactor
		if scale == 0 {
			scale = 1
		}
		if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:             opt.Viewport.Width,
			Height:            opt.Viewport.Height,
			DeviceScaleFactor: scale,
		}); err != nil {
			return nil, fmt.Errorf("page.SetViewport: %w", err)
		}
	}

	if opt.StealthJS != "" {
		if _, err := page.EvalOnNewDocument(opt.StealthJS); err != nil {
			return nil, fmt.Errorf("page.EvalOnNewDocument: %w", err)
		}
	}

	if err := page.Navigate(href); err != nil {
		return nil, fmt.Errorf("page.Navigate: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("page.WaitLoad: %w", err)
	}

	finalURL := href
	if info, err := page.Info(); err == nil && info.URL != "" {
		finalURL = info.URL
		if u, err := url.Parse(finalURL); err == nil {
			code := func(s string) int {
				for _, e := range []string{"404", "403"} {
					if s == e {
						return 400 + int(e[2]-'0')
					}
				}
				return 0
			}
			for seg := range strings.SplitSeq(u.Path, "/") {
				if c := code(seg); c != 0 {
					return nil, &FetchError{Status: c, Href: href}
				}
			}
			for _, vals := range u.Query() {
				for _, v := range vals {
					if c := code(v); c != 0 {
						return nil, &FetchError{Status: c, Href: href}
					}
				}
			}
		}
	}

	status := 0
	if v, err := page.Eval(`() => { const e = performance.getEntriesByType("navigation")[0]; return e ? e.responseStatus : 0 }`); err == nil {
		status = v.Value.Int()
		if status >= 400 {
			return nil, &FetchError{Status: status, Href: href}
		}
	}

	_ = page.WaitIdle(opt.IdleWait)

	if opt.SettleJS != "" {
		settleCtx, settleCancel := context.WithTimeout(ctx, opt.IdleWait)
		_, _ = page.Context(settleCtx).Eval(opt.SettleJS)
		settleCancel()
	}

	htmlSrc, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("page.HTML: %w", err)
	}

	article, err := readability.FromReader(strings.NewReader(htmlSrc), parsed)
	if err != nil {
		return nil, fmt.Errorf("readability: %w", err)
	}

	content := strings.TrimSpace(article.Content)
	if content == "" {
		content = htmlSrc
	}
	md, err := HTMLToMarkdown(content, href, opt.KeepLinks)
	if err != nil {
		return nil, fmt.Errorf("HTMLToMarkdown: %w", err)
	}
	if md == "" {
		return nil, fmt.Errorf("empty content")
	}
	if opt.MaxLength > 0 && len(md) > opt.MaxLength {
		md = md[:opt.MaxLength]
	}

	result := &FetchResult{
		Href:     href,
		FinalURL: finalURL,
		Markdown: md,
		Title:    article.Title,
		Author:   article.Byline,
		Excerpt:  article.Excerpt,
		Status:   status,
	}
	if article.PublishedTime != nil {
		result.PublishedAt = article.PublishedTime.Format(time.RFC3339)
	}
	return result, nil
}
