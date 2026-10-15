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

// r3RowFailures menguji tiga cabang galat sebuah pembaca baris: kueri gagal, Scan gagal,
// dan galat penelusuran baris. read dipanggil sekali per cabang dan galatnya dikembalikan.
func r3RowFailures(t *testing.T, mock sqlmock.Sqlmock, name string, cols int, read func() error) {
	t.Helper()
	be4Expect(mock, be4Step{name: name, cols: cols}, be4Fail)
	require.ErrorIs(t, read(), be4Boom, "kueri gagal")
	// Satu kolom lebih banyak dari tujuan Scan membuat Scan pasti gagal.
	extra := make([]driver.Value, cols+1)
	mock.ExpectQuery(be4Q(name)).WillReturnRows(sqlmock.NewRows(be4Cols(cols + 1)).AddRow(extra...))
	require.ErrorContains(t, read(), "registrasi/sqlstore", "scan gagal")
	be4Expect(mock, be4Step{name: name, cols: cols}, be4RowErr)
	require.ErrorIs(t, read(), be4Boom, "galat baris")
}

// Wilayah: tingkat asing ditolak, induk kosong tanpa kueri, nama kosong dilewati.
func TestAreaDirectoryOptions(t *testing.T) {
	db, mock := be4DB(t)
	d := NewAreaDirectory(db)
	ctx := context.Background()

	_, err := d.Options(ctx, registrasi.AreaLevel("planet"), "")
	require.ErrorIs(t, err, registrasi.ErrUnknownAreaLevel)

	got, err := d.Options(ctx, registrasi.AreaProvince, "  ")
	require.NoError(t, err)
	require.Empty(t, got)
	require.NotNil(t, got)

	mock.ExpectQuery(be4Q("wilayah_negara")).WithArgs().
		WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow(" 1 ", " INDONESIA ", nil).AddRow("2", " ", nil))
	got, err = d.Options(ctx, registrasi.AreaCountry, "abaikan")
	require.NoError(t, err)
	require.Equal(t, []registrasi.AreaOption{{ID: "1", Name: "INDONESIA"}}, got)

	mock.ExpectQuery(be4Q("wilayah_kelurahan")).WithArgs("10000925").
		WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow("9", "KEL. X", " 55281 "))
	got, err = d.Options(ctx, registrasi.AreaVillage, " 10000925 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.AreaOption{{ID: "9", Name: "KEL. X", PostalCode: "55281"}}, got)

	r3RowFailures(t, mock, "wilayah_kota", 3, func() error {
		_, err := d.Options(ctx, registrasi.AreaCity, "10012")
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

// Lampiran: tahun dua digit dari tanggal WIB, kolom kosong menjadi NULL.
func TestAttachmentStoreAdd(t *testing.T) {
	db, mock := be4DB(t)
	at := time.Date(2026, 12, 31, 18, 0, 0, 0, time.UTC) // 1 Jan 2027 WIB
	// INPUTDATE: jam dinding WIB (SYSDATE server +07:00), dikirim sebagai waktu tanpa zona.
	wall := time.Date(2027, 1, 1, 1, 0, 0, 0, time.UTC)
	mock.ExpectExec(be4Q("lampiran_sisip")).
		WithArgs("27", "KEY1", nil, "lod.pdf", nil, "pdf", "IMG-1", "10064", "14901", wall).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewAttachmentStore(db).AddAttachment(context.Background(), registrasi.NewAttachment{
		ClaimKey: "KEY1", Name: "lod.pdf", Extension: "pdf", ImageID: "IMG-1", Category: "10064", SubCategory: "14901", At: at,
	}))

	mock.ExpectExec(be4Q("lampiran_sisip")).WillReturnError(be4Boom)
	err := NewAttachmentStore(db).AddAttachment(context.Background(), registrasi.NewAttachment{ClaimKey: "KEY1", At: at})
	require.ErrorIs(t, err, be4Boom)
	require.ErrorContains(t, err, "menyimpan lampiran klaim KEY1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountDirectoryFind(t *testing.T) {
	db, mock := be4DB(t)
	d := NewAccountDirectory(db)
	ctx := context.Background()
	approved := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("rekening_ambil")).WithArgs("123").
		WillReturnRows(sqlmock.NewRows(be4Cols(11)).
			AddRow(" 123 ", " PT A ", " BANK ", " JKT ", " JL ", " 001 ", " a@b ", " 08 ", nil, approved, " 1 "))
	got, err := d.FindAccount(ctx, " 123 ")
	require.NoError(t, err)
	require.Equal(t, registrasi.BankAccount{
		Number: "123", Name: "PT A", BankName: "BANK", Branch: "JKT", Address: "JL", BankID: "001",
		Email: "a@b", Telephone: "08", CommitteeApprovedAt: approved, Approval: "1",
	}, got)

	mock.ExpectQuery(be4Q("rekening_ambil")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	_, err = d.FindAccount(ctx, "x")
	require.ErrorIs(t, err, registrasi.ErrAccountNotFound)

	mock.ExpectQuery(be4Q("rekening_ambil")).WillReturnError(be4Boom)
	_, err = d.FindAccount(ctx, "x")
	require.ErrorContains(t, err, "membaca master rekening")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kode bank: hanya baris yang sama dengan id bank master yang diterima.
func TestCashierBankGroupID(t *testing.T) {
	db, mock := be4DB(t)
	s := NewCashierStore(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("kasir_kode_bank")).WithArgs("BANK A").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("002").AddRow(" 001 "))
	id, ok, err := s.BankGroupID(ctx, " BANK A ", "001")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "001", id)

	mock.ExpectQuery(be4Q("kasir_kode_bank")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(""))
	id, ok, err = s.BankGroupID(ctx, "BANK A", "")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, id)

	mock.ExpectQuery(be4Q("kasir_kode_bank")).WillReturnError(be4Boom)
	_, _, err = s.BankGroupID(ctx, "BANK A", "001")
	require.ErrorContains(t, err, "membaca kode bank")
	mock.ExpectQuery(be4Q("kasir_kode_bank")).WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow("1", "2"))
	_, _, err = s.BankGroupID(ctx, "BANK A", "001")
	require.ErrorContains(t, err, "membaca baris kode bank")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCashierLogAndMark(t *testing.T) {
	db, mock := be4DB(t)
	s := NewCashierStore(db)
	ctx := context.Background()

	mock.ExpectExec(be4Q("kasir_log")).WithArgs("A1", "PNCN.26.1", nil, "1", "Sedang").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Log(ctx, registrasi.CashierLog{AcceptedNo: "A1", ClaimNumber: "PNCN.26.1", Status: "1", Reason: "Sedang"}))
	mock.ExpectExec(be4Q("kasir_log")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.Log(ctx, registrasi.CashierLog{}), "menulis log kasir")

	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(be4Q("kasir_tandai")).WithArgs(sqlmock.AnyArg(), "ECR-1", "K1", "O", "1", "2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.MarkTransferred(ctx, "K1", "O", 1, 2, at, "ECR-1"))

	mock.ExpectExec(be4Q("kasir_tandai")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorContains(t, s.MarkTransferred(ctx, "K1", "O", 1, 2, at, "ECR-1"), "baris adjustment O/1/2 tidak ditemukan")

	mock.ExpectExec(be4Q("kasir_tandai")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.MarkTransferred(ctx, "K1", "O", 1, 2, at, "ECR-1"), "menandai transfer kasir")

	// RowsAffected yang gagal tidak dianggap baris hilang.
	mock.ExpectExec(be4Q("kasir_tandai")).WillReturnResult(sqlmock.NewErrorResult(be4Boom))
	require.NoError(t, s.MarkTransferred(ctx, "K1", "O", 1, 2, at, "ECR-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupStoreGroupsOf(t *testing.T) {
	db, mock := be4DB(t)
	g := NewGroupStore(db)
	mock.ExpectQuery(be4Q("grup_login")).WithArgs("NIK1").
		WillReturnRows(sqlmock.NewRows([]string{"g"}).AddRow(" PNCADMIN ").AddRow(" ").AddRow(nil).AddRow("KOMITE"))
	got, err := g.GroupsOf(context.Background(), "NIK1")
	require.NoError(t, err)
	require.Equal(t, []string{"PNCADMIN", "KOMITE"}, got)

	r3RowFailures(t, mock, "grup_login", 1, func() error {
		_, err := g.GroupsOf(context.Background(), "NIK1")
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurrencyDirectory(t *testing.T) {
	db, mock := be4DB(t)
	d := NewCurrencyDirectory(db)
	mock.ExpectQuery(be4Q("mata_uang_daftar")).
		WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(" IDR ", " Rupiah ").AddRow(" ", "Kosong"))
	got, err := d.Currencies(context.Background())
	require.NoError(t, err)
	require.Equal(t, []registrasi.CurrencyOption{{ID: "IDR", Name: "Rupiah"}}, got)

	r3RowFailures(t, mock, "mata_uang_daftar", 2, func() error {
		_, err := d.Currencies(context.Background())
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

// Daftar kerja: klaim tanpa nomor tidak ditulis; selebihnya perbarui lalu sisip.
func TestInboxEntryMirror(t *testing.T) {
	db, mock := be4DB(t)
	s := NewInboxEntryStore(db)
	ctx := context.Background()

	require.NoError(t, s.Mirror(ctx, registrasi.InboxEntry{}))

	claim := registrasi.Claim{ID: "K1", Number: "PNCN.26.1", Policy: registrasi.Policy{Number: "POL-1"}}
	entry := registrasi.NewInboxEntry(claim, nil, registrasi.RegisterFlow())
	mock.ExpectExec(be4Q("daftar_kerja_perbarui")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(be4Q("daftar_kerja_sisip")).
		WithArgs(append([]driver.Value{"PNCN.26.1", "PNCN.26.1"}, be4AnyArgs(26, nil)...)...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Mirror(ctx, entry))

	mock.ExpectExec(be4Q("daftar_kerja_perbarui")).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Mirror(ctx, entry))

	mock.ExpectExec(be4Q("daftar_kerja_perbarui")).WillReturnError(be4Boom)
	err := s.Mirror(ctx, entry)
	require.ErrorIs(t, err, be4Boom)
	require.ErrorContains(t, err, "menulis daftar kerja PNCN.26.1")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Nomor komite: pola tahun WIB, urutan dari basis data.
func TestCommitteeNextCaseID(t *testing.T) {
	db, mock := be4DB(t)
	s := NewCommitteeStore(db)
	at := time.Date(2026, 12, 31, 18, 0, 0, 0, time.UTC) // 2027 WIB
	mock.ExpectQuery(be4Q("komite_nomor_berikut")).WithArgs(registrasi.CommitteeCasePrefix + ".27.%").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(7)))
	id, err := s.NextCaseID(context.Background(), at)
	require.NoError(t, err)
	want, _ := registrasi.FormatCommitteeCaseID(2027, 7)
	require.Equal(t, want, id)

	mock.ExpectQuery(be4Q("komite_nomor_berikut")).WillReturnError(be4Boom)
	_, err = s.NextCaseID(context.Background(), at)
	require.ErrorContains(t, err, "menerbitkan nomor komite")
	require.NoError(t, mock.ExpectationsWereMet())
}

func r3CommitteeCase() registrasi.CommitteeCase {
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	return registrasi.CommitteeCase{
		ID: "KM.26.1", ClaimID: "K1", ClaimNumber: "PNCN.26.1", ObjectID: "O", CoverageSeq: 1, AdjustmentSeq: 2,
		PaymentType: "1", Currency: "IDR", Value: 900, CreatedAt: at,
		Members: []registrasi.CommitteeMember{
			{Operator: "KOMITE01", Level: 1, CaseStatus: "0", CreatedAt: at},
			{Operator: "KOMITE02", Level: 2, CaseStatus: "0", CreatedAt: at},
		},
	}
}

func TestCommitteeSave(t *testing.T) {
	db, mock := be4DB(t)
	s := NewCommitteeStore(db)
	c := r3CommitteeCase()
	mock.ExpectExec(be4Q("komite_kepala_perbarui")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(be4Q("komite_kepala_sisip")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("komite_perbarui")).
		WithArgs(append([]driver.Value{"PNCN.26.1"}, be4AnyArgs(12, map[int]driver.Value{9: "KM.26.1", 10: "KOMITE01", 11: "1"})...)...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("komite_perbarui")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(be4Q("komite_sisip")).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Save(context.Background(), c))

	mock.ExpectExec(be4Q("komite_kepala_perbarui")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.Save(context.Background(), c), "menyimpan kepala komite KM.26.1")

	mock.ExpectExec(be4Q("komite_kepala_perbarui")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("komite_perbarui")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.Save(context.Background(), c), "menyimpan komite KM.26.1 jenjang 1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func r3CommitteeMemberRow(rows *sqlmock.Rows, operator, level string) *sqlmock.Rows {
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	return rows.AddRow(" KM.26.1 ", " PNCN.26.1 ", " "+operator+" ", " "+level+" ", " 1 ", " 2 ", "catatan", " T ", " 1 ",
		int64(1_000_000), int64(900), at, nil)
}

func TestCommitteeGetAndPending(t *testing.T) {
	db, mock := be4DB(t)
	s := NewCommitteeStore(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(be4Q("komite_kepala_ambil")).WithArgs("KM.26.1").
		WillReturnRows(sqlmock.NewRows(be4Cols(20)).AddRow(
			" KM.26.1 ", " K1 ", " PNCN.26.1 ", " O ", " 1 ", " 2 ", " 1 ", " T ", " NONMBU ", " 1 ",
			" IDR ", int64(10_000), int64(900), int64(800), " NIK1 ", at, " NIK1 ", at, " NIK2 ", nil))
	mock.ExpectQuery(be4Q("komite_ambil")).WithArgs("KM.26.1").
		WillReturnRows(r3CommitteeMemberRow(sqlmock.NewRows(be4Cols(13)), "KOMITE01", "1"))
	c, err := s.Get(ctx, " KM.26.1 ")
	require.NoError(t, err)
	require.Equal(t, "KM.26.1", c.ID)
	require.Equal(t, 1, c.CoverageSeq)
	require.Equal(t, 2, c.AdjustmentSeq)
	require.Equal(t, "NONMBU", c.Line)
	require.Equal(t, registrasi.ExchangeRate(10_000), c.Rate)
	require.Equal(t, registrasi.Money(800), c.Value)
	require.Equal(t, "NIK2", c.UpdatedBy)
	require.True(t, c.DecidedAt.IsZero())
	require.Len(t, c.Members, 1)
	m := c.Members[0]
	require.Equal(t, "KOMITE01", m.Operator)
	require.Equal(t, 1, m.Level)
	require.Equal(t, "catatan", m.Note)
	require.Equal(t, registrasi.Percent(1_000_000), m.ShareASM)

	mock.ExpectQuery(be4Q("komite_kepala_ambil")).WillReturnRows(sqlmock.NewRows(be4Cols(20)))
	_, err = s.Get(ctx, "X")
	require.ErrorIs(t, err, registrasi.ErrCommitteeNotFound)
	mock.ExpectQuery(be4Q("komite_kepala_ambil")).WillReturnError(be4Boom)
	_, err = s.Get(ctx, "X")
	require.ErrorContains(t, err, "membaca kepala komite")
	mock.ExpectQuery(be4Q("komite_kepala_ambil")).
		WillReturnRows(sqlmock.NewRows(be4Cols(20)).AddRow(make([]driver.Value, 20)...))
	mock.ExpectQuery(be4Q("komite_ambil")).WillReturnError(be4Boom)
	_, err = s.Get(ctx, "X")
	require.ErrorContains(t, err, "membaca komite")

	mock.ExpectQuery(be4Q("komite_tertunda")).WithArgs("KOMITE02").
		WillReturnRows(r3CommitteeMemberRow(sqlmock.NewRows(be4Cols(13)), "KOMITE02", "2"))
	pending, err := s.Pending(ctx, "KOMITE02")
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, 2, pending[0].Level)

	r3RowFailures(t, mock, "komite_tertunda", 13, func() error {
		_, err := s.Pending(ctx, "KOMITE02")
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFaceSheetStoreNames(t *testing.T) {
	db, mock := be4DB(t)
	s := NewFaceSheetStore(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("cfs_penyebab")).WithArgs("11817").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(" KEBAKARAN "))
	name, err := s.CauseOfLossName(ctx, " 11817 ")
	require.NoError(t, err)
	require.Equal(t, "KEBAKARAN", name)
	mock.ExpectQuery(be4Q("cfs_penyebab")).WillReturnRows(sqlmock.NewRows([]string{"n"}))
	name, err = s.CauseOfLossName(ctx, "x")
	require.NoError(t, err)
	require.Empty(t, name)
	mock.ExpectQuery(be4Q("cfs_penyebab")).WillReturnError(be4Boom)
	_, err = s.CauseOfLossName(ctx, "x")
	require.ErrorContains(t, err, `membaca penyebab kerugian "x"`)

	name, err = s.OperatorName(ctx, "  ")
	require.NoError(t, err)
	require.Empty(t, name)
	mock.ExpectQuery(be4Q("cfs_operator")).WithArgs("NIK1", "NIK1").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(" ").AddRow(" BUDI "))
	name, err = s.OperatorName(ctx, " NIK1 ")
	require.NoError(t, err)
	require.Equal(t, "BUDI", name)
	mock.ExpectQuery(be4Q("cfs_operator")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(""))
	name, err = s.OperatorName(ctx, "NIK1")
	require.NoError(t, err)
	require.Empty(t, name)
	r3RowFailures(t, mock, "cfs_operator", 1, func() error {
		_, err := s.OperatorName(ctx, "NIK1")
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFaceSheetStoreCoinsuranceAndFac(t *testing.T) {
	db, mock := be4DB(t)
	s := NewFaceSheetStore(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("polis_koasuransi")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow(" Y ", " ASM ", "60").AddRow("N", "B", "x"))
	rows, err := s.Coinsurance(ctx, " POL ", " 1 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.CoinsuranceRow{
		{Leader: "Y", CoinsName: "ASM", PercentShare: 600_000, HasShare: true},
		{Leader: "N", CoinsName: "B"},
	}, rows)
	r3RowFailures(t, mock, "polis_koasuransi", 3, func() error {
		_, err := s.Coinsurance(ctx, "POL", "1")
		return err
	})

	mock.ExpectQuery(be4Q("cfs_fac_offer")).WithArgs("POL", "1").
		WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(" RE A ", "25.5").AddRow("RE B", ""))
	fac, err := s.FacReinsurers(ctx, " POL ", " 1 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.FacReinsurer{
		{Name: "RE A", Share: 255_000, HasShare: true},
		{Name: "RE B"},
	}, fac)
	r3RowFailures(t, mock, "cfs_fac_offer", 2, func() error {
		_, err := s.FacReinsurers(ctx, "POL", "1")
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFaceSheetStoreRevisions(t *testing.T) {
	db, mock := be4DB(t)
	s := NewFaceSheetStore(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("cfs_revisi_terakhir")).WithArgs("K1", "O", "1").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(3)))
	n, ok, err := s.LastRevision(ctx, "K1", "O", 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 3, n)
	mock.ExpectQuery(be4Q("cfs_revisi_terakhir")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(nil))
	n, ok, err = s.LastRevision(ctx, "K1", "O", 1)
	require.NoError(t, err)
	require.False(t, ok)
	require.Zero(t, n)
	mock.ExpectQuery(be4Q("cfs_revisi_terakhir")).WillReturnError(be4Boom)
	_, _, err = s.LastRevision(ctx, "K1", "O", 1)
	require.ErrorContains(t, err, "membaca revisi Claim Face Sheet")

	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	rev := registrasi.FaceSheetRevision{
		ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Revision: 2, Date: at, FileName: "CFS.pdf",
		Reserve: []registrasi.FaceSheetAmount{{Currency: "IDR", Value: 500}, {Currency: "USD", Value: 7}},
	}
	mock.ExpectExec(be4Q("cfs_sisip")).WithArgs("K1", "O", "1", 2, at, "CFS.pdf").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("cfs_estimasi_sisip")).WithArgs("K1", "O", "1", 2, 1, "IDR", at, int64(500)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("cfs_estimasi_sisip")).WithArgs("K1", "O", "1", 2, 2, "USD", at, int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.SaveRevision(ctx, rev))

	mock.ExpectExec(be4Q("cfs_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.SaveRevision(ctx, rev), "menyimpan revisi Claim Face Sheet")
	mock.ExpectExec(be4Q("cfs_sisip")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("cfs_estimasi_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.SaveRevision(ctx, rev), "menyimpan estimasi Claim Face Sheet")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tombol Delete: DATA_ATTACHFILE lalu JSON_FORM_KLAIM, keduanya menurut IMAGEID.
func TestAttachmentStoreDelete(t *testing.T) {
	db, mock := be4DB(t)
	mock.ExpectExec(be4Q("lampiran_hapus")).WithArgs("IMG-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("form_klaim_hapus")).WithArgs("IMG-1").WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, NewAttachmentStore(db).DeleteAttachment(context.Background(), "IMG-1"))

	mock.ExpectExec(be4Q("lampiran_hapus")).WillReturnError(be4Boom)
	require.ErrorIs(t, NewAttachmentStore(db).DeleteAttachment(context.Background(), "IMG-1"), be4Boom)
	require.NoError(t, mock.ExpectationsWereMet())
}
