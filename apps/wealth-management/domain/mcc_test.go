package domain

import (
	"reflect"
	"testing"
)

func TestParseMCCList(t *testing.T) {
	tests := map[string][]MCCRange{
		"":                      nil,
		"5411":                  {{From: 5411, To: 5411}},
		"5411, 5499; 5812-5814": {{From: 5411, To: 5411}, {From: 5499, To: 5499}, {From: 5812, To: 5814}},
		"742 4111 - 4112":       {{From: 742, To: 742}, {From: 4111, To: 4112}},
	}
	for input, want := range tests {
		got, err := ParseMCCList(input)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%q: got %v %v want %v", input, got, err, want)
		}
	}
	for _, input := range []string{"54x1", "10000", "5814-5812", "5411-", "-1"} {
		if _, err := ParseMCCList(input); err == nil {
			t.Fatalf("%q should be rejected", input)
		}
	}
}

func TestFormatMCCList(t *testing.T) {
	if got := FormatMCCList([]MCCRange{{From: 742, To: 742}, {From: 5812, To: 5814}}); got != "0742, 5812-5814" {
		t.Fatalf("got %q", got)
	}
}

func TestTransactionMCC(t *testing.T) {
	merchants := []MerchantMCC{{Pattern: "Coop", MCC: 5411}, {Pattern: "Coop Food", MCC: 5499, Confidence: MCCConfirmed}, {Pattern: "Grab", MCC: 4121}}
	payee := func(name string, tags ...string) Transaction {
		return Transaction{Payee: name, Tags: tags}
	}
	tests := []struct {
		name string
		tx   Transaction
		want PaymentMCC
	}{
		{"payee pattern, assumed by default", payee("COOP Mart Q1"), PaymentMCC{5411, MCCAssumed}},
		{"longest pattern wins with its confidence", payee("Coop Food Nguyen Trai"), PaymentMCC{5499, MCCConfirmed}},
		{"whole words only", payee("Grabbit"), PaymentMCC{Status: MCCUnknown}},
		{"tag overrides payee and is confirmed", payee("Grab", "mcc:5812"), PaymentMCC{5812, MCCConfirmed}},
		{"unknown payee", payee("Corner shop"), PaymentMCC{Status: MCCUnknown}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TransactionMCC(tt.tx, merchants)
			if got != tt.want {
				t.Fatalf("got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestTransactionMCCFollowsMerchantChangesOverTime(t *testing.T) {
	may := day(2026, 5, 1)
	june := day(2026, 6, 1)
	merchants := []MerchantMCC{
		{Pattern: "Coop", MCC: 5411},
		{Pattern: "Coop", MCC: 5499, From: &may},
		{Pattern: "Coop", MCC: 5311, From: &june},
		{Pattern: "Coop Food", MCC: 5812, From: &june},
		{Pattern: "New Shop", MCC: 5651, From: &may},
	}
	tests := []struct {
		name  string
		tx    Transaction
		mcc   int
		known bool
	}{
		{"before any change uses the undated mapping", Transaction{Payee: "Coop", Date: day(2026, 4, 30)}, 5411, true},
		{"the change applies from its date", Transaction{Payee: "Coop", Date: may}, 5499, true},
		{"the latest change in effect wins", Transaction{Payee: "Coop", Date: day(2026, 7, 2)}, 5311, true},
		{"a longer pattern applies only once in effect", Transaction{Payee: "Coop Food", Date: day(2026, 5, 20)}, 5499, true},
		{"a longer pattern in effect wins", Transaction{Payee: "Coop Food", Date: june}, 5812, true},
		{"no mapping in effect yet", Transaction{Payee: "New Shop", Date: day(2026, 4, 1)}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TransactionMCC(tt.tx, merchants)
			if got.Code != tt.mcc || got.Known() != tt.known {
				t.Fatalf("got %+v want %d %v", got, tt.mcc, tt.known)
			}
		})
	}
}
