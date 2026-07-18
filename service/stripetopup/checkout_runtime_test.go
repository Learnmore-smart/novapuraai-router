package stripetopup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v85"
)

func TestStripeSandboxRuntimeIntegration(t *testing.T) {
	if os.Getenv("STRIPE_SANDBOX_INTEGRATION") != "1" {
		t.Skip("set STRIPE_SANDBOX_INTEGRATION=1 to verify configured Stripe resources")
	}
	repositoryRoot := filepath.Join("..", "..")
	require.NoError(t, godotenv.Load(filepath.Join(repositoryRoot, ".env")))
	t.Setenv("GIN_MODE", "debug")
	setting.InitStripeEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, ValidateStripeRuntime(ctx))
}

func TestValidateStripeRuntimeResourcesAcceptsMatchingSandboxProduct(t *testing.T) {
	product := &stripe.Product{
		ID:       "prod_sandbox",
		Name:     stripeTopupProductName,
		Active:   true,
		Livemode: false,
	}
	require.NoError(t, validateStripeRuntimeResources("acct_sandbox", "acct_sandbox", "prod_sandbox", false, product))
}

func TestValidateStripeRuntimeResourcesRejectsAccountMismatch(t *testing.T) {
	product := &stripe.Product{ID: "prod_sandbox", Name: stripeTopupProductName, Active: true}
	err := validateStripeRuntimeResources("acct_expected", "acct_other", "prod_sandbox", false, product)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}

func TestValidateStripeRuntimeResourcesRejectsLiveProductInSandbox(t *testing.T) {
	product := &stripe.Product{ID: "prod_sandbox", Name: stripeTopupProductName, Active: true, Livemode: true}
	err := validateStripeRuntimeResources("acct_sandbox", "acct_sandbox", "prod_sandbox", false, product)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "environment")
}

func TestValidateStripeRuntimeResourcesRejectsUnexpectedProduct(t *testing.T) {
	product := &stripe.Product{ID: "prod_other", Name: "Other product", Active: true}
	err := validateStripeRuntimeResources("acct_sandbox", "acct_sandbox", "prod_sandbox", false, product)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Product")
}
