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

// objectQuery dan coverageQuery memetakan tabel sumber sebuah lini ke kueri objek dan
// kueri coverage-nya. Keduanya dipisah supaya pasangan untuk setiap lini terbaca
// berdampingan, bukan tersebar di dalam fungsi yang memakainya.
var objectQuery = map[registrasi.PolicySource]string{
	registrasi.SourcePerson:   "polis_objek_person",
	registrasi.SourceProperty: "polis_objek_property",
	registrasi.SourceCargo:    "polis_objek_cargo",
	registrasi.SourceAneka:    "polis_objek_aneka",
}

var coverageQuery = map[registrasi.PolicySource]string{
	registrasi.SourcePerson:   "polis_coverage_person",
	registrasi.SourceProperty: "polis_coverage_property",
	registrasi.SourceCargo:    "polis_coverage_cargo",
	registrasi.SourceAneka:    "polis_coverage_aneka",
}

// Items membaca objek, coverage, dan spreading polis pada versi snapshot klaim.
//
// Tiga kueri dijalankan, bukan satu per objek: objeknya, SELURUH coverage polis, dan
// SELURUH spreading polis. Penggabungannya terjadi di sini lewat INDEXOBJECT dan
// INDEXCOVERAGE.
//
// # Kenapa digabung di Go, bukan dengan JOIN
//
// Karena satu baris hasil JOIN tiga tabel mengulang kolom objek sebanyak jumlah baris
// spreading-nya, dan kolom nama objek pada lini Fire berukuran besar. Yang lebih
// menentukan: coverage TANPA spreading harus tetap muncul. Ia yang membuat gerbang
// validasi berkata "total spreading bukan 100 persen" alih-alih menghilangkan
// coverage-nya diam-diam. OUTER JOIN dapat melakukannya, tetapi tiga kueri datar jauh
// lebih mudah dibaca dan diperiksa satu per satu saat produksi bermasalah.
func (p *PolicyItems) Items(ctx context.Context, policy registrasi.Policy) ([]registrasi.SourceItem, error) {
	number := strings.TrimSpace(policy.Number)
	version := strings.TrimSpace(policy.ProdKe)
	if number == "" || version == "" {
		return nil, nil
	}

	source := registrasi.SourceOf(policy)

	item, key, err := p.objects(ctx, source, number, version)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, nil
	}

	spreading, err := p.spreading(ctx, number, version)
	if err != nil {
		return nil, err
	}
	coverage, err := p.coverages(ctx, source, number, version, spreading)
	if err != nil {
		return nil, err
	}
	for i := range item {
		item[i].Coverage = coverage[key[i]]
	}

	names, err := p.treatyNames(ctx)
	if err != nil {
		return nil, err
	}
	nameTreaties(item, names)
	return item, nil
}

// objects membaca baris objek polis dan mengembalikan kunci penggabungannya.
//
// Kunci dikembalikan terpisah karena lini Fire memakai OBJECTNO sebagai ID objek klaim
// sementara coverage-nya tergabung lewat INDEXOBJECT. Menyimpan kunci di dalam SourceItem
// akan membocorkan rincian penyimpanan ke tipe domain.
func (p *PolicyItems) objects(
	ctx context.Context,
	source registrasi.PolicySource,
	number, version string,
) ([]registrasi.SourceItem, []string, error) {
	rows, err := p.db.QueryContext(ctx, loadQuery(objectQuery[source]), number, version)
	if err != nil {
		return nil, nil, fmt.Errorf("registrasi/sqlstore: membaca objek polis (%s): %w", source, err)
	}
	defer func() { _ = rows.Close() }()

	var item []registrasi.SourceItem
	var key []string
	for rows.Next() {
		var index, id, name, location sql.NullString
		if err := rows.Scan(&index, &id, &name, &location); err != nil {
			return nil, nil, fmt.Errorf("registrasi/sqlstore: membaca baris objek polis: %w", err)
		}
		item = append(item, registrasi.SourceItem{ID: text(id), Name: text(name), Location: text(location)})
		key = append(key, joinKey(index))
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("registrasi/sqlstore: menelusuri objek polis: %w", err)
	}
	return item, key, nil
}

// coverages membaca seluruh coverage polis, dikelompokkan menurut INDEXOBJECT.
//
// Urutan baris di dalam satu objek adalah urutan kueri, sehingga coverage tampil pada
// urutan yang sama dengan dokumen lama.
func (p *PolicyItems) coverages(
	ctx context.Context,
	source registrasi.PolicySource,
	number, version string,
	spreading spreadingIndex,
) (map[string][]registrasi.SourceCoverage, error) {
	rows, err := p.db.QueryContext(ctx, loadQuery(coverageQuery[source]), number, version)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca coverage polis (%s): %w", source, err)
	}
	defer func() { _ = rows.Close() }()

	result := map[string][]registrasi.SourceCoverage{}
	for rows.Next() {
		var object, index, code, name, tsi, sumTSI, sublimit, deleted sql.NullString
		if err := rows.Scan(&object, &index, &code, &name, &tsi, &sumTSI, &sublimit, &deleted); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris coverage polis: %w", err)
		}
		objectKey := joinKey(object)
		result[objectKey] = append(result[objectKey], registrasi.SourceCoverage{
			Code:        text(code),
			Name:        text(name),
			TSI:         parseMoney(tsi.String),
			SumTSI:      parseMoney(sumTSI.String),
			TSISublimit: parseMoney(sublimit.String),
			Deleted:     isFlagged(deleted),
			Spreading:   spreading.of(objectKey, joinKey(index), text(code)),
		})
	}
	return result, rows.Err()
}

// spreadingIndex menyimpan baris spreading satu polis dengan dua kunci.
//
// byIndex adalah kunci utama (INDEXOBJECT + INDEXCOVERAGE); byCode adalah cadangan
// (INDEXOBJECT + kode coverage) yang dipakai hanya bila kunci utama tidak memberi satu
// baris pun. Alasannya di komentar kueri polis_spreading.
type spreadingIndex struct {
	byIndex map[string][]registrasi.SourceSpreading
	byCode  map[string][]registrasi.SourceSpreading
}

func (s spreadingIndex) of(object, index, code string) []registrasi.SourceSpreading {
	if row := s.byIndex[spreadingKey(object, index)]; len(row) > 0 {
		return row
	}
	if code == "" {
		return nil
	}
	return s.byCode[spreadingKey(object, code)]
}

// spreadingKey menyatukan dua bagian kunci dengan pemisah yang tidak mungkin muncul di
// dalam nilai kolomnya. Perangkaian polos akan membuat pasangan ("1", "23") dan
// ("12", "3") bertabrakan.
func spreadingKey(object, second string) string { return object + "\x00" + second }

// spreading membaca seluruh baris spreading satu polis.
func (p *PolicyItems) spreading(ctx context.Context, number, version string) (spreadingIndex, error) {
	result := spreadingIndex{
		byIndex: map[string][]registrasi.SourceSpreading{},
		byCode:  map[string][]registrasi.SourceSpreading{},
	}

	rows, err := p.db.QueryContext(ctx, loadQuery("polis_spreading"), number, version)
	if err != nil {
		return result, fmt.Errorf("registrasi/sqlstore: membaca spreading polis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var object, index, code, treaty, treatyName, share, deleted sql.NullString
		if err := rows.Scan(&object, &index, &code, &treaty, &treatyName, &share, &deleted); err != nil {
			return result, fmt.Errorf("registrasi/sqlstore: membaca baris spreading polis: %w", err)
		}
		value, _ := parsePercent(share.String)
		row := registrasi.SourceSpreading{
			TreatyType: text(treaty),
			TreatyName: text(treatyName),
			Share:      value,
			Deleted:    isFlagged(deleted),
		}
		objectKey := joinKey(object)

		indexKey := spreadingKey(objectKey, joinKey(index))
		result.byIndex[indexKey] = append(result.byIndex[indexKey], row)

		if c := text(code); c != "" {
			codeKey := spreadingKey(objectKey, c)
			result.byCode[codeKey] = append(result.byCode[codeKey], row)
		}
	}
	return result, rows.Err()
}

// joinKey menormalkan nilai kunci penggabungan.
//
// INDEXOBJECT dan INDEXCOVERAGE bertipe berbeda antartabel — NUMBER pada sebagian,
// VARCHAR2 pada yang lain — sehingga nilai yang sama dapat datang sebagai "1" dan "01".
// Nol di depan karena itu dibuang pada nilai yang seluruhnya angka. Tanpa ini, spreading
// sebuah coverage hilang tanpa galat apa pun, dan gejalanya muncul jauh dari sebabnya:
// gerbang validasi menolak klaim karena total spreading tidak genap seratus.
//
// Nilai yang BUKAN angka dibiarkan apa adanya; memangkas nol di depan teks akan mengubah
// kunci yang memang berbentuk kode.
func joinKey(value sql.NullString) string {
	raw := strings.TrimSpace(value.String)
	if raw == "" {
		return ""
	}
	for _, r := range raw {
		if r < '0' || r > '9' {
			return raw
		}
	}
	if trimmed := strings.TrimLeft(raw, "0"); trimmed != "" {
		return trimmed
	}
	return "0"
}

// isFlagged membaca satu kolom bendera.
//
// Kolomnya bertipe teks pada seluruh tabel coverage dan spreading, tetapi isinya ditulis
// bermacam-macam oleh sistem polis: "1", "0", "Y", kosong, maupun NULL. Yang diperlakukan
// sebagai menyala hanya "1" dan "Y" — sama seperti dokumen lama yang menguji
// FlagDelete sama dengan "1".
func isFlagged(value sql.NullString) bool {
	switch strings.ToUpper(strings.TrimSpace(value.String)) {
	case "1", "Y":
		return true
	default:
		return false
	}
}

// treatyNames membaca NOTE master REINSURANCETYPE per ID — nama yang ditampilkan dropdown
// Nama Treaty Pega (BrowseReinsuranceType_RD, prompt .Note, nilai .ID). Dokumen polis hanya
// menyimpan TreatyType, sehingga tanpa ini kolom Nama Treaty klaim baru selalu kosong.
func (p *PolicyItems) treatyNames(ctx context.Context) (map[string]string, error) {
	rows, err := p.db.QueryContext(ctx, loadQuery("jenis_treaty_nama"))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca nama treaty: %w", err)
	}
	defer func() { _ = rows.Close() }()
	names := map[string]string{}
	for rows.Next() {
		var id, note sql.NullString
		if err := rows.Scan(&id, &note); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris nama treaty: %w", err)
		}
		names[text(id)] = text(note)
	}
	return names, rows.Err()
}

// nameTreaties mengisi TreatyName setiap baris spreading dari peta nama treaty.
func nameTreaties(items []registrasi.SourceItem, names map[string]string) {
	for i := range items {
		for j := range items[i].Coverage {
			for k := range items[i].Coverage[j].Spreading {
				s := &items[i].Coverage[j].Spreading[k]
				if s.TreatyName == "" {
					s.TreatyName = names[strings.TrimSpace(s.TreatyType)]
				}
			}
		}
	}
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

// ItemOptions membaca pilihan Objek item estimasi lini Fire — T_PROPERTYITEMLIST objek
// polis; lini lain mengembalikan daftar kosong (lihat registrasi.ItemFromPropertyList).
// Nama yang sama hanya ditampilkan sekali.
func (p *PolicyItems) ItemOptions(ctx context.Context, policy registrasi.Policy, objectID string) ([]registrasi.ItemOption, error) {
	if !registrasi.ItemFromPropertyList(policy) {
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
		var itemType, group, tsi sql.NullString
		if err := rows.Scan(&itemType, &group, &tsi); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris item properti: %w", err)
		}
		name := text(itemType)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, registrasi.ItemOption{Name: name, Group: text(group), TSI: parseMoney(text(tsi))})
	}
	return result, rows.Err()
}

// TravelBenefits membaca manfaat sebuah plan Travel — pilihan Objek item estimasi lini
// Travel (`SearchCoverageTravel_RD`).
func (p *PolicyItems) TravelBenefits(ctx context.Context, plan string) ([]registrasi.ItemOption, error) {
	plan = strings.TrimSpace(plan)
	if plan == "" {
		return nil, nil
	}
	rows, err := p.db.QueryContext(ctx, loadQuery("item_pilihan_travel"), plan)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca manfaat plan Travel: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []registrasi.ItemOption
	seen := map[string]bool{}
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris manfaat Travel: %w", err)
		}
		n := text(name)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		result = append(result, registrasi.ItemOption{ID: text(id), Name: n})
	}
	return result, rows.Err()
}
