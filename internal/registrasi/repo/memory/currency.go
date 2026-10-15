package memory

import (
	"context"

	"claim-pnc/internal/registrasi"
)

// CurrencyDirectory adalah master mata uang contoh.
type CurrencyDirectory struct{}

// Currencies mengembalikan tiga mata uang contoh dengan kode POOLDATA.CURRENCY.
func (CurrencyDirectory) Currencies(context.Context) ([]registrasi.CurrencyOption, error) {
	return []registrasi.CurrencyOption{
		{ID: "10026", Name: "IDR"},
		{ID: "10001", Name: "USD"},
		{ID: "10002", Name: "SGD"},
	}, nil
}
