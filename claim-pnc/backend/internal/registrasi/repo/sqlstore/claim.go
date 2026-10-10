package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// ClaimStore menyimpan klaim beserta pohon objek–coverage–spreading di bawahnya.
type ClaimStore struct {
	db *sql.DB
}

// NewClaimStore membentuk repo; db wajib sudah terhubung.
func NewClaimStore(db *sql.DB) *ClaimStore { return &ClaimStore{db: db} }

// Save menuliskan klaim beserta seluruh pohon di bawahnya.
//
// # Kenapa UPDATE lalu INSERT, bukan MERGE
//
// Mengikuti keputusan 2.9 pada `docs/keputusan-implementasi.md`: MERGE bukan sintaks yang
// sama antara Oracle dan PostgreSQL, dan `ADR-0005` menetapkan perpindahan ke PostgreSQL
// akan datang. UPDATE-lalu-INSERT berjalan apa adanya di keduanya.
//
// # Kenapa baris anak ditandai, bukan dihapus
//
// Objek yang dibuang petugas TIDAK dihapus dari tabel; ia ditandai lewat DIHAPUS_PADA.
// `ADR-0012` melarang penghapusan fisik, dan pola hapus-lalu-sisip-ulang yang
// menggantikannya masih menunggu `ADR-0013`.
//
// Pemanggil bertanggung jawab atas batas transaksi. Bila context sudah membawa
// transaksi, seluruh pernyataan di sini ikut transaksi itu — itulah yang membuat janji
// "gagal di langkah mana pun tidak meninggalkan satu baris pun" dapat ditepati.
func (r *ClaimStore) Save(ctx context.Context, k registrasi.Claim) error {
	exec := executorFrom(ctx, r.db)

	if err := r.saveHeader(ctx, exec, k); err != nil {
		return err
	}
	if err := r.saveTree(ctx, exec, k); err != nil {
		return err
	}
	return saveReceivers(ctx, exec, k.ID, k.Receiver)
}

func (r *ClaimStore) saveHeader(ctx context.Context, exec executor, k registrasi.Claim) error {
	// Urutannya mengikuti klaim_perbarui dan klaim_sisip kolom demi kolom. Tabelnya
	// diubah Work Owner pada 2026-09-26 menjadi 82 kolom; empat belas medan yang kolomnya
	// dibuang tidak lagi ditulis — lihat restoreDropped untuk cara medan itu dipulihkan.
	args := []any{
		emptyTextAsNil(k.Number),
		k.Portal,
		k.Policy.Number,
		string(k.Policy.Line),
		k.Policy.BusinessType,
		k.Policy.Currency,
		k.Policy.InsuredName,
		k.Policy.BranchCode,
		calendarDateOrNil(k.DateOfLoss),
		calendarDateOrNil(k.ReportDate),
		calendarDateOrNil(k.DateReceived),
		k.Location,
		k.Chronology,
		k.Reporter.Name,
		k.Reporter.Phone,
		k.Reporter.Address,
		k.Reporter.Relation,
		k.Reporter.OtherRelation,
		k.Currency,
		k.SLIKNumber,
		yesNo(k.ExGratia),
		k.TechnicalPIC,
		k.RCVID,
		k.PUCLStatus,
		string(k.ProcessStatus),
		string(k.ClaimStatus),
		flagNOLL(k.LargeLossNoticed),

		// Diisi Pega saat klaim dibuat, dari PolicyData yang dibekukan bersama klaimnya
		// (PEGA_CONVERT_JSONKLAIM_PNC.prc baris 317-373). Disimpan di sini sebagai snapshot
		// polis (D-04): pembukaan klaim berikutnya membacanya dari baris ini, bukan dari
		// dokumen polis yang mungkin sudah berubah.
		emptyTextAsNil(k.Policy.SourceOfBusinessName),
		emptyTextAsNil(k.Policy.SourceOfBusiness),
		emptyTextAsNil(k.Policy.BranchName),
		emptyTextAsNil(k.Policy.BusinessCode),
		emptyTextAsNil(k.Policy.BusinessName),
		emptyTextAsNil(k.Policy.ProdKe),
		emptyTextAsNil(k.Policy.TypeOfCoins),
		emptyTextAsNil(k.Policy.Coinsurance.Name),
		emptyTextAsNil(k.Policy.Coinsurance.Role),
		shareOrNil(k.Policy.Coinsurance),
		emptyTextAsNil(k.Policy.PolicyLeader),

		// Wilayah kejadian dan Prinsip Mengenal Nasabah (migrasi 0012).
		emptyTextAsNil(k.Area.Country),
		emptyTextAsNil(k.Area.CountryID),
		emptyTextAsNil(k.Area.Province),
		emptyTextAsNil(k.Area.ProvinceID),
		emptyTextAsNil(k.Area.City),
		emptyTextAsNil(k.Area.CityID),
		emptyTextAsNil(k.Area.District),
		emptyTextAsNil(k.Area.DistrictID),
		emptyTextAsNil(k.Area.RW),
		emptyTextAsNil(k.Area.RWID),
		emptyTextAsNil(k.Area.PostalCode),
		emptyTextAsNil(k.CustomerPrinciple),
		emptyTextAsNil(k.SuspiciousComment),

		// Isian InputRegisterDetail2_sect.
		emptyTextAsNil(k.EmailLOD),
		emptyTextAsNil(k.RemarkRecommendation),
		emptyTextAsNil(k.SubjectEmail),
		emptyTextAsNil(k.SalvageStatus),
		registerMoment(k.AnalystTransferredAt),
		k.ID,
	}

	result, err := exec.ExecContext(ctx, loadQuery("klaim_perbarui"), args...)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: memperbarui klaim: %w", err)
	}
	row, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca jumlah baris klaim: %w", err)
	}
	if row > 0 {
		return nil
	}

	// REGISTERDATE bertipe DATE: jam dindingnya disimpan apa adanya, sehingga diikat dalam
	// WIB seperti baris Pega — bukan UTC, yang menyimpannya tujuh jam lebih awal.
	args = append(args, k.CreatedBy, registerMoment(k.CreatedAt))
	if _, err := exec.ExecContext(ctx, loadQuery("klaim_sisip"), args...); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyisipkan klaim: %w", err)
	}
	return nil
}

func (r *ClaimStore) saveTree(ctx context.Context, exec executor, k registrasi.Claim) error {
	now := k.UpdatedAt.UTC()

	for i, o := range k.InsuredItem {
		itemSeq := i + 1
		if err := upsert(ctx, exec,
			"objek_perbarui", []any{o.ID, o.Name, o.Location, k.ID, itemSeq},
			"objek_sisip", []any{o.ID, o.Name, o.Location, k.ID, itemSeq},
		); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan objek %d: %w", itemSeq, err)
		}

		for j, c := range o.Coverage {
			coverageSeq := j + 1
			// o.ID ikut dikirim karena OBJECTID `NOT NULL` di POOLDATA.T_CLAIM_OBJECTCOVERAGE.
			// Coverage memang milik sebuah objek; tabel warisan menuntutnya dinyatakan, dan
			// domain sudah memilikinya di tangan.
			if err := upsert(ctx, exec,
				"coverage_perbarui", []any{
					c.ID, c.CauseOfLoss, int64(c.TSI), o.ID, coverageSeq, emptyTextAsNil(c.Name),
					flag(c.AnalystTransferred), flag(c.AnalystTransferred), flag(c.AnalystTransferred),
					k.ID, itemSeq, coverageSeq},
				"coverage_sisip", []any{
					c.ID, c.CauseOfLoss, int64(c.TSI), o.ID, coverageSeq, now,
					k.ID, itemSeq, coverageSeq, emptyTextAsNil(c.Name)},
			); err != nil {
				return fmt.Errorf("registrasi/sqlstore: menyimpan coverage %d.%d: %w", itemSeq, coverageSeq, err)
			}

			if err := r.saveSpreading(ctx, exec, k.ID, o.ID, coverageSeq, c.Spreading); err != nil {
				return fmt.Errorf("registrasi/sqlstore: menyimpan spreading %d.%d: %w",
					itemSeq, coverageSeq, err)
			}
			if err := r.saveItems(ctx, exec, k, o.ID, coverageSeq, c.Item, now); err != nil {
				return fmt.Errorf("registrasi/sqlstore: menyimpan estimasi %d.%d: %w",
					itemSeq, coverageSeq, err)
			}
			if err := r.saveSettlement(ctx, exec, k.ID, o.ID, coverageSeq, c.Settlement); err != nil {
				return fmt.Errorf("registrasi/sqlstore: menyimpan adjustment %d.%d: %w",
					itemSeq, coverageSeq, err)
			}
		}

		if _, err := exec.ExecContext(ctx, loadQuery("coverage_tandai_sisa"),
			now, k.ID, itemSeq, len(o.Coverage)); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menandai sisa coverage: %w", err)
		}
	}

	if _, err := exec.ExecContext(ctx, loadQuery("objek_tandai_sisa"),
		now, k.ID, len(k.InsuredItem)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menandai sisa objek: %w", err)
	}
	return nil
}

// saveSpreading menuliskan pembagian risiko satu coverage.
//
// # Aturan tulisnya ditetapkan Work Owner, bukan disimpulkan
//
// Terhadap POOLDATA.T_CLAIM_SPREADING, aplikasi **hanya menyisipkan bila barisnya belum
// ada, dan tidak pernah menghapus** (Work Owner, 2026-09-26). Karena itu tidak ada
// pembaruan dan tidak ada penandaan sisa di sini — dua hal yang dilakukan tabel anak
// lainnya.
//
// # Baris yang ditandai dibuang petugas tidak disisipkan
//
// Tabelnya tidak punya kolom penanda, dan tidak ada proses hapus. Bila baris ber-`Removed`
// tetap disisipkan, ia akan terbaca kembali sebagai baris yang masih berlaku dan ikut
// terhitung pada aturan total 100% (`I-1`, `D-51`) — mengubah hasil validasi tanpa ada
// yang menyadarinya. Karena itu ia tidak ditulis sama sekali, sejalan dengan sistem lama
// yang membuang baris ber-`FlagDelete = "1"` sebelum perhitungan.
//
// # `FacOfferItem` tidak tersimpan
//
// Tabelnya tidak punya kolomnya. Ia dipakai gerbang kelengkapan Fac Out saat input
// (Group Panel `003`), dan rincian Fac Offer itu sendiri tinggal di snapshot polis
// (`D-04`), bukan di baris spreading. Dicatat sebagai keterbatasan yang diketahui.
func (r *ClaimStore) saveSpreading(
	ctx context.Context,
	exec executor,
	claimID, objectID string,
	coverageSeq int,
	daftar []registrasi.Spreading,
) error {
	coverageID := strconv.Itoa(coverageSeq)

	for n, s := range daftar {
		if s.Removed {
			continue
		}

		baris, err := exec.QueryContext(ctx, loadQuery("spreading_ada"),
			claimID, objectID, coverageID, s.TreatyKind)
		if err != nil {
			return err
		}
		ada := baris.Next()
		if err := baris.Err(); err != nil {
			_ = baris.Close()
			return err
		}
		_ = baris.Close()
		if ada {
			continue
		}

		// Share dibagi 10.000 menjadi persen. Pembagian ini AMAN meski lewat float64:
		// kolomnya NUMBER(9,6) dan nilainya berkisar 0–100, sehingga galat float64
		// (~1e-14) jauh di bawah satu satuan terkecil kolomnya (1e-6). Nilai yang sama
		// dibaca kembali lewat ROUND(... * 10000) — lihat spreading_daftar.
		if _, err := exec.ExecContext(ctx, loadQuery("spreading_sisip"),
			claimID, objectID, coverageID, s.TreatyKind, s.Name,
			float64(s.Share)/10_000, n+1,
		); err != nil {
			return err
		}
	}
	return nil
}

// upsert menjalankan UPDATE lebih dulu dan menyisipkan hanya bila tidak ada baris yang
// terpengaruh.
func upsert(ctx context.Context, exec executor, updateName string, updateArgs []any, insertName string, insertArgs []any) error {
	result, err := exec.ExecContext(ctx, loadQuery(updateName), updateArgs...)
	if err != nil {
		return err
	}
	row, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if row > 0 {
		return nil
	}
	_, err = exec.ExecContext(ctx, loadQuery(insertName), insertArgs...)
	return err
}

// Get mengembalikan klaim berdasarkan pengenal internalnya.
func (r *ClaimStore) Get(ctx context.Context, id string) (registrasi.Claim, error) {
	return r.getBy(ctx, "klaim_ambil", id)
}

// GetByNumber mengembalikan klaim berdasarkan nomor klaimnya.
func (r *ClaimStore) GetByNumber(ctx context.Context, number string) (registrasi.Claim, error) {
	return r.getBy(ctx, "klaim_ambil_per_nomor", number)
}

func (r *ClaimStore) getBy(ctx context.Context, queryName, value string) (registrasi.Claim, error) {
	exec := executorFrom(ctx, r.db)

	var (
		k                                  registrasi.Claim
		number, portal, line, businessType sql.NullString
		policyCurrency, insured, branch    sql.NullString
		lossDate, reportDate, receivedDate sql.NullTime
		location, chronology               sql.NullString
		rName, rPhone, rAddress            sql.NullString
		rRelation                          sql.NullInt64
		rRelationOther                     sql.NullString
		currency, slikNumber, exGratia     sql.NullString
		technicalPIC, rcvID                sql.NullString
		puclStatus                         sql.NullInt64
		processStatus, claimStatus         sql.NullString
		createdBy                          sql.NullString
		createdAt                          sql.NullTime
		flagNoll                           sql.NullString

		sobName, sob, branchName, busCode sql.NullString
		busName, prodKe, typeOfCoins      sql.NullString
		coinsName, coinsRole              sql.NullString
		shareASM                          sql.NullInt64
		policyLeader                      sql.NullString

		country, countryID, province, provinceID sql.NullString
		city, cityID, district, districtID       sql.NullString
		rw, rwID, postalCode                     sql.NullString
		customerPrinciple, suspiciousComment     sql.NullString
		emailLOD, recommendation, subjectEmail   sql.NullString
		salvageStatus                            sql.NullString
		analystTransferredAt                     sql.NullTime
	)

	row := exec.QueryRowContext(ctx, loadQuery(queryName), value)
	err := row.Scan(
		&k.ID, &number, &portal,
		&k.Policy.Number, &line, &businessType, &policyCurrency, &insured, &branch,
		&lossDate, &reportDate, &receivedDate,
		&location, &chronology,
		&rName, &rPhone, &rAddress, &rRelation, &rRelationOther,
		&currency, &slikNumber, &exGratia, &technicalPIC, &rcvID,
		&puclStatus,
		&processStatus, &claimStatus,
		&createdBy, &createdAt,
		&flagNoll,
		&sobName, &sob, &branchName, &busCode, &busName, &prodKe, &typeOfCoins,
		&coinsName, &coinsRole, &shareASM, &policyLeader,
		&country, &countryID, &province, &provinceID, &city, &cityID, &district, &districtID,
		&rw, &rwID, &postalCode, &customerPrinciple, &suspiciousComment,
		&emailLOD, &recommendation, &subjectEmail, &salvageStatus,
		&analystTransferredAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.Claim{}, registrasi.ErrClaimNotFound
	}
	if err != nil {
		return registrasi.Claim{}, fmt.Errorf("registrasi/sqlstore: membaca klaim: %w", err)
	}

	k.Number = number.String
	k.Portal = portal.String
	k.Policy.Line = registrasi.LineOfBusiness(line.String)
	k.Policy.BusinessType = businessType.String
	k.Policy.Currency = policyCurrency.String
	k.Policy.InsuredName = insured.String
	k.Policy.BranchCode = branch.String

	k.DateOfLoss = lossDate.Time
	k.ReportDate = reportDate.Time
	k.DateReceived = receivedDate.Time
	k.Location = location.String
	k.Chronology = chronology.String

	k.Reporter = registrasi.Reporter{
		Name:          rName.String,
		Phone:         rPhone.String,
		Address:       rAddress.String,
		Relation:      int(rRelation.Int64),
		OtherRelation: rRelationOther.String,
	}

	k.Currency = currency.String
	k.SLIKNumber = slikNumber.String
	k.ExGratia = fromYesNo(exGratia.String)
	k.TechnicalPIC = technicalPIC.String
	k.RCVID = rcvID.String
	k.PUCLStatus = int(puclStatus.Int64)

	k.ProcessStatus = registrasi.ProcessStatus(processStatus.String)
	k.ClaimStatus = registrasi.ClaimStatus(claimStatus.String)

	k.CreatedBy = createdBy.String
	k.CreatedAt = createdAt.Time
	k.LargeLossNoticed = fromFlagNOLL(flagNoll.String)

	k.Policy.SourceOfBusinessName = sobName.String
	k.Policy.SourceOfBusiness = sob.String
	k.Policy.BranchName = branchName.String
	k.Policy.BusinessCode = busCode.String
	k.Policy.BusinessName = busName.String
	k.Policy.ProdKe = prodKe.String
	k.Policy.TypeOfCoins = typeOfCoins.String
	k.Policy.PolicyLeader = policyLeader.String
	k.Area = registrasi.Area{
		Country: country.String, CountryID: countryID.String,
		Province: province.String, ProvinceID: provinceID.String,
		City: city.String, CityID: cityID.String,
		District: district.String, DistrictID: districtID.String,
		RW: rw.String, RWID: rwID.String,
		PostalCode: postalCode.String,
	}
	k.CustomerPrinciple = customerPrinciple.String
	k.SuspiciousComment = suspiciousComment.String
	k.EmailLOD = emailLOD.String
	k.RemarkRecommendation = recommendation.String
	k.SubjectEmail = subjectEmail.String
	k.SalvageStatus = strings.TrimSpace(salvageStatus.String)
	if analystTransferredAt.Valid {
		k.AnalystTransferredAt = analystTransferredAt.Time
	}

	k.Policy.Coinsurance = registrasi.Coinsurance{
		Name:     coinsName.String,
		Role:     coinsRole.String,
		ShareASM: registrasi.Percent(shareASM.Int64),
		HasShare: shareASM.Valid,
	}

	if err := r.restoreDropped(ctx, exec, &k); err != nil {
		return registrasi.Claim{}, err
	}
	if err := r.loadTree(ctx, exec, &k); err != nil {
		return registrasi.Claim{}, err
	}

	// Nama Status Klaim dari master V_STS_CLAIM, untuk ditampilkan seperti layar Pega
	// ("Register", bukan 1147). Kode yang tidak ada di master dibiarkan tanpa nama.
	if code := strings.TrimSpace(string(k.ClaimStatus)); code != "" {
		var name sql.NullString
		err := exec.QueryRowContext(ctx, loadQuery("status_nama"), code).Scan(&name)
		if err != nil && err != sql.ErrNoRows {
			return registrasi.Claim{}, fmt.Errorf("registrasi/sqlstore: membaca nama status klaim: %w", err)
		}
		k.ClaimStatusName = strings.TrimSpace(name.String)
	}
	return k, nil
}

// restoreDropped mengisi medan klaim yang kolomnya DIBUANG dari POOLDATA.T_CLAIM_PNC.
//
// Work Owner mengubah T_CLAIM_PNC pada 2026-09-26 15:31: dari 97 menjadi 82 kolom.
// Empat belas kolom tambahan modul ini dibuang, sedangkan tujuh dipertahankan — pilihan
// yang disengaja, dan modul ini menyesuaikan diri padanya, bukan menambahkannya kembali.
//
// Medan yang kolomnya hilang dibagi tiga, menurut apakah ia masih dapat diketahui:
//
//	diturunkan dari sumber lain    CurrentStage            tugas terbuka di CPNC_TUGAS
//	                               CoverageStart/End,      dokumen polis (D-04), dibaca
//	                               Declaration,            lewat kueri yang sama dengan
//	                               CreditGuarantee         pembukaan klaim
//	                               ProgressPositionStatus  selesai bila prosesnya selesai
//
//	nilai tetap yang memang tetap  ClaimFlag               selalu "0" — tidak ada jalur
//	                                                       yang mengubahnya
//
//	masukan per permintaan         ComplianceTransfer,     diisi dari masukan permintaan
//	                               RequestReturn           yang sama, tepat sebelum alur
//	                                                       memilih tahap berikutnya
//
// Dua medan TIDAK dapat dipulihkan: EstimateValue (NILAI_ESTIMASI_SEN) dan Reporter.Email
// (PELAPOR_EMAIL). Tidak ada kolom maupun sumber lain yang memuatnya. Estimasi dikirim
// ulang layar Input Register setiap simpan, sehingga aturan Large Loss tetap berjalan;
// yang hilang adalah isian awalnya dari berkas RCV. Keduanya dicatat, bukan dikarang.
func (r *ClaimStore) restoreDropped(ctx context.Context, exec executor, k *registrasi.Claim) error {
	k.ClaimFlag = registrasi.FlagUnset

	// Posisi progres bernilai Done tepat saat alur mencapai ujungnya — saat yang sama
	// ProcessStatus berhenti menjadi Running dan tahapnya dikosongkan (usecase/service.go).
	k.ProgressPositionStatus = registrasi.PositionInProgress
	if k.ProcessStatus != "" && k.ProcessStatus != registrasi.ProcessRunning {
		k.ProgressPositionStatus = registrasi.PositionDone
	}

	// Tahap kini = tahap tugas yang masih terbuka. Klaim yang alurnya selesai tidak punya
	// tugas terbuka, dan tahapnya memang kosong.
	baris, err := exec.QueryContext(ctx, loadQuery("tugas_terbuka_klaim"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca tahap klaim dari tugasnya: %w", err)
	}
	// Hanya baris pertama yang dibaca — tugas terbuka tertua (urutan DIBUAT_PADA, ID).
	if baris.Next() {
		var (
			id, klaimID, nomor, tahap, antrean, workbasket, pemilik sql.NullString
			dibuat, diambil, selesai                                sql.NullTime
			alasan                                                  sql.NullString
		)
		if err := baris.Scan(&id, &klaimID, &nomor, &tahap, &antrean, &workbasket, &pemilik,
			&dibuat, &diambil, &selesai, &alasan); err != nil {
			_ = baris.Close()
			return fmt.Errorf("registrasi/sqlstore: membaca tugas terbuka: %w", err)
		}
		k.CurrentStage = tahap.String
	}
	if err := baris.Err(); err != nil {
		_ = baris.Close()
		return fmt.Errorf("registrasi/sqlstore: menelusuri tugas terbuka: %w", err)
	}
	_ = baris.Close()

	// Periode dan jenis polis dibaca ulang dari dokumen polisnya. Polis yang tidak lagi
	// ditemukan TIDAK menggagalkan pembukaan klaim: klaimnya tetap dapat dilihat, dan
	// gerbang validasi tanggal yang akan menyatakan periodenya tidak diketahui.
	if strings.TrimSpace(k.Policy.Number) == "" {
		return nil
	}
	polis, err := (&PolicyRepo{db: r.db}).Get(ctx, k.Policy.Number)
	switch {
	case errors.Is(err, registrasi.ErrPolicyNotFound):
		return nil
	case err != nil:
		return err
	}
	k.Policy.CoverageStart = polis.CoverageStart
	k.Policy.CoverageEnd = polis.CoverageEnd
	k.Policy.Declaration = polis.Declaration
	k.Policy.CreditGuarantee = polis.CreditGuarantee
	return nil
}

// loadTree mengisi objek, coverage, dan spreading dalam TIGA kueri, bukan satu kueri
// per objek.
//
// Satu klaim kebakaran besar dapat memuat puluhan objek dengan coverage masing-masing;
// memuatnya satu per satu berarti puluhan perjalanan bolak-balik ke basis data untuk
// membuka satu layar.
func (r *ClaimStore) loadTree(ctx context.Context, exec executor, k *registrasi.Claim) error {
	// Pohon klaim dijodohkan lewat OBJECTID dan OBJECTCOVERAGEID — kolom `NOT NULL` di
	// ketiga tabel — BUKAN lewat URUTAN.
	//
	// URUTAN dan URUTAN_OBJEK adalah kolom yang ditambahkan proyek ini, dan baris yang
	// ditulis Pega tidak pernah mengisinya: 2.611 dari 2.726 baris objek aktif kosong,
	// begitu pula 2.598 dari 2.635 baris coverage (diukur 2026-10-07). Memindainya ke `int`
	// biasa membuat 1.665 dari 1.686 klaim GAGAL DIBUKA — bukan salah sebagian, melainkan
	// seluruh layar yang memuat klaim.
	//
	// URUTAN tetap dibaca, tetapi hanya sebagai jalur cadangan bagi baris baru yang
	// OBJECTID-nya tidak terjodohkan.
	itemIndexByID := map[string]int{}
	itemIDBySeq := map[int64]string{}

	row, err := exec.QueryContext(ctx, loadQuery("objek_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca objek: %w", err)
	}
	defer func() { _ = row.Close() }()
	for row.Next() {
		var (
			seq            sql.NullInt64
			itemID         string
			name, location sql.NullString
			job, birth     sql.NullString
		)
		if err := row.Scan(&seq, &itemID, &name, &location, &job, &birth); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris objek: %w", err)
		}
		id := strings.TrimSpace(itemID)
		// Baris PERTAMA yang menang, bukan terakhir. Pada 4 klaim terdapat dua baris objek
		// ber-OBJECTID sama — dan pada 4 dari 6 pasangnya nama objeknya pun berbeda, jadi
		// ia benar-benar dua objek. Tanpa aturan ini, coverage-nya menempel ke baris mana
		// pun yang kebetulan terbaca belakangan.
		//
		// Yang pertama dipilih karena `ORDER BY URUTAN, OBJECTID` membuatnya deterministik:
		// baris ber-URUTAN terisi selalu mendahului baris warisan.
		if _, taken := itemIndexByID[id]; !taken {
			itemIndexByID[id] = len(k.InsuredItem)
		}
		if seq.Valid {
			itemIDBySeq[seq.Int64] = id
		}
		k.InsuredItem = append(k.InsuredItem, registrasi.InsuredItem{
			ID: itemID, Name: name.String, Location: location.String,
			Job: strings.TrimSpace(job.String), DateOfBirth: strings.TrimSpace(birth.String),
		})
	}
	if err := row.Err(); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menelusuri objek: %w", err)
	}
	_ = row.Close()

	// Kunci peta ini PASANGAN TEKS — OBJECTID induk dan OBJECTCOVERAGEID miliknya sendiri —
	// supaya spreading dapat menemukannya dengan kolom yang memang ada di tabelnya.
	coverageIndex := map[[2]string]int{}

	coverageRow, err := exec.QueryContext(ctx, loadQuery("coverage_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca coverage: %w", err)
	}
	defer func() { _ = coverageRow.Close() }()
	for coverageRow.Next() {
		var (
			parentID, coverageKey string
			itemSeq, seq          sql.NullInt64
			coverageID, cause     sql.NullString
			coverageName          sql.NullString
			tsi                   sql.NullInt64
			analystFlag           sql.NullInt64
		)
		if err := coverageRow.Scan(&parentID, &coverageKey, &itemSeq, &seq,
			&coverageID, &cause, &tsi, &coverageName, &analystFlag); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris coverage: %w", err)
		}
		// URUTAN_OBJEK DICOBA LEBIH DULU bila terisi: ia kunci yang lebih tepat, sebab
		// OBJECTID terbukti dapat kembar dalam satu klaim. Baris warisan tidak punya
		// URUTAN_OBJEK sama sekali, dan untuk mereka OBJECTID-lah satu-satunya jalan.
		parent := strings.TrimSpace(parentID)
		var (
			i  int
			ok bool
		)
		if itemSeq.Valid {
			if id, found := itemIDBySeq[itemSeq.Int64]; found {
				parent = id
				i, ok = itemIndexByID[id]
			}
		}
		if !ok {
			i, ok = itemIndexByID[parent]
		}
		if !ok {
			// Coverage yang objek induknya sudah ditandai terhapus. Ia dilewati, bukan
			// dianggap galat: penandaan induk memang membuat anaknya tidak lagi
			// terlihat.
			continue
		}
		coverageIndex[[2]string{parent, strings.TrimSpace(coverageKey)}] = len(k.InsuredItem[i].Coverage)
		k.InsuredItem[i].Coverage = append(k.InsuredItem[i].Coverage, registrasi.Coverage{
			ID:          coverageID.String,
			Name:        coverageName.String,
			CauseOfLoss: cause.String,
			TSI:         registrasi.Money(tsi.Int64),

			AnalystTransferred: analystFlag.Valid && analystFlag.Int64 == 1,
		})
	}
	if err := coverageRow.Err(); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menelusuri coverage: %w", err)
	}
	_ = coverageRow.Close()

	spreadingRow, err := exec.QueryContext(ctx, loadQuery("spreading_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca spreading: %w", err)
	}
	defer func() { _ = spreadingRow.Close() }()
	for spreadingRow.Next() {
		var (
			objectID, coverageID sql.NullString
			seq                  int
			kind, name           sql.NullString
			share                sql.NullInt64
		)
		if err := spreadingRow.Scan(&objectID, &coverageID, &seq, &kind, &name, &share); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris spreading: %w", err)
		}

		// T_CLAIM_SPREADING menyimpan OBJECTID dan OBJECTCOVERAGEID — kunci yang sama
		// dengan yang dipakai kedua peta di atas. Objek yang sudah ditandai terhapus tidak
		// ada di sana, sehingga spreading di bawahnya ikut terlewati, sama seperti
		// coverage.
		//
		// Sebelumnya OBJECTCOVERAGEID DIURAI menjadi angka lalu dicocokkan ke URUTAN
		// coverage. Pencocokan itu tidak pernah berhasil pada baris warisan — URUTAN-nya
		// kosong — sehingga spreading-nya hilang tanpa satu pun galat.
		parent := strings.TrimSpace(objectID.String)
		i, ok := itemIndexByID[parent]
		if !ok {
			continue
		}
		j, ok := coverageIndex[[2]string{parent, strings.TrimSpace(coverageID.String)}]
		if !ok {
			continue
		}

		// Removed dan FacOfferItem tidak punya kolom di tabel ini; baris ber-Removed
		// memang tidak pernah disisipkan. Lihat saveSpreading.
		k.InsuredItem[i].Coverage[j].Spreading = append(k.InsuredItem[i].Coverage[j].Spreading, registrasi.Spreading{
			TreatyKind: kind.String,
			Name:       name.String,
			Share:      registrasi.Percent(share.Int64),
		})
	}
	if err := spreadingRow.Err(); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menelusuri spreading: %w", err)
	}
	_ = spreadingRow.Close()

	// coverageAt menjodohkan baris rincian item dan settlement ke coverage-nya, memakai
	// kunci teks yang sama dengan kedua peta di atas.
	coverageAt := func(objectID, coverageID string) *registrasi.Coverage {
		parent := strings.TrimSpace(objectID)
		i, ok := itemIndexByID[parent]
		if !ok {
			return nil
		}
		j, ok := coverageIndex[[2]string{parent, strings.TrimSpace(coverageID)}]
		if !ok {
			return nil
		}
		return &k.InsuredItem[i].Coverage[j]
	}
	if err := loadItems(ctx, exec, k, coverageAt); err != nil {
		return err
	}
	if err := loadSettlement(ctx, exec, k.ID, coverageAt); err != nil {
		return err
	}
	receivers, err := loadReceivers(ctx, exec, k.ID)
	if err != nil {
		return err
	}
	k.Receiver = receivers
	return nil
}

// FindDuplicates mencari klaim lain yang memenuhi salah satu kunci duplikasi.
func (r *ClaimStore) FindDuplicates(ctx context.Context, key []registrasi.DuplicateKey, exceptID string) ([]registrasi.DuplicateClaim, error) {
	exec := executorFrom(ctx, r.db)

	var result []registrasi.DuplicateClaim
	seen := map[string]bool{}

	for _, dupKey := range key {
		useLocation := 0
		if dupKey.Location != "" {
			useLocation = 1
		}
		useCause := 0
		if dupKey.CauseOfLoss != "" {
			useCause = 1
		}

		row, err := exec.QueryContext(ctx, loadQuery("klaim_cari_ganda"),
			dupKey.PolicyNumber, exceptID, dupKey.InsuredItemID,
			useLocation, dupKey.Location,
			useCause, dupKey.CauseOfLoss,
		)
		if err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: memeriksa klaim ganda: %w", err)
		}

		for row.Next() {
			var number, insuredItem string
			if err := row.Scan(&number, &insuredItem); err != nil {
				_ = row.Close()
				return nil, fmt.Errorf("registrasi/sqlstore: membaca baris klaim ganda: %w", err)
			}
			marker := number + "|" + insuredItem
			if seen[marker] {
				continue
			}
			seen[marker] = true
			result = append(result, registrasi.DuplicateClaim{Number: number, InsuredItem: insuredItem})
		}
		if err := row.Err(); err != nil {
			_ = row.Close()
			return nil, fmt.Errorf("registrasi/sqlstore: menelusuri klaim ganda: %w", err)
		}
		_ = row.Close()
	}
	return result, nil
}

// calendarDateOrNil mengikat sebuah TANGGAL KALENDER ke kolom Oracle bertipe `DATE`.
//
// # Kenapa WIB, bukan UTC
//
// `DATE` Oracle tidak menyimpan zona — ia menyimpan jam dinding apa adanya. Mengikat waktu
// UTC karena itu menuliskan jam dinding UTC: tanggal kejadian 15 Juli (tengah malam WIB)
// tersimpan sebagai `2026-07-14 17:00`. Saat dibaca kembali, driver memasang zona sesi
// (WIB) pada jam dinding itu, sehingga tanggalnya menjadi **14 Juli**.
//
// Sehari hilang, tanpa galat, hanya pada tanggal-tanggal yang jamnya di bawah 07:00 WIB —
// yakni SELURUH tanggal kalender, karena semuanya bertengah malam. Itu `R-12`, dan ia
// terbukti terhadap Oracle: berkas menyimpan `2026-07-15T00:00+07:00`, klaimnya terbaca
// kembali `2026-07-14`.
//
// Mengubahnya ke WIB lebih dulu membuat jam dinding yang tersimpan sama dengan tanggal
// bisnisnya, sehingga tulis dan baca setangkup. Ia juga menyamai apa yang sudah ada di
// tabel: baris Pega dan baris modul Receive Document sama-sama menyimpan tengah malam WIB.
//
// # Yang TIDAK memakai ini
//
// Kolom `TIMESTAMP(6)` — `DIBUAT_PADA`, `DIUBAH_PADA`, `DIHAPUS_PADA` — menyimpan INSTAN,
// bukan tanggal kalender, dan tetap UTC sesuai `F-5`. Membedakan keduanya adalah inti
// persoalannya: yang satu titik waktu, yang satu hari kerja.
func calendarDateOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.In(clock.ZoneWIB)
}

// shareOrNil mengikat bagian Sinar Mas ke SHAREASM, dalam PERSEN seperti baris Pega.
//
// Share yang tidak diketahui ditulis NULL, bukan nol — Pega pun meninggalkannya NULL
// (lihat DeriveCoinsurance). Pembagian lewat float64 aman di rentang 0–100: galatnya
// ~1e-14, dan pembacaan kembali memakai ROUND(SHAREASM * 10000).
func shareOrNil(c registrasi.Coinsurance) any {
	if !c.HasShare {
		return nil
	}
	return float64(c.ShareASM) / 10_000
}

// registerMoment mengikat saat registrasi ke kolom DATE dalam jam dinding WIB.
func registerMoment(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.In(clock.ZoneWIB)
}

// flag mengubah penanda menjadi 1 atau 0 untuk kolom NUMBER penanda warisan.
func flag(b bool) int {
	if b {
		return 1
	}
	return 0
}

func timeOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

func timePtrOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC()
}

// emptyTextAsNil menjaga kolom NOMOR tetap NULL selama klaim belum bernomor.
//
// Ini bukan kerapian: kolomnya berada di bawah UNIQUE, dan Oracle memperlakukan NULL
// sebagai "tidak diketahui" sehingga banyak baris boleh sama-sama NULL. Bila string
// kosong yang tersimpan, klaim kedua yang belum bernomor akan ditolak constraint.
func emptyTextAsNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var _ registrasi.ClaimRepo = (*ClaimStore)(nil)
