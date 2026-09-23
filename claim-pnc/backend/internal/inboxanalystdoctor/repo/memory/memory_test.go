package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxanalystdoctor/repo/memory"
)

// list menjalankan kueri terhadap antrean contoh dan mengembalikan halamannya.
func list(t *testing.T, operator string, f inboxanalystdoctor.Filter) inboxanalystdoctor.Page {
	t.Helper()

	page, err := memory.NewSampleStore().List(context.Background(), operator, f)
	require.NoError(t, err)
	return page
}

// numbers mengambil nomor case tiap baris, supaya penegasan terbaca sebagai daftar.
func numbers(page inboxanalystdoctor.Page) []string {
	result := make([]string, 0, len(page.Tasks))
	for _, task := range page.Tasks {
		result = append(result, task.ClaimNumber)
	}
	return result
}

// TestAntreanHanyaBerisiTugasMilikPemanggil menguji penyaring B Report Definition.
//
// Data contoh memuat satu baris milik operator lain. Bila batas kewenangan hilang, ia ikut
// tampil — dan tugas penilaian medis milik petugas lain adalah kebocoran, bukan kelebihan
// baris.
func TestAntreanHanyaBerisiTugasMilikPemanggil(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 100})

	for _, task := range page.Tasks {
		require.Equal(t, memory.SampleOperator, task.AssignedOperator)
	}
	require.NotContains(t, numbers(page), "PNCN.26.0222", "baris itu milik operator lain")
}

func TestOperatorLainMelihatAntreannyaSendiri(t *testing.T) {
	page := list(t, memory.SampleOtherOperator, inboxanalystdoctor.Filter{Limit: 100})

	require.Equal(t, []string{"PNCN.26.0222"}, numbers(page))
}

// TestPerbandinganOperatorTidakPekaHurufBesarKecil mengunci keputusan yang dijelaskan di
// CATATAN 3 pada berkas .sql.
//
// Kapitalisasi identitas di sistem lama terbukti tidak seragam (`11-SECURITY.md` §3.1).
// Perbandingan persis akan membuat antrean tampak KOSONG bagi pengguna yang login-nya
// tersimpan berbeda huruf — dan antrean kosong tidak pernah dilaporkan sebagai kerusakan.
func TestPerbandinganOperatorTidakPekaHurufBesarKecil(t *testing.T) {
	atas := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 100})
	bawah := list(t, "adminklaim", inboxanalystdoctor.Filter{Limit: 100})

	require.Equal(t, numbers(atas), numbers(bawah))
	require.NotEmpty(t, numbers(bawah))
}

// TestBarisAntreanComplianceTidakBocorKeSini menguji penyaring A.
//
// Data contoh sengaja memuat satu baris ber-`isComplianceTransfer = "1"`. Ia milik antrean
// Compliance, dan satu-satunya cara membuktikan penyaringnya bekerja adalah menyediakan baris
// yang harus tertolak olehnya.
func TestBarisAntreanComplianceTidakBocorKeSini(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 100})

	require.NotContains(t, numbers(page), "PNCN.26.0231")
}

// TestTugasSelesaiKeluarDariAntrean menguji penyaring C.
func TestTugasSelesaiKeluarDariAntrean(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 100})

	require.NotContains(t, numbers(page), "PNCN.26.0240")
}

// TestKlaimDitolakTETAPMuncul adalah uji yang paling mudah terbalik di modul ini.
//
// Report Definition mengecualikan `Resolved-Completed` SAJA. Menyamakannya dengan Inbox
// Outstanding — yang mengecualikan `Resolved-Rejected` juga — akan menghilangkan klaim yang
// ditolak dari antrean penilaian medis, tanpa satu pun galat yang menandakannya.
func TestKlaimDitolakTETAPMuncul(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 100})

	require.Contains(t, numbers(page), "PNCN.26.0255")
}

// TestUrutanTerbaruLebihDuluDenganPemutusSeri menguji kedua kunci urutan sekaligus.
//
// Dua baris contoh berwaktu daftar SAMA PERSIS. Tanpa pemutus seri `pzInsKey` menurun,
// urutan keduanya tidak tetap — dan begitu halamannya dipotong, satu baris muncul di dua
// halaman sekaligus hilang dari halaman lain.
func TestUrutanTerbaruLebihDuluDenganPemutusSeri(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 100})

	require.Equal(t, []string{
		"PNCN.26.0311", // 18 Sep
		"PNCN.26.0298", // 15 Sep
		"PNCN.26.0290", // 12 Sep — pzInsKey lebih besar, jadi lebih dulu
		"PNCN.26.0289", // 12 Sep
		"PNCN.26.0255", // 5 Sep, Resolved-Rejected
	}, numbers(page))
}

func TestPencarianMenyentuhNomorCaseDanNomorPolis(t *testing.T) {
	byCase := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Search: "0298", Limit: 100})
	require.Equal(t, []string{"PNCN.26.0298"}, numbers(byCase))

	byPolicy := list(t, memory.SampleOperator,
		inboxanalystdoctor.Filter{Search: "26.005.2026", Limit: 100})
	require.Equal(t, []string{"PNCN.26.0289"}, numbers(byPolicy))
}

func TestPencarianTidakPekaHurufBesarKecil(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Search: "pncn.26.0311", Limit: 100})

	require.Equal(t, []string{"PNCN.26.0311"}, numbers(page))
}

// TestTotalAdalahJumlahSELURUHBarisYangCocok menjaga arti Total.
//
// Ia jumlah baris yang cocok, BUKAN jumlah baris di halaman ini. Bilah halaman di layar
// menghitung jumlah halaman dari angka ini; salah mengartikannya membuat halaman kedua
// tampak tidak ada.
func TestTotalAdalahJumlahSELURUHBarisYangCocok(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 2})

	require.Len(t, page.Tasks, 2)
	require.Equal(t, 5, page.Total)
}

func TestHalamanKeduaMelanjutkanYangPertama(t *testing.T) {
	first := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 2})
	second := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 2, Offset: 2})

	require.Equal(t, []string{"PNCN.26.0311", "PNCN.26.0298"}, numbers(first))
	require.Equal(t, []string{"PNCN.26.0290", "PNCN.26.0289"}, numbers(second))
}

// TestHalamanDiLuarJangkauanTetapMembawaTotal menjaga bilah halaman tidak menghilang.
//
// Pengguna yang terlanjur berada di halaman sepuluh kehilangan jalan kembali bila totalnya
// ikut menjadi nol.
func TestHalamanDiLuarJangkauanTetapMembawaTotal(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxanalystdoctor.Filter{Limit: 2, Offset: 500})

	require.Empty(t, page.Tasks)
	require.Equal(t, 5, page.Total)
	require.NotNil(t, page.Tasks, "senarai kosong, bukan nil — supaya JSON-nya [] bukan null")
}

// TestDataContohTidakMemuatOperatorSungguhan menjaga hardcode tidak berpindah diam-diam.
//
// `DRRATNA` adalah Operator ID yang tertanam di `Flow/Register_Flow.xml` dan salah satu dari
// 24 hardcode yang `D-15` perintahkan dihapus. Menuliskannya sebagai data yang dijalankan
// akan memindahkannya ke sistem baru lewat pintu belakang.
func TestDataContohTidakMemuatOperatorSungguhan(t *testing.T) {
	for _, record := range memory.SampleTasks {
		require.NotEqual(t, "DRRATNA", record.Task.AssignedOperator)
	}
}
