package service

import (
	"apps/wealth-management/domain"
	"errors"
	"testing"
)

func seedAccounts(client *fakeSheetClient) {
	client.ranges[accountsSheetRange] = [][]string{
		{"Bank", "", "", "", "0", "1000", "active use", "daily", "VND"},
		{"[DELETED] Gone", "", "", "", "0", "5", "bank"},
		{"Binance", "", "", "", "0", "200", "crypto"},
	}
}

func TestListAccountsExcludesDeletedAndResolvesCurrency(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	svc := newTestDBService(t, client, newFakeCache())

	accounts, err := svc.ListAccounts(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 2 || accounts[1].Name != "Binance" || accounts[1].Currency != domain.CurrencyUSDT {
		t.Fatalf("accounts = %+v", accounts)
	}
}

func TestListAccountsHonoursForce(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	svc := newTestDBService(t, client, newFakeCache())

	for _, force := range []bool{false, false, true} {
		if _, err := svc.ListAccounts(force); err != nil {
			t.Fatal(err)
		}
	}
	if len(client.reads) != 2 {
		t.Fatalf("reads = %v, want cached second read and fresh forced read", client.reads)
	}
}

func TestCreateAccountPersistsCurrencyAndRejectsDuplicates(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	svc := newTestDBService(t, client, newFakeCache())

	if err := svc.CreateAccount(domain.AccountCreateInput{Name: "Binance Futures", Type: domain.AccountTypeCrypto}); err != nil {
		t.Fatal(err)
	}
	row := client.appends[0].rows[0]
	if row[0] != "Binance Futures" || row[8] != "USDT" || client.appends[0].anchor != accountsAnchorRange {
		t.Fatalf("row = %#v", row)
	}
	err := svc.CreateAccount(domain.AccountCreateInput{Name: "Bank", Type: domain.AccountTypeBank})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate err = %v", err)
	}
}

func TestUpdateAccountWritesTypeAndNoteOnly(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	svc := newTestDBService(t, client, newFakeCache())

	account, err := svc.UpdateAccount("Binance", domain.AccountUpdateInput{Note: ptr("cold wallet")})
	if err != nil {
		t.Fatal(err)
	}
	update := client.updates[0]
	if update.rangeValue != "Accounts!G4:H4" || update.values[0] != "crypto" || update.values[1] != "cold wallet" {
		t.Fatalf("update = %+v", update)
	}
	if account.Note == nil || *account.Note != "cold wallet" || account.Type != domain.AccountTypeCrypto {
		t.Fatalf("account = %+v", account)
	}
	if _, err := svc.UpdateAccount("Gone", domain.AccountUpdateInput{Note: ptr("x")}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing err = %v", err)
	}
}

func TestDeleteAccountSoftDeletes(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	svc := newTestDBService(t, client, newFakeCache())

	if err := svc.DeleteAccount("Bank"); err != nil {
		t.Fatal(err)
	}
	if client.updates[0].rangeValue != "Accounts!A2" || client.updates[0].values[0] != "[DELETED] Bank" {
		t.Fatalf("update = %+v", client.updates[0])
	}
	if err := svc.DeleteAccount("[DELETED] Gone"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("already deleted err = %v", err)
	}
}
