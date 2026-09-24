package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/inboxxol/repo/memory"
	"claim-pnc/internal/inboxxol/usecase"
)

const (
	portalUtama = "ASM"
	portalLain  = "SIMASNET"
)

var pemanggil = inboxxol.Caller{Login: "PICTEKNIK01"}

// layanan membentuk Service di atas penyimpanan contoh, dengan pemilih portal yang
// MENOLAK portal selain portal utama.
//
// Penolakannya ditiru dari produksi dengan sengaja: jalur "portal belum siap" ikut
// teruji di sini, bukan hanya nanti saat Oracle ada (`R-20`).
func layanan(t *testing.T) *usecase.Service {
	t.Helper()
	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxxol.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal " + alias + " tidak tersedia")
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service
}

func TestServiceMenolakDirakitTanpaPemilihPortal(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "RepoSelector")
}

func TestDaftarPerjanjianDikembalikanBesertaGroupBusinessnya(t *testing.T) {
	masters, err := layanan(t).ListMasters(context.Background(), portalUtama, pemanggil)
	require.NoError(t, err)
	require.Len(t, masters, 3)

	require.Equal(t, "XOL-001", masters[0].ID)
	require.Equal(t, "Fire, Marine Cargo", masters[0].BusinessGroupNames())

	// Group business tanpa nama menjadi TREATY INWARD, pengganti GET_GROUPBUSINESS_XOL.
	require.Equal(t, "Aneka, TREATY INWARD", masters[1].BusinessGroupNames())
}

// TestNilaiKlaimDibagiKursPerjanjian menguji pemindahan pembagian kurs dari aktivitas
// Pega ke usecase.
//
// Angkanya dipilih supaya kekeliruan membagi dua kali langsung terlihat: 4.650.000.000
// dibagi 15.500 adalah 300.000, sedangkan dibagi dua kali menghasilkan 19,35.
func TestNilaiKlaimDibagiKursPerjanjian(t *testing.T) {
	overview, err := layanan(t).SummarizeClaims(context.Background(), portalUtama, pemanggil, "XOL-001")
	require.NoError(t, err)

	require.Equal(t, "XOL-001", overview.Master.ID)
	require.Len(t, overview.Rows, 2)

	banjir := overview.Rows[0]
	require.Equal(t, "12/03/2024", banjir.LossDate)
	require.Equal(t, "BANJIR", banjir.CauseOfLoss)
	require.InDelta(t, 300_000, banjir.OutstandingValue, 0.001)
	require.InDelta(t, 200_000, banjir.AcceptedValue, 0.001)
}

// TestNamaGroupBusinessDisalinDariPerjanjian menguji perilaku yang mudah terbaca sebagai
// cacat: kolom "Group Business" pada grid utama TIDAK datang dari kueri akumulasi.
//
// `Activity/GetClaimXOL-Act.xml` mengisinya dari `TempMst.Currency`, yaitu nama group
// business perjanjian yang sedang dipilih — sehingga seluruh baris bernilai sama.
func TestNamaGroupBusinessDisalinDariPerjanjian(t *testing.T) {
	overview, err := layanan(t).SummarizeClaims(context.Background(), portalUtama, pemanggil, "XOL-001")
	require.NoError(t, err)

	for _, row := range overview.Rows {
		require.Equal(t, "Fire, Marine Cargo", row.BusinessGroup)
	}
}

// TestPerjanjianTanpaGroupBusinessMenghasilkanDaftarKosongBukanGalat menguji jalur yang
// paling mudah salah ditangani.
//
// Perjanjian yang baru dibuat memang belum punya group business. Menolaknya dengan galat
// akan membuat layar tampak rusak padahal datanya sah.
func TestPerjanjianTanpaGroupBusinessMenghasilkanDaftarKosongBukanGalat(t *testing.T) {
	overview, err := layanan(t).SummarizeClaims(context.Background(), portalUtama, pemanggil, "XOL-003")
	require.NoError(t, err)
	require.Equal(t, "XOL-003", overview.Master.ID)
	require.Empty(t, overview.Rows)
}

func TestPerjanjianYangTidakAdaDitolak(t *testing.T) {
	_, err := layanan(t).SummarizeClaims(context.Background(), portalUtama, pemanggil, "XOL-999")
	require.ErrorIs(t, err, inboxxol.ErrMasterNotFound)
}

func TestPerjanjianWajibDipilih(t *testing.T) {
	_, err := layanan(t).SummarizeClaims(context.Background(), portalUtama, pemanggil, "   ")

	var validation *inboxxol.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxxol.FieldMasterID, validation.Violations[0].Field)
}

// TestRincianMenyatukanKlaimSendiriDanTreatyInward menguji penyusunan DLAList pada
// `GetClaimXOL`: dua kueri, satu daftar, treaty inward selalu paling bawah.
func TestRincianMenyatukanKlaimSendiriDanTreatyInward(t *testing.T) {
	rows, err := layanan(t).Breakdown(context.Background(), portalUtama, pemanggil,
		"XOL-001", "12/03/2024", "BANJIR")
	require.NoError(t, err)
	require.Len(t, rows, 3)

	require.Equal(t, inboxxol.SourceOwnBusiness, rows[0].Source)
	require.Equal(t, inboxxol.SourceOwnBusiness, rows[1].Source)
	require.Equal(t, inboxxol.SourceTreatyInward, rows[2].Source)
}

// TestTreatyInwardTidakIkutDibagiKurs menguji pembedaan yang paling mudah salah.
//
// Baris treaty inward SUDAH dikonversi di kuerinya sendiri, karena setiap barisnya punya
// mata uang berbeda. Membaginya lagi dengan kurs perjanjian akan mengecilkan nilainya
// sebesar 15.500 kali — kekeliruan yang tidak menghasilkan galat apa pun.
func TestTreatyInwardTidakIkutDibagiKurs(t *testing.T) {
	rows, err := layanan(t).Breakdown(context.Background(), portalUtama, pemanggil,
		"XOL-001", "12/03/2024", "BANJIR")
	require.NoError(t, err)

	// Klaim sendiri: 3.100.000.000 / 15.500 = 200.000
	require.InDelta(t, 200_000, rows[0].OutstandingValue, 0.001)
	// Treaty inward: nilainya dibiarkan apa adanya.
	require.InDelta(t, 48_500, rows[2].OutstandingValue, 0.001)
}

// TestBarisTreatyInwardTanpaKursDitandai menguti pengganti `RETURN 1` pada
// `Database/GETCURRENCYSTANDARD.fnc:22`.
//
// Yang diuji bukan angkanya melainkan TANDANYA: nilai nol boleh, nilai karangan tidak.
func TestBarisTreatyInwardTanpaKursDitandai(t *testing.T) {
	rows, err := layanan(t).Breakdown(context.Background(), portalUtama, pemanggil,
		"XOL-001", "28/07/2024", "KEBAKARAN")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	treaty := rows[1]
	require.Equal(t, inboxxol.SourceTreatyInward, treaty.Source)
	require.True(t, treaty.RateMissing, "baris tanpa kurs wajib ditandai")
	require.Zero(t, treaty.OutstandingValue, "nilai tanpa kurs tidak boleh dikarang")
}

func TestRincianMenuntutTanggalDanSebabKerugian(t *testing.T) {
	_, err := layanan(t).Breakdown(context.Background(), portalUtama, pemanggil, "XOL-001", "", "")

	var validation *inboxxol.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2, "seluruh pelanggaran dikembalikan sekaligus (P-5)")

	fields := []string{validation.Violations[0].Field, validation.Violations[1].Field}
	require.ElementsMatch(t, []string{inboxxol.FieldLossDate, inboxxol.FieldCauseOfLoss}, fields)
}

func TestPencarianPemberitahuanMenyaringMenurutTipe(t *testing.T) {
	service := layanan(t)

	pla, err := service.SearchAdvice(context.Background(), portalUtama, pemanggil,
		inboxxol.AdviceFilter{Year: "2024", CauseOfLoss: "BANJIR", Type: inboxxol.AdvicePLA})
	require.NoError(t, err)
	require.Len(t, pla, 2)
	require.Equal(t, "PLA/XOL/2024/0001", pla[0].DisplayNumber())
	require.Equal(t, "PLA/XOL/2024/0002 / 1", pla[1].DisplayNumber())

	dla, err := service.SearchAdvice(context.Background(), portalUtama, pemanggil,
		inboxxol.AdviceFilter{Year: "2024", CauseOfLoss: "BANJIR", Type: inboxxol.AdviceDLA})
	require.NoError(t, err)
	require.Empty(t, dla, "DLA BANJIR memang tidak ada di contoh")
}

func TestPencarianPemberitahuanMenolakTipeYangTidakDikenal(t *testing.T) {
	_, err := layanan(t).SearchAdvice(context.Background(), portalUtama, pemanggil,
		inboxxol.AdviceFilter{Year: "2024", CauseOfLoss: "BANJIR", Type: "SURAT"})

	var validation *inboxxol.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxxol.FieldAdviceType, validation.Violations[0].Field)
}

// TestAntreanPersetujuanMemuatKeduaAntrean menguji tab Komite: dua grid yang berdiri
// sendiri, diambil dalam satu permintaan supaya keduanya berasal dari saat yang sama.
func TestAntreanPersetujuanMemuatKeduaAntrean(t *testing.T) {
	queue, err := layanan(t).ListApprovals(context.Background(), portalUtama, pemanggil)
	require.NoError(t, err)

	require.Len(t, queue.Advices, 2, "satu PLA dan satu DLA yang belum disetujui")
	require.Len(t, queue.Masters, 1, "hanya XOL-003 yang STSKOMITE-nya '0'")
	require.Equal(t, "XOL-003", queue.Masters[0].ID)
}

func TestDaftarSebabKerugianDikembalikan(t *testing.T) {
	causes, err := layanan(t).ListCauseOfLoss(context.Background(), portalUtama, pemanggil)
	require.NoError(t, err)
	require.Len(t, causes, 4)
	require.Equal(t, "12001", causes[0].ID)
	require.Equal(t, "BANJIR", causes[0].Description)
}

// TestPortalLainDitolak menguji `R-20`: tidak ada satu pun jalur yang jatuh ke portal
// utama sebagai cadangan.
//
// Kalau ada, layar akan menampilkan nilai klaim dan nama reasuradur satu badan hukum dari
// basis data badan hukum lain — tanpa satu pun pesan galat.
func TestPortalLainDitolak(t *testing.T) {
	service := layanan(t)
	ctx := context.Background()

	_, err := service.ListMasters(ctx, portalLain, pemanggil)
	require.Error(t, err)

	_, err = service.SummarizeClaims(ctx, portalLain, pemanggil, "XOL-001")
	require.Error(t, err)

	_, err = service.Breakdown(ctx, portalLain, pemanggil, "XOL-001", "12/03/2024", "BANJIR")
	require.Error(t, err)

	_, err = service.SearchAdvice(ctx, portalLain, pemanggil,
		inboxxol.AdviceFilter{Year: "2024", CauseOfLoss: "BANJIR", Type: inboxxol.AdvicePLA})
	require.Error(t, err)

	_, err = service.ListApprovals(ctx, portalLain, pemanggil)
	require.Error(t, err)

	_, err = service.ListCauseOfLoss(ctx, portalLain, pemanggil)
	require.Error(t, err)
}

// TestPemanggilTanpaIdentitasDitolak menjaga setiap jalur menuntut identitas.
func TestPemanggilTanpaIdentitasDitolak(t *testing.T) {
	service := layanan(t)
	ctx := context.Background()
	kosong := inboxxol.Caller{Login: "   "}

	_, err := service.ListMasters(ctx, portalUtama, kosong)
	require.ErrorIs(t, err, inboxxol.ErrCallerUnknown)

	_, err = service.SummarizeClaims(ctx, portalUtama, kosong, "XOL-001")
	require.ErrorIs(t, err, inboxxol.ErrCallerUnknown)

	_, err = service.Breakdown(ctx, portalUtama, kosong, "XOL-001", "12/03/2024", "BANJIR")
	require.ErrorIs(t, err, inboxxol.ErrCallerUnknown)

	_, err = service.SearchAdvice(ctx, portalUtama, kosong,
		inboxxol.AdviceFilter{Year: "2024", CauseOfLoss: "BANJIR", Type: inboxxol.AdvicePLA})
	require.ErrorIs(t, err, inboxxol.ErrCallerUnknown)

	_, err = service.ListApprovals(ctx, portalUtama, kosong)
	require.ErrorIs(t, err, inboxxol.ErrCallerUnknown)

	_, err = service.ListCauseOfLoss(ctx, portalUtama, kosong)
	require.ErrorIs(t, err, inboxxol.ErrCallerUnknown)
}

// TestKursNolTidakMenghasilkanNilaiTakHingga menguti penjagaan pembagian.
//
// Sistem lama membagi tanpa memeriksa apa pun; pembagian dengan nol menghasilkan nilai
// tak hingga yang kemudian ditampilkan sebagai angka. Nol lebih jujur, dan jauh lebih
// mudah terlihat daripada angka yang tampak masuk akal.
func TestKursNolTidakMenghasilkanNilaiTakHingga(t *testing.T) {
	repo := memory.NewRepo(
		memory.WithMasters(inboxxol.MasterXOL{
			ID: "XOL-RUSAK", Year: "2024", ExchangeRate: 0,
			BusinessGroups: []inboxxol.BusinessGroup{{ID: "10", Name: "Fire"}},
		}),
		memory.WithSummaries("2024", inboxxol.ClaimSummary{
			LossDate: "01/01/2024", CauseOfLoss: "BANJIR", OutstandingValue: 1_000_000,
		}),
	)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxxol.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)

	overview, err := service.SummarizeClaims(context.Background(), portalUtama, pemanggil, "XOL-RUSAK")
	require.NoError(t, err)
	require.Len(t, overview.Rows, 1)
	require.Zero(t, overview.Rows[0].OutstandingValue)
}
