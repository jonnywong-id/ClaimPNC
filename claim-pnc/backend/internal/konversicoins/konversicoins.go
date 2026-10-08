// Package konversicoins melayani menu Konversi Coins (`MENU_ID 89`, MENU_PROGRAM
// `KonversiCoins`).
//
// # Apa yang dikerjakan
//
// Koasuransi polis dibaca registrasi, komite, dan PLA dari tabel `POOLDATA.T_COINSLIST`
// (Work Owner, 2026-10-01 — lihat `internal/registrasi/repo/sqlstore/lookup.sql`
// `polis_koasuransi`). Sumber aslinya adalah larik `CoinsList` di dokumen polis
// `POOLDATA.JSON_POLIS.DATA_JSONBLOB`. Modul ini membaca dokumen itu dari basis data LIVE
// untuk sejumlah nomor polis, lalu di basis data TEST:
//
//	JSON_POLIS.DATA_JSONBLOB $.CoinsList[*] (LIVE)  →  T_COINSLIST (TEST), satu baris per anggota
//
// Satu dokumen per versi polis: untuk setiap PRODKE dipakai baris JSON_POLIS ber-DATA_JSONBLOB
// yang TGL_INPUT-nya terbaru. IDPEGA baris tujuan diambil dari baris JSON_POLIS itu.
//
// # LIVE hanya dibaca
//
// Seluruh pembacaan LIVE berjalan di dalam transaksi `SET TRANSACTION READ ONLY`, dan tidak
// ada satu pun pernyataan tulis yang diarahkan ke koneksi LIVE. Modul menolak berjalan bila
// kedua koneksi menunjuk basis data dan pengguna yang sama.
//
// # Pemetaan kolom — ditetapkan dari data, bukan dari nama
//
// Berbeda dengan Konversi Coverage dan Konversi Object Item Fire, nama kunci CoinsList TIDAK
// selalu sama dengan nama kolomnya (`PercentShare` → `PERCENT_SHARE`, `HandingFee` →
// `HANDLINGFEE`). Pemetaan di Columns karena itu ditulis eksplisit, dan diturunkan dengan
// membandingkan baris T_COINSLIST yang ditulis Pega di LIVE terhadap dokumen JSON_POLIS
// pasangannya (2026-10-08, 40 polis, 185 baris):
//
//   - 40/40 polis: jumlah baris T_COINSLIST = panjang CoinsList, dan IDPEGA = JSON_POLIS.IDPEGA.
//   - 185/185 baris: setiap kolom di Columns sama nilainya dengan kunci pasangannya.
//   - FLAGDELETE: CoinsList tidak membawa kunci FlagDelete; Pega menulis '0' pada 178 dari 185
//     baris (7 sisanya NULL, sebabnya tidak terbaca). Yang ditulis di sini '0'.
//
// Kolom yang SENGAJA dibiarkan NULL karena sumbernya tidak dapat dipastikan:
//
//   - OLDPERCENT_SHARE, OLDTSI_SHARE, OLDPREMI_SHARE, OLDDISCOUNT_SHARE, OLDBROKERAGE,
//     OLDHANDLINGFEE, OLDPPH, OLDPPN — cocok dengan `OldCoins[]` anggota yang sama hanya
//     ±75% baris.
//   - SUMTSI, SUMPREMI, SUMDISCOUNT, SUMBROKERAGE dan pasangan OLD_SUM* — terisi di 113 dari 185
//     baris LIVE dan tidak cocok dengan satu sumber mana pun di dokumen.
//   - PERCENTHANDLINGFEE — ada di dokumen, tetapi Pega membiarkannya NULL di seluruh baris
//     LIVE; dibiarkan sama.
//   - FLAGOLDDATA, FLAGEDITDATA — hanya ada di TEST, tidak di LIVE.
//
// Tidak ada pembaca di aplikasi ini yang memakai kolom-kolom itu (yang dibaca hanya COINSID,
// COINSNAME, LEADER, PERCENT_SHARE, FLAGDELETE).
//
// Modul ini sengaja TIDAK mengimpor modul konversi lain (modul tidak saling mengimpor):
// pengurai daftar polis dan pengubah tipe kolomnya disalin, karena keduanya kecil dan stabil.
package konversicoins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Tabel dan kolom yang dikonversi.
const (
	SourceTable = "JSON_POLIS"    // tabel dokumen polis di LIVE
	BlobColumn  = "DATA_JSONBLOB" // kolom BLOB berisi dokumen polis
	CoinsTable  = "T_COINSLIST"   // tabel tujuan di TEST
)

// Mapping adalah satu kolom tujuan beserta kunci CoinsList sumbernya.
type Mapping struct {
	Column string // nama kolom T_COINSLIST
	Key    string // kunci item CoinsList, dicocokkan tanpa memandang huruf besar-kecil
	// Default ditulis bila kuncinya tidak ada atau kosong; nil berarti NULL.
	Default any
}

// Columns adalah pemetaan item CoinsList → kolom T_COINSLIST. Dasarnya ada di dokumentasi paket.
var Columns = []Mapping{
	{Column: "COINSID", Key: "CoinsID"},
	{Column: "COINSNAME", Key: "CoinsName"},
	{Column: "LEADER", Key: "Leader"}, // teks 'true'/'false', sama dengan yang ditulis Pega
	{Column: "PERCENT_SHARE", Key: "PercentShare"},
	{Column: "TSI_SHARE", Key: "TSIShare"},
	{Column: "PREMI_SHARE", Key: "PremiShare"},
	{Column: "DISCOUNT_SHARE", Key: "DiscountShare"},
	{Column: "BROKERAGE", Key: "Brokerage"},
	{Column: "PERCENTBROKERAGE", Key: "PercentBrokerage"},
	{Column: "HANDLINGFEE", Key: "HandingFee"}, // salah ketik "Handing" memang nama kunci Pega
	{Column: "PPH", Key: "PPH"},
	{Column: "PPN", Key: "PPN"},
	{Column: "FLAGDELETE", Key: "FlagDelete", Default: "0"},
}

// Kolom kunci yang diisi dari baris JSON_POLIS, bukan dari item.
const (
	ColIDPega   = "IDPEGA"
	ColPolicyNo = "NOPOLIS"
	ColProdKe   = "PRODKE"
)

// PolicyDoc adalah satu versi dokumen polis di LIVE.
type PolicyDoc struct {
	IDPega   string
	PolicyNo string
	ProdKe   string
	Document []byte // isi DATA_JSONBLOB apa adanya
}

// Row adalah satu baris tujuan: nama kolom → nilai JSON (string, json.Number, bool) atau
// nilai bawaan. Pengubahan ke tipe kolom dilakukan belakangan — lihat Coerce.
type Row map[string]any

// Convert mengurai CoinsList satu dokumen polis menjadi baris T_COINSLIST.
//
// Dokumen tanpa CoinsList, atau dengan CoinsList null/kosong, berarti polis tanpa koasuransi
// pada versi itu: hasilnya nol baris tanpa galat. Peringatan dikembalikan untuk anggota
// tanpa CoinsID — barisnya tetap ditulis, sebab yang dikonversi adalah isi LIVE apa adanya.
func Convert(doc PolicyDoc) ([]Row, []string, error) {
	items, err := coinsArray(doc.Document)
	if err != nil {
		return nil, nil, err
	}
	var (
		rows     []Row
		warnings []string
	)
	for i, raw := range items {
		fields, err := flatten(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("CoinsList ke-%d: %w", i+1, err)
		}
		row := Row{ColIDPega: doc.IDPega, ColPolicyNo: doc.PolicyNo, ColProdKe: doc.ProdKe}
		for _, m := range Columns {
			value, ok := fields[strings.ToUpper(m.Key)]
			if !ok || isBlank(value) {
				if m.Default != nil {
					row[m.Column] = m.Default
				}
				continue
			}
			row[m.Column] = value
		}
		if _, ok := row["COINSID"]; !ok {
			warnings = append(warnings, fmt.Sprintf("PRODKE %s: CoinsList ke-%d tanpa CoinsID", doc.ProdKe, i+1))
		}
		rows = append(rows, row)
	}
	return rows, warnings, nil
}

// coinsArray mengambil larik CoinsList dari akar dokumen polis.
func coinsArray(document []byte) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(document)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var root map[string]json.RawMessage
	if err := decode(trimmed, &root); err != nil {
		return nil, fmt.Errorf("dokumen polis bukan JSON objek yang sah: %w", err)
	}
	for key, raw := range root {
		if !strings.EqualFold(strings.TrimSpace(key), "CoinsList") {
			continue
		}
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 || string(raw) == "null" {
			return nil, nil
		}
		if raw[0] != '[' {
			return nil, fmt.Errorf("CoinsList bukan larik")
		}
		var list []json.RawMessage
		if err := decode(raw, &list); err != nil {
			return nil, fmt.Errorf("larik CoinsList tidak sah: %w", err)
		}
		return list, nil
	}
	return nil, nil
}

// flatten mengubah satu objek JSON menjadi isian skalar berkunci huruf besar. Isian
// bersarang (mis. OldCoins) tidak dipetakan ke kolom mana pun dan dilewati.
func flatten(raw json.RawMessage) (map[string]any, error) {
	var object map[string]json.RawMessage
	if err := decode(raw, &object); err != nil {
		return nil, fmt.Errorf("bukan objek JSON: %w", err)
	}
	fields := make(map[string]any, len(object))
	for key, value := range object {
		value = bytes.TrimSpace(value)
		if len(value) == 0 || value[0] == '{' || value[0] == '[' || string(value) == "null" {
			continue
		}
		var scalar any
		if err := decode(value, &scalar); err != nil {
			return nil, fmt.Errorf("isian %s: %w", key, err)
		}
		fields[strings.ToUpper(strings.TrimSpace(key))] = scalar
	}
	return fields, nil
}

func decode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
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
