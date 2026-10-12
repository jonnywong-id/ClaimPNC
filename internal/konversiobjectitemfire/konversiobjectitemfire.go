// Package konversiobjectitemfire melayani menu Konversi Object Item Fire (`MENU_ID 88`,
// MENU_PROGRAM `KonversiObjekItemFire`).
//
// # Apa yang dikerjakan
//
// Daftar item objek polis Fire (PropertyItemList) tersimpan sebagai dokumen JSON di kolom
// BLOB `POOLDATA.T_PROPERTYLIST.PROPERTYITEMLIST`. Pilihan Objek pada item estimasi klaim Fire
// dibaca dari kolom itu (`GetListPropertyItemListFire`, lihat
// `internal/registrasi/repo/sqlstore/estimation.sql`), dan sebagian polis di basis data TEST
// tidak membawanya — terukur 2026-10-07 pada polis klaim PNCN.26.38.
//
// Modul ini membaca dokumen itu dari basis data LIVE untuk sejumlah nomor polis, lalu di
// basis data TEST:
//
//	T_PROPERTYLIST.PROPERTYITEMLIST (LIVE)  →  T_PROPERTYITEMLIST (TEST), satu baris per item
//	                                         →  T_PROPERTYLIST.PROPERTYITEMLIST (TEST), BLOB disalin
//
// # LIVE hanya dibaca
//
// Seluruh pembacaan LIVE berjalan di dalam transaksi `SET TRANSACTION READ ONLY`, dan tidak
// ada satu pun pernyataan tulis yang diarahkan ke koneksi LIVE. Modul menolak berjalan bila
// kedua koneksi menunjuk basis data dan pengguna yang sama.
//
// # Pemetaan kolom tidak ditulis di kode
//
// Kunci JSON item (`ItemType`, `ItemTypeIndo`, `PropertyItemCode`, `PropertyItemGroup`,
// `TSIObjectItem`, `TSIAdjustment`, `PercentageAdjustment`, `FlagDelete`, …) dicocokkan ke
// kolom T_PROPERTYITEMLIST bernama sama tanpa memandang huruf besar-kecil, menurut kamus data
// basis data TEST (`ALL_TAB_COLUMNS`). Kunci tanpa kolom dilaporkan, bukan dibuang diam-diam
// — kecuali `pxObjClass`, nama kelas Pega yang memang tidak punya kolom.
//
// Modul ini sengaja TIDAK mengimpor modul Konversi Coverage (modul tidak saling mengimpor):
// pengurai JSON dan pengubah tipe kolomnya disalin, karena keduanya kecil dan stabil.
package konversiobjectitemfire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Tabel dan kolom yang dikonversi.
const (
	ObjectTable = "T_PROPERTYLIST"     // tabel objek, LIVE dan TEST
	BlobColumn  = "PROPERTYITEMLIST"   // kolom BLOB berisi dokumen PropertyItemList
	ItemTable   = "T_PROPERTYITEMLIST" // tabel item tujuan di TEST
)

// ObjectRow adalah satu baris objek polis Fire di LIVE beserta dokumen item-nya.
type ObjectRow struct {
	IDPega      string
	PolicyNo    string
	ProdKe      string
	IndexObject string
	Document    []byte // isi BLOB apa adanya; nil bila kolomnya NULL
}

// Row adalah satu baris tujuan: nama kolom (huruf besar) → nilai JSON.
//
// Nilai berupa string, json.Number, bool, atau json.RawMessage untuk objek/larik bersarang.
// Pengubahan ke tipe kolom dilakukan belakangan, setelah tipe kolom tujuan diketahui — lihat
// Coerce.
type Row map[string]any

// ignoredKeys adalah kunci JSON yang memang tidak punya kolom dan tidak perlu dilaporkan.
var ignoredKeys = map[string]bool{"PXOBJCLASS": true}

// Kunci akar dokumen yang berisi larik item.
var itemRootKeys = []string{"PROPERTYITEMLIST"}

// ErrUnknownShape berarti dokumen bukan JSON PropertyItemList yang dikenali.
var ErrUnknownShape = errors.New("dokumen tidak berisi larik PropertyItemList yang dikenali")

// Convert mengurai satu dokumen PropertyItemList menjadi baris T_PROPERTYITEMLIST.
//
// Kolom kunci baris tujuan selalu diambil dari baris objeknya, bukan dari dokumen:
// NOPOLIS, PRODKE, INDEXOBJECT, IDPEGA. INDEXOBJECT adalah kunci yang dipakai Pega
// (`GetListPropertyItemListFire` menyaring `indexobject`), sehingga item tergabung ke objek
// yang sama di kedua sistem.
func Convert(object ObjectRow) ([]Row, error) {
	if len(bytes.TrimSpace(object.Document)) == 0 {
		return nil, nil
	}
	items, err := itemArray(object.Document)
	if err != nil {
		return nil, err
	}
	var rows []Row
	for i, raw := range items {
		fields, err := flatten(raw)
		if err != nil {
			return nil, fmt.Errorf("item ke-%d: %w", i+1, err)
		}
		for key := range ignoredKeys {
			delete(fields, key)
		}
		row := Row(fields)
		row["NOPOLIS"] = object.PolicyNo
		row["PRODKE"] = object.ProdKe
		row["INDEXOBJECT"] = object.IndexObject
		row["IDPEGA"] = object.IDPega
		rows = append(rows, row)
	}
	return rows, nil
}

// itemArray menemukan larik item di dalam dokumen: larik langsung, atau objek yang salah satu
// kuncinya (tanpa memandang huruf besar-kecil) adalah itemRootKeys.
func itemArray(document []byte) ([]json.RawMessage, error) {
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
	upper := map[string]json.RawMessage{}
	for key, value := range root {
		upper[strings.ToUpper(strings.TrimSpace(key))] = value
	}
	for _, key := range itemRootKeys {
		raw, ok := upper[key]
		if !ok {
			continue
		}
		raw = bytes.TrimSpace(raw)
		if string(raw) == "null" {
			return nil, nil
		}
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

// flatten mengubah satu objek JSON menjadi isian berkunci huruf besar. Isian bersarang ikut
// sebagai json.RawMessage, supaya dapat ditulis ke kolom BLOB/CLOB bernama sama bila ada
// (mis. STOCKADJUSTMENTLIST).
func flatten(raw json.RawMessage) (map[string]any, error) {
	var object map[string]json.RawMessage
	if err := decode(raw, &object); err != nil {
		return nil, fmt.Errorf("bukan objek JSON: %w", err)
	}
	fields := make(map[string]any, len(object))
	for key, value := range object {
		name := strings.ToUpper(strings.TrimSpace(key))
		value = bytes.TrimSpace(value)
		if len(value) == 0 {
			continue
		}
		switch value[0] {
		case '{', '[':
			fields[name] = value
		case 'n': // null
		default:
			var scalar any
			if err := decode(value, &scalar); err != nil {
				return nil, fmt.Errorf("isian %s: %w", key, err)
			}
			fields[name] = scalar
		}
	}
	return fields, nil
}

func decode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
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

// ParsePolicyList memecah teks daftar nomor polis menjadi nomor-nomor unik.
//
// Pemisahnya baris baru, koma, titik koma, tab, atau spasi — sehingga daftar yang ditempel
// dari Excel maupun dari teks biasa sama-sama diterima. Urutan masukan dipertahankan.
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

// Filter mempersempit objek yang dikonversi. Isian kosong berarti tanpa batas: Filter
// kosong mengonversi seluruh versi (PRODKE) dan seluruh objek (INDEXOBJECT) polisnya.
// Filter berlaku sama untuk setiap polis di daftar, dan dipasang langsung di kueri LIVE.
//
// Disalin dari modul Konversi Coverage, bukan diimpor — modul tidak saling mengimpor.
type Filter struct {
	ProdKe      string
	IndexObject string
}

// ErrInvalidFilter berarti isian filter bukan bilangan bulat tak negatif.
var ErrInvalidFilter = errors.New("filter tidak sah")

// NewFilter merapikan dan memeriksa isian filter. PRODKE dan INDEXOBJECT adalah kolom
// angka, sehingga hanya digit yang diterima; nol di depan dibuang supaya `01` cocok
// dengan TO_CHAR(PRODKE) = `1`.
func NewFilter(prodKe, indexObject string) (Filter, error) {
	p, err := normalizeNumber("PRODKE", prodKe)
	if err != nil {
		return Filter{}, err
	}
	i, err := normalizeNumber("INDEXOBJECT", indexObject)
	if err != nil {
		return Filter{}, err
	}
	return Filter{ProdKe: p, IndexObject: i}, nil
}

// Empty berarti tidak ada batas sama sekali.
func (f Filter) Empty() bool { return f.ProdKe == "" && f.IndexObject == "" }

// Describe adalah uraian filter untuk pesan dan log, mis. "PRODKE 1, INDEXOBJECT 3".
func (f Filter) Describe() string {
	var parts []string
	if f.ProdKe != "" {
		parts = append(parts, "PRODKE "+f.ProdKe)
	}
	if f.IndexObject != "" {
		parts = append(parts, "INDEXOBJECT "+f.IndexObject)
	}
	return strings.Join(parts, ", ")
}

func normalizeNumber(name, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("%w: %s harus bilangan bulat tak negatif, bukan %q", ErrInvalidFilter, name, text)
		}
	}
	text = strings.TrimLeft(text, "0")
	if text == "" {
		text = "0"
	}
	return text, nil
}
