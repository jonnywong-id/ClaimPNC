package usecase_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/memori"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/platform/waktu"
)

const masaBerlakuUji = 30 * time.Minute

type perkakas struct {
	layanan      *usecase.Layanan
	jam          *waktu.JamTetap
	penggunaRepo *memori.PenggunaRepo
	sesiRepo     *memori.SesiRepo
}

// siapkan merakit layanan lengkap tanpa basis data dan tanpa jaringan: provider
// identitas tiruan dan penyimpanan di memori keduanya bekerja di dalam proses.
func siapkan(t *testing.T) perkakas {
	t.Helper()

	sistemIdentitas, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)

	jamUji := waktu.JamTetapPada(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC))
	penggunaRepo := memori.PenggunaRepoBaru()
	sesiRepo := memori.SesiRepoBaru()

	layanan, err := usecase.LayananBaru(usecase.Opsi{
		Identitas:       sistemIdentitas,
		PenggunaRepo:    penggunaRepo,
		SesiRepo:        sesiRepo,
		Jam:             jamUji,
		MasaBerlakuSesi: masaBerlakuUji,
	})
	require.NoError(t, err)

	return perkakas{layanan: layanan, jam: jamUji, penggunaRepo: penggunaRepo, sesiRepo: sesiRepo}
}

func kredensialSah() auth.Kredensial {
	return auth.Kredensial{NamaPengguna: "adminpnc", KataSandi: "rahasia123"}
}

func TestMasukMenerbitkanSesiYangDapatDipakai(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)
	require.NotEmpty(t, hasil.Token)
	require.Equal(t, "90000001", hasil.Pengguna.Identitas)
	require.Equal(t, p.jam.Sekarang().Add(masaBerlakuUji), hasil.Sesi.BerlakuSampai)

	konteks, err := p.layanan.Periksa(ctx, hasil.Token)
	require.NoError(t, err)
	require.Equal(t, hasil.Pengguna.Identitas, konteks.Pengguna.Identitas)
}

// Token yang diterbitkan tidak memuat kredensial dan tidak memuat data nasabah —
// dibuktikan dengan membongkar isinya (TKT-F3-003).
func TestTokenTidakMemuatKredensialMaupunIdentitas(t *testing.T) {
	p := siapkan(t)

	hasil, err := p.layanan.Masuk(context.Background(), kredensialSah())
	require.NoError(t, err)

	isi, err := base64.RawURLEncoding.DecodeString(string(hasil.Token))
	require.NoError(t, err, "token harus berupa nilai acak yang dapat dibongkar, bukan teks bermakna")
	require.Len(t, isi, 32, "token harus 32 byte acak")

	mentah := strings.ToLower(string(hasil.Token) + " " + string(isi))
	for _, rahasia := range []string{"rahasia123", "adminpnc", "90000001", "contoh administrator", "example.invalid"} {
		require.NotContains(t, mentah, strings.ToLower(rahasia),
			"token tidak boleh memuat kredensial maupun identitas pengguna")
	}
}

// Yang tersimpan adalah sidik token, bukan tokennya. Bocornya isi tabel sesi tidak
// dengan sendirinya memberi orang lain sesi yang dapat dipakai.
func TestTokenMentahTidakPernahTersimpan(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)

	tersimpan, err := p.sesiRepo.AmbilBySidikToken(ctx, hasil.Token.Sidik())
	require.NoError(t, err)
	require.NotEqual(t, string(hasil.Token), tersimpan.SidikToken)
	require.Len(t, tersimpan.SidikToken, 64, "sidik SHA-256 heksadesimal")
	require.NotEqual(t, tersimpan.ID, tersimpan.SidikToken, "pengenal sesi tidak boleh diturunkan dari sidik token")
}

// Token kedaluwarsa ditolak dengan galat yang DAPAT DIBEDAKAN dari token tidak sah
// (TKT-F3-003). Frontend memakai pembedaan ini untuk menyelamatkan isian yang belum
// tersimpan, bukan sekadar melempar pengguna ke layar masuk.
func TestSesiKedaluwarsaDibedakanDariTokenTidakSah(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)

	p.jam.Maju(masaBerlakuUji + time.Second)

	_, err = p.layanan.Periksa(ctx, hasil.Token)
	require.ErrorIs(t, err, auth.ErrSesiKedaluwarsa)
	require.NotErrorIs(t, err, auth.ErrSesiTidakDitemukan)

	_, err = p.layanan.Periksa(ctx, auth.Token("token-yang-tidak-pernah-diterbitkan"))
	require.ErrorIs(t, err, auth.ErrSesiTidakDitemukan)
	require.NotErrorIs(t, err, auth.ErrSesiKedaluwarsa)
}

// Mencabut sesi membuat permintaan berikutnya ditolak SEKETIKA — bukan menunggu masa
// berlaku habis (TKT-F3-003). Jam sengaja tidak digeser sedetik pun di uji ini.
func TestPencabutanBerlakuSeketika(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)

	_, err = p.layanan.Periksa(ctx, hasil.Token)
	require.NoError(t, err)

	require.NoError(t, p.layanan.Keluar(ctx, hasil.Token))

	_, err = p.layanan.Periksa(ctx, hasil.Token)
	require.ErrorIs(t, err, auth.ErrSesiDicabut)
	require.True(t, p.jam.Sekarang().Before(hasil.Sesi.BerlakuSampai),
		"sesi masih dalam masa berlaku; penolakannya harus karena pencabutan")
}

// Keluar mencabut sesi di server, bukan menghapus barisnya (ADR-0012).
func TestKeluarTidakMenghapusBarisSesi(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)
	require.NoError(t, p.layanan.Keluar(ctx, hasil.Token))

	require.Equal(t, 1, p.sesiRepo.Jumlah())
	tersimpan, err := p.sesiRepo.AmbilBySidikToken(ctx, hasil.Token.Sidik())
	require.NoError(t, err)
	require.NotNil(t, tersimpan.DicabutPada)
}

// Dua instans aplikasi mengenali sesi yang sama: sesi diterbitkan instans A dan dipakai
// di instans B. Keduanya berbagi penyimpanan, persis seperti dua instans di belakang
// load balancer berbagi satu basis data (D-27).
//
// CATATAN KETERBATASAN: uji ini memakai penyimpanan di memori yang dibagi dua layanan.
// Ia membuktikan bahwa layanan tidak menyimpan state di dirinya sendiri, TETAPI belum
// membuktikan perilaku yang sama terhadap Oracle. Pembuktian itu menunggu basis data
// pengembangan tersedia.
func TestSesiDikenaliInstansLain(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	sistemIdentitas, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)
	instansB, err := usecase.LayananBaru(usecase.Opsi{
		Identitas:       sistemIdentitas,
		PenggunaRepo:    p.penggunaRepo,
		SesiRepo:        p.sesiRepo,
		Jam:             p.jam,
		MasaBerlakuSesi: masaBerlakuUji,
	})
	require.NoError(t, err)

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)

	konteks, err := instansB.Periksa(ctx, hasil.Token)
	require.NoError(t, err)
	require.Equal(t, hasil.Pengguna.Identitas, konteks.Pengguna.Identitas)

	require.NoError(t, instansB.Keluar(ctx, hasil.Token))
	_, err = p.layanan.Periksa(ctx, hasil.Token)
	require.ErrorIs(t, err, auth.ErrSesiDicabut, "pencabutan di satu instans berlaku di instans lain")
}

func TestPerpanjangMenggeserBatasBerlaku(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)

	p.jam.Maju(20 * time.Minute)
	diperpanjang, err := p.layanan.Perpanjang(ctx, hasil.Token)
	require.NoError(t, err)
	require.Equal(t, p.jam.Sekarang().Add(masaBerlakuUji), diperpanjang.BerlakuSampai)

	p.jam.Maju(masaBerlakuUji - time.Minute)
	_, err = p.layanan.Periksa(ctx, hasil.Token)
	require.NoError(t, err, "sesi yang sudah diperpanjang masih berlaku")
}

func TestSesiKedaluwarsaTidakDapatDiperpanjang(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)

	p.jam.Maju(masaBerlakuUji + time.Second)
	_, err = p.layanan.Perpanjang(ctx, hasil.Token)
	require.ErrorIs(t, err, auth.ErrSesiKedaluwarsa)
}

// Profil yang salah satu fieldnya kosong ditolak — pengguna yang berhasil masuk tetapi
// cabangnya kosong tidak dapat dicocokkan dengan data klaimnya sendiri.
func TestProfilTidakLengkapDitolak(t *testing.T) {
	p := siapkan(t)

	_, err := p.layanan.Masuk(context.Background(),
		auth.Kredensial{NamaPengguna: "profilbolong", KataSandi: "rahasia123"})
	require.Error(t, err)

	var galat *auth.GalatProfilTidakLengkap
	require.ErrorAs(t, err, &galat)
	require.Equal(t, []string{"nama"}, galat.FieldKosong)
	require.Equal(t, 0, p.sesiRepo.Jumlah(), "tidak ada sesi yang boleh terbit")
}

func TestKredensialKosongDijawabSamaDenganKredensialSalah(t *testing.T) {
	p := siapkan(t)

	_, errKosong := p.layanan.Masuk(context.Background(), auth.Kredensial{})
	_, errSalah := p.layanan.Masuk(context.Background(),
		auth.Kredensial{NamaPengguna: "adminpnc", KataSandi: "salah"})

	require.ErrorIs(t, errKosong, auth.ErrKredensialSalah)
	require.ErrorIs(t, errSalah, auth.ErrKredensialSalah)
}

// Pengguna yang dinonaktifkan setelah masuk kehilangan aksesnya pada permintaan
// berikutnya — status aktif dibaca dari basis data, tidak tertanam di sesi.
func TestPenggunaDinonaktifkanDitolakPadaPermintaanBerikutnya(t *testing.T) {
	p := siapkan(t)
	ctx := context.Background()

	hasil, err := p.layanan.Masuk(ctx, kredensialSah())
	require.NoError(t, err)

	p.penggunaRepo.SetAktif(hasil.Pengguna.Identitas, false)

	_, err = p.layanan.Periksa(ctx, hasil.Token)
	require.ErrorIs(t, err, auth.ErrPenggunaTidakAktif)
}

func TestKeluarDenganTokenTidakDikenalBukanGalat(t *testing.T) {
	p := siapkan(t)
	require.NoError(t, p.layanan.Keluar(context.Background(), auth.Token("bukan token siapa pun")))
}

func TestLayananMenolakBahanTidakLengkap(t *testing.T) {
	_, err := usecase.LayananBaru(usecase.Opsi{})
	require.Error(t, err)
}
