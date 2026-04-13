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
