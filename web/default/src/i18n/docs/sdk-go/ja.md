Go では HTTP を直接呼ぶか、OpenAI 互換 SDK を使えます。

> コード例と API パスは技術識別子のため英語のままです。

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
