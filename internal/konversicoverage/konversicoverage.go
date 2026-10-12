// Package konversicoverage melayani menu Konversi Coverage (`MENU_ID 87`).
//
// # Apa yang dikerjakan
//
// Coverage dan spreading polis dulu tersimpan sebagai dokumen JSON di kolom BLOB tabel
// objeknya sendiri. Sejak 2026-10-06 registrasi klaim membacanya dari tabel relasional
// (lihat `internal/registrasi/repo/sqlstore/policyitems.sql`), sehingga basis data TEST
// harus punya isi tabel relasional itu untuk polis yang dipakai menguji.
//
// Modul ini membaca dokumen JSON dari basis data LIVE untuk sejumlah nomor polis, lalu
// menulis ke basis data TEST:
//
//	sumber objek (LIVE)      kolom BLOB      coverage (TEST)          spreading (TEST)
//	-----------------------------------------------------------------------------------
//	T_ANEKALIST              COVERAGELIST    T_COVERAGELIST_ANEKA     T_SPREADINGLIST
//	T_CARGOLIST              COVERAGEDATA    T_COVERAGELIST_CARGO     T_SPREADINGLIST
//	T_PROPERTYLIST           COVERAGELIST    T_COVERAGELIST_FIRE      T_SPREADINGLIST
//	T_PERSONLIST             COVERAGEDATA    T_COVERAGELIST_PERSON    T_SPREADINGLIST
//
// sekaligus menyalin kolom BLOB-nya apa adanya ke baris objek yang sama di TEST.
//
// # LIVE hanya dibaca
//
// Seluruh pembacaan LIVE berjalan di dalam transaksi `SET TRANSACTION READ ONLY`, dan
// tidak ada satu pun pernyataan tulis yang diarahkan ke koneksi LIVE. Modul menolak
// berjalan bila kedua koneksi menunjuk basis data dan pengguna yang sama.
//
// # Pemetaan kolom tidak ditulis di kode
//
// Ketiga tabel coverage dan T_SPREADINGLIST masing-masing punya puluhan kolom, dan DDL
// lengkapnya belum seluruhnya ada di repo. Pemetaannya karena itu DIBACA dari kamus data
// basis data TEST (`ALL_TAB_COLUMNS`): setiap kunci JSON dicocokkan ke kolom bernama sama
// tanpa memandang huruf besar-kecil (`SumTSI` → `SUMTSI`). Kunci yang tidak punya kolom
// dilaporkan, bukan dibuang diam-diam.
package konversicoverage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Line adalah satu lini bisnis beserta keempat tabelnya.
type Line struct {
	Key string // aneka | cargo | property | person

	ObjectTable   string // tabel objek, LIVE dan TEST
	BlobColumn    string // kolom BLOB berisi dokumen coverage
	CoverageTable string // tabel coverage tujuan di TEST
}

// SpreadingTable adalah tabel spreading tujuan, sama untuk keempat lini.
const SpreadingTable = "T_SPREADINGLIST"

// Lines adalah keempat lini yang dikonversi, dalam urutan pemrosesan.
var Lines = []Line{
	{Key: "aneka", ObjectTable: "T_ANEKALIST", BlobColumn: "COVERAGELIST", CoverageTable: "T_COVERAGELIST_ANEKA"},
	{Key: "cargo", ObjectTable: "T_CARGOLIST", BlobColumn: "COVERAGEDATA", CoverageTable: "T_COVERAGELIST_CARGO"},
	{Key: "property", ObjectTable: "T_PROPERTYLIST", BlobColumn: "COVERAGELIST", CoverageTable: "T_COVERAGELIST_FIRE"},
	{Key: "person", ObjectTable: "T_PERSONLIST", BlobColumn: "COVERAGEDATA", CoverageTable: "T_COVERAGELIST_PERSON"},
}

// ObjectRow adalah satu baris objek polis di LIVE beserta dokumen coverage-nya.
type ObjectRow struct {
	IDPega        string
	ApplicationID string // kosong bila tabel objeknya tidak punya kolom APPLICATIONID
	PolicyNo      string
	ProdKe        string
	IndexObject   string
	// ListIndex adalah INDEXTANEKALIST baris objek — kosong bila tabel objeknya tidak
	// punya kolom itu. Hanya lini Aneka yang memakainya; lihat applyKeys.
	ListIndex string
	Document  []byte // isi BLOB apa adanya; nil bila kolomnya NULL
}

// Row adalah satu baris tujuan: nama kolom (huruf besar) → nilai JSON.
//
// Nilai berupa string, json.Number, bool, atau json.RawMessage untuk objek/larik
// bersarang. Pengubahan ke tipe kolom dilakukan belakangan, setelah tipe kolom tujuan
// diketahui — lihat Coerce.
type Row map[string]any

// Converted adalah hasil konversi satu dokumen.
type Converted struct {
	Coverage  []Row
	Spreading []Row
}

// Kunci akar dokumen yang berisi larik coverage. `CoverageList` dipakai Fire, Aneka, dan
// Cargo; `ASMCoverage` dipakai Person (INSERTT_PERSONLIST.prc membungkusnya dengan
// `{"ASMCoverage": [...]}`).
var coverageRootKeys = []string{"COVERAGELIST", "ASMCOVERAGE", "COVERAGEDATA", "COVERAGE"}

// Kunci larik spreading di dalam satu coverage.
var spreadingKeys = []string{"SPREADINGLIST", "ASMSPREADING", "SPREADING"}

// ErrUnknownShape berarti dokumen bukan JSON coverage yang dikenali.
var ErrUnknownShape = errors.New("dokumen tidak berisi larik coverage yang dikenali")

// Convert mengurai satu dokumen coverage menjadi baris coverage dan spreading.
//
// Kolom kunci baris tujuan selalu diambil dari baris objeknya, bukan dari dokumen:
// IDPEGA, NOPOLIS, PRODKE, INDEXOBJECT — dan APPLICATIONID bila objeknya membawanya.
// Registrasi menggabungkan coverage ke objeknya lewat kunci itu, sehingga isi dokumen
// yang berbeda (atau kosong) tidak boleh memutus penggabungannya.
//
// INDEXCOVERAGE dan INDEXSPREADING diambil dari dokumen bila terisi, selain itu dari
// urutan kemunculan mulai 1 — urutan yang sama dengan larik di dokumennya.
func Convert(object ObjectRow, line Line) (Converted, error) {
	if len(bytes.TrimSpace(object.Document)) == 0 {
		return Converted{}, nil
	}
	coverages, err := coverageArray(object.Document)
	if err != nil {
		return Converted{}, err
	}

	var result Converted
	for i, raw := range coverages {
		fields, nested, err := flatten(raw)
		if err != nil {
			return Converted{}, fmt.Errorf("coverage ke-%d: %w", i+1, err)
		}
		row := Row(fields)
		applyKeys(row, object)
		// T_COVERAGELIST_ANEKA TIDAK punya kolom INDEXOBJECT, sehingga kunci objek yang
		// ditulis applyKeys terbuang diam-diam saat disisipkan (kolom yang tidak ada
		// dilewati). Penghubungnya ke T_ANEKALIST adalah INDEXTANEKALIST — terukur
		// 2026-10-07: tanpa kolom ini, registrasi tidak dapat mengetahui coverage milik
		// objek mana. Nilai baris objek menang atas isi dokumen, sama seperti kunci lain.
		if line.Key == "aneka" {
			row["INDEXTANEKALIST"] = firstFilled(object.ListIndex, object.IndexObject)
		}
		if isBlank(row["INDEXCOVERAGE"]) {
			row["INDEXCOVERAGE"] = json.Number(strconv.Itoa(i + 1))
		}
		// T_COVERAGELIST_PERSON tidak punya kolom COVERAGE; kodenya di COVERAGEID, dan
		// registrasi membacanya dari sana. Dokumen Person bisa menyimpannya sebagai
		// `Coverage`, sehingga kodenya disalin bila COVERAGEID belum terisi.
		if line.Key == "person" && isBlank(row["COVERAGEID"]) && !isBlank(row["COVERAGE"]) {
			row["COVERAGEID"] = row["COVERAGE"]
		}
		result.Coverage = append(result.Coverage, row)

		spreads, err := spreadingArray(nested)
		if err != nil {
			return Converted{}, fmt.Errorf("spreading coverage ke-%d: %w", i+1, err)
		}
		for j, s := range spreads {
			sfields, _, err := flatten(s)
			if err != nil {
				return Converted{}, fmt.Errorf("spreading ke-%d coverage ke-%d: %w", j+1, i+1, err)
			}
			srow := Row(sfields)
			applyKeys(srow, object)
			srow["INDEXCOVERAGE"] = row["INDEXCOVERAGE"]
			if isBlank(srow["COVERAGE"]) {
				srow["COVERAGE"] = firstFilled(row["COVERAGE"], row["COVERAGEID"])
			}
			if isBlank(srow["INDEXSPREADING"]) {
				srow["INDEXSPREADING"] = json.Number(strconv.Itoa(j + 1))
			}
			result.Spreading = append(result.Spreading, srow)
		}
	}
	return result, nil
}

// applyKeys menimpa kolom kunci dengan nilai dari baris objeknya.
func applyKeys(row Row, object ObjectRow) {
	row["IDPEGA"] = object.IDPega
	row["NOPOLIS"] = object.PolicyNo
	row["PRODKE"] = object.ProdKe
	row["INDEXOBJECT"] = object.IndexObject
	if object.ApplicationID != "" {
		row["APPLICATIONID"] = object.ApplicationID
	}
}

// coverageArray menemukan larik coverage di dalam dokumen.
//
// Dokumen boleh berupa larik langsung, atau objek yang salah satu kuncinya (tanpa
// memandang huruf besar-kecil) adalah salah satu coverageRootKeys.
func coverageArray(document []byte) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(document)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var list []json.RawMessage
		if err := decode(trimmed, &list); err != nil {
			return nil, fmt.Errorf("JSON tidak sah: %w", err)
		}
		return list, nil
	}
	var root map[string]json.RawMessage
	if err := decode(trimmed, &root); err != nil {
		return nil, fmt.Errorf("JSON tidak sah: %w", err)
	}
	upper := upperKeys(root)
	for _, key := range coverageRootKeys {
		raw, ok := upper[key]
		if !ok {
			continue
		}
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 || raw[0] != '[' {
			continue
		}
		var list []json.RawMessage
		if err := decode(raw, &list); err != nil {
			return nil, fmt.Errorf("larik %s tidak sah: %w", key, err)
		}
		return list, nil
	}
	return nil, fmt.Errorf("%w (kunci akar: %s)", ErrUnknownShape, strings.Join(sortedKeys(upper), ", "))
}

// spreadingArray menemukan larik spreading di antara isian bersarang satu coverage.
func spreadingArray(nested map[string]json.RawMessage) ([]json.RawMessage, error) {
	for _, key := range spreadingKeys {
		raw, ok := nested[key]
		if !ok {
			continue
		}
		var list []json.RawMessage
		if err := decode(raw, &list); err != nil {
			return nil, fmt.Errorf("larik %s tidak sah: %w", key, err)
		}
		return list, nil
	}
	return nil, nil
}

// flatten memisahkan isian skalar (menjadi kolom) dari isian bersarang.
//
// Isian bersarang tetap ikut sebagai json.RawMessage di peta pertama, supaya dapat
// ditulis ke kolom BLOB/CLOB bernama sama bila ada. Ia juga dikembalikan terpisah untuk
// mencari larik spreading.
func flatten(raw json.RawMessage) (map[string]any, map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := decode(raw, &object); err != nil {
		return nil, nil, fmt.Errorf("bukan objek JSON: %w", err)
	}
	fields := make(map[string]any, len(object))
	nested := map[string]json.RawMessage{}
	for key, value := range object {
		name := strings.ToUpper(strings.TrimSpace(key))
		value = bytes.TrimSpace(value)
		if len(value) == 0 {
			continue
		}
		switch value[0] {
		case '{', '[':
			nested[name] = value
			fields[name] = value
		case 'n': // null
		default:
			var scalar any
			if err := decode(value, &scalar); err != nil {
				return nil, nil, fmt.Errorf("isian %s: %w", key, err)
			}
			fields[name] = scalar
		}
	}
	return fields, nested, nil
}

func decode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func upperKeys(m map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(m))
	for key, value := range m {
		result[strings.ToUpper(strings.TrimSpace(key))] = value
	}
	return result
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isBlank(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case json.Number:
		return strings.TrimSpace(string(v)) == ""
	}
	return false
}

func firstFilled(values ...any) any {
	for _, v := range values {
		if !isBlank(v) {
			return v
		}
	}
	return nil
}

// ParsePolicyList memecah teks daftar nomor polis menjadi nomor-nomor unik.
//
// Pemisahnya baris baru, koma, titik koma, tab, atau spasi — sehingga daftar yang
// ditempel dari Excel maupun dari teks biasa sama-sama diterima. Urutan masukan
// dipertahankan.
func ParsePolicyList(text string) []string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';' || r == '\t' || r == ' '
	})
	seen := map[string]bool{}
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		result = append(result, p)
	}
	return result
}
