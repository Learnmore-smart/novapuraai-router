Với Go, gọi HTTP trực tiếp hoặc dùng SDK tương thích OpenAI.

> Ví dụ mã và đường dẫn API giữ nguyên tiếng Anh (định danh kỹ thuật).

Go clients can call the HTTP API directly or use an OpenAI-compatible Go SDK.

## Direct HTTP

```go
package main

import (
  "bytes"
  "fmt"
  "net/http"
  "os"
)

func main() {
  body := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello"}]}`)
  req, _ := http.NewRequest("POST", "https://www.novapuraai.com/v1/chat/completions", bytes.NewReader(body))
  req.Header.Set("Authorization", "Bearer "+os.Getenv("NOVAPURA_API_KEY"))
  req.Header.Set("Content-Type", "application/json")
  resp, err := http.DefaultClient.Do(req)
  if err != nil { panic(err) }
  defer resp.Body.Close()
  fmt.Println(resp.Status)
}
```

## Tips

- Set reasonable timeouts for non-stream and longer timeouts for streaming or media.
- Propagate context cancellation to abort in-flight requests when handlers return.
