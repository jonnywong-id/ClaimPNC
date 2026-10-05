package sqlstore

import "strconv"

// CaseIDPrefix dan bentuk nomor pada kolom `CASEID`.
//
// # Bentuknya MENIRU Pega, dan itu keputusan Work Owner
//
// Pega menerbitkan `CPL-1`, `CPL-15`, `CPL-19` — awalan `CPL-` diikuti angka. Keputusan
// Work Owner 2026-09-24: aplikasi baru **meniru bentuk itu**, bukan memakai tiga segmen
// bertitik seperti nomor klaim `PNCN.YY.xxxx` (`D-71`) maupun nomor laporan `LPK.YY.xxxx`
// yang dipakai modul Pelaporan Klaim.
//
// Pemisahan antara terbitan lama dan baru karena itu dilakukan lewat **rentang angka**,
// bukan lewat bentuk: sequence-nya mulai dari 100.001, sehingga nomor terbitan aplikasi
// baru selalu enam digit sedangkan terbitan Pega masih satu sampai dua digit.
//
// # Tiga konsekuensi yang diterima secara sadar
//
//  1. **Asal sebuah nomor tidak terbaca dari bentuknya.** `CPL-100001` dan `CPL-19`
//     terlihat sejenis. Ini berbeda dari nomor klaim, yang `D-22` sengaja buat dapat
//     dibedakan tanpa tabel pemetaan — dan perbedaan itu memang disadari saat memilih.
//  2. **Rentangnya harus dijaga.** Bila Pega kelak mencapai `CPL-100001`, keduanya
//     bertabrakan, dan tabelnya tidak punya constraint unik yang akan menolaknya.
//  3. **Tanpa segmen tahun.** Berbeda dari `PNCN.YY.xxxx`, nomor ini tidak menyatakan
//     tahun terbitnya. Tahun pengirimannya tetap terbaca dari kolom
//     `TGL_KIRIM_POST_AUDIT`, sehingga tidak ada keterangan yang benar-benar hilang.
//
// Bila bentuk lain kelak dikehendaki, yang berubah hanya berkas ini dan nilai `START WITH`
// pada `migrations/0011_post_audit_compliance.up.sql`.
const CaseIDPrefix = "CPL-"

// Pemilik dan nama sequence penomoran Post Audit.
//
// Keduanya ditulis sebagai konstanta, bukan ditanam di dalam teks SQL, karena dipakai dua
// tempat yang harus selalu menyebut objek yang sama: kueri `post_audit_next_sequence` yang
// memakainya, dan kueri `check_post_audit_sequence` yang memeriksa keberadaannya. Bila
// keduanya boleh menyebut nama sendiri-sendiri, `-periksa` dapat melaporkan hijau atas
// sequence yang bukan yang dipakai jalur tulis.
//
// Nilainya WAJIB sama dengan `migrations/0011_post_audit_compliance.up.sql`. Huruf besar
// disengaja: `ALL_SEQUENCES` menyimpan nama identifier tanpa tanda kutip dalam huruf
// besar, sehingga pencarian dengan huruf kecil tidak akan menemukan apa pun.
const (
	sequenceOwner = "POOLDATA"
	sequenceName  = "CPNC_POST_AUDIT_SEQ"
)

// BuildCaseID merakit nomor Post Audit dari nomor urut yang diterbitkan basis data.
//
// # Kenapa TANPA nol di depan
//
// Modul Pelaporan Klaim memakai nol di depan supaya pengurutan teksnya sesuai urutan
// penerbitan. Di sini nol itu justru akan MERUSAK urutannya.
//
// Sebabnya kolom ini sudah berisi nomor terbitan Pega yang lebarnya tidak seragam — `CPL-1`
// sampai `CPL-19` — dan tab Post Audit mengurutkannya sebagai TEKS, meniru Pega. Menambahkan
// nol di depan pada nomor baru tidak menyeragamkan apa pun terhadap baris lama, karena baris
// lama itu tidak ikut berubah.
//
// # Akibatnya pada urutan, yang TIDAK menguntungkan
//
// Dalam urutan teks menurun, nomor baru jatuh DI ANTARA baris lama, bukan di atasnya:
//
//	CPL-3 · CPL-2 · CPL-19 · CPL-15 · CPL-100002 · CPL-100001 · CPL-1
//
// Perbandingannya berhenti pada karakter keenam — `CPL-1` sama, lalu `0` dibandingkan
// dengan `9`. Petugas yang baru mengirim karena itu harus menggulir untuk menemukan
// barisnya.
//
// Dibiarkan demikian karena urutan teks itulah yang dipakai Pega, dan `D-13` menetapkan
// tampilan ditiru. Urutannya diuji di usecase/list_test.go terhadap urutan yang benar-benar
// terlihat di layar Pega.
//
// Catatan ini sempat menyatakan sebaliknya. Itu penalaran yang tidak diuji, dan
// kekeliruannya ditulis di sini supaya tidak diulang.
func BuildCaseID(sequence int64) string {
	return CaseIDPrefix + strconv.FormatInt(sequence, 10)
}
