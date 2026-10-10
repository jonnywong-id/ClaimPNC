package sqlstore

// Penomoran Post Audit — kolom `CASEID` pada `POOLDATA.T_CLAIM_COMPLIANCE_H`.
//
// # Bentuknya `CPL.YY.xxxx`, dan nomornya dirakit BASIS DATA
//
// Work Owner menetapkan sintaksnya secara harfiah pada 2026-10-06:
//
//	'CPL' || '.' || TO_CHAR(SYSDATE,'RR') || '.'
//	       || TO_CHAR(POOLDATA.CLAIM_COMPLIENCE_SEQ.NEXTVAL)
//
// Hasilnya `CPL.26.1`, sebentuk dengan nomor klaim `PNCN.YY.xxxx` (`D-71`) dan nomor
// laporan `LPK.YY.xxxx`.
//
// Karena seluruhnya dirakit kueri `post_audit_next_sequence`, berkas ini TIDAK LAGI punya
// perakit nomor. Fungsi `BuildCaseID` yang dulu ada di sini dihapus, bukan dibiarkan
// menganggur: perakit kedua yang tidak dipanggil siapa pun adalah tempat paling mudah bagi
// bentuk nomor untuk diam-diam bercabang dua.
//
// # Keputusan 2026-09-24 yang dicabut
//
// Bentuk sebelumnya `CPL-100001` — meniru bentuk Pega (`CPL-1` … `CPL-19`), dengan
// pemisahan terbitan lama dan baru lewat RENTANG ANGKA, bukan lewat bentuk. Tiga
// konsekuensi yang tercatat panjang saat itu ikut tercabut; rinciannya di
// `migrations/0011_post_audit_compliance.up.sql`, dan yang terpenting:
//
//   - asal nomor KINI terbaca dari bentuknya — titik versus tanda hubung pada karakter
//     keempat, sehingga tidak perlu lagi tabel pemetaan maupun penjagaan rentang;
//   - nomor baru KINI tampil paling atas pada pengurutan teks menurun, karena dalam
//     himpunan karakter Oracle `-` berada sebelum `.`.
//
// # Satu masalah yang TETAP ada
//
// `TO_CHAR(...NEXTVAL)` tanpa format mask tidak memberi nol di depan, sehingga di dalam
// bentuk barunya sendiri urutan teks masih tidak sama dengan urutan penerbitan:
// `CPL.26.10` berada di atas `CPL.26.9`.
//
// Sintaks Work Owner dipakai apa adanya, sama seperti `D-71` memperlakukan sintaks nomor
// klaim. Bila lebar tetap kelak dikehendaki, yang berubah hanya kueri itu
// (`TO_CHAR(seq.NEXTVAL,'FM0000')`) — bukan DDL, dan bukan berkas ini.

// Pemilik dan nama sequence penomoran Post Audit.
//
// Keduanya konstanta, bukan ditanam di dalam teks SQL, karena dipakai kueri
// `check_post_audit_sequence` yang memeriksa keberadaannya. Bila pemeriksa boleh menyebut
// nama sendiri, `-periksa` dapat melaporkan hijau atas sequence yang bukan yang dipakai
// jalur tulis.
//
// Nilainya WAJIB sama dengan `migrations/0011_post_audit_compliance.up.sql`. Huruf besar
// disengaja: `ALL_SEQUENCES` menyimpan nama identifier tanpa tanda kutip dalam huruf
// besar, sehingga pencarian dengan huruf kecil tidak akan menemukan apa pun.
//
// Ejaan `COMPLIENCE` sengaja dipertahankan — ejaan Work Owner, dan sejalan dengan sistem
// lama yang access group-nya pun bernama `PncComplience`.
const (
	sequenceOwner = "POOLDATA"
	sequenceName  = "CLAIM_COMPLIENCE_SEQ"
)
