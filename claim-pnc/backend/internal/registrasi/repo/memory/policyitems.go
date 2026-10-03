package memory

import (
	"context"

	"claim-pnc/internal/registrasi"
)

// PolicyItems adalah pengganti tabel objek polis di memori.
type PolicyItems struct {
	byPolicy map[string][]registrasi.SourceItem
}

// NewPolicyItems membentuk sumber objek polis berisi data yang diberikan, dikunci nomor
// polis. Peta kosong berarti setiap polis tanpa objek.
func NewPolicyItems(byPolicy map[string][]registrasi.SourceItem) *PolicyItems {
	copied := map[string][]registrasi.SourceItem{}
	for number, items := range byPolicy {
		copied[number] = append([]registrasi.SourceItem(nil), items...)
	}
	return &PolicyItems{byPolicy: copied}
}

// Items mengembalikan objek polis contoh.
func (p *PolicyItems) Items(_ context.Context, policy registrasi.Policy) ([]registrasi.SourceItem, error) {
	return append([]registrasi.SourceItem(nil), p.byPolicy[policy.Number]...), nil
}

// SamplePolicyItems adalah objek contoh untuk polis contoh NewPolicyStore: satu objek,
// satu coverage, dan spreading OR 100%. Nilainya karangan.
func SamplePolicyItems() map[string][]registrasi.SourceItem {
	one := func(id, name, code, coverageName string, tsi int64) []registrasi.SourceItem {
		return []registrasi.SourceItem{{
			ID: id, Name: name, Location: "Lokasi contoh",
			Coverage: []registrasi.SourceCoverage{{
				Code: code, Name: coverageName, TSI: registrasi.Rupiah(tsi),
				Spreading: []registrasi.SourceSpreading{{TreatyType: "10001", TreatyName: "OR", Share: registrasi.PercentFull}},
			}},
		}}
	}
	return map[string][]registrasi.SourceItem{
		"POL-FIRE-0001": one("1", "Gedung contoh", "100819", "FLEXAS", 1_000_000_000),
		"POL-PA-0002":   one("1", "PESERTA CONTOH", "10003", "Resiko A", 50_000_000),
		"POL-TRV-0003":  one("1", "PESERTA CONTOH", "10001", "Plan contoh", 10_000_000),
		"POL-MAR-0005":  one("1", "Barang contoh", "11022", "INSTITUTE CARGO CLAUSE C", 250_000_000),
	}
}

// ItemOptions mengembalikan pilihan Objek contoh untuk polis Fire contoh.
func (p *PolicyItems) ItemOptions(_ context.Context, policy registrasi.Policy, _ string) ([]registrasi.ItemOption, error) {
	if registrasi.SourceOf(policy) != registrasi.SourceProperty {
		return nil, nil
	}
	return []registrasi.ItemOption{
		{Name: "BUILDING AND CONTENTS", Group: "MATERIAL DAMAGE", TSI: registrasi.Rupiah(1_000_000_000)},
		{Name: "BUSINESS INTERRUPTION", Group: "CONSEQUENTIAL LOSS", TSI: registrasi.Rupiah(500_000_000)},
	}, nil
}
