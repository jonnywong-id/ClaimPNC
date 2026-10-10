package sqlstore

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

func r3DLARow(rows *sqlmock.Rows, number string) *sqlmock.Rows {
	at := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	return rows.AddRow(" "+number+" ", " KOAS ", " 77 ", " COINS ", at, "catatan", " 100.00 ", " 30 ", " 30 ",
		" 1000 ", " IDR ", " IDR ", " QS ", " RI ", " A1 ", at, " e@x ", " L1 ", " ID ", "1", "0")
}

// DLA terbit dan Pre-DLA membaca 21 kolom yang sama, dengan kunci baris dari parameter.
func TestDLARows(t *testing.T) {
	db, mock := be4DB(t)
	s := NewDLAStore(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("dla_terbit")).WithArgs("K1", "O", "1", "2").WillReturnRows(r3DLARow(sqlmock.NewRows(be4Cols(21)), "D1"))
	got, err := s.Issued(ctx, "K1", "O", 1, 2)
	require.NoError(t, err)
	require.Len(t, got, 1)
	d := got[0]
	require.Equal(t, "D1", d.Number)
	require.Equal(t, "K1", d.ClaimID)
	require.Equal(t, 2, d.AdjustmentSeq)
	require.Equal(t, "KOAS", d.Recipient)
	require.Equal(t, "77", d.RecipientCode)
	require.Equal(t, "100.00", d.Value)
	require.Equal(t, "QS", d.QSPQS)
	require.Equal(t, "A1", d.AcceptedNo)
	require.Equal(t, registrasi.PLARecipientInfo{Email: "e@x", Login: "L1", Country: "ID"}, d.Info)
	require.True(t, d.Printed)
	require.False(t, d.Sent)
	require.Equal(t, time.Date(2026, 6, 9, 10, 0, 0, 0, clock.ZoneWIB), d.Date)

	mock.ExpectQuery(be4Q("dla_predla")).WithArgs("K1", "O", "1", "2").WillReturnRows(r3DLARow(sqlmock.NewRows(be4Cols(21)), "P1"))
	pre, err := s.PreDLA(ctx, "K1", "O", 1, 2)
	require.NoError(t, err)
	require.Equal(t, "P1", pre[0].Number)

	r3RowFailures(t, mock, "dla_terbit", 21, func() error {
		_, err := s.Issued(ctx, "K1", "O", 1, 2)
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDLAPrevious(t *testing.T) {
	db, mock := be4DB(t)
	s := NewDLAStore(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("dla_sebelumnya")).WithArgs("K1", "77").WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(" D0 ", at))
	prev, ok, err := s.Previous(ctx, "K1", "77")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "D0", prev.Number)

	mock.ExpectQuery(be4Q("dla_sebelumnya")).WillReturnRows(sqlmock.NewRows(be4Cols(2)))
	_, ok, err = s.Previous(ctx, "K1", "77")
	require.NoError(t, err)
	require.False(t, ok)

	mock.ExpectQuery(be4Q("dla_sebelumnya")).WillReturnError(be4Boom)
	_, _, err = s.Previous(ctx, "K1", "77")
	require.ErrorContains(t, err, "membaca DLA sebelumnya")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDLAReinsuranceCase(t *testing.T) {
	db, mock := be4DB(t)
	s := NewDLAStore(db)
	mock.ExpectQuery(be4Q("dla_jenis_reas")).WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(" 10001 ", " OR ").AddRow("10003", "QS"))
	got, err := s.ReinsuranceCase(context.Background())
	require.NoError(t, err)
	require.Equal(t, map[string]string{"10001": "OR", "10003": "QS"}, got)
	r3RowFailures(t, mock, "dla_jenis_reas", 2, func() error {
		_, err := s.ReinsuranceCase(context.Background())
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

// Treaty QS membaca grup, reasuradur, limit, lalu bagian QS: persentase baris terakhir dan
// seluruh bagian beserta namanya.
func TestDLATreatyQS(t *testing.T) {
	db, mock := be4DB(t)
	s := NewDLAStore(db)
	mock.ExpectQuery(be4Q("dla_treaty_grup")).WithArgs("10140", "2026", "10003").WillReturnRows(sqlmock.NewRows([]string{"g"}).AddRow(" G1 "))
	mock.ExpectQuery(be4Q("dla_treaty_reas")).WithArgs("2026", "G1", "10003").
		WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(" 40 ", " RE A ", " 9 ", " QS "))
	mock.ExpectQuery(be4Q("dla_treaty_limit")).WithArgs("10003", "2026", "G1").WillReturnRows(sqlmock.NewRows([]string{"l"}).AddRow(" 1000 "))
	mock.ExpectQuery(be4Q("dla_treaty_qs")).WithArgs("2026", "10003", "G1").
		WillReturnRows(sqlmock.NewRows([]string{"p", "n"}).AddRow("10", " ASM ").AddRow(" 20 ", "RE A"))
	got, err := s.Treaty(context.Background(), "10140", 2026, "10003")
	require.NoError(t, err)
	require.Equal(t, registrasi.TreatyArrangement{
		Reinsurers: []registrasi.TreatyReinsurer{{PctShare: "40", Name: "RE A", ID: "9", TypeName: "QS"}},
		Limit:      "1000", QSPct: "20",
		QSParts: []registrasi.TreatyQSPart{{Name: "ASM", Pct: "10"}, {Name: "RE A", Pct: "20"}},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Treaty non-QS tanpa grup dan limit tetap berjalan; QS tidak dibaca.
func TestDLATreatyWithoutGroup(t *testing.T) {
	db, mock := be4DB(t)
	s := NewDLAStore(db)
	mock.ExpectQuery(be4Q("dla_treaty_grup")).WillReturnRows(sqlmock.NewRows([]string{"g"}))
	mock.ExpectQuery(be4Q("dla_treaty_reas")).WithArgs("2026", "", "10001").WillReturnRows(sqlmock.NewRows(be4Cols(4)))
	mock.ExpectQuery(be4Q("dla_treaty_limit")).WillReturnRows(sqlmock.NewRows([]string{"l"}))
	got, err := s.Treaty(context.Background(), "10140", 2026, "10001")
	require.NoError(t, err)
	require.Equal(t, registrasi.TreatyArrangement{}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDLATreatyFailures(t *testing.T) {
	ctx := context.Background()
	group := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(be4Q("dla_treaty_grup")).WillReturnRows(sqlmock.NewRows([]string{"g"}).AddRow("G"))
	}
	reas := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(be4Q("dla_treaty_reas")).WillReturnRows(sqlmock.NewRows(be4Cols(4)))
	}
	limit := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(be4Q("dla_treaty_limit")).WillReturnRows(sqlmock.NewRows([]string{"l"}).AddRow("1"))
	}
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  string
	}{
		{"grup", func(m sqlmock.Sqlmock) { m.ExpectQuery(be4Q("dla_treaty_grup")).WillReturnError(be4Boom) }, "membaca grup treaty"},
		{"reas", func(m sqlmock.Sqlmock) {
			group(m)
			m.ExpectQuery(be4Q("dla_treaty_reas")).WillReturnError(be4Boom)
		}, "membaca reasuradur treaty"},
		{"reas scan", func(m sqlmock.Sqlmock) {
			group(m)
			m.ExpectQuery(be4Q("dla_treaty_reas")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))
		}, "membaca baris reasuradur treaty"},
		{"reas baris", func(m sqlmock.Sqlmock) {
			group(m)
			m.ExpectQuery(be4Q("dla_treaty_reas")).WillReturnRows(sqlmock.NewRows(be4Cols(4)).AddRow(nil, nil, nil, nil).RowError(0, be4Boom))
		}, be4Boom.Error()},
		{"limit", func(m sqlmock.Sqlmock) {
			group(m)
			reas(m)
			m.ExpectQuery(be4Q("dla_treaty_limit")).WillReturnError(be4Boom)
		}, "membaca limit treaty"},
		{"qs", func(m sqlmock.Sqlmock) {
			group(m)
			reas(m)
			limit(m)
			m.ExpectQuery(be4Q("dla_treaty_qs")).WillReturnError(be4Boom)
		}, "membaca QS treaty"},
		{"qs scan", func(m sqlmock.Sqlmock) {
			group(m)
			reas(m)
			limit(m)
			m.ExpectQuery(be4Q("dla_treaty_qs")).WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c"}).AddRow("1", "2", "3"))
		}, "sql: expected 3 destination arguments"},
		{"qs baris", func(m sqlmock.Sqlmock) {
			group(m)
			reas(m)
			limit(m)
			m.ExpectQuery(be4Q("dla_treaty_qs")).WillReturnRows(sqlmock.NewRows([]string{"p", "n"}).AddRow("1", "A").RowError(0, be4Boom))
		}, be4Boom.Error()},
	}
	for _, c := range cases {
		db, mock := be4DB(t)
		c.setup(mock)
		_, err := NewDLAStore(db).Treaty(ctx, "10140", 2026, "10019")
		require.ErrorContains(t, err, c.want, c.name)
		require.NoError(t, mock.ExpectationsWereMet(), c.name)
	}
}

func TestDLANextNumber(t *testing.T) {
	db, mock := be4DB(t)
	_, err := NewDLAStore(db).NextNumber(context.Background(), "DLACOINS", 2026)
	require.ErrorContains(t, err, "nomor DLA hanya boleh diterbitkan di dalam transaksi")

	ctx, _ := r3Tx(t, db, mock)
	mock.ExpectQuery(be4Q("pla_site")).WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow(" 1 "))
	mock.ExpectQuery(be4Q("dla_urut")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(12)))
	mock.ExpectExec(be4Q("dla_nomor_sisip")).WithArgs(sqlmock.AnyArg(), "J", "1", "26", "12").WillReturnResult(sqlmock.NewResult(0, 1))
	number, err := NewDLAStore(db).NextNumber(ctx, "J", 2026)
	require.NoError(t, err)
	require.Equal(t, "J261000000000000012", number)
	require.NoError(t, mock.ExpectationsWereMet())

	for step, want := range map[int]string{0: "membaca site aktif", 1: "membaca DLA_SEQ", 2: "mencatat nomor DLA"} {
		db, mock := be4DB(t)
		ctx, _ := r3Tx(t, db, mock)
		r3NumberSteps(mock, "pla_site", "dla_urut", "dla_nomor_sisip", step)
		_, err := NewDLAStore(db).NextNumber(ctx, "J", 2026)
		require.ErrorContains(t, err, want)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

// r3NumberSteps menyiapkan tiga langkah penerbitan nomor dengan langkah ke-failAt gagal.
func r3NumberSteps(mock sqlmock.Sqlmock, site, seq, insert string, failAt int) {
	q := mock.ExpectQuery(be4Q(site))
	if failAt == 0 {
		q.WillReturnError(be4Boom)
		return
	}
	q.WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow("1"))
	q = mock.ExpectQuery(be4Q(seq))
	if failAt == 1 {
		q.WillReturnError(be4Boom)
		return
	}
	q.WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(1)))
	mock.ExpectExec(be4Q(insert)).WillReturnError(be4Boom)
}

func TestDLASaveAndMarkPrinted(t *testing.T) {
	db, mock := be4DB(t)
	s := NewDLAStore(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 3, 0, 0, 0, time.UTC)
	d := registrasi.DLA{
		ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, AdjustmentSeq: 2, Number: "D1", Recipient: "KOAS",
		Type: "COINS", Date: at, Value: "100", RecipientCode: "77", Currency: "IDR", PolicyCurrency: "IDR",
		Percent: "30", ClaimAmount: "1000",
	}
	mock.ExpectExec(be4Q("dla_sisip")).
		WithArgs("K1", "O", "1", "2", "D1", "KOAS", nil, "COINS", time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
			"100", nil, "77", "IDR", "IDR", nil, "30", "1000", nil, nil, nil, nil, nil, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Save(ctx, d))

	d.AcceptedDate = at
	mock.ExpectExec(be4Q("dla_sisip")).
		WithArgs(append(be4AnyArgs(22, nil), driver.Value(time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)))...).
		WillReturnError(be4Boom)
	require.ErrorContains(t, s.Save(ctx, d), "menyimpan DLA D1")

	mock.ExpectExec(be4Q("dla_cetak")).WithArgs("catatan", "K1", "D1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.MarkPrinted(ctx, "K1", "D1", "catatan"))
	mock.ExpectExec(be4Q("dla_cetak")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.MarkPrinted(ctx, "K1", "D1", "catatan"), "menandai DLA D1 tercetak")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Polis DLA: dokumen POLICYDATA sebagai dasar, lalu kolom T_GENERAL, T_OFFERFACIN,
// T_SPREADINGLIST, T_COINSLIST, dan T_FACOFFER menimpanya bila terisi.
func TestDLAPolicyDocument(t *testing.T) {
	db, mock := be4DB(t)
	s := NewDLAStore(db)
	ctx := context.Background()
	rest := func() {
		mock.ExpectQuery(be4Q("pla_koasuransi")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows(be4Cols(5)))
		mock.ExpectQuery(be4Q("dla_fac_offer")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows(be4Cols(2)))
	}

	// Dokumen saja: tabel kosong.
	mock.ExpectQuery(be4Q("dla_polis_dokumen")).WithArgs("POL").
		WillReturnRows(sqlmock.NewRows(be4Cols(1)).AddRow(`{"CaseID":"C1","TypeOfCoins":"2"}`))
	mock.ExpectQuery(be4Q("dla_polis_kepala")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows(be4Cols(7)))
	mock.ExpectQuery(be4Q("dla_offer_facin")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows(be4Cols(1)))
	mock.ExpectQuery(be4Q("dla_spreading")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows(be4Cols(3)))
	rest()
	p, err := s.Policy(ctx, " POL ", " 1 ")
	require.NoError(t, err)
	require.Equal(t, "C1", p.CaseID)
	require.Equal(t, "2", p.TypeOfCoins)

	// Tabel menimpa dokumen.
	mock.ExpectQuery(be4Q("dla_polis_dokumen")).WithArgs("POL").WillReturnRows(sqlmock.NewRows(be4Cols(1)))
	mock.ExpectQuery(be4Q("dla_polis_kepala")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(7)).AddRow(" C2 ", "1000", "1", "1", "006", "A", nil))
	mock.ExpectQuery(be4Q("dla_offer_facin")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(1)).AddRow("25"))
	mock.ExpectQuery(be4Q("dla_spreading")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow("10001", "500", "0").AddRow("10002", "9", "1"))
	rest()
	p, err = s.Policy(ctx, "POL", "1")
	require.NoError(t, err)
	require.Equal(t, "C2", p.CaseID)
	require.Equal(t, "1", p.TypeOfCoins)
	require.True(t, p.Syariah)
	require.Equal(t, "006", p.BusinessCode)
	require.Equal(t, "A", p.StatusBusiness)
	require.Contains(t, p.TSISpreaded, "10001")
	require.NotContains(t, p.TSISpreaded, "10002")

	mock.ExpectQuery(be4Q("dla_polis_dokumen")).WillReturnError(be4Boom)
	_, err = s.Policy(ctx, "POL", "1")
	require.ErrorContains(t, err, `membaca dokumen polis "POL"`)

	mock.ExpectQuery(be4Q("dla_polis_dokumen")).WillReturnRows(sqlmock.NewRows(be4Cols(1)))
	mock.ExpectQuery(be4Q("dla_polis_kepala")).WillReturnError(be4Boom)
	_, err = s.Policy(ctx, "POL", "1")
	require.ErrorContains(t, err, "membaca T_GENERAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWallDateHelpers(t *testing.T) {
	require.Nil(t, nullWallDate(time.Time{}))
	at := time.Date(2026, 6, 9, 20, 0, 0, 5, time.UTC)
	require.Equal(t, time.Date(2026, 6, 10, 3, 0, 0, 0, time.UTC), wallDate(at))
	require.Equal(t, time.Date(2026, 6, 10, 3, 0, 0, 0, time.UTC), nullWallDate(at))
	require.True(t, wallWIB(time.Time{}).IsZero())
	require.Equal(t, time.Date(2026, 6, 9, 20, 0, 0, 0, clock.ZoneWIB), wallWIB(at))
}

func TestPLACoinsMembers(t *testing.T) {
	db, mock := be4DB(t)
	s := NewPLAStore(db)
	mock.ExpectQuery(be4Q("pla_koasuransi")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(5)).AddRow(" C0 ", " ASM ", "true", "70", "0").AddRow("C1", "B", "false", "", "1"))
	got, err := s.CoinsMembers(context.Background(), " POL ", " 1 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.PLACoinsMember{
		{ID: "C0", Name: "ASM", Leader: true, Share: 700_000, HasShare: true},
		{ID: "C1", Name: "B", Deleted: true},
	}, got)
	r3RowFailures(t, mock, "pla_koasuransi", 5, func() error {
		_, err := s.CoinsMembers(context.Background(), "POL", "1")
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPLARecipientAndPrevious(t *testing.T) {
	db, mock := be4DB(t)
	s := NewPLAStore(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("pla_penerima")).WithArgs("C1", "B").WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow(" L ", " ID ", " e@x "))
	info, err := s.Recipient(ctx, "C1", "B")
	require.NoError(t, err)
	require.Equal(t, registrasi.PLARecipientInfo{Login: "L", Country: "ID", Email: "e@x"}, info)
	mock.ExpectQuery(be4Q("pla_penerima")).WillReturnRows(sqlmock.NewRows(be4Cols(3)))
	info, err = s.Recipient(ctx, "C1", "B")
	require.NoError(t, err)
	require.Equal(t, registrasi.PLARecipientInfo{}, info)
	mock.ExpectQuery(be4Q("pla_penerima")).WillReturnError(be4Boom)
	_, err = s.Recipient(ctx, "C1", "B")
	require.ErrorContains(t, err, `membaca penerima PLA "C1"`)

	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("pla_sebelumnya")).WithArgs("K1", "C1").WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(" P0 ", at))
	prev, ok, err := s.Previous(ctx, "K1", "C1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "P0", prev.Number)
	mock.ExpectQuery(be4Q("pla_sebelumnya")).WillReturnRows(sqlmock.NewRows(be4Cols(2)))
	_, ok, err = s.Previous(ctx, "K1", "C1")
	require.NoError(t, err)
	require.False(t, ok)
	mock.ExpectQuery(be4Q("pla_sebelumnya")).WillReturnError(be4Boom)
	_, _, err = s.Previous(ctx, "K1", "C1")
	require.ErrorContains(t, err, "membaca PLA sebelumnya")
	require.NoError(t, mock.ExpectationsWereMet())
}

// PLA disimpan sebagai JSON EstimasiList, lalu terbaca kembali sebagai jumlah yang sama.
func TestPLASaveAndIssuedRoundTrip(t *testing.T) {
	db, mock := be4DB(t)
	s := NewPLAStore(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 3, 0, 0, 0, time.UTC)
	p := registrasi.PLA{
		ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Revision: 0, Number: "P1", Recipient: "B",
		Type: registrasi.PLATypeCoins, Date: at, Note: "n", RecipientCode: "C1", PolicyCurrency: "IDR",
		Amount: []registrasi.PLAAmount{{Currency: "IDR", CurrencyID: "1", Share: 200_000, Reserve: 5_000_000_00,
			Base: 5_000_000_00, Result: 1_000_000_00, ASMCount: -150}},
	}
	var body string
	mock.ExpectExec(be4Q("pla_sisip")).
		WithArgs("K1", "O", "1", "P1", "B", "0", registrasi.PLATypeCoins, time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
			"n", "C1", "IDR", nil, nil, nil, r3Capture(&body)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Save(ctx, p))
	var doc plaJSON
	require.NoError(t, json.Unmarshal([]byte(body), &doc))
	require.Equal(t, "ASM-FW-GCNMFW-Data-PLA", doc.ObjClass)
	require.Equal(t, "20.00", doc.EstimasiList[0].PercentPLA)
	require.Equal(t, "-1.50", doc.EstimasiList[0].ASMCount)
	require.Equal(t, "1000000.00", doc.EstimasiList[0].ResultPLA)

	mock.ExpectQuery(be4Q("pla_terbit")).WithArgs("K1", "O", "1", "0", registrasi.PLATypeCoins, registrasi.PLATypeFacOut,
		registrasi.PLATypeBPPDAN, registrasi.PLATypeEQPool).
		WillReturnRows(sqlmock.NewRows(be4Cols(10)).
			AddRow(" P1 ", " B ", " C1 ", at, "n", " IDR ", body, " e@x ", "COINS", "1").
			AddRow("P2", "C", "C2", at, "", "IDR", nil, nil, "COINS", "0"))
	got, err := s.Issued(ctx, "K1", "O", 1, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "P1", got[0].Number)
	require.Equal(t, "e@x", got[0].Info.Email)
	require.True(t, got[0].Sent)
	require.False(t, got[1].Sent)
	require.Len(t, got[0].Amount, 1)
	require.Equal(t, p.Amount[0].Result, got[0].Amount[0].Result)
	require.Equal(t, p.Amount[0].Share, got[0].Amount[0].Share)
	require.Empty(t, got[1].Amount)

	r3RowFailures(t, mock, "pla_terbit", 10, func() error {
		_, err := s.Issued(ctx, "K1", "O", 1, 0)
		return err
	})

	mock.ExpectExec(be4Q("pla_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.Save(ctx, p), "menyimpan PLA P1")
	require.NoError(t, mock.ExpectationsWereMet())
}

// r3Capture mencocokkan argumen apa pun dan menyimpannya sebagai teks.
type r3Captor struct{ into *string }

func (c r3Captor) Match(v driver.Value) bool {
	s, ok := v.(string)
	if ok {
		*c.into = s
	}
	return ok
}

func r3Capture(into *string) sqlmock.Argument { return r3Captor{into: into} }

func TestPLANextNumber(t *testing.T) {
	db, mock := be4DB(t)
	_, err := NewPLAStore(db).NextNumber(context.Background(), "J", 2026)
	require.ErrorContains(t, err, "nomor PLA hanya boleh diterbitkan di dalam transaksi")

	ctx, _ := r3Tx(t, db, mock)
	mock.ExpectQuery(be4Q("pla_site")).WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow("1"))
	mock.ExpectQuery(be4Q("pla_urut")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(3)))
	mock.ExpectExec(be4Q("pla_nomor_sisip")).WithArgs(sqlmock.AnyArg(), "J", "1", "26", "3").WillReturnResult(sqlmock.NewResult(0, 1))
	number, err := NewPLAStore(db).NextNumber(ctx, "J", 2026)
	require.NoError(t, err)
	require.Equal(t, "J261000000000000003", number)
	require.NoError(t, mock.ExpectationsWereMet())

	for step, want := range map[int]string{0: "membaca site aktif", 1: "membaca PLA_SEQ", 2: "mencatat nomor PLA"} {
		db, mock := be4DB(t)
		ctx, _ := r3Tx(t, db, mock)
		r3NumberSteps(mock, "pla_site", "pla_urut", "pla_nomor_sisip", step)
		_, err := NewPLAStore(db).NextNumber(ctx, "J", 2026)
		require.ErrorContains(t, err, want)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestPLAUpdateNoteSignatureAndEmails(t *testing.T) {
	db, mock := be4DB(t)
	s := NewPLAStore(db)
	ctx := context.Background()

	mock.ExpectExec(be4Q("pla_catatan")).WithArgs("baru", "K1", "P1", "2").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.UpdateNote(ctx, "K1", "P1", 2, "baru"))
	mock.ExpectExec(be4Q("pla_catatan")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.UpdateNote(ctx, "K1", "P1", 2, "baru"), "menyimpan catatan PLA P1")

	mock.ExpectExec(be4Q("pla_email")).WithArgs("a@contoh.co.id", "K1", "P1", "2").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.UpdateEmail(ctx, "K1", "P1", 2, "a@contoh.co.id"))
	mock.ExpectExec(be4Q("pla_email")).WithArgs(nil, "K1", "P1", "2").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.UpdateEmail(ctx, "K1", "P1", 2, ""))
	mock.ExpectExec(be4Q("pla_email")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.UpdateEmail(ctx, "K1", "P1", 2, "x"), "menyimpan email PLA P1")

	png := []byte{0x89, 'P', 'N', 'G'}
	mock.ExpectQuery(be4Q("pla_ttd")).WithArgs("T1").
		WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(" BUDI ", `{"TTDWeb":"`+base64.StdEncoding.EncodeToString(png)+`"}`))
	name, img, err := s.Signature(ctx, "T1")
	require.NoError(t, err)
	require.Equal(t, "BUDI", name)
	require.Equal(t, png, img)
	mock.ExpectQuery(be4Q("pla_ttd")).WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow("A", "bukan json"))
	name, img, err = s.Signature(ctx, "T1")
	require.NoError(t, err)
	require.Equal(t, "A", name)
	require.Nil(t, img)
	mock.ExpectQuery(be4Q("pla_ttd")).WillReturnRows(sqlmock.NewRows(be4Cols(2)))
	name, img, err = s.Signature(ctx, "T1")
	require.NoError(t, err)
	require.Empty(t, name)
	require.Nil(t, img)
	mock.ExpectQuery(be4Q("pla_ttd")).WillReturnError(be4Boom)
	_, _, err = s.Signature(ctx, "T1")
	require.ErrorContains(t, err, "membaca tanda tangan")

	mock.ExpectQuery(be4Q("lod_email_tertanggung")).WithArgs("PNCN.26.1").WillReturnRows(sqlmock.NewRows([]string{"e"}).AddRow(" t@x "))
	mock.ExpectQuery(be4Q("lod_email_pic")).WithArgs("PIC1").WillReturnRows(sqlmock.NewRows([]string{"e"}))
	insured, pic, err := s.LODEmails(ctx, "PNCN.26.1", "PIC1")
	require.NoError(t, err)
	require.Equal(t, "t@x", insured)
	require.Empty(t, pic)

	mock.ExpectQuery(be4Q("lod_email_tertanggung")).WillReturnRows(sqlmock.NewRows([]string{"e"}).AddRow("t@x"))
	insured, pic, err = s.LODEmails(ctx, "PNCN.26.1", " ")
	require.NoError(t, err)
	require.Equal(t, "t@x", insured)
	require.Empty(t, pic)

	mock.ExpectQuery(be4Q("lod_email_tertanggung")).WillReturnError(be4Boom)
	_, _, err = s.LODEmails(ctx, "PNCN.26.1", "PIC1")
	require.ErrorContains(t, err, "lod_email_tertanggung")
	mock.ExpectQuery(be4Q("lod_email_tertanggung")).WillReturnRows(sqlmock.NewRows([]string{"e"}))
	mock.ExpectQuery(be4Q("lod_email_pic")).WillReturnError(be4Boom)
	_, _, err = s.LODEmails(ctx, "PNCN.26.1", "PIC1")
	require.ErrorContains(t, err, "lod_email_pic")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMoneyAndPercentText(t *testing.T) {
	require.Equal(t, "0.00", moneyText(0))
	require.Equal(t, "12.05", moneyText(1205))
	require.Equal(t, "-0.05", moneyText(-5))
	require.Equal(t, "33.33", percentText(333_333))
	require.Equal(t, "100.00", percentText(registrasi.PercentFull))
}

// PLA FAC OUT menyimpan SharePLA dan PercentPLA sebagai NILAI uang (PLAHTML_FACOUT), lalu
// terbaca kembali sebagai FacShare dan FacBase.
func TestPLAFacOutSaveAndIssuedRoundTrip(t *testing.T) {
	db, mock := be4DB(t)
	s := NewPLAStore(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 3, 0, 0, 0, time.UTC)
	p := registrasi.PLA{
		ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Number: "H1", Recipient: "REAS", RecipientCode: "R1",
		Type: registrasi.PLATypeFacOut, Date: at, PolicyCurrency: "IDR",
		Amount: []registrasi.PLAAmount{{Currency: "IDR", Reserve: 25_505_000_00, Base: 12_752_500_00,
			Result: 2_550_500_00, FacShare: 314_000_00, FacBase: 1_570_000_00}},
	}
	var body string
	mock.ExpectExec(be4Q("pla_sisip")).WithArgs(
		"K1", "O", "1", "H1", "REAS", "0", registrasi.PLATypeFacOut, sqlmock.AnyArg(),
		"", "R1", "IDR", nil, nil, nil, r3Capture(&body)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Save(ctx, p))
	var doc plaJSON
	require.NoError(t, json.Unmarshal([]byte(body), &doc))
	require.Equal(t, "314000.00", doc.EstimasiList[0].SharePLA)
	require.Equal(t, "1570000.00", doc.EstimasiList[0].PercentPLA)
	require.Equal(t, "2550500.00", doc.EstimasiList[0].ResultPLA)

	mock.ExpectQuery(be4Q("pla_terbit")).
		WillReturnRows(sqlmock.NewRows(be4Cols(10)).AddRow("H1", "REAS", "R1", at, "", "IDR", body, nil, "FACOUT", "0"))
	got, err := s.Issued(ctx, "K1", "O", 1, 0)
	require.NoError(t, err)
	require.Equal(t, registrasi.PLATypeFacOut, got[0].Type)
	require.Equal(t, p.Amount[0].FacShare, got[0].Amount[0].FacShare)
	require.Equal(t, p.Amount[0].FacBase, got[0].Amount[0].FacBase)
	require.NoError(t, mock.ExpectationsWereMet())
}

// FacOffers PLA: baris ber-JSONDATA diurai; baris tanpa JSONDATA memakai kolom datar; baris
// tanpa keduanya dilewati.
func TestPLAFacOffersJSONAndFlatFallback(t *testing.T) {
	db, mock := be4DB(t)
	s := NewPLAStore(db)
	body := `{"FacOfferList":[{"ReinsurerID":"R1","ReinsurerName":"REAS JSON","PropertyList":[{"ObjectNo":"1","CoverageList":[{"Coverage":"C","TSISublimit":"100","FacOutObjectList":[{"ShareOffered":"10"}],"SpreadingList":[{"TreatyType":"10015","TSISpreaded":"12"}]}]}]}]}`
	mock.ExpectQuery(be4Q("pla_fac_offer")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(4)).
			AddRow("R1", "REAS JSON", "5", body).
			AddRow("R2", " AON ", "3", nil).
			AddRow("R3", "TANPA DATA", nil, nil))
	got, err := s.FacOffers(context.Background(), " POL ", "1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "REAS JSON", got[0].ReinsurerName)
	require.Empty(t, got[0].FlatShare)
	require.Equal(t, "12", got[0].Property[0].Coverage[0].SpreadFacOut)
	require.Equal(t, registrasi.FacOffer{ReinsurerID: "R2", ReinsurerName: "AON", FlatShare: "3"}, got[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

// SpreadingTSI menjumlahkan TSISPREADED baris FAC OUT dan seluruh baris objek-coverage.
func TestPLASpreadingTSI(t *testing.T) {
	db, mock := be4DB(t)
	s := NewPLAStore(db)
	mock.ExpectQuery(be4Q("pla_spreading_tsi")).
		WithArgs("POL", "1", "100829", "006", "40", "006", "POL", "1", "40").
		WillReturnRows(sqlmock.NewRows(be4Cols(2)).
			AddRow("10001", "1782838988.8625").
			AddRow("10015", "1782838988.8625"))
	facOut, total, err := s.SpreadingTSI(context.Background(), registrasi.SpreadingTSIQuery{
		PolicyNumber: " POL ", ProdKe: "1", GroupPanel: "006", ObjectID: "40", Coverage: "100829"})
	require.NoError(t, err)
	require.Equal(t, "1782838988.8625", facOut.FloatString(4))
	require.Equal(t, "3565677977.7250", total.FloatString(4))
	require.NoError(t, mock.ExpectationsWereMet())
}
