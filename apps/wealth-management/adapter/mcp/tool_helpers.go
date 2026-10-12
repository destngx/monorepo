package mcp

import (
	"apps/wealth-management/domain"
	"bytes"
	"encoding/json"
	"fmt"
)

type validator interface{ Validate() error }

func decodeValidated(arguments json.RawMessage, target validator) error {
	if err := decodeArguments(arguments, target); err != nil {
		return err
	}
	return target.Validate()
}

// decodeArguments strictly decodes tool arguments; unknown fields and wrong
// types are validation errors, matching the advertised input schema.
func decodeArguments(arguments json.RawMessage, target any) error {
	if len(bytes.TrimSpace(arguments)) == 0 || bytes.Equal(bytes.TrimSpace(arguments), []byte("null")) {
		arguments = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: invalid arguments: %v", domain.ErrValidation, err)
	}
	return nil
}

var (
	stringSchema      = map[string]any{"type": "string"}
	numberSchema      = map[string]any{"type": "number", "minimum": 0}
	forceSchema       = map[string]any{"type": "boolean", "description": "Bypass the cache and read the sheet directly."}
	yearSchema        = map[string]any{"type": "integer", "minimum": 2000, "maximum": 2100}
	accountTypeSchema = map[string]any{"type": "string", "enum": []string{
		string(domain.AccountTypeActiveUse), string(domain.AccountTypeRarelyUse), string(domain.AccountTypeLongHolding),
		string(domain.AccountTypeDeprecated), string(domain.AccountTypeNegativeActive), string(domain.AccountTypeBank),
		string(domain.AccountTypeCrypto), string(domain.AccountTypeCash), string(domain.AccountTypeInvestment),
	}}
	currencySchema = map[string]any{"type": "string", "enum": []string{string(domain.CurrencyVND), string(domain.CurrencyUSD), string(domain.CurrencyUSDT)}}
)

func transactionSchema() map[string]any {
	return map[string]any{
		"accountName":     stringSchema,
		"date":            map[string]any{"type": "string", "description": "dd/mm/yyyy or yyyy-mm-dd"},
		"payee":           stringSchema,
		"category":        stringSchema,
		"tags":            map[string]any{"type": "array", "items": stringSchema},
		"cleared":         map[string]any{"type": "boolean"},
		"payment":         numberSchema,
		"deposit":         numberSchema,
		"memo":            stringSchema,
		"referenceNumber": stringSchema,
	}
}

func withProperties(base map[string]any, extra map[string]any) map[string]any {
	for key, value := range extra {
		base[key] = value
	}
	return base
}

func object(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func definition(name, description string, inputSchema map[string]any) domain.ToolDefinition {
	if inputSchema == nil {
		inputSchema = object(map[string]any{})
	}
	return domain.ToolDefinition{Name: name, Description: description, Backend: "mcp", InputSchema: inputSchema}
}

func toolContent(text string, isError bool) map[string]any {
	result := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
	if isError {
		result["isError"] = true
	}
	return result
}
