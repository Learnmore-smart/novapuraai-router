用量依請求計量。閘道會預估費用並在需要時預扣額度，再在上游回應後結算。

## 概念

- **Balance / quota** — prepaid credit on the user account (and optionally per key).
- **Model pricing** — configured by administrators (token ratios, fixed prices, or expression-based rules).
- **Pre-consume** — holds estimated quota so concurrent requests cannot overspend.
- **Settle** — adjusts the final charge from actual token usage when available.

## 用戶端需要了解

1. A successful API call may still fail later if upstream returns an error after pre-consume (unused hold is refunded according to platform logic).
2. Streaming responses bill on completed usage when the provider reports tokens.
3. Image, audio, and video products may bill by count, duration, or resolution multipliers — always check Model Square.

## 錢包儲值

End users top up from the **Wallet** page when payment gateways are enabled (for example Stripe). Administrators control currencies, promotions, and limits.

## 監控消耗

- Console usage logs show model, tokens, and quota delta per request.
- Keep separate API keys per product so spend is easy to attribute.

## 額度不足

If a request is rejected for quota, top up the wallet or ask an admin to increase limits. Creating a new key does not create free balance.
