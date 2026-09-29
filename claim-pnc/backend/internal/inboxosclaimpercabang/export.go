package inboxosclaimpercabang

import "claim-pnc/internal/platform/money"

// Berkas ini memuat bentuk baris BERKAS EKSPOR, yang lebih lebar daripada baris grid.
//
// # Kenapa ia tipe tersendiri, bukan WorkItem yang diperlebar
//
// Karena keduanya dilayani kueri yang berbeda, dan salah satu perbedaannya menyangkut uang:
// kolom nilai pada grid dihitung tanpa kurs dan tanpa porsi ASM, sedangkan pada berkas ekspor
// dihitung dengan keduanya (lihat WorkItem.EstimationValue). Menyatukannya ke satu tipe akan
// membuat satu field membawa dua arti tergantung siapa yang mengisinya — persis kelas cacat
// yang penamaan ulang `D-19` ada untuk mencegahnya.
//
// Perbedaan kedua lebih sederhana: 24 kolom pembagian treaty menuntut gabungan lewat DB Link
// `@asmd`, dan grid tidak boleh membayar biaya itu pada setiap kali layar dibuka.

// ExportRow adalah satu baris berkas ekspor.
//
// Ia MEMUAT WorkItem apa adanya, lalu menambahkan kolom yang hanya ada di berkas. Susunan itu
// bukan kenyamanan: ia yang membuat 19 kolom pertama berkas dijamin sama dengan 19 kolom
// pertama grid, sehingga berkas dan layar selalu dapat dicocokkan baris per baris.
type ExportRow struct {
	WorkItem

	// ClaimKey adalah kunci internal klaim — `T_CLAIM_PNC.CLAIMID`.
	//
	// Ia TIDAK ditulis ke berkas. Ia ada karena faktor dominan dipasangkan lewat kunci ini,
	// bukan lewat nomor klaim: `CLAIMID` unik (1.922 nilai berbeda untuk 1.922 baris),
	// sedangkan `CLAIMNO` TIDAK (1.515 nilai berbeda). Memasangkannya lewat nomor klaim akan
	// menggabungkan faktor milik dua klaim yang bernomor sama — terverifikasi ke basis data.
	ClaimKey string

	// PolicyBusinessName adalah nama bisnis pada POLIS — `t_general.businessname`, dialias
	// `District` oleh kueri lama.
	//
	// Nama alias itu tidak dibawa: isinya bukan distrik apa pun. Ia diambil dari perpanjangan
	// polis TERBARU, yaitu `prodke` terbesar sebagai ANGKA.
	//
	// Perhatikan ia BERBEDA dari WorkItem.BusinessName, yang merupakan lini bisnis hasil
	// terjemahan `grouppanel`. Keduanya berdampingan di berkas dan mudah tertukar.
	PolicyBusinessName string

	// InsuredName adalah nama tertanggung — `t_general.theinsured`, dari perpanjangan polis
	// yang sama dengan PolicyBusinessName.
	//
	// Kedua subkueri WAJIB memakai urutan yang identik. Mengubah salah satunya akan
	// memasangkan nama bisnis satu perpanjangan dengan nama tertanggung perpanjangan lain,
	// dan tidak ada satu pun gejala yang menandainya.
	InsuredName string

	// ReserveClaimFull adalah jumlah estimasi apa adanya — `SUM(estimationvalue)`, tanpa
	// kurs dan tanpa porsi ASM.
	//
	// Ia bernilai SAMA dengan WorkItem.EstimationValue pada berkas ekspor, dan keduanya
	// memang dipilih dua kali oleh kueri lama (`ReserveClaimFull` dan kolom grid). Keduanya
	// dipertahankan supaya susunan berkasnya tidak berubah.
	ReserveClaimFull money.Money

	// ReserveClaimASM adalah porsi ASM — `SUM(estimationvalue * kursvalue) * SHAREASM/100`.
	//
	// INILAH nilai yang judul kolom grid janjikan tetapi tidak tampilkan. Lihat
	// WorkItem.EstimationValue.
	ReserveClaimASM money.Money

	// Coinsurance adalah porsi penanggung lain —
	// `SUM(estimationvalue * kursvalue) * (100 - SHAREASM)/100`.
	Coinsurance money.Money

	// TreatyShares adalah ke-24 pembagian treaty, masing-masing porsi ASM dikalikan bagian
	// treaty yang bersangkutan.
	TreatyShares TreatyShares

	// DominantFactors adalah nama faktor dominan klaim, dirangkai dengan koma.
	//
	// Kueri lama merangkainya dengan `LISTAGG` dan memberi `'-'` bila kosong. Perangkaiannya
	// pindah ke Go karena tidak ada bentuk agregasi teks yang berjalan di Oracle 19c maupun
	// PostgreSQL 17+; tanda `'-'` untuk yang kosong dipertahankan.
	DominantFactors string
}

// TreatyShares adalah ke-24 pembagian treaty pada satu baris berkas ekspor.
//
// Seluruhnya berasal dari `treaty_loss@asmd.sinarmas.co.id`, diambil dari baris ber-`no_spk`
// terbesar untuk nomor klaim itu. Nama field mengikuti nama kolom sumbernya tanpa awalan
// `CLAIM_`, karena awalan itu tidak membedakan apa pun — seluruh kolomnya memang tentang klaim.
//
// # Kenapa 24 field bernama, bukan peta
//
// Karena urutannya adalah bagian dari kontrak berkas, dan peta tidak punya urutan. Susunan
// kolomnya dijamin ExportTreatyColumns dan TreatyValues yang berdampingan di bawah, beserta
// uji yang memastikan keduanya tidak pernah berbeda panjang.
type TreatyShares struct {
	OR        money.Money
	FacOut    money.Money
	FacOB     money.Money
	QS        money.Money
	FSPL      money.Money
	SSPL      money.Money
	ER1       money.Money
	ER2       money.Money
	BPPDAN    money.Money
	PSRQS     money.Money
	PSRSPL    money.Money
	ORS       money.Money
	XL        money.Money
	PSROR     money.Money
	QSOR      money.Money
	PSS       money.Money
	PRGBI     money.Money
	PFRA      money.Money
	FSPLNSRI  money.Money
	PSPLNSRI  money.Money
	FSPLNSOR  money.Money
	PSPLNSOR  money.Money
	FacOBSRB  money.Money
	FacOBIndt money.Money
}

// ExportTreatyColumns adalah judul ke-24 kolom treaty, dalam urutan berkas.
//
// Judulnya diambil APA ADANYA dari alias kueri lama — `OR`, `FACOUT`, `FACOB`, dan seterusnya
// — karena itulah yang selama ini dibaca penerima berkas. Menggantinya dengan nama yang lebih
// jelas akan membuat berkas baru tidak dapat ditempelkan ke lembar kerja yang sudah ada.
var ExportTreatyColumns = []string{
	"OR", "FACOUT", "FACOB", "QS", "FSPL", "SSPL", "ER1", "ER2",
	"BPPDAN", "PSRQS", "PSRSPL", "ORS", "XL", "PSROR", "QSOR", "PSS",
	"PRGBI", "PFRA", "FSPLNSRI", "PSPLNSRI", "FSPLNSOR", "PSPLNSOR",
	"FACOBSRB", "FACOBINDT",
}

// TreatyValues mengembalikan ke-24 nilai dalam urutan yang sama dengan ExportTreatyColumns.
//
// Keduanya sengaja berdampingan di berkas ini dan diuji bersama: judul dan nilai yang
// disimpan di dua tempat yang berjauhan adalah tempat paling mudah bergeser satu kolom, dan
// pergeseran satu kolom pada berkas berisi angka uang tidak menghasilkan satu pun galat.
func (t TreatyShares) TreatyValues() []money.Money {
	return []money.Money{
		t.OR, t.FacOut, t.FacOB, t.QS, t.FSPL, t.SSPL, t.ER1, t.ER2,
		t.BPPDAN, t.PSRQS, t.PSRSPL, t.ORS, t.XL, t.PSROR, t.QSOR, t.PSS,
		t.PRGBI, t.PFRA, t.FSPLNSRI, t.PSPLNSRI, t.FSPLNSOR, t.PSPLNSOR,
		t.FacOBSRB, t.FacOBIndt,
	}
}

// ExportPage adalah satu halaman baris ekspor beserta jumlah seluruh baris yang cocok.
type ExportPage struct {
	Items []ExportRow

	// Total adalah jumlah SELURUH baris yang cocok, bukan yang ada di halaman ini.
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan.
	Pagination Pagination
}
