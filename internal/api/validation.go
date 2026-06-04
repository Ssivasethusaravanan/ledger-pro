// Package api implements the HTTP transport layer for LedgerPro.
// This file provides fast, zero-reflection request validation with ISO 4217
// currency validation, account type validation, and posting rule enforcement.
package api

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"ledger_pro/internal/db"
	"ledger_pro/internal/service"
)

// ---------------------------------------------------------------------------
// Validation Result
// ---------------------------------------------------------------------------

// ValidationResult collects validation errors as InvalidParam entries.
type ValidationResult struct {
	Errors []InvalidParam
}

// AddError appends a validation error.
func (v *ValidationResult) AddError(name, reason string) {
	v.Errors = append(v.Errors, InvalidParam{Name: name, Reason: reason})
}

// HasErrors returns true if any validation errors were collected.
func (v *ValidationResult) HasErrors() bool {
	return len(v.Errors) > 0
}

// ToProblem converts the validation result into an RFC 7807 ProblemDetail.
func (v *ValidationResult) ToProblem() *ProblemDetail {
	return ValidationError(
		fmt.Sprintf("%d validation error(s) found", len(v.Errors)),
		v.Errors...,
	)
}

// ---------------------------------------------------------------------------
// ISO 4217 Currency Whitelist
// ---------------------------------------------------------------------------

// validCurrencies is a set of ISO 4217 active currency codes.
// Kept as a map for O(1) lookup performance.
var validCurrencies = map[string]bool{
	"AED": true, "AFN": true, "ALL": true, "AMD": true, "ANG": true,
	"AOA": true, "ARS": true, "AUD": true, "AWG": true, "AZN": true,
	"BAM": true, "BBD": true, "BDT": true, "BGN": true, "BHD": true,
	"BIF": true, "BMD": true, "BND": true, "BOB": true, "BRL": true,
	"BSD": true, "BTN": true, "BWP": true, "BYN": true, "BZD": true,
	"CAD": true, "CDF": true, "CHF": true, "CLP": true, "CNY": true,
	"COP": true, "CRC": true, "CUP": true, "CVE": true, "CZK": true,
	"DJF": true, "DKK": true, "DOP": true, "DZD": true, "EGP": true,
	"ERN": true, "ETB": true, "EUR": true, "FJD": true, "FKP": true,
	"GBP": true, "GEL": true, "GHS": true, "GIP": true, "GMD": true,
	"GNF": true, "GTQ": true, "GYD": true, "HKD": true, "HNL": true,
	"HRK": true, "HTG": true, "HUF": true, "IDR": true, "ILS": true,
	"INR": true, "IQD": true, "IRR": true, "ISK": true, "JMD": true,
	"JOD": true, "JPY": true, "KES": true, "KGS": true, "KHR": true,
	"KMF": true, "KPW": true, "KRW": true, "KWD": true, "KYD": true,
	"KZT": true, "LAK": true, "LBP": true, "LKR": true, "LRD": true,
	"LSL": true, "LYD": true, "MAD": true, "MDL": true, "MGA": true,
	"MKD": true, "MMK": true, "MNT": true, "MOP": true, "MRU": true,
	"MUR": true, "MVR": true, "MWK": true, "MXN": true, "MYR": true,
	"MZN": true, "NAD": true, "NGN": true, "NIO": true, "NOK": true,
	"NPR": true, "NZD": true, "OMR": true, "PAB": true, "PEN": true,
	"PGK": true, "PHP": true, "PKR": true, "PLN": true, "PYG": true,
	"QAR": true, "RON": true, "RSD": true, "RUB": true, "RWF": true,
	"SAR": true, "SBD": true, "SCR": true, "SDG": true, "SEK": true,
	"SGD": true, "SHP": true, "SLE": true, "SOS": true, "SRD": true,
	"SSP": true, "STN": true, "SYP": true, "SZL": true, "THB": true,
	"TJS": true, "TMT": true, "TND": true, "TOP": true, "TRY": true,
	"TTD": true, "TWD": true, "TZS": true, "UAH": true, "UGX": true,
	"USD": true, "UYU": true, "UZS": true, "VES": true, "VND": true,
	"VUV": true, "WST": true, "XAF": true, "XCD": true, "XOF": true,
	"XPF": true, "YER": true, "ZAR": true, "ZMW": true, "ZWL": true,
}

// IsValidCurrency checks if a currency code is a valid ISO 4217 code.
func IsValidCurrency(code string) bool {
	return validCurrencies[strings.ToUpper(code)]
}

// ---------------------------------------------------------------------------
// Account Type Validation
// ---------------------------------------------------------------------------

// validAccountTypes is a set of valid account type enums.
var validAccountTypes = map[db.AccountType]bool{
	db.AccountTypeAsset:     true,
	db.AccountTypeLiability: true,
	db.AccountTypeEquity:    true,
	db.AccountTypeRevenue:   true,
	db.AccountTypeExpense:   true,
}

// IsValidAccountType checks if an account type is valid.
func IsValidAccountType(at db.AccountType) bool {
	return validAccountTypes[at]
}

// ---------------------------------------------------------------------------
// Request Validators
// ---------------------------------------------------------------------------

const (
	maxAccountNameLen     = 255
	minAccountNameLen     = 1
	maxDescriptionLen     = 1000
	maxIdempotencyKeyLen  = 255
	minIdempotencyKeyLen  = 1
	maxPostingsPerTx      = 100
	minPostingsPerTx      = 2
)

// ValidateCreateAccountRequest validates a CreateAccountRequest.
func ValidateCreateAccountRequest(req service.CreateAccountRequest) *ValidationResult {
	v := &ValidationResult{}

	// Account name
	nameLen := utf8.RuneCountInString(req.AccountName)
	if nameLen < minAccountNameLen {
		v.AddError("account_name", "account_name is required")
	} else if nameLen > maxAccountNameLen {
		v.AddError("account_name", fmt.Sprintf("account_name must be at most %d characters", maxAccountNameLen))
	}

	// Account type
	if req.AccountType == "" {
		v.AddError("account_type", "account_type is required (asset, liability, equity, revenue, expense)")
	} else if !IsValidAccountType(req.AccountType) {
		v.AddError("account_type", fmt.Sprintf("invalid account_type '%s' — must be one of: asset, liability, equity, revenue, expense", req.AccountType))
	}

	// Currency
	if req.Currency != "" && !IsValidCurrency(req.Currency) {
		v.AddError("currency", fmt.Sprintf("invalid ISO 4217 currency code '%s'", req.Currency))
	}

	return v
}

// ValidateCreateTransactionRequest validates a CreateTransactionRequest.
func ValidateCreateTransactionRequest(req service.CreateTransactionRequest) *ValidationResult {
	v := &ValidationResult{}

	// Idempotency key
	keyLen := utf8.RuneCountInString(req.IdempotencyKey)
	if keyLen < minIdempotencyKeyLen {
		v.AddError("idempotency_key", "Idempotency-Key header is required")
	} else if keyLen > maxIdempotencyKeyLen {
		v.AddError("idempotency_key", fmt.Sprintf("Idempotency-Key must be at most %d characters", maxIdempotencyKeyLen))
	}

	// Description
	if utf8.RuneCountInString(req.Description) > maxDescriptionLen {
		v.AddError("description", fmt.Sprintf("description must be at most %d characters", maxDescriptionLen))
	}

	// Postings count
	postingCount := len(req.Postings)
	if postingCount < minPostingsPerTx {
		v.AddError("postings", fmt.Sprintf("transaction must have at least %d postings (one debit, one credit)", minPostingsPerTx))
	} else if postingCount > maxPostingsPerTx {
		v.AddError("postings", fmt.Sprintf("transaction must have at most %d postings", maxPostingsPerTx))
	}

	// Individual posting validation
	var debitSum, creditSum int64
	for i, p := range req.Postings {
		field := fmt.Sprintf("postings[%d]", i)

		if p.Amount <= 0 {
			v.AddError(field+".amount", "posting amount must be a positive integer (in smallest currency unit)")
		}

		if p.AccountID <= 0 {
			v.AddError(field+".account_id", "account_id must be a positive integer")
		}

		switch p.Direction {
		case db.PostingDirectionDebit:
			debitSum += p.Amount
		case db.PostingDirectionCredit:
			creditSum += p.Amount
		default:
			v.AddError(field+".direction", fmt.Sprintf("invalid direction '%s' — must be 'debit' or 'credit'", p.Direction))
		}
	}

	// Balance check — debits must equal credits
	if postingCount >= minPostingsPerTx && debitSum != creditSum {
		v.AddError("postings", fmt.Sprintf(
			"debit and credit legs must balance: total debits=%d, total credits=%d (difference=%d)",
			debitSum, creditSum, debitSum-creditSum,
		))
	}

	return v
}
