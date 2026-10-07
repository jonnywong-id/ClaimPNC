package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// pegaWorkKeyPrefix adalah awalan kunci objek kerja Pega pada `T_CLAIM_PNC.CLAIMID`.
//
// Nama kelas internal Pega tertanam di dalam kunci data bisnis — utang teknis §4.1 yang
// `D-22` dan `D-71` hapus untuk klaim baru. Ia ditiru di data contoh karena pencarian di
// layar ini mencocokkan KUNCI itu, bukan nomor klaim: data contoh yang kuncinya berbentuk
// lain akan membuat uji pencarian lolos karena alasan yang salah.
//
// Perhatikan SPASI di ujungnya. Ia bagian dari nilainya.
const pegaWorkKeyPrefix = "ASM-FW-GCNMFW-WORK "

// workKey menyusun kunci objek kerja dari nomor klaim.
func workKey(claimNo string) string { return pegaWorkKeyPrefix + claimNo }

// day membentuk tanggal kalender polos, tanpa jam.
//
// UTC, bukan WIB — sama dengan Query.From dan dengan kolom tanggal di basis data lama,
// yang disimpan tanpa zona waktu. Lihat calendarDate pada query.go.
func day(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
}

// NewSampleStore membentuk penyimpanan berisi data contoh.
//
// # Janji yang dipegang sepuluh klaim di bawah
//
// SETIAP penyaring punya baris yang cocok MAUPUN yang tidak. Yang membuktikan sebuah
// penyaring bekerja bukan baris yang muncul, melainkan baris yang seharusnya TIDAK muncul
// dan memang tidak muncul — dan di layar ini ada tujuh penyaring yang berbeda-beda antar
// tab.
//
// Dua klaim punya tugas tambahan, dan keduanya menjaga kejanggalan yang sengaja dibawa
// dari Pega tetap tertiru:
//
//	PNC-1008  seluruh PLA-nya ber-ISKIRIM='0' -> MUNCUL di daftar, tanggal advice KOSONG
//	PNC-1006  cabangnya kosong (NULL)        -> TERSARING KELUAR dari tab DLA
//
// Tanpa keduanya, kedua kejanggalan itu dapat "diperbaiki" tanpa satu pun uji gagal.
func NewSampleStore() *Store {
	store := NewStore()
	store.Seed(sampleClaims(), sampleAdvices())
	return store
}

// sampleClaims adalah sepuluh baris `T_CLAIM_PNC` contoh.
//
// Nama tertanggung dan nomor polis di sini KARANGAN. Data nasabah tidak pernah disalin ke
// dalam berkas yang di-commit (`D-69`), dan yang dibutuhkan uji hanyalah nilai yang dapat
// dibedakan satu sama lain.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Lengkap: punya PLA, DLA, dan Pre-DLA yang ketiganya memenuhi syarat.
// PLA-nya belum menunjuk reasuradur -> TIDAK masuk tab PLA.
// DLA-nya sudah terkirim            -> TIDAK masuk tab DLA.
// Personal Accident. Dikecualikan KETIGA tab meski dokumennya lengkap.
// Travel. Dikecualikan ketiga tab.
// Cabang ASNET. Masuk tab PLA dan Pre DLA, TIDAK masuk tab DLA.
// Cabang KOSONG — mewakili NULL di Oracle.
//
// Ia TIDAK masuk tab DLA, karena `NULL <> 'ASNET'` menghasilkan UNKNOWN,
// bukan TRUE. Itu perilaku Pega, dan klaim inilah yang menjaganya tertiru.
// Kelompok bisnis 10008. Dikecualikan tab DLA saja.
// SELURUH PLA-nya ber-ISKIRIM='0'.
//
// Ia MUNCUL di tab PLA — penyaring keanggotaan menerima '0' — tetapi kolom
// "Tanggal PLA"-nya KOSONG, karena sub-kueri tanggalnya hanya menerima NULL.
// Kejanggalan Pega yang sengaja dibawa (`P-5`).
// Pre-DLA-nya sudah ber-Nomor Akseptasi -> TIDAK masuk tab Pre DLA.
// Tanpa dokumen sama sekali.
//
// Ia tidak muncul di tab mana pun, tetapi ADA — sehingga permintaan
// rinciannya dijawab daftar kosong, bukan "tidak ditemukan". Itulah satu
// keadaan yang membedakan kedua jawaban itu.
func sampleClaims() []Claim { return sampledata.Must[[]Claim](sampleJSON, "sampleClaims") }

// sampleAdvices adalah baris dokumen contoh untuk ketiga tabel.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// PNC-1001 — lengkap di ketiga tab.
// Alamat dan negara diisi supaya tombol "SEND" dapat dijalankan sampai
// tuntas di lingkungan pengembangan. Negaranya INDONESIA, sehingga
// suratnya berbahasa Indonesia.
// Negaranya BUKAN Indonesia -> suratnya berbahasa Inggris. Pasangan ini
// sengaja ada supaya kedua bahasa terwakili di data contoh.
// Lampirannya sudah ada -> baris ini MUNCUL di panel "Print Pre DLA".
// Pre-DLA kedua pada klaim yang SAMA, tetapi TANPA lampiran.
//
// Ia muncul di tab Pre DLA — penyaring tabnya `NOAKSEP IS NULL`, dan kolom
// itu kosong di sini — tetapi TIDAK muncul di panel "Print Pre DLA", karena
// gabungan ke tabel lampiran menyingkirkannya.
//
// Pasangan ini sengaja ada pada satu klaim: tanpanya, selisih jumlah baris
// antara tab dan panelnya tidak pernah terwakili di data contoh, dan
// penyaring lampiran dapat dihapus tanpa satu pun uji gagal.
// PNC-1002 — dua penolakan sekaligus.
// REINSCODE kosong -> tidak lolos penyaring tab PLA.
// Sudah terkirim -> tidak lolos penyaring tab DLA.
// PNC-1003 dan PNC-1004 — dokumennya memenuhi syarat, klaimnya yang
// dikecualikan. Keduanya ada supaya penyaring lini bisnis benar-benar teruji:
// tanpa dokumen, klaim ini tidak akan muncul walaupun penyaringnya dilepas.
// PNC-1005 (ASNET), PNC-1006 (cabang kosong), PNC-1007 (bisnis 10008).
// Ketiganya punya PLA dan DLA yang memenuhi syarat dokumen, sehingga yang
// menentukan munculnya hanyalah penyaring di sisi klaim.
// PNC-1008 — SELURUH PLA-nya ber-ISKIRIM='0'.
//
// Dua baris, bukan satu, supaya jelas bahwa yang mengosongkan kolom tanggal
// bukanlah ketiadaan dokumen melainkan nilai `'0'` itu sendiri.
// PNC-1009 — Pre-DLA sudah ber-Nomor Akseptasi, sehingga keluar dari antrean.
// Sudah ber-Nomor Akseptasi -> keluar dari tab Pre DLA. Lampirannya tetap
// diisi supaya terlihat bahwa yang menyingkirkannya adalah akseptasinya,
// bukan ketiadaan lampiran.
func sampleAdvices() []Advice { return sampledata.Must[[]Advice](sampleJSON, "sampleAdvices") }
