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
