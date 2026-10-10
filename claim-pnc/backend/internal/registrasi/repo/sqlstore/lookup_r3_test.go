package sqlstore

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Kurs: mata uang kosong dan kurs yang tidak ada menjadi ErrExchangeRateNotFound.
func TestExchangeRateFind(t *testing.T) {
	db, mock := be4DB(t)
	s := NewExchangeRateSource(db)
	ctx := context.Background()
	day := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)

	_, err := s.Find(ctx, " ", day)
	require.ErrorIs(t, err, registrasi.ErrExchangeRateNotFound)

	mock.ExpectQuery(be4Q("kurs_pada_tanggal")).WithArgs("USD", "USD", day).WillReturnRows(sqlmock.NewRows([]string{"k"}).AddRow("15000,5"))
	rate, err := s.Find(ctx, " USD ", day)
	require.NoError(t, err)
	require.Equal(t, registrasi.ExchangeRate(150_005_000), rate)

	mock.ExpectQuery(be4Q("kurs_pada_tanggal")).WillReturnRows(sqlmock.NewRows([]string{"k"}))
	_, err = s.Find(ctx, "USD", day)
	require.ErrorIs(t, err, registrasi.ErrExchangeRateNotFound)
	require.ErrorContains(t, err, "USD pada 2026-06-05")

	mock.ExpectQuery(be4Q("kurs_pada_tanggal")).WillReturnError(be4Boom)
	_, err = s.Find(ctx, "USD", day)
	require.ErrorContains(t, err, "membaca kurs USD")

	mock.ExpectQuery(be4Q("kurs_pada_tanggal")).WillReturnRows(sqlmock.NewRows([]string{"k"}).AddRow("abc"))
	_, err = s.Find(ctx, "USD", day)
	require.ErrorContains(t, err, "kurs USD pada 2026-06-05 tidak terbaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestParseRate(t *testing.T) {
	rate, err := parseRate(" 1 ")
	require.NoError(t, err)
	require.Equal(t, registrasi.ExchangeRateOne, rate)
	rate, err = parseRate("14285.7143")
	require.NoError(t, err)
	require.Equal(t, registrasi.ExchangeRate(142_857_143), rate)

	for raw, want := range map[string]string{
		"":         "nilainya kosong",
		"1.000,50": "titik dan koma sekaligus",
		"x":        "bukan angka",
		"0":        "bukan kurs yang sah",
		"-2":       "bukan kurs yang sah",
	} {
		_, err := parseRate(raw)
		require.ErrorContains(t, err, want, raw)
	}
}

// Parameter: ambang dibaca dari JSON; parameter kosong, rusak, atau tidak positif ditolak.
func TestParameterLargeLossThreshold(t *testing.T) {
	db, mock := be4DB(t)
	p := NewParameter(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("parameter_ambil")).WithArgs(paramLargeLossThreshold).
		WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(`{"nilai_sen":100000000000}`))
	v, err := p.LargeLossThreshold(ctx)
	require.NoError(t, err)
	require.Equal(t, registrasi.Money(100_000_000_000), v)

	for body, want := range map[string]string{
		`{"nilai_sen":1.5}`:  "tidak berisi angka",
		`{"nilai_sen":0}`:    "ambang harus lebih dari nol",
		`bukan json`:         "tidak dapat diuraikan",
		`{"nilai_sen":1.5e}`: "tidak dapat diuraikan",
	} {
		mock.ExpectQuery(be4Q("parameter_ambil")).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(body))
		_, err := p.LargeLossThreshold(ctx)
		require.ErrorContains(t, err, want, body)
	}

	mock.ExpectQuery(be4Q("parameter_ambil")).WillReturnRows(sqlmock.NewRows([]string{"b"}))
	_, err = p.LargeLossThreshold(ctx)
	require.ErrorContains(t, err, "belum diisi di POOLDATA.M_PARAMETER")
	mock.ExpectQuery(be4Q("parameter_ambil")).WillReturnError(be4Boom)
	_, err = p.LargeLossThreshold(ctx)
	require.ErrorContains(t, err, "membaca parameter "+paramLargeLossThreshold)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestParameterLargeLossRecipients(t *testing.T) {
	db, mock := be4DB(t)
	p := NewParameter(db)
	ctx := context.Background()
	body := `{"umum":["dir@x"],"per_lini":{"006":["uw.fire@x"]}}`

	mock.ExpectQuery(be4Q("parameter_ambil")).WithArgs(paramLargeLossRecipients).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(body))
	got, err := p.LargeLossRecipients(ctx, registrasi.LineFire)
	require.NoError(t, err)
	require.Equal(t, []string{"dir@x", "uw.fire@x"}, got)

	mock.ExpectQuery(be4Q("parameter_ambil")).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(`{"per_lini":{}}`))
	_, err = p.LargeLossRecipients(ctx, registrasi.LineFire)
	require.ErrorContains(t, err, "tidak memuat penerima untuk lini")

	mock.ExpectQuery(be4Q("parameter_ambil")).WillReturnError(be4Boom)
	_, err = p.LargeLossRecipients(ctx, registrasi.LineFire)
	require.ErrorIs(t, err, be4Boom)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penugasan: workbasket, operator bernama, pemanggil, admin klaim, PIC teknik, lalu
// petugas teknis paling ringan dari basis data.
func TestAssignerRules(t *testing.T) {
	db, mock := be4DB(t)
	a := NewAssigner(db)
	ctx := context.Background()

	got, err := a.Assign(ctx, registrasi.Stage{Queue: registrasi.QueueWorkbasket, Workbasket: "WB"}, registrasi.Claim{}, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Workbasket: "WB"}, got)

	got, err = a.Assign(ctx, registrasi.Stage{Operator: " DOKTER "}, registrasi.Claim{}, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "DOKTER"}, got)

	current := registrasi.Stage{ID: "S", Router: registrasi.RouterCurrentOperator}
	got, err = a.Assign(ctx, current, registrasi.Claim{}, " NIK1 ")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "NIK1"}, got)
	_, err = a.Assign(ctx, current, registrasi.Claim{}, " ")
	require.ErrorContains(t, err, `tahap "S" menuntut pemanggil`)

	admin := registrasi.Stage{ID: "A", Router: registrasi.RouterPNCAdmin}
	got, err = a.Assign(ctx, admin, registrasi.Claim{CreatedBy: " ADMIN "}, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "ADMIN"}, got)
	got, err = a.Assign(ctx, admin, registrasi.Claim{}, " NIK1 ")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "NIK1"}, got)
	_, err = a.Assign(ctx, admin, registrasi.Claim{}, "")
	require.ErrorContains(t, err, `tahap "A" tidak punya admin klaim maupun pemanggil`)

	technical := registrasi.Stage{ID: "T", Router: registrasi.RouterPNCTechnical}
	got, err = a.Assign(ctx, technical, registrasi.Claim{TechnicalPIC: " PIC1 "}, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "PIC1"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAssignerLightestTechnician(t *testing.T) {
	db, mock := be4DB(t)
	a := NewAssigner(db)
	ctx := context.Background()
	technical := registrasi.Stage{ID: "T", Router: registrasi.RouterPNCTechnical}

	row := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"o"}).AddRow(" TEK1 ") }
	fire := registrasi.Policy{Line: registrasi.LineFire}

	pa := registrasi.Policy{Line: registrasi.LinePersonalAccident, BusinessType: "PA"}
	position := func(v string) *sqlmock.Rows { return sqlmock.NewRows([]string{"p"}).AddRow(v) }

	// Jabatan PA + TKI: GETDATA_PICTEKNIK memilih DIBADYASANTI, lalu COUNTER_QUOTA naik.
	mock.ExpectQuery(be4Q("pic_teknik_jabatan")).WithArgs("NIK1").WillReturnRows(position("PA"))
	mock.ExpectQuery(be4Q("pic_teknik_pa")).WithArgs("DIBADYASANTI").WillReturnRows(row())
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban")).WithArgs(" TEK1 ").WillReturnResult(sqlmock.NewResult(0, 1))
	got, err := a.Assign(ctx, technical, registrasi.Claim{Policy: pa, TKI: true}, " nik1 ")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "TEK1"}, got)

	// Jabatan selain PA (atau tidak ada di M_LOGIN_PNC): cabang TRAVEL, beban paling ringan.
	mock.ExpectQuery(be4Q("pic_teknik_jabatan")).WillReturnRows(sqlmock.NewRows([]string{"p"}))
	mock.ExpectQuery(be4Q("pic_teknik_travel")).WillReturnRows(row())
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban")).WithArgs(" TEK1 ").WillReturnResult(sqlmock.NewResult(0, 1))
	_, err = a.Assign(ctx, technical, registrasi.Claim{Policy: pa}, "NIK1")
	require.NoError(t, err)

	// Petugas PA tidak ada di master: prosedur keluar tanpa PIC.
	mock.ExpectQuery(be4Q("pic_teknik_jabatan")).WillReturnRows(position("PA"))
	mock.ExpectQuery(be4Q("pic_teknik_pa")).WithArgs("ESTHERSIMBOLON").WillReturnRows(sqlmock.NewRows([]string{"o"}))
	got, err = a.Assign(ctx, technical, registrasi.Claim{Policy: pa}, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: registrasi.OperatorUnassigned}, got)

	mock.ExpectQuery(be4Q("pic_teknik_jabatan")).WillReturnError(be4Boom)
	_, err = a.Assign(ctx, technical, registrasi.Claim{Policy: pa}, "NIK1")
	require.ErrorContains(t, err, "membaca jabatan operator")

	// Rotasi Team A/B kolom FLAG2 (< 1M): terakhir A → B.
	rotateSmall := func(last any, next string) {
		mock.ExpectQuery(be4Q("rotasi_tim_baca_kecil")).WillReturnRows(sqlmock.NewRows([]string{"f"}).AddRow(last))
		mock.ExpectExec(be4Q("rotasi_tim_tulis_kecil")).WithArgs(next).WillReturnResult(sqlmock.NewResult(0, 1))
	}

	// NONMBU < 1M, ASM leader: rotasi, tanpa saringan tim C, lalu COUNTER_QUOTA petugas terpilih
	// naik (AddTJobCounterPIC_SQL) supaya giliran berputar.
	rotateSmall("A", "B")
	mock.ExpectQuery(be4Q("pic_teknik_nonmbu")).WithArgs(registrasi.ExcludedTechnicalPIC, "N").WillReturnRows(row())
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban")).WithArgs("TEK1").WillReturnResult(sqlmock.NewResult(0, 1))
	_, err = a.Assign(ctx, technical, registrasi.Claim{Policy: fire}, "NIK1")
	require.NoError(t, err)

	// NONMBU > 1M, ASM member: tim C, diurutkan dan dinaikkan COUNTER_QUOTA2 (AddTJobCounterPIC_SQL_22).
	member := fire
	member.Coinsurance.Role = "MEMBER"
	mock.ExpectQuery(be4Q("pic_teknik_nonmbu_besar")).WithArgs(registrasi.ExcludedTechnicalPIC, "Y").WillReturnRows(row())
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban_besar")).WithArgs("TEK1").WillReturnResult(sqlmock.NewResult(0, 1))
	_, err = a.Assign(ctx, technical, registrasi.Claim{Policy: member, EstimateValue: registrasi.Rupiah(2_000_000_000)}, "NIK1")
	require.NoError(t, err)

	// Kandidat tetap per sumber bisnis: tanpa kueri daftar, tetapi bebannya tetap naik.
	ibs := member
	ibs.SourceOfBusiness = "10001551"
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban")).WithArgs("YOSECHRISTOFER").WillReturnResult(sqlmock.NewResult(0, 1))
	got, err = a.Assign(ctx, technical, registrasi.Claim{Policy: ibs}, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "YOSECHRISTOFER"}, got)

	// Asuransi Kredit: PIC = admin, tanpa kueri dan tanpa beban.
	credit := fire
	credit.BusinessCode = "10165"
	got, err = a.Assign(ctx, technical, registrasi.Claim{Policy: credit, CreatedBy: "ADMIN1"}, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: "ADMIN1"}, got)

	claim := registrasi.Claim{Policy: fire}
	// Tanpa petugas aktif: PNCTeknikRouter langkah 2 menugaskan ke ServicePNC. Rotasi tetap jalan.
	rotateSmall(nil, "A")
	mock.ExpectQuery(be4Q("pic_teknik_nonmbu")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	got, err = a.Assign(ctx, technical, claim, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: registrasi.OperatorUnassigned}, got)

	rotateSmall("B", "A")
	mock.ExpectQuery(be4Q("pic_teknik_nonmbu")).WillReturnError(be4Boom)
	_, err = a.Assign(ctx, technical, claim, "NIK1")
	require.ErrorContains(t, err, "memilih petugas teknis")

	// Rotasi > 1M memakai FLAG; galat bacanya menghentikan pemilihan.
	big := registrasi.Claim{Policy: fire, EstimateValue: registrasi.Rupiah(2_000_000_000)}
	mock.ExpectQuery(be4Q("rotasi_tim_baca_besar")).WillReturnRows(sqlmock.NewRows([]string{"f"}))
	mock.ExpectExec(be4Q("rotasi_tim_tulis_besar")).WithArgs("A").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(be4Q("pic_teknik_nonmbu_besar")).WithArgs(registrasi.ExcludedTechnicalPIC, "N").WillReturnRows(row())
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban_besar")).WithArgs("TEK1").WillReturnResult(sqlmock.NewResult(0, 1))
	_, err = a.Assign(ctx, technical, big, "NIK1")
	require.NoError(t, err)
	mock.ExpectQuery(be4Q("rotasi_tim_baca_besar")).WillReturnError(be4Boom)
	_, err = a.Assign(ctx, technical, big, "NIK1")
	require.ErrorContains(t, err, "membaca rotasi tim FLAG")
	mock.ExpectQuery(be4Q("rotasi_tim_baca_besar")).WillReturnRows(sqlmock.NewRows([]string{"f"}).AddRow("B"))
	mock.ExpectExec(be4Q("rotasi_tim_tulis_besar")).WillReturnError(be4Boom)
	_, err = a.Assign(ctx, technical, big, "NIK1")
	require.ErrorContains(t, err, "menulis rotasi tim FLAG")

	mock.ExpectQuery(be4Q("pic_teknik_jabatan")).WillReturnRows(position("PA"))
	mock.ExpectQuery(be4Q("pic_teknik_pa")).WillReturnRows(sqlmock.NewRows([]string{"o"}).AddRow("TEK1"))
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban")).WillReturnError(be4Boom)
	_, err = a.Assign(ctx, technical, registrasi.Claim{Policy: pa}, "NIK1")
	require.ErrorContains(t, err, `menaikkan beban petugas "TEK1"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLookupHelpers(t *testing.T) {
	require.Equal(t, "A", fallback(" A ", "B"))
	require.Equal(t, "B", fallback(" ", "B"))

	require.True(t, parsePegaTime(" ").IsZero())
	require.True(t, parsePegaTime("kemarin").IsZero())
	require.Equal(t, time.Date(2026, 9, 24, 3, 54, 21, 349_000_000, time.UTC), parsePegaTime("20260924T035421.349 GMT"))
	require.Equal(t, time.Date(2026, 9, 24, 3, 54, 21, 0, time.UTC), parsePegaTime("20260924T035421"))
	require.Equal(t, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), parsePegaTime("20260924"))

	for _, s := range []string{"finished", " Complete ", "COMPLETED", "1", "y", "YES"} {
		require.True(t, isSpreadingReady(s), s)
	}
	require.False(t, isSpreadingReady("DRAFT"))

	p, ok := parsePercent("-33.33335")
	require.True(t, ok)
	require.Equal(t, registrasi.Percent(-333_334), p)
	_, ok = parsePercent("abc")
	require.False(t, ok)
	_, ok = parsePercent(" ")
	require.False(t, ok)
}

// Objek polis: tanpa nomor/versi tidak dibaca; tiap sumber memakai kuerinya sendiri.
//
// Urutan kueri sejak coverage pindah ke tabel relasional (Work Owner, 2026-10-06) adalah
// objek, spreading, coverage, lalu nama treaty. Urutan itu bukan selera: spreading dibaca
// SEBELUM coverage karena setiap baris coverage langsung memungut spreading miliknya.
func TestPolicyItemsItems(t *testing.T) {
	db, mock := be4DB(t)
	p := NewPolicyItems(db)
	ctx := context.Background()

	got, err := p.Items(ctx, registrasi.Policy{Number: "POL"})
	require.NoError(t, err)
	require.Nil(t, got)

	mock.ExpectQuery(be4Q("polis_objek_cargo")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(" 1 ", " OBJ1 ", " Kargo ", " Laut "))
	mock.ExpectQuery(be4Q("polis_spreading")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(7)).AddRow("1", "1", "11022", "10001", "OR", "100.0000", "0"))
	mock.ExpectQuery(be4Q("polis_coverage_cargo")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(8)).AddRow("1", "1", " 11022 ", " ICC C ", "100", nil, nil, "0"))
	mock.ExpectQuery(be4Q("jenis_treaty_nama")).WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow("10001", "OR"))
	got, err = p.Items(ctx, registrasi.Policy{Number: " POL ", ProdKe: " 1 ", Line: registrasi.LineMarineCargo})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "OBJ1", got[0].ID)
	require.Equal(t, "Kargo", got[0].Name)
	require.Equal(t, "Laut", got[0].Location)
	require.Len(t, got[0].Coverage, 1)
	require.Equal(t, "11022", got[0].Coverage[0].Code)
	require.Equal(t, "ICC C", got[0].Coverage[0].Name)
	require.Equal(t, registrasi.Rupiah(100), got[0].Coverage[0].TSI)
	require.Equal(t, []registrasi.SourceSpreading{
		{TreatyType: "10001", TreatyName: "OR", Share: registrasi.PercentFull},
	}, got[0].Coverage[0].Spreading)

	mock.ExpectQuery(be4Q("polis_objek_aneka")).WillReturnError(be4Boom)
	_, err = p.Items(ctx, registrasi.Policy{Number: "POL", ProdKe: "1", Line: registrasi.LineMiscellaneous})
	require.ErrorContains(t, err, "membaca objek polis")
	mock.ExpectQuery(be4Q("polis_objek_aneka")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))
	_, err = p.Items(ctx, registrasi.Policy{Number: "POL", ProdKe: "1", Line: registrasi.LineMiscellaneous})
	require.ErrorContains(t, err, "membaca baris objek polis")
	mock.ExpectQuery(be4Q("polis_objek_aneka")).WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(nil, nil, nil, nil).RowError(0, be4Boom))
	_, err = p.Items(ctx, registrasi.Policy{Number: "POL", ProdKe: "1", Line: registrasi.LineMiscellaneous})
	require.ErrorContains(t, err, "menelusuri objek polis")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Objek tanpa satu baris pun TIDAK menjalankan kueri coverage maupun spreading.
//
// Tanpa jalan pintas ini, membuka polis yang objeknya belum ada menembak basis data tiga
// kali untuk mendapat nol baris.
func TestPolicyItemsSkipsCoverageWhenNoObject(t *testing.T) {
	db, mock := be4DB(t)
	p := NewPolicyItems(db)

	mock.ExpectQuery(be4Q("polis_objek_cargo")).WillReturnRows(sqlmock.NewRows(be4Cols(4)))
	got, err := p.Items(context.Background(), registrasi.Policy{Number: "POL", ProdKe: "1", Line: registrasi.LineMarineCargo})
	require.NoError(t, err)
	require.Nil(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Objek properti: kunci penggabungannya INDEXOBJECT, sementara ID objek klaim OBJECTNO.
//
// Lini Fire satu-satunya yang kedua nilainya berbeda, sehingga ia pula yang akan patah
// lebih dulu bila penggabungan keliru memakai ID.
func TestPolicyItemsPropertyCoverages(t *testing.T) {
	db, mock := be4DB(t)
	p := NewPolicyItems(db)
	ctx := context.Background()
	fire := registrasi.Policy{Number: "POL", ProdKe: "1", Line: registrasi.LineFire}

	object := func() {
		mock.ExpectQuery(be4Q("polis_objek_property")).WithArgs("POL", "1").
			WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(" 7 ", " OBJ1 ", " Gudang ", " Jakarta "))
	}
	noSpreading := func() {
		mock.ExpectQuery(be4Q("polis_spreading")).WillReturnRows(sqlmock.NewRows(be4Cols(7)))
	}

	object()
	noSpreading()
	mock.ExpectQuery(be4Q("polis_coverage_property")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(8)).
			AddRow("7", "1", "C1", "Kebakaran", "10", nil, nil, "0").
			AddRow("7", "2", "C2", "Gempa", "20", nil, "25", "0"))
	mock.ExpectQuery(be4Q("jenis_treaty_nama")).WillReturnRows(sqlmock.NewRows(be4Cols(2)))
	got, err := p.Items(ctx, fire)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "OBJ1", got[0].ID)
	require.Len(t, got[0].Coverage, 2)
	require.Equal(t, "C2", got[0].Coverage[1].Code)
	// TSISUBLIMIT menang atas TSI bila terisi (coverageTSI).
	require.Equal(t, registrasi.Rupiah(25), got[0].Coverage[1].TSISublimit)

	object()
	noSpreading()
	mock.ExpectQuery(be4Q("polis_coverage_property")).WillReturnRows(sqlmock.NewRows(be4Cols(8)))
	mock.ExpectQuery(be4Q("jenis_treaty_nama")).WillReturnError(be4Boom)
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca nama treaty")

	object()
	mock.ExpectQuery(be4Q("polis_spreading")).WillReturnError(be4Boom)
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca spreading polis")

	object()
	mock.ExpectQuery(be4Q("polis_spreading")).WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow("1"))
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca baris spreading polis")

	object()
	noSpreading()
	mock.ExpectQuery(be4Q("polis_coverage_property")).WillReturnError(be4Boom)
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca coverage polis")

	object()
	noSpreading()
	mock.ExpectQuery(be4Q("polis_coverage_property")).WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow("1", "2"))
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca baris coverage polis")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPolicyItemsItemOptions(t *testing.T) {
	db, mock := be4DB(t)
	p := NewPolicyItems(db)
	ctx := context.Background()

	got, err := p.ItemOptions(ctx, registrasi.Policy{Line: registrasi.LineTravel}, "O")
	require.NoError(t, err)
	require.Nil(t, got)

	fire := registrasi.Policy{Number: " POL ", ProdKe: " 1 ", Line: registrasi.LineFire}
	cols := []string{"ITEMTYPE", "PROPERTYITEMGROUP", "TSIOBJECTITEM"}
	mock.ExpectQuery(be4Q("item_pilihan_properti")).WithArgs("POL", "1", "OBJ1").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow(" Bangunan ", "G1", "1000.5").
			AddRow("Bangunan", "G1", "1").
			AddRow("", "G2", "5").
			AddRow(nil, nil, nil))
	got, err = p.ItemOptions(ctx, fire, " OBJ1 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.ItemOption{{Name: "Bangunan", Group: "G1", TSI: 100_050}}, got)

	mock.ExpectQuery(be4Q("item_pilihan_properti")).WillReturnError(be4Boom)
	_, err = p.ItemOptions(ctx, fire, "OBJ1")
	require.ErrorContains(t, err, "membaca item properti polis")
	mock.ExpectQuery(be4Q("item_pilihan_properti")).WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow("1"))
	_, err = p.ItemOptions(ctx, fire, "OBJ1")
	require.ErrorContains(t, err, "membaca baris item properti")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPolicyItemsTravelBenefits(t *testing.T) {
	db, mock := be4DB(t)
	p := NewPolicyItems(db)
	ctx := context.Background()

	got, err := p.TravelBenefits(ctx, "  ")
	require.NoError(t, err)
	require.Nil(t, got, "tanpa plan tidak ada kueri")

	mock.ExpectQuery(be4Q("item_pilihan_travel")).WithArgs("10089").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "INDCOVERAGENAME"}).
			AddRow("10919", "   B.1. Bagasi  ").
			AddRow("10920", "B.1. Bagasi").
			AddRow("10921", "B.2. Keterlambatan").
			AddRow("10922", nil))
	got, err = p.TravelBenefits(ctx, " 10089 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.ItemOption{
		{ID: "10919", Name: "B.1. Bagasi"},
		{ID: "10921", Name: "B.2. Keterlambatan"},
	}, got)

	mock.ExpectQuery(be4Q("item_pilihan_travel")).WillReturnError(be4Boom)
	_, err = p.TravelBenefits(ctx, "10089")
	require.ErrorContains(t, err, "membaca manfaat plan Travel")
	mock.ExpectQuery(be4Q("item_pilihan_travel")).WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow("1"))
	_, err = p.TravelBenefits(ctx, "10089")
	require.ErrorContains(t, err, "membaca baris manfaat Travel")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJSONTextAndParseMoney(t *testing.T) {
	var v jsonText
	require.NoError(t, v.UnmarshalJSON([]byte(" null ")))
	require.Equal(t, jsonText(""), v)
	require.NoError(t, v.UnmarshalJSON([]byte(`" a "`)))
	require.Equal(t, jsonText("a"), v)
	require.NoError(t, v.UnmarshalJSON([]byte(` 12.5 `)))
	require.Equal(t, jsonText("12.5"), v)
	require.Error(t, v.UnmarshalJSON([]byte(`"tak tertutup`)))

	require.Equal(t, registrasi.Money(0), parseMoney(" "))
	require.Equal(t, registrasi.Money(0), parseMoney("x"))
	require.Equal(t, registrasi.Money(-150), parseMoney("-1.495"))
	require.Equal(t, registrasi.Money(9_720_000_000), parseMoney("97200000.0000"))
}

// fakeAttendance menjawab absensi per petugas; tanggal dan nomor klaim yang diminta dicatat.
type fakeAttendance struct {
	by    map[string]registrasi.Attendance
	err   map[string]error
	asked []string
}

func (f *fakeAttendance) Attendance(_ context.Context, operator string, date time.Time, claimNumber string) (registrasi.Attendance, error) {
	f.asked = append(f.asked, operator+"|"+date.Format("20060102")+"|"+claimNumber)
	return f.by[operator], f.err[operator]
}

// TestAssignerAttendanceLoop: getRandomTeam_act step 15 — lewati yang tidak masuk, berhenti
// pada akhir pekan/libur, dan kegagalan layanan dianggap hadir.
func TestAssignerAttendanceLoop(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	claim := registrasi.Claim{Number: "PNCN.26.1", DateOfLoss: start.AddDate(0, 3, 0),
		Policy: registrasi.Policy{Line: registrasi.LineFire, Number: "P1", CoverageStart: start, CoverageEnd: start.AddDate(1, 0, 0)}}
	now := time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC) // 8 Oktober WIB
	att := &fakeAttendance{by: map[string]registrasi.Attendance{
		"A": {RuleTimeIn: "000000"},
		"B": {RuleTimeIn: "080000"},
	}, err: map[string]error{}}
	a := NewAssigner(nil).WithAttendance(att, nil)
	a.now = func() time.Time { return now }

	require.Equal(t, "B", a.firstPresent(context.Background(), claim, []string{"A", " B ", "C"}))
	require.Equal(t, []string{"A|20261008|PNCN.26.1", "B|20261008|PNCN.26.1"}, att.asked)

	att.by["A"] = registrasi.Attendance{Day: "SABTU"}
	require.Empty(t, a.firstPresent(context.Background(), claim, []string{"A", "B"}), "akhir pekan: tanpa PIC")

	att.by["A"] = registrasi.Attendance{RuleTimeIn: "000000"}
	att.by["B"] = registrasi.Attendance{RuleTimeIn: "000000"}
	require.Empty(t, a.firstPresent(context.Background(), claim, []string{"A", "B"}), "semua tidak masuk")

	att.err["A"] = be4Boom
	require.Equal(t, "A", a.firstPresent(context.Background(), claim, []string{"A", "B"}), "layanan gagal = hadir")

	// Tanggal kejadian di luar periode polis: jawaban absensi diabaikan.
	outside := claim
	outside.DateOfLoss = start.AddDate(-1, 0, 0)
	att.asked = nil
	require.Equal(t, "A", a.firstPresent(context.Background(), outside, []string{"A", "B"}))
	require.Empty(t, att.asked)

	// Tanpa sumber absensi: kandidat pertama.
	require.Equal(t, "A", NewAssigner(nil).firstPresent(context.Background(), claim, []string{"A", "B"}))
}

// TestTaskStoreUnassigned: kueri agent PIC Teknik otomatis.
func TestTaskStoreUnassigned(t *testing.T) {
	db, mock := be4DB(t)
	s := NewTaskStore(db)
	ctx := context.Background()
	created := time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC)
	row := []driver.Value{"T1", "K1", "PNCN.26.1", "pilih-surveyor", "WORKLIST", nil, "ServicePNC", created, nil, nil, nil}

	mock.ExpectQuery(be4Q("tugas_belum_bertuan")).WithArgs("ServicePNC").
		WillReturnRows(sqlmock.NewRows(be4Cols(11)).AddRow(row...))
	got, err := s.UnassignedTechnicalTasks(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "ServicePNC", got[0].Owner)

	mock.ExpectQuery(be4Q("tugas_kunci_belum_bertuan")).WithArgs("T1", "ServicePNC").
		WillReturnRows(sqlmock.NewRows(be4Cols(11)).AddRow(row...))
	locked, err := s.LockUnassigned(ctx, "T1")
	require.NoError(t, err)
	require.Equal(t, "T1", locked.ID)
	mock.ExpectQuery(be4Q("tugas_kunci_belum_bertuan")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	_, err = s.LockUnassigned(ctx, "T1")
	require.ErrorIs(t, err, registrasi.ErrTaskNotFound)
	mock.ExpectQuery(be4Q("tugas_kunci_belum_bertuan")).WillReturnError(be4Boom)
	_, err = s.LockUnassigned(ctx, "T1")
	require.ErrorContains(t, err, "mengunci tugas T1")

	mock.ExpectExec(be4Q("tugas_pindah_dari_antrean")).WithArgs("PIC1", "T1", "ServicePNC").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Reassign(ctx, "T1", "PIC1"))
	mock.ExpectExec(be4Q("tugas_pindah_dari_antrean")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, s.Reassign(ctx, "T1", "PIC1"), registrasi.ErrTaskAlreadyClaimed)
	mock.ExpectExec(be4Q("tugas_pindah_dari_antrean")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.Reassign(ctx, "T1", "PIC1"), "memindahkan tugas T1")
	require.NoError(t, mock.ExpectationsWereMet())
}
