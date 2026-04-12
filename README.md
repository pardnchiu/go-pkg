# go-utils

個人開發常用的 Go 工具函式集合，從過往專案中逐步累積而成。

## 內容

### http

泛型 HTTP GET/POST，自動處理 JSON/XML 解碼。

```go
import "github.com/pardnchiu/go-utils/http"

// GET
data, status, err := http.GET[MyStruct](ctx, nil, "https://api.example.com/data", nil)

// POST（JSON）
data, status, err := http.POST[MyStruct](ctx, nil, "https://api.example.com/data", nil, body, "json")

// POST（Form）
data, status, err := http.POST[MyStruct](ctx, nil, "https://api.example.com/data", nil, body, "form")
```

### filesystem

原子化檔案寫入（先寫 `.tmp` 再 rename）。

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
