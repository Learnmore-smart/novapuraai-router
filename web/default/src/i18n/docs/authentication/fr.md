Chaque requête de relais doit présenter une clé API NovaPuraAI. Les clés sont gérées dans la console et validées par le middleware d’authentification du gateway.

> Les exemples de code et chemins d’API restent en anglais (identifiants techniques).

Every relay request must present a NovaPuraAI API key. Keys are managed in the console and validated by `TokenAuth` on the gateway.

## Header format

Send the key as a Bearer token:

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

Some OpenAI clients also accept `api_key` in the SDK constructor — that value becomes the same Authorization header.

## Where to create keys

1. Sign in → **API Keys**.
2. Create a key with an optional name.
3. Configure model allowlists, remaining quota, IP limits, and expiry if needed.
4. Save the secret immediately. The full secret is only shown once.

## Security best practices

- Prefer environment variables (`NOVAPURA_API_KEY`) over hard-coding.
- Use separate keys per environment (dev / staging / production).
- Rotate keys if a client is compromised.
- Restrict keys to the minimum set of models your app needs.
- Do not embed keys in public frontend bundles.

## Common failures

| Symptom            | Likely cause                                       |
| ------------------ | -------------------------------------------------- |
| `401 Unauthorized` | Missing/invalid key, revoked key, or wrong header  |
| `403 Forbidden`    | Model not allowed for this key, or module disabled |
| `429`              | Rate limit exceeded                                |
| Insufficient quota | Balance too low or key quota exhausted             |

## Multi-user setups

Administrators can issue keys to end users with independent quotas. Each key is billed against its owner’s balance according to platform settings.
