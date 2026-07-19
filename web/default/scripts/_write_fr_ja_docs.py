# -*- coding: utf-8 -*-
"""Write full fr.md / ja.md for all 24 official docs sections."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "src" / "i18n" / "docs"

DOCS: dict[str, dict[str, str]] = {}

# ---------------------------------------------------------------------------
# api-chat
# ---------------------------------------------------------------------------
DOCS["api-chat"] = {
    "fr": r"""`POST /v1/chat/completions` est le principal endpoint de chat compatible OpenAI.

## Requête

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are helpful."},
      {"role": "user", "content": "Summarize NovaPuraAI in one sentence."}
    ],
    "temperature": 0.5,
    "max_tokens": 256
  }'
```

## Champs importants

| Champ | Notes |
| --- | --- |
| `model` | Obligatoire. Doit être activé pour votre compte |
| `messages` | Tableau de messages de chat au format OpenAI |
| `stream` | `true` pour le streaming de tokens SSE |
| `temperature` / `top_p` | Contrôles d’échantillonnage |
| `max_tokens` / `max_completion_tokens` | Limites de sortie (selon le fournisseur) |
| `tools` / `tool_choice` | Appel de fonctions lorsque le modèle amont le prend en charge |

## Streaming

Définissez `"stream": true`. La réponse est `text/event-stream` avec des fragments `data: {...}` se terminant par `data: [DONE]`.

## Compatibilité

La plupart des outils qui acceptent une base URL OpenAI personnalisée fonctionnent sans modification. Pointez-les vers `https://www.novapuraai.com/v1` et utilisez votre clé NovaPuraAI.
""",
    "ja": r"""`POST /v1/chat/completions` は、OpenAI 互換の主要なチャットエンドポイントです。

## リクエスト

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are helpful."},
      {"role": "user", "content": "Summarize NovaPuraAI in one sentence."}
    ],
    "temperature": 0.5,
    "max_tokens": 256
  }'
```

## 重要なフィールド

| フィールド | 説明 |
| --- | --- |
| `model` | 必須。アカウントで有効化されている必要があります |
| `messages` | OpenAI 形式のチャットメッセージ配列 |
| `stream` | SSE トークンストリーミングには `true` |
| `temperature` / `top_p` | サンプリング制御 |
| `max_tokens` / `max_completion_tokens` | 出力上限（プロバイダー依存） |
| `tools` / `tool_choice` | 上流モデルが対応している場合の関数呼び出し |

## ストリーミング

`"stream": true` を設定します。応答は `text/event-stream` で、`data: {...}` チャンクが続き、最後に `data: [DONE]` で終了します。

## 互換性

カスタムの OpenAI Base URL を受け付けるツールの多くは、そのまま動作します。`https://www.novapuraai.com/v1` を指定し、NovaPuraAI のキーを使用してください。
""",
}

# ---------------------------------------------------------------------------
# api-embeddings
# ---------------------------------------------------------------------------
DOCS["api-embeddings"] = {
    "fr": r"""`POST /v1/embeddings` crée des embeddings vectoriels pour la recherche, le RAG et le clustering.

## Exemple

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## Notes

- `input` peut être une chaîne ou un tableau de chaînes (sous réserve des limites du fournisseur).
- Les dimensions et la normalisation dépendent du modèle d’embedding amont.
- La facturation est généralement proportionnelle aux tokens d’entrée.

## Python

```python
from openai import OpenAI
client = OpenAI(api_key="sk-YOUR_KEY", base_url="https://www.novapuraai.com/v1")
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello world",
)
print(len(emb.data[0].embedding))
```
""",
    "ja": r"""`POST /v1/embeddings` は、検索・RAG・クラスタリング向けのベクトル埋め込みを生成します。

## 例

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## 注意事項

- `input` は文字列、または文字列の配列にできます（プロバイダーの制限に従います）。
- 次元数と正規化は上流の埋め込みモデルに依存します。
- 課金は通常、入力トークン量に比例します。

## Python

```python
from openai import OpenAI
client = OpenAI(api_key="sk-YOUR_KEY", base_url="https://www.novapuraai.com/v1")
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello world",
)
print(len(emb.data[0].embedding))
```
""",
}

# ---------------------------------------------------------------------------
# api-errors
# ---------------------------------------------------------------------------
DOCS["api-errors"] = {
    "fr": r"""Les erreurs sont renvoyées en JSON avec un code de statut HTTP. Le texte du message peut être localisé ou spécifique au fournisseur.

## Codes de statut courants

| Code | Signification |
| --- | --- |
| 400 | Corps de requête ou paramètres invalides |
| 401 | Clé API manquante ou invalide |
| 403 | Non autorisé (modèle, module ou permission) |
| 404 | Route ou modèle inconnu |
| 429 | Limite de débit atteinte |
| 500 / 502 / 503 | Défaillance de la passerelle ou de l’amont |

## Exemple de corps d’erreur

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

Certains endpoints de console utilisent `{ "success": false, "message": "..." }`. Les routes de relais privilégient les objets d’erreur au style OpenAI.

## Liste de contrôle pour le débogage

1. Journalisez l’identifiant de requête s’il apparaît dans la réponse ou les journaux de la console.
2. Réessayez les GET idempotents ; soyez prudent avec les nouvelles tentatives POST.
3. Comparez avec un curl fonctionnel depuis la carte « First API request » du tableau de bord.
4. Vérifiez la santé des canaux avec votre administrateur si seuls certains modèles échouent.
""",
    "ja": r"""エラーは HTTP ステータスコード付きの JSON で返されます。メッセージ文言はローカライズされている場合や、プロバイダー固有の場合があります。

## よくあるステータスコード

| コード | 意味 |
| --- | --- |
| 400 | リクエスト本体またはパラメータが無効 |
| 401 | API キーが欠落または無効 |
| 403 | 許可されていない（モデル、モジュール、または権限） |
| 404 | 不明なルートまたはモデル |
| 429 | レート制限 |
| 500 / 502 / 503 | ゲートウェイまたは上流の障害 |

## エラー本体の例

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

一部のコンソール API は `{ "success": false, "message": "..." }` を使います。中継ルートは OpenAI 形式のエラーオブジェクトを優先します。

## デバッグチェックリスト

1. レスポンスやコンソールログにリクエスト ID が出ていれば記録します。
2. べき等な GET は再試行してよいですが、POST の再試行には注意してください。
3. ダッシュボードの「First API request」カードにある動作する curl と比較します。
4. 一部のモデルだけ失敗する場合は、管理者にチャネルの健全性を確認してもらいます。
""",
}

# ---------------------------------------------------------------------------
# api-gemini
# ---------------------------------------------------------------------------
DOCS["api-gemini"] = {
    "fr": r"""Le trafic compatible Gemini est disponible via des chemins de style Google sous `/v1beta` lorsque les administrateurs activent des canaux Gemini.

## Générer du contenu

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Write a haiku about APIs."}]
    }]
  }'
```

## Conseils

- Les identifiants de modèle exacts dépendent de la configuration des canaux de votre déploiement.
- Les parties multimodales (données inline / URI de fichiers) suivent les formes de requête Gemini ; respectez les limites de taille du corps de la passerelle.
- Vous pouvez aussi accéder à certains modèles Gemini via le chat compatible OpenAI si un adaptateur de canal les mappe.

## Authentification

Utilisez la même clé NovaPuraAI `sk-`. N’envoyez pas de clés Google AI Studio à NovaPuraAI, sauf si vous êtes administrateur et configurez des canaux amont.
""",
    "ja": r"""管理者が Gemini チャネルを有効にしている場合、Google 形式のパス `/v1beta` 経由で Gemini 互換トラフィックを利用できます。

## コンテンツ生成

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Write a haiku about APIs."}]
    }]
  }'
```

## ヒント

- 正確なモデル ID は、デプロイメントのチャネル設定に依存します。
- マルチモーダルの parts（インラインデータ / ファイル URI）は Gemini のリクエスト形状に従います。ペイロードはゲートウェイのボディ上限内に収めてください。
- チャネルアダプターがマッピングしていれば、OpenAI 互換チャット経由で一部の Gemini モデルにもアクセスできます。

## 認証

同じ NovaPuraAI の `sk-` キーを使用します。上流チャネルを設定する管理者でない限り、Google AI Studio のキーを NovaPuraAI に送らないでください。
""",
}

# ---------------------------------------------------------------------------
# api-media
# ---------------------------------------------------------------------------
DOCS["api-media"] = {
    "fr": r"""NovaPuraAI proxifie certains endpoints média lorsque les canaux correspondants sont activés.

## Images

`POST /v1/images/generations`

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A minimal logo for an API platform",
    "size": "1024x1024"
  }'
```

## Transcription audio

`POST /v1/audio/transcriptions` (formulaire multipart)

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## Synthèse vocale

`POST /v1/audio/speech` renvoie des octets audio pour les modèles TTS pris en charge.

## Rerank

`POST /v1/rerank` accepte une requête + des documents pour les rerankers de style Cohere/Jina lorsqu’ils sont configurés.

## Note de facturation

Les endpoints média facturent souvent par nombre d’images, secondes ou nombre de documents — pas seulement par tokens. Consultez Model Square avant les traitements en masse.
""",
    "ja": r"""対応するチャネルが有効な場合、NovaPuraAI は選択されたメディアエンドポイントをプロキシします。

## 画像

`POST /v1/images/generations`

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A minimal logo for an API platform",
    "size": "1024x1024"
  }'
```

## 音声書き起こし

`POST /v1/audio/transcriptions`（multipart フォーム）

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## 音声合成

`POST /v1/audio/speech` は、対応する TTS モデル向けに音声バイトを返します。

## Rerank

設定されている場合、`POST /v1/rerank` は Cohere/Jina 形式の reranker 向けに query と documents を受け付けます。

## 課金に関する注意

メディアエンドポイントは、トークンだけでなく、画像枚数・秒数・ドキュメント数などで課金されることが多いです。一括ジョブの前に Model Square を確認してください。
""",
}

# ---------------------------------------------------------------------------
# api-messages
# ---------------------------------------------------------------------------
DOCS["api-messages"] = {
    "fr": r"""`POST /v1/messages` accepte des charges utiles de style Anthropic Messages pour les modèles compatibles Claude configurés sur la passerelle.

## Exemple

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Explain prepaid API billing briefly."}
    ]
  }'
```

## Notes

- Les noms de modèles doivent exister dans votre catalogue NovaPuraAI.
- Certains en-têtes spécifiques à Anthropic sont acceptés et transmis lorsque c’est pertinent.
- Vous pouvez souvent appeler les mêmes modèles Claude via le format de chat OpenAI si l’adaptateur de canal le prend en charge — préférez le format attendu par votre SDK.

## Erreurs

Un schéma invalide ou des champs non pris en charge renvoient un `4xx` avec un corps d’erreur JSON. Vérifiez que `max_tokens` est présent lorsque l’API Messages l’exige.
""",
    "ja": r"""`POST /v1/messages` は、ゲートウェイ上に設定された Claude 互換モデル向けに、Anthropic Messages 形式のペイロードを受け付けます。

## 例

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Explain prepaid API billing briefly."}
    ]
  }'
```

## 注意事項

- モデル名は NovaPuraAI のカタログに存在する必要があります。
- 一部の Anthropic 専用ヘッダーは、関連する場合に受け付けて転送されます。
- チャネルアダプターが対応していれば、同じ Claude モデルを OpenAI チャット形式でも呼べることが多いです。SDK が期待する形式を優先してください。

## エラー

無効なスキーマや未対応フィールドは、JSON エラー本体付きの `4xx` を返します。Messages API で必須の場合は `max_tokens` があることを確認してください。
""",
}

# ---------------------------------------------------------------------------
# api-models
# ---------------------------------------------------------------------------
DOCS["api-models"] = {
    "fr": r"""`GET /v1/models` liste les modèles disponibles pour la clé authentifiée.

## Exemple

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-YOUR_KEY"
```

## Forme de la réponse

La charge utile suit l’objet liste OpenAI avec des entrées `data[]` contenant au minimum `id` et `object`. Des métadonnées supplémentaires peuvent apparaître selon la version de la passerelle et les paramètres.

## Lorsqu’un modèle est manquant

1. Confirmez que le modèle est activé dans les canaux / capacités d’administration pour votre groupe.
2. Confirmez que votre clé n’est pas restreinte hors de ce modèle.
3. Actualisez Model Square dans l’interface pour les tarifs et la disponibilité.

## Mise en cache

Les clients peuvent mettre la liste en cache pour un TTL court. Rechargez après des changements administratifs ou en cas d’erreurs `404 model_not_found`.
""",
    "ja": r"""`GET /v1/models` は、認証済みキーで利用可能なモデル一覧を返します。

## 例

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-YOUR_KEY"
```

## レスポンスの形

ペイロードは OpenAI の list オブジェクトに従い、`data[]` の各要素には少なくとも `id` と `object` が含まれます。ゲートウェイのバージョンや設定によっては追加のメタデータが付く場合があります。

## モデルが見つからないとき

1. 管理画面のチャネル / 能力設定で、自分のグループ向けにモデルが有効か確認します。
2. キーがそのモデルから制限されていないか確認します。
3. UI の Model Square を更新して、価格と可用性を確認します。

## キャッシュ

クライアントは短い TTL で一覧をキャッシュして構いません。管理側の変更後、または `404 model_not_found` のときは再取得してください。
""",
}

# ---------------------------------------------------------------------------
# authentication
# ---------------------------------------------------------------------------
DOCS["authentication"] = {
    "fr": r"""Chaque requête de relais doit présenter une clé API NovaPuraAI. Les clés sont gérées dans la console et validées par `TokenAuth` sur la passerelle.

## Format d’en-tête

Envoyez la clé comme jeton Bearer :

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

Certains clients OpenAI acceptent aussi `api_key` dans le constructeur du SDK — cette valeur devient le même en-tête Authorization.

## Où créer des clés

1. Connectez-vous → **API Keys**.
2. Créez une clé avec un nom facultatif.
3. Configurez si besoin les listes de modèles autorisés, le quota restant, les limites IP et l’expiration.
4. Enregistrez le secret immédiatement. Le secret complet n’est affiché qu’une seule fois.

## Bonnes pratiques de sécurité

- Préférez les variables d’environnement (`NOVAPURA_API_KEY`) au codage en dur.
- Utilisez des clés distinctes par environnement (dev / staging / production).
- Faites tourner les clés si un client est compromis.
- Limitez les clés au jeu minimal de modèles dont votre application a besoin.
- N’intégrez pas de clés dans des bundles frontend publics.

## Échecs courants

| Symptôme | Cause probable |
| --- | --- |
| `401 Unauthorized` | Clé manquante/invalide, clé révoquée, ou mauvais en-tête |
| `403 Forbidden` | Modèle non autorisé pour cette clé, ou module désactivé |
| `429` | Limite de débit dépassée |
| Quota insuffisant | Solde trop bas ou quota de clé épuisé |

## Configurations multi-utilisateurs

Les administrateurs peuvent délivrer des clés aux utilisateurs finaux avec des quotas indépendants. Chaque clé est facturée sur le solde de son propriétaire selon les paramètres de la plateforme.
""",
    "ja": r"""すべての中継リクエストは NovaPuraAI の API キーを提示する必要があります。キーはコンソールで管理され、ゲートウェイ上の `TokenAuth` によって検証されます。

## ヘッダー形式

キーを Bearer トークンとして送信します:

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

一部の OpenAI クライアントは SDK コンストラクタの `api_key` も受け付けます。その値は同じ Authorization ヘッダーになります。

## キーの作成場所

1. ログイン → **API Keys**。
2. 任意の名前付きでキーを作成します。
3. 必要に応じて、モデル許可リスト、残りのクォータ、IP 制限、有効期限を設定します。
4. シークレットはすぐに保存してください。完全なシークレットは一度だけ表示されます。

## セキュリティのベストプラクティス

- ハードコードせず、環境変数（`NOVAPURA_API_KEY`）を優先します。
- 環境ごと（dev / staging / production）に別キーを使います。
- クライアントが漏洩したらキーをローテーションします。
- アプリに必要な最小限のモデルセットにキーを制限します。
- 公開フロントエンドのバンドルにキーを埋め込まないでください。

## よくある失敗

| 症状 | 想定される原因 |
| --- | --- |
| `401 Unauthorized` | キー欠落/無効、失効、またはヘッダー誤り |
| `403 Forbidden` | このキーでモデルが許可されていない、またはモジュール無効 |
| `429` | レート制限超過 |
| クォータ不足 | 残高不足、またはキーのクォータ枯渇 |

## マルチユーザー構成

管理者はエンドユーザーに独立したクォータ付きキーを発行できます。各キーは、プラットフォーム設定に従い所有者の残高に対して課金されます。
""",
}

# ---------------------------------------------------------------------------
# base-url
# ---------------------------------------------------------------------------
DOCS["base-url"] = {
    "fr": r"""NovaPuraAI fournit une passerelle unifiée. Les clients pointent vers l’origine publique ; la passerelle achemine vers les fournisseurs amont.

## Base URL recommandée

| Type de client | Base URL |
| --- | --- |
| SDK OpenAI / outils compatibles OpenAI | `https://www.novapuraai.com/v1` |
| HTTP brut (le chemin inclut déjà `/v1/...`) | `https://www.novapuraai.com` |

## Endpoints principaux

| Méthode | Chemin | Objectif |
| --- | --- | --- |
| POST | `/v1/chat/completions` | Chat (OpenAI) |
| POST | `/v1/completions` | Complétions de texte |
| POST | `/v1/responses` | OpenAI Responses API |
| POST | `/v1/messages` | Anthropic Messages |
| POST | `/v1/embeddings` | Embeddings |
| POST | `/v1/images/generations` | Génération d’images |
| POST | `/v1/audio/transcriptions` | Parole vers texte |
| POST | `/v1/audio/speech` | Texte vers parole |
| POST | `/v1/rerank` | Rerank |
| GET | `/v1/models` | Liste des modèles |
| POST | `/v1beta/models/{model}:generateContent` | Style Gemini |

Les routes de tâches Midjourney et d’autres peuvent aussi être disponibles selon la configuration administrateur.

## Authentification à chaque appel

Tous les chemins ci-dessus exigent :

```http
Authorization: Bearer sk-YOUR_KEY
```

## Santé de la passerelle

La console d’administration et les endpoints de statut publics indiquent si le site est prêt. En production, maintenez la passerelle et la base de données hautement disponibles ; ne comptez pas sur un stockage local éphémère pour les données critiques de facturation.
""",
    "ja": r"""NovaPuraAI は統合ゲートウェイを提供します。クライアントは公開オリジンを指定し、ゲートウェイが上流プロバイダーへルーティングします。

## 推奨 Base URL

| クライアント種別 | Base URL |
| --- | --- |
| OpenAI SDK / OpenAI 互換ツール | `https://www.novapuraai.com/v1` |
| 生 HTTP（パスに既に `/v1/...` を含む） | `https://www.novapuraai.com` |

## 主要エンドポイント

| メソッド | パス | 用途 |
| --- | --- | --- |
| POST | `/v1/chat/completions` | チャット（OpenAI） |
| POST | `/v1/completions` | テキスト補完 |
| POST | `/v1/responses` | OpenAI Responses API |
| POST | `/v1/messages` | Anthropic Messages |
| POST | `/v1/embeddings` | Embeddings |
| POST | `/v1/images/generations` | 画像生成 |
| POST | `/v1/audio/transcriptions` | 音声からテキスト |
| POST | `/v1/audio/speech` | テキストから音声 |
| POST | `/v1/rerank` | Rerank |
| GET | `/v1/models` | モデル一覧 |
| POST | `/v1beta/models/{model}:generateContent` | Gemini 形式 |

管理者設定によっては、Midjourney などのタスクルートも利用できる場合があります。

## すべての呼び出しでの認証

上記のパスはすべて次を必要とします:

```http
Authorization: Bearer sk-YOUR_KEY
```

## ゲートウェイの健全性

管理コンソールと公開ステータスエンドポイントが、サイトの準備状況を報告します。本番ではゲートウェイとデータベースを高可用性で運用し、課金に関わる重要データに一時的なローカルストレージを依存しないでください。
""",
}

# ---------------------------------------------------------------------------
# billing
# ---------------------------------------------------------------------------
DOCS["billing"] = {
    "fr": r"""L’usage est mesuré par requête. La passerelle estime le coût, préconsomme le quota si nécessaire, puis solde après la réponse amont.

## Concepts

- **Solde / quota** — crédit prépayé sur le compte utilisateur (et éventuellement par clé).
- **Tarification des modèles** — configurée par les administrateurs (ratios de tokens, prix fixes ou règles basées sur des expressions).
- **Préconsommation** — retient le quota estimé pour que les requêtes concurrentes ne puissent pas trop dépenser.
- **Solde (settle)** — ajuste la facture finale à partir de l’usage réel en tokens lorsque disponible.

## Ce que les clients doivent savoir

1. Un appel API réussi côté client peut encore échouer ensuite si l’amont renvoie une erreur après la préconsommation (la retenue non utilisée est remboursée selon la logique de la plateforme).
2. Les réponses en streaming facturent l’usage terminé lorsque le fournisseur signale les tokens.
3. Les produits image, audio et vidéo peuvent facturer par nombre, durée ou multiplicateurs de résolution — consultez toujours Model Square.

## Recharges du portefeuille

Les utilisateurs finaux rechargent depuis la page **Wallet** lorsque des passerelles de paiement sont activées (par exemple Stripe). Les administrateurs contrôlent les devises, promotions et limites.

## Suivi des dépenses

- Les journaux d’usage de la console montrent le modèle, les tokens et le delta de quota par requête.
- Conservez des clés API distinctes par produit pour faciliter l’attribution des dépenses.

## Quota insuffisant

Si une requête est rejetée pour quota, rechargez le portefeuille ou demandez à un administrateur d’augmenter les limites. Créer une nouvelle clé ne crée pas de solde gratuit.
""",
    "ja": r"""利用量はリクエスト単位で計測されます。ゲートウェイはコストを見積もり、必要に応じてクォータを事前消費し、上流応答の後に精算します。

## 概念

- **残高 / クォータ** — ユーザーアカウント上の前払いクレジット（キー単位でも設定可）。
- **モデル価格** — 管理者が設定（トークン比率、固定価格、または式ベースのルール）。
- **事前消費** — 見積もりクォータを確保し、同時リクエストが使いすぎないようにします。
- **精算（settle）** — 実際のトークン使用量が分かる場合に最終課金を調整します。

## クライアントが知っておくべきこと

1. クライアント側で成功に見えても、事前消費の後に上流がエラーを返すと失敗することがあります（未使用の保留分はプラットフォームのロジックに従い返金されます）。
2. ストリーミング応答は、プロバイダーがトークンを報告した完了使用量で課金されます。
3. 画像・音声・動画は枚数・秒数・解像度倍率などで課金されることがあります。必ず Model Square を確認してください。

## ウォレットへのチャージ

決済ゲートウェイが有効な場合（例: Stripe）、エンドユーザーは **Wallet** ページからチャージします。通貨、プロモーション、上限は管理者が制御します。

## 支出の監視

- コンソールの利用ログには、リクエストごとのモデル、トークン、クォータ差分が表示されます。
- 製品ごとに API キーを分けると、支出の帰属が容易になります。

## クォータ不足

クォータ不足で拒否された場合は、ウォレットをチャージするか、管理者に上限引き上げを依頼してください。新しいキーを作成しても無料残高は増えません。
""",
}

# ---------------------------------------------------------------------------
# faq
# ---------------------------------------------------------------------------
DOCS["faq"] = {
    "fr": r"""Réponses aux questions fréquentes sur l’utilisation de la passerelle API NovaPuraAI. Pour l’aide sur l’interface produit au-delà de l’API, utilisez les canaux de support de la console.

## Qu’est-ce que NovaPuraAI ?

NovaPuraAI est une passerelle API compatible OpenAI (produit basé sur new-api). Vous envoyez des requêtes vers une seule base URL avec une clé API ; la passerelle authentifie, achemine par modèle, facture le quota et journalise l’usage.

## Où se trouvent les docs ?

L’interface de documentation développeur officielle est à **`/docs`** sur le déploiement (par exemple `https://www.novapuraai.com/docs`).

## Quelle base URL dois-je utiliser ?

- **Origine** : `https://www.novapuraai.com`
- **SDK OpenAI `base_url`** : `https://www.novapuraai.com/v1`

Voir [Base URL et endpoints](/docs/base-url).

## Comment obtenir une clé API ?

Connectez-vous → **Dashboard → API Keys / Tokens** → créez une clé → copiez le secret `sk-...`. Détails : [Authentification](/docs/authentication).

## Pourquoi ai-je une 401 Unauthorized ?

Causes courantes : en-tête `Authorization` manquant, clé tronquée, jeton désactivé, ou utilisation d’une clé OpenAI Platform au lieu d’une clé NovaPuraAI.

## Pourquoi mon modèle est-il introuvable ?

Les catalogues de modèles sont spécifiques au déploiement et au groupe. Appelez `GET /v1/models` et utilisez un `id` de la réponse. La configuration des canaux administrateur peut aussi nécessiter une mise à jour.

## Prenez-vous en charge les API natives Claude / Gemini ?

Oui :

- Claude Messages : `POST /v1/messages`
- Gemini : `/v1beta/models/{model}:{action}`

OpenAI Chat Completions reste le chemin le plus courant pour les applications multi-fournisseurs.

## Comment la facturation est-elle calculée ?

Selon les règles de tarification des modèles configurées sur la passerelle — en général des tokens pour le chat/embeddings, et des unités spécifiques à la modalité pour les images/audio. Voir [Facturation et quota](/docs/billing) et les journaux d’usage de la console pour les montants faisant autorité.

## Que se passe-t-il si je n’ai plus de quota ?

Les requêtes échouent avec une erreur de type quota insuffisant. Rechargez ou échangez des crédits dans la console, puis réessayez. Les limites par clé peuvent s’épuiser avant le solde du compte.

## Les limites de débit sont-elles la même chose que le quota ?

Non. **429** signifie ralentir ; un quota insuffisant signifie ajouter du solde. Voir [Limites de débit](/docs/rate-limits).

## Puis-je utiliser les SDK OpenAI officiels ?

Oui — définissez la clé API sur votre clé NovaPuraAI et `base_url` / `baseURL` sur `{ORIGIN}/v1`. Exemples : [Python](/docs/sdk-python), [Node.js](/docs/sdk-node), [Go](/docs/sdk-go), [curl](/docs/sdk-curl).

## Le streaming est-il pris en charge ?

Oui pour les modèles/canaux qui le supportent. Utilisez `"stream": true` sur Chat Completions ou les endpoints de streaming natifs pour Claude/Gemini.

## Comment sécuriser les clés en production ?

Gardez les clés sur des serveurs ou des magasins de secrets, faites-les tourner après une fuite, appliquez des listes blanches IP et de modèles lorsque c’est possible, et évitez d’embarquer des secrets dans des clients mobiles/web.

## Où puis-je voir l’historique des requêtes ?

Dans les **journaux d’usage** de la console (et les graphiques du tableau de bord lorsqu’ils sont activés). Incluez horodatages et corps d’erreur lors d’une escalade à un administrateur — n’envoyez jamais les clés brutes.

## Toujours bloqué ?

1. Reproduisez avec curl depuis [Votre première requête](/docs/first-request).
2. Consultez [Erreurs](/docs/api-errors).
3. Confirmez modèles, quota et limites de débit dans la console.
4. Contactez l’administrateur du déploiement avec des détails de requête anonymisés.
""",
    "ja": r"""NovaPuraAI API ゲートウェイの利用に関するよくある質問への回答です。API 以外の製品 UI のヘルプは、コンソールのサポート窓口を利用してください。

## NovaPuraAI とは？

NovaPuraAI は OpenAI 互換の API ゲートウェイ（new-api ベースの製品）です。1 つの Base URL と API キーでリクエストを送ると、ゲートウェイが認証、モデル別ルーティング、クォータ課金、利用ログを行います。

## ドキュメントはどこ？

公式の開発者向けドキュメント UI は、デプロイメント上の **`/docs`** です（例: `https://www.novapuraai.com/docs`）。

## どの Base URL を使うべき？

- **オリジン**: `https://www.novapuraai.com`
- **OpenAI SDK の `base_url`**: `https://www.novapuraai.com/v1`

[Base URL とエンドポイント](/docs/base-url) を参照してください。

## API キーの取得方法は？

ログイン → **Dashboard → API Keys / Tokens** → キーを作成 → `sk-...` シークレットをコピー。詳細: [認証](/docs/authentication)。

## なぜ 401 Unauthorized になる？

よくある原因: `Authorization` ヘッダー欠落、キーの途切れ、無効化されたトークン、OpenAI Platform キーを NovaPuraAI キーの代わりに使っている。

## なぜモデルが見つからない？

モデルカタログはデプロイメントとグループに依存します。`GET /v1/models` を呼び、応答の `id` を使ってください。管理側のチャネル設定の更新が必要な場合もあります。

## Claude / Gemini のネイティブ API は対応している？

はい:

- Claude Messages: `POST /v1/messages`
- Gemini: `/v1beta/models/{model}:{action}`

マルチプロバイダーアプリでは、OpenAI Chat Completions がいちばん一般的な経路です。

## 課金はどう計算される？

ゲートウェイ上で設定されたモデル価格ルールによります。通常、チャット/埋め込みはトークン、画像/音声はモダリティ固有の単位です。正式な金額は [課金とクォータ](/docs/billing) とコンソールの利用ログを確認してください。

## クォータが尽きたらどうなる？

クォータ不足系のエラーでリクエストが失敗します。コンソールでチャージまたはクレジット交換後に再試行してください。キー単位の上限は、アカウント残高より先に枯渇することがあります。

## レート制限とクォータは同じ？

いいえ。**429** は速度を落とせという意味、クォータ不足は残高を追加する必要があるという意味です。[レート制限](/docs/rate-limits) を参照してください。

## 公式 OpenAI SDK は使える？

はい。API キーに NovaPuraAI キーを、`base_url` / `baseURL` に `{ORIGIN}/v1` を設定します。例: [Python](/docs/sdk-python)、[Node.js](/docs/sdk-node)、[Go](/docs/sdk-go)、[curl](/docs/sdk-curl)。

## ストリーミングは対応している？

ストリーミング対応のモデル/チャネルでは可能です。Chat Completions では `"stream": true`、Claude/Gemini ではプロトコル固有のストリーミングエンドポイントを使います。

## 本番でキーをどう守る？

キーはサーバーまたはシークレットストアに置き、漏洩後はローテーションし、利用可能なら IP とモデルの許可リストを適用し、モバイル/Web クライアントに秘密を埋め込まないでください。

## リクエスト履歴はどこで見られる？

コンソールの **利用ログ**（有効ならダッシュボードのチャート）です。管理者へエスカレーションする際はタイムスタンプとエラー本体を含め、生キーは送らないでください。

## まだ解決しない場合

1. [最初のリクエスト](/docs/first-request) の curl で再現します。
2. [エラー](/docs/api-errors) を確認します。
3. コンソールでモデル、クォータ、レート制限を確認します。
4. 機微情報を伏せたリクエスト詳細をデプロイ管理者へ連絡します。
""",
}

# ---------------------------------------------------------------------------
# first-request
# ---------------------------------------------------------------------------
DOCS["first-request"] = {
    "fr": r"""Cette page guide un premier appel réussi de bout en bout et explique comment lire la réponse.

## Liste de contrôle

- [ ] Vous avez une clé commençant par `sk-`
- [ ] Votre compte a un solde / quota positif
- [ ] Vous connaissez au moins un nom de modèle activé (voir **Model Square** ou `GET /v1/models`)

## curl

```bash
export NOVAPURA_API_KEY=sk-YOUR_KEY
export NOVAPURA_BASE=https://www.novapuraai.com

curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are a concise assistant."},
      {"role": "user", "content": "Say hello in one sentence."}
    ],
    "temperature": 0.7
  }'
```

## Réponse réussie (forme)

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "Hello!"},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  }
}
```

## Streaming

Ajoutez `"stream": true` et lisez les Server-Sent Events :

```bash
curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to five."}]
  }'
```

## Dépannage

1. Confirmez que le nom du modèle correspond exactement à un modèle activé.
2. Confirmez que la base URL inclut `/v1` pour les SDK OpenAI.
3. Confirmez HTTPS et que votre reverse proxy / CDN autorise les flux de longue durée si vous streamez.
""",
    "ja": r"""このページでは、最初の成功する呼び出しの流れと、レスポンスの読み方を説明します。

## チェックリスト

- [ ] `sk-` で始まるキーを持っている
- [ ] アカウントに正の残高 / クォータがある
- [ ] 有効なモデル名を少なくとも 1 つ把握している（**Model Square** または `GET /v1/models`）

## curl

```bash
export NOVAPURA_API_KEY=sk-YOUR_KEY
export NOVAPURA_BASE=https://www.novapuraai.com

curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are a concise assistant."},
      {"role": "user", "content": "Say hello in one sentence."}
    ],
    "temperature": 0.7
  }'
```

## 成功レスポンス（形）

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "Hello!"},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  }
}
```

## ストリーミング

`"stream": true` を追加し、Server-Sent Events を読み取ります:

```bash
curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to five."}]
  }'
```

## トラブルシューティング

1. モデル名が有効なモデルと完全に一致しているか確認します。
2. OpenAI SDK 向けの Base URL に `/v1` が含まれているか確認します。
3. HTTPS であること、ストリーム利用時はリバースプロキシ / CDN が長時間接続を許可しているか確認します。
""",
}

# ---------------------------------------------------------------------------
# integration-cursor
# ---------------------------------------------------------------------------
DOCS["integration-cursor"] = {
    "fr": r"""Cursor peut utiliser NovaPuraAI comme fournisseur de modèles compatible OpenAI. Pointez Cursor vers l’API `/v1` de votre origine de déploiement et authentifiez-vous avec une clé API NovaPuraAI.

## Ce dont vous avez besoin

- Une clé API NovaPuraAI depuis **Dashboard → API Keys / Tokens**
- Votre origine de déploiement, par exemple `https://www.novapuraai.com`
- Un quota suffisant pour les modèles que vous sélectionnez dans Cursor

## Configurer l’accès compatible OpenAI

Les libellés de l’interface Cursor évoluent selon les versions. Cherchez **OpenAI API**, **OpenAI Compatible**, **Override OpenAI Base URL** ou **Custom model provider**.

Valeurs typiques :

| Paramètre | Valeur |
| --- | --- |
| API key | `sk-xxxxxxxx` (votre clé NovaPuraAI) |
| Base URL | `https://www.novapuraai.com/v1` |
| Model | Un ID de modèle depuis `GET /v1/models` |

Si un champ demande la « OpenAI base URL » sans `/v1`, essayez les deux formes et confirmez avec une petite invite. La forme qui fonctionne est presque toujours **`{ORIGIN}/v1`**.

## Vérifier d’abord hors de Cursor

```bash
export NOVAPURA_BASE_URL="https://www.novapuraai.com"
export NOVAPURA_API_KEY="sk-xxxxxxxx"

curl "${NOVAPURA_BASE_URL}/v1/models" \
  -H "Authorization: Bearer ${NOVAPURA_API_KEY}"
```

Si cela échoue, corrigez la clé/le quota avant de déboguer l’IDE.

## Conseils de sélection de modèle

- Préférez les modèles qui supportent les tools / le long contexte si vous utilisez les fonctions agent.
- Si Cursor affiche « model not found », l’ID n’est pas activé pour votre groupe — listez les modèles et choisissez un `id` disponible.
- Conservez un modèle moins cher pour l’autocomplétion et un modèle plus fort pour le mode agent lorsque le coût compte.

## Dépannage

| Problème | Correctif |
| --- | --- |
| Erreurs d’auth dans Cursor | Recollez la clé complète y compris `sk-` |
| Échecs de type réseau / CORS | Cursor est un client de bureau ; ce n’est généralement pas du CORS — vérifiez les fautes d’URL et le VPN |
| Réponses vides | Confirmez le quota et essayez le même modèle via curl |
| Limites de débit | Réduisez les exécutions d’agents en parallèle ; voir [Limites de débit](/docs/rate-limits) |

## Sécurité

- Ne partagez pas de fichiers de paramètres d’espace de travail qui embarquent des secrets.
- Faites tourner les clés si une machine est perdue ou qu’une clé a été commitée.

## Voir aussi

- [Authentification](/docs/authentication)
- [Liste des modèles](/docs/api-models)
- [Chat Completions](/docs/api-chat)
""",
    "ja": r"""Cursor は NovaPuraAI を OpenAI 互換のモデルプロバイダーとして利用できます。デプロイメントオリジンの `/v1` API を指定し、NovaPuraAI の API キーで認証します。

## 必要なもの

- **Dashboard → API Keys / Tokens** で取得した NovaPuraAI API キー
- デプロイメントオリジン（例: `https://www.novapuraai.com`）
- Cursor で選ぶモデルに十分なクォータ

## OpenAI 互換アクセスの設定

Cursor の UI ラベルはリリースごとに変わります。**OpenAI API**、**OpenAI Compatible**、**Override OpenAI Base URL**、**Custom model provider** などの設定を探してください。

典型的な値:

| 設定 | 値 |
| --- | --- |
| API key | `sk-xxxxxxxx`（NovaPuraAI キー） |
| Base URL | `https://www.novapuraai.com/v1` |
| Model | `GET /v1/models` のモデル ID |

フィールドが `/v1` なしの「OpenAI base URL」を求める場合は、両方の形式を試し、短いプロンプトで確認してください。動作する形はほぼ常に **`{ORIGIN}/v1`** です。

## まず Cursor の外で確認

```bash
export NOVAPURA_BASE_URL="https://www.novapuraai.com"
export NOVAPURA_API_KEY="sk-xxxxxxxx"

curl "${NOVAPURA_BASE_URL}/v1/models" \
  -H "Authorization: Bearer ${NOVAPURA_API_KEY}"
```

失敗する場合は、IDE のデバッグ前にキー/クォータを直してください。

## モデル選択のヒント

- エージェント機能を使う場合は、tools / 長いコンテキストに対応したモデルを優先します。
- Cursor が「model not found」を出す場合、その ID はグループで有効ではありません。一覧から利用可能な `id` を選びます。
- コストを意識するなら、オートコンプリート用に安価なモデル、エージェント用に強力なモデルを使い分けます。

## トラブルシューティング

| 問題 | 対処 |
| --- | --- |
| Cursor での認証エラー | `sk-` を含む完全なキーを貼り直す |
| ネットワーク / CORS 風の失敗 | Cursor はデスクトップクライアントで通常 CORS ではない — URL 誤記と VPN を確認 |
| 空の応答 | クォータを確認し、同じモデルを curl で試す |
| レート制限 | 並列エージェント実行を減らす。[レート制限](/docs/rate-limits) を参照 |

## セキュリティ

- シークレットを埋め込んだワークスペース設定ファイルを共有しないでください。
- 端末紛失やキーのコミットがあったらキーをローテーションします。

## 関連

- [認証](/docs/authentication)
- [モデル一覧](/docs/api-models)
- [Chat Completions](/docs/api-chat)
""",
}

# ---------------------------------------------------------------------------
# integration-dify
# ---------------------------------------------------------------------------
DOCS["integration-dify"] = {
    "fr": r"""Dify peut appeler NovaPuraAI comme fournisseur de modèles personnalisé compatible OpenAI. Cela permet aux apps, agents et workflows Dify d’utiliser les modèles acheminés via votre passerelle.

## Prérequis

- Un espace de travail Dify (auto-hébergé ou cloud) avec la permission d’ajouter des fournisseurs de modèles
- Clé API NovaPuraAI et quota
- Origine telle que `https://www.novapuraai.com`

## Ajouter un fournisseur compatible OpenAI-API

Dans Dify **Settings → Model Providers** (ou **Model Supplier**) :

1. Choisissez **OpenAI-API-compatible** (le nom peut légèrement varier).
2. Définissez les identifiants :

| Champ | Valeur |
| --- | --- |
| API Key | `sk-xxxxxxxx` |
| API endpoint URL | `https://www.novapuraai.com/v1` |

3. Ajoutez un ou plusieurs modèles avec le **nom exact** renvoyé par NovaPuraAI (par exemple `gpt-4o-mini`).
4. Configurez la longueur de contexte / le mode (chat vs completion) selon le type de modèle.
5. Enregistrez et lancez le test de connexion Dify s’il est disponible.

## Attentes sur les endpoints

Dify demandera typiquement :

- `POST /v1/chat/completions` pour les modèles de chat
- `POST /v1/embeddings` lorsque des modèles d’embedding sont configurés
- `GET /v1/models` seulement si l’intégration du fournisseur fait de la découverte

Confirmez avec un appel curl direct avant de déboguer les graphes Dify.

## Conseils pour agents et workflows

- Créez des entrées de modèle Dify distinctes pour les modèles NovaPuraAI économiques vs premium.
- Définissez des limites max tokens raisonnables dans la configuration des nœuds pour maîtriser le coût.
- Pour les tools / l’appel de fonctions, choisissez des modèles dont les canaux supportent les tools.

## Échecs courants

| Symptôme | Cause probable |
| --- | --- |
| Validation failed | Mauvais endpoint (`/v1` manquant) ou clé |
| Model not found | Écart de nom par rapport à `GET /v1/models` |
| Timeout dans de longues chaînes | Augmentez les timeouts ; réduisez les sauts LLM séquentiels |
| Quota insuffisant en cours d’exécution | Rechargez le solde ; plafonnez les retries du workflow |

## Voir aussi

- [Facturation et quota](/docs/billing)
- [Embeddings](/docs/api-embeddings)
- [Chat Completions](/docs/api-chat)
""",
    "ja": r"""Dify は NovaPuraAI をカスタムの OpenAI 互換モデルプロバイダーとして呼び出せます。これにより、Dify 上のアプリ・エージェント・ワークフローがゲートウェイ経由のモデルを利用できます。

## 前提条件

- モデルプロバイダー追加権限のある Dify ワークスペース（セルフホストまたはクラウド）
- NovaPuraAI の API キーとクォータ
- オリジン（例: `https://www.novapuraai.com`）

## OpenAI-API 互換プロバイダーの追加

Dify の **Settings → Model Providers**（または **Model Supplier**）で:

1. **OpenAI-API-compatible** を選びます（名称は多少異なる場合があります）。
2. 認証情報を設定します:

| フィールド | 値 |
| --- | --- |
| API Key | `sk-xxxxxxxx` |
| API endpoint URL | `https://www.novapuraai.com/v1` |

3. NovaPuraAI が返す **正確なモデル名**（例: `gpt-4o-mini`）で 1 つ以上のモデルを追加します。
4. モデル種別に合わせてコンテキスト長 / モード（chat と completion）を設定します。
5. 保存し、利用可能なら Dify の接続テストを実行します。

## エンドポイントの想定

Dify は通常、次を呼び出します:

- チャットモデル向け `POST /v1/chat/completions`
- 埋め込みモデル設定時の `POST /v1/embeddings`
- プロバイダー連携が発見を行う場合のみ `GET /v1/models`

Dify グラフをデバッグする前に、直接 curl で確認してください。

## エージェントとワークフローのヒント

- 安価なモデルとプレミアムな NovaPuraAI モデルで、Dify のモデル登録を分けます。
- ノード設定で妥当な max token 上限を置き、コストを制御します。
- tools / 関数呼び出しには、チャネルが tools に対応したモデルを選びます。

## よくある失敗

| 症状 | 想定される原因 |
| --- | --- |
| Validation failed | エンドポイント誤り（`/v1` 欠落）またはキー誤り |
| Model not found | `GET /v1/models` との名前不一致 |
| 長いチェーンでの Timeout | タイムアウトを延ばす、連続 LLM ホップを減らす |
| 実行途中のクォータ不足 | 残高をチャージ、ワークフローのリトライを制限 |

## 関連

- [課金とクォータ](/docs/billing)
- [Embeddings](/docs/api-embeddings)
- [Chat Completions](/docs/api-chat)
""",
}

# ---------------------------------------------------------------------------
# integration-langchain
# ---------------------------------------------------------------------------
DOCS["integration-langchain"] = {
    "fr": r"""LangChain et LlamaIndex peuvent appeler NovaPuraAI via leurs intégrations OpenAI en redéfinissant la base URL et la clé API. La passerelle achemine ensuite les noms de modèles vers les canaux configurés.

## Configuration partagée

```bash
export NOVAPURA_API_KEY="sk-xxxxxxxx"
export NOVAPURA_BASE_URL="https://www.novapuraai.com"   # origin only
```

Les clients SDK ont en général besoin de `base_url` **avec** `/v1`.

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
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
    temperature=0.2,
)

print(llm.invoke("Hello from NovaPuraAI").content)
```

### Embeddings avec LangChain

```python
from langchain_openai import OpenAIEmbeddings

emb = OpenAIEmbeddings(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)
vector = emb.embed_query("gateway documentation")
```

## LangChain.js

```bash
npm install @langchain/openai
```

```typescript
import { ChatOpenAI } from "@langchain/openai";

const llm = new ChatOpenAI({
  model: "gpt-4o-mini",
  apiKey: process.env.NOVAPURA_API_KEY,
  configuration: {
    baseURL: `${process.env.NOVAPURA_BASE_URL}/v1`,
  },
});

const res = await llm.invoke("Hello from NovaPuraAI");
console.log(res.content);
```

## LlamaIndex (Python)

```bash
pip install llama-index-llms-openai llama-index-embeddings-openai
```

```python
import os
from llama_index.llms.openai import OpenAI
from llama_index.embeddings.openai import OpenAIEmbedding

llm = OpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

embed = OpenAIEmbedding(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

print(llm.complete("Hello from NovaPuraAI"))
```

Les noms de paramètres (`api_base` vs `base_url`) varient légèrement selon les versions de LlamaIndex — préférez le mot-clé accepté par le paquet installé.

## Liste de contrôle RAG

1. Utilisez le **même** modèle d’embedding à l’indexation et à la requête.
2. Stockez l’ID du modèle avec les métadonnées de l’index vectoriel.
3. Limitez la concurrence pour respecter les [Limites de débit](/docs/rate-limits).
4. Surveillez les journaux d’usage NovaPuraAI pendant l’évaluation de la qualité de retrieval pour garder un coût prévisible.

## Voir aussi

- [SDK Python](/docs/sdk-python)
- [Embeddings](/docs/api-embeddings)
- [Facturation et quota](/docs/billing)
""",
    "ja": r"""LangChain と LlamaIndex は、Base URL と API キーを上書きすることで、それぞれの OpenAI 連携経由で NovaPuraAI を呼び出せます。ゲートウェイはモデル名を設定済みチャネルへルーティングします。

## 共通設定

```bash
export NOVAPURA_API_KEY="sk-xxxxxxxx"
export NOVAPURA_BASE_URL="https://www.novapuraai.com"   # origin only
```

SDK クライアントは一般に、`/v1` **付き**の `base_url` が必要です。

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
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
    temperature=0.2,
)

print(llm.invoke("Hello from NovaPuraAI").content)
```

### LangChain での Embeddings

```python
from langchain_openai import OpenAIEmbeddings

emb = OpenAIEmbeddings(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)
vector = emb.embed_query("gateway documentation")
```

## LangChain.js

```bash
npm install @langchain/openai
```

```typescript
import { ChatOpenAI } from "@langchain/openai";

const llm = new ChatOpenAI({
  model: "gpt-4o-mini",
  apiKey: process.env.NOVAPURA_API_KEY,
  configuration: {
    baseURL: `${process.env.NOVAPURA_BASE_URL}/v1`,
  },
});

const res = await llm.invoke("Hello from NovaPuraAI");
console.log(res.content);
```

## LlamaIndex（Python）

```bash
pip install llama-index-llms-openai llama-index-embeddings-openai
```

```python
import os
from llama_index.llms.openai import OpenAI
from llama_index.embeddings.openai import OpenAIEmbedding

llm = OpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

embed = OpenAIEmbedding(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

print(llm.complete("Hello from NovaPuraAI"))
```

パラメータ名（`api_base` と `base_url`）は LlamaIndex のバージョンでやや異なります。インストール済みパッケージが受け付けるキーワードを使ってください。

## RAG チェックリスト

1. インデックス時とクエリ時で **同じ** 埋め込みモデルを使います。
2. ベクトルインデックスのメタデータにモデル ID を保存します。
3. [レート制限](/docs/rate-limits) を守るため同時実行数を制限します。
4. 検索品質を評価する間、NovaPuraAI の利用ログを監視してコストを予測可能に保ちます。

## 関連

- [Python SDK](/docs/sdk-python)
- [Embeddings](/docs/api-embeddings)
- [課金とクォータ](/docs/billing)
""",
}

# ---------------------------------------------------------------------------
# integration-nextchat
# ---------------------------------------------------------------------------
DOCS["integration-nextchat"] = {
    "fr": r"""NextChat (ChatGPT-Next-Web et forks compatibles) peut parler à NovaPuraAI via des paramètres compatibles OpenAI. Configurez une fois la base URL, la clé API et le modèle par défaut, puis utilisez l’interface web comme d’habitude.

## Prérequis

- NextChat en cours d’exécution (auto-hébergé ou local)
- Clé API NovaPuraAI
- Origine de déploiement telle que `https://www.novapuraai.com`

## Paramètres à appliquer

Dans NextChat **Settings** (le libellé peut varier selon le fork/version) :

| Champ | Valeur recommandée |
| --- | --- |
| Endpoint / API base | `https://www.novapuraai.com/v1` |
| API key | `sk-xxxxxxxx` |
| Model | Un ID depuis `GET /v1/models` |

Si l’interface stocke uniquement l’origine et ajoute toujours `/v1` elle-même, utilisez `https://www.novapuraai.com` sans dupliquer `/v1`. En cas de doute, ouvrez les outils réseau du navigateur et confirmez que le chemin final est `/v1/chat/completions`.

## Déploiements par variables d’environnement

De nombreuses images Docker NextChat acceptent :

```bash
OPENAI_API_KEY=sk-xxxxxxxx
BASE_URL=https://www.novapuraai.com/v1
# some images use OPENAI_API_BASE / OPENAI_BASE_URL — check your image docs
```

Redémarrez le conteneur après modification des variables d’environnement.

## Test de fumée

```bash
curl "https://www.novapuraai.com/v1/chat/completions" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}]
  }'
```

## Problèmes courants

| Symptôme | Cause | Correctif |
| --- | --- | --- |
| 404 sur le chat | Mauvaise base URL (`/v1` manquant ou doublé) | Alignez avec le chemin de l’onglet réseau |
| 401 | Clé non transmise ou incorrecte | Collez la clé NovaPuraAI, pas une clé OpenAI Platform |
| Liste de modèles vide | Le frontend ne peut pas appeler `/v1/models` | Vérifiez CORS/proxy et les permissions de la clé |
| Erreurs de solde | Pas de quota | Rechargez dans la console NovaPuraAI |

## Notes de sécurité

- Préférez le mode proxy côté serveur si votre build NextChat permet de masquer les clés du navigateur.
- Pour les démos publiques, utilisez des clés à faible quota et des listes blanches de modèles strictes.

## Voir aussi

- [Base URL et endpoints](/docs/base-url)
- [Authentification](/docs/authentication)
- [FAQ](/docs/faq)
""",
    "ja": r"""NextChat（ChatGPT-Next-Web および互換フォーク）は、OpenAI 互換設定経由で NovaPuraAI と通信できます。Base URL・API キー・デフォルトモデルを一度設定すれば、通常どおり Web UI を使えます。

## 前提条件

- 稼働中の NextChat（セルフホストまたはローカル）
- NovaPuraAI API キー
- デプロイメントオリジン（例: `https://www.novapuraai.com`）

## 適用する設定

NextChat の **Settings**（フォーク/バージョンで文言は異なる場合があります）:

| フィールド | 推奨値 |
| --- | --- |
| Endpoint / API base | `https://www.novapuraai.com/v1` |
| API key | `sk-xxxxxxxx` |
| Model | `GET /v1/models` の ID |

UI がオリジンだけを保存し、自身で常に `/v1` を付与する場合は、`/v1` を重複させず `https://www.novapuraai.com` を使います。迷ったらブラウザのネットワークツールで最終パスが `/v1/chat/completions` であることを確認してください。

## 環境変数スタイルのデプロイ

多くの NextChat Docker イメージは次を受け付けます:

```bash
OPENAI_API_KEY=sk-xxxxxxxx
BASE_URL=https://www.novapuraai.com/v1
# some images use OPENAI_API_BASE / OPENAI_BASE_URL — check your image docs
```

環境変数変更後はコンテナを再起動してください。

## スモークテスト

```bash
curl "https://www.novapuraai.com/v1/chat/completions" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}]
  }'
```

## よくある問題

| 症状 | 原因 | 対処 |
| --- | --- | --- |
| チャットで 404 | Base URL 誤り（`/v1` 欠落または二重） | ネットワークタブのパスに合わせる |
| 401 | キー未送信または誤り | OpenAI Platform キーではなく NovaPuraAI キーを貼る |
| モデル一覧が空 | フロントエンドが `/v1/models` を呼べない | CORS/プロキシとキー権限を確認 |
| 残高エラー | クォータなし | NovaPuraAI コンソールでチャージ |

## セキュリティ注意

- NextChat ビルドが対応していれば、ブラウザからキーを隠すサーバーサイドプロキシモードを優先します。
- 公開デモでは低クォータキーと厳格なモデル許可リストを使います。

## 関連

- [Base URL とエンドポイント](/docs/base-url)
- [認証](/docs/authentication)
- [FAQ](/docs/faq)
""",
}

# ---------------------------------------------------------------------------
# integration-openwebui
# ---------------------------------------------------------------------------
DOCS["integration-openwebui"] = {
    "fr": r"""Open WebUI peut utiliser NovaPuraAI comme backend compatible OpenAI. Configurez une connexion avec votre clé API et votre base URL, puis sélectionnez les modèles de la liste renvoyée par NovaPuraAI.

## Prérequis

- Open WebUI installé (Docker est courant)
- Clé API NovaPuraAI avec quota
- Origine telle que `https://www.novapuraai.com`

## Paramètres de connexion administrateur

Dans les paramètres administrateur d’Open WebUI, ouvrez la section **Connections** / **OpenAI** (les libellés varient selon la version) et ajoutez :

| Champ | Valeur |
| --- | --- |
| API Base URL | `https://www.novapuraai.com/v1` |
| API Key | `sk-xxxxxxxx` |

Enregistrez, puis actualisez les modèles. Open WebUI appellera `GET /v1/models` et `POST /v1/chat/completions` contre votre passerelle.

## Exemple d’environnement Docker

Certains déploiements injectent les paramètres du fournisseur via l’environnement :

```bash
OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
OPENAI_API_KEY=sk-xxxxxxxx
```

Les noms exacts des variables dépendent de la version d’Open WebUI — confirmez dans sa documentation si l’interface n’est pas disponible.

## Sélection des modèles

- Seuls les modèles activés pour votre clé apparaissent.
- Si un modèle manque, vérifiez avec curl sur `/v1/models`.
- Pour le chat multimodal, choisissez des modèles dont vos canaux supportent la vision ; la capacité dépend du canal.

## Streaming et tools

- Le chat en streaming fonctionne via le chemin compatible OpenAI lorsque le modèle/canal le supporte.
- L’appel d’outils / de fonctions exige à la fois les feature flags d’Open WebUI et un modèle qui supporte les tools.

## Dépannage

| Problème | À vérifier |
| --- | --- |
| « Incorrect API key » | Préfixe de clé, espaces, jeton désactivé |
| Liste de modèles vide | La base URL doit inclure `/v1` ; le réseau doit atteindre la passerelle |
| 429 sous charge multi-utilisateurs | Limites de débit par clé/groupe ; créez des clés séparées ou relevez les limites |
| Premier token lent | Latence amont ; essayez un autre modèle |

## Voir aussi

- [Modèles et routage](/docs/routing)
- [Limites de débit](/docs/rate-limits)
- [Chat Completions](/docs/api-chat)
""",
    "ja": r"""Open WebUI は NovaPuraAI を OpenAI 互換バックエンドとして利用できます。API キーと Base URL で接続を設定し、NovaPuraAI が返す一覧からモデルを選びます。

## 前提条件

- インストール済みの Open WebUI（Docker が一般的）
- クォータ付きの NovaPuraAI API キー
- オリジン（例: `https://www.novapuraai.com`）

## 管理者の接続設定

Open WebUI の管理設定で **Connections** / **OpenAI** セクション（ラベルはバージョンで異なる）を開き、次を追加します:

| フィールド | 値 |
| --- | --- |
| API Base URL | `https://www.novapuraai.com/v1` |
| API Key | `sk-xxxxxxxx` |

保存後、モデルを更新します。Open WebUI はゲートウェイに対して `GET /v1/models` と `POST /v1/chat/completions` を呼び出します。

## Docker 環境の例

一部のデプロイでは環境変数でプロバイダー設定を注入します:

```bash
OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
OPENAI_API_KEY=sk-xxxxxxxx
```

正確な変数名は Open WebUI のバージョンに依存します。UI が使えない場合は公式ドキュメントで確認してください。

## モデルの選択

- キーで有効なモデルだけが表示されます。
- モデルが無い場合は `/v1/models` を curl で確認します。
- マルチモーダルチャットでは、チャネルが vision に対応したモデルを選びます。能力はチャネル依存です。

## ストリーミングと tools

- モデル/チャネルが対応していれば、OpenAI 互換パスでストリーミングチャットが動作します。
- ツール呼び出し / 関数機能には、Open WebUI の feature flag と tools 対応モデルの両方が必要です。

## トラブルシューティング

| 問題 | 確認すること |
| --- | --- |
| 「Incorrect API key」 | キー接頭辞、空白、無効トークン |
| モデルドロップダウンが空 | Base URL に `/v1` が必要。ネットワークがゲートウェイに届くこと |
| マルチユーザー負荷時の 429 | キー/グループ単位のレート制限。キー分割または上限引き上げ |
| 最初のトークンが遅い | 上流レイテンシ。別モデルを試す |

## 関連

- [モデルとルーティング](/docs/routing)
- [レート制限](/docs/rate-limits)
- [Chat Completions](/docs/api-chat)
""",
}

# ---------------------------------------------------------------------------
# quickstart
# ---------------------------------------------------------------------------
DOCS["quickstart"] = {
    "fr": r"""NovaPuraAI expose une API HTTP compatible OpenAI. Avec une clé API valide et un quota disponible, vous pouvez appeler des modèles via une seule base URL.

## Ce dont vous avez besoin

1. Un compte NovaPuraAI sur le déploiement (par exemple `https://www.novapuraai.com`).
2. Une clé API (`sk-...`) depuis **Console → API Keys**.
3. Un solde ou un quota pour les modèles que vous souhaitez utiliser.

## Base URL

Pour les SDK compatibles OpenAI, définissez `base_url` / `baseURL` sur l’origine du site plus `/v1` :

```text
https://www.novapuraai.com/v1
```

## Créer une clé

1. Connectez-vous à la console.
2. Ouvrez **API Keys** (tokens).
3. Créez une clé. Vous pouvez restreindre les modèles, définir un quota et une expiration.
4. Copiez le secret une seule fois et stockez-le dans une variable d’environnement. Ne le committez jamais.

## Première requête de chat

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello from NovaPuraAI"}]
  }'
```

## SDK OpenAI officiel

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-YOUR_KEY",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(resp.choices[0].message.content)
```

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(resp.choices[0].message.content);
```

## Étapes suivantes

- [Authentification](/docs/authentication)
- [Votre première requête](/docs/first-request)
- [Base URL et endpoints](/docs/base-url)
""",
    "ja": r"""NovaPuraAI は OpenAI 互換の HTTP API を公開します。有効な API キーと利用可能なクォータがあれば、1 つの Base URL 経由でモデルを呼び出せます。

## 必要なもの

1. デプロイメント上の NovaPuraAI アカウント（例: `https://www.novapuraai.com`）。
2. **Console → API Keys** で発行した API キー（`sk-...`）。
3. 利用したいモデル向けの残高またはクォータ。

## Base URL

OpenAI 互換 SDK では、`base_url` / `baseURL` をサイトのオリジンに `/v1` を付けた値に設定します:

```text
https://www.novapuraai.com/v1
```

## キーの作成

1. コンソールにログインします。
2. **API Keys**（トークン）を開きます。
3. キーを作成します。モデル制限、クォータ、有効期限を任意で設定できます。
4. シークレットは一度だけコピーし、環境変数に保存します。リポジトリにコミットしないでください。

## 最初のチャットリクエスト

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello from NovaPuraAI"}]
  }'
```

## 公式 OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-YOUR_KEY",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(resp.choices[0].message.content)
```

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(resp.choices[0].message.content);
```

## 次のステップ

- [認証](/docs/authentication)
- [最初のリクエスト](/docs/first-request)
- [Base URL とエンドポイント](/docs/base-url)
""",
}

# ---------------------------------------------------------------------------
# rate-limits
# ---------------------------------------------------------------------------
DOCS["rate-limits"] = {
    "fr": r"""Les limites de débit protègent la plateforme et les fournisseurs amont. Les limites peuvent s’appliquer au niveau IP, utilisateur ou jeton selon les paramètres administrateur.

## Symptômes typiques

- HTTP `429 Too Many Requests`
- Messages d’erreur mentionnant rate limit ou frequency

## Conseils côté client

1. Backoff exponentiel avec jitter sur `429` et `5xx` transitoires.
2. Réutilisez les connexions HTTP ; évitez d’ouvrir une nouvelle session TLS pour chaque petite requête lorsque c’est possible.
3. Regroupez le travail lorsque l’API le permet (par exemple tableaux d’embeddings).
4. Mettez en cache les listes de modèles et la configuration statique.

## Streaming

Les flux de longue durée occupent une connexion. Concevez des limites de concurrence pour ne pas ouvrir plus de flux parallèles que votre plan n’autorise.

## Réglages côté admin

Les administrateurs peuvent ajuster les limites de débit globales et par modèle dans les paramètres système. Contactez l’opérateur du site si le trafic légitime est trop agressivement limité.
""",
    "ja": r"""レート制限はプラットフォームと上流プロバイダーを保護します。管理者設定により、IP・ユーザー・トークン単位で適用される場合があります。

## 典型的な症状

- HTTP `429 Too Many Requests`
- rate limit や frequency に言及するエラーメッセージ

## クライアント側の指針

1. `429` と一時的な `5xx` ではジッター付き指数バックオフを使います。
2. HTTP 接続を再利用し、可能なら小さなリクエストごとに新しい TLS セッションを開かないようにします。
3. API が対応している場合はバッチ処理します（例: embeddings の配列）。
4. モデル一覧と静的設定をキャッシュします。

## ストリーミング

長時間のストリームは接続を保持します。プランで許可される以上の並列ストリームを開かないよう、同時実行上限を設計してください。

## 管理側の調整

管理者はシステム設定で全体およびモデル別のレート制限を調整できます。正当なトラフィックが過剰に制限される場合はサイト運営者に連絡してください。
""",
}

# ---------------------------------------------------------------------------
# routing
# ---------------------------------------------------------------------------
DOCS["routing"] = {
    "fr": r"""Lorsqu’un client demande un nom de modèle, NovaPuraAI sélectionne un canal amont capable de servir ce modèle, sous réserve des permissions de groupe, de la santé du canal et des règles de routage administrateur.

## Noms de modèles

- Utilisez les identifiants de modèle affichés dans **Model Square** ou `GET /v1/models`.
- Les noms sont sensibles à la casse et doivent correspondre à la configuration administrateur.
- Le même modèle logique peut être servi par plusieurs clés amont pour la bascule (failover).

## Fonctionnement du routage (conceptuel)

1. Authentifier la clé API et résoudre le groupe utilisateur.
2. Trouver les canaux qui exposent le modèle demandé pour ce groupe.
3. Préférer les canaux sains ; ignorer les canaux désactivés ou en panne.
4. Transmettre la requête, traduire les formats de fournisseur si besoin, et facturer l’usage.

## Failover et fiabilité

Les administrateurs peuvent configurer retries, cooldowns et règles de désactivation automatique. Du point de vue client, vous appelez toujours une base URL stable — NovaPuraAI absorbe le churn des fournisseurs lorsque plusieurs canaux sont disponibles.

## Groupes

Les utilisateurs peuvent appartenir à des groupes avec des catalogues de modèles et des ratios différents. Si un modèle fonctionne pour un compte mais pas un autre, comparez les capacités de groupe et les restrictions de clé.

## Bonnes pratiques

- Fixez les noms de modèles dans la configuration, pas en chaînes codées en dur éparpillées dans de nombreux services.
- Préférez lister les modèles via l’API au démarrage si votre produit a besoin d’une découverte dynamique.
- Gérez les `5xx` et timeouts avec des retries côté client pour les lectures idempotentes ; évitez les retries à l’aveugle pour les effets de bord non idempotents.
""",
    "ja": r"""クライアントがモデル名を要求すると、NovaPuraAI はグループ権限・チャネル健全性・管理者のルーティング規則に従い、そのモデルを提供できる上流チャネルを選びます。

## モデル名

- **Model Square** または `GET /v1/models` に表示されるモデル識別子を使います。
- 名前は大文字小文字を区別し、管理者設定と一致する必要があります。
- 同じ論理モデルを、フェイルオーバー用に複数の上流キーで提供できます。

## ルーティングの仕組み（概念）

1. API キーを認証し、ユーザーグループを解決します。
2. そのグループ向けに要求モデルを公開しているチャネルを探します。
3. 健全なチャネルを優先し、無効または障害中のチャネルはスキップします。
4. リクエストを転送し、必要に応じてプロバイダー形式を変換し、利用量を課金します。

## フェイルオーバーと信頼性

管理者はリトライ、クールダウン、自動無効化ルールを設定できます。クライアントからは常に 1 つの安定した Base URL を呼びます。複数チャネルがある場合、NovaPuraAI がプロバイダー側の変動を吸収します。

## グループ

ユーザーは、モデルカタログや比率が異なるグループに所属できます。あるアカウントでは動くモデルが別アカウントで動かない場合は、グループ能力とキー制限を比較してください。

## ベストプラクティス

- モデル名は設定に固定し、多数のサービスにハードコードした一回限りの文字列を散らばせないでください。
- 動的発見が必要な製品では、起動時に API でモデル一覧を取得することを推奨します。
- べき等な読み取りでは `5xx` とタイムアウトにクライアント側リトライを使い、非べき等な副作用には盲目的な再試行を避けます。
""",
}

# ---------------------------------------------------------------------------
# sdk-curl
# ---------------------------------------------------------------------------
DOCS["sdk-curl"] = {
    "fr": r"""curl est idéal pour le débogage et les tests de fumée en CI.

## Chat

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

## Lister les modèles

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## Afficher le JSON en clair

Passez par `jq` lorsqu’il est disponible :

```bash
curl -s ... | jq .
```

## Débogage verbeux

Ajoutez `-v` pour inspecter TLS et les en-têtes. Masquez Authorization lorsque vous partagez des journaux.
""",
    "ja": r"""curl はデバッグと CI のスモークテストに適しています。

## チャット

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

## モデル一覧

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## JSON の整形表示

利用可能なら `jq` にパイプします:

```bash
curl -s ... | jq .
```

## 詳細デバッグ

`-v` を付けて TLS とヘッダーを確認します。ログを共有するときは Authorization を伏せてください。
""",
}

# ---------------------------------------------------------------------------
# sdk-go
# ---------------------------------------------------------------------------
DOCS["sdk-go"] = {
    "fr": r"""Les clients Go peuvent appeler l’API HTTP directement ou utiliser un SDK Go compatible OpenAI.

## HTTP direct

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

## Conseils

- Définissez des timeouts raisonnables pour le non-stream et des timeouts plus longs pour le streaming ou les médias.
- Propagez l’annulation de contexte pour interrompre les requêtes en vol lorsque les handlers se terminent.
""",
    "ja": r"""Go クライアントは HTTP API を直接呼ぶか、OpenAI 互換の Go SDK を使えます。

## 直接 HTTP

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

## ヒント

- 非ストリームでは妥当なタイムアウト、ストリーミングやメディアではより長いタイムアウトを設定します。
- ハンドラが戻るときに進行中リクエストを中断できるよう、コンテキストキャンセルを伝播します。
""",
}

# ---------------------------------------------------------------------------
# sdk-node
# ---------------------------------------------------------------------------
DOCS["sdk-node"] = {
    "fr": r"""Utilisez le paquet npm officiel `openai`.

## Installation

```bash
npm install openai
# or: bun add openai
```

## Client

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
  messages: [{ role: "user", content: "Hello" }],
});
console.log(completion.choices[0].message.content);
```

## Streaming

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
""",
    "ja": r"""公式の `openai` npm パッケージを使用します。

## インストール

```bash
npm install openai
# or: bun add openai
```

## クライアント

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
```

## チャット

```javascript
const completion = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(completion.choices[0].message.content);
```

## ストリーミング

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
""",
}

# ---------------------------------------------------------------------------
# sdk-python
# ---------------------------------------------------------------------------
DOCS["sdk-python"] = {
    "fr": r"""Utilisez le paquet Python officiel `openai` avec une base URL personnalisée.

## Installation

```bash
pip install openai
```

## Client

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
    messages=[{"role": "user", "content": "Hello"}],
)
print(completion.choices[0].message.content)
```

## Streaming

```python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Stream a short poem."}],
    stream=True,
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

## Embeddings

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="NovaPuraAI gateway",
)
vector = emb.data[0].embedding
```
""",
    "ja": r"""公式の `openai` Python パッケージを、カスタム Base URL 付きで使用します。

## インストール

```bash
pip install openai
```

## クライアント

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## チャット

```python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(completion.choices[0].message.content)
```

## ストリーミング

```python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Stream a short poem."}],
    stream=True,
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

## Embeddings

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="NovaPuraAI gateway",
)
vector = emb.data[0].embedding
```
""",
}


EXPECTED = [
    "api-chat",
    "api-embeddings",
    "api-errors",
    "api-gemini",
    "api-media",
    "api-messages",
    "api-models",
    "authentication",
    "base-url",
    "billing",
    "faq",
    "first-request",
    "integration-cursor",
    "integration-dify",
    "integration-langchain",
    "integration-nextchat",
    "integration-openwebui",
    "quickstart",
    "rate-limits",
    "routing",
    "sdk-curl",
    "sdk-go",
    "sdk-node",
    "sdk-python",
]


def strip_fence(text: str) -> str:
    return text.strip() + "\n"


def main() -> None:
    missing = [s for s in EXPECTED if s not in DOCS]
    if missing:
        raise SystemExit(f"Missing sections in DOCS: {missing}")

    written = 0
    failures: list[str] = []
    for section in EXPECTED:
        for lang in ("fr", "ja"):
            path = ROOT / section / f"{lang}.md"
            try:
                body = strip_fence(DOCS[section][lang])
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(body, encoding="utf-8")
                written += 1
                print(f"OK {path.relative_to(ROOT.parent.parent.parent.parent)}")
            except Exception as e:
                failures.append(f"{section}/{lang}: {e}")
                print(f"FAIL {section}/{lang}: {e}")

    print(f"\nWritten: {written}")
    print(f"Failures: {len(failures)}")
    for f in failures:
        print(" -", f)

    # Verification checks
    banned_phrases = [
        "Self-hosting? Replace the host",
        "examples remain in English",
        "Les exemples de code et chemins",
        "コード例と API パスは技術識別子",
    ]
    verify_fail = 0
    for section in EXPECTED:
        for lang in ("fr", "ja"):
            path = ROOT / section / f"{lang}.md"
            text = path.read_text(encoding="utf-8")
            if text.startswith("# "):
                print(f"VERIFY FAIL top-level H1: {section}/{lang}")
                verify_fail += 1
            for phrase in banned_phrases:
                if phrase in text:
                    print(f"VERIFY FAIL banned phrase in {section}/{lang}: {phrase}")
                    verify_fail += 1
            # Outside fences, detect long pure-English intro leftovers
            if "is the primary OpenAI-compatible" in text or "exposes an OpenAI-compatible HTTP API" in text:
                print(f"VERIFY FAIL leftover English prose: {section}/{lang}")
                verify_fail += 1
    print(f"Verify failures: {verify_fail}")
    if verify_fail or failures:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
