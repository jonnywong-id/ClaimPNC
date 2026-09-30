package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/repo/memory"
	"claim-pnc/internal/inboxbandinghargasalvage/usecase"
)

const portalUtama = "ASM"

func layanan(t *testing.T, store inboxbandinghargasalvage.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxbandinghargasalvage.Repo, error) {
			if alias != portalUtama {
				return nil, errPortalTidakDikenal
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

var errPortalTidakDikenal = errors.New("portal tidak dikenal")

func komite(login string) inboxbandinghargasalvage.Caller {
	return inboxbandinghargasalvage.Caller{Login: login}
}

func TestLayananMenolakDibentukTanpaRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Bentuk layar tidak menyentuh basis data sama sekali: daftar tab dan kolomnya sama di
// seluruh entitas, karena ia bentuk layar — bukan data entitas.
func TestBentukLayarTidakMenyentuhPenyimpanan(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxbandinghargasalvage.Repo, error) {
			t.Fatal("Metadata tidak boleh memilih penyimpanan")
			return nil, nil
		},
	})
	require.NoError(t, err)

	meta := service.Metadata()
	require.Len(t, meta.Tabs, 2)
	require.Equal(t, inboxbandinghargasalvage.DefaultTab, meta.DefaultTab)
	require.Equal(t, "Cari No Klaim", meta.SearchLabel)
	require.NotEmpty(t, meta.PlannedDifferences)
	require.NotEmpty(t, meta.Limitations)
}

func TestPermintaanKePortalYangTidakDikenalDitolak(t *testing.T) {
	service := layanan(t, memory.NewSampleStore())

	_, err := service.List(
		context.Background(), "ENTAH", komite(memory.SampleOwner),
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Pagination{})

	require.ErrorIs(t, err, errPortalTidakDikenal)
}

// Portal diperiksa SETELAH permintaannya sah, tetapi identitas pemanggil diperiksa lebih
// dulu — sehingga sesi yang tidak lengkap dijawab sebagai sesi yang tidak lengkap, bukan
// sebagai portal yang salah.
func TestPermintaanTanpaIdentitasDitolakSebelumMemilihPortal(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxbandinghargasalvage.Repo, error) {
			t.Fatal("penyimpanan tidak boleh dipilih sebelum identitas terbaca")
			return nil, nil
		},
	})
	require.NoError(t, err)

	_, listErr := service.List(
		context.Background(), portalUtama, komite("  "),
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Pagination{})

	require.ErrorIs(t, listErr, inboxbandinghargasalvage.ErrCallerUnknown)
}

// Permintaan yang dikembalikan memuat penyaring yang BENAR-BENAR dipakai, bukan yang dikirim
// layar. Itulah yang membuat kotak cari selalu memperlihatkan kata kunci yang menyaring.
func TestHasilMembawaPermintaanYangBenarBenarDipakai(t *testing.T) {
	service := layanan(t, memory.NewSampleStore())

	listed, err := service.List(
		context.Background(), portalUtama, komite(memory.SampleOwner),
		inboxbandinghargasalvage.QueryInput{Keyword: "  pncn.26.0451 "},
		inboxbandinghargasalvage.Pagination{})

	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0451", listed.Query.Keyword)
	require.Equal(t, inboxbandinghargasalvage.DefaultTab, listed.Query.Tab.Code)
	require.Equal(t, memory.SampleOwner, listed.Query.Reviewer.Name)
}

// Tabel ringkas menghitung KEDUA tab, bukan hanya yang sedang terbuka.
func TestRingkasanMenghitungKeduaTab(t *testing.T) {
	service := layanan(t, memory.NewSampleStore())

	summary, err := service.Summary(context.Background(), portalUtama, komite(memory.SampleOwner))

	require.NoError(t, err)
	require.Len(t, summary.Rows, 2)
	require.Equal(t, "Request Banding Harga", summary.Rows[0].Label)
	require.Equal(t, inboxbandinghargasalvage.TabRequest, summary.Rows[0].Tab)
	require.Equal(t, "History Cheker", summary.Rows[1].Label)
	require.Equal(t, inboxbandinghargasalvage.TabHistory, summary.Rows[1].Tab)
}

// Selisih terencana nomor 4, diuji lewat lapisan yang benar-benar dipakai layar: angka pada
// tabel ringkas harus sama dengan jumlah baris yang dikembalikan daftarnya.
func TestAngkaRingkasSamaDenganJumlahBarisDaftarnya(t *testing.T) {
	service := layanan(t, memory.NewSampleStore())
	ctx := context.Background()

	summary, err := service.Summary(ctx, portalUtama, komite(memory.SampleOwner))
	require.NoError(t, err)

	for _, row := range summary.Rows {
		listed, err := service.List(
			ctx, portalUtama, komite(memory.SampleOwner),
			inboxbandinghargasalvage.QueryInput{Tab: row.Tab},
			inboxbandinghargasalvage.Pagination{Page: 1, Size: inboxbandinghargasalvage.MaxPageSize})
		require.NoError(t, err)

		require.Equal(t, row.Count, listed.Page.Total, "tab %s", row.Tab)
	}
}

// Ringkasan ikut disaring menurut komitenya. Tanpa itu, angka di kepala layar akan menyatakan
// pekerjaan orang lain.
func TestRingkasanDisaringMenurutKomitePemanggil(t *testing.T) {
	service := layanan(t, memory.NewSampleStore())
	ctx := context.Background()

	milikSaya, err := service.Summary(ctx, portalUtama, komite(memory.SampleOwner))
	require.NoError(t, err)

	milikOrangLain, err := service.Summary(ctx, portalUtama, komite("KOMITELAIN"))
	require.NoError(t, err)

	require.NotEqual(t, milikSaya.Rows[0].Count, milikOrangLain.Rows[0].Count)
}

// Satu Operator ID membaca antrean komite lain — aturan bernama orang yang ditiru dari Pega.
// Uji ini memastikan ia benar-benar mengalir sampai ke penyimpanan, bukan berhenti di domain.
func TestAntreanYangDiwakilkanBenarBenarMembacaAntreanKomiteLain(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0001", DetailObject: "D1",
			CommitteeName: "BAMBANGSETIADJIGUNAWAN",
			RequestPrice:  "100", InDetailSalvage: true,
		},
	)
	service := layanan(t, store)

	listed, err := service.List(
		context.Background(), portalUtama, komite("MARIATRIELSA"),
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Pagination{})

	require.NoError(t, err)
	require.True(t, listed.Query.Reviewer.Delegated)
	require.Len(t, listed.Page.Items, 1,
		"barisnya milik komite lain, dan memang itulah yang harus tampil")
	require.Equal(t, "BAMBANGSETIADJIGUNAWAN", listed.Page.Items[0].CommitteeName)
}

// ============================================================================
// Tombol Approve dan Reject
// ============================================================================

// layananPenulis menyerahkan layanan yang PEMBACA dan PENULIS-nya berbagi satu penyimpanan,
// sehingga akibat sebuah keputusan langsung terbaca pada daftarnya.
func layananPenulis(t *testing.T) (*usecase.Service, *memory.Store) {
	t.Helper()

	store := memory.NewSampleStore()
	writer := memory.NewWriter(store)

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxbandinghargasalvage.Repo, error) {
			if alias != portalUtama {
				return nil, errPortalTidakDikenal
			}
			return store, nil
		},
		WriterSelector: func(alias string) (inboxbandinghargasalvage.Writer, error) {
			if alias != portalUtama {
				return nil, errPortalTidakDikenal
			}
			return writer, nil
		},
	})
	require.NoError(t, err)

	return service, store
}

func isian(detail, salvage string, setujui bool) inboxbandinghargasalvage.DecisionInput {
	return inboxbandinghargasalvage.DecisionInput{
		DetailObject: detail,
		SalvageID:    salvage,
		RequestPrice: "9750000.00",
		Note:         "sudah dibandingkan",
		Approve:      setujui,
	}
}

func TestKeputusanTersimpanDanTerbacaPadaDaftarYangSama(t *testing.T) {
	service, _ := layananPenulis(t)
	ctx := context.Background()
	pemanggil := komite(memory.SampleOwner)

	result, err := service.Decide(ctx, portalUtama, pemanggil,
		isian("PNC-0451/1", "451", true))
	require.NoError(t, err)
	require.True(t, result.Recorded)

	// Ia hilang dari antrean, dan klaimnya muncul di riwayat putusan.
	decided, err := service.Decisions(ctx, portalUtama, pemanggil, "PNCN.26.0451")
	require.NoError(t, err)
	require.Len(t, decided.Items, 1)
	require.Equal(t, inboxbandinghargasalvage.DecisionApproved, decided.Items[0].Status)
}

// Layanan ini TIDAK menyaring sendiri milik siapa barisnya — penyaringnya ada di kueri, dan
// di sini yang diuji adalah bahwa galatnya sampai ke pemanggil apa adanya.
func TestKeputusanAtasBarisYangSudahDiputusDitolak(t *testing.T) {
	service, _ := layananPenulis(t)
	ctx := context.Background()
	pemanggil := komite(memory.SampleOwner)

	_, err := service.Decide(ctx, portalUtama, pemanggil, isian("PNC-0451/1", "451", true))
	require.NoError(t, err)

	_, err = service.Decide(ctx, portalUtama, pemanggil, isian("PNC-0451/1", "451", true))
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrAlreadyDecided)
}

func TestKeputusanTanpaIdentitasDitolakSebelumMenyentuhPenyimpanan(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxbandinghargasalvage.Repo, error) {
			return memory.NewSampleStore(), nil
		},
		WriterSelector: func(string) (inboxbandinghargasalvage.Writer, error) {
			t.Fatal("penyimpanan tidak boleh dipilih sebelum identitas terbaca")
			return nil, nil
		},
	})
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalUtama,
		inboxbandinghargasalvage.Caller{}, isian("PNC-0451/1", "451", true))

	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)
}

// Tanpa WriterSelector, layanan ini MEMBACA seperti biasa tetapi menolak menulis. Itu keadaan
// yang nyata: penyimpanan yang tidak menyediakan penulis tetap sah dipakai untuk membaca.
func TestTanpaPenulisLayananTetapMembacaTetapiMenolakMenulis(t *testing.T) {
	service := layanan(t, memory.NewSampleStore())
	ctx := context.Background()
	pemanggil := komite(memory.SampleOwner)

	_, err := service.List(ctx, portalUtama, pemanggil,
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Pagination{})
	require.NoError(t, err)

	_, err = service.Decide(ctx, portalUtama, pemanggil,
		isian("PNC-0451/1", "451", true))
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrWriteNotAvailable)
}

// Keputusan dijalankan pada portal yang diminta, bukan pada koneksi bawaan (`R-20`).
func TestKeputusanPadaPortalTakDikenalDitolak(t *testing.T) {
	service, _ := layananPenulis(t)

	_, err := service.Decide(context.Background(), "PORTAL-ASING",
		komite(memory.SampleOwner), isian("PNC-0451/1", "451", true))

	require.ErrorIs(t, err, errPortalTidakDikenal)
}

// Hasilnya membedakan "tercatat" dari "harganya berubah". Bagi komite biasa keduanya TIDAK
// sama, dan layar menyatakannya — bukan menampilkan "berhasil" yang menyiratkan lebih.
func TestHasilMembedakanTercatatDariHargaBerubah(t *testing.T) {
	service, _ := layananPenulis(t)
	ctx := context.Background()

	biasa, err := service.Decide(ctx, portalUtama, komite(memory.SampleOwner),
		isian("PNC-0451/1", "451", true))
	require.NoError(t, err)
	require.True(t, biasa.Recorded)
	require.False(t, biasa.PriceApplied)

	terakhir, err := service.Decide(ctx, portalUtama, komite("DANIELLISWANDI"),
		isian("PNC-0461/1", "461", true))
	require.NoError(t, err)
	require.True(t, terakhir.Recorded)
	require.True(t, terakhir.PriceApplied)
}
