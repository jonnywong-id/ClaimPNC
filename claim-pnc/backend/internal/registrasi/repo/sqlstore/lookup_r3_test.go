package sqlstore

import (
	"context"
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

	mock.ExpectQuery(be4Q("kurs_pada_tanggal")).WithArgs("USD", day).WillReturnRows(sqlmock.NewRows([]string{"k"}).AddRow("15000,5"))
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

	for line, group := range map[registrasi.LineOfBusiness]string{
		registrasi.LinePersonalAccident: "PA",
		registrasi.LineTravel:           "TRAVEL",
		registrasi.LineFire:             "NONMBU",
	} {
		mock.ExpectQuery(be4Q("pic_teknik_paling_ringan")).WithArgs(group).WillReturnRows(sqlmock.NewRows([]string{"o"}).AddRow(" TEK1 "))
		mock.ExpectExec(be4Q("pic_teknik_naikkan_beban")).WithArgs(" TEK1 ").WillReturnResult(sqlmock.NewResult(0, 1))
		got, err := a.Assign(ctx, technical, registrasi.Claim{Policy: registrasi.Policy{Line: line}}, "NIK1")
		require.NoError(t, err)
		require.Equal(t, registrasi.Assignee{Operator: "TEK1"}, got)
	}

	claim := registrasi.Claim{Policy: registrasi.Policy{Line: registrasi.LineFire}}
	// Tanpa petugas aktif: PNCTeknikRouter langkah 2 menugaskan ke ServicePNC.
	mock.ExpectQuery(be4Q("pic_teknik_paling_ringan")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	got, err := a.Assign(ctx, technical, claim, "NIK1")
	require.NoError(t, err)
	require.Equal(t, registrasi.Assignee{Operator: registrasi.OperatorUnassigned}, got)

	mock.ExpectQuery(be4Q("pic_teknik_paling_ringan")).WillReturnError(be4Boom)
	_, err = a.Assign(ctx, technical, claim, "NIK1")
	require.ErrorContains(t, err, "memilih petugas teknis")

	mock.ExpectQuery(be4Q("pic_teknik_paling_ringan")).WillReturnRows(sqlmock.NewRows([]string{"o"}).AddRow("TEK1"))
	mock.ExpectExec(be4Q("pic_teknik_naikkan_beban")).WillReturnError(be4Boom)
	_, err = a.Assign(ctx, technical, claim, "NIK1")
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
func TestPolicyItemsItems(t *testing.T) {
	db, mock := be4DB(t)
	p := NewPolicyItems(db)
	ctx := context.Background()

	got, err := p.Items(ctx, registrasi.Policy{Number: "POL"})
	require.NoError(t, err)
	require.Nil(t, got)

	cov := `{"CoverageList":[{"Coverage":"11022","CoverageNote":"ICC C","TSI":"100"}]}`
	mock.ExpectQuery(be4Q("polis_objek_cargo")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(" OBJ1 ", " Kargo ", " Laut ", cov))
	mock.ExpectQuery(be4Q("jenis_treaty_nama")).WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow("10001", "OR"))
	got, err = p.Items(ctx, registrasi.Policy{Number: " POL ", ProdKe: " 1 ", Line: registrasi.LineMarineCargo})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "OBJ1", got[0].ID)
	require.Equal(t, "Kargo", got[0].Name)
	require.Equal(t, "Laut", got[0].Location)
	require.Equal(t, "11022", got[0].Coverage[0].Code)

	// Dokumen coverage rusak menghentikan pembacaan.
	mock.ExpectQuery(be4Q("polis_objek_person")).WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow("O", "N", "L", "{rusak"))
	_, err = p.Items(ctx, registrasi.Policy{Number: "POL", ProdKe: "1", Line: registrasi.LinePersonalAccident})
	require.ErrorContains(t, err, "dokumen coverage objek O")

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

// Objek properti: coverage dibaca per INDEXOBJECT, kode ganda dibuang.
func TestPolicyItemsPropertyCoverages(t *testing.T) {
	db, mock := be4DB(t)
	p := NewPolicyItems(db)
	ctx := context.Background()
	fire := registrasi.Policy{Number: "POL", ProdKe: "1", Line: registrasi.LineFire}

	mock.ExpectQuery(be4Q("polis_objek_property")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(" IDX1 ", " OBJ1 ", " Gudang ", " Jakarta "))
	a := `{"CoverageList":[{"Coverage":"C1","TSI":"10"}]}`
	b := `{"CoverageList":[{"Coverage":"C1","TSI":"10"},{"Coverage":"C2","TSI":"20"}]}`
	mock.ExpectQuery(be4Q("polis_coverage_property")).WithArgs("POL", "1", "IDX1").
		WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow(a).AddRow(b))
	mock.ExpectQuery(be4Q("jenis_treaty_nama")).WillReturnError(be4Boom)
	_, err := p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca nama treaty")

	mock.ExpectQuery(be4Q("polis_objek_property")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(" IDX1 ", " OBJ1 ", " Gudang ", " Jakarta "))
	mock.ExpectQuery(be4Q("polis_coverage_property")).WithArgs("POL", "1", "IDX1").
		WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow(a).AddRow(b))
	mock.ExpectQuery(be4Q("jenis_treaty_nama")).WillReturnRows(sqlmock.NewRows(be4Cols(2)))
	got, err := p.Items(ctx, fire)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "OBJ1", got[0].ID)
	require.Len(t, got[0].Coverage, 2)
	require.Equal(t, "C2", got[0].Coverage[1].Code)

	object := func() {
		mock.ExpectQuery(be4Q("polis_objek_property")).WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow("IDX1", "OBJ1", "G", "J"))
	}
	object()
	mock.ExpectQuery(be4Q("polis_coverage_property")).WillReturnError(be4Boom)
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca coverage objek properti IDX1")
	object()
	mock.ExpectQuery(be4Q("polis_coverage_property")).WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow("1", "2"))
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "membaca dokumen coverage properti")
	object()
	mock.ExpectQuery(be4Q("polis_coverage_property")).WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow("{rusak"))
	_, err = p.Items(ctx, fire)
	require.ErrorContains(t, err, "dokumen coverage properti IDX1")
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
	doc := `{"PropertyItemList":[{"ItemType":" Bangunan ","PropertyItemGroup":"G1","TSIObjectItem":"1000.5"},
		{"ItemType":"Bangunan","PropertyItemGroup":"G1","TSIObjectItem":"1"},
		{"ItemType":"Hapus","FlagDelete":"1"},{"ItemType":""}]}`
	mock.ExpectQuery(be4Q("item_pilihan_properti")).WithArgs("POL", "1", "OBJ1").
		WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow(doc).AddRow(" "))
	got, err = p.ItemOptions(ctx, fire, " OBJ1 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.ItemOption{{Name: "Bangunan", Group: "G1", TSI: 100_050}}, got)

	mock.ExpectQuery(be4Q("item_pilihan_properti")).WillReturnError(be4Boom)
	_, err = p.ItemOptions(ctx, fire, "OBJ1")
	require.ErrorContains(t, err, "membaca item properti polis")
	mock.ExpectQuery(be4Q("item_pilihan_properti")).WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow("1", "2"))
	_, err = p.ItemOptions(ctx, fire, "OBJ1")
	require.ErrorContains(t, err, "membaca dokumen item properti")
	mock.ExpectQuery(be4Q("item_pilihan_properti")).WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow("{rusak"))
	_, err = p.ItemOptions(ctx, fire, "OBJ1")
	require.ErrorContains(t, err, "dokumen item properti")
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
