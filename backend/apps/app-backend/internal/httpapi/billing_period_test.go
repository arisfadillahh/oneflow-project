package httpapi

import (
	"errors"
	"testing"
)

func TestResolveCheckoutBillingPeriod(t *testing.T) {
	tests := []struct {
		name            string
		requestedPeriod string
		packagePeriod   string
		wantPeriod      string
		wantErr         error
	}{
		{name: "legacy monthly request remains valid", packagePeriod: "monthly", wantPeriod: "monthly"},
		{name: "explicit monthly request matches package", requestedPeriod: "monthly", packagePeriod: "monthly", wantPeriod: "monthly"},
		{name: "annual derives from monthly service package", requestedPeriod: "annual", packagePeriod: "monthly", wantPeriod: "annual"},
		{name: "yearly alias derives to annual period", requestedPeriod: "yearly", packagePeriod: "monthly", wantPeriod: "annual"},
		{name: "explicit top up request matches package", requestedPeriod: "one_time", packagePeriod: "one_time", wantPeriod: "one_time"},
		{name: "annual catalog row is unsupported", packagePeriod: "annual", wantErr: errUnsupportedCheckoutPeriod},
		{name: "mismatched package period is rejected", requestedPeriod: "monthly", packagePeriod: "one_time", wantErr: errCheckoutBillingPeriodMismatch},
		{name: "unknown catalog period is rejected", packagePeriod: "quarterly", wantErr: errUnsupportedCheckoutPeriod},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			period, err := resolveCheckoutBillingPeriod(test.requestedPeriod, test.packagePeriod)
			if test.wantErr == nil && err != nil {
				t.Fatalf("resolve checkout period returned %v, want nil", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("resolve checkout period returned %v, want %v", err, test.wantErr)
			}
			if err == nil && period != test.wantPeriod {
				t.Fatalf("resolve checkout period returned %q, want %q", period, test.wantPeriod)
			}
		})
	}
}

func TestAnnualCheckoutAmount(t *testing.T) {
	tests := []struct {
		name     string
		monthly  float64
		discount float64
		want     float64
	}{
		{name: "starter with default discount", monthly: 99000, discount: 10, want: 1069200},
		{name: "growth with default discount", monthly: 299000, discount: 10, want: 3229200},
		{name: "discount is clamped below zero", monthly: 99000, discount: -5, want: 1188000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := annualCheckoutAmount(test.monthly, test.discount); got != test.want {
				t.Fatalf("annual checkout amount returned %.0f, want %.0f", got, test.want)
			}
		})
	}
}
