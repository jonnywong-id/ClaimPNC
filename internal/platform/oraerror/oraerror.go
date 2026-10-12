// Package oraerror menyusun pesan galat Oracle yang dapat ditampilkan di layar.
//
// # Kenapa galat basis data ditampilkan apa adanya
//
// Aturan umum aplikasi ini menyembunyikan rincian galat internal dari peramban. Untuk modul
// pelaporan klaim dan registrasi (termasuk tombol Register Klaim) Work Owner meminta
// sebaliknya (2026-10-10): kegagalan penyimpanan ke tabel tetap menampilkan pesan Oracle-nya,
// mis. `Gagal penyimpanan ke tabel POOLDATA.T_CLAIM_PNC: ORA-01427: single-row subquery
// returns more than one row`, supaya petugas dapat melaporkannya tanpa membuka log server.
//
// Yang dikirim hanya nomor dan teks galat Oracle beserta nama tabelnya — bukan teks SQL,
// bukan nilai yang diikat, dan bukan alamat atau kredensial basis data.
package oraerror

import (
	"regexp"
	"strings"
)

var (
	// oraPattern menangkap satu baris galat Oracle, mis. `ORA-01427: single-row ...`.
	oraPattern = regexp.MustCompile(`ORA-\d{5}:[^\r\n]*`)

	// quotedTable menangkap `"OWNER"."TABEL"` yang disebut Oracle sendiri (ORA-12899,
	// ORA-01400, …).
	quotedTable = regexp.MustCompile(`"([A-Z0-9_$#]+)"\."([A-Z0-9_$#]+)"`)

	// namedTable menangkap nama tabel yang disebut pembungkus galat di sqlstore.
	namedTable = regexp.MustCompile(`\b(?:[A-Z][A-Z0-9_]*\.)?(?:T_|CPNC_|M_|MST_|GCNM_|DATA_|PC_|LST_|V_)[A-Z0-9_]+\b`)

	// writeWords menandai pembungkus galat yang berasal dari penulisan.
	writeWords = []string{
		"menyimpan", "menyisipkan", "memperbarui", "menghapus", "menandai", "memasang",
		"menerbitkan", "menulis", "mencatat", "memindahkan", "mengunci", "menutup transaksi",
		"mengambil nomor",
	}
)

// Describe mengembalikan pesan layar untuk galat yang memuat galat Oracle (`ORA-nnnnn`).
// ok false bila galatnya bukan galat Oracle — pemanggil tetap memakai pesan umumnya.
func Describe(err error) (message string, ok bool) {
	if err == nil {
		return "", false
	}
	text := err.Error()
	ora := oraPattern.FindString(text)
	if ora == "" {
		return "", false
	}
	ora = strings.TrimSpace(ora)

	// Konteks pembungkus: teks sebelum galat Oracle-nya.
	context := text[:strings.Index(text, ora)]
	table := tableOf(ora, context)

	verb := "Gagal mengakses basis data"
	if isWrite(context) {
		verb = "Gagal penyimpanan ke basis data"
	}
	if table != "" {
		if verb == "Gagal penyimpanan ke basis data" {
			verb = "Gagal penyimpanan ke tabel " + table
		} else {
			verb = "Gagal membaca tabel " + table
		}
	}
	return verb + ": " + ora, true
}

func tableOf(ora, context string) string {
	if m := quotedTable.FindStringSubmatch(ora); m != nil {
		return m[1] + "." + m[2]
	}
	// Pembungkus terdalam paling dekat dengan galatnya — ambil sebutan tabel terakhir.
	if all := namedTable.FindAllString(context, -1); len(all) > 0 {
		return all[len(all)-1]
	}
	return ""
}

func isWrite(context string) bool {
	lower := strings.ToLower(context)
	for _, w := range writeWords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}
