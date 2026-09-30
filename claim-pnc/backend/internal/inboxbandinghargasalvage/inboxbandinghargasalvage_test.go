package inboxbandinghargasalvage_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Nama uji menyebutkan ATURANNYA, bukan nama fungsinya, sehingga daftar uji terbaca sebagai
// daftar aturan yang berlaku (`14-TESTING-STRATEGY.md` §3.2).

func TestKeduaTabTerdaftarDenganKodeDanParameterPegaNya(t *testing.T) {
	tabs := inboxbandinghargasalvage.Tabs()
	require.Len(t, tabs, 2, "layar ini punya DUA grid — lihat Section/InboxReqSalvageASM")

	require.Equal(t, inboxbandinghargasalvage.TabRequest, tabs[0].Code)
	require.Equal(t, "1", tabs[0].PegaParam, "grid Request dipanggil dengan tipe=1")
	require.False(t, tabs[0].Decided)

	require.Equal(t, inboxbandinghargasalvage.TabHistory, tabs[1].Code)
	require.Equal(t, "2", tabs[1].PegaParam, "grid History dipanggil dengan tipe=2")
	require.True(t, tabs[1].Decided)
}

// Judul tab disalin harfiah dari `Local.LOOP` pada activity pencacahnya — termasuk salah
// ketiknya. Uji ini ada supaya "Cheker" tidak diperbaiki menjadi "Checker" tanpa sadar; itu
// akan membuat judul di layar berbeda dari yang dibaca pengguna hari ini (`D-13`).
func TestSalahKetikCheckerDipertahankanSepertiDiLayarLama(t *testing.T) {
	tab, known := inboxbandinghargasalvage.FindTab(inboxbandinghargasalvage.TabHistory)
	require.True(t, known)
	require.Equal(t, "History Cheker", tab.Name)
}

func TestTabBawaanAdalahRequestBandingHarga(t *testing.T) {
	require.Equal(t, inboxbandinghargasalvage.TabRequest, inboxbandinghargasalvage.DefaultTab)
}

// Kesembilan kolom grid Request, berurutan persis seperti sel gridnya di Pega.
func TestKolomGridRequestSamaDenganSelGridPega(t *testing.T) {
	tab, known := inboxbandinghargasalvage.FindTab(inboxbandinghargasalvage.TabRequest)
	require.True(t, known)

	titles := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		titles = append(titles, column.Title)
	}

	require.Equal(t, []string{
		"Tanggal Request", "No Klaim", "Detail Object", "Nama Barang",
		"Harga Barang", "Harga Request", "Note Request", "Aging", "Note Checker",
	}, titles)
}

func TestKolomGridHistorySamaDenganSelGridPega(t *testing.T) {
	tab, known := inboxbandinghargasalvage.FindTab(inboxbandinghargasalvage.TabHistory)
	require.True(t, known)

	titles := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		titles = append(titles, column.Title)
	}

	require.Equal(t, []string{"No Klaim", "Object Name", "Lokasi Salvage", "PIC"}, titles)
}

// Hanya kedua kolom harga yang ditandai angka. Aging TIDAK, meski isinya bermula dari
// bilangan: yang dikirim ke layar adalah teks `<n> days`, dan memformatnya sebagai bilangan
// akan menghasilkan "12 days" yang gagal diurai lalu digambar apa adanya — kebetulan benar,
// tetapi karena alasan yang salah.
func TestHanyaKolomHargaYangDitandaiAngka(t *testing.T) {
	tab, known := inboxbandinghargasalvage.FindTab(inboxbandinghargasalvage.TabRequest)
	require.True(t, known)

	numeric := map[string]bool{}
	for _, column := range tab.Columns {
		numeric[column.Key] = column.Numeric
	}

	require.True(t, numeric[inboxbandinghargasalvage.FieldItemPrice])
	require.True(t, numeric[inboxbandinghargasalvage.FieldRequestPrice])
	require.False(t, numeric[inboxbandinghargasalvage.FieldAging])
	require.False(t, numeric[inboxbandinghargasalvage.FieldClaimNo])
}

// Tabs() menyerahkan salinan. Tanpa penyalinan dalam, satu pemanggil yang menulisi hasilnya
// akan mengubah judul kolom bagi SELURUH permintaan berikutnya — `tabs` hidup selama aplikasi
// berjalan.
func TestDaftarKolomTidakDapatDiubahLewatHasilTabs(t *testing.T) {
	first := inboxbandinghargasalvage.Tabs()
	first[0].Columns[0].Title = "DIUBAH"

	second := inboxbandinghargasalvage.Tabs()
	require.Equal(t, "Tanggal Request", second[0].Columns[0].Title)
}

func TestPermintaanTanpaIdentitasPemanggilDitolak(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewQuery(
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Caller{Login: "  "})

	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)
}

func TestTabKosongJatuhKeTabBawaan(t *testing.T) {
	query, err := inboxbandinghargasalvage.NewQuery(
		inboxbandinghargasalvage.QueryInput{Tab: ""},
		inboxbandinghargasalvage.Caller{Login: "KOMITESATU"})

	require.NoError(t, err)
	require.Equal(t, inboxbandinghargasalvage.DefaultTab, query.Tab.Code)
}

func TestTabTidakDikenalMenghasilkanGalatValidasi(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewQuery(
		inboxbandinghargasalvage.QueryInput{Tab: "entah-apa"},
		inboxbandinghargasalvage.Caller{Login: "KOMITESATU"})

	var validation *inboxbandinghargasalvage.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxbandinghargasalvage.FieldTab, validation.Violations[0].Field)
}

// Kata kunci diseragamkan huruf besar dan spasinya dipangkas — lihat NewQuery.
func TestKataKunciDiseragamkanHurufBesarDanDipangkas(t *testing.T) {
	query, err := inboxbandinghargasalvage.NewQuery(
		inboxbandinghargasalvage.QueryInput{Keyword: "  pncn.26.0451  "},
		inboxbandinghargasalvage.Caller{Login: "KOMITESATU"})

	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0451", query.Keyword)
	require.True(t, query.Searching())
}

func TestKataKunciKosongBerartiTidakSedangMencari(t *testing.T) {
	query, err := inboxbandinghargasalvage.NewQuery(
		inboxbandinghargasalvage.QueryInput{Keyword: "   "},
		inboxbandinghargasalvage.Caller{Login: "KOMITESATU"})

	require.NoError(t, err)
	require.False(t, query.Searching())
}

// ============================================================================
// Aturan bernama orang — ditiru dari Pega atas keputusan Work Owner 2026-09-29
// ============================================================================
//
// Ketiga uji berikut menjaga agar aturan itu tidak hilang diam-diam, DAN agar ia tidak
// meluas ke orang lain. Keduanya sama pentingnya: yang pertama menjaga kesetaraan dengan
// Pega, yang kedua menjaga agar kebocoran antrean tidak bertambah.

func TestPemanggilBiasaMelihatAntreannyaSendiri(t *testing.T) {
	reviewer := inboxbandinghargasalvage.ReviewerFor(
		inboxbandinghargasalvage.Caller{Login: "KOMITESATU"})

	require.Equal(t, "KOMITESATU", reviewer.Name)
	require.False(t, reviewer.Delegated)
	require.Empty(t, reviewer.WaitFor)
	require.Empty(t, reviewer.DelegationNotice())
	require.Empty(t, reviewer.QueueNotice())
}

func TestSatuOperatorMelihatAntreanKomiteLain(t *testing.T) {
	reviewer := inboxbandinghargasalvage.ReviewerFor(
		inboxbandinghargasalvage.Caller{Login: "MARIATRIELSA"})

	require.Equal(t, "BAMBANGSETIADJIGUNAWAN", reviewer.Name)
	require.True(t, reviewer.Delegated)
	require.Contains(t, reviewer.DelegationNotice(), "BAMBANGSETIADJIGUNAWAN")
}

// `Activity/GCNMCountRequestSalvage_act-Act.xml` langkah 5 memasukkan WULANINDRIPAAT ke dalam
// perwakilan yang sama, sementara `SetReqSalvage_Act` langkah 8 TIDAK. Yang berlaku adalah
// versi daftarnya, karena Work Owner memutuskan pencacah disamakan dengan daftar.
//
// Uji ini menjaga keputusan itu: bila seseorang kelak "melengkapi" perwakilan dari activity
// pencacah, angka ringkas dan jumlah baris grid akan berselisih lagi bagi satu orang.
func TestOperatorKeduaDiPencacahTIDAKIkutMewakili(t *testing.T) {
	reviewer := inboxbandinghargasalvage.ReviewerFor(
		inboxbandinghargasalvage.Caller{Login: "WULANINDRIPAAT"})

	require.Equal(t, "WULANINDRIPAAT", reviewer.Name)
	require.False(t, reviewer.Delegated)
}

func TestSatuOperatorMenungguGiliranKomiteSebelumnya(t *testing.T) {
	reviewer := inboxbandinghargasalvage.ReviewerFor(
		inboxbandinghargasalvage.Caller{Login: "DANIELLISWANDI"})

	require.Equal(t, "DANIELLISWANDI", reviewer.Name)
	require.Equal(t, "BAMBANGSETIADJIGUNAWAN", reviewer.WaitFor)
	require.False(t, reviewer.Delegated, "ia melihat antreannya SENDIRI, hanya tertunda")
	require.Contains(t, reviewer.QueueNotice(), "BAMBANGSETIADJIGUNAWAN")
}

// Di Pega nama dibandingkan apa adanya, sehingga login berhuruf kecil tidak pernah cocok dan
// orangnya diam-diam melihat antrean yang salah. Itu bukan aturan bisnis; lihat ReviewerFor.
func TestPerbandinganNamaTidakPekaHurufBesarKecil(t *testing.T) {
	reviewer := inboxbandinghargasalvage.ReviewerFor(
		inboxbandinghargasalvage.Caller{Login: " mariatrielsa "})

	require.Equal(t, "BAMBANGSETIADJIGUNAWAN", reviewer.Name)
	require.True(t, reviewer.Delegated)
}

// ============================================================================
// Paginasi
// ============================================================================

func TestUkuranHalamanBawaanMengikutiPageSizePega(t *testing.T) {
	clean := inboxbandinghargasalvage.Pagination{}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, inboxbandinghargasalvage.DefaultPageSize, clean.Size)
}

func TestPaginasiDiLuarRentangDIBETULKAN(t *testing.T) {
	clean := inboxbandinghargasalvage.Pagination{Page: -3, Size: 5000}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, inboxbandinghargasalvage.MaxPageSize, clean.Size)
}

func TestHalamanKosongTetapDihitungSatuHalaman(t *testing.T) {
	page := inboxbandinghargasalvage.Page{
		Pagination: inboxbandinghargasalvage.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 1, page.TotalPages(), "layar tidak boleh menggambar 'halaman 1 dari 0'")
}

func TestSisaBarisDihitungSebagaiSatuHalamanTambahan(t *testing.T) {
	page := inboxbandinghargasalvage.Page{
		Total:      26,
		Pagination: inboxbandinghargasalvage.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 2, page.TotalPages())
}

// ============================================================================
// Selisih terencana dan keterbatasan
// ============================================================================

// Keempat selisih yang disetujui Work Owner harus BENAR-BENAR dinyatakan ke pengguna. Tanpa
// itu, uji kesetaraan gerbang 1 akan melaporkannya sebagai bug (`D-54`).
func TestSelisihTerencanaMenyebutKeempatnya(t *testing.T) {
	differences := inboxbandinghargasalvage.PlannedDifferences()
	require.NotEmpty(t, differences)

	joined := ""
	for _, line := range differences {
		joined += line + "\n"
	}

	require.Contains(t, joined, "Aging")
	require.Contains(t, joined, "Note Checker")
	require.Contains(t, joined, "ORDER BY")
	require.Contains(t, joined, "ringkas")

	// Kedua selisih jalur TULIS. Keduanya wajib dinyatakan lebih dulu, karena uji kesetaraan
	// akan menemukannya: yang pertama mengubah keadaan akhir saat gagal, yang kedua mengubah
	// isi tabel dokumen.
	require.Contains(t, joined, "SATU transaksi")
	require.Contains(t, joined, "SALAVAGEDOCUMENT.NOKLAIM")
}

// Keterbatasan harus menyebut artefak yang BENAR-BENAR menahan, bukan artefak yang sudah
// diterima.
//
// Ini bukan kerapian. Artefaknya datang dalam LIMA putaran, dan setiap kali satu berkas
// tiba, nama yang menahan BERGESER ke rule di baliknya — `ButtonApproveRejectedRequest`
// menjadi `ApprovalCheckerSalvage`, lalu menjadi keempat rule yang dipanggilnya. Keterangan
// yang masih menyebut berkas yang sudah diterima akan membuat permintaan berikutnya salah
// sasaran, dan itu memboroskan satu putaran penuh.
func TestKeterbatasanMenyebutArtefakYangBenarBenarMenahan(t *testing.T) {
	joined := ""
	for _, line := range inboxbandinghargasalvage.Limitations() {
		joined += line + "\n"
	}

	// Tidak ada lagi artefak Pega yang menahan layar ini: seluruh rule-nya sudah diterima,
	// dan yang terakhir kurang (`GetAttachmentReqSalvage`) tidak perlu ditebak karena
	// kueri yang sama sudah terbukti berjalan di modul inboxpladla.
	//
	// Yang tersisa karena itu bukan permintaan artefak melainkan keterangan tentang
	// PERILAKU — dan kedua butir di bawah adalah yang paling mudah disalahpahami sebagai
	// kerusakan bila tidak dinyatakan.
	require.Contains(t, joined, "IDBALAILELANG IS NULL")
	require.Contains(t, joined, "BandingHarga")

	// Berkas yang SUDAH diterima tidak boleh lagi disebut sebagai satu-satunya penahan.
	// Keterangan yang menyebutnya begitu akan membuat permintaan berikutnya salah sasaran —
	// dan itu memboroskan satu putaran penuh, yang sudah terjadi tiga kali di layar ini.
	for _, sudahAda := range []string{
		"ButtonApproveRejectedRequest",
		"ShowDtlHistoryReqSalvage_Act",
		"ApprovalCheckerSalvage",
		"UpdateDataReqSalvage",
		"UpdateHargaSalvage",
		"DetailHistReqSalvage_SQL",
	} {
		require.NotContains(t, joined, sudahAda,
			"%q sudah diterima; ia tidak lagi menahan apa pun", sudahAda)
	}

	// Tidak satu pun berkas Pega boleh lagi disebut sebagai penahan, termasuk yang tiba
	// paling akhir.
	for _, sudahAda := range []string{"DokumenBandingSalvage", "LihatDokRequestSalvage"} {
		require.NotContains(t, joined, sudahAda,
			"%q sudah diterima; ia tidak lagi menahan apa pun", sudahAda)
	}

	// Tombol Approve dan Reject kini SUDAH dibangun, sehingga keterangan yang menyebutnya
	// "belum dibangun" justru menyesatkan.
	require.NotContains(t, joined, "baru MEMBACA")

	// Yang tersisa bukan lagi tombolnya, melainkan AKIBAT dari membangunnya tanpa pemanggilan
	// REST: putusannya tersimpan tetapi balai lelang tidak mengetahuinya. Itu keterbatasan
	// yang harus terbaca pengguna, bukan hanya tercatat di dokumen.
	require.Contains(t, joined, "BELUM DIKIRIM")
	require.Contains(t, joined, "balai lelang")
}

// Arti kedua kode status tidak lagi dijaga lewat kalimat keterangan.
//
// Ia sempat dijaga begitu, ketika satu-satunya tempat arti itu tertulis adalah Limitations.
// Sejak panel rincian dibangun, artinya hidup di DecisionLabel — kode yang benar-benar
// dijalankan — dan diuji di TestKodeKeputusanDiterjemahkanSepertiKueriAslinya beserta
// bentuk pecahannya. Menahan satu kalimat tertentu di teks yang dibaca pengguna hanya
// membuat penyuntingan kalimat itu gagal tanpa ada yang rusak.

// ============================================================================
// Panel rincian History Cheker
// ============================================================================

func TestKolomPanelRincianSamaDenganSelGridPega(t *testing.T) {
	titles := make([]string, 0, 7)
	for _, column := range inboxbandinghargasalvage.DecisionColumns() {
		titles = append(titles, column.Title)
	}

	require.Equal(t, []string{
		"Tgl Approve", "Detail Object", "Nama Barang", "Harga Barang",
		"Harga Request", "Jawaban Checker", "Nama Checker",
	}, titles)
}

// Ketiga teksnya disalin harfiah dari `CASE WHEN` kuerinya — termasuk "PROSES" yang
// berhuruf kapital seluruhnya.
func TestKodeKeputusanDiterjemahkanSepertiKueriAslinya(t *testing.T) {
	require.Equal(t, "Setuju", inboxbandinghargasalvage.DecisionLabel("1"))
	require.Equal(t, "Tidak setuju", inboxbandinghargasalvage.DecisionLabel("0"))
	require.Equal(t, "PROSES", inboxbandinghargasalvage.DecisionLabel(""))
	require.Equal(t, "PROSES", inboxbandinghargasalvage.DecisionLabel("9"))
}

// `STATUSAPPROVE` dibandingkan sebagai ANGKA di kueri aslinya, sehingga kolomnya kemungkinan
// NUMBER — dan driver dapat menyerahkannya sebagai `"1.0"`. Tanpa perapian, SETIAP keputusan
// akan terbaca "PROSES": yang sudah diputus tampil seolah belum, tanpa satu pun galat.
func TestKodeKeputusanBentukPecahanTetapTerbaca(t *testing.T) {
	require.Equal(t, "Setuju", inboxbandinghargasalvage.DecisionLabel("1.0"))
	require.Equal(t, "Setuju", inboxbandinghargasalvage.DecisionLabel(" 1 "))
	require.Equal(t, "Tidak setuju", inboxbandinghargasalvage.DecisionLabel("0.00"))

	// Yang BUKAN nol di belakang koma tidak boleh dipangkas — `1.5` bukan `1`.
	require.Equal(t, "PROSES", inboxbandinghargasalvage.DecisionLabel("1.5"))
}

func TestPermintaanPanelRincianTanpaNomorKlaimDitolak(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDecisionQuery(
		"   ", inboxbandinghargasalvage.Caller{Login: "KOMITESATU"})

	var validation *inboxbandinghargasalvage.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, inboxbandinghargasalvage.FieldClaimNo, validation.Violations[0].Field)
}

func TestPermintaanPanelRincianTanpaIdentitasDitolak(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDecisionQuery(
		"PNCN.26.0440", inboxbandinghargasalvage.Caller{Login: ""})

	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)
}

// Panel rincian menyaring menurut komite pemanggil, sama seperti kedua grid — dan aturan
// bernama orang berlaku di sini pula.
func TestPanelRincianMemakaiAntreanYangSamaDenganGridnya(t *testing.T) {
	query, err := inboxbandinghargasalvage.NewDecisionQuery(
		" pncn.26.0440 ", inboxbandinghargasalvage.Caller{Login: "MARIATRIELSA"})

	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0440", query.ClaimNo)
	require.Equal(t, "BAMBANGSETIADJIGUNAWAN", query.Reviewer.Name)
	require.True(t, query.Reviewer.Delegated)
}

// ============================================================================
// Tombol Approve dan Reject
// ============================================================================

func keputusan(
	t *testing.T, login string, setujui bool,
) inboxbandinghargasalvage.DecisionCommand {
	t.Helper()

	command, err := inboxbandinghargasalvage.NewDecisionCommand(
		inboxbandinghargasalvage.DecisionInput{
			DetailObject: "DTL-1",
			SalvageID:    "SLV-1",
			RequestPrice: "9750000",
			Note:         "sudah dibandingkan",
			Approve:      setujui,
		},
		inboxbandinghargasalvage.Caller{Login: login},
	)
	require.NoError(t, err)
	return command
}

func TestMenyetujuiDanMenolakMenghasilkanKodeStatusYangBenar(t *testing.T) {
	require.Equal(t, inboxbandinghargasalvage.DecisionApproved,
		keputusan(t, "KOMITESAYA", true).Status)
	require.Equal(t, inboxbandinghargasalvage.DecisionRejected,
		keputusan(t, "KOMITESAYA", false).Status)
}

// Catatan TIDAK diwajibkan — di Pega ia isian biasa tanpa penanda wajib. Uji ini menjaga
// agar kewajiban itu tidak diselipkan lewat validasi tanpa keputusan siapa pun.
func TestCatatanKosongTetapDapatDiputuskan(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDecisionCommand(
		inboxbandinghargasalvage.DecisionInput{
			DetailObject: "DTL-1", SalvageID: "SLV-1", RequestPrice: "100", Approve: true,
		},
		inboxbandinghargasalvage.Caller{Login: "KOMITESAYA"},
	)
	require.NoError(t, err)
}

// Harga hanya dituntut saat MENYETUJUI — menolak tidak menyentuh harga sama sekali.
func TestHargaKosongHanyaMenghalangiPersetujuan(t *testing.T) {
	kosong := inboxbandinghargasalvage.DecisionInput{
		DetailObject: "DTL-1", SalvageID: "SLV-1", RequestPrice: "",
	}
	komite := inboxbandinghargasalvage.Caller{Login: "KOMITESAYA"}

	kosong.Approve = true
	_, err := inboxbandinghargasalvage.NewDecisionCommand(kosong, komite)
	require.Error(t, err)

	kosong.Approve = false
	_, err = inboxbandinghargasalvage.NewDecisionCommand(kosong, komite)
	require.NoError(t, err, "menolak tidak menyentuh harga")
}

func TestKeputusanTanpaBarangDitolakDenganSeluruhPelanggarannya(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDecisionCommand(
		inboxbandinghargasalvage.DecisionInput{Approve: true},
		inboxbandinghargasalvage.Caller{Login: "KOMITESAYA"},
	)

	var validation *inboxbandinghargasalvage.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 3,
		"seluruh pelanggaran dikumpulkan sekaligus, bukan yang pertama saja")
}

// ============================================================================
// Rencana langkah — aturan bernama orang
// ============================================================================

// Bagi komite biasa, menyetujui HANYA mencatat putusan. Penerapan harga di Pega menuntut
// `AgentID == "DANIELLISWANDI"`, dan bagi pengguna lain syarat itu tidak pernah benar.
//
// Ini perilaku layar lama apa adanya, dan uji ini menjaganya tetap terbaca — bukan
// "diperbaiki" menjadi masuk akal oleh orang berikutnya.
func TestKomiteBiasaMenyetujuiTanpaMenerapkanHarga(t *testing.T) {
	command := keputusan(t, "KOMITESAYA", true)

	require.False(t, command.ApplyPrice)
	require.False(t, command.MarkDocument)
	require.Empty(t, command.CascadeTo)
}

func TestMenolakSelaluMenandaiDokumen(t *testing.T) {
	command := keputusan(t, "KOMITESAYA", false)

	require.True(t, command.MarkDocument)
	require.False(t, command.ApplyPrice)
}

// Jenjang TERAKHIR yang menyetujui-lah yang menerapkan harga.
func TestJenjangTerakhirMenyetujuiMenerapkanHarga(t *testing.T) {
	command := keputusan(t, "DANIELLISWANDI", true)

	require.True(t, command.ApplyPrice)
	require.Empty(t, command.CascadeTo)
}

// Penolakan oleh jenjang PERTAMA menutup jenjang berikutnya sekaligus — tidak ada gunanya
// menanyakan persetujuan atas harga yang sudah ditolak.
func TestPenolakanJenjangPertamaMenutupJenjangBerikutnya(t *testing.T) {
	// MARIATRIELSA memutuskan atas nama BAMBANGSETIADJIGUNAWAN — jenjang pertama.
	command := keputusan(t, "MARIATRIELSA", false)

	require.Equal(t, "BAMBANGSETIADJIGUNAWAN", command.Reviewer.Name)
	require.Equal(t, "DANIELLISWANDI", command.CascadeTo)
	require.True(t, command.MarkDocument)
}

// Persetujuan jenjang pertama TIDAK menutup jenjang berikutnya, dan tidak menerapkan harga —
// ia hanya meneruskan giliran.
func TestPersetujuanJenjangPertamaHanyaMeneruskanGiliran(t *testing.T) {
	command := keputusan(t, "BAMBANGSETIADJIGUNAWAN", true)

	require.Empty(t, command.CascadeTo)
	require.False(t, command.ApplyPrice)
	require.False(t, command.MarkDocument)
}
