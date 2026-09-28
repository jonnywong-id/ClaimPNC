package sqlstore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"claim-pnc/internal/registrasi"
)

// PolicyItems membaca objek polis dari tabel sumber lininya.
type PolicyItems struct {
	db *sql.DB
}

// NewPolicyItems membentuk pembaca objek polis.
func NewPolicyItems(db *sql.DB) *PolicyItems { return &PolicyItems{db: db} }

// Items membaca objek dan coverage polis pada versi snapshot klaim.
func (p *PolicyItems) Items(ctx context.Context, policy registrasi.Policy) ([]registrasi.SourceItem, error) {
	number := strings.TrimSpace(policy.Number)
	version := strings.TrimSpace(policy.ProdKe)
	if number == "" || version == "" {
		return nil, nil
	}

	source := registrasi.SourceOf(policy)
	name := map[registrasi.PolicySource]string{
		registrasi.SourcePerson:   "polis_objek_person",
		registrasi.SourceProperty: "polis_objek_property",
		registrasi.SourceCargo:    "polis_objek_cargo",
		registrasi.SourceAneka:    "polis_objek_aneka",
	}[source]

	rows, err := p.db.QueryContext(ctx, loadQuery(name), number, version)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca objek polis (%s): %w", source, err)
	}
	defer func() { _ = rows.Close() }()

	var result []registrasi.SourceItem
	var index []string // INDEXOBJECT tiap objek property, untuk kueri coverage-nya
	for rows.Next() {
		var a, b, c, d sql.NullString
		if err := rows.Scan(&a, &b, &c, &d); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris objek polis: %w", err)
		}
		if source == registrasi.SourceProperty {
			// INDEXOBJECT, OBJECTNO, OBJECTNAME, ASMADDRESS.
			result = append(result, registrasi.SourceItem{ID: text(b), Name: text(c), Location: text(d)})
			index = append(index, text(a))
			continue
		}
		coverage, err := parseCoverages(d.String)
		if err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: dokumen coverage objek %s: %w", text(a), err)
		}
		result = append(result, registrasi.SourceItem{ID: text(a), Name: text(b), Location: text(c), Coverage: coverage})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: menelusuri objek polis: %w", err)
	}
	_ = rows.Close()

	for i := range result {
		if source != registrasi.SourceProperty {
			break
		}
		coverage, err := p.propertyCoverages(ctx, number, version, index[i])
		if err != nil {
			return nil, err
		}
		result[i].Coverage = coverage
	}
	return result, nil
}

// propertyCoverages menggabungkan coverage seluruh baris satu INDEXOBJECT.
//
// Satu objek Fire dapat tercatat di lebih dari satu baris (satu per item properti), dan
// activity lama mengadopsi dokumen tiap baris ke halaman yang sama. Coverage yang sama
// karena itu dapat muncul lebih dari sekali; yang dipakai adalah kemunculan pertama.
func (p *PolicyItems) propertyCoverages(ctx context.Context, number, version, indexObject string) ([]registrasi.SourceCoverage, error) {
	rows, err := p.db.QueryContext(ctx, loadQuery("polis_coverage_property"), number, version, indexObject)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca coverage objek properti %s: %w", indexObject, err)
	}
	defer func() { _ = rows.Close() }()

	var result []registrasi.SourceCoverage
	seen := map[string]bool{}
	for rows.Next() {
		var doc sql.NullString
		if err := rows.Scan(&doc); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca dokumen coverage properti: %w", err)
		}
		list, err := parseCoverages(doc.String)
		if err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: dokumen coverage properti %s: %w", indexObject, err)
		}
		for _, c := range list {
			if seen[c.Code] {
				continue
			}
			seen[c.Code] = true
			result = append(result, c)
		}
	}
	return result, rows.Err()
}

// coverageDoc adalah bagian dokumen coverage polis yang dipakai.
//
// PA dan Travel menyimpan daftarnya di ASMCoverage; lini lain di CoverageList.
type coverageDoc struct {
	CoverageList []coverageJSON `json:"CoverageList"`
	ASMCoverage  []coverageJSON `json:"ASMCoverage"`
}

type coverageJSON struct {
	Coverage      jsonText        `json:"Coverage"`
	CoverageNote  jsonText        `json:"CoverageNote"`
	TSI           jsonText        `json:"TSI"`
	SumTSI        jsonText        `json:"SumTSI"`
	TSISublimit   jsonText        `json:"TSISublimit"`
	FlagDelete    jsonText        `json:"FlagDelete"`
	SpreadingList []spreadingJSON `json:"SpreadingList"`
}

type spreadingJSON struct {
	TreatyType      jsonText `json:"TreatyType"`
	SharePercentage jsonText `json:"SharePercentage"`
	FlagDelete      jsonText `json:"FlagDelete"`
}

// jsonText menerima nilai JSON berupa teks maupun angka. Dokumen polis mencampur
// keduanya ("100.0000" dan 100 untuk kolom yang sama).
type jsonText string

func (t *jsonText) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*t = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = jsonText(strings.TrimSpace(s))
		return nil
	}
	*t = jsonText(strings.TrimSpace(string(data)))
	return nil
}

// parseCoverages mengurai dokumen coverage. Dokumen kosong berarti tanpa coverage.
func parseCoverages(document string) ([]registrasi.SourceCoverage, error) {
	document = strings.TrimSpace(document)
	if document == "" {
		return nil, nil
	}
	var doc coverageDoc
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		return nil, err
	}
	list := doc.CoverageList
	if len(list) == 0 {
		list = doc.ASMCoverage
	}

	result := make([]registrasi.SourceCoverage, 0, len(list))
	for _, c := range list {
		coverage := registrasi.SourceCoverage{
			Code:        string(c.Coverage),
			Name:        string(c.CoverageNote),
			TSI:         parseMoney(string(c.TSI)),
			SumTSI:      parseMoney(string(c.SumTSI)),
			TSISublimit: parseMoney(string(c.TSISublimit)),
			Deleted:     string(c.FlagDelete) == "1",
		}
		for _, s := range c.SpreadingList {
			share, _ := parsePercent(string(s.SharePercentage))
			coverage.Spreading = append(coverage.Spreading, registrasi.SourceSpreading{
				TreatyType: string(s.TreatyType),
				Share:      share,
				Deleted:    string(s.FlagDelete) == "1",
			})
		}
		result = append(result, coverage)
	}
	return result, nil
}

// parseMoney membaca rupiah bertipe teks ("97200000.0000") menjadi sen, dibulatkan ke
// sen terdekat. Teks yang bukan angka dianggap nol.
func parseMoney(raw string) registrasi.Money {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	v, ok := new(big.Rat).SetString(raw)
	if !ok {
		return 0
	}
	v.Mul(v, big.NewRat(100, 1))
	half := big.NewRat(1, 2)
	if v.Sign() < 0 {
		half.Neg(half)
	}
	v.Add(v, half)
	return registrasi.Money(new(big.Int).Quo(v.Num(), v.Denom()).Int64())
}

// text memangkas nilai kolom teks yang boleh kosong.
func text(value sql.NullString) string { return strings.TrimSpace(value.String) }

// ItemOptions membaca pilihan Objek item estimasi. Hanya lini Fire yang punya sumbernya —
// PropertyItemList objek polis; lini lain mengembalikan daftar kosong.
func (p *PolicyItems) ItemOptions(ctx context.Context, policy registrasi.Policy, objectID string) ([]registrasi.ItemOption, error) {
	if registrasi.SourceOf(policy) != registrasi.SourceProperty {
		return nil, nil
	}
	rows, err := p.db.QueryContext(ctx, loadQuery("item_pilihan_properti"),
		strings.TrimSpace(policy.Number), strings.TrimSpace(policy.ProdKe), strings.TrimSpace(objectID))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca item properti polis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []registrasi.ItemOption
	seen := map[string]bool{}
	for rows.Next() {
		var doc sql.NullString
		if err := rows.Scan(&doc); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca dokumen item properti: %w", err)
		}
		if strings.TrimSpace(doc.String) == "" {
			continue
		}
		var parsed struct {
			PropertyItemList []struct {
				ItemType          jsonText `json:"ItemType"`
				PropertyItemGroup jsonText `json:"PropertyItemGroup"`
				TSIObjectItem     jsonText `json:"TSIObjectItem"`
				FlagDelete        jsonText `json:"FlagDelete"`
			} `json:"PropertyItemList"`
		}
		if err := json.Unmarshal([]byte(doc.String), &parsed); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: dokumen item properti: %w", err)
		}
		for _, it := range parsed.PropertyItemList {
			name := strings.TrimSpace(string(it.ItemType))
			if name == "" || string(it.FlagDelete) == "1" || seen[name] {
				continue
			}
			seen[name] = true
			result = append(result, registrasi.ItemOption{
				Name: name, Group: string(it.PropertyItemGroup), TSI: parseMoney(string(it.TSIObjectItem)),
			})
		}
	}
	return result, rows.Err()
}
