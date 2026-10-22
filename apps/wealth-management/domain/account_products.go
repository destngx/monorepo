package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Accounts linked to a flexible savings product (D41): the account is the
// user's balance in the product (e.g. Zalopay Số dư sinh lời), so its
// current rate is the product's live published rate instead of a rate
// written in the account note. Saved in idleCash.accountProducts as a JSON
// object of account name to product ID, e.g.
// {"Zalopay":"zalopay:so-du-sinh-loi"}.

// productIDPattern is the shape of a product ID: "provider:product".
var productIDPattern = regexp.MustCompile(`^[a-z0-9-]+:[a-z0-9-]+$`)

// AccountProduct links one account to a product.
type AccountProduct struct {
	Account string `json:"account"`
	Product string `json:"product"`
}

// AccountProductSettings are the saved links.
type AccountProductSettings struct {
	// Accounts are sorted by name ignoring case; names that match no
	// account and products the server no longer lists are reported by the
	// idle cash check, never fatal.
	Accounts []AccountProduct `json:"accounts"`
	// ETag fingerprints the saved value (also when unset).
	ETag string `json:"etag"`
}

// For returns the product linked to an account, names compared ignoring
// case and surrounding spaces.
func (s AccountProductSettings) For(account string) (string, bool) {
	for _, link := range s.Accounts {
		if sameAccountName(link.Account, account) {
			return link.Product, true
		}
	}
	return "", false
}

// ParseAccountProducts reads idleCash.accountProducts: blank is none,
// otherwise a JSON object of account name to product ID. Bad entries (no
// name, a product ID that is not "provider:product", a name repeated
// ignoring case) are returned as problems and skipped; a value that is not
// such an object is an error. The result is sorted by name ignoring case.
func ParseAccountProducts(raw string) ([]AccountProduct, []string, error) {
	links, problems := []AccountProduct{}, []string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return links, problems, nil
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil || parsed == nil {
		return links, problems, fmt.Errorf(`value is not a JSON object such as {"Zalopay":"zalopay:so-du-sinh-loi"}`)
	}
	names := make([]string, 0, len(parsed))
	for name := range parsed {
		names = append(names, name)
	}
	slices.SortFunc(names, compareAccountNames)
	for _, rawName := range names {
		name, product := strings.TrimSpace(rawName), strings.ToLower(strings.TrimSpace(parsed[rawName]))
		switch {
		case name == "":
			problems = append(problems, "has an entry without an account name")
		case !productIDPattern.MatchString(product):
			problems = append(problems, fmt.Sprintf("%q needs a product ID such as \"zalopay:so-du-sinh-loi\"", name))
		case slices.ContainsFunc(links, func(kept AccountProduct) bool { return sameAccountName(kept.Account, name) }):
			problems = append(problems, fmt.Sprintf("%q is set more than once (names ignore case); the first is used", name))
		default:
			links = append(links, AccountProduct{Account: name, Product: product})
		}
	}
	return links, problems, nil
}

// EncodeAccountProducts writes idleCash.accountProducts for column B: a
// JSON object sorted by name ignoring case, or "" for none.
func EncodeAccountProducts(links []AccountProduct) (string, error) {
	sorted := slices.Clone(links)
	slices.SortStableFunc(sorted, func(a, b AccountProduct) int { return compareAccountNames(a.Account, b.Account) })
	if len(sorted) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, link := range sorted {
		name, err := json.Marshal(link.Account)
		if err != nil {
			return "", fmt.Errorf("encode account products: %w", err)
		}
		product, err := json.Marshal(link.Product)
		if err != nil {
			return "", fmt.Errorf("encode account products: %w", err)
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(name)
		b.WriteByte(':')
		b.Write(product)
	}
	b.WriteByte('}')
	return b.String(), nil
}

// AccountProductInput links an account to a product, or unlinks it when
// Product is blank. ETag is the AccountProductSettings ETag read with the
// setting.
type AccountProductInput struct {
	Account string `json:"account"`
	Product string `json:"product"`
	ETag    string `json:"etag"`
}

// Removes reports whether the input unlinks the account.
func (in AccountProductInput) Removes() bool {
	return strings.TrimSpace(in.Product) == ""
}

// Validate checks the account, the ETag and the product ID's shape.
func (in AccountProductInput) Validate() error {
	var problems []string
	if strings.TrimSpace(in.Account) == "" {
		problems = append(problems, "account is required")
	}
	if strings.TrimSpace(in.ETag) == "" {
		problems = append(problems, "etag is required")
	}
	if !in.Removes() && !productIDPattern.MatchString(strings.ToLower(strings.TrimSpace(in.Product))) {
		problems = append(problems, `product must be a product ID such as "zalopay:so-du-sinh-loi" (blank unlinks)`)
	}
	return validationError(problems)
}

// ApplyAccountProduct returns the preferences after linking (or, for a
// blank product, unlinking) the account. A name matching an entry ignoring
// case replaces it. It fails with ErrConflict when the setting changed
// since the ETag was read. A row the change adds has Row 0.
func ApplyAccountProduct(current Preferences, input AccountProductInput) (Preferences, error) {
	if err := input.Validate(); err != nil {
		return Preferences{}, err
	}
	if strings.TrimSpace(input.ETag) != current.IdleCash.AccountProducts.ETag {
		return Preferences{}, fmt.Errorf("%w: the linked products changed since they were read; reload and try again", ErrConflict)
	}
	account := strings.TrimSpace(input.Account)
	links := slices.DeleteFunc(slices.Clone(current.IdleCash.AccountProducts.Accounts), func(link AccountProduct) bool { return sameAccountName(link.Account, account) })
	if !input.Removes() {
		links = append(links, AccountProduct{Account: account, Product: strings.ToLower(strings.TrimSpace(input.Product))})
	}
	slices.SortStableFunc(links, func(a, b AccountProduct) int { return compareAccountNames(a.Account, b.Account) })

	next := Preferences{TabExists: current.TabExists, Entries: slices.Clone(current.Entries), Issues: []string{}}
	index := slices.IndexFunc(next.Entries, func(entry PreferenceEntry) bool { return entry.Key == PrefIdleCashAccountProducts })
	if index >= 0 {
		next.Entries[index].Products = links
	} else {
		spec, _ := PreferenceSpecFor(PrefIdleCashAccountProducts)
		next.Entries = append(next.Entries, PreferenceEntry{Key: PrefIdleCashAccountProducts, Products: links, Note: spec.Note})
	}
	next.resolve()
	return next, nil
}
