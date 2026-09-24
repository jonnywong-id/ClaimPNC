package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/repo/memory"
)

func claim(id, number, policy, panel, pic, branch, stage string, daysAgo int) inboxoutstanding.OutstandingClaim {
	return inboxoutstanding.OutstandingClaim{
		ClaimID:       id,
		ClaimNumber:   number,
		PolicyNumber:  policy,
		GroupPanel:    panel,
		TechnicalPIC:  pic,
		BranchName:    branch,
		CurrentStage:  stage,
		ProcessStatus: "New",
		// Seluruh klaim contoh dimiliki `pemilik`: daftar TANPA pemilik tidak lagi
		// mungkin, sehingga uji yang menguji hal lain tetap butuh pemilik yang cocok.
		CurrentHolder: pemilik,
		RegisteredAt:  time.Now().UTC().AddDate(0, 0, -daysAgo),
	}
}

func repoWith(claims ...inboxoutstanding.OutstandingClaim) *memory.Repo {
	r := memory.NewRepo()
	r.Add(claims...)
	return r
}

// Pemilik pekerjaan yang dipakai hampir seluruh uji. Daftar TANPA pemilik tidak lagi
// mungkin — lihat Filter.AssignedTo.
const pemilik = "BUDISANTOSO"

func TestTanpaBatasMenampilkanSeluruhLini(t *testing.T) {
	r := repoWith(
		claim("1", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 1),
		claim("2", "PNCN.26.0002", "POL-2", "005", "PIC2", "SBY", "Komite", 2),
	)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{AssignedTo: pemilik})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
}

// Daftar TERIKAT pada pemiliknya — janji terpenting layar ini.//// Tanpa penyaring ini, layar bernama "My Inbox" menampilkan pekerjaan seluruh operator:// terisi, tampak wajar, dan salah tanpa satu pun galat.func TestDaftarHanyaMemuatPekerjaanPemiliknya(t *testing.T) {	r := repoWith(		claim("milik-budi", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 1),		milikOrangLain(claim("milik-siti", "PNCN.26.0002", "POL-2", "005", "PIC2", "SBY", "Komite", 2)),	)	r.Claims()[0].CurrentHolder = pemilik	page, err := r.List(context.Background(), inboxoutstanding.Filter{AssignedTo: pemilik})	require.NoError(t, err)	require.Equal(t, 1, page.Total, "hanya pekerjaan pemanggil yang boleh terlihat")}

// Inilah uji yang paling penting di berkas ini.
//
// Batas data yang cacat TIDAK menghasilkan galat — ia hanya menampilkan baris yang
// seharusnya tersembunyi. Karena itu keadaan "scope kosong" diuji secara khusus: ia harus

func TestPencarianMenelusuriNomorKlaimNomorPolisDanPIC(t *testing.T) {
	r := repoWith(
		claim("1", "PNCN.26.0001", "POL-FIRE-77", "002", "BUDISANTOSO", "JKT", "Komite", 1),
		claim("2", "PNCN.26.0002", "POL-PA-88", "002", "SITIRAHAYU", "JKT", "Komite", 2),
	)

	cases := []struct {
		search string
		want   string
	}{
		{"PNCN.26.0001", "1"}, // nomor klaim
		{"POL-PA-88", "2"},    // nomor polis
		{"BUDISANTOSO", "1"},  // PIC teknik
		{"budisantoso", "1"},  // tidak peka besar-kecil huruf
		{"FIRE", "1"},         // cocok sebagian
	}

	for _, c := range cases {
		page, err := r.List(context.Background(), inboxoutstanding.Filter{
			AssignedTo: pemilik,
			Search:     c.search,
		})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total, "cari %q", c.search)
		require.Equal(t, c.want, page.Claims[0].ClaimID, "cari %q", c.search)
	}
}

func TestPencarianTidakMenelusuriNamaTertanggung(t *testing.T) {
	// Layar lama mencari pada tiga field saja — "No Klaim / No Polis / PIC". Menambah
	// nama tertanggung diam-diam akan membuat hasil berbeda dari Pega pada uji
	// kesetaraan, dan perbedaan itu tidak akan terjelaskan.
	c := claim("1", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 1)
	c.InsuredName = "PT Bumi Contoh Sentosa"
	r := repoWith(c)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: pemilik,
		Search:     "Bumi",
	})
	require.NoError(t, err)
	require.Equal(t, 0, page.Total)
}

func TestPenyaringTahapDanCabang(t *testing.T) {
	r := repoWith(
		claim("1", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 1),
		claim("2", "PNCN.26.0002", "POL-2", "002", "PIC2", "SBY", "Input Register", 2),
	)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: pemilik,
		Stage:      "komite", // tidak peka besar-kecil huruf
	})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "1", page.Claims[0].ClaimID)

	page, err = r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: pemilik,
		BranchCode: "SBY",
	})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "2", page.Claims[0].ClaimID)
}

func TestUrutanTerbaruDiAtas(t *testing.T) {
	r := repoWith(
		claim("lama", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 30),
		claim("baru", "PNCN.26.0002", "POL-2", "002", "PIC2", "JKT", "Komite", 1),
	)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{AssignedTo: pemilik})
	require.NoError(t, err)
	require.Equal(t, "baru", page.Claims[0].ClaimID)
	require.Equal(t, "lama", page.Claims[1].ClaimID)
}

func TestTotalAdalahSeluruhYangCocokBukanJumlahHalaman(t *testing.T) {
	var claims []inboxoutstanding.OutstandingClaim
	for i := 0; i < 10; i++ {
		claims = append(claims, claim(
			string(rune('a'+i)), "PNCN.26.000"+string(rune('0'+i)),
			"POL", "002", "PIC", "JKT", "Komite", i+1))
	}
	r := repoWith(claims...)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: pemilik,
		Limit:      3,
	})
	require.NoError(t, err)
	require.Len(t, page.Claims, 3, "halaman memuat tiga baris")
	require.Equal(t, 10, page.Total, "total tetap menyebut seluruh yang cocok")
}

func TestHalamanDiLuarJangkauanMengembalikanDaftarKosongBukanGalat(t *testing.T) {
	r := repoWith(claim("1", "PNCN.26.0001", "POL-1", "002", "PIC", "JKT", "Komite", 1))

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: pemilik,
		Offset:     500,
	})
	require.NoError(t, err)
	require.Empty(t, page.Claims)
	require.Equal(t, 1, page.Total)
}

// Data contoh memuat keadaan yang mudah membuat layar salah gambar.//// Diperiksa lewat SELURUH isi repo, bukan lewat List: sejak daftar terikat pada// pemiliknya, sebagian keadaan itu memang tidak boleh muncul di My Inbox.func TestDataContohMemuatKeadaanYangMudahMembuatLayarSalahGambar(t *testing.T) {	semua := memory.NewRepoWithSamples().Claims()	require.NotEmpty(t, semua)	var adaTanpaNomor, adaTanpaPemegang, adaTanpaTanggalLapor bool	panels := map[string]bool{}	for _, c := range semua {		adaTanpaNomor = adaTanpaNomor || c.ClaimNumber == ""		adaTanpaPemegang = adaTanpaPemegang || c.CurrentHolder == ""		adaTanpaTanggalLapor = adaTanpaTanggalLapor || c.ReportDate == nil		panels[c.GroupPanel] = true	}	require.True(t, adaTanpaNomor, "klaim belum bernomor")	require.True(t, adaTanpaPemegang, "tugas Workbasket yang belum bertuan")	require.True(t, adaTanpaTanggalLapor, "tanggal lapor kosong")	require.GreaterOrEqual(t, len(panels), 4, "beberapa Group Panel")}// Tugas yang BELUM BERTUAN tidak muncul di My Inbox — ia bukan pekerjaan siapa pun.func TestTugasBelumBertuanTidakMunculDiMyInbox(t *testing.T) {	r := memory.NewRepoWithSamples()	page, err := r.List(context.Background(), inboxoutstanding.Filter{		AssignedTo: "BUDISANTOSO",		Limit:      inboxoutstanding.MaxLimit,	})	require.NoError(t, err)	for _, c := range page.Claims {		require.NotEmpty(t, c.CurrentHolder, "klaim %s tanpa pemegang ikut tampil", c.ClaimID)	}}

// milikOrangLain memindahkan kepemilikan sebuah klaim ke operator lain.
func milikOrangLain(c inboxoutstanding.OutstandingClaim) inboxoutstanding.OutstandingClaim {
	c.CurrentHolder = "SITIRAHAYU"
	return c
}
