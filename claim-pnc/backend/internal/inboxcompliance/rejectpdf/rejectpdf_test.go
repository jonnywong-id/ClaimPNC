package rejectpdf_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/rejectpdf"
)

// contoh membentuk dokumen surat berisi data KARANGAN seluruhnya (`D-69`).
//
// Enam isian, dan memang hanya itu: templat Pega tidak punya merge field untuk isian
// form mana pun. Lihat catatan paket rejectpdf.
func contoh() inboxcompliance.RejectLetterDocument {
	return inboxcompliance.RejectLetterDocument{
		LetterDate:       "08 Oktober 2026",
		LetterNumber:     "1532/CL.AHID.ASM/10/2026",
		RecipientCompany: "PT CONTOH SEJAHTERA",
		RecipientAddress: "JL. CONTOH NO. 1\nKOTA CONTOH",
		SubjectName:      "PESERTA CONTOH",
	}
}

// Suratnya DUA halaman — templat memasang `page-break-before: always` di tengahnya.
// REJECT_PDF_OUT=<folder> menyimpan hasilnya untuk diperiksa mata.
func TestSuratTerbitDuaHalaman(t *testing.T) {
	t.Parallel()

	berkas, err := rejectpdf.Renderer{}.Render(contoh())
	require.NoError(t, err)
	require.Equal(t, "%PDF", string(berkas[:4]))
	assert.Equal(t, 2, bytes.Count(berkas, []byte("/Type /Page\n")))

	if out := os.Getenv("REJECT_PDF_OUT"); out != "" {
		require.NoError(t, os.WriteFile(filepath.Join(out, inboxcompliance.RejectLetterFileName), berkas, 0o600))
	}
}

// Dokumen KOSONG pun tetap menghasilkan surat yang sah.
//
// Bukan kasus karangan: alamat tertanggung memang belum terbawa dari mana pun, dan
// klaim yang belum punya objek pertanggungan membuat SubjectName kosong. Surat yang
// gagal terbit karena itu jauh lebih buruk daripada surat yang satu barisnya kosong.
func TestDokumenKosongTetapMenghasilkanSurat(t *testing.T) {
	t.Parallel()

	berkas, err := rejectpdf.Renderer{}.Render(inboxcompliance.RejectLetterDocument{})
	require.NoError(t, err)
	require.Equal(t, "%PDF", string(berkas[:4]))
	assert.Equal(t, 2, bytes.Count(berkas, []byte("/Type /Page\n")))
}

// Nama tertanggung yang panjang membungkus, bukan merusak tata letak.
func TestIsianPanjangMembungkusBukanMerusak(t *testing.T) {
	t.Parallel()

	d := contoh()
	d.RecipientCompany = strings.Repeat("PT CONTOH DENGAN NAMA PANJANG SEKALI ", 6)
	d.RecipientAddress = strings.Repeat("JL. CONTOH YANG SANGAT PANJANG NO. 1, ", 8)

	berkas, err := rejectpdf.Renderer{}.Render(d)
	require.NoError(t, err)
	require.Equal(t, "%PDF", string(berkas[:4]))
	assert.GreaterOrEqual(t, bytes.Count(berkas, []byte("/Type /Page\n")), 2)
}

// Karakter di luar Latin-1 tidak merusak berkas — ia diganti, bukan menggagalkan surat.
func TestKarakterDiLuarLatin1TidakMerusakBerkas(t *testing.T) {
	t.Parallel()

	d := contoh()
	d.RecipientCompany = "PT “CONTOH” — 中文"

	berkas, err := rejectpdf.Renderer{}.Render(d)
	require.NoError(t, err)
	require.Equal(t, "%PDF", string(berkas[:4]))
}

// Baris tembusan mengikuti kedua `pega:when` templat.
func TestBarisTembusanMengikutiTemplat(t *testing.T) {
	t.Parallel()

	for _, uji := range []struct {
		nama       string
		perusahaan string
		perorangan string
		mau        string
	}{
		{
			nama:       "badan usaha saja menjadi Direktur",
			perusahaan: "PT CONTOH SEJAHTERA",
			mau:        "Direktur PT CONTOH SEJAHTERA",
		},
		{
			nama:       "perorangan saja menjadi Pihak Sdr.",
			perorangan: "NAMA CONTOH",
			mau:        "Pihak Sdr. NAMA CONTOH",
		},
		{
			// Templat menguji "perorangan kosong" lebih dulu, sehingga bentuk badan usaha
			// yang menang. Ini bukan pilihan kami — ini urutan `pega:when`-nya.
			nama:       "keduanya terisi dimenangkan bentuk badan usaha",
			perusahaan: "PT CONTOH SEJAHTERA",
			perorangan: "NAMA CONTOH",
			mau:        "Direktur PT CONTOH SEJAHTERA",
		},
		{
			// Tanpa nama sama sekali, kata "Direktur" sendirian tidak berguna tetapi juga
			// tidak merusak. Yang penting ia tidak menyisakan spasi menggantung.
			nama: "keduanya kosong tidak menyisakan spasi menggantung",
			mau:  "Direktur",
		},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			t.Parallel()
			ada := rejectpdf.RecipientLine(inboxcompliance.RejectLetterDocument{
				RecipientCompany: uji.perusahaan,
				RecipientPerson:  uji.perorangan,
			})
			assert.Equal(t, uji.mau, ada)
		})
	}
}
