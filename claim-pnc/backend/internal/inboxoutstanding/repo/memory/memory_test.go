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
		ProcessStatus: "BERJALAN",
		RegisteredAt:  time.Now().UTC().AddDate(0, 0, -daysAgo),
	}
}

func repoWith(claims ...inboxoutstanding.OutstandingClaim) *memory.Repo {
	r := memory.NewRepo()
	r.Add(claims...)
	return r
}

// unrestricted dipakai uji yang tidak sedang menguji batas data itu sendiri.
var unrestricted = inboxoutstanding.LineScope{Unrestricted: true}

func TestTanpaBatasMenampilkanSeluruhLini(t *testing.T) {
	r := repoWith(
		claim("1", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 1),
		claim("2", "PNCN.26.0002", "POL-2", "005", "PIC2", "SBY", "Komite", 2),
	)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{Scope: unrestricted})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
}

func TestBatasLiniHanyaMeloloskanGroupPanelYangDiizinkan(t *testing.T) {
	r := repoWith(
		claim("pa", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 1),
		claim("travel", "PNCN.26.0002", "POL-2", "005", "PIC2", "SBY", "Komite", 2),
		claim("fire", "PNCN.26.0003", "POL-3", "006", "PIC3", "BDG", "Komite", 3),
	)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		Scope: inboxoutstanding.ScopeFor(inboxoutstanding.LinePA),
	})
	require.NoError(t, err)

	require.Equal(t, 1, page.Total, "hanya klaim Group Panel 002 yang boleh terlihat")
	require.Equal(t, "pa", page.Claims[0].ClaimID)
}

// Inilah uji yang paling penting di berkas ini.
//
// Batas data yang cacat TIDAK menghasilkan galat — ia hanya menampilkan baris yang
// seharusnya tersembunyi. Karena itu keadaan "scope kosong" diuji secara khusus: ia harus
// gagal TERTUTUP, bukan meloloskan semuanya.
func TestScopeKosongTidakMeloloskanApaPun(t *testing.T) {
	r := repoWith(
		claim("1", "PNCN.26.0001", "POL-1", "002", "PIC1", "JKT", "Komite", 1),
	)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		Scope: inboxoutstanding.LineScope{}, // bukan Unrestricted, tanpa satu pun panel
	})
	require.NoError(t, err)
	require.Equal(t, 0, page.Total, "scope kosong hampir pasti cacat; ia tidak boleh meloloskan data")
}

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
			Scope:  unrestricted,
			Search: c.search,
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
		Scope:  unrestricted,
		Search: "Bumi",
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
		Scope: unrestricted,
		Stage: "komite", // tidak peka besar-kecil huruf
	})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "1", page.Claims[0].ClaimID)

	page, err = r.List(context.Background(), inboxoutstanding.Filter{
		Scope:      unrestricted,
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

	page, err := r.List(context.Background(), inboxoutstanding.Filter{Scope: unrestricted})
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
		Scope: unrestricted,
		Limit: 3,
	})
	require.NoError(t, err)
	require.Len(t, page.Claims, 3, "halaman memuat tiga baris")
	require.Equal(t, 10, page.Total, "total tetap menyebut seluruh yang cocok")
}

func TestHalamanDiLuarJangkauanMengembalikanDaftarKosongBukanGalat(t *testing.T) {
	r := repoWith(claim("1", "PNCN.26.0001", "POL-1", "002", "PIC", "JKT", "Komite", 1))

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		Scope:  unrestricted,
		Offset: 500,
	})
	require.NoError(t, err)
	require.Empty(t, page.Claims)
	require.Equal(t, 1, page.Total)
}

func TestLiniBisnisPenggunaTidakPekaBesarKecilHuruf(t *testing.T) {
	r := memory.NewRepo()
	r.SetLineBusiness("budisantoso", inboxoutstanding.LineNonMBU)

	line, err := r.LineBusinessFor(context.Background(), "  BUDISANTOSO ")
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineNonMBU, line)
}

func TestPenggunaTanpaBarisMengembalikanKosongTanpaGalat(t *testing.T) {
	r := memory.NewRepo()

	line, err := r.LineBusinessFor(context.Background(), "SIAPAPUN")
	require.NoError(t, err, "tidak punya baris adalah keadaan biasa, bukan kegagalan")
	require.Equal(t, "", line)
}

func TestDataContohMemuatKeadaanYangMudahMembuatLayarSalahGambar(t *testing.T) {
	r := memory.NewRepoWithSamples()

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		Scope: unrestricted,
		Limit: inboxoutstanding.MaxLimit,
	})
	require.NoError(t, err)
	require.NotEmpty(t, page.Claims)

	var adaTanpaNomor, adaTanpaPemegang, adaTanpaTanggalLapor bool
	panels := map[string]bool{}
	for _, c := range page.Claims {
		if c.ClaimNumber == "" {
			adaTanpaNomor = true
		}
		if c.CurrentHolder == "" {
			adaTanpaPemegang = true
		}
		if c.ReportDate == nil {
			adaTanpaTanggalLapor = true
		}
		panels[c.GroupPanel] = true
	}

	require.True(t, adaTanpaNomor, "klaim belum bernomor")
	require.True(t, adaTanpaPemegang, "tugas Workbasket yang belum bertuan")
	require.True(t, adaTanpaTanggalLapor, "tanggal lapor kosong")
	require.GreaterOrEqual(t, len(panels), 4, "beberapa Group Panel agar batas data dapat diuji")
}
