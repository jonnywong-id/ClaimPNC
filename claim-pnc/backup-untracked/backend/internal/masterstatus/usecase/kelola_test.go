package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatus/repo/memori"
	"claim-pnc/internal/masterstatus/usecase"
)

func layananUji(t *testing.T, isi ...masterstatus.StatusKlaim) (*usecase.Layanan, *memori.Repo) {
	t.Helper()
	if len(isi) == 0 {
		isi = memori.DaftarContoh()
	}
	repo := memori.RepoBaru(isi...)
	layanan, err := usecase.LayananBaru(usecase.Opsi{Repo: repo})
	require.NoError(t, err)
	return layanan, repo
}

func TestLayananMenolakBahanTidakLengkap(t *testing.T) {
	_, err := usecase.LayananBaru(usecase.Opsi{})
	require.Error(t, err, "kegagalan terjadi saat start, bukan saat pengguna membuka layar")
}

func TestDaftarMengembalikan33StatusTerurut(t *testing.T) {
	layanan, _ := layananUji(t)

	daftar, err := layanan.Daftar(context.Background())
	require.NoError(t, err)
	require.Len(t, daftar, 33)

	for i := 1; i < len(daftar); i++ {
		require.Less(t, daftar[i-1].Kode, daftar[i].Kode, "daftar harus terurut menurut kode")
	}
}

func TestAmbilStatusYangAda(t *testing.T) {
	layanan, _ := layananUji(t)

	status, err := layanan.Ambil(context.Background(), "1163")
	require.NoError(t, err)
	require.Equal(t, "Paid", status.Label)
}

func TestAmbilStatusYangTidakAda(t *testing.T) {
	layanan, _ := layananUji(t)

	_, err := layanan.Ambil(context.Background(), "9999")
	require.ErrorIs(t, err, masterstatus.ErrTidakDitemukan)
}

// Kode dibuat penyimpanan, melanjutkan deret yang sudah ada — persis seperti procedure
// lama yang menerima sentinel "UnknownID" lalu menentukan kodenya sendiri.
func TestTambahMenerbitkanKodeBerikutnya(t *testing.T) {
	layanan, _ := layananUji(t)

	status, err := layanan.Tambah(context.Background(), "Status Percobaan")
	require.NoError(t, err)
	require.Equal(t, "1167", status.Kode, "melanjutkan 1166, tidak mengulang dari awal")
	require.Equal(t, "Status Percobaan", status.Label)
	require.Empty(t, status.KodeLama, "status baru tidak pernah diberi penomoran lama")

	daftar, err := layanan.Daftar(context.Background())
	require.NoError(t, err)
	require.Len(t, daftar, 34)
}

func TestTambahDenganLabelKosongDitolak(t *testing.T) {
	layanan, _ := layananUji(t)

	_, err := layanan.Tambah(context.Background(), "   ")

	var validasi *masterstatus.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Pelanggaran, 1)
	require.Equal(t, masterstatus.FieldLabel, validasi.Pelanggaran[0].Field)

	daftar, _ := layanan.Daftar(context.Background())
	require.Len(t, daftar, 33, "tidak ada yang tersimpan saat validasi gagal")
}

func TestTambahDenganLabelYangSudahDipakaiDitolak(t *testing.T) {
	layanan, _ := layananUji(t)

	_, err := layanan.Tambah(context.Background(), "Paid")
	require.ErrorIs(t, err, masterstatus.ErrLabelSudahAda)
}

// Keunikan tidak boleh dapat ditembus hanya dengan mengubah besar-kecil huruf atau
// menambah spasi.
func TestKeunikanLabelMengabaikanBesarKecilHurufDanSpasi(t *testing.T) {
	for _, masukan := range []string{"PAID", "paid", "  Paid  ", "pAiD"} {
		layanan, _ := layananUji(t)
		_, err := layanan.Tambah(context.Background(), masukan)
		require.ErrorIs(t, err, masterstatus.ErrLabelSudahAda,
			"%q seharusnya dikenali sama dengan status Paid yang sudah ada", masukan)
	}
}

func TestUbahMenggantiLabel(t *testing.T) {
	layanan, _ := layananUji(t)

	status, err := layanan.Ubah(context.Background(), "1163", "Sudah Dibayar")
	require.NoError(t, err)
	require.Equal(t, "1163", status.Kode, "kode tidak ikut berubah")
	require.Equal(t, "Sudah Dibayar", status.Label)

	dibaca, err := layanan.Ambil(context.Background(), "1163")
	require.NoError(t, err)
	require.Equal(t, "Sudah Dibayar", dibaca.Label, "perubahan benar-benar tersimpan")
}

// Kode lama adalah jejak sejarah; mengubah label tidak boleh menghapusnya.
func TestUbahTidakMenghilangkanKodeLama(t *testing.T) {
	layanan, _ := layananUji(t)

	sebelum, err := layanan.Ambil(context.Background(), "1134")
	require.NoError(t, err)
	require.Equal(t, "01", sebelum.KodeLama, "prasyarat uji")

	sesudah, err := layanan.Ubah(context.Background(), "1134", "Laporan Ringkas")
	require.NoError(t, err)
	require.Equal(t, "01", sesudah.KodeLama)
}

// Menyimpan ulang tanpa mengubah label tidak boleh ditolak karena bentrok dengan
// dirinya sendiri — kesalahan klasik pada pemeriksaan keunikan.
func TestUbahDenganLabelYangSamaTidakDianggapBentrok(t *testing.T) {
	layanan, _ := layananUji(t)

	status, err := layanan.Ubah(context.Background(), "1163", "Paid")
	require.NoError(t, err)
	require.Equal(t, "Paid", status.Label)
}

func TestUbahKeLabelMilikStatusLainDitolak(t *testing.T) {
	layanan, _ := layananUji(t)

	_, err := layanan.Ubah(context.Background(), "1163", "Register")
	require.ErrorIs(t, err, masterstatus.ErrLabelSudahAda)
}

func TestUbahLabelKosongDitolak(t *testing.T) {
	layanan, _ := layananUji(t)

	_, err := layanan.Ubah(context.Background(), "1163", "")

	var validasi *masterstatus.GalatValidasi
	require.ErrorAs(t, err, &validasi)

	tetap, _ := layanan.Ambil(context.Background(), "1163")
	require.Equal(t, "Paid", tetap.Label, "label lama tidak tertimpa saat validasi gagal")
}

// Mengubah status yang tidak ada dijawab "tidak ditemukan", bukan galat lain yang
// menyesatkan — kode diperiksa sebelum keunikan label.
func TestUbahStatusYangTidakAda(t *testing.T) {
	layanan, _ := layananUji(t)

	_, err := layanan.Ubah(context.Background(), "9999", "Apa Saja")
	require.ErrorIs(t, err, masterstatus.ErrTidakDitemukan)
}

func TestUbahStatusYangTidakAdaDenganLabelBentrok(t *testing.T) {
	layanan, _ := layananUji(t)

	_, err := layanan.Ubah(context.Background(), "9999", "Paid")
	require.ErrorIs(t, err, masterstatus.ErrTidakDitemukan,
		"kode yang tidak ada lebih dulu dilaporkan daripada label yang bentrok")
}

// Galat penyimpanan dibungkus konteks, tidak ditelan dan tidak disamarkan menjadi
// "tidak ditemukan".
func TestGalatPenyimpananDibungkusBukanDitelan(t *testing.T) {
	layanan, repo := layananUji(t)
	rusak := errors.New("koneksi terputus")
	repo.SetGalat(rusak)

	_, err := layanan.Daftar(context.Background())
	require.ErrorIs(t, err, rusak)
	require.Contains(t, err.Error(), "masterstatus/usecase", "jejaknya terbaca dari pesan")

	_, err = layanan.Ambil(context.Background(), "1163")
	require.ErrorIs(t, err, rusak)
	require.NotErrorIs(t, err, masterstatus.ErrTidakDitemukan)
}

// Spasi tepi pada masukan pengguna dibuang sebelum disimpan, supaya label tidak
// tersimpan dengan spasi yang tak terlihat siapa pun.
func TestSpasiTepiDibuangSebelumDisimpan(t *testing.T) {
	layanan, _ := layananUji(t)

	status, err := layanan.Tambah(context.Background(), "   Status Baru   ")
	require.NoError(t, err)
	require.Equal(t, "Status Baru", status.Label)
}

// Tidak ada Hapus — dan itu disengaja. Uji ini mengunci ketiadaan itu supaya penambahan
// operasi hapus menjadi keputusan sadar, bukan kelalaian yang lolos review.
func TestSeamTidakMenyediakanOperasiHapus(t *testing.T) {
	var seam masterstatus.Repo = memori.RepoBaru()

	// Bila seseorang menambah Hapus pada masterstatus.Repo, penegasan tipe di bawah
	// akan mulai berhasil dan uji ini gagal — memaksa keputusannya dibicarakan.
	_, punyaHapus := seam.(interface {
		Hapus(ctx context.Context, kode string) error
	})
	require.False(t, punyaHapus,
		"layar Pega tidak punya tombol hapus dan ADR-0012 melarang master dihapus permanen")
}
