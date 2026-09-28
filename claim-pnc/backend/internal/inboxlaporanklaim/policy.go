package inboxlaporanklaim

import "strings"

// Policy adalah data polis yang mengisi form Input Receive Document saat Nomor Polis
// diisi.
//
// # Asalnya
//
// `Activity/PolisReceiveInternalExternal_ACT.xml`, yang dijalankan setiap kali isian
// Nomor Polis berubah (`Section/InputReceiveDocument_sect.xml`, event `change`). Datanya
// dibaca kueri `RDB List/BroswsePolisByPolicyNo-SQL.xml` dari POOLDATA.T_GENERAL, pada
// versi endorsemen terakhir polis itu (`PRODKE` terbesar).
//
// Langkah 12 activity itu mengisi lima properti, dan kelimanya dibawa di sini:
//
//	.ReceiveDocument.PolicyNo            := nomor yang sudah dirapikan
//	.ReceiveDocument.QQName              := QQNAME             -> InsuredName
//	.Policy.Quotation.BusinessCode       := BUSINESSCODE       -> BusinessCode
//	.Policy.Quotation.BusinessName       := BUSINESSNAME       -> BusinessName
//	.Policy.BookNo                       := REFNO (REGISTERID) -> ReferenceNumber
//	.Policy.Quotation.GroupPanel         := GROUPPANEL         -> GroupPanel
type Policy struct {
	Number          string
	InsuredName     string
	BusinessCode    string
	BusinessName    string
	ReferenceNumber string
	GroupPanel      string

	// Syariah adalah `SYARIAHSTATUS = 1`. Polis Syariah tidak dilaporkan lewat aplikasi
	// ini (langkah 8 dan 9).
	Syariah bool
}

// NormalizePolicyNumber merapikan nomor polis seperti langkah 2 activity lama:
// `@toUpperCase(@replaceAll(.ReceiveDocument.PolicyNo, ".", ""))`.
//
// Spasi di tepi ikut dibuang. Pega tidak melakukannya, tetapi nomor yang ditempel dari
// surel sering membawa spasi, dan dengan spasi itu kuerinya tidak akan pernah cocok.
func NormalizePolicyNumber(number string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(number), ".", ""))
}

// Kode dan pesan penolakan polis, disalin dari activity lama.
const (
	PolicyNotFound    = "polis_tidak_tersedia"
	PolicySyariah     = "polis_syariah"
	PolicyNotPNC      = "polis_bukan_pnc"
	msgPolicyNotFound = "Nomor Polis tidak tersedia"

	// Teks aslinya menyertakan alamat Pega SMAS. Alamat itu tidak ditulis di kode yang
	// di-commit (`D-69`); yang tersisa adalah instruksinya.
	msgPolicySyariah = "Polis Syariah harus registrasi klaim melalui Pega SMAS"
	msgPolicyNotPNC  = "Aplikasi ini hanya untuk pelaporan klaim PNC."
)

// nonPNCPanels adalah Group Panel yang ditolak langkah 13: 001, 007, dan 008.
var nonPNCPanels = map[string]bool{"001": true, "007": true, "008": true}

// PolicyNotice adalah satu pesan yang ditampilkan setelah polis dicari.
type PolicyNotice struct {
	Code    string
	Message string

	// Blocking berarti tombol yang menulis berkas — Simpan, Transfer, dan Register Klaim
	// — dimatikan. Di layar lama itu `pyDisabledWhen` "SyariahStatus = '1' ||
	// TempError.ErrorNotes = '1'" pada ketiga tombol.
	Blocking bool
}

// Notices menurunkan pesan untuk hasil pencarian polis.
//
// Urutannya mengikuti urutan langkah activity lama: tidak ditemukan, Syariah, lalu
// bukan PNC.
//
// # "Tidak tersedia" TIDAK memblokir
//
// Di sistem lama pesan itu hanya muncul untuk access group `PNCReportClaimInternal`
// (langkah 11), dan ia tidak menyentuh penanda yang mematikan tombol. Tabel peran belum
// ada (`TKT-F3-004`), sehingga pesannya ditampilkan untuk semua pengguna, dan tetap
// tidak memblokir: laporan boleh masuk sebelum polisnya terbit.
func Notices(policy Policy, found bool) []PolicyNotice {
	if !found {
		return []PolicyNotice{{Code: PolicyNotFound, Message: msgPolicyNotFound}}
	}
	var result []PolicyNotice
	if policy.Syariah {
		result = append(result, PolicyNotice{Code: PolicySyariah, Message: msgPolicySyariah, Blocking: true})
	}
	if nonPNCPanels[strings.TrimSpace(policy.GroupPanel)] {
		result = append(result, PolicyNotice{Code: PolicyNotPNC, Message: msgPolicyNotPNC, Blocking: true})
	}
	return result
}

// Blocked menyatakan salah satu pesan mematikan tombol tulis.
func Blocked(notice []PolicyNotice) bool {
	for _, n := range notice {
		if n.Blocking {
			return true
		}
	}
	return false
}
