package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/kasir"
	"claim-pnc/internal/masterrekening/notifikasi"
	"claim-pnc/internal/masterrekening/repo/memori"
	"claim-pnc/internal/masterrekening/usecase"
	"claim-pnc/internal/platform/waktu"
)

// Seluruh uji di berkas ini berjalan tanpa basis data dan tanpa jaringan: penyimpanan
// di memori dan Kasir tiruan keduanya hidup di dalam proses.

func TestPengajuanBaruSelaluLahirMenungguKeputusanKomite(t *testing.T) {
	rakit := rakitan(t, "ASM")

	rek, err := rakit.layanan.Ajukan(context.Background(), pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	assert.Equal(t, masterrekening.StatusMenunggu, rek.Status,
		"tidak boleh ada jalan bagi pengaju untuk menerbitkan rekening yang langsung disetujui")
	assert.Equal(t, "3171999", rek.DiinputOleh)
	assert.Equal(t, rakit.jam.Sekarang(), rek.DiinputPada)
}

func TestPengajuanTidakLengkapDitolakSebelumMenyentuhPenyimpanan(t *testing.T) {
	rakit := rakitan(t, "ASM")

	p := pengajuanLengkap()
	p.NIK = ""

	_, err := rakit.layanan.Ajukan(context.Background(), p, pengaju())

	var validasi *masterrekening.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	assert.Contains(t, validasi.Field, "nik")

	_, jumlah, err := rakit.layanan.Daftar(context.Background(), masterrekening.Filter{})
	require.NoError(t, err)
	assert.Zero(t, jumlah, "pengajuan yang gagal validasi tidak boleh tersimpan")
}

func TestNomorRekeningYangMasihMenungguTidakDapatDiajukanLagi(t *testing.T) {
	rakit := rakitan(t, "ASM")
	_, err := rakit.layanan.Ajukan(context.Background(), pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	_, err = rakit.layanan.Ajukan(context.Background(), pengajuanLengkap(), pengaju())
	assert.ErrorIs(t, err, masterrekening.ErrSudahAda)
}

func TestNomorRekeningYangDitolakKomiteDapatDiajukanUlang(t *testing.T) {
	// Perilaku sistem lama dipertahankan apa adanya (P-5): baris bekas penolakan
	// dibuang, lalu pengajuan baru disisipkan.
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	_, err = rakit.layanan.Putuskan(ctx, rek.KunciDari(), usecase.Keputusan{
		Status:  masterrekening.StatusDitolak,
		Catatan: "Buku rekening tidak sesuai.",
	}, komite(), nil)
	require.NoError(t, err)

	ulang, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusMenunggu, ulang.Status)

	_, jumlah, err := rakit.layanan.Daftar(ctx, masterrekening.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, jumlah, "baris bekas penolakan tidak boleh tertinggal sebagai baris kedua")
}

func TestKomiteTidakDapatMenyetujuiTanpaBukuRekening(t *testing.T) {
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	p := pengajuanLengkap()
	p.IDDokumen = ""
	rek, err := rakit.layanan.Ajukan(ctx, p, pengaju())
	require.NoError(t, err)

	_, err = rakit.layanan.Putuskan(ctx, rek.KunciDari(), usecase.Keputusan{
		Status:  masterrekening.StatusDisetujui,
		Catatan: "Disetujui atasan.",
	}, komite(), nil)

	var validasi *masterrekening.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	assert.Contains(t, validasi.Field, "id_dokumen")

	tersimpan, err := rakit.layanan.Ambil(ctx, rek.KunciDari())
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusMenunggu, tersimpan.Status,
		"persetujuan yang ditolak validasi tidak boleh mengubah status")
}

func TestPenolakanTidakMenuntutBukuRekening(t *testing.T) {
	// Rekening ditolak justru sering karena buktinya tidak ada; menuntut buku rekening
	// untuk menolak akan mengunci komite pada rekening yang justru ingin ditolaknya.
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	p := pengajuanLengkap()
	p.IDDokumen = ""
	rek, err := rakit.layanan.Ajukan(ctx, p, pengaju())
	require.NoError(t, err)

	ditolak, err := rakit.layanan.Putuskan(ctx, rek.KunciDari(), usecase.Keputusan{
		Status:  masterrekening.StatusDitolak,
		Catatan: "Buku rekening tidak dilampirkan.",
	}, komite(), nil)
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusDitolak, ditolak.Status)
}

func TestKeputusanKomiteTidakDapatDiambilDuaKali(t *testing.T) {
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	_, err = rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	_, err = rakit.layanan.Putuskan(ctx, rek.KunciDari(), usecase.Keputusan{
		Status:  masterrekening.StatusDitolak,
		Catatan: "Berubah pikiran.",
	}, komite(), nil)
	assert.ErrorIs(t, err, masterrekening.ErrSudahDiputuskan)
}

func TestRekeningYangDisetujuiDidaftarkanKeKasir(t *testing.T) {
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	hasil, err := rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	require.Len(t, rakit.kasir.Didaftarkan, 1)
	assert.Empty(t, rakit.kasir.Diperbarui)
	assert.Equal(t, "BERHASIL", hasil.StatusLayanan)
	assert.Equal(t, "TIRUAN-0001", hasil.IDRekeningKasir)
	assert.Equal(t, "Rekening diterima sistem Kasir.", hasil.ResponsKasir,
		"pesan Kasir dipangkas sampai setelah tanda ] seperti layar lama")
}

func TestRekeningPenggantiDikirimLewatJalurPembaruanKasir(t *testing.T) {
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	p := pengajuanLengkap()
	p.NomorRekeningLama = "0987654321"
	p.KodeBankLama = "008"
	rek, err := rakit.layanan.Ajukan(ctx, p, pengaju())
	require.NoError(t, err)

	_, err = rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	assert.Empty(t, rakit.kasir.Didaftarkan)
	require.Len(t, rakit.kasir.Diperbarui, 1)
}

func TestPortalDiLuarASMDanSIMASNETTidakMenyentuhKasir(t *testing.T) {
	// Prasyarat aslinya: TempGetApp.LSC_ID=="ASM" || TempGetApp.LSC_ID=="SIMASNET".
	rakit := rakitan(t, "SAS")
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	hasil, err := rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	assert.Empty(t, rakit.kasir.Didaftarkan)
	assert.Empty(t, hasil.StatusLayanan)
	assert.Equal(t, masterrekening.StatusDisetujui, hasil.Status,
		"portal tanpa Kasir tetap boleh menyetujui rekening")
}

func TestKasirYangMatiTidakMembatalkanKeputusanKomite(t *testing.T) {
	// Keputusan komite adalah fakta bisnis yang sudah terjadi. Kegagalan jaringan
	// tidak boleh membuangnya dan memaksa komite memutuskan ulang.
	rakit := rakitan(t, "ASM")
	rakit.kasir.Galat = errors.New("koneksi terputus")
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	hasil, err := rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusDisetujui, hasil.Status)
	assert.Equal(t, "GAGAL", hasil.StatusLayanan)

	tersimpan, err := rakit.layanan.Ambil(ctx, rek.KunciDari())
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusDisetujui, tersimpan.Status)

	// TIDAK ada surel. Sistem lama hanya mengirim surel pada ResponseCode "9", dan
	// alurnya dipertahankan apa adanya — tidak ada pemicu baru yang ditambahkan.
	// Kegagalannya tetap terlihat: tercatat di log dan pada kolom Kasir di layar.
	assert.Empty(t, rakit.notifier.Terkirim,
		"kegagalan menghubungi Kasir tidak memicu surel di sistem lama")
}

func TestKodeResponsSembilanMemicuPeringatanKeTimIT(t *testing.T) {
	// Meniru SendEmailAlertRekening: CekStatus.pxResults(1).ResponseCode == "9".
	rakit := rakitan(t, "ASM")
	rakit.kasir.Jawaban = masterrekening.HasilKasir{
		Berhasil: false,
		Kode:     "9",
		Pesan:    "[ERR-09] Rekening sudah terdaftar di Kasir.",
	}
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	hasil, err := rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	assert.Equal(t, "GAGAL", hasil.StatusLayanan)
	assert.Equal(t, "Rekening sudah terdaftar di Kasir.", hasil.ResponsKasir)

	require.Len(t, rakit.notifier.Terkirim, 1)
	peringatan := rakit.notifier.Terkirim[0]
	assert.Equal(t, "9", peringatan.Kode)
	assert.Equal(t, "1234567890", peringatan.Rekening.NomorRekening)
	// Komite yang memutuskan ikut disebut sebagai KETERANGAN, supaya Tim IT tahu
	// kepada siapa harus bertanya. Ia bukan penerima surelnya.
	assert.Equal(t, "Komite Contoh", peringatan.Diputuskan.Nama)
}

func TestKodeResponsSatuGagalTanpaMemicuPeringatan(t *testing.T) {
	// Kode "1" gagal tetapi tidak mengirim surel; hanya "9" yang memicunya.
	rakit := rakitan(t, "ASM")
	rakit.kasir.Jawaban = masterrekening.HasilKasir{Berhasil: false, Kode: "1", Pesan: "[ERR-01] Ditolak."}
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)

	hasil, err := rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	assert.Equal(t, "GAGAL", hasil.StatusLayanan)
	assert.Empty(t, rakit.notifier.Terkirim)
}

func TestRekeningYangSudahDiputuskanTidakDapatDiubah(t *testing.T) {
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	rek, err := rakit.layanan.Ajukan(ctx, pengajuanLengkap(), pengaju())
	require.NoError(t, err)
	_, err = rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	p := pengajuanLengkap()
	p.NamaPemilik = "NAMA LAIN"
	_, err = rakit.layanan.Ubah(ctx, rek.KunciDari(), p, pengaju())
	assert.ErrorIs(t, err, masterrekening.ErrSudahDiputuskan)
}

func TestSaringanStatusMemisahkanKeempatTabLayar(t *testing.T) {
	rakit := rakitan(t, "ASM")
	ctx := context.Background()

	menunggu := pengajuanLengkap()
	_, err := rakit.layanan.Ajukan(ctx, menunggu, pengaju())
	require.NoError(t, err)

	disetujui := pengajuanLengkap()
	disetujui.NomorRekening = "2222222222"
	rek, err := rakit.layanan.Ajukan(ctx, disetujui, pengaju())
	require.NoError(t, err)
	_, err = rakit.layanan.Putuskan(ctx, rek.KunciDari(), setuju(), komite(), nil)
	require.NoError(t, err)

	_, jumlahMenunggu, err := rakit.layanan.Daftar(ctx, masterrekening.Filter{Status: masterrekening.StatusMenunggu})
	require.NoError(t, err)
	assert.Equal(t, 1, jumlahMenunggu)

	_, jumlahDisetujui, err := rakit.layanan.Daftar(ctx, masterrekening.Filter{Status: masterrekening.StatusDisetujui})
	require.NoError(t, err)
	assert.Equal(t, 1, jumlahDisetujui)

	_, jumlahDitolak, err := rakit.layanan.Daftar(ctx, masterrekening.Filter{Status: masterrekening.StatusDitolak})
	require.NoError(t, err)
	assert.Zero(t, jumlahDitolak)
}

// --- bahan uji ---

type bahan struct {
	layanan  *usecase.Layanan
	kasir    *kasir.Tiruan
	notifier *notifikasi.Tiruan
	jam      *waktu.JamTetap
}

func rakitan(t *testing.T, portal string) bahan {
	t.Helper()

	jam := waktu.JamTetapPada(time.Date(2026, 9, 17, 3, 0, 0, 0, time.UTC))
	tiruan := kasir.TiruanBaru()
	notifier := &notifikasi.Tiruan{}

	return bahan{
		layanan: usecase.LayananBaru(usecase.Opsi{
			Repo:        memori.RepoBaru(),
			Bank:        memori.BankRepoBaru(memori.DaftarBankContoh()...),
			Kasir:       tiruan,
			Notifier:    notifier,
			Jam:         jam,
			PortalAlias: portal,
			KomiteBaku:  "KOMITE-01",
		}),
		kasir:    tiruan,
		notifier: notifier,
		jam:      jam,
	}
}

func pengajuanLengkap() usecase.Pengajuan {
	return usecase.Pengajuan{
		NomorRekening: "1234567890",
		NamaPemilik:   "BENGKEL CONTOH SEJAHTERA",
		NamaBank:      "BANK CONTOH",
		CabangBank:    "JAKARTA PUSAT",
		AlamatBank:    "JL. CONTOH NO. 1",
		KodeBank:      "014",
		TipeRekening:  "BIASA",
		Email:         "keuangan@contoh.co.id",
		Telepon:       "0211234567",
		NIK:           "3171000000000000",
		IDDokumen:     "DOK-001",
		Aktif:         true,
	}
}

func pengaju() usecase.Pengaju {
	return usecase.Pengaju{Identitas: "3171999", Nama: "Petugas Contoh", Email: "petugas@sinarmas.id"}
}

func komite() usecase.Komite {
	return usecase.Komite{
		Identitas: "KOMITE-01",
		Nama:      "Komite Contoh",
		Email:     "komite@sinarmas.id",
	}
}

func setuju() usecase.Keputusan {
	return usecase.Keputusan{
		Status:  masterrekening.StatusDisetujui,
		Catatan: "Disetujui atasan, buku rekening sesuai.",
	}
}
