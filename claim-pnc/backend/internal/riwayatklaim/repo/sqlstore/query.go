// Package sqlstore memenuhi seam riwayatklaim.Repo dan riwayatklaim.ProtectionRepo
// dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis, dan di modul ini cacatnya yang
// paling parah di seluruh export: kedua belas kueri pencarian lama menyisipkan nilai
// langsung ke teks SQL, dan gerbang proteksinya menjalankan SQL yang SELURUHNYA berasal
// dari sebuah properti klipboard (`RDB List/GetFileOnPc_link_attachmentGCNM-SQL.xml`
// berisi tepat satu baris: `{ASIS:TempClaimAttach.AlasanKlaim}`).
//
// Larangan itu TIDAK ikut dikecualikan oleh keputusan Work Owner "replikasi apa adanya"
// atas cacat aturan bisnis: yang direplikasi adalah perilaku, bukan celah injeksi.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "riwayatklaim/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "riwayatklaim/sqlstore", name) }

// resultColumns adalah keenam belas alias yang dikembalikan SETIAP kueri pencarian.
//
// Urutannya WAJIB sama dengan urutan kolom di riwayatklaim.sql dan dengan urutan pemindai
// scanClaim. Ia ditulis lengkap, bukan `SELECT *`: kolom disebut namanya tanpa
// perkecualian (`08-TECHNICAL-STRATEGY.md` §4.3), dan menyebutkannya di sini pula yang
// membuat ketiga tempat itu dapat diuji kesesuaiannya di query_test.go.
const resultColumns = `REFERENCE, CLAIM_NUMBER, POLICY_NUMBER, INSURED_NAME, LOSS_DATE,
       BUSINESS_NAME, BRANCH_NAME, WORK_STATUS, CLAIM_POSITION,
       CLOSE_DATE, CLOSE_NOTE, TECHNICAL_PIC,
       ACCEPTANCE_NUMBER, AUCTION_HOUSE_ID, INSURED_ITEM_NAME, BIRTH_DATE`

// paged membungkus satu kueri pencarian menjadi satu halaman hasil.
//
// # Kenapa dirangkai, bukan ditulis dua kali per tipe
//
// Kesebelas kueri pencarian masing-masing butuh dua bentuk: penghitung seluruh baris yang
// cocok, dan pengambil satu halaman. Menuliskan keduanya di berkas .sql berarti dua puluh
// dua blok yang harus berubah berpasangan, dan satu yang tertinggal akan membuat jumlah
// di layar tidak sesuai dengan isinya — cacat yang tidak menghasilkan galat, hanya angka
// yang salah.
//
// # Kenapa ini BUKAN perangkaian SQL yang dilarang
//
// Yang dirangkai adalah teks kueri milik kita sendiri dari berkas .sql, seluruhnya
// konstanta saat kompilasi. Tidak ada satu pun nilai dari pengguna yang menyentuhnya —
// nilai pencarian, offset, dan ukuran halaman semuanya tetap lewat parameter binding.
// Yang dilarang `08-TECHNICAL-STRATEGY.md` §4.3 adalah merangkai NILAI ke dalam teks SQL,
// persis pola `{ASIS:…}` warisan.
//
// Urutannya menurut REFERENCE — CLAIMID, yang unik per klaim. Tanpa urutan yang
// ditetapkan, paginasi terhadap hasil yang urutannya diserahkan ke basis data membuat satu
// baris muncul di dua halaman sekaligus hilang dari halaman lain.
func paged(name string) string {
	return "SELECT " + resultColumns + "\n" +
		"  FROM (" + query(name) + ")\n" +
		" ORDER BY REFERENCE\n" +
		" OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY"
}

// counted membungkus satu kueri pencarian menjadi penghitung baris.
//
// Alasan merangkainya sama dengan paged, dan sama pula alasan ia tidak melanggar larangan
// perangkaian SQL.
func counted(name string) string {
	return "SELECT COUNT(*) FROM (" + query(name) + ")"
}
