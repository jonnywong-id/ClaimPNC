package inboxmanageradmin

import "strings"

// Nilai `PYSTATUSWORK` sebagaimana tersimpan di `POOLDATA.T_CLAIMLIST_ADMIN`.
//
// Dua di antaranya MESTINYA tidak pernah terbaca layar ini: kuerinya sudah menyaring
// `PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')`. Keduanya tetap
// diturunkan lengkap, dan itu bukan kelebihan — lihat catatan jebakan di DisplayStatusFor.
const (
	workStatusRunning   = "New"
	workStatusCompleted = "Resolved-Completed"
	workStatusRejected  = "Resolved-Rejected"
)

// Label Status Klaim yang DILIHAT pengguna pada berkas ekspor.
//
// Berbahasa Inggris karena itulah yang tertulis di layar lama, dan `D-13` menetapkan teks
// yang dilihat pengguna mengikuti Pega apa adanya.
const (
	DisplayOnProgress = "On Progress"
	DisplayClose      = "Close"
	DisplayReject     = "Reject"
)

// DisplayStatusFor menurunkan label Status Klaim dari `PYSTATUSWORK`.
//
// # Dari mana pemetaannya
//
// `RDB List/BrowseClaimALL-SQL.xml`, yang menurunkan hal yang sama di dalam SQL:
//
//	WHEN a.pystatuswork = 'New'                THEN 'On Progress'
//	WHEN a.pystatuswork = 'Resolved-Rejected'  THEN 'Reject'
//	WHEN a.pystatuswork = 'Resolved-Completed' THEN 'Close'
//
// Modul Inbox Outstanding menurunkannya dengan cara yang sama
// (`inboxoutstanding.DisplayStatus`). Pemetaannya DISALIN, bukan diimpor: modul tidak saling
// mengimpor domainnya, dan tiga baris yang sama lebih murah daripada kopling antarmodul.
//
// # Kenapa TIDAK ada cabang default yang menghasilkan "On Progress"
//
// Karena cabang seperti itu sudah pernah menggigit, dan dicatat di
// `inboxoutstanding.go`: nilai yang tidak dikenali — termasuk `'Resolved-Completed'` —
// jatuh ke default dan berlabel **"On Progress" untuk klaim yang sudah ditutup**. Kesalahan
// itu tidak menghasilkan galat apa pun; yang membongkarnya satu baris data produksi.
//
// Nilai yang tidak dikenali karena itu dikembalikan APA ADANYA. Status asing yang menyamar
// sebagai "On Progress" tidak pernah ditanyakan siapa pun; status asing yang tampil apa
// adanya ditanyakan pada hari pertama.
func DisplayStatusFor(workStatus string) string {
	switch strings.TrimSpace(workStatus) {
	case workStatusRunning:
		return DisplayOnProgress
	case workStatusCompleted:
		return DisplayClose
	case workStatusRejected:
		return DisplayReject
	default:
		return strings.TrimSpace(workStatus)
	}
}
