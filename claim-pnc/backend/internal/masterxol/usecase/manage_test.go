package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/masterxol/notification"
	"claim-pnc/internal/masterxol/repo/memory"
	"claim-pnc/internal/masterxol/usecase"
)

const portalASM = "ASM"

// build menyiapkan layanan di atas penyimpanan memori berisi contoh yang meniru produksi.
func build(t *testing.T) (*usecase.Service, *memory.Repo, *notification.Fake) {
	t.Helper()

	repo := memory.NewSampleRepo()
	notifier := notification.NewFake()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterxol.Repo, error) {
			if alias != portalASM {
				return nil, masterxol.ErrNotFound
			}
			return repo, nil
		},
		Notifier: notifier,
	})
	require.NoError(t, err)
	return service, repo, notifier
}

func TestLayananMenolakBahanYangTidakLengkap(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Notifier: notification.NewFake()})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterxol.Repo, error) { return nil, nil },
	})
	require.Error(t, err, "Notifier wajib — tiruan dipasang bila surel tidak dikehendaki, bukan nil")
}

func TestDaftarTidakMembawaAnaknya(t *testing.T) {
	service, _, _ := build(t)

	list, err := service.List(context.Background(), portalASM)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	for _, m := range list {
		require.Empty(t, m.Layer, "grid induk tidak menampilkan layer")
		require.Empty(t, m.Business, "grid induk tidak menampilkan bisnis")
	}
}

func TestDaftarTerurutNumerikBukanLeksikografis(t *testing.T) {
	service, _, _ := build(t)

	list, err := service.List(context.Background(), portalASM)
	require.NoError(t, err)
	require.Equal(t, []string{"10001", "10002", "10004", "10008"}, idOf(list))
}

func TestMembukaSatuIndukMembawaSeluruhTingkatnya(t *testing.T) {
	service, _, _ := build(t)

	master, err := service.Get(context.Background(), portalASM, "10001")
	require.NoError(t, err)
	require.Len(t, master.Business, 4)
	require.Len(t, master.Layer, 3)
	require.Len(t, master.Layer[0].Reinsurer, 6)

	// LimitIDR dihitung dari limit dolar dikali kurs induk.
	require.Equal(t, masterxol.Amount(14_782_500_000), master.Layer[0].ConvertedLimit)
}

func TestIndukTidakDikenalMenghasilkanErrNotFound(t *testing.T) {
	service, _, _ := build(t)

	_, err := service.Get(context.Background(), portalASM, "99999")
	require.ErrorIs(t, err, masterxol.ErrNotFound)
}

func TestMenyimpanIndukBaruMenerbitkanNomorDanMengajukanKeKomite(t *testing.T) {
	service, _, notifier := build(t)

	result, err := service.Save(context.Background(), portalASM, masterxol.Master{
		Name:         "Section 9",
		Year:         "2026",
		ExchangeRate: 16_000,
		Type:         masterxol.TypeProperty,
		RemarkPIC:    "pengajuan baru",
		Layer: []masterxol.Layer{{
			Name:      "Layer 1",
			Limit:     1_000_000,
			Excess:    500_000,
			Reinsurer: []masterxol.Reinsurer{{ID: "10036322", Name: "SWISS RE", Share: 100}},
		}},
	}, "JONNY")
	require.NoError(t, err)

	// Nomor diterbitkan penyimpanan, melanjutkan yang terbesar — bukan yang terakhir.
	require.Equal(t, "10009", result.Master.ID)
	// Nomor layer berjalan global lintas induk.
	require.Equal(t, "10018", result.Master.Layer[0].ID)
	require.Equal(t, masterxol.Amount(16_000_000_000), result.Master.Layer[0].ConvertedLimit)

	// Simpan SEKALIGUS mengajukan ke komite — itulah yang dilakukan layar lama, tanpa
	// syarat apa pun.
	require.Equal(t, "JONNY", result.Master.PIC)
	require.Equal(t, masterxol.CommitteePending, result.Master.CommitteeStatus)

	require.Equal(t, 1, notifier.Count())
	require.Equal(t, "10009", notifier.Sent()[0].MasterID)
	require.Equal(t, "JONNY", notifier.Sent()[0].SubmittedBy)
}

func TestMenyimpanTetapBerhasilMeskiTotalShareBelum100(t *testing.T) {
	service, _, _ := build(t)

	result, err := service.Save(context.Background(), portalASM, masterxol.Master{
		Name:         "Section 9",
		Year:         "2026",
		ExchangeRate: 16_000,
		Layer: []masterxol.Layer{{
			Name:      "Layer 1",
			Reinsurer: []masterxol.Reinsurer{{ID: "1", Name: "SWISS RE", Share: 60}},
		}},
	}, "JONNY")

	// Tersimpan, dengan peringatan — bukan ditolak. Inilah perilaku sistem lama.
	require.NoError(t, err)
	require.NotEmpty(t, result.Master.ID)
	require.Len(t, result.Warning, 1)
	require.Contains(t, result.Warning[0], "Layer 1")
}

func TestMenyimpanIndukYangSudahDisetujuiMengembalikannyaKeMenunggu(t *testing.T) {
	service, _, _ := build(t)

	// Induk 10004 berstatus disetujui pada data contoh.
	before, err := service.Get(context.Background(), portalASM, "10004")
	require.NoError(t, err)
	require.Equal(t, masterxol.CommitteeApproved, before.CommitteeStatus)

	before.RemarkPIC = "kurs dikoreksi"
	result, err := service.Save(context.Background(), portalASM, before, "JONNY")
	require.NoError(t, err)

	// Strukturnya berubah, sehingga persetujuan atas bentuk sebelumnya tidak lagi berlaku.
	require.Equal(t, masterxol.CommitteePending, result.Master.CommitteeStatus)
	require.Equal(t, "JONNY", result.Master.PIC)
}

func TestIsianTerlaluPanjangMenggagalkanPenyimpanan(t *testing.T) {
	service, _, notifier := build(t)

	_, err := service.Save(context.Background(), portalASM, masterxol.Master{
		Name: string(make([]byte, masterxol.MaxNameLength+1)),
	}, "JONNY")

	var validation *masterxol.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Zero(t, notifier.Count(), "tidak ada pengajuan bila penyimpanan gagal")
}

func TestMenghapusIndukIkutMenghapusSeluruhAnaknya(t *testing.T) {
	service, _, _ := build(t)

	require.NoError(t, service.DeleteMaster(context.Background(), portalASM, "10001"))

	_, err := service.Get(context.Background(), portalASM, "10001")
	require.ErrorIs(t, err, masterxol.ErrNotFound)

	// Nomor layer miliknya tidak boleh dipakai ulang oleh induk lain — anaknya memang
	// ikut terbuang, bukan tertinggal sebagai baris yatim seperti di sistem lama.
	list, err := service.List(context.Background(), portalASM)
	require.NoError(t, err)
	require.Equal(t, []string{"10002", "10004", "10008"}, idOf(list))
}

func TestMenghapusLayerIkutMenghapusReasnya(t *testing.T) {
	service, _, _ := build(t)

	require.NoError(t, service.DeleteLayer(context.Background(), portalASM, "10001", "10001"))

	master, err := service.Get(context.Background(), portalASM, "10001")
	require.NoError(t, err)
	require.Len(t, master.Layer, 2)
	for _, l := range master.Layer {
		require.NotEqual(t, "10001", l.ID)
	}
}

func TestMenghapusLayerMilikIndukLainDitolak(t *testing.T) {
	service, _, _ := build(t)

	// Layer 10011 milik induk 10004, bukan 10001.
	err := service.DeleteLayer(context.Background(), portalASM, "10001", "10011")
	require.ErrorIs(t, err, masterxol.ErrLayerNotFound)
}

func TestMenghapusSatuBarisBisnis(t *testing.T) {
	service, _, _ := build(t)

	require.NoError(t, service.DeleteBusiness(context.Background(), portalASM, "10001", "10013"))

	master, err := service.Get(context.Background(), portalASM, "10001")
	require.NoError(t, err)
	require.Len(t, master.Business, 3)
}

func TestMenghapusSatuBarisReas(t *testing.T) {
	service, _, _ := build(t)

	require.NoError(t, service.DeleteReinsurer(context.Background(), portalASM, "10001", "10036322"))

	master, err := service.Get(context.Background(), portalASM, "10001")
	require.NoError(t, err)
	require.Len(t, master.Layer[0].Reinsurer, 5)
}

func TestBekalLayarMembawaTahunDanTypeSekaligus(t *testing.T) {
	service, _, _ := build(t)

	option, err := service.Form(context.Background(), portalASM)
	require.NoError(t, err)
	require.NotEmpty(t, option.Year)
	require.Len(t, option.Type, 3)
	require.Equal(t, masterxol.TypeProperty, option.Type[0].Code)
	require.Equal(t, "Property / Motor / Engineering", option.Type[0].Label)
}

func TestPilihanBisnisMengikutiTypeXOL(t *testing.T) {
	service, _, _ := build(t)

	property, err := service.BusinessGroup(context.Background(), portalASM, masterxol.TypeProperty)
	require.NoError(t, err)
	require.Contains(t, nameOf(property), "ENGINEERING")
	require.Contains(t, nameOf(property), "MOTOR VEHICLE")
	// HEAVY EQUIPMENT muncul pada Type 1, bukan Type 3, karena grup treaty induknya
	// ENGINEERING. Janggal, dan memang begitu di produksi.
	require.Contains(t, nameOf(property), "HEAVY EQUIPMENT")

	marine, err := service.BusinessGroup(context.Background(), portalASM, masterxol.TypeMarine)
	require.NoError(t, err)
	require.Contains(t, nameOf(marine), "MARINE CARGO")
	require.NotContains(t, nameOf(marine), "HEAVY EQUIPMENT")
}

func TestPenyaringTypeDuaMemangCacatDanCacatnyaDireproduksi(t *testing.T) {
	// Ini bukan uji yang membuktikan sesuatu bekerja — ia membuktikan CACAT PRODUKSI
	// ditiru apa adanya (keputusan Work Owner 2026-09-20, `P-5`).
	//
	// Penyaring `%PA%` / `%GA%` dimaksudkan menjaring PA dan GA, tetapi keduanya bernaung
	// di bawah grup treaty "GENERAL ACCIDENT" yang tidak memuat potongan itu. Yang justru
	// terjaring adalah AVIATION HULL, karena induknya "AVIATION & AEROSPACE" memuat "PA"
	// di dalam kata AEROSPACE.
	service, _, _ := build(t)

	accident, err := service.BusinessGroup(context.Background(), portalASM, masterxol.TypeAccident)
	require.NoError(t, err)

	require.Contains(t, nameOf(accident), "AVIATION HULL")
	require.NotContains(t, nameOf(accident), "PA")
	require.NotContains(t, nameOf(accident), "GA (OTHERS)")
}

func TestTreatyInwardSelaluIkutSebagaiPilihanTanpaID(t *testing.T) {
	service, _, _ := build(t)

	for _, tipe := range masterxol.KnownType() {
		list, err := service.BusinessGroup(context.Background(), portalASM, tipe)
		require.NoError(t, err)

		last := list[len(list)-1]
		require.Equal(t, masterxol.TreatyInwardName, last.Name, string(tipe))
		require.Empty(t, last.ID, "TREATY INWARD memang tidak punya ID di BUSINESSGROUP")
	}
}

func TestPortalYangTidakSiapDitolakBukanDialihkanKePortalUtama(t *testing.T) {
	service, _, _ := build(t)

	// Mengembalikan repo portal utama sebagai cadangan berarti menulis struktur treaty
	// satu badan hukum ke basis data badan hukum lain tanpa pesan galat (`R-20`).
	require.Error(t, service.EnsurePortalReady("SMAS"))
	require.NoError(t, service.EnsurePortalReady(portalASM))

	_, err := service.List(context.Background(), "SMAS")
	require.Error(t, err)
}

func idOf(list []masterxol.Master) []string {
	result := make([]string, 0, len(list))
	for _, m := range list {
		result = append(result, m.ID)
	}
	return result
}

func nameOf(list []masterxol.Business) []string {
	result := make([]string, 0, len(list))
	for _, b := range list {
		result = append(result, b.Name)
	}
	return result
}
