package mastersupplier

import "time"

// # Satu-satunya tempat modul ini mengubah waktu menjadi teks
//
// `08-TECHNICAL-STRATEGY.md` §4.4 menetapkan waktu disimpan UTC dan dikonversi ke WIB di
// SATU tempat saja, dan melarang penambahan tujuh jam manual di mana pun. Berkas ini
// tempat itu untuk modul Master Supplier.
//
// Yang dikonversi hanya satu nilai: kunci `TGL_INSERT` di dalam dokumen supplier.

// jakartaOffset adalah selisih WIB terhadap UTC.
//
// Ia dipakai HANYA sebagai cadangan bila basis data zona waktu sistem operasi tidak
// tersedia — lihat jakartaLocation. Ia BUKAN penambahan tujuh jam manual yang dilarang
// `08-TECHNICAL-STRATEGY.md` §4.4: yang dilarang adalah menambahkannya tersebar di
// puluhan tempat seperti `Set7Hours` di sistem lama. Di sini ia satu konstanta di satu
// berkas, dan seluruh modul melewatinya.
const jakartaOffset = 7 * 60 * 60

// jakartaLocation adalah zona waktu WIB.
//
// `time.LoadLocation` dapat gagal pada mesin tanpa basis data zona waktu — wadah yang
// sangat ramping, misalnya. Kegagalannya TIDAK dibiarkan menghentikan aplikasi dan TIDAK
// pula diam-diam jatuh ke UTC: yang kedua akan menggeser tanggal tujuh jam tepat pada
// kasus di sekitar tengah malam, persis kelas cacat yang `R-12` catat.
//
// Yang dipakai sebagai gantinya adalah zona tetap +07:00. WIB tidak mengenal waktu musim
// panas dan tidak pernah bergeser sejak 1964, sehingga keduanya menghasilkan tanggal yang
// sama — perbedaannya hanya pada nama zonanya, dan nama itu tidak pernah ditulis ke mana
// pun oleh modul ini.
var jakartaLocation = loadJakarta()

func loadJakarta() *time.Location {
	if loaded, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loaded
	}
	return time.FixedZone("WIB", jakartaOffset)
}

// FormatJakartaDate mengubah sebuah waktu menjadi tanggal WIB berformat `dd/MM/yyyy`.
//
// # Kenapa bentuknya teks, dan kenapa justru itu yang benar di sini
//
// Karena yang ditulis adalah kunci `TGL_INSERT` di dalam dokumen JSON yang DIBACA BERSAMA
// Pega selama masa paralel (`D-21`), dan Pega menulisnya persis begitu:
//
//	Activity/CreateNewMasterSupplier_post step 6
//	Activity/EditMasterSupplier_post step 7
//	  MasterSupplier.TGL_INSERT :=
//	      @DateTime.FormatDateTime(@DateTime.CurrentDateTime(),"dd/MM/yyyy","in_ID","Asia/Jakarta")
//
// Menyimpannya sebagai waktu yang benar akan membuat layar Pega membaca sesuatu yang
// berbeda dari yang ditulisnya sendiri.
//
// # Utang yang disadari, bukan yang tidak terlihat
//
// Tanggal sebagai teks `dd/MM/yyyy` tidak dapat diurutkan, tidak dapat disaring sebagai
// rentang, dan tidak memakai index — persis utang yang `03-CURRENT-ARCHITECTURE.md` §4.4
// dan `09-DATABASE-STRATEGY.md` §3.2 catat. Ia dipertahankan HANYA di dalam dokumen yang
// dibagi dengan Pega. Baris permintaan persetujuan, yang tidak dibaca layar mana pun,
// memakai waktu sungguhan — lihat ApprovalRequest.RequestedAt.
//
// # Yang hilang, dan itu memang sudah hilang sejak dulu
//
// Jamnya. Pega pun hanya menyimpan tanggalnya, sehingga dua penyimpanan pada hari yang
// sama tidak dapat dibedakan urutannya. Menambahkan jam di sini akan membuat nilainya
// tidak lagi terbaca sama oleh Pega.
func FormatJakartaDate(at time.Time) string {
	return at.In(jakartaLocation).Format("02/01/2006")
}
