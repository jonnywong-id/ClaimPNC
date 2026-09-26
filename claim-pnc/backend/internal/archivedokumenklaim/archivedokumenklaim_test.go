package archivedokumenklaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
)

// # Saringan lini bisnis pada daftar kirim ke cabang
//
// Uji ini menahan perilaku yang TAMPAK TERBALIK, dan itu memang tugasnya. Petugas
// berjabatan PA tidak melihat berkas Personal Accident, dan petugas Travel tidak melihat
// berkas Travel.
//
// Sumbernya `Activity/GetDataArchiveCabangKlaim-Act.xml`, dan Work Owner memutuskan
// 2026-09-24 ia direplikasi apa adanya (`P-5` murni). Tanpa uji ini, orang berikutnya yang
// membaca kodenya akan mengira itu salah ketik lalu "membetulkannya" — dan pembetulan itu
// mengubah siapa melihat berkas siapa tanpa satu pun galat.
func TestSaringanLiniBisnisMengikutiJabatan(t *testing.T) {
	cases := []struct {
		position string
		hidden   []string
		reason   string
	}{
		{
			position: archivedokumenklaim.PositionNonMBU,
			hidden: []string{
				archivedokumenklaim.GroupPanelPersonalAccident,
				archivedokumenklaim.GroupPanelTravel,
			},
			reason: "NONMBU tidak melihat PA maupun Travel",
		},
		{
			position: archivedokumenklaim.PositionPA,
			hidden:   []string{archivedokumenklaim.GroupPanelPersonalAccident},
			reason:   "jabatan PA justru TIDAK melihat berkas PA — replikasi sadar",
		},
		{
			position: archivedokumenklaim.PositionTravel,
			hidden:   []string{archivedokumenklaim.GroupPanelTravel},
			reason:   "jabatan TRAVEL justru TIDAK melihat berkas Travel — replikasi sadar",
		},
		{
			position: "PICTEKNIK",
			hidden:   []string{},
			reason:   "jabatan lain tidak disaring sama sekali",
		},
		{
			position: "",
			hidden:   []string{},
			reason:   "jabatan kosong tidak disaring sama sekali",
		},
	}

	for _, c := range cases {
		t.Run(c.position, func(t *testing.T) {
			scope := archivedokumenklaim.BranchScopeFor(c.position)
			require.ElementsMatch(t, c.hidden, scope.ExcludedGroupPanels, c.reason)
			require.Len(t, scope.ExcludedGroupPanels, len(c.hidden), c.reason)
		})
	}
}

// Kapitalisasi jabatan tidak menentukan.
//
// Sistem lama membandingkannya dengan `==` apa adanya, sehingga jabatan yang tersimpan
// berhuruf kecil jatuh ke cabang "lainnya" dan petugasnya melihat SELURUH lini bisnis.
// Penormalan di sini menutupnya, mengikuti perlakuan yang sama pada kapitalisasi peran di
// `D-58`.
func TestSaringanLiniBisnisTidakPekaBesarKecilHuruf(t *testing.T) {
	require.Equal(t,
		archivedokumenklaim.BranchScopeFor("NONMBU"),
		archivedokumenklaim.BranchScopeFor(" nonmbu "),
	)
}

func TestPencarianKataKunciWajibDiisi(t *testing.T) {
	_, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Mode: string(archivedokumenklaim.ModeKeyword),
	})

	var validation *archivedokumenklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, archivedokumenklaim.FieldKeyword, validation.Violations[0].Field)
}

// Kata kunci DIBESARKAN hurufnya sebelum dikirim ke kueri.
//
// Kueri lama membandingkan `UPPER(NOKLAIM)` dengan nilai yang TIDAK dibesarkan, sehingga
// nomor klaim berhuruf kecil tidak pernah cocok. Perbedaan hasilnya disebut terang di
// docs/keputusan-implementasi.md.
func TestKataKunciDibesarkanHurufnya(t *testing.T) {
	criteria, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Mode:    string(archivedokumenklaim.ModeKeyword),
		Keyword: "  pnc-100001  ",
	})

	require.NoError(t, err)
	require.Equal(t, "PNC-100001", criteria.Keyword)
}

// Isian yang tidak berlaku bagi mode terpilih DIBUANG, bukan dibawa diam-diam.
//
// Nilai sisa dari mode sebelumnya yang ikut masuk kueri adalah kelas cacat yang tidak ada
// di sistem lama — di sana tiap isian punya propertinya sendiri.
func TestIsianDiLuarModeDibuang(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	byKeyword, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Mode:    string(archivedokumenklaim.ModeKeyword),
		Keyword: "BOX-A-01",
		From:    &from,
		To:      &to,
	})
	require.NoError(t, err)
	require.Nil(t, byKeyword.From)
	require.Nil(t, byKeyword.To)

	byDate, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Mode:    string(archivedokumenklaim.ModeInputDate),
		Keyword: "BOX-A-01",
		From:    &from,
		To:      &to,
	})
	require.NoError(t, err)
	require.Empty(t, byDate.Keyword)
}

func TestRentangTanggalTerbalikDitolak(t *testing.T) {
	from := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	_, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Mode: string(archivedokumenklaim.ModeInputDate),
		From: &from,
		To:   &to,
	})

	var validation *archivedokumenklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, archivedokumenklaim.FieldTo, validation.Violations[0].Field)
}

// Jam dibuang dari rentang tanggal.
//
// Kueri lama membandingkannya dengan `trunc(TGLINPUT)`, dan membawa serta jam akan membuat
// pencarian gagal tepat pada baris yang jamnya bukan tengah malam.
func TestRentangTanggalMembuangJam(t *testing.T) {
	from := time.Date(2024, 3, 1, 16, 45, 0, 0, time.UTC)
	to := time.Date(2024, 3, 31, 23, 59, 0, 0, time.UTC)

	criteria, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Mode: string(archivedokumenklaim.ModeInputDate),
		From: &from,
		To:   &to,
	})

	require.NoError(t, err)
	require.Equal(t, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), *criteria.From)
	require.Equal(t, time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC), *criteria.To)
}

func TestTipeInputKlaimWajibDikenal(t *testing.T) {
	_, err := archivedokumenklaim.NewClaimCriteria("nomor_ktp", "12345")

	var validation *archivedokumenklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, archivedokumenklaim.FieldClaimSearchType, validation.Violations[0].Field)
}

// Nama tertanggung TIDAK dibesarkan hurufnya.
//
// Kedua kueri lama membandingkannya apa adanya dengan QQNAME — tanpa UPPER di kedua sisi.
// Membesarkannya di lapisan domain akan mengubah hasil, bukan memperbaikinya; penyeragaman
// huruf untuk pencarian klaim terjadi di repo, tempat kolomnya ikut dibesarkan.
func TestNilaiPencarianKlaimTidakDiubahHurufnya(t *testing.T) {
	criteria, err := archivedokumenklaim.NewClaimCriteria(
		string(archivedokumenklaim.ClaimByInsuredName), "  Contoh Tertanggung  ")

	require.NoError(t, err)
	require.Equal(t, "Contoh Tertanggung", criteria.Value)
}

// Seluruh pelanggaran formulir dikumpulkan sekaligus, tidak berhenti pada yang pertama.
//
// Ini bukan kerapian: formulir yang mengembalikan satu galat per percobaan membuat
// pengguna menekan Simpan enam kali untuk mengetahui enam isian yang salah.
func TestFormulirMengumpulkanSeluruhPelanggaran(t *testing.T) {
	_, err := archivedokumenklaim.NewDraft(
		archivedokumenklaim.DraftInput{},
		archivedokumenklaim.Caller{Login: "USERARSIP1"},
		"01",
	)

	var validation *archivedokumenklaim.ValidationError
	require.ErrorAs(t, err, &validation)

	fields := make([]string, 0, len(validation.Violations))
	for _, v := range validation.Violations {
		fields = append(fields, v.Field)
	}

	require.ElementsMatch(t, []string{
		archivedokumenklaim.FieldClaimNumber,
		archivedokumenklaim.FieldSheetCount,
		archivedokumenklaim.FieldDocumentType,
		archivedokumenklaim.FieldDocumentKind,
		archivedokumenklaim.FieldBoxName,
		archivedokumenklaim.FieldFillingCode,
		archivedokumenklaim.FieldReceivedDate,
	}, fields)
}

// Identitas yang tidak terbaca BUKAN pelanggaran isian.
//
// Ia dijawab berbeda oleh transport — 409, bukan 422 — karena yang salah bukan isian yang
// dapat diperbaiki pengguna di formulir.
func TestFormulirTanpaIdentitasBukanGalatValidasi(t *testing.T) {
	_, err := archivedokumenklaim.NewDraft(
		validDraftInput(),
		archivedokumenklaim.Caller{},
		"01",
	)

	require.ErrorIs(t, err, archivedokumenklaim.ErrCallerUnknown)
}

func TestFormulirSahMenghasilkanDraft(t *testing.T) {
	draft, err := archivedokumenklaim.NewDraft(
		validDraftInput(),
		archivedokumenklaim.Caller{Login: "  USERARSIP1  ", Position: "NONMBU"},
		"  01  ",
	)

	require.NoError(t, err)
	require.True(t, draft.IsNew())
	require.Equal(t, "USERARSIP1", draft.InputUser)
	require.Equal(t, "01", draft.BranchCode)
	require.Equal(t, "006", draft.GroupPanel)
}

func TestJumlahLembarTidakWajarDitolak(t *testing.T) {
	input := validDraftInput()
	input.SheetCount = archivedokumenklaim.MaxSheetCount + 1

	_, err := archivedokumenklaim.NewDraft(
		input, archivedokumenklaim.Caller{Login: "USERARSIP1"}, "01")

	var validation *archivedokumenklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, archivedokumenklaim.FieldSheetCount, validation.Violations[0].Field)
}

// Isian yang ikut dari klaim DIPOTONG, bukan menolak seluruh penyimpanan.
//
// Keenamnya tidak diketik pengguna. Menolak penyimpanan karena nama tertanggung di basis
// data kebetulan panjang akan membuat berkas klaim itu tidak pernah dapat diarsipkan, dan
// pengguna tidak punya cara memperbaikinya dari layar ini.
func TestIsianTurunanDipotongBukanDitolak(t *testing.T) {
	input := validDraftInput()
	input.InsuredName = strings500()

	draft, err := archivedokumenklaim.NewDraft(
		input, archivedokumenklaim.Caller{Login: "USERARSIP1"}, "01")

	require.NoError(t, err)
	require.Len(t, draft.InsuredName, archivedokumenklaim.MaxNameLength)
}

// Bentuk NoDokumen adalah kontrak dengan sistem Arsip milik tim lain.
//
// Ketiga bagiannya dirangkai dengan garis miring, dan itulah satu-satunya pengenal yang
// diterima sistem itu. Mengubah pemisahnya berarti mengubah kontrak.
func TestNomorDokumenPengirimanDirangkaiDenganGarisMiring(t *testing.T) {
	shipment := archivedokumenklaim.Shipment{
		ID:          42,
		BoxName:     " BOX-A-01 ",
		FillingCode: " FIL-2024-001 ",
	}

	require.Equal(t, "42/BOX-A-01/FIL-2024-001", shipment.DocumentNumber())
}

// Bagian yang kosong tetap menyisakan pemisahnya, persis perangkaian teks sistem lama.
func TestNomorDokumenMenyisakanPemisahSaatBagianKosong(t *testing.T) {
	shipment := archivedokumenklaim.Shipment{ID: 7}
	require.Equal(t, "7//", shipment.DocumentNumber())
}

func TestPaginasiDibetulkanBukanDitolak(t *testing.T) {
	clean := archivedokumenklaim.Pagination{Page: 0, Size: 0}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, archivedokumenklaim.DefaultPageSize, clean.Size)

	capped := archivedokumenklaim.Pagination{Page: 3, Size: 5000}.Normalize()
	require.Equal(t, archivedokumenklaim.MaxPageSize, capped.Size)
}

func TestTotalHalamanMinimalSatu(t *testing.T) {
	page := archivedokumenklaim.ArchivePage{
		Total:      0,
		Pagination: archivedokumenklaim.Pagination{Page: 1, Size: 20},
	}
	require.Equal(t, 1, page.TotalPages())
}

// validDraftInput adalah formulir yang seluruh isiannya sah.
func validDraftInput() archivedokumenklaim.DraftInput {
	received := time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC)
	loss := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

	return archivedokumenklaim.DraftInput{
		ClaimNumber:          "PNC-100001",
		PolicyNumber:         "POL-2024-000001",
		InsuredName:          "CONTOH TERTANGGUNG",
		LossDate:             &loss,
		TechnicalPIC:         "PICTEKNIK1",
		GroupPanel:           "006",
		DocumentReceivedDate: &received,
		SheetCount:           12,
		DocumentTypeCode:     "0001",
		DocumentKindCode:     "000101",
		BoxName:              "BOX-A-01",
		FillingCode:          "FIL-2024-001",
	}
}

// strings500 adalah teks 500 karakter untuk menguji pemotongan.
func strings500() string {
	buffer := make([]byte, 500)
	for index := range buffer {
		buffer[index] = 'A'
	}
	return string(buffer)
}
