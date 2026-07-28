公式 npm パッケージ `openai` を利用します。

> コード例と API パスは技術識別子のため英語のままです。

Use the official `openai` npm package.

## Install

```bash
npm install openai
# or: bun add openai
```

## Client

```javascript
import OpenAI from 'openai'

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: 'https://www.novapuraai.com/v1',
})
```

## Chat

```javascript
const completion = await client.chat.completions.create({
  model: 'gpt-4o-mini',
  messages: [{ role: 'user', content: 'Hello' }],
})
console.log(completion.choices[0].message.content)
```

## Streaming

```javascript
const stream = await client.chat.completions.create({
  model: 'gpt-4o-mini',
  messages: [{ role: 'user', content: 'Stream digits 1-5' }],
  stream: true,
})
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '')
}
```
