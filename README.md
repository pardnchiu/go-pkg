# go-utils

個人開發常用的 Go 工具函式集合，從過往專案中逐步累積而成。

## 內容

### http

泛型 HTTP GET/POST/PUT/PATCH/DELETE，自動處理 JSON/XML 解碼。

<details>
<summary>範例</summary>

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

</details>

### database

PostgreSQL 連線建構與 migration runner。

`NewPostgresql` 從環境變數讀取預設值（`PG_DSN` / `PG_HOST` / `PG_PORT` / `PG_USER` / `PG_PASSWORD` / `PG_DATABASE` / `PG_SSLMODE`），`cfg` 非零欄位覆寫之。

`PostgresqlMigrate` 遞迴掃描 `dir` 下所有 `.sql` 檔，以相對路徑為版本鍵排序執行，透過 `schema_migrations` 表確保冪等，每筆 migration 以 transaction 包裹。

<details>
<summary>範例</summary>

```go
import (
	_ "github.com/lib/pq"
	"github.com/pardnchiu/go-utils/database"
)

db, _ := database.NewPostgresql(ctx, nil)
defer db.Close()

err := database.PostgresqlMigrate(ctx, db, "./migrations")
```

</details>

### rod

go-rod 打包：Chromium 抓取網頁，以 readability 擷取主文，輸出 `*FetchResult`（含 `Href` 原始網址 / `FinalURL` 轉址後最終網址 / `Markdown` / `Title` / `Author` / `PublishedAt` / `Excerpt` / `Status`）。內含 HTML→Markdown 轉換與跨平台 Chrome 偵測。另支援透過 `FetchWS` 連接既有 Chrome 的 remote debugging WebSocket（`--remote-debugging-port`），用於沿用使用者登入 session 的場景。`Fetch` / `FetchWS` 可併發呼叫，共用單一 browser 並各自開獨立 tab；全域併發上限預設 8，可透過 `SetMaxConcurrency(n)` 調整。

內建 stealth.js 注入（抗爬蟲偵測）、3 秒 settle 等待（等動態內容穩定）、page-level viewport（預設 1280×960），均可透過 `FetchOption` 覆寫。`KeepLinks=false`（預設）為純文字模式，剝除 `nav` / `header` / `footer` / `aside` / `img` / `a`；`KeepLinks=true` 輸出完整 markdown。

`Fetch` 依環境自動選模式：有 display 時使用 headful（視窗以 off-screen position 隱藏），無 display 時使用 headless。Browser instance 常駐複用，閒置 5 分鐘自動關閉釋放資源。`FetchWS` 行為不變。

遇到 HTTP 錯誤、空內容、challenge page（Cloudflare 等）時，error 為 `*FetchError{Status, Href}`，`Status` 可能為 4xx/5xx、`204`（空內容）或 `403`（challenge / URL heuristic），可用 `errors.As` 分流。

<details>
<summary>範例</summary>

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

</details>

### filesystem

檔案系統工具集：

- **Policy 注入**：`New(Policy{DeniedMap, ExcludeList})` 一次性注入 sandbox 規則（`sync.Once`），未注入時 `IsDenied` 永遠 `false`、`IsExcluded` 僅讀執行期 `.gitignore`，**不**回 error（既有 caller 零破壞）。`IsDenied` / `IsExcluded` / `RealPath` / `AbsPath(root, path, AbsPathOption{HomeOnly, NeedExclude})` 為 public 工具；`WriteFile` / `WriteText` / `WriteJSON` / `AppendText` / `Copy` / `Move` / `Remove` 自動套 `IsDenied`（read-side 不套）
- **原子寫入**：`WriteFile` / `WriteText` / `WriteJSON` / `Copy` 透過 `.tmp` + `os.Rename`
- **目錄**：`CheckDir(path, create)` 檢查路徑是否為目錄（`create=true` 時不存在會以 `0755` 建立）；`ListFiles` / `ListDirs` 非遞迴列出名稱（僅 regular file / dir，**不**收 symlink／device／socket）；`ListAll` 非遞迴回 `[]os.DirEntry`（全 type 不過濾，caller 自行分類）；`WalkFiles` 遞迴並回傳相對路徑（slash 分隔，跳過點開頭目錄）。四者皆接受 variadic `ListOption{SkipExcluded bool}`：`SkipExcluded=true` 時依 `IsExcluded` 過濾命中項，Walk 對命中目錄回 `filepath.SkipDir`
- **存在性**：`Exists` / `IsFile` / `IsDir` 統一處理 stat error；`IsEmpty` 區分目錄空與檔案 size=0
- **讀寫**：`ReadText` / `WriteText` / `AppendText`；泛型 `ReadJSON[T]` / `WriteJSON`
- **搬移**：`Move` 跨 device 自動 fallback 為 copy + remove；`Remove` 忽略不存在錯誤

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-utils/filesystem"

// Policy（optional，未呼叫則無 deny / exclude 限制）
err := filesystem.New(filesystem.Policy{
    DeniedMap:   deniedJSON,   // {"dirs":[...],"files":[...],"prefixes":[...],"extensions":[...]}
    ExcludeList: excludeJSON,  // []string，gitignore-like patterns
})

// Sandbox 工具
deny := filesystem.IsDenied("/etc/passwd")
excl := filesystem.IsExcluded("/work/root", "/work/root/node_modules/x.js")
real, err := filesystem.RealPath("/path/maybe/symlink")
abs, err := filesystem.AbsPath("/work/root", "~/Desktop", filesystem.AbsPathOption{
    HomeOnly:    true,
    NeedExclude: false,
})

// 原子寫入（自動套 IsDenied）
err = filesystem.WriteFile("/path/to/file.txt", "content", 0644)
err = filesystem.WriteText("/path/to/file.txt", "content")
err = filesystem.AppendText("/path/to/log.txt", "line\n")

// 目錄
err = filesystem.CheckDir("/path/to/dir", true)         // 不存在則建立
files, err := filesystem.ListFiles("/path/to/dir")      // 非遞迴，僅 regular file
dirs, err := filesystem.ListDirs("/path/to/dir")        // 非遞迴，僅 dir
entries, err := filesystem.ListAll("/path/to/dir")      // 非遞迴，全 type 不過濾（[]os.DirEntry）
all, err := filesystem.WalkFiles("/path/to/root")       // 遞迴

// 套 IsExcluded（讀 root/dir 內的 .gitignore 與 Policy.ExcludeList）
files, err = filesystem.ListFiles("/work", filesystem.ListOption{SkipExcluded: true})
dirs, err = filesystem.ListDirs("/work", filesystem.ListOption{SkipExcluded: true})
entries, err = filesystem.ListAll("/work", filesystem.ListOption{SkipExcluded: true})
all, err = filesystem.WalkFiles("/work", filesystem.ListOption{SkipExcluded: true})

// 存在性
ok := filesystem.Exists("/path/to/x")
ok = filesystem.IsFile("/path/to/x")
ok = filesystem.IsDir("/path/to/x")
empty, err := filesystem.IsEmpty("/path/to/x")

// 讀取
content, err := filesystem.ReadText("/path/to/file.txt")

// JSON
type Config struct{ Host string }
cfg, err := filesystem.ReadJSON[Config]("/path/to/cfg.json")
err = filesystem.WriteJSON("/path/to/cfg.json", cfg, true)  // formatted
err = filesystem.WriteJSON("/path/to/cfg.json", cfg, false) // compact

// 搬移
err = filesystem.Copy("/src", "/dst")
err = filesystem.Move("/src", "/dst")               // 跨 device 自動 fallback
err = filesystem.Remove("/path/to/x")
```

</details>

### utils

通用小工具。`UUID()` 產生 RFC 4122 v4 UUID。`GetWithDefault` / `GetWithDefaultInt` / `GetWithDefaultFloat` 讀取環境變數並於未設定、空字串或解析失敗時回傳預設值。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-utils/utils"

id := utils.UUID() // e.g. "f47ac10b-58cc-4372-a567-0e02b2c3d479"

host := utils.GetWithDefault("HOST", "localhost")
port := utils.GetWithDefaultInt("PORT", 8080)
ratio := utils.GetWithDefaultFloat("RATIO", 1.5)
```

</details>

### filesystem/keychain

跨平台密鑰存取（macOS Keychain / Linux secret-tool / 檔案 fallback）。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-utils/filesystem/keychain"

// 初始化（sync.Once，僅首次生效）
keychain.Init("MyApp", fallbackDir)

val := keychain.Get("API_KEY")
err := keychain.Set("API_KEY", "secret")
err := keychain.Delete("API_KEY")
```

</details>
