package mastersupplier

import (
	"context"
	"strings"
	"time"
)

// # Antrean persetujuan
//
// Setiap penambahan dan setiap penyimpanan di Pega menyisipkan satu baris ke
// `pooldata.proteksi_klaimmbu` (`RDB List/InsertProteksiKlaimMBU_SQL-SQL.xml`):
//
//	INSERT INTO pooldata.proteksi_klaimmbu
//	  (PROTEKSI_ID,TGL_INPUT,PROTEKSI_TIPE,APPROVAL,USER_REQ,CATATAN,ALASAN_REQ,POSISI,JSONDATA,NO_KLAIM)
//	values
//	  ({InputKonversi.ID}, SYSDATE, '18', {InputKonversi.APPROVAL},
//	   {OperatorID.pyUserIdentifier}, {InputKonversi.CATATAN}, {InputKonversi.NOTE_1},
//	   {InputKonversi.POSISI}, {InputKonversi.JSONDATA}, {InputKonversi.ID_SUPPLIER})
//
// # Sisi pemutusnya TIDAK ADA di export
//
// Hanya penyisipannya yang ada. Tidak ada satu pun rule di seluruh export yang MEMBACA
// tabel itu, menyetujuinya, menolaknya, atau memajukan POSISI-nya — pencarian atas
// `proteksi_klaimmbu` dan `PROTEKSI_TIPE` menemukan tepat satu berkas, yaitu kueri di
// atas. Kolom POSISI yang ditampilkan grid `Section/InboxMasterSupplier-Section.xml` pun
// tidak punya sumber yang terbaca (`R-16`).
//
// Yang dikerjakan modul ini karena itu dibatasi pada apa yang benar-benar terbaca:
// **barisnya disisipkan, persis seperti sekarang**. Layar yang memutuskannya adalah
// lingkup tersendiri yang menunggu artefaknya, dan sampai itu tiba antrean persetujuan
// yang berjalan hari ini tidak putus.
//
// Menghapus penyisipannya akan lebih sederhana, dan justru itu yang berbahaya: supplier
// baru tersimpan tanpa pernah muncul di antrean siapa pun, dan tidak ada satu pun pesan
// yang mengatakannya.

// ProtectionTypeSupplier adalah nilai PROTEKSI_TIPE untuk permintaan master supplier.
//
// Literal `'18'` pada kueri, bukan parameter — ia yang membedakan permintaan supplier
// dari jenis permintaan lain di tabel yang sama.
const ProtectionTypeSupplier = "18"

// PositionRequested adalah nilai POSISI pada baris yang baru diminta.
//
// `CreateNewMasterSupplier_post` step 17 dan `EditMasterSupplier_post` step 12 sama-sama
// menetapkan `"1"`. Apa arti angka-angka sesudahnya, dan berapa posisi terakhirnya, tidak
// terbaca di mana pun — lihat catatan di atas.
const PositionRequested = "1"

// ApprovalRequest adalah satu baris permintaan persetujuan.
//
// Nama field-nya menyebut ISINYA, bukan nama kolomnya, mengikuti `D-19`. Satu di antaranya
// perlu dijelaskan: kolom `NO_KLAIM` diisi **ID supplier**, bukan nomor klaim. Itu kolom
// berarti ganda persis seperti yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat; kolomnya
// tetap ditulis apa adanya supaya barisnya terbaca sistem lama, tetapi di dalam modul ini
// ia bernama SupplierID.
type ApprovalRequest struct {
	// ID adalah kolom PROTEKSI_ID — kunci baris permintaan.
	//
	// # Ia TIDAK dapat ditiru, dan itu harus dinyatakan
	//
	// Di Pega nilainya adalah `ChildPageProtection.pyID`, yaitu ID work object dari case
	// `ASM-FW-GKM-Work-Protection` yang dibuat tiga langkah sebelumnya lewat
	// `CreateWorkPage` dan `AddWork`. Sistem baru tidak punya work object Pega, dan tidak
	// ada satu pun rule di export yang memperlihatkan bentuk ID-nya.
	//
	// Yang dipakai sebagai gantinya dibentuk ComposeApprovalID, dan bentuknya sengaja
	// dibuat TIDAK MUNGKIN tertukar dengan ID work object Pega — supaya baris yang dibuat
	// aplikasi ini terbaca langsung asalnya, dengan alasan yang sama seperti prefiks
	// `PNCN` pada nomor klaim (`D-22`).
	//
	// BELUM DIPUTUSKAN: apakah bentuk ini diterima, atau permintaan persetujuan dari
	// aplikasi baru harus memakai sequence tersendiri. Yang kedua menuntut objek basis
	// data baru, dan itu menempuh `D-63`.
	ID string

	// SupplierID adalah kolom NO_KLAIM. Lihat catatan tipe ini.
	SupplierID string

	// RequestedBy adalah kolom USER_REQ — `OperatorID.pyUserIdentifier` di Pega.
	RequestedBy string

	// RequestedAt adalah kolom TGL_INPUT.
	//
	// Pega mengisinya `SYSDATE`, yaitu jam server basis data. Di sini ia datang dari seam
	// Clock dan disimpan sebagai waktu, bukan teks — berbeda dari TGL_INSERT di dalam
	// dokumen supplier, yang bentuknya terikat karena dibaca bersama Pega. Kolom ini
	// tidak dibaca layar mana pun, sehingga tidak ada yang mengikatnya.
	RequestedAt time.Time

	// Reason adalah kolom ALASAN_REQ, diisi KETERANGAN supplier
	// (`InputKonversi.NOTE_1 := MasterSupplier.KETERANGAN`).
	Reason string

	// Note adalah kolom CATATAN. Kedua activity menetapkannya kosong tanpa syarat apa
	// pun; ia tetap ada di sini supaya kolomnya ditulis eksplisit alih-alih dibiarkan
	// bernilai bawaan yang tidak diketahui (`R-08`).
	Note string

	// Decision adalah kolom APPROVAL. Kedua activity menetapkannya kosong — keputusannya
	// memang belum ada saat barisnya lahir.
	Decision string

	// Position adalah kolom POSISI. Selalu PositionRequested pada jalur ini.
	Position string

	// Snapshot adalah isi supplier pada saat diminta, yang tersimpan di kolom JSONDATA.
	//
	// Salinan itu bukan kelalaian melainkan perilaku nyata: `InputKonversi.JSONDATA` masih
	// memegang hasil `@ASM.GetPageJSONString()` dari langkah sebelumnya ketika RDB-Save
	// dijalankan, sehingga barisnya membawa isi supplier apa adanya. Yang memutuskan
	// karena itu dapat melihat apa yang disetujuinya tanpa membaca tabel master.
	//
	// Ia bertipe Supplier, BUKAN teks JSON yang sudah jadi. Alasannya batas lapisan:
	// bentuk dokumen JSON — nama kuncinya, urutannya, apa yang ikut dan apa yang tidak —
	// adalah urusan penyimpanan, dan lapisan aplikasi tidak boleh mengetahuinya. Yang
	// merangkainya adalah adapter yang sama dengan yang merangkai dokumen master, sehingga
	// keduanya tidak pernah dapat berbeda bentuk.
	Snapshot Supplier
}

// ComposeApprovalID membentuk PROTEKSI_ID untuk permintaan yang lahir dari aplikasi ini.
//
// Bentuknya `SUP.<id supplier>.<waktu>`, dengan waktu dalam UTC berformat
// `yyyyMMddHHmmss`. Tiga hal yang diperolehnya:
//
//  1. **Asalnya terbaca.** Awalan `SUP.` tidak mungkin diterbitkan Pega, sehingga baris
//     buatan aplikasi ini dapat dibedakan dari baris warisan tanpa tabel pemetaan.
//  2. **Barisnya tertaut ke supplier-nya** tanpa harus membaca kolom NO_KLAIM yang
//     namanya justru menyesatkan.
//  3. **Penyimpanan berulang dalam detik yang sama ditolak** basis data alih-alih
//     menghasilkan dua permintaan kembar — bila kolomnya memang berkunci utama. Itu
//     bergantung pada DDL yang belum ada (`R-08`); bila ternyata tidak berkunci, dua
//     baris kembar akan lahir dan itu sama dengan perilaku Pega hari ini.
//
// Waktu dipakai dalam UTC, bukan WIB, dan itu disengaja: ia kunci teknis, bukan nilai
// yang dibaca pengguna. Memakai zona waktu setempat pada sebuah kunci berarti kuncinya
// berulang satu jam sekali setiap kali zona waktunya bergeser.
func ComposeApprovalID(supplierID string, at time.Time) string {
	return "SUP." + strings.TrimSpace(supplierID) + "." + at.UTC().Format("20060102150405")
}

// ApprovalRepo adalah seam ke antrean persetujuan SATU portal.
//
// Ia terpisah dari Repo meski tabelnya berada di basis data yang sama, karena ia menjawab
// pertanyaan yang berbeda dan dimiliki proses yang berbeda: master supplier dimiliki layar
// ini, sedangkan antrean persetujuan dimiliki proses yang layarnya belum ada.
type ApprovalRepo interface {
	// RequestApproval menyisipkan satu baris permintaan.
	//
	// Ia dipanggil DI DALAM transaksi yang sama dengan penyimpanan master bila
	// adapternya mampu — supplier yang tersimpan tanpa baris permintaannya akan tertahan
	// selamanya tanpa satu pun tanda, dan baris permintaan tanpa supplier-nya menunjuk
	// baris yang tidak ada.
	RequestApproval(ctx context.Context, request ApprovalRequest) error
}
