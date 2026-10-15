package inboxsalvage

import (
	"context"
	"errors"
	"strings"
)

// Kode STSTRANSFER yang ditulis setelah pengajuan dikirim ke balai lelang.
//
// Keduanya DISALIN dari `Activity/Insert_salvageToSimasBid-Act.xml`, yang menilai jawaban
// SimasBid dengan satu ekspresi:
//
//	@if(@contains(MyServicePage.ResponseService.ResponseMessage,"Sukses"),"1","9")
//
// Angkanya bukan pilihan bebas: `STSTRANSFER = '1'` adalah penyaring daftar "Sudah Dikirim
// Ke Balai Lelang" (lihat tab.go), sehingga memakai kode lain berarti pengajuan yang sudah
// terkirim tidak pernah muncul di daftarnya.
const (
	// TransferStatusSentToAuction berarti balai lelang menerima pengajuan ini.
	TransferStatusSentToAuction = "1"

	// TransferStatusAuctionRejected berarti balai lelang menjawab, tetapi jawabannya bukan
	// penerimaan.
	//
	// Ia DIBEDAKAN dari "belum pernah dikirim": yang satu sudah ditembak dan ditolak, yang
	// lain belum pernah ditembak sama sekali. Menyamakan keduanya membuat pengajuan yang
	// ditolak terlihat seperti pengajuan yang masih menunggu giliran.
	TransferStatusAuctionRejected = "9"
)

// auctionSuccessMarker adalah kata yang dicari di dalam jawaban balai lelang.
//
// Mencocokkan SUBSTRING, bukan kesamaan penuh, karena begitulah `@contains` di sistem lama
// bekerja — dan jawaban SimasBid memang kalimat, bukan kode. Pencocokannya dibuat tidak
// peduli huruf besar-kecil; itu satu-satunya penyimpangan, dan ia melonggarkan, tidak
// mengetatkan.
const auctionSuccessMarker = "sukses"

// ErrAuctionNotAvailable berarti pengiriman ke balai lelang diminta sementara alamatnya
// belum dikonfigurasi.
//
// Ia TIDAK menggagalkan penyimpanan pengajuan. Lihat catatan pada usecase.Create.
var ErrAuctionNotAvailable = errors.New(
	"inboxsalvage: pengiriman ke balai lelang belum dikonfigurasi")

// AuctionDocument adalah satu berkas yang ikut dikirim ke balai lelang.
//
// Isinya base64, sama seperti `MyServicePage.ClaimData.Lelang.DocumentList(<LAST>).Base64`
// di sistem lama.
type AuctionDocument struct {
	FileName string
	Format   string
	Content  string
}

// AuctionSubmission adalah satu pengajuan yang dikirim ke balai lelang.
//
// Nama isiannya mengikuti `MyServicePage.ClaimData.Lelang.*` pada
// `Activity/Insert_salvageToSimasBid-Act.xml`, supaya penelusuran ke rule aslinya tetap
// mungkin.
type AuctionSubmission struct {
	// ItemID adalah `Lelang.IDObject` — kunci pengajuan ini di sisi balai lelang.
	//
	// Bentuknya BERBEDA menurut entitas. Pada portal Insurtech ia
	// `<nomor klaim>/<id salvage>`, pada portal lain ia ID detail salvage:
	//
	//	@if(TempGetApp.LSC_ID=="SIMASNET", Param.NoKlaim+"/"+Param.IDSalvage,
	//	    Param.IDDetailSalvage)
	//
	// Yang memilihnya adalah pemanggil, bukan pengisi seam — perbedaan entitas adalah
	// aturan bisnis, dan `11-CROSSCUTTING.md` §3.4 melarang menyimpulkannya dari nama
	// server.
	ItemID string

	// ClaimNo dan SalvageID dibawa untuk jejak, bukan untuk badan permintaan.
	ClaimNo   string
	SalvageID string

	ItemName        string
	ItemDescription string
	Location        string

	// InJabodetabek adalah `Lelang.FlagJabodetabek`.
	InJabodetabek bool

	// Price adalah `Lelang.Harga`, dibawa sebagai TEKS dengan alasan yang sama seperti
	// SubmissionNotice.MinimumValue.
	Price string

	// EntityFlag adalah `Lelang.Flag` — `"ASI"` bagi portal Insurtech, `"ASM"` bagi yang
	// lain.
	EntityFlag string

	Documents []AuctionDocument
}

// AuctionReceipt adalah jawaban balai lelang.
type AuctionReceipt struct {
	// AuctionID adalah nomor yang diberikan balai lelang, bila ada. Disimpan ke
	// `POOLDATA.PNC_SALVAGE.IDSIMASBID`.
	AuctionID string

	// Message adalah jawaban apa adanya, untuk jejak dan untuk ditampilkan ke petugas.
	Message string
}

// Accepted menyatakan jawaban ini berarti pengajuan DITERIMA.
//
// Penilaiannya ada di DOMAIN, bukan di pengisi seam, karena ia menentukan `STSTRANSFER` —
// kolom yang menentukan di daftar mana baris ini muncul. Menaruhnya di pengisi berarti
// pengisi HTTP dan tiruan pengujian dapat menilainya berbeda, dan uji kesetaraan tidak
// pernah menangkapnya.
func (r AuctionReceipt) Accepted() bool {
	return strings.Contains(strings.ToLower(r.Message), auctionSuccessMarker)
}

// TransferStatus menerjemahkan jawaban menjadi kode `STSTRANSFER`.
func (r AuctionReceipt) TransferStatus() string {
	if r.Accepted() {
		return TransferStatusSentToAuction
	}
	return TransferStatusAuctionRejected
}

// AuctionHouse adalah seam ke balai lelang SimasBid.
//
// Dua pengisi nyata: klien HTTP, dan tiruan untuk pengujian.
//
// # Kenapa ia seam TERSENDIRI, bukan digabung dengan Notifier
//
// Karena kegagalan dan aturan ulangnya berbeda (`09-API-STRATEGY.md` §8.1). Surel yang
// gagal cukup dikirim ulang; pengajuan yang gagal sampai ke balai lelang meninggalkan baris
// yang `STSTRANSFER`-nya menyatakan hal yang berbeda dari keadaan sebenarnya di sana.
//
// # Nil berarti TIDAK DIPASANG
//
// Sama seperti Notifier, dan dengan konsekuensi yang dinyatakan ke pengguna alih-alih
// disamarkan. Alamatnya memang belum dapat dipasang di seluruh lingkungan: satu-satunya
// alamat yang terbaca dari export menunjuk host SANDBOX di dalam ruleset produksi
// (`R-18`), dan ia tidak boleh menjadi nilai bawaan siapa pun.
type AuctionHouse interface {
	SendSalvage(ctx context.Context, submission AuctionSubmission) (AuctionReceipt, error)
}
