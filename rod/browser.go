package rod

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type cachedBrowser struct {
	b        *rod.Browser
	lastUsed time.Time
}

const (
	browserIdleTTL   = 5 * time.Minute
	browserIdleCheck = 1 * time.Minute
)

var (
	mu         sync.Mutex
	browser    *cachedBrowser
	wsBrowsers = map[string]*rod.Browser{}
	fetchSem   = make(chan struct{}, 8)
	evictOnce  sync.Once
)

func startEvictor() {
	evictOnce.Do(func() {
		go func() {
			t := time.NewTicker(browserIdleCheck)
			defer t.Stop()
			for range t.C {
				mu.Lock()
				if browser != nil && time.Since(browser.lastUsed) > browserIdleTTL {
					_ = browser.b.Close()
					browser = nil
				}
				mu.Unlock()
			}
		}()
	})
}

func SetMaxConcurrency(n int) {
	if n <= 0 {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	fetchSem = make(chan struct{}, n)
}

func acquireSem(ctx context.Context) (func(), error) {
	mu.Lock()
	sem := fetchSem
	mu.Unlock()

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func ensureBrowser(userAgent string, headless bool) (*rod.Browser, error) {
	mu.Lock()
	defer mu.Unlock()
	startEvictor()
	if browser != nil {
		browser.lastUsed = time.Now()
		return browser.b, nil
	}

	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	l := launcher.New().
		Headless(headless).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox", "").
		Set("disable-dev-shm-usage", "").
		Set("window-size", "1280,960").
		Set("user-agent", userAgent)

	if !headless {
		l = l.Set("window-position", "-32000,-32000")
	}

	if bin := chromePath(); bin != "" {
		l = l.Bin(bin)
	}

	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launcher.Launch: %w", err)
	}

	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("browser.Connect: %w", err)
	}
	browser = &cachedBrowser{b: b, lastUsed: time.Now()}
	return b, nil
}

func ensureBrowserWS(controlURL string) (*rod.Browser, error) {
	if controlURL == "" {
		return nil, fmt.Errorf("empty control url")
	}
	mu.Lock()
	defer mu.Unlock()
	if b, ok := wsBrowsers[controlURL]; ok {
		return b, nil
	}
	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("browser.Connect: %w", err)
	}
	wsBrowsers[controlURL] = b
	return b, nil
}

func resetBrowserWS(controlURL string) {
	mu.Lock()
	defer mu.Unlock()
	delete(wsBrowsers, controlURL)
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if browser != nil {
		_ = browser.b.Close()
		browser = nil
	}
	for url := range wsBrowsers {
		delete(wsBrowsers, url)
	}
}

func hasDisplay() bool {
	if runtime.GOOS == "darwin" {
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func chromePath() string {
	switch runtime.GOOS {
	case "darwin":
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "linux":
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
			if p, err := exec.LookPath(name); err == nil {
				return p
			}
		}
	case "windows":
		for _, p := range []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}
