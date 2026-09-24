package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/repo/memory"
	"claim-pnc/internal/komite/usecase"
	"claim-pnc/internal/platform/clock"
)

// sekarangUji dibekukan supaya Aging dan waktu keputusan dapat diperiksa dengan angka
// pasti, bukan dengan toleransi.
var sekarangUji = time.Date(2026, 9, 20, 2, 0, 0, 0, time.UTC)

func layananInbox(t *testing.T) (*usecase.InboxService, *memory.InboxStore) {
	t.Helper()

	penyimpanan := memory.NewSampleInboxStore()
	layanan, err := usecase.NewInboxService(usecase.InboxOptions{
		Cases:     penyimpanan,
		Decisions: penyimpanan,
		IDs:       memory.IDGenerator{},
		Clock:     clock.FixedAt(sekarangUji),
	})
	require.NoError(t, err)
	return layanan, penyimpanan
}

func penyaring(kotak komite.InboxKind) komite.InboxFilter {
	return komite.InboxFilter{Operator: memory.SampleOperator, Kind: kotak}
}

func nomorKasus(hasil usecase.InboxResult) []string {
	var daftar []string
	for _, c := range hasil.Cases {
		daftar = append(daftar, c.CaseID)
	}
	return daftar
}

func pemeranUji() usecase.Actor {
	return usecase.Actor{Login: memory.SampleOperator, Name: "Ellen Supriyati"}
}

// Inbox adalah daftar pekerjaan MILIK SESEORANG, dan batas itu harus terbukti — bukan
// diandaikan benar karena penyaringnya kebetulan ada.
func TestInboxTidakBocorAntarOperator(t *testing.T) {
	layanan, _ := layananInbox(t)

	hasil, err := layanan.Inbox(context.Background(), penyaring(komite.InboxOutstanding))
	require.NoError(t, err)
	require.NotContains(t, nomorKasus(hasil), "K-2606", "kasus milik operator lain tidak boleh muncul")

	// Dan dari sisi sebaliknya: pemiliknya memang melihatnya.
	milikOrangLain, err := layanan.Inbox(context.Background(), komite.InboxFilter{
		Operator: "INDRAGUNAWAN",
		Kind:     komite.InboxOutstanding,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"K-2606"}, nomorKasus(milikOrangLain))
}

// Operator kosong menghasilkan NOL BARIS, bukan seluruh antrean.
//
// Kegagalan yang aman pada sebuah inbox adalah menampilkan terlalu sedikit. Penyaring
// pemilik yang gagal terisi lalu diam-diam diartikan "semua" akan membocorkan nilai klaim
// dan nama tertanggung seluruh perusahaan kepada siapa pun yang punya sesi.
func TestOperatorKosongTidakMenghasilkanBaris(t *testing.T) {
	layanan, _ := layananInbox(t)

	hasil, err := layanan.Inbox(context.Background(), komite.InboxFilter{Kind: komite.InboxOutstanding})
	require.NoError(t, err)
	require.Empty(t, hasil.Cases)
	require.Zero(t, hasil.Total)
}

// Kotak Outstanding hanya berisi PEKERJAAN — yang sudah tuntas di Pega tidak ikut.
//
// Ditiru apa adanya dari `A.PYSTATUSWORK <> 'Resolved-Completed'` pada
// `GetKomitePAOutstanding`.
func TestKotakOutstandingMengecualikanCaseYangSudahTuntasDiPega(t *testing.T) {
	layanan, _ := layananInbox(t)

	hasil, err := layanan.Inbox(context.Background(), penyaring(komite.InboxOutstanding))
	require.NoError(t, err)
	require.NotContains(t, nomorKasus(hasil), "K-2607")
	require.NotContains(t, nomorKasus(hasil), "K-2604", "yang sudah diputuskan di Pega bukan pekerjaan")
}

// Kasus TANPA penilaian AI tetap muncul.
//
// Kueri lama menyambungkan `T_CLAIM_DATA_RESULTS_AI` dengan OUTER JOIN
// (`A.pyID = AI.KOMITE(+)`). Mengubahnya menjadi INNER akan membuat pekerjaan menghilang
// dari inbox tanpa satu pun tanda — kelas cacat paling mahal yang bisa ada di sini.
func TestKasusTanpaPenilaianAITetapMuncul(t *testing.T) {
	layanan, _ := layananInbox(t)

	hasil, err := layanan.Inbox(context.Background(), penyaring(komite.InboxOutstanding))
	require.NoError(t, err)
	require.Contains(t, nomorKasus(hasil), "K-2603")

	for _, c := range hasil.Cases {
		if c.CaseID == "K-2603" {
			require.False(t, c.HasAIAssessment)
			require.Empty(t, c.AIResult)
		}
	}
}

// Riwayat keputusan yang dibuat di PEGA tetap terbaca selama masa paralel.
//
// Tanpa ini, seorang anggota komite yang sudah bertahun-tahun bekerja akan membuka layar
// baru dan menemukan riwayatnya kosong — yang terbaca sebagai data hilang, bukan sebagai
// sistem baru.
func TestRiwayatWarisanTetapTerbaca(t *testing.T) {
	layanan, _ := layananInbox(t)

	diterima, err := layanan.Inbox(context.Background(), penyaring(komite.InboxAccepted))
	require.NoError(t, err)
	require.Equal(t, []string{"K-2604"}, nomorKasus(diterima))

	ditolak, err := layanan.Inbox(context.Background(), penyaring(komite.InboxRejected))
	require.NoError(t, err)
	require.Equal(t, []string{"K-2605"}, nomorKasus(ditolak))
}

// Yang paling lama menunggu berada di atas, persis `ORDER BY "AgingKomite" DESC`.
func TestYangPalingLamaMenungguBeradaDiAtas(t *testing.T) {
	layanan, _ := layananInbox(t)

	hasil, err := layanan.Inbox(context.Background(), penyaring(komite.InboxOutstanding))
	require.NoError(t, err)
	require.Equal(t, []string{"K-2601", "K-2603", "K-2602"}, nomorKasus(hasil))

	require.Equal(t, 9, hasil.Cases[0].AgingDays(hasil.Now))
	require.Equal(t, 1, hasil.Cases[2].AgingDays(hasil.Now))
}

// Lencana tab ikut menyaring pencarian, tetapi tidak menyaring kotak.
//
// Tanpa itu, pengguna melihat angka pada tab lain, berpindah ke sana, lalu menemukan
// tabel kosong — karena angkanya menjawab pertanyaan yang berbeda dari isinya.
func TestLencanaTabMengikutiPencarianYangSedangBerlaku(t *testing.T) {
	layanan, _ := layananInbox(t)

	f := penyaring(komite.InboxOutstanding)
	f.Search = "0104" // hanya ada di kotak Diterima

	hasil, err := layanan.Inbox(context.Background(), f)
	require.NoError(t, err)
	require.Empty(t, hasil.Cases)
	require.Zero(t, hasil.Summary.Outstanding)
	require.Equal(t, 1, hasil.Summary.Accepted, "lencana menjawab pencarian yang sama")
	require.Zero(t, hasil.Summary.Rejected)
}

// Satu kotak pencarian untuk DUA kolom, persis prompt "No Komite / No Klaim".
func TestPencarianMencocokkanNomorCaseMaupunNomorKlaim(t *testing.T) {
	layanan, _ := layananInbox(t)

	perNomorCase := penyaring(komite.InboxOutstanding)
	perNomorCase.Search = "k-2602"
	hasil, err := layanan.Inbox(context.Background(), perNomorCase)
	require.NoError(t, err)
	require.Equal(t, []string{"K-2602"}, nomorKasus(hasil))

	perNomorKlaim := penyaring(komite.InboxOutstanding)
	perNomorKlaim.Search = "26.0102"
	hasil, err = layanan.Inbox(context.Background(), perNomorKlaim)
	require.NoError(t, err)
	require.Equal(t, []string{"K-2602"}, nomorKasus(hasil))
}

// Rentang tanggal inklusif di KEDUA ujung.
func TestRentangTanggalInklusifDiKeduaUjung(t *testing.T) {
	layanan, _ := layananInbox(t)

	f := penyaring(komite.InboxOutstanding)
	f.DateFrom = clock.AddDays(sekarangUji, -4)
	f.DateTo = clock.AddDays(sekarangUji, -4)

	hasil, err := layanan.Inbox(context.Background(), f)
	require.NoError(t, err)
	require.Equal(t, []string{"K-2603"}, nomorKasus(hasil), "kasus tepat pada tanggal itu ikut")
}

func TestRentangTanggalTerbalikDitolakSebagaiValidasi(t *testing.T) {
	layanan, _ := layananInbox(t)

	f := penyaring(komite.InboxOutstanding)
	f.DateFrom = sekarangUji
	f.DateTo = clock.AddDays(sekarangUji, -10)

	_, err := layanan.Inbox(context.Background(), f)
	var validasi *komite.ValidationError
	require.ErrorAs(t, err, &validasi)
}

// Persetujuan mengeluarkan kasus dari kotak Outstanding milik orang itu, dan
// memindahkannya ke riwayatnya.
func TestPersetujuanMemindahkanKasusKeRiwayat(t *testing.T) {
	layanan, _ := layananInbox(t)
	ctx := context.Background()

	sesudah, err := layanan.Decide(ctx, komite.DecisionCommand{
		CaseID: "K-2601",
		Kind:   komite.DecisionApprove,
	}, pemeranUji())
	require.NoError(t, err)
	require.Len(t, sesudah.Progress.Decisions, 1)

	outstanding, err := layanan.Inbox(ctx, penyaring(komite.InboxOutstanding))
	require.NoError(t, err)
	require.NotContains(t, nomorKasus(outstanding), "K-2601")

	diterima, err := layanan.Inbox(ctx, penyaring(komite.InboxAccepted))
	require.NoError(t, err)
	require.Contains(t, nomorKasus(diterima), "K-2601")
}

func TestPenolakanMemindahkanKasusKeKotakDitolak(t *testing.T) {
	layanan, _ := layananInbox(t)
	ctx := context.Background()

	_, err := layanan.Decide(ctx, komite.DecisionCommand{
		CaseID: "K-2601",
		Kind:   komite.DecisionReject,
		Note:   "Nilai melebihi sisa TSI.",
	}, pemeranUji())
	require.NoError(t, err)

	ditolak, err := layanan.Inbox(ctx, penyaring(komite.InboxRejected))
	require.NoError(t, err)
	require.Contains(t, nomorKasus(ditolak), "K-2601")
}

// Pengembalian diperlakukan sama dengan penolakan pada penempatan kotak: keduanya
// menghentikan komite.
func TestPengembalianJugaMasukKotakDitolak(t *testing.T) {
	layanan, _ := layananInbox(t)
	ctx := context.Background()

	_, err := layanan.Decide(ctx, komite.DecisionCommand{
		CaseID: "K-2602",
		Kind:   komite.DecisionReturn,
		Note:   "Dokumen survei belum dilampirkan.",
	}, pemeranUji())
	require.NoError(t, err)

	ditolak, err := layanan.Inbox(ctx, penyaring(komite.InboxRejected))
	require.NoError(t, err)
	require.Contains(t, nomorKasus(ditolak), "K-2602")
}

// Jenjang ditetapkan SERVER, tidak pernah dikirim klien.
//
// Klien yang boleh menyebut jenjangnya sendiri dapat menyetujui jenjang yang bukan
// gilirannya — dan jenjang menentukan siapa berwenang menyetujui uang.
func TestJenjangDitetapkanServerBukanKlien(t *testing.T) {
	layanan, _ := layananInbox(t)

	sesudah, err := layanan.Decide(context.Background(), komite.DecisionCommand{
		CaseID: "K-2601",
		Kind:   komite.DecisionApprove,
	}, pemeranUji())
	require.NoError(t, err)

	require.Len(t, sesudah.Progress.Decisions, 1)
	require.Equal(t, 1, sesudah.Progress.Decisions[0].Tier)
	require.Equal(t, sekarangUji, sesudah.Progress.Decisions[0].DecidedAt)
	require.Equal(t, memory.SampleOperator, sesudah.Progress.Decisions[0].ActorLogin)
	require.NotEmpty(t, sesudah.Progress.Decisions[0].ID)
}

// Satu orang memutuskan SEKALI pada satu kasus.
//
// Ini menjaga keadaan yang benar-benar terjadi: dua tab terbuka bersamaan, keduanya
// ditekan. Tanpa pemeriksaan ini, keputusan kedua tercatat sebagai jenjang yang sama dua
// kali dan penjenjangan berhenti dapat dipercaya.
func TestSatuOrangMemutuskanSekaliSaja(t *testing.T) {
	layanan, _ := layananInbox(t)
	ctx := context.Background()
	perintah := komite.DecisionCommand{CaseID: "K-2601", Kind: komite.DecisionApprove}

	_, err := layanan.Decide(ctx, perintah, pemeranUji())
	require.NoError(t, err)

	_, err = layanan.Decide(ctx, perintah, pemeranUji())
	require.ErrorIs(t, err, komite.ErrDecisionClosed)
}

// Keputusan atas kasus milik orang lain DITOLAK.
//
// Penyaring di kueri membatasi apa yang TERLIHAT; pemeriksaan ini membatasi apa yang
// dapat DILAKUKAN. Nomor case yang terlihat di satu layar dapat dikirim ke endpoint mana
// pun oleh siapa pun yang punya sesi.
func TestKeputusanAtasKasusOrangLainDitolak(t *testing.T) {
	layanan, _ := layananInbox(t)

	_, err := layanan.Decide(context.Background(), komite.DecisionCommand{
		CaseID: "K-2606",
		Kind:   komite.DecisionApprove,
	}, pemeranUji())
	require.ErrorIs(t, err, komite.ErrNotAssigned)
}

func TestKeputusanAtasKasusYangTidakAdaDitolak(t *testing.T) {
	layanan, _ := layananInbox(t)

	_, err := layanan.Decide(context.Background(), komite.DecisionCommand{
		CaseID: "K-9999",
		Kind:   komite.DecisionApprove,
	}, pemeranUji())
	require.ErrorIs(t, err, komite.ErrCaseNotFound)
}

// Perintah yang cacat ditolak SEBELUM kasusnya dibaca, dan seluruh pelanggarannya
// dikembalikan sekaligus.
func TestPerintahCacatDitolakSebagaiValidasi(t *testing.T) {
	layanan, _ := layananInbox(t)

	_, err := layanan.Decide(context.Background(), komite.DecisionCommand{
		CaseID: "K-2601",
		Kind:   komite.DecisionReject,
	}, pemeranUji())

	var validasi *komite.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violations, 1)
	require.Equal(t, komite.FieldNote, validasi.Violations[0].Field)
}

// Jumlah jenjang yang belum diketahui DITAMPILKAN apa adanya, tidak disamarkan menjadi
// angka.
//
// Keadaannya nyata hari ini: lini bisnis sebuah kasus tidak ada di sumber mana pun yang
// dibaca modul ini, sehingga jumlah jenjangnya tidak dapat dihitung. Menyamarkannya
// menjadi "1 dari 1" akan membuat persetujuan pertama tampak menutup seluruh komite.
func TestJumlahJenjangYangBelumDiketahuiTidakDisamarkan(t *testing.T) {
	layanan, _ := layananInbox(t)

	sesudah, err := layanan.Decide(context.Background(), komite.DecisionCommand{
		CaseID: "K-2601",
		Kind:   komite.DecisionApprove,
	}, pemeranUji())
	require.NoError(t, err)

	require.True(t, sesudah.Progress.TierCountUnknown())
	require.False(t, sesudah.Progress.Closed(), "komite tidak dinyatakan selesai tanpa tahu jumlah jenjangnya")
	require.Equal(t, komite.OutcomePending, sesudah.Progress.Outcome)
}

// Halaman dipotong, TETAPI Total melaporkan keseluruhan.
//
// Inilah yang membedakannya dari `pyMaxRecords=500` sistem lama, yang memotong tanpa
// pernah menyebut ada yang terpotong (`T-12`).
func TestHalamanDipotongTetapiTotalTetapUtuh(t *testing.T) {
	layanan, _ := layananInbox(t)

	f := penyaring(komite.InboxOutstanding)
	f.Limit = 2

	hasil, err := layanan.Inbox(context.Background(), f)
	require.NoError(t, err)
	require.Len(t, hasil.Cases, 2)
	require.Equal(t, 3, hasil.Total)

	f.Offset = 2
	halamanDua, err := layanan.Inbox(context.Background(), f)
	require.NoError(t, err)
	require.Len(t, halamanDua.Cases, 1)
	require.Equal(t, 3, halamanDua.Total)
}

// Galat penyimpanan diteruskan, tidak ditelan menjadi inbox kosong.
//
// Inbox yang kosong karena basis datanya bermasalah terbaca persis seperti inbox yang
// memang tidak punya pekerjaan — dan yang pertama menuntut tindakan sementara yang kedua
// tidak.
func TestGalatPenyimpananDiteruskan(t *testing.T) {
	layanan, penyimpanan := layananInbox(t)
	gagal := errors.New("koneksi terputus")
	penyimpanan.SetError(gagal)

	_, err := layanan.Inbox(context.Background(), penyaring(komite.InboxOutstanding))
	require.ErrorIs(t, err, gagal)
}

func TestBahanYangTidakLengkapDitolakSaatStart(t *testing.T) {
	penyimpanan := memory.NewSampleInboxStore()

	_, err := usecase.NewInboxService(usecase.InboxOptions{Decisions: penyimpanan, IDs: memory.IDGenerator{}})
	require.Error(t, err)

	_, err = usecase.NewInboxService(usecase.InboxOptions{Cases: penyimpanan, IDs: memory.IDGenerator{}})
	require.Error(t, err)

	_, err = usecase.NewInboxService(usecase.InboxOptions{Cases: penyimpanan, Decisions: penyimpanan})
	require.Error(t, err)
}
