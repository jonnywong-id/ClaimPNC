package masterrekening_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
)

func TestPeriksaMenyebutSeluruhFieldKosongSekaligus(t *testing.T) {
	// Pengguna yang mengisi sembilan kolom berhak tahu seluruh yang kurang dalam satu
	// kali, bukan menemukan satu kesalahan baru pada setiap kali menekan simpan.
	err := masterrekening.Rekening{}.Periksa()
	require.Error(t, err)

	var validasi *masterrekening.GalatValidasi
	require.ErrorAs(t, err, &validasi)

	assert.ElementsMatch(t, []string{
		"nomor_rekening", "nama_pemilik", "nama_bank", "cabang_bank", "alamat_bank",
		"kode_bank", "tipe_rekening", "email", "nik",
	}, kunci(validasi.Field))
}

func TestPeriksaMenerimaRekeningLengkap(t *testing.T) {
	require.NoError(t, rekeningLengkap().Periksa())
}

func TestPeriksaMenolakEmailTanpaBentukYangMasukAkal(t *testing.T) {
	r := rekeningLengkap()
	r.Email = "bukan-email"

	var validasi *masterrekening.GalatValidasi
	require.ErrorAs(t, r.Periksa(), &validasi)
	assert.Contains(t, validasi.Field, "email")
}

func TestKomiteTidakDapatMenyetujuiTanpaBukuRekeningDanKeterangan(t *testing.T) {
	// Dua syarat ini hanya berlaku saat MENYETUJUI. Komite tidak boleh menyetujui
	// rekening yang buktinya tidak dapat dilihat, dan alasannya harus tercatat.
	r := rekeningLengkap()
	r.IDDokumen = ""
	r.Catatan = ""

	var validasi *masterrekening.GalatValidasi
	require.ErrorAs(t, r.PeriksaSebelumDisetujui(), &validasi)
	assert.ElementsMatch(t, []string{"id_dokumen", "catatan"}, kunci(validasi.Field))
}

func TestKomiteDapatMenyetujuiSetelahBuktiDanKeteranganAda(t *testing.T) {
	r := rekeningLengkap()
	r.IDDokumen = "DOK-001"
	r.Catatan = "Disetujui atasan, buku rekening sesuai."

	require.NoError(t, r.PeriksaSebelumDisetujui())
}

func TestNomorBaruSelaluBolehDidaftarkan(t *testing.T) {
	assert.True(t, masterrekening.BolehDidaftarkanUlang(nil))
}

func TestNomorYangDitolakKomiteBolehDiajukanUlang(t *testing.T) {
	ditolak := rekeningLengkap()
	ditolak.Status = masterrekening.StatusDitolak

	assert.True(t, masterrekening.BolehDidaftarkanUlang([]masterrekening.Rekening{ditolak}))
}

func TestNomorYangMasihMenungguAtauSudahDisetujuiTidakBolehDiajukanUlang(t *testing.T) {
	for _, status := range []masterrekening.StatusApproval{
		masterrekening.StatusMenunggu,
		masterrekening.StatusDisetujui,
	} {
		r := rekeningLengkap()
		r.Status = status
		assert.Falsef(t, masterrekening.BolehDidaftarkanUlang([]masterrekening.Rekening{r}),
			"status %q seharusnya menahan pengajuan ulang", status)
	}
}

func TestRekeningDapatDipakaiHanyaBilaDisetujuiDanAktif(t *testing.T) {
	// Dua syarat, bukan satu. Rekening yang dinonaktifkan setelah disetujui tidak
	// boleh lagi menjadi tujuan pembayaran.
	kasus := []struct {
		nama   string
		status masterrekening.StatusApproval
		aktif  bool
		mau    bool
	}{
		{"disetujui dan aktif", masterrekening.StatusDisetujui, true, true},
		{"disetujui tetapi nonaktif", masterrekening.StatusDisetujui, false, false},
		{"menunggu walau aktif", masterrekening.StatusMenunggu, true, false},
		{"ditolak walau aktif", masterrekening.StatusDitolak, true, false},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			r := rekeningLengkap()
			r.Status = k.status
			r.Aktif = k.aktif
			assert.Equal(t, k.mau, r.DapatDipakai())
		})
	}
}

func TestPangkasResponsKasirMengambilBagianSetelahKurungSiku(t *testing.T) {
	// Sistem lama memangkasnya di dalam SQL dengan SUBSTR/INSTR. Pemangkasannya
	// pindah ke Go; hasilnya wajib sama.
	assert.Equal(t, "Rekening sudah terdaftar",
		masterrekening.PangkasResponsKasir("[ERR-01] Rekening sudah terdaftar"))

	// Pesan tanpa kurung siku dikembalikan utuh, bukan menjadi kosong.
	assert.Equal(t, "Berhasil", masterrekening.PangkasResponsKasir("Berhasil"))
	assert.Equal(t, "", masterrekening.PangkasResponsKasir(""))
}

func TestLabelStatusMengikutiSebutanLayarLama(t *testing.T) {
	assert.Equal(t, "Menunggu", masterrekening.StatusMenunggu.Label())
	assert.Equal(t, "Komite Approve", masterrekening.StatusDisetujui.Label())
	assert.Equal(t, "Komite Reject", masterrekening.StatusDitolak.Label())
	assert.Equal(t, "", masterrekening.StatusApproval("7").Label())
}

func rekeningLengkap() masterrekening.Rekening {
	return masterrekening.Rekening{
		NomorRekening: "1234567890",
		NamaPemilik:   "BENGKEL CONTOH SEJAHTERA",
		NamaBank:      "BANK CONTOH",
		CabangBank:    "JAKARTA PUSAT",
		AlamatBank:    "JL. CONTOH NO. 1",
		KodeBank:      "014",
		TipeRekening:  "BIASA",
		Email:         "keuangan@contoh.co.id",
		NIK:           "3171000000000000",
		Aktif:         true,
		Status:        masterrekening.StatusMenunggu,
	}
}

func kunci(m map[string]string) []string {
	hasil := make([]string, 0, len(m))
	for k := range m {
		hasil = append(hasil, k)
	}
	return hasil
}
