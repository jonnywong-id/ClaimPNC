package inboxacceptopenprotection

import (
	"strings"
	"time"
)

// Penerapan perubahan yang disetujui ke DATA KLAIM.
//
// # Kenapa akseptasi harus menyentuh klaim sama sekali
//
// Menyetujui permintaan tipe '7' berarti menyetujui **perubahan Tanggal Kejadian klaim**.
// Bila keputusannya tersimpan tetapi klaimnya tidak berubah, persetujuan itu tidak
// menimbulkan akibat apa pun — dan tidak ada gejala yang menandainya.
//
// Pega melakukannya di `Activity/InsertOpenProtectionCase-Act.xml`:
//
//	TempPNCOPEN.ClaimData.DateOfLoss <- @substring(.ClaimDataProtect.DateOfLoss,0,8)+"T000000.000 GMT"
//
// `TempPNCOPEN` adalah work object klaim. Sasarannya di sistem baru ditetapkan Work Owner
// 2026-09-26, setelah sempat keliru mengarah ke tabel datar:
//
//	"tabel t_claimlist_admin hanya untuk dashboard, jadi datanya tidak diganti-ganti.
//	 untuk perubahan data dari open proteksi, update data di t_claim_pnc saja."
//
// Jadi yang diubah adalah **`POOLDATA.T_CLAIM_PNC`**. `T_CLAIMLIST_ADMIN` tetap tabel BACA
// untuk dashboard dan daftar; menulisinya akan membuat dua sumber kebenaran untuk hal yang
// sama, dan yang satu akan menyimpang dari yang lain tanpa gejala.
//
// # Kuncinya CLAIMID, bukan nomor klaim
//
// `T_CLAIM_PNC` dikunci `CLAIMID` — bagi klaim warisan berbentuk
// `ASM-FW-GCNMFW-WORK PNC-1865`. Nilai itu tersimpan pada `T_CLAIM_OPENPROTECTION.ID_CLAIM`
// dan dibawa `Protection.ClaimReference`.
//
// # Tanggalnya DIPOTONG ke tengah malam, dan itu ditiru
//
// `@substring(...,0,8)` memotong bentuk `YYYYMMDD`, lalu menempelkan `T000000.000 GMT`.
// Jadi yang tersimpan adalah TANGGAL, tanpa jam. Ditiru apa adanya: DOL adalah tanggal
// kejadian, dan jam yang ikut tersimpan akan membuat perbandingan "DOL + 7 hari" pada aturan
// registrasi bergeser tanpa terlihat.

// LossDateToApply menyatakan tanggal kejadian baru yang harus diterapkan ke klaim, bila ada.
//
// Mengembalikan `false` bila keputusan ini tidak mengubah klaim — dan itu keadaan yang
// paling sering: hanya tipe '7' yang mengubah DOL.
//
// # Ditolak TIDAK mengubah klaim, dan itu SELISIH dari Pega
//
// Di `InsertOpenProtectionCase`, perubahan DOL **tidak dijaga status persetujuan sama
// sekali**: `Local.acceptstatus` hanya dipakai sekali, untuk menyusun teks catatan
// (`@if(acceptstatus=="1","Pengajuan Diaksep oleh Manager","Pengajuan Direject Manager")`).
// Perubahan DOL-nya sendiri hanya berprakondisi `Local.typeprotection=="7"`.
//
// Dibaca apa adanya, itu berarti **menolak permintaan perubahan DOL tetap mengubah DOL
// klaim** — permintaan yang ditolak tetap terjadi. Perilaku itu TIDAK dibawa.
//
// Ini selisih terencana yang wajib dinyatakan di muka pada uji kesetaraan (`P-5`), dan ia
// segolongan dengan `R-19`: cacat pada aturan yang mengubah data klaim tidak boleh
// direplikasi hanya karena ia ada di sistem lama.
func LossDateToApply(p Protection, d Decision) (time.Time, bool) {
	if d != DecisionApprove {
		return time.Time{}, false
	}
	if !ShowsChangeDetail(p.Type) || p.Type != TypeChangeLossDate {
		return time.Time{}, false
	}
	if p.Change.LossDateAfter == nil {
		// Baris warisan Pega tidak punya kolom asal untuk OLD_DATA/NEW_DATA, sehingga
		// tanggal barunya kosong. Menyetujuinya tetap sah — yang tidak terjadi hanyalah
		// perubahan DOL, karena tidak ada nilai yang hendak diterapkan.
		return time.Time{}, false
	}

	return truncateToDay(*p.Change.LossDateAfter), true
}

// CauseOfLossChange adalah perubahan Penyebab Kerugian yang harus diterapkan ke SATU baris
// coverage klaim.
//
// Ketiganya dibawa bersama karena ketiganya tidak berarti apa-apa sendiri-sendiri: kode baru
// tanpa sasaran tidak dapat dituliskan, dan sasaran tanpa kode baru tidak mengubah apa pun.
type CauseOfLossChange struct {
	// ObjectID dan ObjectCoverageID adalah kunci barisnya di
	// `POOLDATA.T_CLAIM_OBJECTCOVERAGE`, bersama CLAIMID klaimnya.
	ObjectID         string
	ObjectCoverageID string

	// CauseOfLossID adalah `D_COL_ID` yang diminta berlaku — nilai yang dipilih pemohon dari
	// dropdown "Next Cause Of Loss", tersimpan apa adanya di `NEW_DATA`.
	//
	// Deskripsinya TIDAK ikut di sini: ia diambil dari master saat penerapan, bukan dibawa
	// dari permintaan. Permintaan yang dibuat bulan lalu lalu disetujui hari ini harus
	// menuliskan teks yang berlaku HARI INI, bukan teks yang kebetulan tersimpan saat itu.
	CauseOfLossID string
}

// CauseOfLossToApply menyatakan perubahan Penyebab Kerugian yang harus diterapkan, bila ada.
//
// Mengembalikan `false` bila keputusan ini tidak mengubah coverage mana pun — dan itu keadaan
// yang paling sering: hanya tipe '8' yang mengubahnya.
//
// # Ditolak TIDAK mengubah klaim
//
// Sama dengan LossDateToApply, dan atas sebab yang sama. `InsertOpenProtectionCase` menjaga
// perubahan tipe '8' hanya dengan prakondisi `Local.typeprotection=="8"`, tanpa memeriksa
// status persetujuan sama sekali — dibaca apa adanya, MENOLAK permintaan tetap mengubah
// Penyebab Kerugian klaim.
//
// Perilaku itu TIDAK dibawa. Ia selisih terencana yang wajib dinyatakan di muka pada uji
// kesetaraan (`P-5`), segolongan dengan `R-19`.
//
// # Tiga keadaan yang menghasilkan `false`, dan kenapa ketiganya BUKAN galat
//
// Permintaan tipe '8' dapat sampai ke sini tanpa kode baru, atau tanpa sasaran, atau tanpa
// keduanya. Seluruhnya terjadi pada BARIS WARISAN PEGA: `OLD_DATA`, `NEW_DATA`, `OBJECT_ID`,
// dan `OBJECT_COVERAGE_ID` semuanya kolom yang baru ada di sistem ini, dan baris warisan
// menyimpan sasarannya di dalam blob properti work object.
//
// Menolak menyetujuinya akan membuat seluruh antrean warisan tipe '8' TIDAK DAPAT diputuskan
// siapa pun — padahal persetujuannya sendiri tetap bermakna. Yang tidak terjadi hanyalah
// perubahan coverage-nya, karena memang tidak ada yang hendak diterapkan.
//
// Ini BERBEDA dari sasaran yang ada tetapi barisnya tidak ketemu: yang itu ditangani adapter
// sebagai ErrClaimNotSynced, karena di sana ada perubahan yang hendak diterapkan dan ia
// gagal.
func CauseOfLossToApply(p Protection, d Decision) (CauseOfLossChange, bool) {
	if d != DecisionApprove {
		return CauseOfLossChange{}, false
	}
	if !ShowsChangeDetail(p.Type) || p.Type != TypeChangeCauseOfLoss {
		return CauseOfLossChange{}, false
	}

	perubahan := CauseOfLossChange{
		ObjectID:         strings.TrimSpace(p.Change.ObjectID),
		ObjectCoverageID: strings.TrimSpace(p.Change.ObjectCoverageID),
		CauseOfLossID:    strings.TrimSpace(p.Change.CauseOfLossAfter),
	}
	if perubahan.CauseOfLossID == "" ||
		perubahan.ObjectID == "" || perubahan.ObjectCoverageID == "" {
		return CauseOfLossChange{}, false
	}

	return perubahan, true
}

// truncateToDay membuang komponen jam, meniru pemotongan `@substring(...,0,8)` di Pega.
//
// Zona waktunya DIPERTAHANKAN: tanggal yang dipilih pengguna adalah tanggal WIB, dan
// memindahkannya ke UTC lebih dulu akan menggeser sebagian tanggal mundur satu hari
// (`R-12`).
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
