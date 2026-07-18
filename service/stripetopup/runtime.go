package stripetopup

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/stripe/stripe-go/v85"
)

const stripeTopupProductName = "NovaPuraAI API Credits Top-up"

// ValidateStripeRuntime verifies that the selected credential profile resolves
// the expected Stripe account and the single Product used by dynamic price_data.
func ValidateStripeRuntime(ctx context.Context) error {
	if err := ValidateStripeSecrets(); err != nil {
		return err
	}

	client := stripe.NewClient(strings.TrimSpace(setting.StripeApiSecret))
	account, err := client.V1Accounts.Retrieve(ctx, nil)
	if err != nil {
		return fmt.Errorf("retrieve stripe account: %w", err)
	}
	product, err := client.V1Products.Retrieve(ctx, strings.TrimSpace(setting.StripeTopupProductID), nil)
	if err != nil {
		return fmt.Errorf("retrieve stripe top-up Product: %w", err)
	}
	expectedLivemode := !setting.StripeRequireTestKeys
	return validateStripeRuntimeResources(
		strings.TrimSpace(setting.StripeAccountID),
		account.ID,
		strings.TrimSpace(setting.StripeTopupProductID),
		expectedLivemode,
		product,
	)
}

func validateStripeRuntimeResources(expectedAccountID, actualAccountID, expectedProductID string, expectedLivemode bool, product *stripe.Product) error {
	if expectedAccountID == "" || actualAccountID != expectedAccountID {
		return fmt.Errorf("stripe account mismatch: configured account does not match active credentials")
	}
	if product == nil || product.ID != expectedProductID || product.Name != stripeTopupProductName {
		return fmt.Errorf("stripe top-up Product mismatch")
	}
	if !product.Active {
		return fmt.Errorf("stripe top-up Product is inactive")
	}
	if product.Livemode != expectedLivemode {
		return fmt.Errorf("stripe Product environment does not match active credentials")
	}
	return nil
}
