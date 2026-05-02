# go-pkg

個人開發常用的 Go 工具函式集合，從過往專案中逐步累積而成。

## 內容

### http

泛型 HTTP GET/POST/PUT/PATCH/DELETE，自動處理 JSON/XML 解碼。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-pkg/http"

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
	"github.com/pardnchiu/go-pkg/database"
)

db, _ := database.NewPostgresql(ctx, nil)
defer db.Close()

err := database.PostgresqlMigrate(ctx, db, "./migrations")
```

</details>

### rod

go-rod 打包：Chromium 抓取網頁，以 readability 擷取主文，輸出 `*FetchResult`（含 `Href` 原始網址 / `FinalURL` 轉址後最終網址 / `Markdown` / `Title` / `Author` / `PublishedAt` / `Excerpt` / `Status`）。內含 HTML→Markdown 轉換與跨平台 Chrome 偵測。另支援透過 `FetchWS` 連接既有 Chrome 的 remote debugging WebSocket（`--remote-debugging-port`），用於沿用使用者登入 session 的場景。`Fetch` / `FetchWS` 可併發呼叫，共用單一 browser 並各自開獨立 tab；全域併發上限預設 8，可透過 `SetMaxConcurrency(n)` 調整。

`Fetch` / `FetchWS` 將整體 timeout 提升為函式參數（caller 自填，傳 `0` 表示不套 timeout，僅吃 parent ctx）；其餘參數（`IdleWait` / `MaxLength` / `Viewport` 等）仍透過 `FetchOption` 覆寫。載入策略改為 `WaitDOMStable(IdleWait, 0.01)`：等 DOM 連續 `IdleWait` 秒內變動 ≤ 1% 即返回，不等網路 idle、不被 GA／廣告 beacon 拖住。內建 stealth.js 注入（抗爬蟲偵測）、page-level viewport（預設 1280×960）。`KeepLinks=false`（預設）為純文字模式，剝除 `nav` / `header` / `footer` / `aside` / `img` / `a`；`KeepLinks=true` 輸出完整 markdown。

`Fetch` 依環境自動選模式：有 display 時使用 headful（視窗以 off-screen position 隱藏），無 display 時使用 headless。Browser instance 常駐複用，閒置 5 分鐘自動關閉釋放資源。`FetchWS` 行為不變。

遇到 HTTP 錯誤、空內容、challenge page（Cloudflare 等）時，error 為 `*FetchError{Status, Href}`，`Status` 可能為 4xx/5xx、`204`（空內容）或 `403`（challenge / URL heuristic），可用 `errors.As` 分流。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-pkg/rod"

defer rod.Close()

result, err := rod.Fetch(ctx, "https://example.com/article", 30*time.Second, nil)
// result.Href / result.FinalURL / result.Title / result.Author / result.PublishedAt / result.Excerpt / result.Status / result.Markdown

// 連接既有 Chrome（需以 --remote-debugging-port=9222 啟動，見下方）
result, err = rod.FetchWS(ctx, "http://127.0.0.1:9222", "https://example.com/article", 30*time.Second, nil)
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
result, err := rod.Fetch(ctx, "https://example.com/article", 20*time.Second, &rod.FetchOption{
	IdleWait:  2 * time.Second,
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

檔案系統工具集（write-side + policy）：

- **Policy 注入**：`New(Policy{DeniedMap, ExcludeList})` 一次性注入 sandbox 規則（`sync.Once`），未注入時 `IsDenied` 永遠 `false`、`IsExcluded` 僅讀執行期 `.gitignore`，**不**回 error（既有 caller 零破壞）。`IsDenied` / `IsExcluded` / `RealPath` / `AbsPath(root, path, AbsPathOption{HomeOnly, NeedExclude})` 為 public 工具；`WriteFile` / `WriteText` / `WriteJSON` / `AppendText` / `Copy` / `Move` / `Remove` 自動套 `IsDenied`（read-side 不套）
- **原子寫入**：`WriteFile` / `WriteText` / `WriteJSON` / `Copy` 透過 `.tmp` + `os.Rename`
- **目錄**：`CheckDir(path, create)` 檢查路徑是否為目錄（`create=true` 時不存在會以 `0755` 建立）
- **讀寫**：`ReadText` / `WriteText` / `AppendText`；`WriteJSON(path, v, format)`（`format=true` 縮排，`false` 緊湊）
- **搬移**：`Move` 跨 device 自動 fallback 為 copy + remove；`Remove` 忽略不存在錯誤

讀取／列舉／搜尋類 API 已拆至 `filesystem/reader` 子套件（見下節）。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-pkg/filesystem"

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
err = filesystem.CheckDir("/path/to/dir", true) // 不存在則建立

// 讀取
content, err := filesystem.ReadText("/path/to/file.txt")

// JSON
type Config struct{ Host string }
err = filesystem.WriteJSON("/path/to/cfg.json", cfg, true)  // formatted
err = filesystem.WriteJSON("/path/to/cfg.json", cfg, false) // compact

// 搬移
err = filesystem.Copy("/src", "/dst")
err = filesystem.Move("/src", "/dst")               // 跨 device 自動 fallback
err = filesystem.Remove("/path/to/x")
```

</details>

### filesystem/reader

讀取側工具：存在性檢查、列舉、遞迴 walk、glob、regex 內容搜尋。共用 variadic `ListOption{SkipExcluded, IgnoreWalkError, IncludeNonRegular bool}`：

- `SkipExcluded=true` 時依 `filesystem.IsExcluded` 過濾命中項，walk 對命中目錄回 `filepath.SkipDir`
- `IgnoreWalkError=true` 僅作用於 `WalkFiles` / `SearchFiles`，walker 收到 err 時跳過該節點而不 abort 整個 walk（`SearchFiles` 預設為 false → halt-on-error，與 `WalkFiles` 一致）
- `IncludeNonRegular=true` 放寬 `IsRegular()` 限制以收錄 symlink／device／socket／pipe（作用於 `ListFiles` / `WalkFiles` / `SearchFiles`）

| API | 行為 |
|---|---|
| `Exists` / `IsFile` / `IsDir` | 包 stat 回 bool |
| `IsEmpty(path)` | 區分目錄空（`Readdirnames(1)` EOF）與檔案 size=0 |
| `ListFiles(dir, opts...)` | 非遞迴 `[]FileMeta{Name, Path, IsDir, Size, ModTime}`，僅 regular file |
| `ListDirs(dir, opts...)` | 非遞迴名稱，僅 dir |
| `ListAll(dir, opts...)` | 非遞迴 `[]os.DirEntry`，全 type 不過濾 |
| `WalkFiles(root, opts...)` | 遞迴 slash 相對路徑，跳過 dot-prefix 目錄 |
| `GlobFiles(root, pattern)` | 支援 `**` 遞迴，回 `[]FileMeta`；入口 `filepath.Match` 驗 pattern，malformed 即冒 `ErrBadPattern` |
| `SearchFiles(root, regex, filePatterns, maxSize, opts...)` | regex 內容搜尋，回 `[]File{Path, Matches []Line{Line, Text}}`；`maxSize <= 0` fallback 為 1MB；自動跳 binary ext 與 dot-prefix |

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-pkg/filesystem/reader"

// 存在性
ok := reader.Exists("/path/to/x")
ok = reader.IsFile("/path/to/x")
ok = reader.IsDir("/path/to/x")
empty, err := reader.IsEmpty("/path/to/x")

// 列舉（非遞迴）
files, err := reader.ListFiles("/path/to/dir") // []FileMeta{Name, Path, IsDir, Size, ModTime}
dirs, err := reader.ListDirs("/path/to/dir")
entries, err := reader.ListAll("/path/to/dir")

// 遞迴 walk
all, err := reader.WalkFiles("/work", reader.ListOption{SkipExcluded: true})

// 容錯：跳過 stat 失敗節點而非 abort 整個 walk
all, err = reader.WalkFiles("/work", reader.ListOption{
    SkipExcluded:    true,
    IgnoreWalkError: true,
})

// 收錄 symlink／device／socket／pipe
files, err = reader.ListFiles("/work", reader.ListOption{IncludeNonRegular: true})

// Glob（** 遞迴；回 []FileMeta）
matches, err := reader.GlobFiles("/work", "**/*.go")

// regex 內容搜尋（每檔收集所有命中行）
hits, err := reader.SearchFiles("/work", `TODO\(.*\)`, []string{"**/*.go"}, 0,
    reader.ListOption{SkipExcluded: true})
for _, h := range hits {
    // h.Path / h.Matches[i].Line / h.Matches[i].Text
}
```

</details>

### filesystem/parser

多格式文件抽取：Markdown / PDF / DOCX / PPTX 統一回 `(string, []Chunk, error)` — 第一個 string 為全文、第二個為段落分塊。每個 `Chunk` 保留 `Source` / `Index` / `Total` / `Content`，超過 65535 rune 的段落自動以句末標點切片（中英標點皆認）。空文件回 `parser.ErrEmpty` sentinel，可用 `errors.Is` 判斷。

| API | 後端 |
|---|---|
| `Markdown(ctx, path)` | `os.ReadFile` 後依 `\n\n` / `\r\n\r\n` 分段 |
| `PDF(ctx, path)` | `pdftotext -layout` CLI（需先安裝 poppler-utils），以 `\f` 分頁 |
| `Docx(ctx, path)` | `archive/zip` + `encoding/xml`，涵蓋 `word/document.xml` 主文與 `header*` / `footer*` / `footnotes` / `endnotes` |
| `PPTX(ctx, path)` | `archive/zip` + `encoding/xml`，按 `slide(N).xml` 數字排序 |

<details>
<summary>範例</summary>

```go
import (
    "errors"
    "github.com/pardnchiu/go-pkg/filesystem/parser"
)

text, chunks, err := parser.Markdown(ctx, "./README.md")
text, chunks, err = parser.PDF(ctx, "./paper.pdf")
text, chunks, err = parser.Docx(ctx, "./report.docx")
text, chunks, err = parser.PPTX(ctx, "./slides.pptx")

if errors.Is(err, parser.ErrEmpty) {
    // 解析成功但內容為空（text 仍可能非空）
}

for _, c := range chunks {
    // c.Source / c.Index / c.Total / c.Content
}
```

</details>

### filesystem/keychain

跨平台密鑰存取（macOS Keychain / Linux secret-tool / 檔案 fallback）。`Init` 一次性綁定 service 名稱與 fallback 路徑（`sync.Once`）；`Get` 在 keychain 查無時 fallback 至 `os.Getenv`。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-pkg/filesystem/keychain"

// 初始化（sync.Once，僅首次生效）
keychain.Init("MyApp", fallbackDir)

val := keychain.Get("API_KEY")
err := keychain.Set("API_KEY", "secret")
err := keychain.Delete("API_KEY")
```

</details>

### sandbox

跨平台子行程隔離：macOS 用 `sandbox-exec`（seatbelt profile）、Linux 用 `bwrap`（bubblewrap）。`workDir` 必須在 `$HOME` 之下（強制 + symlink 解析）。

`New(deniedJSON)` 一次性注入 deny dirs / files（路徑相對 home）。`Wrap(ctx, binary, args, workDir, opt)` 回 `*exec.Cmd`，caller 自行 `Run` / `Output`。

`Option` 欄位：

| 欄位 | 平台 | 說明 |
|---|---|---|
| `CPUPercent` | 兩者 | CPU 上限（%）|
| `MemoryMB` | Linux only | 記憶體上限（MB）；macOS 傳值會回 err |
| `Network` | 兩者 | `NetworkAllow` / `NetworkDeny` |
| `DropCaps` | Linux only | 移除所有 capability |
| `MinimalBinds` | 兩者 | `WriteScope` (`WriteWork` / `WriteHome`) + `ReadOnly` / `ReadWrite` 路徑清單 |

`ParseMemory("512m")` / `ParseMemory("2GiB")` 解析常見格式為 MB 整數。`CheckDependence()` 在 macOS 永遠 nil；Linux 檢查 `bwrap` 並嘗試以 apt / dnf / yum / pacman / apk 安裝。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-pkg/sandbox"

// 一次性 deny（optional）
sandbox.New([]byte(`{"dirs":[".ssh",".aws"],"files":[".netrc"]}`))

mem, _ := sandbox.ParseMemory("1GiB") // 1024

cmd, err := sandbox.Wrap(ctx, "/usr/bin/python3", []string{"script.py"}, "/Users/me/work",
    &sandbox.Option{
        CPUPercent: 50,
        MemoryMB:   mem,
        Network:    sandbox.NetworkDeny,
        DropCaps:   true,
        MinimalBinds: &sandbox.BindSpec{
            WriteScope: sandbox.WriteWork,
            ReadOnly:   []string{"/usr", "/lib"},
        },
    })
out, err := cmd.CombinedOutput()
```

</details>

### utils

通用小工具。`UUID()` 產生 RFC 4122 v4 UUID。`GetWithDefault` / `GetWithDefaultInt` / `GetWithDefaultFloat` 讀取環境變數並於未設定、空字串或解析失敗時回傳預設值。

<details>
<summary>範例</summary>

```go
import "github.com/pardnchiu/go-pkg/utils"

id := utils.UUID() // e.g. "f47ac10b-58cc-4372-a567-0e02b2c3d479"

host := utils.GetWithDefault("HOST", "localhost")
port := utils.GetWithDefaultInt("PORT", 8080)
ratio := utils.GetWithDefaultFloat("RATIO", 1.5)
```

</details>
