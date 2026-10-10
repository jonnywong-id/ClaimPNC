package sqlstore

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// keempatKueriEkspor adalah seluruh kueri ekspor tab KPI PIC Teknik.
var keempatKueriEkspor = []string{
	"ekspor_pic_progress", "ekspor_pic_sla",
	"ekspor_pic_akseptasi", "ekspor_pic_analisis",
}

// Setiap pilihan "Pilih Data KPI" menunjuk kueri yang BENAR-BENAR ada.
//
// Pemetaannya lewat `switch` dengan cabang `default`, yang berarti nilai tak terduga
// diam-diam jatuh ke Progress. Uji ini memastikan keempat nilai yang sah menunjuk kuerinya
// masing-masing — bukan tiga di antaranya kebetulan jatuh ke cabang yang sama.
func TestSetiapPilihanDataKPIPunyaKueri(t *testing.T) {
	seen := map[string]bool{}
	for _, option := range reportkpi.PICExportOptions() {
		name := option.Code.QueryName()
		require.NotEmptyf(t, query(name), "kueri %q kosong", name)
		require.Falsef(t, seen[name],
			"pilihan %q (%s) memakai kueri yang sama dengan pilihan lain: %q",
			option.Code, option.Label, name)
		seen[name] = true
	}
	require.Len(t, seen, 4)
}

// Tidak satu pun kueri ekspor merangkai NILAI ke dalam teks SQL.
//
// # Kenapa uji ini ada
//
// Keempat kueri Pega menulis `IN {ASIS:TempDateReport.UserTeknis}` — daftar PIC disisipkan
// sebagai TEKS. Itu persis pola yang `D-20` tutup, dan di sini ia harus menjadi penanda
// bind.
//
// Yang diperiksa: penanda /*PIC_LIST*/ ada, dan tidak satu pun sisa `{ASIS` atau `{Temp`
// tertinggal dari penyalinan.
func TestKueriEksporTidakMerangkaiNilai(t *testing.T) {
	for _, name := range keempatKueriEkspor {
		text := query(name)
		require.Containsf(t, text, picListMarker,
			"kueri %q kehilangan penanda daftar PIC", name)
		require.NotContainsf(t, text, "{ASIS",
			"kueri %q masih menyisipkan nilai sebagai teks", name)
		require.NotContainsf(t, text, "{Temp",
			"kueri %q masih memuat rujukan properti Pega", name)
	}
}

// Pemformatan tanggal TIDAK dilakukan di SQL.
//
// `D-20` memindahkannya ke Go supaya kuerinya portabel. Kueri Pega memakai
// `TO_CHAR(..., 'dd/mm/yyyy')` pada tiga puluhan kolom; satu saja yang tertinggal akan
// menghasilkan kolom bertipe teks yang tidak ikut diformat lapisan Go — dan selisihnya
// hanya terlihat bila seseorang membuka berkasnya.
func TestKueriEksporTidakMemformatTanggalDiSQL(t *testing.T) {
	for _, name := range keempatKueriEkspor {
		require.NotContainsf(t, strings.ToUpper(query(name)), "TO_CHAR",
			"kueri %q masih memformat di SQL; pemformatan tanggal milik Go", name)
	}
}

// Penanda bind disusun RAPAT dan nilainya lengkap.
//
// Daftar PIC panjangnya berubah-ubah, sehingga penanda bind-nya dihitung saat dijalankan.
// Salah hitung tidak menghasilkan galat sintaks — ia menghasilkan ORA-01008 yang baru
// muncul di tangan pengguna.
func TestPenandaBindDaftarPICRapat(t *testing.T) {
	span := reportkpi.PICQuery{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
	}
	pics := []string{"PICSATU", "PICDUA", "PICTIGA"}

	statement, args := picExportStatement(query("ekspor_pic_sla"), span, pics)

	require.Len(t, args, 5, "dua tanggal ditambah tiga PIC")
	require.Contains(t, statement, "(:3, :4, :5)")
	require.NotContains(t, statement, picListMarker, "penanda harus TERGANTI")
	// Nilainya ikut, bukan hanya penandanya.
	require.Equal(t, "PICSATU", args[2])
	require.Equal(t, "PICTIGA", args[4])
}

// Kolom tahun dibedakan dari kolom tanggal penuh.
//
// `ENDDATE` dan `EDMDATE` keduanya berasal dari `DATEOFLOSS`; yang pertama ditulis sebagai
// TAHUN saja. Tanpa pembedaan ini keduanya tergambar identik, dan satu kolom berkas
// menjadi salah tanpa satu pun tanda.
func TestKolomTahunDitulisTahunSaja(t *testing.T) {
	hari := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)

	require.Equal(t, "2026", selEkspor(hari, true))
	require.Equal(t, "17/03/2026", selEkspor(hari, false))
	require.True(t, kolomTahun["ENDDATE"])
	require.False(t, kolomTahun["EDMDATE"],
		"EDMDATE memuat tanggal penuh; menandainya sebagai tahun akan menghapus hari dan bulannya")
}

// Nilai kosong ditulis sebagai teks kosong, bukan "<nil>".
//
// Berkas ini dibandingkan baris per baris dengan berkas Pega. Satu sel bertuliskan
// "<nil>" akan menghasilkan selisih pada setiap baris yang kolomnya memang kosong.
func TestSelKosongDitulisKosong(t *testing.T) {
	require.Equal(t, "", selEkspor(nil, false))
	require.Equal(t, "", selEkspor(nil, true))
}
