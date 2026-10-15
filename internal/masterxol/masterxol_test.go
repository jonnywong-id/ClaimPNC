package masterxol_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterxol"
)

// Uji di berkas ini menguji ATURAN, bukan fungsi. Namanya karena itu menyebut aturannya,
// sehingga daftar uji terbaca sebagai dokumentasi aturan yang selalu mutakhir
// (`docs/Steering/14-TESTING-STRATEGY.md` §3.2).

func TestLimitIDRAdalahLimitDolarDikaliKursInduk(t *testing.T) {
	// Angkanya diambil dari baris produksi yang sebenarnya: lapisan 10001 milik induk
	// 10001 — limit 1.095.000 dolar, kurs 13.500, CONVERT_LIMIT 14.782.500.000.
	require.Equal(t, masterxol.Amount(14_782_500_000), masterxol.ConvertedLimit(1_095_000, 13_500))
}

func TestLimitIDRDihitungUlangSaatDirapikan(t *testing.T) {
	// Nilai yang dikirim pemanggil sengaja dibuat salah. Clean WAJIB menimpanya, karena
	// LimitIDR tidak pernah boleh datang dari luar.
	master := masterxol.Master{
		ExchangeRate: 14_000,
		Layer: []masterxol.Layer{
			{Limit: 1_000_000, ConvertedLimit: 999},
		},
	}
	require.Equal(t, masterxol.Amount(14_000_000_000), master.Clean().Layer[0].ConvertedLimit)
}

func TestTotalShareSebuahLayerAdalahJumlahSeluruhReasnya(t *testing.T) {
	layer := masterxol.Layer{Reinsurer: []masterxol.Reinsurer{
		{Share: 5}, {Share: 75}, {Share: 4}, {Share: 10}, {Share: 2}, {Share: 4},
	}}
	require.Equal(t, masterxol.FullShare, layer.TotalShare())
}

func TestTotalShareBukan100MenghasilkanPeringatanBukanPenolakan(t *testing.T) {
	master := masterxol.Master{
		ExchangeRate: 13_500,
		Layer: []masterxol.Layer{
			{Name: "Layer 1", Reinsurer: []masterxol.Reinsurer{
				{ID: "10036322", Name: "SWISS RE", Share: 40},
				{ID: "10036329", Name: "TUGU REASURANSI INDONESIA", Share: 50},
			}},
		},
	}

	// Peringatan ada …
	warning := masterxol.ShareWarning(master)
	require.Len(t, warning, 1)
	require.Contains(t, warning[0], "Layer 1")
	require.Contains(t, warning[0], "90%")

	// … tetapi TIDAK ada pelanggaran. Inilah perilaku yang ditiru dari sistem lama:
	// langkah penyimpanan di `InsertUpdateMasterXOL` tidak punya prasyarat apa pun,
	// sehingga datanya tersimpan lebih dulu dan pesannya muncul sesudahnya.
	require.Empty(t, masterxol.CheckMaster(master))
}

func TestLayerTanpaReasIkutDiperingatkan(t *testing.T) {
	// Keadaan ini nyata: lapisan 10004 milik induk 10002 tersimpan tanpa satu pun
	// reasuradur, sehingga totalnya 0.
	master := masterxol.Master{Layer: []masterxol.Layer{{Name: "Sub Layer"}}}

	warning := masterxol.ShareWarning(master)
	require.Len(t, warning, 1)
	require.Contains(t, warning[0], "Sub Layer")
}

func TestLayerTepat100PersenTidakDiperingatkan(t *testing.T) {
	master := masterxol.Master{Layer: []masterxol.Layer{
		{Name: "Layer 1", Reinsurer: []masterxol.Reinsurer{{Share: 25}, {Share: 75}}},
	}}
	require.Empty(t, masterxol.ShareWarning(master))
}

func TestLayerTanpaNamaDiperingatkanDenganNomorBarisnya(t *testing.T) {
	// Lapisan tanpa nama benar-benar ada (10017). Pesan yang menyebut nama kosong tidak
	// dapat ditindaklanjuti, jadi yang disebut adalah nomor barisnya.
	master := masterxol.Master{Layer: []masterxol.Layer{
		{Name: "Layer 1", Reinsurer: []masterxol.Reinsurer{{Share: 100}}},
		{},
	}}

	warning := masterxol.ShareWarning(master)
	require.Len(t, warning, 1)
	require.Contains(t, warning[0], "layer baris 2")
}

func TestIsianKosongDiterimaSepertiLayarLama(t *testing.T) {
	// Keempat isian induk bertanda pyRequired=false di layar lama, dan kolomnya nullable.
	// Menolaknya di sini akan menolak data yang hari ini sah.
	require.Empty(t, masterxol.CheckMaster(masterxol.Master{}))
}

func TestTypeXOLKosongDiterima(t *testing.T) {
	// Dua dari delapan induk produksi menyimpan TYPEXOL NULL.
	require.Empty(t, masterxol.CheckMaster(masterxol.Master{Type: masterxol.TypeUnknown}))
	require.Equal(t, "", masterxol.TypeLabel(masterxol.TypeUnknown))
}

func TestIsianTerlaluPanjangDitolakSebelumMenyentuhBasisData(t *testing.T) {
	master := masterxol.Master{
		Name: panjang(masterxol.MaxNameLength + 1),
		Year: panjang(masterxol.MaxYearLength + 1),
	}

	violation := masterxol.CheckMaster(master)
	require.Len(t, violation, 2)
	require.Equal(t, masterxol.FieldName, violation[0].Field)
	require.Equal(t, masterxol.FieldYear, violation[1].Field)
}

func TestSeluruhPelanggaranDikumpulkanSekaligus(t *testing.T) {
	// Layar lama menampilkan pesannya bersamaan; mengembalikannya satu per satu akan
	// membuat pengguna menekan Simpan berkali-kali untuk menemukan kesalahan berikutnya.
	master := masterxol.Master{
		ExchangeRate: -1,
		Name:         panjang(masterxol.MaxNameLength + 1),
		Layer: []masterxol.Layer{
			{Limit: -1, Excess: -1, Reinsurer: []masterxol.Reinsurer{{ID: "1", Share: -5}}},
		},
	}
	require.Len(t, masterxol.CheckMaster(master), 5)
}

func TestAngkaNegatifDitolakMeskiLayarLamaMenerimanya(t *testing.T) {
	// Selisih terencana: kurs, limit, excess, dan share negatif tidak punya arti bisnis
	// apa pun, dan akibatnya baru terlihat jauh kemudian pada perhitungan PLA/DLA.
	violation := masterxol.CheckMaster(masterxol.Master{ExchangeRate: -13500})
	require.Len(t, violation, 1)
	require.Equal(t, masterxol.FieldExchangeRate, violation[0].Field)
}

func TestBarisBisnisKosongSeluruhnyaDitolak(t *testing.T) {
	// Baris ber-ID kosong DITERIMA selama namanya ada — itulah bentuk "TREATY INWARD".
	// Yang ditolak adalah baris yang keduanya kosong, karena ia tidak menyatakan apa pun.
	require.Empty(t, masterxol.CheckMaster(masterxol.Master{
		Business: []masterxol.Business{{Name: masterxol.TreatyInwardName}},
	}))

	violation := masterxol.CheckMaster(masterxol.Master{Business: []masterxol.Business{{}}})
	require.Len(t, violation, 1)
	require.Equal(t, masterxol.FieldBusiness, violation[0].Field)
}

func TestNomorBerikutnyaDimulaiDari10001(t *testing.T) {
	// Ditiru dari `INSERT_UPDATE_MST_XOL.prc:18-22`, dan 10001 memang nomor terkecil yang
	// ada di produksi.
	require.Equal(t, "10001", masterxol.NextID(nil))
	require.Equal(t, "10001", masterxol.NextID([]string{}))
}

func TestNomorBerikutnyaMengikutiYangTerbesarBukanYangTerakhir(t *testing.T) {
	// Nomor produksi berlubang — 10003 pernah dihapus — sehingga "yang terakhir" bukan
	// selalu "yang terbesar".
	require.Equal(t, "10010", masterxol.NextID([]string{"10001", "10009", "10004", "10002"}))
}

func TestNomorBukanAngkaDilewatiBukanMenggagalkanPenerbitan(t *testing.T) {
	require.Equal(t, "10005", masterxol.NextID([]string{"10004", "XX", " "}))
}

func TestPenyaringBisnisMengikutiTypeXOLApaAdanya(t *testing.T) {
	// Ketiganya disalin dari `ShowDetailGroupBisnisXol_Act`, TERMASUK salah ketiknya.
	require.Equal(t,
		[]string{"%PROPERTY%", "%MOTOR%", "%ENGINEERING%"},
		masterxol.BusinessGroupPattern(masterxol.TypeProperty))

	require.Equal(t,
		[]string{"%PA%", "%GA%", "%GA%"},
		masterxol.BusinessGroupPattern(masterxol.TypeAccident))

	// "EQUPMENT", bukan "EQUIPMENT". Salah ketiknya dipertahankan (`P-5`) dan terbukti
	// tidak berakibat: nama itu tidak ada di TREATYGROUPNAME dalam ejaan mana pun.
	require.Equal(t,
		[]string{"%MARINE%", "%HEAVY EQUPMENT%", "%HEAVY EQUPMENT%"},
		masterxol.BusinessGroupPattern(masterxol.TypeMarine))
}

func TestPenyaringBisnisSelaluTigaPola(t *testing.T) {
	// Supaya kuerinya tetap satu teks tetap dengan tiga parameter, bukan dirangkai sesuai
	// jumlah pola — perangkaian teks SQL itulah yang dihapus dari modul ini.
	for _, t2 := range []masterxol.Type{
		masterxol.TypeUnknown, masterxol.TypeProperty, masterxol.TypeAccident, masterxol.TypeMarine,
		masterxol.Type("ngawur"),
	} {
		require.Len(t, masterxol.BusinessGroupPattern(t2), 3, string(t2))
	}
}

func TestSpasiTepiDibuangDariSeluruhIsian(t *testing.T) {
	master := masterxol.Master{
		Name:      "  Section 1  ",
		Year:      " 2018 ",
		Type:      " 1 ",
		RemarkPIC: "  revisi  ",
		Business:  []masterxol.Business{{ID: " 10004 ", Name: " MOTOR VEHICLE "}},
		Layer: []masterxol.Layer{{
			ID:        " 10001 ",
			Name:      " Sub Layer ",
			Reinsurer: []masterxol.Reinsurer{{ID: " 10036322 ", Name: " SWISS RE "}},
		}},
	}.Clean()

	require.Equal(t, "Section 1", master.Name)
	require.Equal(t, "2018", master.Year)
	require.Equal(t, masterxol.TypeProperty, master.Type)
	require.Equal(t, "revisi", master.RemarkPIC)
	require.Equal(t, "10004", master.Business[0].ID)
	require.Equal(t, "MOTOR VEHICLE", master.Business[0].Name)
	require.Equal(t, "10001", master.Layer[0].ID)
	require.Equal(t, "Sub Layer", master.Layer[0].Name)
	require.Equal(t, "10036322", master.Layer[0].Reinsurer[0].ID)
	require.Equal(t, "SWISS RE", master.Layer[0].Reinsurer[0].Name)
}

func TestLabelTypeXOLDiturunkanDariIsiPenyaringnya(t *testing.T) {
	require.Equal(t, "Property / Motor / Engineering", masterxol.TypeLabel(masterxol.TypeProperty))
	require.Equal(t, "PA / GA", masterxol.TypeLabel(masterxol.TypeAccident))
	require.Equal(t, "Marine / Heavy Equipment", masterxol.TypeLabel(masterxol.TypeMarine))
	require.Len(t, masterxol.KnownType(), 3)
}

func panjang(n int) string {
	buffer := make([]byte, n)
	for i := range buffer {
		buffer[i] = 'x'
	}
	return string(buffer)
}
