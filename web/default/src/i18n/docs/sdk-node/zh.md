使用官方 `openai` npm 包。

## 安装

```bash
npm install openai
# or: bun add openai
```

## 客户端

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

## 对话

```javascript
const completion = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(completion.choices[0].message.content);
```

## 流式输出

```javascript
const stream = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Stream digits 1-5" }],
  stream: true,
});
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || "");
}
```
