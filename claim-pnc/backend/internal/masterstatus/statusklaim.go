// Package masterstatus adalah inti modul Master Status Klaim (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Status Klaim adalah salah satu dari empat konsep status yang `D-18` tetapkan memang
// berbeda dan semuanya dipertahankan. Ia menjawab pertanyaan **"klaim ini berada di
// keadaan bisnis apa"** — Register, Claim Committee, Paid, Rejected Claim, dan
// seterusnya. Domainnya **33 kode `1134`–`1166`** (`R-06` tertutup setelah isi master
// diterima), bukan 10 kode `1142`–`1151` seperti yang sempat diduga dokumen awal.
//
// # Kenapa kodenya terlihat seperti angka acak
//
// Tidak acak. Sistem lama membentuknya `id_site || lpad(urutan, 3, '0')`
// (`Database/PEGA_M_STS_CLAIM.prc:19`), dan pola yang sama dipakai SELURUH master di
// sistem itu — `PEGA_M_CAUSE_OF_LOSS`, `PEGA_M_SURVEYORS`, `PEGA_M_PANEL_HE`. Dengan
// `id_site = 1` dan urutan 134…166, hasilnya tepat `1134`…`1166`.
//
// Skema itu DIPERTAHANKAN, bukan diganti: 23 rule Pega masih membaca kode ini lewat
// `POOLDATA.V_STS_CLAIM` selama masa paralel, dan menggantinya berarti memutus
// keduanya. Pembentukannya ada di repo/sqlstore — bukan di sini.
//
// # Arti kode tidak boleh disimpulkan dari pemakaian
//
// Tiga arti yang sempat disimpulkan dari rule ternyata SELURUHNYA salah: `1143` bukan
// status awal melainkan Close Claim, `1150` bukan penanda terdaftar melainkan LOD
// Report, `1151` bukan Investigator melainkan Analyst
// (`docs/Steering/16-RISK-ANALYSIS.md` `R-06`). Karena itu modul ini tidak memuat satu
// pun arti kode di dalam kode program; seluruhnya dibaca dari master.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterstatus

import (
	"context"
	"strings"
	"unicode/utf8"
)

// PanjangLabelMaksimum membatasi panjang label status.
//
// Batas ini adalah KEBUTUHAN BARU, bukan peniruan: layar Pega tidak membatasi panjang
// sama sekali (`Section/BrowseStatusClaim-Section.xml` — `pyRequired=false`, tanpa
// `pyMaxLength`). Angkanya diambil dari lebar kolom yang dibuat migrasi `0002`, dan
// label terpanjang yang benar-benar ada hari ini hanya 27 karakter
// ("Close Claim for this object").
const PanjangLabelMaksimum = 100

// StatusKlaim adalah satu baris master status.
//
// Ketiga field memetakan langsung ke kolom yang dibaca Pega lewat
// `POOLDATA.V_STS_CLAIM`, dan namanya mengikuti `CONTEXT.md` — bukan nama kolomnya.
// Pemetaannya ada di repo/sqlstore, satu tempat saja.
type StatusKlaim struct {
	// Kode adalah LSC_ID. Dibuat sistem saat status ditambahkan dan TIDAK PERNAH
	// berubah sesudahnya — layar Pega pun menandainya read-only.
	Kode string

	// Label adalah LSC_NOTE, teks yang dibaca pengguna. Di layar Pega ia berlabel
	// "Status".
	Label string

	// KodeLama adalah OLD_LSC_ID, penomoran `01`–`11` yang melekat pada sebelas kode
	// pertama (`1134`–`1144`). Kosong untuk 22 kode sisanya, dan tidak pernah diisi
	// untuk status baru — ia jejak sejarah, bukan field yang dikelola.
	KodeLama string
}

// Bersih mengembalikan salinan dengan spasi tepi dibuang.
//
// Perapian ini nyata gunanya, bukan kerapian kosmetik: isi OLD_LSC_ID pada master
// tersimpan sebagai CHAR berisi padding — `"01  "`, bukan `"01"` (lihat
// `Database/v_sts_claim.csv`). Membiarkannya membuat perbandingan dan tampilan
// membawa spasi yang tidak dimaksudkan siapa pun.
func (s StatusKlaim) Bersih() StatusKlaim {
	return StatusKlaim{
		Kode:     strings.TrimSpace(s.Kode),
		Label:    strings.TrimSpace(s.Label),
		KodeLama: strings.TrimSpace(s.KodeLama),
	}
}

// KunciLabel adalah bentuk label yang dipakai menguji keunikan.
//
// Perbandingan mengabaikan besar-kecil huruf dan spasi tepi. Alasannya ada presedennya
// di repo ini: `docs/Steering/11-SECURITY.md` §3.1 mencatat tiga nama access group Pega
// muncul dalam dua kapitalisasi berbeda karena perbandingan rule lama tidak konsisten
// soal itu. "Paid" dan "PAID" adalah status yang sama bagi pengguna, dan harus
// diperlakukan sama oleh sistem.
//
// Indeks unik di basis data memakai ekspresi yang SAMA (`UPPER(TRIM(LSC_NOTE))`),
// sehingga pemeriksaan di sini dan penegakan di sana tidak dapat berbeda pendapat.
func KunciLabel(label string) string {
	return strings.ToUpper(strings.TrimSpace(label))
}

// Repo adalah seam ke penyimpanan master status.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memori (pengujian dan pengembangan
// tanpa basis data). Antarmukanya berbicara dalam istilah domain — Daftar, Ambil,
// Sisip, Perbarui — bukan istilah SQL.
type Repo interface {
	// Daftar mengembalikan seluruh status, terurut menurut kode.
	Daftar(ctx context.Context) ([]StatusKlaim, error)

	// Ambil mengembalikan satu status. Mengembalikan ErrTidakDitemukan bila kodenya
	// tidak ada.
	Ambil(ctx context.Context, kode string) (StatusKlaim, error)

	// Sisip menyimpan status baru dan mengembalikannya LENGKAP DENGAN KODE yang
	// dibuat penyimpanan. Kode tidak pernah datang dari pemanggil.
	Sisip(ctx context.Context, label string) (StatusKlaim, error)

	// Perbarui mengubah label status yang sudah ada. Kode dan kode lama tidak ikut
	// berubah. Mengembalikan ErrTidakDitemukan bila kodenya tidak ada.
	Perbarui(ctx context.Context, kode, label string) (StatusKlaim, error)
}

// PeriksaLabel mengumpulkan SELURUH pelanggaran aturan label, bukan berhenti pada yang
// pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama
// menampilkan seluruh pesan validasi sekaligus, dan mengembalikannya satu per satu akan
// membuat pengguna menekan Simpan berkali-kali untuk menemukan kesalahan berikutnya
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
//
// Keunikan TIDAK diperiksa di sini: ia menuntut membaca penyimpanan, sedangkan fungsi
// ini murni dan dapat diuji tanpa apa pun. Pemeriksaannya ada di usecase.
func PeriksaLabel(label string) []Pelanggaran {
	bersih := strings.TrimSpace(label)
	var pelanggaran []Pelanggaran

	if bersih == "" {
		// Perbedaan yang DISENGAJA dari sistem lama, disetujui Work Owner 2026-09-17.
		// Layar Pega mengizinkan label kosong, dan label kosong akan tampil sebagai
		// status kosong di 23 rule yang membaca V_STS_CLAIM — termasuk laporan TAT dan
		// KPI yang dibaca manajemen.
		pelanggaran = append(pelanggaran, Pelanggaran{
			Field: FieldLabel,
			Pesan: "Status wajib diisi.",
		})
	}

	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	if panjang := utf8.RuneCountInString(bersih); panjang > PanjangLabelMaksimum {
		pelanggaran = append(pelanggaran, Pelanggaran{
			Field: FieldLabel,
			Pesan: "Status paling panjang " + itoa(PanjangLabelMaksimum) + " karakter.",
		})
	}

	return pelanggaran
}

// itoa mengubah bilangan kecil menjadi teks tanpa menarik strconv ke lapisan domain
// hanya untuk satu pesan.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var angka []byte
	for n > 0 {
		angka = append([]byte{byte('0' + n%10)}, angka...)
		n /= 10
	}
	return string(angka)
}
