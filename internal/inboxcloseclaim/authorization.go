package inboxcloseclaim

// Kewenangan mengajukan ReOpen dan Copy Klaim.
//
// ============================================================================
// PENJAGANYA ADALAH `IsGCNMUser`, DAN RULE ITU SELALU SALAH
// ============================================================================
//
// Work Owner menetapkan 2026-09-23 bahwa yang boleh melakukan ReOpen dan Copy Klaim adalah
// **`IsGCNMUser`**. Rule itu ADA di export, dan isinya satu kondisi tunggal
// (`When/IsGCNMUser-When.xml`):
//
//	pyConditionValue1 = @(Pega-RULES:ExpressionEvaluators).compareTwoValues(1, "=", 2)
//	pyLogic           = A
//
// Yaitu **`1 = 2`** — sebuah kondisi yang tidak pernah benar.
//
// Akibatnya dinyatakan kepada Work Owner sebelum diterapkan: kedua tombol menjadi tidak
// dapat dipakai SIAPA PUN. Work Owner menegaskan pilihannya — rule ditiru apa adanya
// (`P-5`). Keputusan itu dihormati dan diterapkan penuh di sini.
//
// # Apa yang dilakukan rule ini di sistem lama
//
// Ia dipakai **13 kali** di `Navigation/pyCaseWorkerNavigation-Navigation.xml` untuk
// MENYEMBUNYIKAN butir menu — antara lain "Report Adjuster", "My Work", "Calendar",
// "Lost Adjuster", dan "Inbox Banding Harga Salvage". Di Pega ia berfungsi sebagai sakelar
// "jangan tampilkan ini", bukan sebagai pemeriksaan peran.
//
// Perlu dicatat supaya tidak salah dibaca kelak: rule ini **tidak** menjaga kedua tombol di
// `Section/InboxManagerReopen1_Sec-Section.xml`. Di sana tombolnya tidak punya penjaga
// visibilitas sama sekali. Menjadikannya penjaga di sini karena itu **keputusan Work Owner**,
// bukan peniruan perilaku layar lama.
//
// # Kenapa ia ditulis sebagai fungsi, bukan sebagai `const false`
//
// Supaya isinya terbaca sebagai RULE-nya, bukan sebagai kesimpulannya. Yang tertulis di
// bawah adalah kedua angka yang benar-benar dibandingkan Pega — sehingga hari ketika
// Work Owner mengubah aturannya, yang disunting adalah satu tempat yang jelas apa
// hubungannya dengan export.
//
// # Bagaimana menyalakannya kelak
//
// Ganti isi `EvaluateIsGCNMUser` dengan aturan yang sesungguhnya, lalu hapus uji
// `TestIsGCNMUserSelaluSalah` yang sengaja mengunci keadaan hari ini. Tidak ada tempat lain
// yang perlu disentuh: usecase dan transport keduanya memanggil fungsi ini.

// EvaluateIsGCNMUser mengembalikan hasil When rule `IsGCNMUser`.
//
// Kedua angka di bawah disalin apa adanya dari `pyParametersParamValue` pada rule-nya.
func EvaluateIsGCNMUser() bool {
	const kiri, kanan = 1, 2
	return kiri == kanan
}

// CanRequestAction menyatakan apakah pemanggil boleh mengajukan ReOpen atau Copy Klaim.
//
// Ia satu-satunya tempat kewenangan itu ditentukan. Usecase memakainya untuk MENOLAK, dan
// transport memakainya untuk MEMBERI TAHU layar — keduanya membaca sumber yang sama,
// sehingga tombol yang tampil aktif dan endpoint yang menerima tidak dapat berselisih.
//
// Pemanggilnya belum menjadi parameter, dan itu disengaja: rule yang ditiru tidak memeriksa
// pengguna sama sekali. Menambahkan parameter yang tidak dipakai akan menyiratkan adanya
// pembedaan per pengguna yang sebenarnya tidak ada.
func CanRequestAction() bool {
	return EvaluateIsGCNMUser()
}

// AlasanTidakBolehMengajukan adalah keterangan yang dibaca PENGGUNA.
//
// Ia ada di domain, bukan di transport maupun di layar, karena ia menjelaskan sebuah ATURAN
// — dan aturan beserta keterangannya tidak boleh hidup di dua tempat yang dapat menyimpang.
//
// Teksnya menyebut apa yang harus dilakukan, bukan hanya bahwa aksinya ditolak. Pesan
// "tidak berwenang" tanpa jalan keluar akan berubah menjadi laporan gangguan.
const AlasanTidakBolehMengajukan = "Pengajuan buka kembali dan salin klaim sedang tidak " +
	"tersedia untuk semua pengguna. Hubungi administrator bila Anda membutuhkannya."
