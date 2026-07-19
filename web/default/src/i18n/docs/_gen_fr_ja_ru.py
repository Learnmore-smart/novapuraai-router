# -*- coding: utf-8 -*-
"""One-shot generator: official NovaPuraAI docs for fr / ja / ru."""
from pathlib import Path

ROOT = Path(__file__).resolve().parent

# ---------------------------------------------------------------------------
# Content: section -> {lang: markdown}
# No top-level # titles. Use ## / ### only.
# Brand names and code identifiers stay in English where appropriate.
# ---------------------------------------------------------------------------

DOCS: dict[str, dict[str, str]] = {}

# ========================= quickstart =========================
DOCS["quickstart"] = {
    "fr": r"""## Bienvenue

NovaPuraAI est une passerelle API compatible OpenAI. Elle unifie de nombreux fournisseurs d’IA derrière une seule base URL, une authentification simple et une facturation centralisée.

Avec NovaPuraAI, vous pouvez :

- Appeler les modèles via l’API **Chat Completions** (`/v1/chat/completions`)
- Utiliser les formats natifs **Claude Messages** et **Gemini**
- Générer des embeddings, images, audio et résultats de rerank
- Brancher Cursor, LangChain, Dify, OpenWebUI et d’autres outils OpenAI-compatibles

## Prérequis

1. Un compte sur [NovaPuraAI](https://www.novapuraai.com)
2. Une clé API (`sk-xxxxx`) créée dans la console
3. Un solde ou un quota suffisant pour vos tests

## En 3 étapes

### 1. Obtenir une clé API

Connectez-vous à la console, ouvrez **API Keys**, puis créez une clé. Conservez-la en secret ; elle ne sera plus affichée en clair après création.

### 2. Configurer la base URL

Utilisez :

```text
https://www.novapuraai.com
```

Pour la plupart des SDK OpenAI, la base effective est :

```text
https://www.novapuraai.com/v1
```

### 3. Envoyer une première requête

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Bonjour !"}
    ]
  }'
```

## Suite recommandée

- [Authentification](/docs/authentication)
- [Votre première requête](/docs/first-request)
- [Base URL et endpoints](/docs/base-url)
- [Chat Completions](/docs/api-chat)
""",
    "ja": r"""## はじめに

NovaPuraAI は OpenAI 互換の API ゲートウェイです。複数の AI プロバイダーを 1 つの Base URL・共通認証・一元課金の背後に集約します。

NovaPuraAI でできること:

- **Chat Completions** API（`/v1/chat/completions`）でモデルを呼び出す
- ネイティブな **Claude Messages** / **Gemini** 形式を利用する
- Embeddings、画像、音声、Rerank を実行する
- Cursor、LangChain、Dify、OpenWebUI など OpenAI 互換ツールと接続する

## 前提条件

1. [NovaPuraAI](https://www.novapuraai.com) のアカウント
2. コンソールで発行した API キー（`sk-xxxxx`）
3. テストに十分な残高またはクォータ

## 3 ステップで開始

### 1. API キーを取得

コンソールにログインし、**API Keys** からキーを作成します。作成後は平文で再表示されないため、安全に保管してください。

### 2. Base URL を設定

次を使用します:

```text
https://www.novapuraai.com
```

多くの OpenAI SDK では、実質的な base は次のとおりです:

```text
https://www.novapuraai.com/v1
```

### 3. 最初のリクエストを送信

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "こんにちは！"}
    ]
  }'
```

## 次のステップ

- [認証](/docs/authentication)
- [最初のリクエスト](/docs/first-request)
- [Base URL とエンドポイント](/docs/base-url)
- [Chat Completions](/docs/api-chat)
""",
    "ru": r"""## Добро пожаловать

NovaPuraAI — API-шлюз, совместимый с OpenAI. Он объединяет множество AI-провайдеров за единым Base URL, простой аутентификацией и централизованным биллингом.

С NovaPuraAI вы можете:

- Вызывать модели через API **Chat Completions** (`/v1/chat/completions`)
- Использовать нативные форматы **Claude Messages** и **Gemini**
- Создавать embeddings, изображения, аудио и результаты rerank
- Подключать Cursor, LangChain, Dify, OpenWebUI и другие OpenAI-совместимые инструменты

## Требования

1. Аккаунт на [NovaPuraAI](https://www.novapuraai.com)
2. API-ключ (`sk-xxxxx`), созданный в консоли
3. Достаточный баланс или квота для тестов

## За 3 шага

### 1. Получите API-ключ

Войдите в консоль, откройте **API Keys** и создайте ключ. Храните его в секрете — после создания он больше не показывается в открытом виде.

### 2. Укажите Base URL

Используйте:

```text
https://www.novapuraai.com
```

Для большинства OpenAI SDK фактический base:

```text
https://www.novapuraai.com/v1
```

### 3. Отправьте первый запрос

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Привет!"}
    ]
  }'
```

## Что дальше

- [Аутентификация](/docs/authentication)
- [Ваш первый запрос](/docs/first-request)
- [Base URL и эндпоинты](/docs/base-url)
- [Chat Completions](/docs/api-chat)
""",
}

# ========================= authentication =========================
DOCS["authentication"] = {
    "fr": r"""## Vue d’ensemble

Toutes les requêtes vers l’API NovaPuraAI doivent être authentifiées. Le mode principal est un jeton **Bearer** dans l’en-tête `Authorization`.

## Clés API

Les clés sont créées dans la console (**API Keys**). Elles commencent généralement par `sk-`.

Exemple d’en-tête :

```http
Authorization: Bearer sk-xxxxx
```

### Bonnes pratiques

- Ne commitez jamais une clé dans un dépôt public
- Préférez les variables d’environnement (`NOVAPURA_API_KEY`, `OPENAI_API_KEY`, etc.)
- Limitez le périmètre et le quota des clés de production
- Révoquez immédiatement une clé compromise

## Formats d’authentification

### OpenAI-compatible (recommandé)

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

### Claude Messages

Pour `/v1/messages`, vous pouvez utiliser le style Anthropic :

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "x-api-key: sk-xxxxx" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "messages": [{"role": "user", "content": "Bonjour"}]
  }'
```

L’en-tête `Authorization: Bearer sk-xxxxx` est aussi accepté sur la plupart des chemins.

### Gemini

Pour les routes Gemini (`/v1beta/models/...`), la clé peut être fournie via :

- `Authorization: Bearer sk-xxxxx`
- ou le paramètre de requête `key=sk-xxxxx` (style Google)

## Erreurs d’authentification courantes

| Code | Signification | Action |
|------|---------------|--------|
| `401` | Clé manquante ou invalide | Vérifier l’en-tête et la clé |
| `403` | Accès refusé / clé désactivée | Contrôler le statut de la clé dans la console |
| `429` | Limite de débit ou quota | Ralentir ou recharger le solde |

## Sécurité

- Transmettez toujours la clé en HTTPS
- N’exposez pas la clé côté navigateur non protégé
- Utilisez des clés distinctes par environnement (dev / staging / prod)
""",
    "ja": r"""## 概要

NovaPuraAI API へのすべてのリクエストは認証が必要です。主な方式は `Authorization` ヘッダーの **Bearer** トークンです。

## API キー

キーはコンソール（**API Keys**）で作成します。通常 `sk-` で始まります。

ヘッダー例:

```http
Authorization: Bearer sk-xxxxx
```

### ベストプラクティス

- 公開リポジトリにキーをコミットしない
- 環境変数（`NOVAPURA_API_KEY`、`OPENAI_API_KEY` など）を使う
- 本番キーの権限とクォータを最小化する
- 漏洩したキーは直ちに無効化する

## 認証フォーマット

### OpenAI 互換（推奨）

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

### Claude Messages

`/v1/messages` では Anthropic 形式も利用できます:

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "x-api-key: sk-xxxxx" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "messages": [{"role": "user", "content": "こんにちは"}]
  }'
```

ほとんどのパスで `Authorization: Bearer sk-xxxxx` も利用できます。

### Gemini

Gemini ルート（`/v1beta/models/...`）では次のいずれかでキーを渡せます:

- `Authorization: Bearer sk-xxxxx`
- クエリパラメータ `key=sk-xxxxx`（Google 形式）

## よくある認証エラー

| コード | 意味 | 対処 |
|--------|------|------|
| `401` | キー欠落または無効 | ヘッダーとキーを確認 |
| `403` | アクセス拒否 / キー無効化 | コンソールでキー状態を確認 |
| `429` | レート制限またはクォータ | 間隔を空ける、残高を補充 |

## セキュリティ

- 常に HTTPS でキーを送信する
- 保護されていないブラウザ側にキーを置かない
- 環境ごとにキーを分ける（dev / staging / prod）
""",
    "ru": r"""## Обзор

Все запросы к API NovaPuraAI должны быть аутентифицированы. Основной способ — токен **Bearer** в заголовке `Authorization`.

## API-ключи

Ключи создаются в консоли (**API Keys**). Обычно они начинаются с `sk-`.

Пример заголовка:

```http
Authorization: Bearer sk-xxxxx
```

### Рекомендации

- Не коммитьте ключи в публичные репозитории
- Используйте переменные окружения (`NOVAPURA_API_KEY`, `OPENAI_API_KEY` и т.д.)
- Ограничивайте область и квоту production-ключей
- Немедленно отзывайте скомпрометированный ключ

## Форматы аутентификации

### OpenAI-compatible (рекомендуется)

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

### Claude Messages

Для `/v1/messages` можно использовать стиль Anthropic:

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "x-api-key: sk-xxxxx" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "messages": [{"role": "user", "content": "Привет"}]
  }'
```

Заголовок `Authorization: Bearer sk-xxxxx` также принимается на большинстве путей.

### Gemini

Для маршрутов Gemini (`/v1beta/models/...`) ключ можно передать так:

- `Authorization: Bearer sk-xxxxx`
- или query-параметр `key=sk-xxxxx` (стиль Google)

## Типичные ошибки аутентификации

| Код | Значение | Действие |
|-----|----------|----------|
| `401` | Ключ отсутствует или неверен | Проверить заголовок и ключ |
| `403` | Доступ запрещён / ключ отключён | Проверить статус ключа в консоли |
| `429` | Rate limit или квота | Снизить частоту или пополнить баланс |

## Безопасность

- Всегда передавайте ключ по HTTPS
- Не храните ключ в незащищённом клиентском коде
- Используйте отдельные ключи для dev / staging / prod
""",
}

# ========================= first-request =========================
DOCS["first-request"] = {
    "fr": r"""## Objectif

Envoyer une requête de chat réussie et lire la réponse.

## Exemple curl

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "Tu es un assistant utile."},
      {"role": "user", "content": "Explique NovaPuraAI en une phrase."}
    ],
    "temperature": 0.7
  }'
```

## Exemple Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxx",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "user", "content": "Bonjour depuis Python"},
    ],
)
print(resp.choices[0].message.content)
```

## Exemple Node.js

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "sk-xxxxx",
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Bonjour depuis Node.js" }],
});

console.log(resp.choices[0].message.content);
```

## Comprendre la réponse

Une réponse réussie ressemble à :

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 20,
    "total_tokens": 32
  }
}
```

Les champs utiles :

- `choices[0].message.content` — texte généré
- `usage` — tokens facturés
- `finish_reason` — motif d’arrêt (`stop`, `length`, etc.)

## Streaming

Ajoutez `"stream": true` pour recevoir des événements SSE :

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Compte jusqu’à 5"}]
  }'
```

## Dépannage rapide

- **401** : clé incorrecte ou en-tête manquant
- **400** : corps JSON invalide ou modèle inconnu
- **402 / solde insuffisant** : recharger le compte dans la console
- **404 modèle** : lister les modèles via `GET /v1/models`
""",
    "ja": r"""## 目的

成功するチャットリクエストを送り、レスポンスを読み取ります。

## curl の例

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "あなたは役立つアシスタントです。"},
      {"role": "user", "content": "NovaPuraAI を一文で説明してください。"}
    ],
    "temperature": 0.7
  }'
```

## Python の例

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxx",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "user", "content": "Python からこんにちは"},
    ],
)
print(resp.choices[0].message.content)
```

## Node.js の例

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "sk-xxxxx",
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Node.js からこんにちは" }],
});

console.log(resp.choices[0].message.content);
```

## レスポンスの読み方

成功時の例:

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 20,
    "total_tokens": 32
  }
}
```

主なフィールド:

- `choices[0].message.content` — 生成テキスト
- `usage` — 課金対象トークン
- `finish_reason` — 終了理由（`stop`、`length` など）

## ストリーミング

`"stream": true` を付けると SSE で受信できます:

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "1 から 5 まで数えて"}]
  }'
```

## 簡単なトラブルシュート

- **401**: キー不正またはヘッダー不足
- **400**: JSON 不正または不明なモデル
- **402 / 残高不足**: コンソールでチャージ
- **404 モデル**: `GET /v1/models` で一覧確認
""",
    "ru": r"""## Цель

Отправить успешный chat-запрос и разобрать ответ.

## Пример curl

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "Ты полезный ассистент."},
      {"role": "user", "content": "Объясни NovaPuraAI одним предложением."}
    ],
    "temperature": 0.7
  }'
```

## Пример Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxx",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "user", "content": "Привет из Python"},
    ],
)
print(resp.choices[0].message.content)
```

## Пример Node.js

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "sk-xxxxx",
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Привет из Node.js" }],
});

console.log(resp.choices[0].message.content);
```

## Как читать ответ

Успешный ответ выглядит так:

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 20,
    "total_tokens": 32
  }
}
```

Полезные поля:

- `choices[0].message.content` — сгенерированный текст
- `usage` — токены для биллинга
- `finish_reason` — причина остановки (`stop`, `length` и т.д.)

## Streaming

Добавьте `"stream": true`, чтобы получать SSE-события:

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Посчитай до 5"}]
  }'
```

## Быстрая диагностика

- **401**: неверный ключ или отсутствует заголовок
- **400**: неверный JSON или неизвестная модель
- **402 / недостаточно баланса**: пополните счёт в консоли
- **404 модель**: список моделей через `GET /v1/models`
""",
}

# ========================= base-url =========================
DOCS["base-url"] = {
    "fr": r"""## Base URL

Base URL officielle :

```text
https://www.novapuraai.com
```

Pour les clients OpenAI SDK, configurez généralement :

```text
https://www.novapuraai.com/v1
```

Le SDK ajoute ensuite les chemins relatifs (`/chat/completions`, `/embeddings`, …).

## Endpoints principaux

| Capacité | Méthode | Chemin |
|----------|---------|--------|
| Chat Completions | `POST` | `/v1/chat/completions` |
| Completions (legacy) | `POST` | `/v1/completions` |
| Responses | `POST` | `/v1/responses` |
| Claude Messages | `POST` | `/v1/messages` |
| Embeddings | `POST` | `/v1/embeddings` |
| Images | `POST` | `/v1/images/generations` |
| Audio Speech (TTS) | `POST` | `/v1/audio/speech` |
| Audio Transcription | `POST` | `/v1/audio/transcriptions` |
| Rerank | `POST` | `/v1/rerank` |
| Models | `GET` | `/v1/models` |
| Gemini generateContent | `POST` | `/v1beta/models/{model}:generateContent` |
| Gemini streamGenerateContent | `POST` | `/v1beta/models/{model}:streamGenerateContent` |

## Configuration SDK

### Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxx",
    base_url="https://www.novapuraai.com/v1",
)
```

### Node.js

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

### Variables d’environnement courantes

```bash
export OPENAI_API_KEY="sk-xxxxx"
export OPENAI_BASE_URL="https://www.novapuraai.com/v1"
```

## Remarques

- N’ajoutez pas de slash final superflu selon le client : la plupart des SDK gèrent `/v1` sans slash final.
- Si un outil demande « API Host » sans `/v1`, essayez `https://www.novapuraai.com` puis `https://www.novapuraai.com/v1`.
- Les chemins Claude et Gemini restent sous le même hôte, avec leurs préfixes respectifs.
""",
    "ja": r"""## Base URL

公式 Base URL:

```text
https://www.novapuraai.com
```

OpenAI SDK では通常次を設定します:

```text
https://www.novapuraai.com/v1
```

SDK がその後の相対パス（`/chat/completions`、`/embeddings` など）を付与します。

## 主なエンドポイント

| 機能 | メソッド | パス |
|------|----------|------|
| Chat Completions | `POST` | `/v1/chat/completions` |
| Completions（レガシー） | `POST` | `/v1/completions` |
| Responses | `POST` | `/v1/responses` |
| Claude Messages | `POST` | `/v1/messages` |
| Embeddings | `POST` | `/v1/embeddings` |
| Images | `POST` | `/v1/images/generations` |
| Audio Speech（TTS） | `POST` | `/v1/audio/speech` |
| Audio Transcription | `POST` | `/v1/audio/transcriptions` |
| Rerank | `POST` | `/v1/rerank` |
| Models | `GET` | `/v1/models` |
| Gemini generateContent | `POST` | `/v1beta/models/{model}:generateContent` |
| Gemini streamGenerateContent | `POST` | `/v1beta/models/{model}:streamGenerateContent` |

## SDK 設定

### Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxx",
    base_url="https://www.novapuraai.com/v1",
)
```

### Node.js

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

### よく使う環境変数

```bash
export OPENAI_API_KEY="sk-xxxxx"
export OPENAI_BASE_URL="https://www.novapuraai.com/v1"
```

## 注意点

- 末尾スラッシュの扱いはクライアントによって異なります。多くの SDK は末尾スラッシュなしの `/v1` が安全です。
- ツールが `/v1` なしの「API Host」を求める場合は、`https://www.novapuraai.com` と `https://www.novapuraai.com/v1` の両方を試してください。
- Claude / Gemini のパスも同じホスト上で、それぞれのプレフィックスを使います。
""",
    "ru": r"""## Base URL

Официальный Base URL:

```text
https://www.novapuraai.com
```

Для OpenAI SDK обычно указывают:

```text
https://www.novapuraai.com/v1
```

Далее SDK добавляет относительные пути (`/chat/completions`, `/embeddings` и т.д.).

## Основные эндпоинты

| Возможность | Метод | Путь |
|-------------|-------|------|
| Chat Completions | `POST` | `/v1/chat/completions` |
| Completions (legacy) | `POST` | `/v1/completions` |
| Responses | `POST` | `/v1/responses` |
| Claude Messages | `POST` | `/v1/messages` |
| Embeddings | `POST` | `/v1/embeddings` |
| Images | `POST` | `/v1/images/generations` |
| Audio Speech (TTS) | `POST` | `/v1/audio/speech` |
| Audio Transcription | `POST` | `/v1/audio/transcriptions` |
| Rerank | `POST` | `/v1/rerank` |
| Models | `GET` | `/v1/models` |
| Gemini generateContent | `POST` | `/v1beta/models/{model}:generateContent` |
| Gemini streamGenerateContent | `POST` | `/v1beta/models/{model}:streamGenerateContent` |

## Настройка SDK

### Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxx",
    base_url="https://www.novapuraai.com/v1",
)
```

### Node.js

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

### Типичные переменные окружения

```bash
export OPENAI_API_KEY="sk-xxxxx"
export OPENAI_BASE_URL="https://www.novapuraai.com/v1"
```

## Примечания

- Лишний завершающий слэш зависит от клиента: для большинства SDK безопасен `/v1` без слэша.
- Если инструмент просит «API Host» без `/v1`, попробуйте `https://www.novapuraai.com`, затем `https://www.novapuraai.com/v1`.
- Пути Claude и Gemini находятся на том же хосте со своими префиксами.
""",
}

# ========================= routing =========================
DOCS["routing"] = {
    "fr": r"""## Modèles et routage

NovaPuraAI route chaque requête vers un canal (channel) amont capable de servir le modèle demandé. Vous choisissez le **nom de modèle** ; la passerelle sélectionne le fournisseur et le chemin adaptés.

## Choisir un modèle

1. Consultez la liste dans la console (pricing / modèles)
2. Ou interrogez l’API :

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

Utilisez exactement l’identifiant retourné dans le champ `id` (ex. `gpt-4o-mini`, `claude-3-5-sonnet-20241022`, `gemini-2.0-flash`).

## Comment fonctionne le routage

Pour une requête typique :

1. Authentification de la clé API
2. Validation du modèle et des paramètres
3. Sélection d’un canal disponible pour ce modèle
4. Relais vers le fournisseur amont
5. Normalisation de la réponse (selon le format d’API)
6. Décompte du quota / usage

## Formats multi-protocole

Le même modèle peut parfois être appelé via plusieurs formats :

| Format | Endpoint | Usage typique |
|--------|----------|---------------|
| OpenAI Chat | `/v1/chat/completions` | SDK OpenAI, la plupart des apps |
| Claude Messages | `/v1/messages` | Outils Anthropic-native |
| Gemini | `/v1beta/models/*` | Clients Google GenAI |

Choisissez le format attendu par votre client ; la passerelle s’occupe du relais.

## Groupes et priorités

Selon la configuration du compte / de la plateforme :

- Certains modèles peuvent être limités à des groupes d’utilisateurs
- Des canaux de secours (fallback) peuvent basculer en cas d’erreur amont
- La disponibilité réelle dépend de l’état des canaux

Si un modèle figure dans la liste mais renvoie une erreur temporaire, réessayez ou basculez vers un modèle équivalent.

## Conseils

- Préférez des IDs de modèle stables documentés dans la console
- Ne supposez pas qu’un alias marketing est toujours accepté tel quel
- Pour la production, testez le modèle cible avec une clé dédiée avant le basculement
""",
    "ja": r"""## モデルとルーティング

NovaPuraAI は、要求されたモデルを提供できる上流チャネルへ各リクエストをルーティングします。開発者は **モデル名** を指定し、ゲートウェイがプロバイダーと経路を選択します。

## モデルの選び方

1. コンソールの料金 / モデル一覧を確認する
2. または API で取得する:

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

返却された `id` をそのまま使ってください（例: `gpt-4o-mini`、`claude-3-5-sonnet-20241022`、`gemini-2.0-flash`）。

## ルーティングの流れ

典型的なリクエストでは:

1. API キーの認証
2. モデルとパラメータの検証
3. そのモデルに対応する利用可能チャネルの選択
4. 上流プロバイダーへのリレー
5. 応答の正規化（API 形式に応じて）
6. クォータ / 使用量の計上

## マルチプロトコル形式

同じモデルを複数形式で呼べる場合があります:

| 形式 | エンドポイント | 典型的な用途 |
|------|----------------|--------------|
| OpenAI Chat | `/v1/chat/completions` | OpenAI SDK、多くのアプリ |
| Claude Messages | `/v1/messages` | Anthropic ネイティブツール |
| Gemini | `/v1beta/models/*` | Google GenAI クライアント |

クライアントが期待する形式を選び、リレーはゲートウェイに任せます。

## グループと優先度

アカウント / プラットフォーム設定により:

- 一部モデルはユーザーグループに制限されることがあります
- 上流障害時にフォールバックチャネルへ切り替わることがあります
- 実際の可用性はチャネル状態に依存します

一覧にあるモデルでも一時エラーが出る場合は、再試行するか同等モデルへ切り替えてください。

## ヒント

- コンソールに記載の安定したモデル ID を使う
- マーケティング上の別名がそのまま通るとは限らない
- 本番切替前に専用キーで対象モデルを検証する
""",
    "ru": r"""## Модели и маршрутизация

NovaPuraAI направляет каждый запрос на upstream-канал, способный обслужить запрошенную модель. Вы указываете **имя модели**; шлюз выбирает провайдера и маршрут.

## Как выбрать модель

1. Посмотрите список в консоли (pricing / models)
2. Или запросите API:

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

Используйте точный `id` из ответа (например `gpt-4o-mini`, `claude-3-5-sonnet-20241022`, `gemini-2.0-flash`).

## Как работает маршрутизация

Типичный запрос:

1. Аутентификация API-ключа
2. Валидация модели и параметров
3. Выбор доступного канала для модели
4. Релей на upstream-провайдера
5. Нормализация ответа (в зависимости от формата API)
6. Списание квоты / usage

## Мультипротокольные форматы

Одну и ту же модель иногда можно вызывать в разных форматах:

| Формат | Endpoint | Типичное использование |
|--------|----------|------------------------|
| OpenAI Chat | `/v1/chat/completions` | OpenAI SDK, большинство приложений |
| Claude Messages | `/v1/messages` | Anthropic-native инструменты |
| Gemini | `/v1beta/models/*` | Google GenAI клиенты |

Выбирайте формат, который ожидает ваш клиент; релей выполняет шлюз.

## Группы и приоритеты

В зависимости от настроек аккаунта / платформы:

- Некоторые модели могут быть ограничены группами пользователей
- Fallback-каналы могут переключаться при ошибках upstream
- Фактическая доступность зависит от состояния каналов

Если модель есть в списке, но временно недоступна, повторите запрос или переключитесь на эквивалентную модель.

## Советы

- Предпочитайте стабильные model ID из консоли
- Не предполагайте, что маркетинговый алиас всегда принимается as-is
- Перед production-переключением проверьте целевую модель отдельным ключом
""",
}

# ========================= billing =========================
DOCS["billing"] = {
    "fr": r"""## Facturation et quota

NovaPuraAI facture l’usage via un **quota** (crédit) associé à votre compte et/ou à votre clé API. Chaque requête réussie consomme du quota selon le modèle, les tokens et d’éventuels multiplicateurs (image, audio, etc.).

## Comment le quota est calculé

En résumé :

1. Estimation / pré-consommation avant l’appel amont (selon le type de requête)
2. Exécution de la requête
3. Règlement final selon l’usage réel (tokens, unités média, etc.)
4. Remboursement partiel possible si l’estimation dépassait l’usage réel

Les détails de prix par modèle sont affichés dans la console (page de tarification).

## Ce qui influence le coût

- **Modèle** : les modèles plus puissants coûtent généralement plus cher
- **Tokens d’entrée / sortie** : longueur du prompt et de la complétion
- **Streaming** : le streaming n’est en général pas un supplément de format, le coût suit l’usage
- **Media** : images, audio, vidéo et rerank peuvent utiliser des unités ou ratios distincts
- **Paramètres** : `n`, résolution, durée, qualité, etc. peuvent multiplier le coût

## Suivi de l’usage

Dans la console :

- Consultez le solde / quota restant
- Inspectez les logs de consommation par requête
- Gérez les recharges (top-up) et promotions éventuelles

## Clés et limites de budget

Vous pouvez souvent :

- Créer plusieurs clés pour isoler des projets
- Appliquer des plafonds de quota par clé
- Révoquer une clé pour stopper immédiatement la consommation

## Exemple de lecture `usage`

```json
{
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 80,
    "total_tokens": 200
  }
}
```

Ces compteurs servent de base à la facturation pour les endpoints texte. Les endpoints média peuvent renvoyer d’autres indicateurs selon le modèle.

## Conseils pour maîtriser les coûts

- Fixez `max_tokens` raisonnable
- Utilisez des modèles plus petits pour le brouillon / classification
- Surveillez les retries automatiques côté client
- Séparez les clés de dev et de production
""",
    "ja": r"""## 課金とクォータ

NovaPuraAI は、アカウントおよび/または API キーに紐づく **クォータ**（クレジット）で利用量を課金します。成功したリクエストごとに、モデル・トークン数・倍率（画像、音声など）に応じてクォータが消費されます。

## クォータ計算の流れ

概要:

1. 上流呼び出し前の見積もり / 事前消費（リクエスト種別による）
2. リクエスト実行
3. 実使用量（トークン、メディア単位など）に基づく最終精算
4. 見積もりが実使用を上回った場合の部分返金の可能性

モデル別の価格はコンソールの料金ページで確認できます。

## コストに影響する要素

- **モデル**: 高性能モデルほど一般に高額
- **入力 / 出力トークン**: プロンプトと完了の長さ
- **ストリーミング**: 通常は形式自体の割増ではなく、使用量に従う
- **メディア**: 画像・音声・動画・Rerank は別単位や比率になることがある
- **パラメータ**: `n`、解像度、時間、品質などが倍率になることがある

## 使用量の確認

コンソールで:

- 残高 / 残りクォータを確認
- リクエスト単位の消費ログを確認
- チャージ（top-up）とプロモーションを管理

## キーと予算制限

多くの場合、次が可能です:

- プロジェクト分離用に複数キーを作成
- キー単位のクォータ上限を設定
- キー無効化で即時に消費を停止

## `usage` の読み方

```json
{
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 80,
    "total_tokens": 200
  }
}
```

テキスト系エンドポイントではこれらが課金の基礎になります。メディア系はモデルにより別指標が返ることがあります。

## コスト管理のヒント

- `max_tokens` を適切に設定する
- 下書き / 分類には小型モデルを使う
- クライアントの自動リトライを監視する
- 開発用と本番用のキーを分ける
""",
    "ru": r"""## Биллинг и квота

NovaPuraAI тарифицирует использование через **квоту** (кредит), привязанную к аккаунту и/или API-ключу. Каждый успешный запрос списывает квоту в зависимости от модели, токенов и множителей (image, audio и т.д.).

## Как считается квота

В общих чертах:

1. Оценка / pre-consume до upstream-вызова (зависит от типа запроса)
2. Выполнение запроса
3. Финальный расчёт по фактическому usage (токены, media-единицы и т.д.)
4. Частичный возврат, если оценка превысила фактическое потребление

Цены по моделям смотрите в консоли (страница pricing).

## Что влияет на стоимость

- **Модель**: более мощные модели обычно дороже
- **Входные / выходные токены**: длина prompt и completion
- **Streaming**: обычно не отдельная наценка формата; стоимость следует usage
- **Media**: images, audio, video и rerank могут использовать иные единицы/коэффициенты
- **Параметры**: `n`, resolution, duration, quality и т.п. могут умножать стоимость

## Мониторинг usage

В консоли:

- Смотрите остаток баланса / квоты
- Проверяйте логи потребления по запросам
- Управляйте пополнением (top-up) и акциями

## Ключи и лимиты бюджета

Часто доступно:

- Создавать несколько ключей для изоляции проектов
- Задавать потолок квоты на ключ
- Отзывать ключ, чтобы мгновенно остановить расход

## Как читать `usage`

```json
{
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 80,
    "total_tokens": 200
  }
}
```

Для текстовых эндпоинтов эти счётчики — база биллинга. Media-эндпоинты могут возвращать другие показатели.

## Как контролировать расходы

- Задавайте разумный `max_tokens`
- Для черновиков / классификации используйте более лёгкие модели
- Следите за автоматическими retry на клиенте
- Разделяйте dev- и production-ключи
""",
}

# ========================= rate-limits =========================
DOCS["rate-limits"] = {
    "fr": r"""## Limites de débit

NovaPuraAI applique des limites de débit (rate limits) pour protéger la plateforme et garantir une qualité de service équitable. Les limites peuvent s’appliquer par clé, utilisateur, modèle ou globalement selon la configuration.

## Symptômes

En cas de dépassement, l’API renvoie généralement **HTTP 429** avec un message d’erreur indiquant un trop grand nombre de requêtes.

## Comportement recommandé côté client

### Backoff exponentiel

```python
import time
import random
import requests

def post_with_retry(url, headers, json, max_attempts=5):
    delay = 0.5
    for attempt in range(max_attempts):
        r = requests.post(url, headers=headers, json=json, timeout=60)
        if r.status_code != 429:
            return r
        time.sleep(delay + random.random() * 0.2)
        delay = min(delay * 2, 16)
    return r
```

### Bonnes pratiques

- Évitez les boucles serrées sans attente
- Mutualisez les connexions HTTP (keep-alive)
- Parallelisez avec un plafond de concurrence raisonnable
- Préférez le batching quand le cas d’usage le permet
- Surveillez les 429 dans vos métriques

## Rate limit vs quota

| Concept | Signification |
|---------|----------------|
| **Rate limit** | Trop de requêtes par unité de temps |
| **Quota / solde** | Crédit insuffisant pour payer la requête |

Les deux peuvent produire des erreurs « trop de requêtes » ou d’accès refusé selon le cas ; lisez le corps d’erreur pour distinguer.

## Streaming et longues requêtes

Les requêtes longues (streaming, médias) occupent des ressources plus longtemps. Même à faible QPS, un grand nombre de connexions simultanées peut déclencher des limites. Dimensionnez la concurrence, pas seulement le débit moyen.

## Que faire en production

1. Implémenter retry + jitter uniquement sur les erreurs retriables
2. Mettre en file d’attente les pics de trafic
3. Répartir la charge sur plusieurs clés seulement si la politique du compte l’autorise
4. Contacter le support si vous avez besoin de limites plus élevées pour un usage légitime
""",
    "ja": r"""## レート制限

NovaPuraAI はプラットフォーム保護と公平な品質のためレート制限を適用します。制限はキー、ユーザー、モデル、または全体単位で設定される場合があります。

## 症状

上限超過時、API は通常 **HTTP 429** と「リクエスト過多」を示すエラーを返します。

## クライアント側の推奨動作

### 指数バックオフ

```python
import time
import random
import requests

def post_with_retry(url, headers, json, max_attempts=5):
    delay = 0.5
    for attempt in range(max_attempts):
        r = requests.post(url, headers=headers, json=json, timeout=60)
        if r.status_code != 429:
            return r
        time.sleep(delay + random.random() * 0.2)
        delay = min(delay * 2, 16)
    return r
```

### ベストプラクティス

- 待機なしの高頻度ループを避ける
- HTTP keep-alive で接続を再利用する
- 並列度に上限を設ける
- 可能なケースではバッチ処理を使う
- メトリクスで 429 を監視する

## レート制限とクォータ

| 概念 | 意味 |
|------|------|
| **Rate limit** | 単位時間あたりのリクエスト過多 |
| **Quota / 残高** | リクエスト支払いに十分なクレジットがない |

どちらも「多すぎる」やアクセス拒否として見えることがあります。エラー本文で区別してください。

## ストリーミングと長時間リクエスト

長いリクエスト（streaming、メディア）はリソースをより長く占有します。QPS が低くても同時接続が多いと制限に達することがあります。平均スループットだけでなく同時実行数も設計してください。

## 本番での対応

1. 再試行可能なエラーにのみ retry + jitter を実装
2. トラフィックのピークをキューイング
3. アカウントポリシーが許す場合のみ複数キーへ分散
4. 正当な高負荷用途ではサポートへ上限引き上げを相談
""",
    "ru": r"""## Ограничения частоты (rate limits)

NovaPuraAI применяет rate limits для защиты платформы и справедливого качества сервиса. Лимиты могут действовать на ключ, пользователя, модель или глобально — в зависимости от конфигурации.

## Симптомы

При превышении API обычно возвращает **HTTP 429** и сообщение о слишком большом числе запросов.

## Рекомендуемое поведение клиента

### Экспоненциальный backoff

```python
import time
import random
import requests

def post_with_retry(url, headers, json, max_attempts=5):
    delay = 0.5
    for attempt in range(max_attempts):
        r = requests.post(url, headers=headers, json=json, timeout=60)
        if r.status_code != 429:
            return r
        time.sleep(delay + random.random() * 0.2)
        delay = min(delay * 2, 16)
    return r
```

### Практики

- Избегайте плотных циклов без пауз
- Переиспользуйте HTTP-соединения (keep-alive)
- Ограничивайте параллелизм
- Используйте batching, когда это уместно
- Отслеживайте 429 в метриках

## Rate limit vs квота

| Понятие | Смысл |
|---------|--------|
| **Rate limit** | Слишком много запросов за единицу времени |
| **Quota / баланс** | Недостаточно кредита для оплаты запроса |

Оба случая могут выглядеть как «слишком много» или отказ в доступе — читайте тело ошибки.

## Streaming и длинные запросы

Длинные запросы (streaming, media) дольше занимают ресурсы. Даже при низком QPS большое число одновременных соединений может упереться в лимиты. Проектируйте concurrency, а не только средний throughput.

## Что делать в production

1. Retry + jitter только для retriable ошибок
2. Ставить пики трафика в очередь
3. Распределять нагрузку по ключам только если это разрешено политикой аккаунта
4. Обратиться в support, если нужны более высокие лимиты для легитимной нагрузки
""",
}

# ========================= api-chat =========================
DOCS["api-chat"] = {
    "fr": r"""## Chat Completions

Endpoint principal compatible OpenAI pour le dialogue et la génération de texte.

```http
POST https://www.novapuraai.com/v1/chat/completions
Authorization: Bearer sk-xxxxx
Content-Type: application/json
```

## Corps de requête (principaux champs)

| Champ | Type | Description |
|-------|------|-------------|
| `model` | string | ID du modèle |
| `messages` | array | Historique des messages (`system` / `user` / `assistant` / `tool`) |
| `temperature` | number | Aléatoire de l’échantillonnage |
| `top_p` | number | Nucleus sampling |
| `max_tokens` | integer | Limite de tokens de sortie (selon modèle) |
| `stream` | boolean | Active le SSE streaming |
| `stop` | string / array | Séquences d’arrêt |
| `tools` / `tool_choice` | object | Appels d’outils (si supportés) |
| `response_format` | object | Ex. JSON mode (si supporté) |

## Exemple minimal

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Résume les avantages d’une passerelle API."}
    ]
  }'
```

## Streaming

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")

stream = client.chat.completions.create(
    model="gpt-4o-mini",
    stream=True,
    messages=[{"role": "user", "content": "Écris un haïku sur les API."}],
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

## Multimodal (vision)

Si le modèle le supporte, `content` peut être un tableau de parties texte/image :

```json
{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "Que vois-tu ?"},
        {
          "type": "image_url",
          "image_url": {"url": "https://example.com/image.png"}
        }
      ]
    }
  ]
}
```

## Réponse

La structure suit le schéma OpenAI `chat.completion` (ou chunks `chat.completion.chunk` en streaming). Vérifiez toujours `choices` et `usage`.

## Erreurs fréquentes

- Modèle non autorisé pour la clé
- `messages` vide ou rôles invalides
- `max_tokens` hors plage
- Timeout côté client trop court pour les longues générations
""",
    "ja": r"""## Chat Completions

対話とテキスト生成向けの OpenAI 互換メインエンドポイントです。

```http
POST https://www.novapuraai.com/v1/chat/completions
Authorization: Bearer sk-xxxxx
Content-Type: application/json
```

## リクエスト本文（主要フィールド）

| フィールド | 型 | 説明 |
|------------|-----|------|
| `model` | string | モデル ID |
| `messages` | array | メッセージ履歴（`system` / `user` / `assistant` / `tool`） |
| `temperature` | number | サンプリングのランダム性 |
| `top_p` | number | Nucleus sampling |
| `max_tokens` | integer | 出力トークン上限（モデル依存） |
| `stream` | boolean | SSE ストリーミングを有効化 |
| `stop` | string / array | 停止シーケンス |
| `tools` / `tool_choice` | object | ツール呼び出し（対応モデル） |
| `response_format` | object | 例: JSON mode（対応時） |

## 最小例

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "API ゲートウェイの利点を要約して。"}
    ]
  }'
```

## ストリーミング

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")

stream = client.chat.completions.create(
    model="gpt-4o-mini",
    stream=True,
    messages=[{"role": "user", "content": "API についての俳句を書いて。"}],
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

## マルチモーダル（vision）

対応モデルでは `content` を text/image の配列にできます:

```json
{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "何が見えますか？"},
        {
          "type": "image_url",
          "image_url": {"url": "https://example.com/image.png"}
        }
      ]
    }
  ]
}
```

## レスポンス

構造は OpenAI の `chat.completion`（streaming 時は `chat.completion.chunk`）に従います。常に `choices` と `usage` を確認してください。

## よくあるエラー

- キーに許可されていないモデル
- 空の `messages` や無効な role
- 範囲外の `max_tokens`
- 長い生成に対するクライアント timeout が短すぎる
""",
    "ru": r"""## Chat Completions

Основной OpenAI-совместимый эндпоинт для диалога и генерации текста.

```http
POST https://www.novapuraai.com/v1/chat/completions
Authorization: Bearer sk-xxxxx
Content-Type: application/json
```

## Тело запроса (основные поля)

| Поле | Тип | Описание |
|------|-----|----------|
| `model` | string | ID модели |
| `messages` | array | История сообщений (`system` / `user` / `assistant` / `tool`) |
| `temperature` | number | Случайность sampling |
| `top_p` | number | Nucleus sampling |
| `max_tokens` | integer | Лимит выходных токенов (зависит от модели) |
| `stream` | boolean | Включает SSE streaming |
| `stop` | string / array | Стоп-последовательности |
| `tools` / `tool_choice` | object | Tool calls (если поддерживается) |
| `response_format` | object | Например JSON mode (если поддерживается) |

## Минимальный пример

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Кратко перечисли преимущества API-шлюза."}
    ]
  }'
```

## Streaming

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")

stream = client.chat.completions.create(
    model="gpt-4o-mini",
    stream=True,
    messages=[{"role": "user", "content": "Напиши хайку про API."}],
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

## Multimodal (vision)

Если модель поддерживает, `content` может быть массивом text/image частей:

```json
{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "Что ты видишь?"},
        {
          "type": "image_url",
          "image_url": {"url": "https://example.com/image.png"}
        }
      ]
    }
  ]
}
```

## Ответ

Структура следует схеме OpenAI `chat.completion` (или chunks `chat.completion.chunk` при streaming). Всегда проверяйте `choices` и `usage`.

## Частые ошибки

- Модель не разрешена для ключа
- Пустой `messages` или неверные roles
- `max_tokens` вне допустимого диапазона
- Слишком короткий client timeout для длинной генерации
""",
}

# Continue in part 2 - remaining sections will be appended via second write or same file
# For manageability this file continues below.

DOCS["api-messages"] = {
    "fr": r"""## Messages (Claude)

Endpoint compatible Anthropic Messages pour les modèles Claude.

```http
POST https://www.novapuraai.com/v1/messages
```

## Authentification

Deux styles courants :

```http
Authorization: Bearer sk-xxxxx
```

ou

```http
x-api-key: sk-xxxxx
anthropic-version: 2023-06-01
```

## Exemple

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 512,
    "messages": [
      {"role": "user", "content": "Explique la différence entre REST et gRPC."}
    ]
  }'
```

## Champs importants

| Champ | Description |
|-------|-------------|
| `model` | ID Claude disponible sur la plateforme |
| `max_tokens` | Obligatoire dans le style Anthropic |
| `messages` | Liste de tours user/assistant |
| `system` | Prompt système (string ou blocs) |
| `stream` | Streaming SSE style Anthropic |
| `temperature` / `top_p` | Contrôle d’échantillonnage |

## Streaming

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -N \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "stream": true,
    "messages": [{"role": "user", "content": "Dis bonjour"}]
  }'
```

## Quand utiliser Messages vs Chat Completions

- **Messages** : clients et SDK Anthropic, payloads Claude natifs
- **Chat Completions** : écosystème OpenAI, la plupart des intégrations

Les deux passent par NovaPuraAI ; choisissez le format de votre outil.
""",
    "ja": r"""## Messages（Claude）

Claude モデル向けの Anthropic Messages 互換エンドポイントです。

```http
POST https://www.novapuraai.com/v1/messages
```

## 認証

一般的な 2 つのスタイル:

```http
Authorization: Bearer sk-xxxxx
```

または

```http
x-api-key: sk-xxxxx
anthropic-version: 2023-06-01
```

## 例

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 512,
    "messages": [
      {"role": "user", "content": "REST と gRPC の違いを説明して。"}
    ]
  }'
```

## 重要なフィールド

| フィールド | 説明 |
|------------|------|
| `model` | プラットフォームで利用可能な Claude ID |
| `max_tokens` | Anthropic 形式では必須 |
| `messages` | user/assistant の会話ターン |
| `system` | システムプロンプト（文字列またはブロック） |
| `stream` | Anthropic 形式の SSE ストリーミング |
| `temperature` / `top_p` | サンプリング制御 |

## ストリーミング

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -N \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "stream": true,
    "messages": [{"role": "user", "content": "こんにちはと言って"}]
  }'
```

## Messages と Chat Completions の使い分け

- **Messages**: Anthropic クライアント / SDK、Claude ネイティブ payload
- **Chat Completions**: OpenAI エコシステム、多くの連携

どちらも NovaPuraAI 経由です。ツールが期待する形式を選んでください。
""",
    "ru": r"""## Messages (Claude)

Anthropic Messages-совместимый эндпоинт для моделей Claude.

```http
POST https://www.novapuraai.com/v1/messages
```

## Аутентификация

Два распространённых стиля:

```http
Authorization: Bearer sk-xxxxx
```

или

```http
x-api-key: sk-xxxxx
anthropic-version: 2023-06-01
```

## Пример

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 512,
    "messages": [
      {"role": "user", "content": "Объясни разницу между REST и gRPC."}
    ]
  }'
```

## Важные поля

| Поле | Описание |
|------|----------|
| `model` | ID Claude, доступный на платформе |
| `max_tokens` | Обязателен в стиле Anthropic |
| `messages` | Ходы user/assistant |
| `system` | System prompt (string или блоки) |
| `stream` | SSE streaming в стиле Anthropic |
| `temperature` / `top_p` | Управление sampling |

## Streaming

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -N \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "stream": true,
    "messages": [{"role": "user", "content": "Скажи привет"}]
  }'
```

## Когда Messages, а когда Chat Completions

- **Messages**: Anthropic клиенты/SDK, нативные Claude payload
- **Chat Completions**: экосистема OpenAI, большинство интеграций

Оба пути идут через NovaPuraAI; выбирайте формат вашего инструмента.
""",
}

DOCS["api-gemini"] = {
    "fr": r"""## Gemini

NovaPuraAI expose les routes style Google Gemini sous `/v1beta`.

## Endpoints typiques

```http
POST /v1beta/models/{model}:generateContent
POST /v1beta/models/{model}:streamGenerateContent
GET  /v1beta/models
```

Exemple complet :

```text
https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent
```

## Authentification

```http
Authorization: Bearer sk-xxxxx
```

Certaines clients Google utilisent aussi `?key=sk-xxxxx`.

## Exemple generateContent

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [{"text": "Donne 3 idées de noms de produit."}]
      }
    ]
  }'
```

## Streaming

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse" \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "contents": [
      {"role": "user", "parts": [{"text": "Compte de 1 à 5"}]}
    ]
  }'
```

## Alternative OpenAI-compatible

Si votre stack est déjà en OpenAI SDK, vous pouvez souvent appeler un modèle Gemini via `/v1/chat/completions` **lorsque ce modèle est exposé** sur la passerelle en format OpenAI. Vérifiez la liste des modèles.

## Conseils

- Respectez le format `contents` / `parts` Gemini
- Vérifiez les noms exacts des modèles dans la console
- Pour les SDK Google officiels, pointez l’API endpoint / base URL vers NovaPuraAI selon la doc du client
""",
    "ja": r"""## Gemini

NovaPuraAI は Google Gemini 形式のルートを `/v1beta` 配下で提供します。

## 代表的なエンドポイント

```http
POST /v1beta/models/{model}:generateContent
POST /v1beta/models/{model}:streamGenerateContent
GET  /v1beta/models
```

完全な例:

```text
https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent
```

## 認証

```http
Authorization: Bearer sk-xxxxx
```

一部の Google クライアントは `?key=sk-xxxxx` も使います。

## generateContent の例

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [{"text": "商品名のアイデアを 3 つ出して。"}]
      }
    ]
  }'
```

## ストリーミング

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse" \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "contents": [
      {"role": "user", "parts": [{"text": "1 から 5 まで数えて"}]}
    ]
  }'
```

## OpenAI 互換の代替

すでに OpenAI SDK を使っている場合、ゲートウェイ上で OpenAI 形式として公開されている Gemini モデルを `/v1/chat/completions` 経由で呼べることがあります。モデル一覧で確認してください。

## ヒント

- Gemini の `contents` / `parts` 形式に従う
- コンソールで正確なモデル名を確認する
- 公式 Google SDK では、クライアントの設定に従い NovaPuraAI を endpoint / base URL に指定する
""",
    "ru": r"""## Gemini

NovaPuraAI предоставляет маршруты в стиле Google Gemini под `/v1beta`.

## Типичные эндпоинты

```http
POST /v1beta/models/{model}:generateContent
POST /v1beta/models/{model}:streamGenerateContent
GET  /v1beta/models
```

Полный пример:

```text
https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent
```

## Аутентификация

```http
Authorization: Bearer sk-xxxxx
```

Некоторые Google-клиенты также используют `?key=sk-xxxxx`.

## Пример generateContent

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [{"text": "Дай 3 идеи названий продукта."}]
      }
    ]
  }'
```

## Streaming

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse" \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "contents": [
      {"role": "user", "parts": [{"text": "Посчитай от 1 до 5"}]}
    ]
  }'
```

## OpenAI-compatible альтернатива

Если ваш стек уже на OpenAI SDK, модель Gemini иногда можно вызывать через `/v1/chat/completions`, **если она опубликована** на шлюзе в OpenAI-формате. Проверьте список моделей.

## Советы

- Соблюдайте формат Gemini `contents` / `parts`
- Проверяйте точные имена моделей в консоли
- Для официальных Google SDK указывайте NovaPuraAI как API endpoint / base URL согласно документации клиента
""",
}

DOCS["api-embeddings"] = {
    "fr": r"""## Embeddings

Générez des vecteurs pour la recherche sémantique, le clustering et le RAG.

```http
POST https://www.novapuraai.com/v1/embeddings
Authorization: Bearer sk-xxxxx
Content-Type: application/json
```

## Exemple

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "NovaPuraAI unifie l’accès aux modèles."
  }'
```

## Entrée multiple

```json
{
  "model": "text-embedding-3-small",
  "input": [
    "premier document",
    "deuxième document"
  ]
}
```

## Python

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")
resp = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello embeddings",
)
vector = resp.data[0].embedding
print(len(vector))
```

## Notes

- La dimension dépend du modèle
- Facturation généralement basée sur les tokens d’entrée
- Vérifiez les modèles d’embedding disponibles via `/v1/models` ou la console
""",
    "ja": r"""## Embeddings

セマンティック検索、クラスタリング、RAG 用のベクトルを生成します。

```http
POST https://www.novapuraai.com/v1/embeddings
Authorization: Bearer sk-xxxxx
Content-Type: application/json
```

## 例

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "NovaPuraAI はモデルアクセスを統一します。"
  }'
```

## 複数入力

```json
{
  "model": "text-embedding-3-small",
  "input": [
    "最初の文書",
    "2 番目の文書"
  ]
}
```

## Python

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")
resp = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello embeddings",
)
vector = resp.data[0].embedding
print(len(vector))
```

## 注意

- 次元数はモデル依存
- 課金は通常入力トークン基準
- 利用可能な embedding モデルは `/v1/models` またはコンソールで確認
""",
    "ru": r"""## Embeddings

Генерируйте векторы для семантического поиска, кластеризации и RAG.

```http
POST https://www.novapuraai.com/v1/embeddings
Authorization: Bearer sk-xxxxx
Content-Type: application/json
```

## Пример

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "NovaPuraAI унифицирует доступ к моделям."
  }'
```

## Несколько входов

```json
{
  "model": "text-embedding-3-small",
  "input": [
    "первый документ",
    "второй документ"
  ]
}
```

## Python

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")
resp = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello embeddings",
)
vector = resp.data[0].embedding
print(len(vector))
```

## Примечания

- Размерность зависит от модели
- Биллинг обычно по входным токенам
- Доступные embedding-модели смотрите в `/v1/models` или в консоли
""",
}

DOCS["api-media"] = {
    "fr": r"""## Images, audio et rerank

NovaPuraAI relaie plusieurs endpoints média et utilitaires au format OpenAI (et rerank dédié).

## Images

```http
POST https://www.novapuraai.com/v1/images/generations
```

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "Un renard minimaliste en style flat design",
    "n": 1,
    "size": "1024x1024"
  }'
```

Édition d’image (si le modèle le supporte) :

```http
POST /v1/images/edits
POST /v1/edits
```

## Audio

### Text-to-speech

```http
POST https://www.novapuraai.com/v1/audio/speech
```

```bash
curl https://www.novapuraai.com/v1/audio/speech \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini-tts",
    "voice": "alloy",
    "input": "Bonjour depuis NovaPuraAI"
  }' \
  --output speech.mp3
```

### Transcription

```http
POST https://www.novapuraai.com/v1/audio/transcriptions
```

Envoyez un `multipart/form-data` avec le fichier audio, le `model`, etc.

### Translation

```http
POST https://www.novapuraai.com/v1/audio/translations
```

## Rerank

```http
POST https://www.novapuraai.com/v1/rerank
```

```bash
curl https://www.novapuraai.com/v1/rerank \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "rerank-multilingual-v3.0",
    "query": "passerelle API IA",
    "documents": [
      "NovaPuraAI est une passerelle compatible OpenAI.",
      "La météo demain sera ensoleillée.",
      "Les embeddings servent à la recherche sémantique."
    ]
  }'
```

## Facturation média

Les unités de facturation peuvent différer du simple comptage de tokens (nombre d’images, durée audio, etc.). Consultez la page de tarification de la console avant la production.
""",
    "ja": r"""## 画像・音声・Rerank

NovaPuraAI は OpenAI 形式のメディア系エンドポイントと専用 Rerank を中継します。

## 画像

```http
POST https://www.novapuraai.com/v1/images/generations
```

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "フラットデザインのミニマルなキツネ",
    "n": 1,
    "size": "1024x1024"
  }'
```

画像編集（対応モデル時）:

```http
POST /v1/images/edits
POST /v1/edits
```

## 音声

### Text-to-speech

```http
POST https://www.novapuraai.com/v1/audio/speech
```

```bash
curl https://www.novapuraai.com/v1/audio/speech \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini-tts",
    "voice": "alloy",
    "input": "NovaPuraAI からこんにちは"
  }' \
  --output speech.mp3
```

### 文字起こし

```http
POST https://www.novapuraai.com/v1/audio/transcriptions
```

音声ファイル、`model` などを `multipart/form-data` で送信します。

### 翻訳

```http
POST https://www.novapuraai.com/v1/audio/translations
```

## Rerank

```http
POST https://www.novapuraai.com/v1/rerank
```

```bash
curl https://www.novapuraai.com/v1/rerank \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "rerank-multilingual-v3.0",
    "query": "AI API ゲートウェイ",
    "documents": [
      "NovaPuraAI は OpenAI 互換ゲートウェイです。",
      "明日の天気は晴れです。",
      "Embeddings はセマンティック検索に使われます。"
    ]
  }'
```

## メディア課金

課金単位は単純なトークン数と異なる場合があります（画像枚数、音声時間など）。本番前にコンソールの料金ページを確認してください。
""",
    "ru": r"""## Images, audio и rerank

NovaPuraAI ретранслирует media-эндпоинты в формате OpenAI и отдельный rerank.

## Images

```http
POST https://www.novapuraai.com/v1/images/generations
```

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "Минималистичная лиса в flat design",
    "n": 1,
    "size": "1024x1024"
  }'
```

Редактирование изображений (если модель поддерживает):

```http
POST /v1/images/edits
POST /v1/edits
```

## Audio

### Text-to-speech

```http
POST https://www.novapuraai.com/v1/audio/speech
```

```bash
curl https://www.novapuraai.com/v1/audio/speech \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini-tts",
    "voice": "alloy",
    "input": "Привет от NovaPuraAI"
  }' \
  --output speech.mp3
```

### Transcription

```http
POST https://www.novapuraai.com/v1/audio/transcriptions
```

Отправляйте `multipart/form-data` с аудиофайлом, `model` и т.д.

### Translation

```http
POST https://www.novapuraai.com/v1/audio/translations
```

## Rerank

```http
POST https://www.novapuraai.com/v1/rerank
```

```bash
curl https://www.novapuraai.com/v1/rerank \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "rerank-multilingual-v3.0",
    "query": "AI API gateway",
    "documents": [
      "NovaPuraAI — OpenAI-совместимый шлюз.",
      "Завтра будет солнечно.",
      "Embeddings используются для семантического поиска."
    ]
  }'
```

## Биллинг media

Единицы тарификации могут отличаться от простого подсчёта токенов (число изображений, длительность audio и т.д.). Перед production смотрите pricing в консоли.
""",
}

DOCS["api-models"] = {
    "fr": r"""## Liste des modèles

```http
GET https://www.novapuraai.com/v1/models
Authorization: Bearer sk-xxxxx
```

## Exemple

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

Réponse (schéma OpenAI) :

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o-mini",
      "object": "model",
      "created": 0,
      "owned_by": "custom"
    }
  ]
}
```

## Détail d’un modèle

```http
GET /v1/models/{model}
```

## Variantes de format

Selon les en-têtes, la liste peut être adaptée :

- Style OpenAI (par défaut)
- Style Anthropic si `x-api-key` + `anthropic-version`
- Style Gemini via `/v1beta/models` ou paramètres Google

## Utilisation pratique

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")
for m in client.models.list().data:
    print(m.id)
```

Utilisez uniquement des IDs présents dans cette liste (ou explicitement documentés dans la console) pour éviter les erreurs 404 / modèle indisponible.
""",
    "ja": r"""## モデル一覧

```http
GET https://www.novapuraai.com/v1/models
Authorization: Bearer sk-xxxxx
```

## 例

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

レスポンス（OpenAI スキーマ）:

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o-mini",
      "object": "model",
      "created": 0,
      "owned_by": "custom"
    }
  ]
}
```

## モデル詳細

```http
GET /v1/models/{model}
```

## 形式のバリエーション

ヘッダーにより一覧の形式が変わることがあります:

- OpenAI 形式（デフォルト）
- `x-api-key` + `anthropic-version` の場合は Anthropic 形式
- Gemini 形式は `/v1beta/models` または Google パラメータ

## 実務での使い方

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")
for m in client.models.list().data:
    print(m.id)
```

404 / モデル不可用を避けるため、この一覧（またはコンソール記載）にある ID のみを使用してください。
""",
    "ru": r"""## Список моделей

```http
GET https://www.novapuraai.com/v1/models
Authorization: Bearer sk-xxxxx
```

## Пример

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-xxxxx"
```

Ответ (схема OpenAI):

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o-mini",
      "object": "model",
      "created": 0,
      "owned_by": "custom"
    }
  ]
}
```

## Детали модели

```http
GET /v1/models/{model}
```

## Варианты формата

В зависимости от заголовков список может адаптироваться:

- Стиль OpenAI (по умолчанию)
- Стиль Anthropic при `x-api-key` + `anthropic-version`
- Стиль Gemini через `/v1beta/models` или Google-параметры

## Практическое использование

```python
from openai import OpenAI

client = OpenAI(api_key="sk-xxxxx", base_url="https://www.novapuraai.com/v1")
for m in client.models.list().data:
    print(m.id)
```

Используйте только ID из этого списка (или явно указанные в консоли), чтобы избежать 404 / unavailable model.
""",
}

DOCS["api-errors"] = {
    "fr": r"""## Erreurs API

NovaPuraAI renvoie des codes HTTP standard et un corps d’erreur JSON, souvent au style OpenAI.

## Exemple de corps

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

## Codes HTTP courants

| HTTP | Signification typique |
|------|------------------------|
| `400` | Requête invalide (JSON, paramètres, modèle) |
| `401` | Non authentifié |
| `403` | Interdit / clé ou modèle non autorisé |
| `404` | Ressource ou modèle introuvable |
| `429` | Rate limit |
| `500` | Erreur serveur / relais |
| `502` / `503` | Upstream indisponible ou surcharge |
| `504` | Timeout amont |

## Stratégie de retry

Retriable en général :

- `429`
- `502`, `503`, `504`
- certaines `500` transitoires

Non retriable sans correction :

- `400`, `401`, `403`, `404`

Ajoutez un backoff avec jitter et un plafond de tentatives.

## Messages utiles à journaliser

Côté client, journalisez :

- `request_id` / en-têtes de corrélation s’ils sont présents
- code HTTP
- `error.message` et `error.code`
- modèle et endpoint (sans exposer la clé)

## Validation préventive

- Validez le JSON avant envoi
- Vérifiez le modèle via `/v1/models`
- Bornez `max_tokens`, `n`, durées et tailles de fichiers
- Gérez le solde insuffisant avant les jobs batch
""",
    "ja": r"""## API エラー

NovaPuraAI は標準的な HTTP ステータスと、多くの場合 OpenAI 形式の JSON エラー本文を返します。

## 本文の例

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

## よくある HTTP コード

| HTTP | 典型的な意味 |
|------|----------------|
| `400` | 不正なリクエスト（JSON、パラメータ、モデル） |
| `401` | 未認証 |
| `403` | 禁止 / キーまたはモデル未許可 |
| `404` | リソースまたはモデル未検出 |
| `429` | レート制限 |
| `500` | サーバー / リレーエラー |
| `502` / `503` | 上流不可または過負荷 |
| `504` | 上流タイムアウト |

## リトライ方針

一般に再試行可能:

- `429`
- `502`、`503`、`504`
- 一部の一時的な `500`

修正なしでは再試行不可:

- `400`、`401`、`403`、`404`

jitter 付きバックオフと試行回数上限を設けてください。

## ログに残すべき情報

クライアントでは次を記録:

- ある場合は `request_id` / 相関ヘッダー
- HTTP コード
- `error.message` と `error.code`
- モデルとエンドポイント（キーは出さない）

## 予防的バリデーション

- 送信前に JSON を検証
- `/v1/models` でモデルを確認
- `max_tokens`、`n`、時間、ファイルサイズに上限
- バッチ前に残高不足を処理
""",
    "ru": r"""## Ошибки API

NovaPuraAI возвращает стандартные HTTP-коды и JSON-тело ошибки, часто в стиле OpenAI.

## Пример тела

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

## Частые HTTP-коды

| HTTP | Типичный смысл |
|------|----------------|
| `400` | Неверный запрос (JSON, параметры, модель) |
| `401` | Не аутентифицирован |
| `403` | Запрещено / ключ или модель не разрешены |
| `404` | Ресурс или модель не найдены |
| `429` | Rate limit |
| `500` | Ошибка сервера / релей |
| `502` / `503` | Upstream недоступен или перегружен |
| `504` | Timeout upstream |

## Стратегия retry

Обычно retriable:

- `429`
- `502`, `503`, `504`
- некоторые транзиентные `500`

Не retriable без исправления:

- `400`, `401`, `403`, `404`

Добавляйте backoff с jitter и лимит попыток.

## Что логировать

На клиенте:

- `request_id` / correlation headers, если есть
- HTTP-код
- `error.message` и `error.code`
- модель и endpoint (без ключа)

## Превентивная валидация

- Проверяйте JSON до отправки
- Проверяйте модель через `/v1/models`
- Ограничивайте `max_tokens`, `n`, длительности и размеры файлов
- Обрабатывайте insufficient balance до batch-задач
""",
}

DOCS["sdk-python"] = {
    "fr": r"""## SDK Python

Utilisez le SDK officiel `openai` en pointant `base_url` vers NovaPuraAI.

## Installation

```bash
pip install openai
```

## Configuration

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## Chat

```python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "system", "content": "Tu es concis."},
        {"role": "user", "content": "Qu’est-ce que NovaPuraAI ?"},
    ],
)
print(completion.choices[0].message.content)
```

## Streaming

```python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    stream=True,
    messages=[{"role": "user", "content": "Écris un paragraphe court."}],
)
for event in stream:
    piece = event.choices[0].delta.content or ""
    print(piece, end="")
```

## Embeddings

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input=["doc A", "doc B"],
)
print(len(emb.data[0].embedding))
```

## Async

```python
from openai import AsyncOpenAI

client = AsyncOpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## Astuces

- Définissez des timeouts adaptés aux longues générations
- Capturez `openai.APIStatusError` pour les codes HTTP
- Ne loggez jamais la clé API
""",
    "ja": r"""## Python SDK

公式 `openai` SDK の `base_url` を NovaPuraAI に向けます。

## インストール

```bash
pip install openai
```

## 設定

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## Chat

```python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "system", "content": "簡潔に答えてください。"},
        {"role": "user", "content": "NovaPuraAI とは何ですか？"},
    ],
)
print(completion.choices[0].message.content)
```

## ストリーミング

```python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    stream=True,
    messages=[{"role": "user", "content": "短い段落を書いて。"}],
)
for event in stream:
    piece = event.choices[0].delta.content or ""
    print(piece, end="")
```

## Embeddings

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input=["doc A", "doc B"],
)
print(len(emb.data[0].embedding))
```

## Async

```python
from openai import AsyncOpenAI

client = AsyncOpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## ヒント

- 長い生成向けに timeout を適切に設定
- HTTP コードは `openai.APIStatusError` で捕捉
- API キーをログに出さない
""",
    "ru": r"""## Python SDK

Используйте официальный SDK `openai`, направив `base_url` на NovaPuraAI.

## Установка

```bash
pip install openai
```

## Настройка

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## Chat

```python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "system", "content": "Отвечай кратко."},
        {"role": "user", "content": "Что такое NovaPuraAI?"},
    ],
)
print(completion.choices[0].message.content)
```

## Streaming

```python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    stream=True,
    messages=[{"role": "user", "content": "Напиши короткий абзац."}],
)
for event in stream:
    piece = event.choices[0].delta.content or ""
    print(piece, end="")
```

## Embeddings

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input=["doc A", "doc B"],
)
print(len(emb.data[0].embedding))
```

## Async

```python
from openai import AsyncOpenAI

client = AsyncOpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## Советы

- Задавайте timeout под длинную генерацию
- Ловите `openai.APIStatusError` для HTTP-кодов
- Никогда не логируйте API-ключ
""",
}

DOCS["sdk-node"] = {
    "fr": r"""## SDK Node.js

## Installation

```bash
npm install openai
# ou
pnpm add openai
# ou
bun add openai
```

## Configuration

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

## Chat

```javascript
const completion = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [
    { role: "user", content: "Donne-moi une checklist de mise en production API." },
  ],
});

console.log(completion.choices[0].message.content);
```

## Streaming

```javascript
const stream = await client.chat.completions.create({
  model: "gpt-4o-mini",
  stream: true,
  messages: [{ role: "user", content: "Liste 5 bons noms de variables." }],
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || "");
}
```

## TypeScript

Le package `openai` fournit des types. Gardez `baseURL` et `apiKey` hors du code source (env / secret manager).

## Erreurs

```javascript
try {
  await client.chat.completions.create({
    model: "gpt-4o-mini",
    messages: [{ role: "user", content: "ping" }],
  });
} catch (err) {
  console.error(err);
}
```
""",
    "ja": r"""## Node.js SDK

## インストール

```bash
npm install openai
# または
pnpm add openai
# または
bun add openai
```

## 設定

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

## Chat

```javascript
const completion = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [
    { role: "user", content: "API 本番公開のチェックリストをください。" },
  ],
});

console.log(completion.choices[0].message.content);
```

## ストリーミング

```javascript
const stream = await client.chat.completions.create({
  model: "gpt-4o-mini",
  stream: true,
  messages: [{ role: "user", content: "良い変数名を 5 個挙げて。" }],
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || "");
}
```

## TypeScript

`openai` パッケージは型を提供します。`baseURL` と `apiKey` はソースに直書きせず、環境変数 / シークレット管理を使ってください。

## エラー

```javascript
try {
  await client.chat.completions.create({
    model: "gpt-4o-mini",
    messages: [{ role: "user", content: "ping" }],
  });
} catch (err) {
  console.error(err);
}
```
""",
    "ru": r"""## Node.js SDK

## Установка

```bash
npm install openai
# или
pnpm add openai
# или
bun add openai
```

## Настройка

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

## Chat

```javascript
const completion = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [
    { role: "user", content: "Дай чеклист вывода API в production." },
  ],
});

console.log(completion.choices[0].message.content);
```

## Streaming

```javascript
const stream = await client.chat.completions.create({
  model: "gpt-4o-mini",
  stream: true,
  messages: [{ role: "user", content: "Перечисли 5 хороших имён переменных." }],
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || "");
}
```

## TypeScript

Пакет `openai` предоставляет типы. Храните `baseURL` и `apiKey` вне исходников (env / secret manager).

## Ошибки

```javascript
try {
  await client.chat.completions.create({
    model: "gpt-4o-mini",
    messages: [{ role: "user", content: "ping" }],
  });
} catch (err) {
  console.error(err);
}
```
""",
}

DOCS["sdk-go"] = {
    "fr": r"""## SDK Go

Vous pouvez utiliser un client OpenAI pour Go en définissant la base URL NovaPuraAI.

## Exemple avec `openai-go`

```bash
go get github.com/openai/openai-go
```

```go
package main

import (
        "context"
        "fmt"
        "os"

        "github.com/openai/openai-go"
        "github.com/openai/openai-go/option"
)

func main() {
        client := openai.NewClient(
                option.WithAPIKey(os.Getenv("NOVAPURA_API_KEY")),
                option.WithBaseURL("https://www.novapuraai.com/v1"),
        )

        chat, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
                Model: openai.F("gpt-4o-mini"),
                Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
                        openai.UserMessage("Bonjour depuis Go"),
                }),
        })
        if err != nil {
                panic(err)
        }
        fmt.Println(chat.Choices[0].Message.Content)
}
```

> Note : les signatures exactes du SDK peuvent évoluer ; adaptez aux versions installées.

## HTTP brut

```go
req, _ := http.NewRequest("POST", "https://www.novapuraai.com/v1/chat/completions", body)
req.Header.Set("Authorization", "Bearer "+apiKey)
req.Header.Set("Content-Type", "application/json")
```

## Conseils

- Propager un `context.Context` avec timeout
- Réutiliser `http.Client` avec pool de connexions
- Traiter explicitement les statuts `429` et `5xx`
""",
    "ja": r"""## Go SDK

Go 向け OpenAI クライアントで NovaPuraAI の Base URL を設定します。

## `openai-go` の例

```bash
go get github.com/openai/openai-go
```

```go
package main

import (
        "context"
        "fmt"
        "os"

        "github.com/openai/openai-go"
        "github.com/openai/openai-go/option"
)

func main() {
        client := openai.NewClient(
                option.WithAPIKey(os.Getenv("NOVAPURA_API_KEY")),
                option.WithBaseURL("https://www.novapuraai.com/v1"),
        )

        chat, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
                Model: openai.F("gpt-4o-mini"),
                Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
                        openai.UserMessage("Go からこんにちは"),
                }),
        })
        if err != nil {
                panic(err)
        }
        fmt.Println(chat.Choices[0].Message.Content)
}
```

> 注: SDK の正確なシグネチャはバージョンで変わることがあります。インストール版に合わせて調整してください。

## 素の HTTP

```go
req, _ := http.NewRequest("POST", "https://www.novapuraai.com/v1/chat/completions", body)
req.Header.Set("Authorization", "Bearer "+apiKey)
req.Header.Set("Content-Type", "application/json")
```

## ヒント

- timeout 付き `context.Context` を渡す
- 接続プール付き `http.Client` を再利用
- `429` と `5xx` を明示的に処理
""",
    "ru": r"""## Go SDK

Используйте OpenAI-клиент для Go, указав Base URL NovaPuraAI.

## Пример с `openai-go`

```bash
go get github.com/openai/openai-go
```

```go
package main

import (
        "context"
        "fmt"
        "os"

        "github.com/openai/openai-go"
        "github.com/openai/openai-go/option"
)

func main() {
        client := openai.NewClient(
                option.WithAPIKey(os.Getenv("NOVAPURA_API_KEY")),
                option.WithBaseURL("https://www.novapuraai.com/v1"),
        )

        chat, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
                Model: openai.F("gpt-4o-mini"),
                Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
                        openai.UserMessage("Привет из Go"),
                }),
        })
        if err != nil {
                panic(err)
        }
        fmt.Println(chat.Choices[0].Message.Content)
}
```

> Примечание: точные сигнатуры SDK могут меняться — адаптируйте под установленную версию.

## Сырой HTTP

```go
req, _ := http.NewRequest("POST", "https://www.novapuraai.com/v1/chat/completions", body)
req.Header.Set("Authorization", "Bearer "+apiKey)
req.Header.Set("Content-Type", "application/json")
```

## Советы

- Прокидывайте `context.Context` с timeout
- Переиспользуйте `http.Client` с connection pool
- Явно обрабатывайте статусы `429` и `5xx`
""",
}

DOCS["sdk-curl"] = {
    "fr": r"""## Exemples curl

Les exemples suivants utilisent la base officielle `https://www.novapuraai.com`.

## Chat Completions

```bash
export NOVAPURA_API_KEY="sk-xxxxx"

curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Ping"}]
  }'
```

## Streaming

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Compte jusqu’à 3"}]
  }'
```

## Models

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## Embeddings

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "texte à vectoriser"
  }'
```

## Messages (Claude)

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 128,
    "messages": [{"role": "user", "content": "Salut"}]
  }'
```

## Rerank

```bash
curl https://www.novapuraai.com/v1/rerank \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "rerank-multilingual-v3.0",
    "query": "facturation API",
    "documents": ["doc 1", "doc 2"]
  }'
```

## Débogage

Ajoutez `-i` pour afficher les en-têtes de réponse, ou `-v` pour le détail HTTP.
""",
    "ja": r"""## curl の例

以下は公式ベース `https://www.novapuraai.com` を使います。

## Chat Completions

```bash
export NOVAPURA_API_KEY="sk-xxxxx"

curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Ping"}]
  }'
```

## ストリーミング

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "3 まで数えて"}]
  }'
```

## Models

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## Embeddings

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "ベクトル化するテキスト"
  }'
```

## Messages（Claude）

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 128,
    "messages": [{"role": "user", "content": "こんにちは"}]
  }'
```

## Rerank

```bash
curl https://www.novapuraai.com/v1/rerank \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "rerank-multilingual-v3.0",
    "query": "API 課金",
    "documents": ["doc 1", "doc 2"]
  }'
```

## デバッグ

応答ヘッダーを見るには `-i`、HTTP 詳細には `-v` を付けます。
""",
    "ru": r"""## Примеры curl

Примеры используют официальный base `https://www.novapuraai.com`.

## Chat Completions

```bash
export NOVAPURA_API_KEY="sk-xxxxx"

curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Ping"}]
  }'
```

## Streaming

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Посчитай до 3"}]
  }'
```

## Models

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## Embeddings

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "текст для векторизации"
  }'
```

## Messages (Claude)

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 128,
    "messages": [{"role": "user", "content": "Привет"}]
  }'
```

## Rerank

```bash
curl https://www.novapuraai.com/v1/rerank \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "rerank-multilingual-v3.0",
    "query": "биллинг API",
    "documents": ["doc 1", "doc 2"]
  }'
```

## Отладка

Добавьте `-i` для заголовков ответа или `-v` для подробного HTTP-лога.
""",
}

DOCS["integration-cursor"] = {
    "fr": r"""## Intégration Cursor

Cursor peut utiliser un fournisseur OpenAI-compatible. Pointez-le vers NovaPuraAI.

## Étapes générales

1. Ouvrez les paramètres de Cursor
2. Section modèles / OpenAI-compatible / API override (libellés selon la version)
3. Définissez :
   - **API Key** : `sk-xxxxx`
   - **Base URL** : `https://www.novapuraai.com/v1`
4. Choisissez un modèle disponible sur NovaPuraAI
5. Enregistrez et testez avec une invite simple

## Valeurs à renseigner

| Champ | Valeur |
|-------|--------|
| API Key | clé console `sk-xxxxx` |
| Base URL | `https://www.novapuraai.com/v1` |
| Model | ex. `gpt-4o-mini` (selon catalogue) |

## Dépannage

- Si Cursor ajoute déjà `/v1`, essayez `https://www.novapuraai.com` sans suffixe
- Vérifiez que le modèle est listé par `GET /v1/models`
- Contrôlez le solde et les permissions de la clé
- Désactivez les proxies locaux qui réécrivent HTTPS si nécessaire

## Sécurité

N’utilisez pas une clé de production partagée dans un environnement non fiable. Préférez une clé Cursor dédiée avec plafond de quota.
""",
    "ja": r"""## Cursor 連携

Cursor は OpenAI 互換プロバイダーを利用できます。NovaPuraAI を指定します。

## 一般的な手順

1. Cursor の設定を開く
2. Models / OpenAI-compatible / API override などの項目（表記はバージョン依存）
3. 次を設定:
   - **API Key**: `sk-xxxxx`
   - **Base URL**: `https://www.novapuraai.com/v1`
4. NovaPuraAI で利用可能なモデルを選択
5. 保存し、簡単なプロンプトでテスト

## 入力値

| 項目 | 値 |
|------|-----|
| API Key | コンソールの `sk-xxxxx` |
| Base URL | `https://www.novapuraai.com/v1` |
| Model | 例: `gpt-4o-mini`（カタログによる） |

## トラブルシュート

- Cursor が既に `/v1` を付ける場合は、接尾辞なしの `https://www.novapuraai.com` を試す
- モデルが `GET /v1/models` に存在するか確認
- 残高とキー権限を確認
- 必要なら HTTPS を書き換えるローカルプロキシを無効化

## セキュリティ

信頼できない環境で共有の本番キーを使わないでください。クォータ上限付きの Cursor 専用キーを推奨します。
""",
    "ru": r"""## Интеграция Cursor

Cursor может использовать OpenAI-compatible провайдера. Направьте его на NovaPuraAI.

## Общие шаги

1. Откройте настройки Cursor
2. Раздел models / OpenAI-compatible / API override (названия зависят от версии)
3. Укажите:
   - **API Key**: `sk-xxxxx`
   - **Base URL**: `https://www.novapuraai.com/v1`
4. Выберите модель, доступную на NovaPuraAI
5. Сохраните и проверьте простым промптом

## Какие значения вводить

| Поле | Значение |
|------|----------|
| API Key | ключ из консоли `sk-xxxxx` |
| Base URL | `https://www.novapuraai.com/v1` |
| Model | напр. `gpt-4o-mini` (по каталогу) |

## Troubleshooting

- Если Cursor уже добавляет `/v1`, попробуйте `https://www.novapuraai.com` без суффикса
- Проверьте, что модель есть в `GET /v1/models`
- Проверьте баланс и права ключа
- При необходимости отключите локальные proxy, переписывающие HTTPS

## Безопасность

Не используйте общий production-ключ в недоверенной среде. Лучше отдельный ключ для Cursor с потолком квоты.
""",
}

DOCS["integration-nextchat"] = {
    "fr": r"""## Intégration NextChat

NextChat (ChatGPT-Next-Web et forks) supporte les backends OpenAI-compatible.

## Configuration typique

Variables d’environnement (selon le déploiement) :

```bash
OPENAI_API_KEY=sk-xxxxx
BASE_URL=https://www.novapuraai.com
# parfois :
OPENAI_URL=https://www.novapuraai.com/v1
CUSTOM_MODELS=gpt-4o-mini
```

Dans l’UI (si disponible) :

1. Ouvrez les paramètres
2. Endpoint / API base : `https://www.novapuraai.com` ou `.../v1`
3. Clé API : `sk-xxxxx`
4. Modèle par défaut : un ID valide NovaPuraAI

## Vérification

- Envoyez « ping » dans le chat
- Si 401 : clé incorrecte
- Si 404 modèle : mettez à jour la liste de modèles
- Si CORS en self-host : configurez le reverse proxy correctement

## Notes

- Certaines builds imposent un suffixe `/v1` ; testez les deux formes de base URL
- Pour Claude via NextChat, le support dépend de la version de l’UI ; le chemin natif reste `/v1/messages`
""",
    "ja": r"""## NextChat 連携

NextChat（ChatGPT-Next-Web および派生）は OpenAI 互換バックエンドに対応しています。

## 典型的な設定

デプロイに応じた環境変数:

```bash
OPENAI_API_KEY=sk-xxxxx
BASE_URL=https://www.novapuraai.com
# 場合により:
OPENAI_URL=https://www.novapuraai.com/v1
CUSTOM_MODELS=gpt-4o-mini
```

UI（ある場合）:

1. 設定を開く
2. Endpoint / API base: `https://www.novapuraai.com` または `.../v1`
3. API キー: `sk-xxxxx`
4. 既定モデル: 有効な NovaPuraAI ID

## 確認

- チャットで「ping」を送る
- 401: キー不正
- 404 モデル: モデル一覧を更新
- self-host の CORS: リバースプロキシを正しく設定

## 注意

- 一部ビルドは `/v1` 接尾辞を前提にします。両方の Base URL を試してください
- NextChat 経由の Claude は UI バージョン依存です。ネイティブ経路は `/v1/messages`
""",
    "ru": r"""## Интеграция NextChat

NextChat (ChatGPT-Next-Web и форки) поддерживает OpenAI-compatible backend.

## Типичная конфигурация

Переменные окружения (зависит от деплоя):

```bash
OPENAI_API_KEY=sk-xxxxx
BASE_URL=https://www.novapuraai.com
# иногда:
OPENAI_URL=https://www.novapuraai.com/v1
CUSTOM_MODELS=gpt-4o-mini
```

В UI (если доступно):

1. Откройте настройки
2. Endpoint / API base: `https://www.novapuraai.com` или `.../v1`
3. API key: `sk-xxxxx`
4. Модель по умолчанию: валидный ID NovaPuraAI

## Проверка

- Отправьте «ping» в чат
- 401: неверный ключ
- 404 model: обновите список моделей
- CORS при self-host: корректно настройте reverse proxy

## Примечания

- Некоторые сборки ожидают суффикс `/v1` — проверьте обе формы Base URL
- Claude через NextChat зависит от версии UI; нативный путь остаётся `/v1/messages`
""",
}

DOCS["integration-openwebui"] = {
    "fr": r"""## Intégration OpenWebUI

Open WebUI peut se connecter à des API OpenAI-compatible externes.

## Configuration

1. Ouvrez **Admin Panel** → **Settings** → **Connections** (libellés selon version)
2. Ajoutez une connexion OpenAI
3. Renseignez :
   - **API Base URL** : `https://www.novapuraai.com/v1`
   - **API Key** : `sk-xxxxx`
4. Synchronisez / rafraîchissez la liste des modèles
5. Sélectionnez un modèle dans le sélecteur de chat

## Docker (aperçu)

```yaml
environment:
  - OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
  - OPENAI_API_KEY=sk-xxxxx
```

Les noms exacts de variables peuvent varier selon la version d’Open WebUI.

## Vérifications

- La liste des modèles se charge sans erreur
- Un message simple renvoie une réponse
- Les logs Open WebUI ne montrent pas d’URL mal formée (`/v1/v1/...`)

## Si double `/v1`

Si les requêtes partent vers `/v1/v1/chat/completions`, retirez le suffixe `/v1` de la base configurée (ou l’inverse selon le client).
""",
    "ja": r"""## OpenWebUI 連携

Open WebUI は外部の OpenAI 互換 API に接続できます。

## 設定

1. **Admin Panel** → **Settings** → **Connections** を開く（表記はバージョン依存）
2. OpenAI 接続を追加
3. 入力:
   - **API Base URL**: `https://www.novapuraai.com/v1`
   - **API Key**: `sk-xxxxx`
4. モデル一覧を同期 / 更新
5. チャットのモデル選択で対象を選ぶ

## Docker（概要）

```yaml
environment:
  - OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
  - OPENAI_API_KEY=sk-xxxxx
```

変数名は Open WebUI のバージョンで異なる場合があります。

## 確認項目

- モデル一覧がエラーなく読み込まれる
- 簡単なメッセージに応答が返る
- Open WebUI ログに不正 URL（`/v1/v1/...`）が出ない

## 二重の `/v1` が出る場合

リクエストが `/v1/v1/chat/completions` になる場合は、設定した base から `/v1` を外す（または逆）を試してください。
""",
    "ru": r"""## Интеграция OpenWebUI

Open WebUI может подключаться к внешним OpenAI-compatible API.

## Настройка

1. Откройте **Admin Panel** → **Settings** → **Connections** (названия зависят от версии)
2. Добавьте OpenAI-подключение
3. Укажите:
   - **API Base URL**: `https://www.novapuraai.com/v1`
   - **API Key**: `sk-xxxxx`
4. Синхронизируйте / обновите список моделей
5. Выберите модель в чате

## Docker (обзор)

```yaml
environment:
  - OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
  - OPENAI_API_KEY=sk-xxxxx
```

Точные имена переменных могут отличаться в разных версиях Open WebUI.

## Проверки

- Список моделей загружается без ошибок
- Простое сообщение получает ответ
- В логах Open WebUI нет кривого URL (`/v1/v1/...`)

## Если появляется двойной `/v1`

Если запросы идут на `/v1/v1/chat/completions`, уберите суффикс `/v1` из base (или наоборот — в зависимости от клиента).
""",
}

DOCS["integration-dify"] = {
    "fr": r"""## Intégration Dify

Dify permet d’ajouter des fournisseurs de modèles compatibles OpenAI.

## Étapes

1. Ouvrez les paramètres du workspace Dify
2. Allez dans **Model Providers**
3. Ajoutez / configurez un fournisseur **OpenAI-API-compatible**
4. Renseignez :
   - **API Key** : `sk-xxxxx`
   - **API Endpoint URL** : `https://www.novapuraai.com/v1`
5. Ajoutez les modèles (chat, embedding, etc.) avec leurs IDs exacts
6. Testez la connexion depuis Dify

## Exemple de modèles

- Chat : `gpt-4o-mini`
- Embedding : `text-embedding-3-small`

Utilisez les IDs réellement disponibles sur votre instance NovaPuraAI.

## Workflows et agents

Une fois le provider enregistré, les apps Dify (chatbot, workflow, agent) peuvent sélectionner ces modèles dans les nœuds LLM.

## Dépannage

- Erreur de connexion : vérifiez endpoint et clé
- Modèle introuvable : l’ID doit correspondre à `/v1/models`
- Timeout : augmentez les timeouts Dify pour les longues tâches
""",
    "ja": r"""## Dify 連携

Dify では OpenAI 互換のモデルプロバイダーを追加できます。

## 手順

1. Dify ワークスペース設定を開く
2. **Model Providers** へ移動
3. **OpenAI-API-compatible** プロバイダーを追加 / 設定
4. 入力:
   - **API Key**: `sk-xxxxx`
   - **API Endpoint URL**: `https://www.novapuraai.com/v1`
5. chat / embedding などのモデルを正確な ID で追加
6. Dify から接続テスト

## モデル例

- Chat: `gpt-4o-mini`
- Embedding: `text-embedding-3-small`

ご利用の NovaPuraAI で実際に有効な ID を使ってください。

## Workflow / Agent

プロバイダー登録後、Dify のアプリ（chatbot、workflow、agent）の LLM ノードでこれらのモデルを選択できます。

## トラブルシュート

- 接続エラー: endpoint とキーを確認
- モデル未検出: ID が `/v1/models` と一致しているか
- Timeout: 長時間ジョブ向けに Dify の timeout を延長
""",
    "ru": r"""## Интеграция Dify

Dify позволяет добавлять OpenAI-compatible провайдеры моделей.

## Шаги

1. Откройте настройки workspace Dify
2. Перейдите в **Model Providers**
3. Добавьте / настройте провайдер **OpenAI-API-compatible**
4. Укажите:
   - **API Key**: `sk-xxxxx`
   - **API Endpoint URL**: `https://www.novapuraai.com/v1`
5. Добавьте модели (chat, embedding и т.д.) с точными ID
6. Проверьте соединение из Dify

## Примеры моделей

- Chat: `gpt-4o-mini`
- Embedding: `text-embedding-3-small`

Используйте ID, реально доступные на вашем NovaPuraAI.

## Workflows и agents

После регистрации provider приложения Dify (chatbot, workflow, agent) могут выбирать эти модели в LLM-нодах.

## Troubleshooting

- Ошибка соединения: проверьте endpoint и ключ
- Модель не найдена: ID должен совпадать с `/v1/models`
- Timeout: увеличьте timeouts Dify для длинных задач
""",
}

DOCS["integration-langchain"] = {
    "fr": r"""## LangChain / LlamaIndex

Les écosystèmes LangChain et LlamaIndex fonctionnent via des clients OpenAI en changeant base URL et clé.

## LangChain (Python)

```bash
pip install langchain-openai
```

```python
import os
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
    temperature=0.2,
)

print(llm.invoke("Dis bonjour en une phrase.").content)
```

### Embeddings LangChain

```python
from langchain_openai import OpenAIEmbeddings

emb = OpenAIEmbeddings(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## LangChain.js

```typescript
import { ChatOpenAI } from "@langchain/openai";

const llm = new ChatOpenAI({
  model: "gpt-4o-mini",
  apiKey: process.env.NOVAPURA_API_KEY,
  configuration: {
    baseURL: "https://www.novapuraai.com/v1",
  },
});
```

## LlamaIndex

```python
from llama_index.llms.openai import OpenAI

llm = OpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base="https://www.novapuraai.com/v1",
)
```

## Conseils RAG

- Alignez le modèle d’embedding d’indexation et de requête
- Surveillez le coût des étapes retrieve + generate
- Mettez en cache les embeddings stables
""",
    "ja": r"""## LangChain / LlamaIndex

LangChain と LlamaIndex は、Base URL とキーを差し替えた OpenAI クライアント経由で動作します。

## LangChain（Python）

```bash
pip install langchain-openai
```

```python
import os
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
    temperature=0.2,
)

print(llm.invoke("一文で挨拶して。").content)
```

### LangChain Embeddings

```python
from langchain_openai import OpenAIEmbeddings

emb = OpenAIEmbeddings(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## LangChain.js

```typescript
import { ChatOpenAI } from "@langchain/openai";

const llm = new ChatOpenAI({
  model: "gpt-4o-mini",
  apiKey: process.env.NOVAPURA_API_KEY,
  configuration: {
    baseURL: "https://www.novapuraai.com/v1",
  },
});
```

## LlamaIndex

```python
from llama_index.llms.openai import OpenAI

llm = OpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base="https://www.novapuraai.com/v1",
)
```

## RAG のヒント

- 索引時とクエリ時の embedding モデルを揃える
- retrieve + generate のコストを監視する
- 安定した embeddings をキャッシュする
""",
    "ru": r"""## LangChain / LlamaIndex

LangChain и LlamaIndex работают через OpenAI-клиенты со сменой base URL и ключа.

## LangChain (Python)

```bash
pip install langchain-openai
```

```python
import os
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
    temperature=0.2,
)

print(llm.invoke("Поздоровайся одним предложением.").content)
```

### Embeddings LangChain

```python
from langchain_openai import OpenAIEmbeddings

emb = OpenAIEmbeddings(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## LangChain.js

```typescript
import { ChatOpenAI } from "@langchain/openai";

const llm = new ChatOpenAI({
  model: "gpt-4o-mini",
  apiKey: process.env.NOVAPURA_API_KEY,
  configuration: {
    baseURL: "https://www.novapuraai.com/v1",
  },
});
```

## LlamaIndex

```python
from llama_index.llms.openai import OpenAI

llm = OpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base="https://www.novapuraai.com/v1",
)
```

## Советы по RAG

- Согласуйте embedding-модель индексации и запроса
- Следите за стоимостью retrieve + generate
- Кэшируйте стабильные embeddings
""",
}

DOCS["faq"] = {
    "fr": r"""## FAQ

### Quelle est la base URL officielle ?

`https://www.novapuraai.com` — pour la plupart des SDK OpenAI : `https://www.novapuraai.com/v1`.

### Comment s’authentifier ?

Envoyez `Authorization: Bearer sk-xxxxx` avec une clé créée dans la console.

### NovaPuraAI est-il compatible OpenAI ?

Oui. Les chemins principaux (`/v1/chat/completions`, `/v1/embeddings`, `/v1/models`, …) suivent le style OpenAI. Des formats Claude et Gemini sont aussi disponibles.

### Où trouver la liste des modèles ?

- Console (tarification / modèles)
- `GET /v1/models`

### Pourquoi 401 ?

Clé manquante, mal copiée, révoquée, ou en-tête incorrect.

### Pourquoi 429 ?

Rate limit. Réduisez le débit et appliquez un backoff.

### Pourquoi le modèle est introuvable ?

L’ID n’existe pas pour votre compte, n’est pas activé, ou la base URL est incorrecte (double `/v1`, mauvais hôte).

### Streaming supporté ?

Oui pour de nombreux modèles via `"stream": true` (Chat) ou les endpoints de stream Gemini / Claude selon le format.

### Comment contrôler les coûts ?

- Choisir le bon modèle
- Limiter `max_tokens`
- Séparer clés dev/prod
- Surveiller les logs d’usage dans la console

### Puis-je utiliser Cursor / Dify / LangChain ?

Oui. Configurez-les en OpenAI-compatible avec la base URL et la clé NovaPuraAI. Voir les pages d’intégration.

### Où obtenir de l’aide ?

Consultez la console, les logs de requête, et le support / canal d’assistance indiqué sur le site.
""",
    "ja": r"""## FAQ

### 公式 Base URL は？

`https://www.novapuraai.com` — 多くの OpenAI SDK では `https://www.novapuraai.com/v1`。

### 認証方法は？

コンソールで作成したキーを `Authorization: Bearer sk-xxxxx` で送ってください。

### NovaPuraAI は OpenAI 互換ですか？

はい。主要パス（`/v1/chat/completions`、`/v1/embeddings`、`/v1/models` など）は OpenAI 形式です。Claude / Gemini 形式も利用できます。

### モデル一覧はどこ？

- コンソール（料金 / モデル）
- `GET /v1/models`

### 401 になる理由は？

キー欠落、コピーミス、失効、またはヘッダー不正。

### 429 になる理由は？

レート制限です。頻度を下げ、バックオフを入れてください。

### モデルが見つからない理由は？

アカウントに存在しない / 未有効、または Base URL 不正（二重 `/v1`、ホスト誤り）。

### ストリーミングは使えますか？

多くのモデルで `"stream": true`（Chat）、または Gemini / Claude のストリームエンドポイントに対応します。

### コスト制御の方法は？

- 適切なモデルを選ぶ
- `max_tokens` を制限
- 開発 / 本番キーを分離
- コンソールの使用ログを監視

### Cursor / Dify / LangChain は使えますか？

はい。OpenAI 互換として Base URL とキーを設定します。各連携ページを参照してください。

### サポートは？

コンソール、リクエストログ、サイト記載のサポート窓口を利用してください。
""",
    "ru": r"""## FAQ

### Какой официальный Base URL?

`https://www.novapuraai.com` — для большинства OpenAI SDK: `https://www.novapuraai.com/v1`.

### Как аутентифицироваться?

Отправляйте `Authorization: Bearer sk-xxxxx` с ключом из консоли.

### NovaPuraAI совместим с OpenAI?

Да. Основные пути (`/v1/chat/completions`, `/v1/embeddings`, `/v1/models`, …) следуют стилю OpenAI. Также доступны форматы Claude и Gemini.

### Где список моделей?

- Консоль (pricing / models)
- `GET /v1/models`

### Почему 401?

Ключ отсутствует, скопирован неверно, отозван, или неверный заголовок.

### Почему 429?

Rate limit. Снизьте частоту и добавьте backoff.

### Почему модель не найдена?

ID недоступен для аккаунта, не включён, или неверный Base URL (двойной `/v1`, неверный host).

### Поддерживается streaming?

Да для многих моделей через `"stream": true` (Chat) или stream-эндпоинты Gemini / Claude в соответствующем формате.

### Как контролировать расходы?

- Выбирать подходящую модель
- Ограничивать `max_tokens`
- Разделять dev/prod ключи
- Смотреть usage-логи в консоли

### Можно ли Cursor / Dify / LangChain?

Да. Настройте их как OpenAI-compatible с Base URL и ключом NovaPuraAI. См. страницы интеграций.

### Куда обращаться за помощью?

Используйте консоль, логи запросов и support / канал поддержки, указанный на сайте.
""",
}

EXPECTED = [
    "quickstart",
    "authentication",
    "first-request",
    "base-url",
    "routing",
    "billing",
    "rate-limits",
    "api-chat",
    "api-messages",
    "api-gemini",
    "api-embeddings",
    "api-media",
    "api-models",
    "api-errors",
    "sdk-python",
    "sdk-node",
    "sdk-go",
    "sdk-curl",
    "integration-cursor",
    "integration-nextchat",
    "integration-openwebui",
    "integration-dify",
    "integration-langchain",
    "faq",
]

LANGS = ("fr", "ja", "ru")


def main() -> None:
    missing = [s for s in EXPECTED if s not in DOCS]
    if missing:
        raise SystemExit(f"Missing sections in DOCS: {missing}")

    counts = {lang: 0 for lang in LANGS}
    failures: list[str] = []

    for section in EXPECTED:
        for lang in LANGS:
            body = DOCS[section].get(lang)
            if not body or not body.strip():
                failures.append(f"{section}/{lang}: empty")
                continue
            text = body.strip() + "\n"
            if text.lstrip().startswith("# ") and not text.lstrip().startswith("## "):
                # allow only if first heading is ## — soft check
                pass
            if text.strip() == "# WIP" or text.strip().startswith("# WIP"):
                failures.append(f"{section}/{lang}: still WIP")
                continue
            # Forbid top-level single-# title as first non-empty line
            first = next((ln for ln in text.splitlines() if ln.strip()), "")
            if first.startswith("# ") and not first.startswith("## "):
                failures.append(f"{section}/{lang}: has top-level # title")
                continue

            path = ROOT / section / f"{lang}.md"
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8", newline="\n")
            counts[lang] += 1

    print("Written counts:", counts)
    print("Total files:", sum(counts.values()))
    if failures:
        print("FAILURES:")
        for f in failures:
            print(" -", f)
        raise SystemExit(1)
    print("OK")


if __name__ == "__main__":
    main()
