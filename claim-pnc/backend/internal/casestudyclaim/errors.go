package casestudyclaim

import (
	"errors"
	"fmt"
	"strings"
)

// Galat domain modul Case Study Claim.
//
// Seluruhnya TIPE, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan
// domain adalah tipe, bukan string).
var (
	// ErrPeriodRequired berarti rentang tahun tidak terisi.
	//
	// # Kenapa ia GAGAL, padahal Pega diam saja
	//
	// Kueri lama membandingkan `A.THNREGIS BETWEEN TO_CHAR(TO_DATE(awal),'yyyy') AND …`.
	// Bila salah satu isiannya kosong, `TO_DATE` menghasilkan NULL, `BETWEEN` menghasilkan
	// NULL, dan kueri mengembalikan **nol baris** — tanpa satu pun pesan.
	//
	// Menolaknya karena itu TIDAK mengubah satu baris pun yang tampil: hasilnya kosong
	// pada kedua sistem. Yang berubah hanyalah pengguna diberi tahu sebabnya, alih-alih
	// menatap grid kosong dan menyimpulkan tidak ada klaim besar pada periode itu.
	//
	// Selisih ini disengaja, tidak mengubah data, dan dicatat supaya tidak terbaca sebagai
	// cacat saat uji kesetaraan `S-8` dijalankan.
	ErrPeriodRequired = errors.New("casestudyclaim: rentang periode wajib diisi")

	// ErrPeriodReversed berarti tahun awal melewati tahun akhir.
	//
	// Di Pega ini pun menghasilkan nol baris tanpa pesan — `BETWEEN` dengan batas
	// terbalik tidak pernah benar. Alasan menolaknya sama dengan di atas.
	ErrPeriodReversed = errors.New("casestudyclaim: tahun awal melewati tahun akhir")

	// ErrUnknownBusiness berarti kode bisnis yang diminta tidak dikenal.
	ErrUnknownBusiness = errors.New("casestudyclaim: pilihan bisnis tidak dikenal")

	// ErrUnknownStatus berarti kode status yang diminta tidak dikenal.
	ErrUnknownStatus = errors.New("casestudyclaim: pilihan status tidak dikenal")

	// ErrClaimRequired berarti nomor klaim yang catatannya hendak disimpan tidak disebut.
	ErrClaimRequired = errors.New("casestudyclaim: nomor klaim wajib disebut")

	// ErrClaimNotFound berarti nomor klaimnya tidak ada.
	//
	// Ia BUKAN kegagalan sistem. Baris dapat hilang di antara saat daftar dibaca dan saat
	// Save ditekan — klaimnya ditutup, nomornya berubah, atau penyaringnya tidak lagi
	// mencakupnya. Yang pantas dilakukan pengguna adalah memuat ulang daftarnya, dan
	// pesan "terjadi kesalahan sistem" justru menyuruhnya menghubungi tim teknis.
	ErrClaimNotFound = errors.New("casestudyclaim: nomor klaim tidak ditemukan")

	// ErrRemarkTooLong berarti catatan melebihi batas kolomnya.
	ErrRemarkTooLong = errors.New("casestudyclaim: catatan melebihi batas panjang")

	// ErrRemarkRejectedByColumn berarti basis data MENOLAK catatan karena kolomnya lebih
	// sempit daripada batas yang diperiksa aplikasi.
	//
	// # Kenapa ia galat tersendiri, bukan dilebur ke ErrRemarkTooLong
	//
	// Keduanya sama-sama "terlalu panjang", tetapi yang satu diketahui SEBELUM menyentuh
	// basis data dan yang lain hanya diketahui SESUDAHNYA. Pembedaan itu membuat jejaknya
	// berguna: kemunculan galat ini berarti MaxRemarkLength lebih longgar daripada lebar
	// kolom yang sebenarnya — bukti langsung bahwa dugaan di sana meleset, dan
	// satu-satunya cara mengetahuinya selama DDL belum diterima (`R-08`).
	//
	// Work Owner memutuskan pada 2026-09-26 untuk TIDAK mengejar DDL-nya dan berjalan
	// seperti Pega. Galat ini yang membuat keputusan itu aman: pengguna tetap diberi tahu
	// bahwa catatannya terlalu panjang, bukan "terjadi kesalahan pada sistem".
	ErrRemarkRejectedByColumn = errors.New(
		"casestudyclaim: catatan ditolak basis data karena melebihi lebar kolom")
)

// MaxRemarkLength membatasi panjang catatan telaah.
//
// # Angkanya BELUM berasal dari DDL
//
// `POOLDATA.T_CLAIM_PNC.REMARKRECOMENDATION` belum pernah diterima definisinya (`R-08`),
// sehingga lebar sebenarnya tidak diketahui. Angka di bawah adalah batas pengaman yang
// dipilih sadar — cukup longgar untuk catatan telaah yang panjang, cukup ketat untuk
// menolak kiriman yang jelas akan ditolak basis data.
//
// Memilih untuk TIDAK membatasi sama sekali berarti galat ORA-12899 sampai ke pengguna
// sebagai "terjadi kesalahan sistem" setelah ia mengetik satu halaman penuh. Itu lebih
// buruk daripada batas yang mungkin sedikit meleset.
//
// Begitu DDL-nya tiba, angka ini disesuaikan — dan hanya angka ini.
const MaxRemarkLength = 2000

// ValidateRemark memeriksa isian catatan telaah sebelum disimpan.
//
// Catatan KOSONG diterima, dan itu disengaja: mengosongkan kembali catatan yang salah
// ketik adalah hal yang wajar, dan kolomnya nullable. Pega pun tidak memeriksanya —
// `SaveRemarksRecommendation_act` meneruskan apa pun yang ada di sel.
func ValidateRemark(claimNumber, remark string) error {
	if strings.TrimSpace(claimNumber) == "" {
		return ErrClaimRequired
	}
	if len([]rune(remark)) > MaxRemarkLength {
		return fmt.Errorf("%w: %d dari %d karakter",
			ErrRemarkTooLong, len([]rune(remark)), MaxRemarkLength)
	}
	return nil
}

// ValidatePeriod memeriksa rentang tahun sebelum kueri dijalankan.
//
// Keduanya dibandingkan sebagai TEKS, sama seperti kueri lama membandingkan `A.THNREGIS`
// terhadap keluaran `TO_CHAR(…,'yyyy')`. Pada tahun empat digit, urutan teks dan urutan
// angka identik — dan memakai perbandingan yang sama dengan kueri membuat pemeriksaan ini
// tidak dapat menyimpang dari penyaringnya.
func ValidatePeriod(fromYear, toYear string) error {
	from := strings.TrimSpace(fromYear)
	to := strings.TrimSpace(toYear)

	if from == "" || to == "" {
		return ErrPeriodRequired
	}
	if from > to {
		return ErrPeriodReversed
	}
	return nil
}
