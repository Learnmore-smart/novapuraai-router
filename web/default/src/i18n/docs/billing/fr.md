L’usage est facturé par requête. Le gateway estime le coût, pré-consomme le quota si nécessaire, puis solde après la réponse amont.

> Les exemples de code et chemins d’API restent en anglais (identifiants techniques).

Usage is metered per request. The gateway estimates cost, pre-consumes quota when required, then settles after the upstream response.

## Concepts

- **Balance / quota** — prepaid credit on the user account (and optionally per key).
- **Model pricing** — configured by administrators (token ratios, fixed prices, or expression-based rules).
- **Pre-consume** — holds estimated quota so concurrent requests cannot overspend.
- **Settle** — adjusts the final charge from actual token usage when available.

## What clients should know

1. A successful API call may still fail later if upstream returns an error after pre-consume (unused hold is refunded according to platform logic).
2. Streaming responses bill on completed usage when the provider reports tokens.
3. Image, audio, and video products may bill by count, duration, or resolution multipliers — always check Model Square.

## Wallet top-ups

End users top up from the **Wallet** page when payment gateways are enabled (for example Stripe). Administrators control currencies, promotions, and limits.

## Monitoring spend

- Console usage logs show model, tokens, and quota delta per request.
- Keep separate API keys per product so spend is easy to attribute.

## Insufficient quota

If a request is rejected for quota, top up the wallet or ask an admin to increase limits. Creating a new key does not create free balance.
