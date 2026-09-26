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
