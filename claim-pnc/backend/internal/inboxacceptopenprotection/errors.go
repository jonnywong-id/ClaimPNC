package inboxacceptopenprotection

import "errors"

// ErrNotFound dikembalikan ketika proteksi yang diminta tidak ada.
var ErrNotFound = errors.New("inboxacceptopenprotection: proteksi tidak ditemukan")

// ErrAlreadyDecided dikembalikan ketika proteksi sudah diakseptasi orang lain.
//
// # Kenapa ia galat tersendiri, bukan sekadar "tidak dapat diubah"
//
// Layar ini antrean BERSAMA: `Flow/CreateProtection_Flow.xml` menempatkan akseptasi di
// workbasket `ProtectionPNC`, sehingga beberapa petugas melihat baris yang sama pada saat
// yang sama. Dua orang menekan tombol atas baris yang sama bukan kejadian langka melainkan
// kejadian yang WAJAR.
//
// Yang kedua harus diberi tahu bahwa keputusannya SUDAH DIAMBIL, beserta oleh siapa — bukan
// diberi pesan galat teknis yang membuatnya mencoba lagi.
var ErrAlreadyDecided = errors.New("inboxacceptopenprotection: proteksi sudah diakseptasi")

// ErrIncomplete dikembalikan ketika proteksi belum layak diakseptasi.
//
// Aturannya dari penyaring `InboxOpenProtection2_RD`, yang menuntut `CaseID IS NOT NULL` dan
// `PolicyNo IS NOT NULL`. Proteksi yang belum tertaut klaim seharusnya tidak pernah muncul
// di antrean ini; bila ia sampai juga — lewat tautan yang disimpan, misalnya — keputusan
// atasnya ditolak, bukan diterima diam-diam.
var ErrIncomplete = errors.New("inboxacceptopenprotection: proteksi belum tertaut klaim dan belum layak diakseptasi")

// ErrUnknownDecision dikembalikan ketika keputusan yang dikirim bukan setuju maupun tolak.
//
// Tidak ada nilai bawaan. Memperlakukan nilai asing sebagai "tolak" akan menolak proteksi
// yang tidak seorang pun bermaksud menolaknya; memperlakukannya sebagai "setuju" jauh lebih
// buruk lagi.
var ErrUnknownDecision = errors.New("inboxacceptopenprotection: keputusan akseptasi tidak dikenali")

// ErrForbidden dikembalikan ketika pemanggil tidak berwenang atas layar atau antrean ini.
//
// # Kenapa ia DIBEDAKAN dari ErrNotFound
//
// Keduanya dapat dijawab 404 demi tidak membocorkan keberadaan sebuah baris, dan itu pilihan
// yang sah pada sistem yang menghadap publik. Di sini tidak: aplikasi ini internal, dan
// petugas yang salah membuka layar perlu tahu bahwa ia MEMANG TIDAK BERHAK — bukan mengira
// datanya hilang lalu melaporkannya sebagai gangguan.
//
// Transport menjawabnya 403, dan `10-API-STRATEGY.md` §5 membedakannya dari 401: yang ini
// sudah masuk, hanya tidak berwenang.
var ErrForbidden = errors.New("inboxacceptopenprotection: tidak berwenang atas layar ini")

// ErrClaimNotSynced dikembalikan ketika perubahan yang disetujui tidak dapat diterapkan ke
// data klaim.
//
// # Kenapa keputusannya DIBATALKAN, bukan diteruskan
//
// Menyetujui permintaan tipe '7' berarti menyetujui perubahan Tanggal Kejadian klaim. Bila
// keputusannya tersimpan tetapi klaimnya tidak berubah, yang tertinggal adalah persetujuan
// atas perubahan yang tidak pernah terjadi — dan tidak ada gejala yang menandainya. Petugas
// melihat "disetujui", sedangkan klaim tetap memakai DOL lama.
//
// Dua sebab yang mungkin: proteksinya tidak punya `ID_CLAIM`, atau tidak ada baris
// ber-`CLAIMID` itu di `POOLDATA.T_CLAIM_PNC`.
//
// Karena itu pesannya menyebut klaimnya, bukan menyalahkan pengguna: yang perlu dibereskan
// ada di data, bukan pada tindakannya.
var ErrClaimNotSynced = errors.New(
	"inboxacceptopenprotection: perubahan tidak dapat diterapkan ke data klaim")
