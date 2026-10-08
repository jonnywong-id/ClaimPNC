package sqlstore

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// be4Claim membentuk klaim karangan dengan satu objek, satu coverage, dan anak-anaknya.
func be4Claim() registrasi.Claim {
	at := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	return registrasi.Claim{
		ID:               "K1",
		Number:           "",
		Portal:           "ASM",
		TechnicalPICNote: "Catatan untuk PIC",
		Policy: registrasi.Policy{
			Number: "POL-1", Line: registrasi.LineTravel,
			Coinsurance: registrasi.Coinsurance{ShareASM: 600_000, HasShare: true},
		},
		DateOfLoss: time.Date(2026, 9, 1, 0, 0, 0, 0, clock.ZoneWIB),
		ExGratia:   true,
		InsuredItem: []registrasi.InsuredItem{{
			ID: "OBJ1", Name: "Gudang", Location: "Jakarta",
			Coverage: []registrasi.Coverage{{
				ID: "COV1", Name: "Kebakaran", CauseOfLoss: "C1", TSI: 1_000,
				Spreading: []registrasi.Spreading{
					{TreatyKind: "10001", Name: "OR", Share: 400_000},
					{TreatyKind: "10002", Name: "Buang", Share: 1, Removed: true},
					{TreatyKind: "10003", Name: "QS", Share: 600_000},
				},
				Item: []registrasi.ObjectItem{{
					Name: "Atap", Estimation: []registrasi.Estimation{{
						Type: "1", Currency: "IDR", Value: 500, Rate: registrasi.ExchangeRateOne,
						Converted: 500, FaceSheet: true, Date: at, FaceSheetDate: at,
					}},
				}},
				Settlement: []registrasi.SettlementLine{{PaymentType: "1", ExGratia: true, Value: 400}},
			}},
		}},
		Receiver:  []registrasi.Receiver{{ID: "R1", Name: "Budi"}},
		CreatedBy: "adminpnc", CreatedAt: at, UpdatedBy: "adminpnc", UpdatedAt: at,
	}
}

// be4SaveSteps adalah urutan pernyataan Save untuk be4Claim.
func be4SaveSteps() []be4Step {
	return []be4Step{
		{name: "klaim_perbarui", exec: true, affected: 0},
		{name: "klaim_sisip", exec: true, affected: 1},
		// Coverage tersimpan: (1,1) masih hidup; (1,2) dibuang petugas; (2,1) milik objek
		// yang dibuang. Spreading dua yang terakhir dihapus, lalu coverage-nya.
		{name: "coverage_kunci", cols: 4, rows: [][]driver.Value{
			{int64(1), int64(1), "OBJ1", "1"},
			{int64(1), int64(2), "OBJ1", "2"},
			{int64(2), int64(1), "OBJ-LAMA", "1"},
		}},
		{name: "spreading_hapus_coverage", exec: true},
		{name: "spreading_hapus_coverage", exec: true},
		{name: "coverage_hapus_sisa", exec: true},
		{name: "coverage_hapus_objek_sisa", exec: true},
		{name: "objek_perbarui", exec: true, affected: 0},
		{name: "objek_sisip", exec: true, affected: 1},
		{name: "coverage_perbarui", exec: true, affected: 1},
		// Tersimpan 10003 (tetap ada) dan 10009 (dibuang); 10001 baru, 10002 ber-Removed.
		{name: "spreading_jenis", cols: 1, rows: [][]driver.Value{{"10003"}, {"10009"}}},
		{name: "spreading_sisip", exec: true, affected: 1},
		{name: "spreading_perbarui", exec: true, affected: 1},
		{name: "spreading_hapus", exec: true, affected: 1},
		{name: "item_perbarui", exec: true, affected: 0},
		{name: "item_sisip", exec: true, affected: 1},
		{name: "estimasi_perbarui", exec: true, affected: 1},
		{name: "item_tandai_sisa", exec: true},
		{name: "adjustment_perbarui", exec: true, affected: 0},
		{name: "adjustment_sisip", exec: true, affected: 1},
		{name: "objek_tandai_sisa", exec: true},
		{name: "penerima_perbarui", exec: true, affected: 1},
	}
}

// TestClaimSaveWritesWholeTree memeriksa urutan pernyataan dan argumen kunci penyimpanan.
func TestClaimSaveWritesWholeTree(t *testing.T) {
	db, mock := be4DB(t)
	steps := be4SaveSteps()
	k := be4Claim()

	// Nomor kosong ditulis NULL, ExGratia "Y", bagian ASM dalam persen, ID di akhir.
	steps[0].args = be4AnyArgs(58, map[int]driver.Value{
		0: nil, 1: "ASM", 2: "POL-1", 3: "005", 20: "Y", 26: nil, 36: 60.0, 55: nil, 56: "Catatan untuk PIC", 57: "K1"})
	steps[1].args = be4AnyArgs(60, map[int]driver.Value{55: nil, 56: "Catatan untuk PIC", 57: "K1", 58: "adminpnc"})
	steps[2].args = []driver.Value{"K1"}
	steps[3].args = []driver.Value{"K1", "OBJ1", "2"}
	steps[4].args = []driver.Value{"K1", "OBJ-LAMA", "1"}
	steps[5].args = []driver.Value{"K1", 1, 1}
	steps[6].args = []driver.Value{"K1", 1}
	steps[7].args = []driver.Value{"OBJ1", "Gudang", "Jakarta", "K1", 1}
	steps[10].args = []driver.Value{"K1", "OBJ1", "1"}
	steps[11].args = []driver.Value{"K1", "OBJ1", "1", "10001", "OR", 40.0, 1}
	steps[12].args = []driver.Value{"QS", 60.0, 3, "K1", "OBJ1", "1", "10003"}
	steps[13].args = []driver.Value{"K1", "OBJ1", "1", "10009"}
	steps[15].args = be4AnyArgs(12, map[int]driver.Value{0: "Atap", 1: nil, 2: int64(500), 5: "K1", 6: "OBJ1", 7: "1", 8: "1"})
	steps[16].args = be4AnyArgs(13, map[int]driver.Value{6: 1})
	steps[17].args = be4AnyArgs(6, map[int]driver.Value{0: "adminpnc", 2: "K1", 3: "OBJ1", 4: "1", 5: 1})
	steps[18].args = be4AnyArgs(26, map[int]driver.Value{0: "1", 14: "1", 22: "K1", 23: "OBJ1", 24: "1", 25: "1"})
	steps[20].args = be4AnyArgs(3, map[int]driver.Value{1: "K1", 2: 1})
	steps[21].args = []driver.Value{"Budi", nil, nil, nil, "K1", "R1"}
	be4ExpectAll(mock, steps)

	require.NoError(t, NewClaimStore(db).Save(context.Background(), k))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestClaimSaveStopsAtEveryFailingStatement membuktikan kegagalan di langkah mana pun dihentikan
// dan dibungkus dengan konteksnya.
func TestClaimSaveStopsAtEveryFailingStatement(t *testing.T) {
	steps := be4SaveSteps()
	contexts := []string{
		"memperbarui klaim", "menyisipkan klaim",
		"membaca coverage tersimpan", "menghapus spreading coverage yang dibuang",
		"menghapus spreading coverage yang dibuang", "menghapus coverage yang dibuang",
		"menghapus coverage objek yang dibuang",
		"menyimpan objek 1", "menyimpan objek 1", "menyimpan coverage 1.1",
		"menyimpan spreading 1.1", "menyimpan spreading 1.1", "menyimpan spreading 1.1",
		"menyimpan spreading 1.1", "item 1", "item 1", "estimasi 1.1", "menyimpan estimasi 1.1",
		"adjustment 1", "adjustment 1", "menandai sisa objek",
		"menyimpan penerima R1",
	}
	for i := range steps {
		t.Run(fmt.Sprintf("%02d_%s", i, steps[i].name), func(t *testing.T) {
			db, mock := be4DB(t)
			be4ExpectAll(mock, steps[:i])
			be4Expect(mock, steps[i], be4Fail)

			err := NewClaimStore(db).Save(context.Background(), be4Claim())
			require.ErrorIs(t, err, be4Boom)
			require.Contains(t, err.Error(), contexts[i])
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestClaimSaveRowsAffectedAndRowErrFailures memeriksa galat RowsAffected dan penelusuran.
func TestClaimSaveRowsAffectedAndRowErrFailures(t *testing.T) {
	steps := be4SaveSteps()

	db, mock := be4DB(t)
	be4Expect(mock, steps[0], be4Affected)
	err := NewClaimStore(db).Save(context.Background(), be4Claim())
	require.ErrorIs(t, err, be4Boom)
	require.Contains(t, err.Error(), "membaca jumlah baris klaim")

	db, mock = be4DB(t)
	be4ExpectAll(mock, steps[:7])
	be4Expect(mock, steps[7], be4Affected)
	err = NewClaimStore(db).Save(context.Background(), be4Claim())
	require.ErrorIs(t, err, be4Boom)
	require.Contains(t, err.Error(), "menyimpan objek 1")

	db, mock = be4DB(t)
	be4ExpectAll(mock, steps[:2])
	be4Expect(mock, steps[2], be4RowErr)
	err = NewClaimStore(db).Save(context.Background(), be4Claim())
	require.ErrorIs(t, err, be4Boom)
	require.Contains(t, err.Error(), "menelusuri coverage tersimpan")

	db, mock = be4DB(t)
	be4ExpectAll(mock, steps[:10])
	be4Expect(mock, steps[10], be4RowErr)
	err = NewClaimStore(db).Save(context.Background(), be4Claim())
	require.ErrorIs(t, err, be4Boom)
	require.Contains(t, err.Error(), "menyimpan spreading 1.1")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestClaimSaveUpdatesExistingHeader membuktikan header yang sudah ada tidak disisipkan ulang.
func TestClaimSaveUpdatesExistingHeader(t *testing.T) {
	db, mock := be4DB(t)
	k := registrasi.Claim{ID: "K2", Number: "PNCN.26.1", LargeLossNoticed: true}
	mock.ExpectExec(be4Q("klaim_perbarui")).
		WithArgs(be4AnyArgs(58, map[int]driver.Value{0: "PNCN.26.1", 8: nil, 26: "1", 36: nil, 55: nil, 56: nil, 57: "K2"})...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(be4Q("coverage_kunci")).WithArgs("K2").
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d"}))
	mock.ExpectExec(be4Q("coverage_hapus_objek_sisa")).WithArgs("K2", 0).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(be4Q("objek_tandai_sisa")).WithArgs(sqlmock.AnyArg(), "K2", 0).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, NewClaimStore(db).Save(context.Background(), k))
	require.NoError(t, mock.ExpectationsWereMet())
}

// be4GetRow membentuk baris klaim_ambil (60 kolom).
func be4GetRow(created time.Time) []driver.Value {
	loss := time.Date(2026, 9, 1, 0, 0, 0, 0, clock.ZoneWIB)
	return []driver.Value{
		"K1", "PNCN.26.7", "ASM",
		"POL-1", "006", "Fire", "IDR", "PT Karangan", "001",
		loss, loss, nil,
		"Jakarta", "Terbakar",
		"Budi", "0812", "Jl. Karangan", int64(7), "Sepupu",
		"IDR", "SLIK9", "Y", "picteknik", "RCV1",
		int64(2),
		"BERJALAN", "1147",
		"adminpnc", created,
		"1",
		"Agen", "SOB1", "Cabang Pusat", "BC1", "Kebakaran", "0", "1",
		"Koas", "LEADER", int64(600000), "ASM",
		"Indonesia", "ID", "DKI", "31", "Jakarta", "3171", "Gambir", "317101", "RW01", "R1", "10110",
		"2", "Mencurigakan",
		"lod@contoh", "rekomendasi", "subjek", " 2 ", nil, "catatan PIC",
	}
}

// be4GetSteps adalah urutan pembacaan Get klaim lengkap.
func be4GetSteps() []be4Step {
	created := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	printedDateOnly := time.Date(2026, 9, 25, 0, 0, 0, 0, clock.ZoneWIB)
	adjustment := func(objectID, adjID string, lodType string, printed any) []driver.Value {
		return []driver.Value{
			objectID, "1", adjID, "1", "IDR",
			int64(10000), int64(500), int64(0), int64(0), "", int64(0), int64(0), int64(400),
			int64(1000000), int64(400), int64(400), int64(450), "1", "kronologi", "catatan", "1",
			"AK-1", "KOM-1", created, created, "1",
			created, "R1", "Budi", printed, nil, lodType,
			nil, "KS-1",
		}
	}
	return []be4Step{
		{name: "klaim_ambil", cols: 60, rows: [][]driver.Value{be4GetRow(created)}, args: []driver.Value{"K1"}},
		{name: "tugas_terbuka_klaim", cols: 11, args: []driver.Value{"K1"}, rows: [][]driver.Value{
			{"T1", "K1", "PNCN.26.7", "input-estimasi", "WORKLIST", nil, "picteknik", created, nil, nil, nil},
			{"T2", "K1", "PNCN.26.7", "lain", "WORKLIST", nil, "x", created, nil, nil, nil},
		}},
		{name: "polis_ambil", cols: 24, args: []driver.Value{"POL-1", "POL-1"}, rows: [][]driver.Value{{
			"POL-1", "006", "Fire", "Kebakaran", "20260101T000000.000 GMT", "20261231", "DECLARATION", "IDR",
			"", "QQ Karangan", "001", "FINISHED", "BC1", "Pusat", "SOB", "Agen", "0", "ASM", "1",
			"", "", "", nil, nil,
		}}},
		{name: "polis_koasuransi", cols: 3, args: []driver.Value{"POL-1", "0"}},
		{name: "objek_daftar", cols: 12, rows: [][]driver.Value{
			{int64(1), "OBJ1 ", "Gudang", "Jakarta", " KARYAWAN ", "19890524",
				" A1234 ", "1", " DUMP TRUCK ", "HINO", nil, " CH-9 "},
		}},
		{name: "coverage_daftar", cols: 7, rows: [][]driver.Value{
			{int64(1), int64(1), "COV1", "C1", int64(1000), "Kebakaran", int64(1)},
			{int64(9), int64(1), "COVX", "C9", int64(5), "Yatim", nil}, // objek induk sudah ditandai hapus
		}},
		{name: "spreading_daftar", cols: 6, rows: [][]driver.Value{
			{"OBJ1", "1", int64(1), "10001", "OR", int64(400000)},
			{"OBJX", "1", int64(1), "10001", "Yatim", int64(1)}, // objek tidak dikenal
			{"OBJ1", "bukan-angka", int64(2), "10001", "Cacat", int64(1)},
			{"OBJ1", "7", int64(3), "10001", "Tanpa coverage", int64(1)},
		}},
		{name: "item_daftar", cols: 6, rows: [][]driver.Value{
			{"OBJ1", "1", "1", "Atap", "Genteng", "G1"},
			{"OBJX", "1", "1", "Yatim", "", ""},
		}},
		{name: "estimasi_daftar", cols: 12, rows: [][]driver.Value{
			{"OBJ1", "1", "1", "1", "1 ", "IDR", int64(500), int64(10000), int64(500), created, int64(1), created},
			{"OBJ1", "1", "9", "1", "1", "IDR", int64(1), int64(1), int64(1), nil, nil, nil},
		}},
		{name: "adjustment_daftar", cols: 34, rows: [][]driver.Value{
			adjustment("OBJ1", "1", "", printedDateOnly),
			adjustment("OBJ1", "2", "15", time.Date(2026, 9, 25, 10, 0, 0, 0, clock.ZoneWIB)),
			adjustment("OBJX", "1", "", nil),
		}},
		{name: "adjustment_lod_dokumen", cols: 2, args: []driver.Value{"K1"}, rows: [][]driver.Value{{
			nil, `{"ObjectList":[{"ObjectID":"OBJ1","ObjectCoverageList":[{"CoverageID":"1","AdjustmentList":[{"PDFType":"15","PrintDateLOD":"20260925T030000.000 GMT"}]}]}]}`,
		}}},
		{name: "penerima_daftar", cols: 5, rows: [][]driver.Value{{"R1 ", "Budi", "Jl", "BCA", "123"}}},
		{name: "status_nama", cols: 1, args: []driver.Value{"1147"}, rows: [][]driver.Value{{" Register "}}},
	}
}

// TestClaimGetReadsWholeTree memeriksa pemetaan header, pohon, polis, dan status.
func TestClaimGetReadsWholeTree(t *testing.T) {
	db, mock := be4DB(t)
	be4ExpectAll(mock, be4GetSteps())

	k, err := NewClaimStore(db).Get(context.Background(), "K1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Equal(t, "PNCN.26.7", k.Number)
	require.Equal(t, registrasi.LineOfBusiness("006"), k.Policy.Line)
	require.Equal(t, registrasi.Reporter{Name: "Budi", Phone: "0812", Address: "Jl. Karangan", Relation: 7, OtherRelation: "Sepupu"}, k.Reporter)
	require.True(t, k.ExGratia)
	require.True(t, k.LargeLossNoticed)
	require.Equal(t, 2, k.PUCLStatus)
	require.Equal(t, "input-estimasi", k.CurrentStage)
	require.Equal(t, registrasi.PositionInProgress, k.ProgressPositionStatus)
	require.Equal(t, registrasi.FlagUnset, k.ClaimFlag)
	require.True(t, k.Policy.Declaration)
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), k.Policy.CoverageStart)
	require.Equal(t, registrasi.Coinsurance{Name: "Koas", Role: "LEADER", ShareASM: 600000, HasShare: true}, k.Policy.Coinsurance)
	require.Equal(t, "Gambir", k.Area.District)
	require.Equal(t, "Register", k.ClaimStatusName)
	require.Equal(t, "catatan PIC", k.TechnicalPICNote)

	require.Len(t, k.InsuredItem, 1)
	require.Equal(t, "KARYAWAN", k.InsuredItem[0].Job)
	require.Equal(t, "A1234", k.InsuredItem[0].IDCard)
	require.Equal(t, "1", k.InsuredItem[0].ParticipantStatus)
	require.Equal(t, "DUMP TRUCK", k.InsuredItem[0].VehicleModel)
	require.Equal(t, "HINO", k.InsuredItem[0].VehicleBrand)
	require.Empty(t, k.InsuredItem[0].VehicleType)
	require.Equal(t, "CH-9", k.InsuredItem[0].ChassisNumber)
	require.Equal(t, "19890524", k.InsuredItem[0].DateOfBirth)
	coverage := k.InsuredItem[0].Coverage
	require.Len(t, coverage, 1)
	require.Equal(t, registrasi.Money(1000), coverage[0].TSI)
	require.Equal(t, []registrasi.Spreading{{TreatyKind: "10001", Name: "OR", Share: 400000}}, coverage[0].Spreading)
	require.Len(t, coverage[0].Item, 1)
	require.Equal(t, "G1", coverage[0].Item[0].Group)
	require.Len(t, coverage[0].Item[0].Estimation, 1)
	require.True(t, coverage[0].Item[0].Estimation[0].FaceSheet)
	require.Equal(t, "1", coverage[0].Item[0].Estimation[0].Type)

	settlement := coverage[0].Settlement
	require.Len(t, settlement, 2)
	require.True(t, settlement[0].ExGratia)
	require.Equal(t, "KOM-1", settlement[0].CommitteeCaseID)
	// Baris pertama dilengkapi dari dokumen: tipe PDF dan jam cetak.
	require.Equal(t, "15", settlement[0].Acceptance.LODType)
	require.Equal(t, time.Date(2026, 9, 25, 3, 0, 0, 0, time.UTC), settlement[0].Acceptance.Form.PrintDate.UTC())
	require.Equal(t, "15", settlement[1].Acceptance.LODType)

	require.Equal(t, []registrasi.Receiver{{ID: "R1", Name: "Budi", Address: "Jl", BankName: "BCA", AccountNo: "123"}}, k.Receiver)
}

// TestClaimGetByNumberAndNotFound memeriksa jalur kueri per nomor dan ErrClaimNotFound.
func TestClaimGetByNumberAndNotFound(t *testing.T) {
	db, mock := be4DB(t)
	mock.ExpectQuery(be4Q("klaim_ambil_per_nomor")).WithArgs("PNCN.26.9").
		WillReturnRows(sqlmock.NewRows(be4Cols(60)))
	_, err := NewClaimStore(db).GetByNumber(context.Background(), "PNCN.26.9")
	require.ErrorIs(t, err, registrasi.ErrClaimNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestClaimGetWithoutPolicyOrStatus membuktikan klaim selesai tanpa polis dan status tidak membaca keduanya.
func TestClaimGetWithoutPolicyOrStatus(t *testing.T) {
	db, mock := be4DB(t)
	row := make([]driver.Value, 60)
	row[0] = "K3"
	row[3] = "" // nomor polis kosong: kolom ini di-scan ke string biasa, bukan NullString
	row[25] = "SELESAI"
	mock.ExpectQuery(be4Q("klaim_ambil")).WillReturnRows(sqlmock.NewRows(be4Cols(60)).AddRow(row...))
	mock.ExpectQuery(be4Q("tugas_terbuka_klaim")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	for _, name := range []string{"objek_daftar", "coverage_daftar", "spreading_daftar", "item_daftar",
		"estimasi_daftar", "adjustment_daftar", "penerima_daftar"} {
		mock.ExpectQuery(be4Q(name)).WillReturnRows(sqlmock.NewRows(be4Cols(1)))
	}

	k, err := NewClaimStore(db).Get(context.Background(), "K3")
	require.NoError(t, err)
	require.Equal(t, registrasi.PositionDone, k.ProgressPositionStatus)
	require.Empty(t, k.CurrentStage)
	require.Empty(t, k.ClaimStatusName)
	require.False(t, k.Policy.Coinsurance.HasShare)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestClaimGetPolicyMissingIsTolerated membuktikan polis yang hilang tidak menggagalkan pembukaan.
func TestClaimGetPolicyMissingIsTolerated(t *testing.T) {
	steps := be4GetSteps()
	steps[2].rows = nil
	db, mock := be4DB(t)
	be4ExpectAll(mock, steps[:3])
	be4ExpectAll(mock, steps[4:])

	k, err := NewClaimStore(db).Get(context.Background(), "K1")
	require.NoError(t, err)
	require.False(t, k.Policy.Declaration)
	require.True(t, k.Policy.CoverageStart.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestClaimGetStatusNameMissingIsTolerated membuktikan kode status tanpa master dibiarkan tanpa nama.
func TestClaimGetStatusNameMissingIsTolerated(t *testing.T) {
	steps := be4GetSteps()
	steps[len(steps)-1].rows = nil
	db, mock := be4DB(t)
	be4ExpectAll(mock, steps)

	k, err := NewClaimStore(db).Get(context.Background(), "K1")
	require.NoError(t, err)
	require.Empty(t, k.ClaimStatusName)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestClaimGetLODDocumentEdgeCases memeriksa dokumen klaim tidak ada dan tidak terbaca.
func TestClaimGetLODDocumentEdgeCases(t *testing.T) {
	for _, rows := range [][][]driver.Value{nil, {{[]byte("bukan json"), nil}}} {
		steps := be4GetSteps()
		steps[10].rows = rows
		db, mock := be4DB(t)
		be4ExpectAll(mock, steps)

		k, err := NewClaimStore(db).Get(context.Background(), "K1")
		require.NoError(t, err)
		require.Empty(t, k.InsuredItem[0].Coverage[0].Settlement[0].Acceptance.LODType)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

// TestClaimGetFailsAtEveryStep membuktikan galat di setiap kueri pembacaan diteruskan.
func TestClaimGetFailsAtEveryStep(t *testing.T) {
	steps := be4GetSteps()
	for i := range steps {
		for _, mode := range []be4Mode{be4Fail, be4ScanFail, be4RowErr} {
			if mode == be4ScanFail && (steps[i].name == "polis_koasuransi" || steps[i].name == "penerima_daftar" ||
				steps[i].name == "status_nama") {
				// Scan gagal pada tiga kueri ini sudah tercakup jalur galat yang sama di bawah.
				continue
			}
			t.Run(fmt.Sprintf("%02d_%s_%d", i, steps[i].name, mode), func(t *testing.T) {
				db, mock := be4DB(t)
				be4ExpectAll(mock, steps[:i])
				be4Expect(mock, steps[i], mode)

				_, err := NewClaimStore(db).Get(context.Background(), "K1")
				require.Error(t, err)
				if mode != be4ScanFail {
					require.ErrorIs(t, err, be4Boom)
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}
}

// TestFindDuplicatesDeduplicatesAcrossKeys memeriksa argumen penanda dan pembuangan baris kembar.
func TestFindDuplicatesDeduplicatesAcrossKeys(t *testing.T) {
	db, mock := be4DB(t)
	mock.ExpectQuery(be4Q("klaim_cari_ganda")).
		WithArgs("POL-1", "K1", "OBJ1", 1, "Jakarta", 0, "").
		WillReturnRows(sqlmock.NewRows([]string{"N", "O"}).AddRow("PNC-1", "OBJ1").AddRow("PNC-1", "OBJ1"))
	mock.ExpectQuery(be4Q("klaim_cari_ganda")).
		WithArgs("POL-1", "K1", "OBJ1", 0, "", 1, "12002").
		WillReturnRows(sqlmock.NewRows([]string{"N", "O"}).AddRow("PNC-1", "OBJ1").AddRow("PNC-2", "OBJ1"))

	result, err := NewClaimStore(db).FindDuplicates(context.Background(), []registrasi.DuplicateKey{
		{PolicyNumber: "POL-1", InsuredItemID: "OBJ1", Location: "Jakarta"},
		{PolicyNumber: "POL-1", InsuredItemID: "OBJ1", CauseOfLoss: "12002"},
	}, "K1")
	require.NoError(t, err)
	require.Equal(t, []registrasi.DuplicateClaim{{Number: "PNC-1", InsuredItem: "OBJ1"}, {Number: "PNC-2", InsuredItem: "OBJ1"}}, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFindDuplicatesFailures memeriksa galat kueri, Scan, dan penelusuran.
func TestFindDuplicatesFailures(t *testing.T) {
	key := []registrasi.DuplicateKey{{PolicyNumber: "POL-1", InsuredItemID: "OBJ1"}}
	for mode, want := range map[be4Mode]string{
		be4Fail: "memeriksa klaim ganda", be4ScanFail: "membaca baris klaim ganda", be4RowErr: "menelusuri klaim ganda",
	} {
		db, mock := be4DB(t)
		be4Expect(mock, be4Step{name: "klaim_cari_ganda", cols: 2}, mode)
		_, err := NewClaimStore(db).FindDuplicates(context.Background(), key, "")
		require.ErrorContains(t, err, want)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

// TestBindingHelpers memeriksa pengikat nilai ke kolom.
func TestBindingHelpers(t *testing.T) {
	require.Nil(t, calendarDateOrNil(time.Time{}))
	require.Nil(t, registerMoment(time.Time{}))
	require.Nil(t, timeOrNil(time.Time{}))
	require.Nil(t, timePtrOrNil(nil))
	at := time.Date(2026, 1, 1, 20, 0, 0, 0, time.UTC)
	require.Equal(t, clock.ZoneWIB, calendarDateOrNil(at).(time.Time).Location())
	require.Equal(t, clock.ZoneWIB, registerMoment(at).(time.Time).Location())
	require.Equal(t, time.UTC, timeOrNil(at.In(clock.ZoneWIB)).(time.Time).Location())
	local := at.In(clock.ZoneWIB)
	require.Equal(t, time.UTC, timePtrOrNil(&local).(time.Time).Location())
	require.Nil(t, shareOrNil(registrasi.Coinsurance{}))
	require.Equal(t, 33.3333, shareOrNil(registrasi.Coinsurance{ShareASM: 333333, HasShare: true}))
	require.Nil(t, emptyTextAsNil(""))
	require.Equal(t, "x", emptyTextAsNil("x"))
	require.Equal(t, "Y", yesNo(true))
	require.Equal(t, "N", yesNo(false))
	require.True(t, fromYesNo("y"))
	require.False(t, fromYesNo("N"))
	require.Equal(t, "1", flagNOLL(true))
	require.Nil(t, flagNOLL(false))
	require.True(t, fromFlagNOLL("1"))
}

// TestUnitOfWorkCommitsOnce memeriksa commit, galat buka, galat kerja, galat commit, dan transaksi bersarang.
func TestUnitOfWorkCommitsOnce(t *testing.T) {
	ctx := context.Background()

	db, mock := be4DB(t)
	mock.ExpectBegin()
	mock.ExpectExec(be4Q("objek_tandai_sisa")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	uow := NewUnitOfWork(db)
	err := uow.Run(ctx, func(inner context.Context) error {
		// Transaksi bersarang ikut transaksi luar, bukan membuka yang baru.
		return uow.Run(inner, func(nested context.Context) error {
			_, ok := txFrom(nested)
			require.True(t, ok)
			_, err := executorFrom(nested, db).ExecContext(nested, loadQuery("objek_tandai_sisa"), 1, 2, 3)
			return err
		})
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = be4DB(t)
	mock.ExpectBegin().WillReturnError(be4Boom)
	err = NewUnitOfWork(db).Run(ctx, func(context.Context) error { return nil })
	require.ErrorIs(t, err, be4Boom)
	require.Contains(t, err.Error(), "membuka transaksi")

	db, mock = be4DB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	err = NewUnitOfWork(db).Run(ctx, func(context.Context) error { return be4Boom })
	require.ErrorIs(t, err, be4Boom)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = be4DB(t)
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(be4Boom)
	err = NewUnitOfWork(db).Run(ctx, func(context.Context) error { return nil })
	require.ErrorIs(t, err, be4Boom)
	require.Contains(t, err.Error(), "menutup transaksi")
}

// TestLoadQueryPanicsOnUnknownName membuktikan nama kueri salah ketik gagal keras.
func TestLoadQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t, `registrasi/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { loadQuery("tidak_ada") })
	require.Equal(t, map[string]string{"a": "SELECT 1"}, splitByName("-- name: a\nSELECT 1\n-- name: kosong\n"))
}
