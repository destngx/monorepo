package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestGivenTheProductRowWhenParsingPreferencesThenLinksAreReadAndBadOnesReported(t *testing.T) {
	prefs := ParsePreferences([][]string{
		{"idleCash.accountProducts", `{" Wallet Z ":" Zalopay:So-Du-Sinh-Loi ","Bad":"not an id","":"momo:tui-than-tai"}`, "note"},
	}, 2)
	settings := prefs.IdleCash.AccountProducts
	if len(settings.Accounts) != 1 || settings.Accounts[0] != (AccountProduct{Account: "Wallet Z", Product: "zalopay:so-du-sinh-loi"}) ||
		settings.ETag == preferenceETag("") {
		t.Fatalf("settings = %+v", settings)
	}
	if len(prefs.Issues) != 2 || !strings.Contains(strings.Join(prefs.Issues, "\n"), `"Bad" needs a product ID`) {
		t.Fatalf("issues = %q", prefs.Issues)
	}
	if product, ok := settings.For("wallet z"); !ok || product != "zalopay:so-du-sinh-loi" {
		t.Fatalf("lookup ignores case and spaces: %q %v", product, ok)
	}
	broken := ParsePreferences([][]string{{"idleCash.accountProducts", `["Wallet Z"]`}}, 2)
	if len(broken.IdleCash.AccountProducts.Accounts) != 0 || len(broken.Issues) != 1 {
		t.Fatalf("a value that is not an object links nothing: %+v", broken)
	}
}

func TestGivenALinkWhenSettingItThenTheRowIsAddedChangedAndRemovedWithTheETag(t *testing.T) {
	current := ParsePreferences(nil, 2)
	next, err := ApplyAccountProduct(current, AccountProductInput{Account: " Wallet Z ", Product: "zalopay:so-du-sinh-loi", ETag: current.IdleCash.AccountProducts.ETag})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(next.Entries) != 1 || next.Entries[0].Row != 0 || next.Entries[0].Note == "" ||
		len(next.IdleCash.AccountProducts.Accounts) != 1 || next.IdleCash.AccountProducts.Accounts[0].Account != "Wallet Z" {
		t.Fatalf("next = %+v", next)
	}
	encoded, err := EncodeAccountProducts(next.Entries[0].Products)
	if err != nil || encoded != `{"Wallet Z":"zalopay:so-du-sinh-loi"}` {
		t.Fatalf("encoded = %q %v", encoded, err)
	}
	saved := ParsePreferences([][]string{{string(PrefIdleCashAccountProducts), encoded}}, 2)

	changed, err := ApplyAccountProduct(saved, AccountProductInput{Account: "wallet z", Product: "momo:tui-than-tai", ETag: saved.IdleCash.AccountProducts.ETag})
	if err != nil || len(changed.IdleCash.AccountProducts.Accounts) != 1 || changed.IdleCash.AccountProducts.Accounts[0].Product != "momo:tui-than-tai" {
		t.Fatalf("changed = %+v %v", changed.IdleCash.AccountProducts, err)
	}
	cleared, err := ApplyAccountProduct(saved, AccountProductInput{Account: "WALLET Z", ETag: saved.IdleCash.AccountProducts.ETag})
	if err != nil || len(cleared.IdleCash.AccountProducts.Accounts) != 0 {
		t.Fatalf("a blank product unlinks: %+v %v", cleared.IdleCash.AccountProducts, err)
	}
	if empty, err := EncodeAccountProducts(nil); empty != "" || err != nil {
		t.Fatalf("no links clear the cell: %q %v", empty, err)
	}
	if _, err := ApplyAccountProduct(saved, AccountProductInput{Account: "Wallet Z", Product: "momo:tui-than-tai", ETag: current.IdleCash.AccountProducts.ETag}); !errors.Is(err, ErrConflict) {
		t.Fatalf("a stale etag is a conflict: %v", err)
	}
	for _, bad := range []AccountProductInput{
		{Account: "Wallet Z", Product: "momo:tui-than-tai"},
		{Product: "momo:tui-than-tai", ETag: "x"},
		{Account: "Wallet Z", Product: "MoMo Túi Thần Tài", ETag: "x"},
	} {
		if _, err := ApplyAccountProduct(saved, bad); !errors.Is(err, ErrValidation) {
			t.Fatalf("%+v: err = %v", bad, err)
		}
	}
}

func TestGivenTheProductKeyWhenSavingItAsANumberThenTheRouteIsNamed(t *testing.T) {
	err := PreferencesUpdate{Values: []PreferenceValue{{Key: PrefIdleCashAccountProducts, Value: money(1)}}}.Validate()
	if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "/api/bills/idle-cash/account-products") {
		t.Fatalf("err = %v", err)
	}
}
