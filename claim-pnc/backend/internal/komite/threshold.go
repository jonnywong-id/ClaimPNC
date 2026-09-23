// Package komite adalah inti modul Penjenjangan Komite (`B-7`).
//
// Nama paketnya berbahasa Indonesia karena ia **nama modul bisnis** (`D-81`); seluruh isinya
// berbahasa Inggris (`D-80`).
//
// # Apa yang dikerjakan modul ini
//
// Menjawab satu pertanyaan: untuk sebuah nilai klaim pada sebuah lini bisnis, SIAPA SAJA
// yang harus menyetujui, dan dalam urutan apa. Ini menentukan siapa berwenang menyetujui
// uang — salah hitung ke bawah berarti klaim besar disetujui terlalu sedikit orang,
// salah ke atas berarti klaim kecil tertahan tanpa alasan.
//
// # Aturannya KUMULATIF, dan ini bagian yang paling mudah disalahpahami
//
// Bukan memilih satu jenjang dari sebuah matriks. Melainkan: SETIAP jenjang yang ambang
// bawahnya sudah terlampaui nilai klaim ikut menyetujui.
//
//	jumlah jenjang = jumlah baris master yang LIMIT_BOTTOM <= nilai klaim
//
// Buktinya ada di dua tempat yang saling menguatkan:
//
//   - `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` menetapkan
//     `KomiteLoop := TempRDBSearchEmailKomite.pxResultCount` — jumlah jenjang SAMA
//     DENGAN jumlah baris yang dikembalikan kueri.
//   - `When/IsKomiteLoop-When.xml` memutar alurnya selama
//     `.AcceptStatus = "1"` DAN `.KomiteCount <= .KomiteLoop`.
//
// Dan penyaring kuerinya hanya batas bawah: `LIMIT_BOTTOM` difilter di 11 rule SQL,
// `LIMIT_TOP` di NOL rule (`ADR-0014`).
//
// # Dua kesalahan yang nyaris diambil, dan kenapa keduanya merusak
//
// Pertama, mengganti penyaring menjadi rentang tertutup
// `LIMIT_BOTTOM <= nilai <= LIMIT_TOP`. Itu akan mengembalikan TEPAT SATU baris,
// sehingga `KomiteLoop = 1` dan setiap klaim hanya butuh SATU persetujuan berapa pun
// nilainya — penjenjangan yang menjadi inti `D-14` hilang seluruhnya (`D-47`).
//
// Kedua, memberlakukan pemilihan pita ke semua lini. Dihitung dari master yang berlaku:
// PA Rp 5.000.000 menjadi NOL penyetuju dan Travel Rp 150.000.000 menjadi NOL
// penyetuju — klaimnya mandek. Itu sebabnya `D-70` membatasi pita HANYA ke Non-MBU.
//
// # Yang sengaja TIDAK ada di sini
//
// Modul ini MEMBACA master ambang, tidak memilikinya — pengelolaannya milik `F-4`.
// Layar keputusan komite, pencatatan setuju/tolak/kembalikan, dan perpindahan antar
// jenjang adalah `TKT-B07-002`, yang bergantung pada `B-5` dan `B-6` dan belum dapat
// dikerjakan. Persetujuan otomatis `AutoAcceptKomite` adalah `TKT-B07-003` dan
// diputuskan TIDAK dibawa (Work Owner, 2026-09-17).
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package komite

import (
	"context"
	"strings"

	"claim-pnc/internal/platform/money"
)

// BusinessLine adalah lini bisnis yang dipakai master ambang untuk mengelompokkan jenjang.
//
// Ia memetakan langsung ke kolom TYPE_BUSINESS pada POOLDATA.EMAILKOMITE. Nilainya
// TIDAK didaftar sebagai konstanta tertutup di sini: `D-15` menetapkan tidak ada nilai
// bisnis yang boleh di-hardcode, dan daftar lini yang berlaku adalah apa pun yang ada di
// master — hari ini NONMBU, NONMBUAB, NONMBUC, PA, TRAVEL, dan BONDING, besok mungkin
// bertambah.
//
// Konstanta di bawah hanyalah nama untuk lini yang perilakunya BERBEDA dan karena itu
// harus disebut namanya di dalam kode.
type BusinessLine string

// BusinessLineNonMBU adalah satu-satunya lini yang memakai pemilihan pita nilai.
//
// Ia disebut namanya karena `D-70` menetapkan perlakuannya berbeda dari lini lain, bukan
// karena daftar lini di-hardcode. Lini lain tidak perlu dikenali sama sekali — mesin
// penjenjangan memperlakukan semuanya sama.
const BusinessLineNonMBU BusinessLine = "NONMBU"

// Normalized menormalkan penulisan lini.
//
// Perapian ini nyata gunanya. Isi master tersimpan dengan penulisan yang tidak konsisten
// — `docs/Steering/11-SECURITY.md` §3.1 mencatat tiga nama access group Pega muncul
// dalam dua kapitalisasi berbeda karena perbandingan rule lama tidak konsisten soal itu.
// Nilai yang datang dari layar pun mengikuti apa yang diketik pengguna.
func (l BusinessLine) Normalized() BusinessLine {
	return BusinessLine(strings.ToUpper(strings.TrimSpace(string(l))))
}

// Threshold adalah SATU BARIS master ambang komite — satu jenjang persetujuan untuk satu
// lini bisnis.
//
// Nama field memakai padanan Inggris dari `CONTEXT.md`, bukan nama kolom aslinya;
// pemetaannya ada di repo/sqlstore, satu tempat saja.
type Threshold struct {
	// ID adalah kunci baris di master. Dipakai sebagai pemecah seri saat dua baris
	// punya Tier yang sama — lihat catatan pada Determine.
	ID string

	// Name adalah nama penyetuju sebagaimana tertulis di master (kolom NAME).
	Name string

	// OperatorID adalah identitas penyetuju di sistem lama (kolom OPERATOR_ID).
	//
	// Inilah yang membuat penugasan komite berbeda dari tahap lain: komite ditugaskan
	// ke OPERATOR BERNAMA, bukan ke workbasket bersama (`T-7`). Akibatnya
	// ketidakhadiran seseorang dapat menghentikan klaim — lihat Absent.
	OperatorID string

	// BusinessLine adalah lini bisnis baris ini berlaku (kolom TYPE_BUSINESS).
	BusinessLine BusinessLine

	// CommitteeType adalah kolom TYPE_KOMITE APA ADANYA, dan kolom itu MEMIKUL DUA ARTI
	// yang berbeda:
	//
	//   - Pada lini Non-MBU ia adalah PITA NILAI — "1" untuk klaim sampai
	//     Rp 100.000.000, "2" untuk di atasnya (`D-52`).
	//   - Pada lini PA ia adalah VARIAN JALUR — membedakan PA reguler dari PA TKI,
	//     dan sama sekali bukan pita nilai (`D-70`).
	//
	// Buktinya ada di isi master: pada PA nilainya berselang-seling 2 · 1 · 1 · 2
	// menaiki tangga, yang mustahil bila ia pita nilai.
	//
	// Pemisahannya menjadi dua kolom dipertimbangkan dan TIDAK diambil, sehingga arti
	// gandanya wajib didokumentasikan — inilah tempatnya.
	CommitteeType string

	// LowerBound adalah LIMIT_BOTTOM: nilai klaim minimum yang membuat jenjang ini ikut
	// menyetujui. Inilah SATU-SATUNYA penyaring nilai yang dipakai.
	LowerBound money.Money

	// UpperBound adalah LIMIT_TOP, dan ia TIDAK PERNAH dipakai memilih baris.
	//
	// `D-47` menetapkan perannya: validasi integritas master — menemukan tangga yang
	// tumpang tindih atau berlubang. Pemakaiannya ada di integrity.go, bukan di mesin
	// penjenjangan.
	UpperBound money.Money

	// Tier adalah DEGREE, yang menentukan URUTAN menyetujui.
	//
	// Ia BUKAN jumlah jenjang. Nilai DEGREE pada master hari ini adalah 1, 1, 2, 3, 4
	// untuk Non-MBU — angka 1 muncul dua kali — sementara jumlah penyetuju ditentukan
	// jumlah baris yang cocok (`pxResultCount`), bukan oleh angka ini (`D-52`).
	Tier int

	// Active adalah STS_AKTIF. Baris tidak aktif tidak pernah ikut menyetujui.
	Active bool

	// ForAdjustment adalah STS_ADJ — baris ini berlaku untuk komite atas NILAI klaim.
	//
	// Inilah penyaring yang memisahkan jenjang persetujuan dari baris berjenis lain di
	// tabel yang sama. Baris ber-STS_REG, misalnya, hanya penerima notifikasi saat
	// registrasi dan sama sekali bukan jenjang.
	ForAdjustment bool

	// ForRegistration adalah STS_REG dan ForRejection adalah STS_REJECT. Keduanya
	// dibaca supaya layar master dapat menjelaskan kenapa sebuah baris tidak muncul
	// sebagai jenjang, alih-alih membuatnya hilang tanpa keterangan.
	ForRegistration bool
	ForRejection    bool

	// Absent adalah STS_ABS, penanda penyetuju sedang tidak dapat mengerjakan.
	//
	// Perilakunya BELUM DIRUMUSKAN: master menyediakan kolomnya, tetapi tidak ada rule
	// di export yang memperlihatkan apa yang terjadi bila ia menyala (`TKT-B07-002`).
	// Karena itu modul ini MEMBAWA nilainya tanpa memakainya untuk menyaring —
	// menebaknya berarti mengarang aturan yang menentukan siapa menyetujui uang.
	Absent bool
}

// Normalized mengembalikan salinan dengan penulisan yang sudah dirapikan.
//
// Perapian ini menutup cacat data yang nyata, bukan kerapian kosmetik: pada
// `Database/emailkomite.csv` baris ID 4, isi OPERATOR_ID adalah
// "MARTENPETRUSLALAMENTIK_1" DIIKUTI BARIS BARU. Membiarkannya membuat pencocokan
// identitas penyetuju gagal tanpa satu pun pesan galat.
func (t Threshold) Normalized() Threshold {
	t.ID = strings.TrimSpace(t.ID)
	t.Name = strings.TrimSpace(t.Name)
	t.OperatorID = strings.TrimSpace(t.OperatorID)
	t.BusinessLine = t.BusinessLine.Normalized()
	t.CommitteeType = strings.TrimSpace(t.CommitteeType)
	return t
}

// IsApprovalTier menyatakan baris ini adalah jenjang persetujuan nilai klaim, bukan
// baris berjenis lain di tabel yang sama.
//
// Dua syarat pertama meniru penyaring yang dipakai SELURUH kueri penjenjangan di sistem
// lama: `STS_AKTIF = '1'` dan `STS_ADJ = '1'`.
//
// # Syarat ketiga: DEGREE bukan nol
//
// Work Owner menetapkan 2026-09-18 bahwa **baris ber-DEGREE 0 tidak dipakai**.
//
// Penyaringnya ditulis eksplisit meski hari ini tidak mengubah satu hasil pun: pada
// master yang berlaku, satu-satunya baris ber-DEGREE 0 yang masih aktif adalah ID 9, dan
// ia sudah tersaring lebih dulu karena STS_ADJ-nya kosong — ia penerima pemberitahuan
// registrasi, bukan jenjang.
//
// Menuliskannya tetap perlu, karena tanpa itu aturannya hanya BERLAKU SECARA KEBETULAN.
// Satu baris baru ber-DEGREE 0 dengan STS_ADJ menyala akan diam-diam ikut menyetujui
// uang, dan tidak ada yang akan menyadarinya.
func (t Threshold) IsApprovalTier() bool {
	return t.Active && t.ForAdjustment && t.Tier > 0
}

// OperatorKey adalah bentuk Operator ID yang dipakai membandingkan identitas.
//
// Perbandingan mengabaikan besar-kecil huruf dan spasi tepi, dan itu bukan kelonggaran:
// `docs/Steering/11-SECURITY.md` §3.1 mencatat tiga nama access group Pega muncul dalam
// dua kapitalisasi berbeda karena perbandingan rule lama tidak konsisten, dan OPERATOR_ID
// pada `Database/emailkomite.csv` baris ID 4 bahkan diakhiri BARIS BARU.
//
// Ketepatannya menentukan sesuatu yang nyata: pada mode satu-penyetuju, perbandingan
// inilah yang mengeluarkan orang yang menginput dari daftar calon penyetuju. Bila ia
// gagal, penginput dapat terpilih menyetujui pengajuannya sendiri — dan kegagalannya
// TIDAK terlihat, karena hasilnya tetap berupa nama yang masuk akal.
func OperatorKey(operator string) string {
	return strings.ToUpper(strings.TrimSpace(operator))
}

// Repo adalah seam ke master ambang komite.
//
// # Kenapa hanya satu metode, dan kenapa penyaringan TIDAK dikerjakan SQL
//
// Seluruh master berisi 30 baris dan bertambah beberapa baris per tahun; membacanya utuh
// tidak berbiaya. Yang diperoleh sebagai gantinya jauh lebih berharga: aturan
// penjenjangan — kumulatif, pemilihan pita, urutan — hidup di satu tempat sebagai fungsi
// murni yang dapat diuji TANPA basis data, dan perilakunya dijamin sama persis antara
// Oracle dan penyimpanan di memori.
//
// Menaruh penyaringan di klausa WHERE akan memecah aturan itu menjadi dua salinan yang
// dapat berbeda pendapat — persis pola yang membuat sistem lama menyebarkan satu aturan
// bisnis ke activity, SQL, dan stored procedure sekaligus.
type Repo interface {
	// ListThresholds mengembalikan SELURUH baris master apa adanya, termasuk yang tidak
	// aktif dan yang bukan jenjang persetujuan. Penyaringan adalah urusan pemanggil.
	ListThresholds(ctx context.Context) ([]Threshold, error)
}
