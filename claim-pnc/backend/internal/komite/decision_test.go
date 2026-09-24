package komite_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
)

var waktuKeputusan = time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)

// keputusan membentuk satu baris keputusan dengan bahan seperlunya.
func keputusan(jenjang int, jenis komite.DecisionKind, selang time.Duration) komite.Decision {
	return komite.Decision{
		ID:          "K" + strings.Repeat("0", 3),
		CaseID:      "K-1",
		ClaimNumber: "PNCN.26.1",
		Tier:        jenjang,
		Kind:        jenis,
		Note:        "catatan",
		ActorLogin:  "ELLENSUPRIYATI",
		DecidedAt:   waktuKeputusan.Add(selang),
	}
}

// Inilah aturan inti penjenjangan: setiap persetujuan memajukan SATU jenjang, dan komite
// baru selesai setelah seluruh jenjang menyetujui.
//
// Angka jenjangnya datang dari jumlah baris master yang cocok (`KomiteLoop`), bukan dari
// `DEGREE` — perbedaan yang `D-52` tegaskan dan yang paling mudah tertukar.
func TestPersetujuanMemajukanSatuJenjangSampaiHabis(t *testing.T) {
	kosong := komite.Evaluate(nil, 3)
	require.Equal(t, komite.OutcomePending, kosong.Outcome)
	require.Equal(t, 1, kosong.CurrentTier, "belum ada keputusan berarti jenjang pertama yang menunggu")

	satu := komite.Evaluate([]komite.Decision{
		keputusan(1, komite.DecisionApprove, 0),
	}, 3)
	require.Equal(t, komite.OutcomePending, satu.Outcome)
	require.Equal(t, 2, satu.CurrentTier)
	require.Equal(t, 1, satu.ApprovedTiers)

	semua := komite.Evaluate([]komite.Decision{
		keputusan(1, komite.DecisionApprove, 0),
		keputusan(2, komite.DecisionApprove, time.Hour),
		keputusan(3, komite.DecisionApprove, 2*time.Hour),
	}, 3)
	require.Equal(t, komite.OutcomeApproved, semua.Outcome)
	require.Zero(t, semua.CurrentTier, "tidak ada jenjang yang menunggu setelah seluruhnya menyetujui")
	require.True(t, semua.Closed())
}

// Penolakan satu jenjang membatalkan SELURUH sisa jenjang — bukan hanya jenjang itu.
//
// Diambil apa adanya dari `Activity/KomitePost_Adjustment-Act.xml` step 65, yang saat
// `AcceptStatus = 2` memaksa `KomiteCount := KomiteLoop` sehingga perulangan
// `IsKomiteLoop` berhenti.
func TestPenolakanSatuJenjangMenghentikanSeluruhKomite(t *testing.T) {
	hasil := komite.Evaluate([]komite.Decision{
		keputusan(1, komite.DecisionApprove, 0),
		keputusan(2, komite.DecisionReject, time.Hour),
	}, 4)

	require.Equal(t, komite.OutcomeRejected, hasil.Outcome)
	require.Zero(t, hasil.CurrentTier)
	require.True(t, hasil.Closed())
}

func TestPengembalianJugaMenghentikanKomite(t *testing.T) {
	hasil := komite.Evaluate([]komite.Decision{
		keputusan(1, komite.DecisionReturn, 0),
	}, 4)

	require.Equal(t, komite.OutcomeReturned, hasil.Outcome)
	require.Zero(t, hasil.CurrentTier)
	require.True(t, hasil.Closed())
}

// Jenjang yang belum diketahui TIDAK boleh menghasilkan "selesai".
//
// Keadaannya nyata hari ini: nilai klaim datang dari `B-5` yang belum ada, sehingga
// jumlah jenjang sebuah kasus sering belum dapat dihitung. Bila ketidaktahuan itu
// diperlakukan sebagai nol, persetujuan PERTAMA akan langsung menutup komite — dan
// akseptasi terbuka tanpa dasar, persis yang invarian `I-5` larang.
func TestJenjangYangBelumDiketahuiTidakPernahDinyatakanSelesai(t *testing.T) {
	hasil := komite.Evaluate([]komite.Decision{
		keputusan(1, komite.DecisionApprove, 0),
		keputusan(2, komite.DecisionApprove, time.Hour),
	}, 0)

	require.True(t, hasil.TierCountUnknown())
	require.Equal(t, komite.OutcomePending, hasil.Outcome)
	require.False(t, hasil.Closed())
	require.Equal(t, 3, hasil.CurrentTier, "jenjang berikutnya tetap dihitung walau totalnya belum diketahui")
}

// Urutan masukan tidak boleh mengubah kesimpulan.
//
// Basis data tidak menjamin urutan tanpa ORDER BY, dan penolakan yang kebetulan terbaca
// lebih dulu tidak boleh membuat kasus yang sebenarnya disetujui tampak ditolak.
func TestKesimpulanTidakBergantungUrutanMasukan(t *testing.T) {
	urut := []komite.Decision{
		keputusan(1, komite.DecisionApprove, 0),
		keputusan(2, komite.DecisionApprove, time.Hour),
	}
	terbalik := []komite.Decision{urut[1], urut[0]}

	require.Equal(t, komite.Evaluate(urut, 2).Outcome, komite.Evaluate(terbalik, 2).Outcome)
	require.Equal(t, komite.OutcomeApproved, komite.Evaluate(terbalik, 2).Outcome)
}

// Catatan WAJIB pada tolak dan kembalikan, dan itu PENAMBAHAN terhadap sistem lama.
//
// Sistem lama tidak mewajibkan `NOTEKOMITE` pada keadaan mana pun. Penolakan tanpa alasan
// tidak dapat ditindaklanjuti siapa pun — pengaju tidak tahu apa yang harus diperbaiki.
func TestCatatanWajibPadaTolakDanKembalikanSaja(t *testing.T) {
	tolak := komite.DecisionCommand{CaseID: "K-1", Kind: komite.DecisionReject}.Normalize()
	require.Error(t, tolak.Validate())

	kembalikan := komite.DecisionCommand{CaseID: "K-1", Kind: komite.DecisionReturn}.Normalize()
	require.Error(t, kembalikan.Validate())

	setuju := komite.DecisionCommand{CaseID: "K-1", Kind: komite.DecisionApprove}.Normalize()
	require.NoError(t, setuju.Validate(), "persetujuan tidak menuntut catatan")
}

func TestKeputusanYangTidakDikenalDitolak(t *testing.T) {
	perintah := komite.DecisionCommand{CaseID: "K-1", Kind: "ditunda"}.Normalize()

	err := perintah.Validate()
	require.Error(t, err)

	var validasi *komite.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violations, 1)
	require.Equal(t, komite.FieldDecision, validasi.Violations[0].Field)
}

// Seluruh pelanggaran dikembalikan sekaligus, bukan satu per satu.
func TestValidasiMengumpulkanSeluruhPelanggaran(t *testing.T) {
	perintah := komite.DecisionCommand{
		CaseID: "K-1",
		Kind:   "entahlah",
		Note:   strings.Repeat("a", komite.MaxNoteLength+1),
	}.Normalize()

	err := perintah.Validate()
	var validasi *komite.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violations, 2)
}

// Panjang catatan dihitung per RUNE, bukan per byte.
//
// Catatan komite berbahasa Indonesia dan sering memuat huruf di luar ASCII. Menghitungnya
// per byte akan menolak kalimat yang panjangnya wajar hanya karena hurufnya beraksen.
func TestPanjangCatatanDihitungPerHuruf(t *testing.T) {
	// Setiap "é" memakan dua byte tetapi satu rune.
	pasKeBatas := komite.DecisionCommand{
		CaseID: "K-1",
		Kind:   komite.DecisionApprove,
		Note:   strings.Repeat("é", komite.MaxNoteLength),
	}.Normalize()
	require.NoError(t, pasKeBatas.Validate())

	lewatBatas := komite.DecisionCommand{
		CaseID: "K-1",
		Kind:   komite.DecisionApprove,
		Note:   strings.Repeat("é", komite.MaxNoteLength+1),
	}.Normalize()
	require.Error(t, lewatBatas.Validate())
}

// Penulisan keputusan dirapikan sebelum dibandingkan, sehingga "  SETUJU " tetap dikenali.
func TestPenulisanKeputusanDirapikan(t *testing.T) {
	perintah := komite.DecisionCommand{CaseID: " K-1 ", Kind: "  SETUJU "}.Normalize()

	require.Equal(t, komite.DecisionApprove, perintah.Kind)
	require.Equal(t, "K-1", perintah.CaseID)
	require.NoError(t, perintah.Validate())
}
