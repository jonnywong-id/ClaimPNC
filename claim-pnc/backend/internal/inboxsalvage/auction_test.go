package inboxsalvage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Penilaian jawaban balai lelang adalah SATU aturan di satu tempat.
//
// Ia menentukan `STSTRANSFER`, yaitu kolom yang menentukan di daftar mana sebuah pengajuan
// muncul. Menaruhnya di pengisi seam berarti klien HTTP dan tiruan pengujian dapat
// menilainya berbeda, dan uji kesetaraan tidak akan pernah menangkapnya.
func TestAuctionReceiptAcceptsOnlyWhenTheAnswerSaysSukses(t *testing.T) {
	cases := map[string]struct {
		message  string
		accepted bool
		status   string
	}{
		// Bentuk yang benar-benar dijawab SimasBid, menurut `@contains(…,"Sukses")` pada
		// `Activity/Insert_salvageToSimasBid-Act.xml`.
		"sukses apa adanya": {"Sukses", true, inboxsalvage.TransferStatusSentToAuction},

		// SUBSTRING, bukan kesamaan penuh — jawabannya memang kalimat, bukan kode.
		"sukses di tengah kalimat": {
			"Data Sukses disimpan", true, inboxsalvage.TransferStatusSentToAuction},

		// Satu-satunya penyimpangan dari `@contains`, dan ia MELONGGARKAN: pencocokannya
		// tidak peduli huruf besar-kecil. Jawaban yang benar tetapi dieja lain tidak boleh
		// terbaca sebagai penolakan.
		"beda huruf besar kecil": {
			"SUKSES", true, inboxsalvage.TransferStatusSentToAuction},

		"penolakan": {
			"Gagal: ID sudah ada", false, inboxsalvage.TransferStatusAuctionRejected},

		// Jawaban kosong adalah penolakan, bukan penerimaan. Memperlakukannya sebaliknya
		// berarti satu layanan yang menjawab 200 tanpa isi akan menandai pengajuan sudah
		// terkirim padahal tidak ada yang menerimanya.
		"kosong": {"", false, inboxsalvage.TransferStatusAuctionRejected},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			receipt := inboxsalvage.AuctionReceipt{Message: c.message}
			require.Equal(t, c.accepted, receipt.Accepted())
			require.Equal(t, c.status, receipt.TransferStatus())
		})
	}
}

// Kode STSTRANSFER yang dipakai TIDAK boleh bergeser diam-diam.
//
// `'1'` adalah penyaring daftar "Sudah Dikirim Ke Balai Lelang". Mengubahnya membuat
// pengajuan yang benar-benar terkirim tidak pernah muncul di daftarnya — kegagalan yang
// tidak menghasilkan satu pun galat.
func TestAuctionTransferStatusCodesMatchTheTabFilter(t *testing.T) {
	require.Equal(t, "1", inboxsalvage.TransferStatusSentToAuction)
	require.Equal(t, "9", inboxsalvage.TransferStatusAuctionRejected)
}
