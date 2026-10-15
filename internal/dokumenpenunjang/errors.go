package dokumenpenunjang

import "errors"

// ErrBerkasKosong dikembalikan ketika tidak ada isi yang diunggah.
//
// Diperiksa tersendiri, bukan diserahkan ke layanan penyimpanan: berkas 0 byte terunggah
// dengan sukses di sana, dan yang tertinggal adalah baris metadata yang menunjuk berkas
// kosong — terlihat berhasil di layar, tidak dapat dibuka isinya.
var ErrBerkasKosong = errors.New("dokumenpenunjang: berkas kosong")

// ErrNamaBerkasKosong dikembalikan ketika namanya tidak menyisakan satu pun huruf atau
// angka setelah dibersihkan.
//
// Nama seperti `"___.pdf"` menjadi `"pdf"`, tetapi `"___"` menjadi kosong — dan nama kosong
// membuat berkasnya tidak dapat dikenali lagi di daftar.
var ErrNamaBerkasKosong = errors.New("dokumenpenunjang: nama berkas tidak mengandung huruf atau angka")

// ErrBerkasTerlaluBesar dikembalikan ketika isinya melampaui BatasUkuranBerkas.
//
// Ditolak SEBELUM base64 dirakit, bukan sesudah: perakitannya sendiri yang membengkakkan
// memori, sehingga memeriksanya belakangan tidak menolong apa pun.
var ErrBerkasTerlaluBesar = errors.New("dokumenpenunjang: ukuran berkas melampaui batas")

// ErrPengunggahKosong dikembalikan ketika login pengunggah tidak diketahui.
//
// Kolom `USERINPUT` pada catatan akses dan `UserInput` pada muatan unggah keduanya memuat
// siapa yang mengunggah. Membiarkannya kosong menghapus satu-satunya jejak siapa yang
// menaruh sebuah berkas — dan `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang.
var ErrPengunggahKosong = errors.New("dokumenpenunjang: pengunggah tidak diketahui")

// ErrTautanGagal dikembalikan ketika alamat baca berkas yang kedaluwarsa tidak dapat
// diperpanjang layanan penyimpanan.
var ErrTautanGagal = errors.New("dokumenpenunjang: alamat berkas tidak dapat diperpanjang")

// ErrHapusGagal dikembalikan ketika layanan penyimpanan tidak menyatakan berkasnya terhapus.
var ErrHapusGagal = errors.New("dokumenpenunjang: berkas tidak terhapus dari penyimpanan")

// ErrTidakDitemukan dikembalikan ketika dokumen yang diminta tidak ada.
var ErrTidakDitemukan = errors.New("dokumenpenunjang: dokumen tidak ditemukan")

// ErrFolderAplikasiTidakAda dikembalikan ketika master folder penyimpanan tidak memuat
// aplikasi ini.
//
// # Kenapa ia MENGGAGALKAN unggahan, bukan jatuh ke nilai bawaan
//
// Nama folder menentukan DI MANA berkasnya mendarat. Menebaknya berarti menaruh berkas
// klaim di folder aplikasi lain — dan tidak ada galat yang menandainya, sebab unggahannya
// tetap berhasil. Yang terlihat baru belakangan, ketika berkasnya dicari dan tidak ada.
var ErrFolderAplikasiTidakAda = errors.New(
	"dokumenpenunjang: folder penyimpanan untuk aplikasi ini belum terdaftar")

// ErrLinkTakTerjangkau dikembalikan ketika basis data portal tidak dapat menjangkau
// GENERAL lewat DB link `@asmd.sinarmas.co.id` — ORA-12541, ORA-12170, ORA-02019, dan
// sejenisnya. Tabel folder, izin unggah, dan metadata dokumen seluruhnya berada di sana.
//
// Dibedakan dari galat internal biasa karena penyebabnya di luar aplikasi (listener atau
// jaringan database ASMD) dan pengguna perlu tahu apa yang dilaporkan: tanpanya layar hanya
// menampilkan "Terjadi kesalahan pada sistem." dan penelusurannya dimulai dari tempat yang
// salah. Belum ada berkas yang terkirim saat galat ini muncul, sehingga mengulang aman.
var ErrLinkTakTerjangkau = errors.New(
	"dokumenpenunjang: basis data penyimpanan dokumen (DB link ASMD) tidak dapat dihubungi")

// ErrKonversiGagal dikembalikan ketika layanan konversi gambar menolak berkasnya.
//
// # Kenapa ia MEMBATALKAN unggahan, bukan meneruskan berkas aslinya
//
// Itu perilaku Pega, dan perilakunya benar. `Activity/Convert_Avif-Act.xml:1408`
// menyetel `Param.ErrMsg := "Gagal Konversi Avif"` ketika `Status=="false"`, dan
// `TempAviff.City` — penampung hasilnya — TIDAK diisi. Pemanggilnya lalu menyalin
// penampung kosong itu ke `Param.Base64`.
//
// Dibaca apa adanya: meneruskan berarti mengunggah **berkas kosong** yang tercatat sebagai
// dokumen yang sah. `Activity/InsertDokumenPNC-Act.xml:2649` menahannya dengan keluar dari
// activity ketika `Param.ErrMsg` terisi, dan penahanan itulah yang ditiru.
//
// Meneruskan berkas ASLI juga ditolak, meski tampak lebih ramah: yang tersimpan kemudian
// bukan yang diminta, dan tidak ada satu pun penanda yang membedakannya.
var ErrKonversiGagal = errors.New("dokumenpenunjang: berkas gagal dikonversi")

// ErrUnggahGagal membungkus kegagalan layanan penyimpanan.
//
// Dibedakan dari kegagalan pencatatan metadata supaya transport dapat menjawab berbeda:
// yang ini dapat dicoba ulang oleh pengguna, yang itu tidak.
var ErrUnggahGagal = errors.New("dokumenpenunjang: layanan penyimpanan menolak berkas")

// ErrMetadataGagal dikembalikan ketika berkasnya SUDAH terunggah tetapi metadatanya gagal
// dicatat.
//
// # Kenapa ia galat tersendiri, dan kenapa pesannya menyebut itu
//
// Keadaan ini TIDAK dapat dibatalkan: layanan penyimpanan sudah memegang berkasnya, dan
// tidak ada operasi hapus yang dipakai Pega maupun yang kita punya. Yang tertinggal adalah
// berkas tanpa catatan — tidak muncul di daftar mana pun, dan tidak ada yang tahu ia ada.
//
// Pengguna harus diberi tahu bahwa berkasnya TERKIRIM tetapi tidak tercatat, supaya ia
// tidak mengunggah ulang berkali-kali dan meninggalkan salinan yatim di setiap percobaan.
var ErrMetadataGagal = errors.New(
	"dokumenpenunjang: berkas sudah terkirim ke penyimpanan tetapi metadatanya gagal dicatat")

// ErrKodeAksesDitolak dikembalikan ketika layanan penyimpanan menolak `KodeString`.
//
// # Kenapa ia dipisahkan dari ErrUnggahGagal
//
// Keduanya sama-sama berakhir di layanan penyimpanan, tetapi yang dapat dilakukan pengguna
// BERBEDA sejauh-jauhnya:
//
//	ErrUnggahGagal        layanan sedang bermasalah  -> ulangi, mungkin berhasil
//	ErrKodeAksesDitolak   kredensial salah           -> mengulang TIDAK AKAN berhasil
//
// Sebelum pemisahan ini, keduanya menghasilkan kalimat "Layanan penyimpanan dokumen sedang
// tidak dapat dihubungi. Berkas belum tersimpan; silakan coba lagi." Kalimat itu salah pada
// kasus kedua di dua hal sekaligus: layanannya TERHUBUNG dengan baik — ia menjawab HTTP 200
// dengan sebab yang jelas — dan mengulang hanya mengulang penolakan yang sama.
//
// # Harganya nyata, dan terukur
//
// Pada 2026-10-13 penelusuran sebuah unggahan yang gagal dimulai dari proxy dan jaringan,
// karena itulah yang disebut layar. Sebab sebenarnya — `invalid kodestring data`, jawaban
// layanan atas token sekali pakai pada folder `klaimasmpnctest` — hanya ada di log server.
// Deretan kejadiannya terbaca dari data: unggahan terakhir yang berhasil di folder itu
// 2026-09-29, dan token pertama yang ditolak 2026-09-30.
//
// # Pengenalannya lewat kata `kodestring`, bukan kalimat utuh
//
// Layanan memakai dua kalimat berbeda untuk sebab yang sama:
//
//	"invalid kodestring data"                          kode tidak dikenal
//	"Error, can't Kodestring null value in json data"  kode tidak dikirim
//
// Keduanya hanya berbagi satu kata, dan kata itulah yang dicocokkan — tidak peka huruf.
// Mencocokkan kalimat utuh akan luput begitu seberang mengubah satu kata; mencocokkan
// `ErrorCode` 500 akan menjaring kegagalan lain yang kebetulan bernomor sama.
var ErrKodeAksesDitolak = errors.New(
	"dokumenpenunjang: layanan penyimpanan menolak kode akses (KodeString)")
