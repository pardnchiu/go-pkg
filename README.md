# go-utils

個人開發常用的 Go 工具函式集合，從過往專案中逐步累積而成。

## 內容

### http

泛型 HTTP GET/POST/PUT/PATCH/DELETE，自動處理 JSON/XML 解碼。

```go
import "github.com/pardnchiu/go-utils/http"

// GET
data, status, err := http.GET[MyStruct](ctx, nil, "https://api.example.com/data", nil)

// POST（JSON）
data, status, err := http.POST[MyStruct](ctx, nil, "https://api.example.com/data", nil, body, "json")

// POST（Form）
data, status, err := http.POST[MyStruct](ctx, nil, "https://api.example.com/data", nil, body, "form")

// PUT
data, status, err := http.PUT[MyStruct](ctx, nil, "https://api.example.com/data", nil, body, "json")

// PATCH
data, status, err := http.PATCH[MyStruct](ctx, nil, "https://api.example.com/data", nil, body, "json")

// DELETE
data, status, err := http.DELETE[MyStruct](ctx, nil, "https://api.example.com/data", nil, body, "json")
```

### database

PostgreSQL 連線建構與 migration runner。

`NewPostgresql` 從環境變數讀取預設值（`PG_DSN` / `PG_HOST` / `PG_PORT` / `PG_USER` / `PG_PASSWORD` / `PG_DATABASE` / `PG_SSLMODE`），`cfg` 非零欄位覆寫之。

`PostgresqlMigrate` 遞迴掃描 `dir` 下所有 `.sql` 檔，以相對路徑為版本鍵排序執行，透過 `schema_migrations` 表確保冪等，每筆 migration 以 transaction 包裹。

```go
import (
	_ "github.com/lib/pq"
	"github.com/pardnchiu/go-utils/database"
)

db, _ := database.NewPostgresql(ctx, nil)
defer db.Close()

err := database.PostgresqlMigrate(ctx, db, "./migrations")
```

### rod

go-rod 打包：Chromium 抓取網頁，以 readability 擷取主文，輸出 `*FetchResult`（含 `Href` 原始網址 / `FinalURL` 轉址後最終網址 / `Markdown` / `Title` / `Author` / `PublishedAt` / `Excerpt` / `Status`）。內含 HTML→Markdown 轉換與跨平台 Chrome 偵測。另支援透過 `FetchWS` 連接既有 Chrome 的 remote debugging WebSocket（`--remote-debugging-port`），用於沿用使用者登入 session 的場景。`Fetch` / `FetchWS` 可併發呼叫，共用單一 browser 並各自開獨立 tab；全域併發上限預設 8，可透過 `SetMaxConcurrency(n)` 調整。

內建 stealth.js 注入（抗爬蟲偵測）、3 秒 settle 等待（等動態內容穩定）、page-level viewport（預設 1280×960），均可透過 `FetchOption` 覆寫。`KeepLinks=false`（預設）為純文字模式，剝除 `nav` / `header` / `footer` / `aside` / `img` / `a`；`KeepLinks=true` 輸出完整 markdown。

`Fetch` 依環境自動選模式：有 display 時使用 headful（視窗以 off-screen position 隱藏），無 display 時使用 headless。Browser instance 常駐複用，閒置 5 分鐘自動關閉釋放資源。`FetchWS` 行為不變。

遇到 HTTP 錯誤、空內容、challenge page（Cloudflare 等）時，error 為 `*FetchError{Status, Href}`，`Status` 可能為 4xx/5xx、`204`（空內容）或 `403`（challenge / URL heuristic），可用 `errors.As` 分流。

```go
import "github.com/pardnchiu/go-utils/rod"

defer rod.Close()

result, err := rod.Fetch(ctx, "https://example.com/article", nil)
// result.Href / result.FinalURL / result.Title / result.Author / result.PublishedAt / result.Excerpt / result.Status / result.Markdown

// 連接既有 Chrome（需以 --remote-debugging-port=9222 啟動，見下方）
result, err = rod.FetchWS(ctx, "http://127.0.0.1:9222", "https://example.com/article", nil)
```

**以 remote debugging 啟動 Chrome**

```bash
# macOS
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --remote-debugging-port=9222 \
  --user-data-dir="$HOME/.chrome-debug"

# Linux
google-chrome \
  --remote-debugging-port=9222 \
  --user-data-dir="$HOME/.chrome-debug" &
```

```go
result, err := rod.Fetch(ctx, "https://example.com/article", &rod.FetchOption{
	Timeout:   20 * time.Second,
	MaxLength: 50 << 10,
	KeepLinks: true,
	Viewport:  &rod.Viewport{Width: 1920, Height: 1080, DeviceScaleFactor: 1},
})

// HTTP 錯誤分流
var fe *rod.FetchError
if errors.As(err, &fe) {
	_ = fe.Status // 404 / 503 / ...
}

// 單獨使用 HTML→Markdown
out, err := rod.HTMLToMarkdown(htmlFragment, baseURL, true) // keepLinks=true
```

### filesystem

原子化檔案寫入（自動建立目錄、先寫 `.tmp` 再 rename）。

```go
import "github.com/pardnchiu/go-utils/filesystem"

err := filesystem.WriteFile("/path/to/file.txt", "content", 0644)
```

### filesystem/keychain

跨平台密鑰存取（macOS Keychain / Linux secret-tool / 檔案 fallback）。

```go
import "github.com/pardnchiu/go-utils/filesystem/keychain"

// 初始化（sync.Once，僅首次生效）
keychain.Init("MyApp", fallbackDir)

val := keychain.Get("API_KEY")
err := keychain.Set("API_KEY", "secret")
err := keychain.Delete("API_KEY")
```
