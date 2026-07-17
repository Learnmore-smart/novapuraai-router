package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAdminCompleteTopupRequest(t *testing.T) {
	tests := []struct {
		name    string
		request AdminCompleteTopupRequest
		wantErr bool
	}{
		{
			name:    "valid",
			request: AdminCompleteTopupRequest{TradeNo: " order-123 ", Reason: " bank transfer confirmed "},
		},
		{
			name:    "missing trade number",
			request: AdminCompleteTopupRequest{Reason: "bank transfer confirmed"},
			wantErr: true,
		},
		{
			name:    "missing audit reason",
			request: AdminCompleteTopupRequest{TradeNo: "order-123", Reason: "   "},
			wantErr: true,
		},
		{
			name:    "trade number too long",
			request: AdminCompleteTopupRequest{TradeNo: strings.Repeat("x", 256), Reason: "confirmed"},
			wantErr: true,
		},
		{
			name:    "reason too long",
			request: AdminCompleteTopupRequest{TradeNo: "order-123", Reason: strings.Repeat("x", 513)},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := test.request
			err := validateAdminCompleteTopupRequest(&request)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "order-123", request.TradeNo)
			assert.Equal(t, "bank transfer confirmed", request.Reason)
		})
	}
}
