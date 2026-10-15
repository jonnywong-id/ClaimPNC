package mastersurveyors_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
)

// TestNameKeyMengabaikanHurufDanSeluruhSpasi menguji aturan keunikan nama.
//
// Ia meniru `Activity/ValidasiMasterSurveyor-Act.xml`:
//
//	@toUpperCase(@replaceAll(.NAME," ","")) == @toUpperCase(@replaceAll(TempDetailSurveyors.NAME," ",""))
//
// Uji ini yang menjaga agar aturannya tidak diam-diam dilonggarkan menjadi
// `UPPER(TRIM(...))` seperti milik modul induk — yang akan membuat nama ganda lolos.
func TestNameKeyMengabaikanHurufDanSeluruhSpasi(t *testing.T) {
	sama := []string{
		"BUDI SANTOSO",
		"Budi Santoso",
		"budi santoso",
		"budisantoso",
		"  Budi   Santoso  ",
		"BudiSantoso",
	}

	first := mastersurveyors.NameKey(sama[0])
	require.Equal(t, "BUDISANTOSO", first)

	for _, name := range sama[1:] {
		require.Equal(t, first, mastersurveyors.NameKey(name),
			"%q seharusnya dianggap nama yang sama dengan %q", name, sama[0])
	}
}

// TestNameKeyMembedakanNamaYangMemangBerbeda memastikan aturannya tidak terlalu longgar.
func TestNameKeyMembedakanNamaYangMemangBerbeda(t *testing.T) {
	require.NotEqual(t,
		mastersurveyors.NameKey("Budi Santoso"),
		mastersurveyors.NameKey("Budi Santosa"),
	)
}

// TestRequiresAppLoginHanyaUntukInternalSurveyor menguji aturan dari langkah 8
// `CNMInsertDetailSurveyors_act`: "error if \"internal surveyor\" & LOGIN_APLIKASI is null".
func TestRequiresAppLoginHanyaUntukInternalSurveyor(t *testing.T) {
	internal := mastersurveyors.Surveyor{TypeCode: mastersurveyors.InternalTypeCode}
	require.True(t, internal.RequiresAppLogin())

	for _, code := range []string{"1002", "1003", "1004", ""} {
		other := mastersurveyors.Surveyor{TypeCode: code}
		require.False(t, other.RequiresAppLogin(), "tipe %q bukan internal surveyor", code)
	}
}

// TestCheckMenolakLoginKosongPadaSurveyorInternal menguji aturan wajib bersyaratnya.
func TestCheckMenolakLoginKosongPadaSurveyorInternal(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{
		TypeCode: mastersurveyors.InternalTypeCode,
		Name:     "Surveyor Contoh",
	}

	err := surveyor.Check()
	require.Error(t, err)

	var violation *mastersurveyors.ValidationError
	require.ErrorAs(t, err, &violation)
	require.Contains(t, violation.Field, "login_aplikasi")
}

// TestCheckMenerimaLoginKosongPadaSurveyorEksternal adalah pasangan uji di atas.
//
// Ia yang menjaga agar kewajiban login tidak diam-diam diberlakukan ke seluruh tipe —
// kesalahan yang akan menolak 24 dari 43 surveyor yang ada hari ini.
func TestCheckMenerimaLoginKosongPadaSurveyorEksternal(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{
		TypeCode: "1002",
		Name:     "Adjuster Contoh",
		// Email diisi karena ia WAJIB bagi seluruh tipe (langkah 5). Yang diuji di sini
		// hanya bahwa LOGIN kosong tidak ditolak untuk tipe non-internal.
		Email: "adjuster@contoh.invalid",
	}
	require.NoError(t, surveyor.Check())
}

// TestCheckMengumpulkanSeluruhPelanggaranSekaligus menguji kesetaraan perilaku dengan
// sistem lama, yang menampilkan seluruh pesan validasi bersamaan.
func TestCheckMengumpulkanSeluruhPelanggaranSekaligus(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{Email: "bukan-email"}

	err := surveyor.Check()
	require.Error(t, err)

	var violation *mastersurveyors.ValidationError
	require.ErrorAs(t, err, &violation)

	// Tipe kosong, nama kosong, dan email cacat — ketiganya disebut sekaligus, bukan satu
	// per satu.
	require.Contains(t, violation.Field, "kode_tipe")
	require.Contains(t, violation.Field, "nama")
	require.Contains(t, violation.Field, "email")
}

// TestCheckMenolakNamaTerlaluPanjang menguji batas panjang nama.
func TestCheckMenolakNamaTerlaluPanjang(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{
		TypeCode: "1002",
		Name:     strings.Repeat("A", mastersurveyors.MaxNameLength+1),
	}

	err := surveyor.Check()
	require.Error(t, err)

	var violation *mastersurveyors.ValidationError
	require.ErrorAs(t, err, &violation)
	require.Contains(t, violation.Field, "nama")
}

// TestCheckMewajibkanEmail menguji aturan dari langkah 5 `CNMInsertDetailSurveyors_act`.
//
// Pesannya terbaca apa adanya di langkah 4 — `local.email := "Email harus diisi"` — dan
// dipancarkan langkah 5 dengan prasyarat `TempDetailSurveyors.EMAIL==""`.
//
// Email sempat diperlakukan OPSIONAL di modul ini, dan uji ini yang menjaga agar
// kekeliruan itu tidak kembali.
func TestCheckMewajibkanEmail(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{TypeCode: "1002", Name: "Adjuster Contoh"}

	err := surveyor.Check()
	require.Error(t, err)

	var violation *mastersurveyors.ValidationError
	require.ErrorAs(t, err, &violation)
	require.Contains(t, violation.Field, "email")
}

// TestCheckMenerimaSurveyorLengkap memastikan aturan wajibnya tidak terlalu ketat.
func TestCheckMenerimaSurveyorLengkap(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{
		TypeCode: "1002",
		Name:     "Adjuster Contoh",
		Email:    "adjuster@contoh.invalid",
	}
	require.NoError(t, surveyor.Check())
}

// TestAccountRequestForMeniruKedelapanParameterPega menguji bahwa permintaan akun
// dibentuk persis seperti yang dikirim `CNMInsertDetailSurveyors_act` ke
// `GCNMCreateOperator`.
func TestAccountRequestForMeniruKedelapanParameterPega(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{
		TypeCode: mastersurveyors.InternalTypeCode,
		Name:     "Surveyor Contoh",
		AppLogin: "SURVEYORCONTOH",
	}

	req := mastersurveyors.AccountRequestFor(surveyor)

	require.Equal(t, "ASM", req.Organization)
	require.Equal(t, "PNC", req.Division)
	require.Equal(t, "Internal", req.Unit)
	require.Equal(t, "SURVEYORCONTOH", req.UserID)
	require.Equal(t, "Surveyor Contoh", req.UserName)
	require.Equal(t, "SURVEYORCONTOH123456", req.Password)
	require.Equal(t, "GCNMFW:PNCSurveyor", req.AccessGroup)

	// `GCNMCreateOperator` menyetel pyChangePasswordOnNextLogin = "True", sehingga sandi
	// di atas adalah sandi SEMENTARA. Tanpa penanda ini ia menjadi sandi tetap yang dapat
	// ditebak dari nama login — dan itu perbedaan yang menentukan.
	require.True(t, req.MustChangePassword)
}

// TestAccountRequestUnitEksternalUntukTipeLain menguji cabang keduanya.
//
// Unit BUKAN keterangan tampilan: `docs/Steering/11-SECURITY.md` §3.2 mencatat batas data
// ditegakkan dengan membandingkan `OperatorID.pyOrgUnit != "Eksternal"`, sehingga nilai
// ini menentukan data siapa yang boleh dilihat pemilik akun.
func TestAccountRequestUnitEksternalUntukTipeLain(t *testing.T) {
	surveyor := mastersurveyors.Surveyor{TypeCode: "1002", Name: "Adjuster", AppLogin: "ADJ"}
	require.Equal(t, "Eksternal", mastersurveyors.AccountRequestFor(surveyor).Unit)
}

// TestStatusLabelSamaDenganMasterRekening menjaga sebutan status tetap seragam.
//
// Dua layar persetujuan komite yang memakai sebutan berbeda untuk keadaan yang sama akan
// terbaca sebagai dua hal berbeda oleh peran yang sama.
func TestStatusLabelSamaDenganMasterRekening(t *testing.T) {
	require.Equal(t, "Menunggu", mastersurveyors.StatusPending.Label())
	require.Equal(t, "Committee Approve", mastersurveyors.StatusApproved.Label())
	require.Equal(t, "Committee Reject", mastersurveyors.StatusRejected.Label())
	require.Equal(t, "", mastersurveyors.ApprovalStatus("9").Label())
}

// TestKnownHanyaMenerimaTigaNilai menguji penyaring status.
func TestKnownHanyaMenerimaTigaNilai(t *testing.T) {
	require.True(t, mastersurveyors.StatusPending.Known())
	require.True(t, mastersurveyors.StatusApproved.Known())
	require.True(t, mastersurveyors.StatusRejected.Known())
	require.False(t, mastersurveyors.ApprovalStatus("").Known())
	require.False(t, mastersurveyors.ApprovalStatus("3").Known())
}

// TestAssignableHanyaSetelahDisetujui menguji syarat penugasan.
//
// Satu syarat saja, berbeda dari `masterrekening.Usable` yang punya dua: D_SURVEYORS
// TIDAK punya kolom penanda aktif, dan menambahkannya berarti mengarang keadaan yang
// tidak dapat disimpan di mana pun.
func TestAssignableHanyaSetelahDisetujui(t *testing.T) {
	require.True(t, mastersurveyors.Surveyor{Status: mastersurveyors.StatusApproved}.Assignable())
	require.False(t, mastersurveyors.Surveyor{Status: mastersurveyors.StatusPending}.Assignable())
	require.False(t, mastersurveyors.Surveyor{Status: mastersurveyors.StatusRejected}.Assignable())
}

// TestCleanMembuangSpasiTepi menguji perapian yang menjaga perbandingan kode.
func TestCleanMembuangSpasiTepi(t *testing.T) {
	clean := mastersurveyors.Surveyor{
		ID:       "  1000001  ",
		TypeCode: " 1001 ",
		Name:     "  Surveyor Contoh  ",
		AppLogin: "  LOGIN  ",
	}.Clean()

	require.Equal(t, "1000001", clean.ID)
	require.Equal(t, "1001", clean.TypeCode)
	require.Equal(t, "Surveyor Contoh", clean.Name)
	require.Equal(t, "LOGIN", clean.AppLogin)
}
