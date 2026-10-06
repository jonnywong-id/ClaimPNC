package inboxacceptopenprotection

import "time"

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
// kejadian, dan jam yang ikut tersimpan akan membuat perbandingan tanggal pada aturan
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

// truncateToDay membuang komponen jam, meniru pemotongan `@substring(...,0,8)` di Pega.
//
// Zona waktunya DIPERTAHANKAN: tanggal yang dipilih pengguna adalah tanggal WIB, dan
// memindahkannya ke UTC lebih dulu akan menggeser sebagian tanggal mundur satu hari
// (`R-12`).
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
