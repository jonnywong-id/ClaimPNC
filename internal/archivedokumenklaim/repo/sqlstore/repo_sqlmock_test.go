package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
)

// fileColumns adalah kedua puluh satu kolom yang dibaca scanFile, dalam urutan resultColumns.
var fileColumns = aliasList(resultColumns)

// candidateColumns adalah kedua belas kolom yang dibaca scanCandidate.
var candidateColumns = []string{
	"NUMBER", "POLICY", "INSURED", "LOSS", "BUSINESS", "BRANCH",
	"STATUS", "POSITION", "CLOSE_DATE", "CLOSE_NOTE", "PIC", "GROUP_PANEL",
}

var errBoom = errors.New("basis data mati")

// newMock membentuk repo di atas sqlmock dengan pencocok regexp.
func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// exact mengubah teks kueri menjadi pola regexp yang cocok persis.
func exact(text string) string { return "^" + regexp.QuoteMeta(text) + "$" }

var (
	day1 = time.Date(2024, 3, 11, 0, 0, 0, 0, time.UTC)
	day2 = time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)
)

// fileRow adalah satu baris lengkap kueri daftar, dengan spasi di sekitar teks supaya
// pemangkasan ikut teruji.
func fileRow(id int64) []driver.Value {
	return []driver.Value{
		id, " PNC-1 ", "POL-1", "TERTANGGUNG", day1,
		"PIC", day2, day2, int64(24),
		"0001", "Dokumen Klaim", "000101", "Laporan Kerugian",
		"BOX-A", "FIL-1", "USER1", nil,
		"006", "0", "200", "OK",
	}
}

// Kata kunci hanya mengisi penanda KOLOM YANG DIPILIH; delapan penanda lain NULL.
//
// Itulah yang membuat satu kueri melayani tujuh bentuk penyaring — lihat `search_archive`.
func TestSearchKeywordMenghitungLaluMembacaHalaman(t *testing.T) {
	repo, mock := newMock(t)

	// :1 dan :2 kolom No Klaim; :3..:10 mematikan klausanya sendiri lewat `IS NULL`.
	args := []driver.Value{"PNC-1", "PNC-1", nil, nil, nil, nil, nil, nil, nil, nil}

	mock.ExpectQuery(exact(counted("search_archive"))).
		WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
	mock.ExpectQuery(exact(paged("search_archive"))).
		WithArgs(append(append([]driver.Value{}, args...), 0, 20)...).
		WillReturnRows(sqlmock.NewRows(fileColumns).AddRow(fileRow(7)...))

	page, err := repo.Search(context.Background(), archivedokumenklaim.Criteria{
		Column: archivedokumenklaim.ColumnClaimNumber, Keyword: "PNC-1",
	}, archivedokumenklaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Len(t, page.Files, 1)

	file := page.Files[0]
	require.Equal(t, int64(7), file.ID)
	require.Equal(t, "PNC-1", file.ClaimNumber, "spasi dipangkas")
	require.Equal(t, day1, *file.LossDate)
	require.Nil(t, file.SentDate, "tanggal NULL menjadi nil")
	require.Equal(t, 24, file.SheetCount)
	require.Equal(t, "Laporan Kerugian", file.DocumentKindName)
	require.Equal(t, "OK", file.ServiceNote)
	require.Equal(t, 20, page.Pagination.Size)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Rentang tanggal input menggeser batas atas satu hari.
func TestSearchInputDateMenggeserBatasAtas(t *testing.T) {
	repo, mock := newMock(t)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	until := to.AddDate(0, 0, 1)

	mock.ExpectQuery(exact(counted("search_archive"))).
		WithArgs(nil, nil, nil, nil, nil, nil, from, from, until, until).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(0))

	page, err := repo.Search(context.Background(), archivedokumenklaim.Criteria{
		From: &from, To: &to,
	}, archivedokumenklaim.Pagination{Page: 2, Size: 5})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.Empty(t, page.Files, "hasil kosong tidak menembak kueri halaman")
	require.NotNil(t, page.Files)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Halaman yang berada di luar hasil tidak menembak kueri halaman.
func TestHalamanDiLuarHasilTidakMembacaBaris(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact(counted("pending_all"))).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(3))

	page, err := repo.PendingBranch(context.Background(), archivedokumenklaim.BranchScope{},
		archivedokumenklaim.Pagination{Page: 5, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Empty(t, page.Files)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPendingMemilihKueriMenurutJumlahLini(t *testing.T) {
	cases := []struct {
		name  string
		scope archivedokumenklaim.BranchScope
		args  []driver.Value
	}{
		{"pending_exclude_one", archivedokumenklaim.BranchScope{
			ExcludedGroupPanels: []string{"002"}}, []driver.Value{"002"}},
		{"pending_exclude_two", archivedokumenklaim.BranchScope{
			ExcludedGroupPanels: []string{"002", "005"}}, []driver.Value{"002", "005"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)

			mock.ExpectQuery(exact(counted(c.name))).WithArgs(c.args...).
				WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
			pagedArgs := append(append([]driver.Value{}, c.args...), 0, 20)
			mock.ExpectQuery(exact(paged(c.name))).WithArgs(pagedArgs...).
				WillReturnRows(sqlmock.NewRows(fileColumns).AddRow(fileRow(1)...))

			page, err := repo.PendingBranch(context.Background(), c.scope,
				archivedokumenklaim.Pagination{})
			require.NoError(t, err)
			require.Len(t, page.Files, 1)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPendingMenolakLebihDariDuaLini(t *testing.T) {
	repo, mock := newMock(t)

	_, err := repo.PendingBranch(context.Background(), archivedokumenklaim.BranchScope{
		ExcludedGroupPanels: []string{"001", "002", "003"},
	}, archivedokumenklaim.Pagination{})
	require.ErrorContains(t, err, "3 lini bisnis")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListPageMeneruskanGalat(t *testing.T) {
	criteria := archivedokumenklaim.Criteria{
		Column: archivedokumenklaim.ColumnClaimNumber, Keyword: "X",
	}

	t.Run("hitung", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(counted("search_archive"))).WillReturnError(errBoom)

		_, err := repo.Search(context.Background(), criteria, archivedokumenklaim.Pagination{})
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "menghitung")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("baca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(counted("search_archive"))).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
		mock.ExpectQuery(exact(paged("search_archive"))).WillReturnError(errBoom)

		_, err := repo.Search(context.Background(), criteria, archivedokumenklaim.Pagination{})
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(counted("search_archive"))).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
		mock.ExpectQuery(exact(paged("search_archive"))).
			WillReturnRows(sqlmock.NewRows([]string{"ARCHIVE_ID"}).AddRow(1))

		_, err := repo.Search(context.Background(), criteria, archivedokumenklaim.Pagination{})
		require.ErrorContains(t, err, "membaca baris berkas arsip")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("baris", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(counted("search_archive"))).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
		mock.ExpectQuery(exact(paged("search_archive"))).
			WillReturnRows(sqlmock.NewRows(fileColumns).AddRow(fileRow(1)...).
				RowError(0, errBoom))

		_, err := repo.Search(context.Background(), criteria, archivedokumenklaim.Pagination{})
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindByID(t *testing.T) {
	t.Run("ada", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("find_by_id"))).WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows(fileColumns).AddRow(fileRow(3)...))

		file, exists, err := repo.FindByID(context.Background(), 3)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, int64(3), file.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tidak ada", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("find_by_id"))).WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows(fileColumns))

		_, exists, err := repo.FindByID(context.Background(), 3)
		require.NoError(t, err)
		require.False(t, exists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("galat kueri", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("find_by_id"))).WillReturnError(errBoom)

		_, _, err := repo.FindByID(context.Background(), 3)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("galat baris", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("find_by_id"))).
			WillReturnRows(sqlmock.NewRows(fileColumns).AddRow(fileRow(3)...).
				RowError(0, errBoom))

		_, _, err := repo.FindByID(context.Background(), 3)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("galat pindai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("find_by_id"))).
			WillReturnRows(sqlmock.NewRows([]string{"ARCHIVE_ID"}).AddRow(3))

		_, _, err := repo.FindByID(context.Background(), 3)
		require.ErrorContains(t, err, "membaca baris berkas arsip")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func candidateRow() []driver.Value {
	return []driver.Value{
		" PNC-9 ", "POL-9", "NAMA", day1, "FIRE", "CABANG", "Open", "Register",
		nil, "", "PIC", "006",
	}
}

// Nilai pencarian klaim DIBESARKAN hurufnya oleh repo SQL.
func TestSearchClaimsMemilihKueriMenurutTipe(t *testing.T) {
	t.Run("polis", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("search_claim_by_policy"))).WithArgs("POL-9").
			WillReturnRows(sqlmock.NewRows(candidateColumns).AddRow(candidateRow()...))

		claims, err := repo.SearchClaims(context.Background(), archivedokumenklaim.ClaimCriteria{
			Type: archivedokumenklaim.ClaimByPolicy, Value: "pol-9",
		})
		require.NoError(t, err)
		require.Len(t, claims, 1)
		require.Equal(t, "PNC-9", claims[0].Number)
		require.Equal(t, day1, *claims[0].LossDate)
		require.Nil(t, claims[0].CloseDate)
		require.Equal(t, "006", claims[0].GroupPanel)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("lainnya", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("search_claim_any"))).WithArgs("NAMA", "NAMA", "NAMA").
			WillReturnRows(sqlmock.NewRows(candidateColumns))

		claims, err := repo.SearchClaims(context.Background(), archivedokumenklaim.ClaimCriteria{
			Type: archivedokumenklaim.ClaimByInsuredName, Value: "nama",
		})
		require.NoError(t, err)
		require.Empty(t, claims)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSearchClaimsMeneruskanGalat(t *testing.T) {
	criteria := archivedokumenklaim.ClaimCriteria{Type: archivedokumenklaim.ClaimByNumber, Value: "X"}

	t.Run("kueri", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("search_claim_any"))).WillReturnError(errBoom)
		_, err := repo.SearchClaims(context.Background(), criteria)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("search_claim_any"))).
			WillReturnRows(sqlmock.NewRows([]string{"NUMBER"}).AddRow("X"))
		_, err := repo.SearchClaims(context.Background(), criteria)
		require.ErrorContains(t, err, "membaca baris klaim")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("baris", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("search_claim_any"))).
			WillReturnRows(sqlmock.NewRows(candidateColumns).AddRow(candidateRow()...).
				RowError(0, errBoom))
		_, err := repo.SearchClaims(context.Background(), criteria)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func newDraft(id int64) archivedokumenklaim.Draft {
	return archivedokumenklaim.Draft{
		ID:                   id,
		ClaimNumber:          "PNC-1",
		PolicyNumber:         "",
		InsuredName:          "NAMA",
		LossDate:             nil,
		TechnicalPIC:         "PIC",
		DocumentReceivedDate: &day2,
		SheetCount:           3,
		DocumentTypeCode:     "0001",
		DocumentKindCode:     "000101",
		BoxName:              "BOX",
		FillingCode:          "FIL",
		InputUser:            "USER",
		BranchCode:           "01",
		GroupPanel:           "006",
	}
}

// Penyisipan membaca nomor berikutnya lalu menyisipkan di dalam satu transaksi.
func TestSaveMenyisipkanDenganNomorBerikutnya(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact(query("next_archive_id"))).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(int64(11)))
	mock.ExpectExec(exact(query("insert_archive"))).
		WithArgs(int64(11), "PNC-1", nil, "NAMA", nil, "PIC", day2, 3, "0001", "000101",
			"BOX", "FIL", "USER", "01", "006").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	id, err := repo.Save(context.Background(), newDraft(0))
	require.NoError(t, err)
	require.Equal(t, int64(11), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMengubahBarisYangAda(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(exact(query("update_archive"))).
		WithArgs("PNC-1", nil, "NAMA", nil, "PIC", day2, 3, "0001", "000101", "BOX", "FIL",
			"006", int64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	id, err := repo.Save(context.Background(), newDraft(4))
	require.NoError(t, err)
	require.Equal(t, int64(4), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMeneruskanGalatDanRollback(t *testing.T) {
	t.Run("begin", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBoom)
		_, err := repo.Save(context.Background(), newDraft(0))
		require.ErrorContains(t, err, "membuka transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nomor berikutnya", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact(query("next_archive_id"))).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := repo.Save(context.Background(), newDraft(0))
		require.ErrorContains(t, err, "nomor arsip berikutnya")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sisip", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact(query("next_archive_id"))).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(int64(1)))
		mock.ExpectExec(exact(query("insert_archive"))).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := repo.Save(context.Background(), newDraft(0))
		require.ErrorContains(t, err, "menyisipkan")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ubah", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(exact(query("update_archive"))).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := repo.Save(context.Background(), newDraft(2))
		require.ErrorContains(t, err, "mengubah")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(exact(query("update_archive"))).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errBoom)
		_, err := repo.Save(context.Background(), newDraft(2))
		require.ErrorContains(t, err, "menyimpan berkas arsip")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPenyimpananJawabanLayanan(t *testing.T) {
	sentAt := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	receipt := archivedokumenklaim.Receipt{
		ID: 5, Code: "200", Note: "", Request: `{"a":1}`, SentAt: sentAt,
	}

	for _, name := range []string{"store_receipt", "mark_sent"} {
		t.Run(name, func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectExec(exact(query(name))).
				WithArgs("200", nil, `{"a":1}`, sentAt, int64(5)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			var err error
			if name == "mark_sent" {
				err = repo.MarkSent(context.Background(), receipt)
			} else {
				err = repo.StoreReceipt(context.Background(), receipt)
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("tidak mengenai baris", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectExec(exact(query("mark_sent"))).WillReturnResult(sqlmock.NewResult(0, 0))
		require.ErrorIs(t, repo.MarkSent(context.Background(), receipt),
			archivedokumenklaim.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("galat exec", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectExec(exact(query("store_receipt"))).WillReturnError(errBoom)
		require.ErrorIs(t, repo.StoreReceipt(context.Background(), receipt), errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// Driver yang tidak dapat melaporkan jumlah baris tidak menggagalkan penyimpanan.
	t.Run("rows affected tidak terbaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectExec(exact(query("store_receipt"))).
			WillReturnResult(sqlmock.NewErrorResult(errBoom))
		require.NoError(t, repo.StoreReceipt(context.Background(), receipt))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDocumentTypesDanKinds(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact(query("document_types"))).
		WillReturnRows(sqlmock.NewRows([]string{"CODE", "NAME"}).
			AddRow(" 0001 ", "Dokumen Klaim").AddRow(nil, nil))
	mock.ExpectQuery(exact(query("document_kinds"))).
		WillReturnRows(sqlmock.NewRows([]string{"CODE", "NAME", "TYPE"}).
			AddRow("000101", "Laporan", " 0001 "))

	types, err := repo.DocumentTypes(context.Background())
	require.NoError(t, err)
	require.Equal(t, []archivedokumenklaim.DocumentTypeOption{
		{Code: "0001", Name: "Dokumen Klaim"}, {Code: "", Name: ""},
	}, types)

	kinds, err := repo.DocumentKinds(context.Background())
	require.NoError(t, err)
	require.Equal(t, []archivedokumenklaim.DocumentKindOption{
		{Code: "000101", Name: "Laporan", TypeCode: "0001"},
	}, kinds)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentOptionsMeneruskanGalat(t *testing.T) {
	type call func(r *Repo) error
	types := func(r *Repo) error { _, err := r.DocumentTypes(context.Background()); return err }
	kinds := func(r *Repo) error { _, err := r.DocumentKinds(context.Background()); return err }

	cases := []struct {
		name    string
		query   string
		columns []string
		fn      call
	}{
		{"types", "document_types", []string{"CODE", "NAME"}, types},
		{"kinds", "document_kinds", []string{"CODE", "NAME", "TYPE"}, kinds},
	}

	for _, c := range cases {
		t.Run(c.name+" kueri", func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectQuery(exact(query(c.query))).WillReturnError(errBoom)
			require.ErrorIs(t, c.fn(repo), errBoom)
			require.NoError(t, mock.ExpectationsWereMet())
		})
		t.Run(c.name+" pindai", func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectQuery(exact(query(c.query))).
				WillReturnRows(sqlmock.NewRows([]string{"CODE"}).AddRow("x"))
			require.Error(t, c.fn(repo))
			require.NoError(t, mock.ExpectationsWereMet())
		})
		t.Run(c.name+" baris", func(t *testing.T) {
			repo, mock := newMock(t)
			values := make([]driver.Value, len(c.columns))
			for i := range values {
				values[i] = "x"
			}
			mock.ExpectQuery(exact(query(c.query))).
				WillReturnRows(sqlmock.NewRows(c.columns).AddRow(values...).RowError(0, errBoom))
			require.ErrorIs(t, c.fn(repo), errBoom)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFillingCodesMemakaiPolaLike(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact(query("filling_codes"))).
		WithArgs(`%FIL\_2%`, `%FIL\_2%`).
		WillReturnRows(sqlmock.NewRows([]string{"CODE", "BOX", "USAGE"}).
			AddRow(" FIL_2 ", "BOX", int64(4)))

	codes, err := repo.FillingCodes(context.Background(), " fil_2 ")
	require.NoError(t, err)
	require.Equal(t, []archivedokumenklaim.FillingCodeOption{
		{Code: "FIL_2", BoxName: "BOX", UsageCount: 4},
	}, codes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFillingCodesTanpaKataKunciMemakaiPersen(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact(query("filling_codes"))).WithArgs("%", "%").
		WillReturnRows(sqlmock.NewRows([]string{"CODE", "BOX", "USAGE"}))

	codes, err := repo.FillingCodes(context.Background(), "")
	require.NoError(t, err)
	require.Empty(t, codes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFillingCodesMeneruskanGalat(t *testing.T) {
	t.Run("kueri", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("filling_codes"))).WillReturnError(errBoom)
		_, err := repo.FillingCodes(context.Background(), "")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("pindai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("filling_codes"))).
			WillReturnRows(sqlmock.NewRows([]string{"CODE"}).AddRow("x"))
		_, err := repo.FillingCodes(context.Background(), "")
		require.ErrorContains(t, err, "membaca kode filling")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("baris", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact(query("filling_codes"))).
			WillReturnRows(sqlmock.NewRows([]string{"CODE", "BOX", "USAGE"}).
				AddRow("x", "y", int64(1)).RowError(0, errBoom))
		_, err := repo.FillingCodes(context.Background(), "")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPembantuNilai(t *testing.T) {
	require.Nil(t, nullable("  "))
	require.Equal(t, "A", nullable(" A "))
	require.Nil(t, nullableTime(nil))
	require.Equal(t, day1, nullableTime(&day1))
	require.Nil(t, timeOrNil(sql.NullTime{}))
	require.Equal(t, `%A\%B\\C%`, likePattern(`a%b\c`))
}
