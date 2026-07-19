NovaPuraAI 接続時によくある質問です。

> コード例と API パスは技術識別子のため英語のままです。

## Do I need to deploy a separate API service?

No. NovaPuraAI **is** the API gateway. After you deploy the app (for example on Google Cloud Run) and create keys, clients call your domain directly.

## What is the difference between docs_link and /docs?

`/docs` is the in-app official guide on this site. The optional **Documentation Link** setting is an external URL that can appear in the footer for additional resources.

## Why do I get 401?

The Authorization header is missing, the key is wrong, or the key was deleted/disabled.

## Why do I get model_not_found?

The model string is not enabled for your group, or the key’s model whitelist excludes it.

## Can I use the same key in production and local dev?

Yes, but separate keys are safer for rotation and spend tracking.

## Does Cloud Run work?

Yes. Use Cloud SQL (or another managed database), Redis for multi-instance cache, and stable `SESSION_SECRET` / `CRYPTO_SECRET`. Do not rely on SQLite inside the container for production.

## Where is usage shown?

Console → usage logs / dashboard. Administrators see system-wide metrics.

## Is the API OpenAI compatible?

Yes for the main chat, embeddings, images, and audio routes. Additional Anthropic and Gemini surfaces are also available for supported models.
