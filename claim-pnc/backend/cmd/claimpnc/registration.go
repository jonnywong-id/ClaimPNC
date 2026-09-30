package main

import (
	"database/sql"
	"log/slog"
	"os"
	"strings"

	komitesql "claim-pnc/internal/komite/repo/sqlstore"
	komiteusecase "claim-pnc/internal/komite/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/random"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/facesheetpdf"
	"claim-pnc/internal/registrasi/dlapdf"
	"claim-pnc/internal/registrasi/lodpdf"
	"claim-pnc/internal/registrasi/plapdf"
	"claim-pnc/internal/registrasi/repo/cashierlink"
	"claim-pnc/internal/registrasi/repo/komitelink"
	registrasimemory "claim-pnc/internal/registrasi/repo/memory"
	"claim-pnc/internal/registrasi/repo/premiumlink"
	registrasisql "claim-pnc/internal/registrasi/repo/sqlstore"
	registrasiusecase "claim-pnc/internal/registrasi/usecase"
)

// assembleRegistration menyusun modul Registrasi Klaim di balik seam-nya masing-masing.
//
// # Tanpa Oracle, SELURUH seam memakai penyimpanan memori
//
// Bukan sebagian. Modul yang separuh membaca data nyata dan separuh membaca contoh akan
// menghasilkan klaim yang sebagian benar — dan itu lebih sulit dikenali daripada modul
// yang jelas berjalan atas data contoh.
//
// # Dengan Oracle, sepuluh dari sebelas seam membaca data nyata
//
// Empat di antaranya baru tersambung 2026-09-24, dan sumbernya ternyata SUDAH ADA di
// basis data meski Steering mencatatnya menunggu DBA:
//
//   - PolicyRepo         → POOLDATA.JSON_POLIS (`D-04`; hanya dibaca, tidak pernah ditulis)
//   - ExchangeRateSource → POOLDATA.M_CURRENCYSTANDARD (`ADR-0015`, `D-48`)
//   - Parameter          → POOLDATA.M_PARAMETER (`F-4`)
//   - Assigner           → beban PIC Teknik dari tabel yang sama seperti Pega (`R-04`)
//
// Yang TETAP di memori hanyalah `IDGenerator`, dan ia memang tidak pernah butuh basis
// data.
//
// # Yang masih kosong disebutkan, bukan ditambal
//
// Dua seam di atas dapat menjawab "tidak ada isinya" dan itu bukan kegagalan teknis:
// daftar penerima Notice of Large Losses (`PNC.PENERIMA_KERUGIAN_BESAR`) belum diisi Work
// Owner, dan tiga aturan routing tidak ada di export (`R-04`) sehingga pemilihan petugas
// direkonstruksi dari kueri beban — bukan dibaca dari rule aslinya. Keduanya diperingatkan
// saat start supaya tidak dikira sudah lengkap.
func assembleRegistration(
	db *sql.DB,
	logger *slog.Logger,
	documents registrasi.DocumentUploader,
	catalog premiumlink.ServiceCatalog,
	cashier config.Cashier,
) (*registrasiusecase.Service, error) {
	idGenerator := registrasimemory.IDGenerator{}
	clock := clock.System{}

	options := registrasiusecase.Options{
		IDGenerator:       idGenerator,
		Clock:             clock,
		FaceSheetRenderer: facesheetpdf.Renderer{},
		PLARenderer:       plapdf.Renderer{},
		DLARenderer:       dlapdf.Renderer{},
		LODRenderer:       lodpdf.Renderer{},
	}

	if db != nil {
		options.ClaimRepo = registrasisql.NewClaimStore(db)
		options.TaskRepo = registrasisql.NewTaskStore(db)
		options.NumberIssuer = registrasisql.NewNumberIssuer(db)
		options.AuditRecorder = registrasisql.NewAuditRecorder(db, idGenerator)
		options.Notifier = registrasisql.NewNotifier(db, idGenerator)
		options.UnitOfWork = registrasisql.NewUnitOfWork(db)

		options.PolicyRepo = registrasisql.NewPolicyRepo(db)
		options.Parameter = registrasisql.NewParameter(db)
		options.ExchangeRateSource = registrasisql.NewExchangeRateSource(db)
		options.Assigner = registrasisql.NewAssigner(db)
		options.ClaimReportLink = registrasisql.NewClaimReportLink(db)
		options.AreaDirectory = registrasisql.NewAreaDirectory(db)
		options.PolicyItems = registrasisql.NewPolicyItems(db)
		options.CurrencyDirectory = registrasisql.NewCurrencyDirectory(db)
		options.ItemOptions = options.PolicyItems.(registrasi.ItemOptionSource)
		options.ClaimRecords = registrasisql.NewClaimRecords(db)
		options.FaceSheet = registrasisql.NewFaceSheetStore(db)
		options.PLA = registrasisql.NewPLAStore(db)
		options.DLA = registrasisql.NewDLAStore(db)
		options.Cashier = registrasisql.NewCashierStore(db)
		options.Groups = registrasisql.NewGroupStore(db)
		options.Inbox = registrasisql.NewInboxEntryStore(db)
		options.Accounts = registrasisql.NewAccountDirectory(db)
		options.Committees = registrasisql.NewCommitteeStore(db)
		// Unggah Dokumen: berkas lewat modul dokumen penunjang, baris di DATA_ATTACHFILE.
		options.Documents = documents
		options.Attachments = registrasisql.NewAttachmentStore(db)
		options.Acceptance = registrasisql.NewAcceptanceStore(db)
		// Transfer Kasir: alamat dari POOLDATA.GCNM_CONNECT_REST (KASIRPAID…), kredensial dari
		// KASIR_USER / KASIR_PASSWORD.
		if catalog != nil {
			options.CashierGateway = cashierlink.New(catalog, cashier.User, cashier.Password, nil)
			if cashier.User == "" {
				logger.Warn("Transfer Kasir tanpa kredensial — isi KASIR_USER dan KASIR_PASSWORD bila Kasir menuntutnya")
			}
		} else {
			options.CashierGateway = registrasimemory.NewCashier()
			logger.Warn("Transfer Kasir memakai layanan contoh yang tidak pernah berhasil — daftar alamat layanan tidak tersedia")
		}
		// Status premi (akseptasi): alamat layanan dari POOLDATA.GCNM_CONNECT_REST.
		if catalog != nil {
			options.Premium = premiumlink.New(catalog, nil)
		} else {
			options.Premium = registrasimemory.NewPremium()
			logger.Warn("layanan status premi memakai jawaban contoh — daftar alamat layanan tidak tersedia")
		}

		// Penjenjangan komite membaca POOLDATA.EMAILKOMITE lewat modul komite, di atas
		// koneksi yang sama dengan klaimnya.
		tiering, err := komiteusecase.NewService(komiteusecase.Options{
			Repo:       komitesql.NewRepo(db),
			Randomizer: random.System{},
		})
		if err != nil {
			return nil, err
		}
		options.CommitteeTiering = komitelink.New(tiering)

		logger.Warn("modul registrasi berjalan, dengan dua sumber yang belum lengkap",
			slog.String("penerima_kerugian_besar",
				"M_PARAMETER PNC.PENERIMA_KERUGIAN_BESAR belum diisi — Notice of Large Losses "+
					"terbit tanpa penerima"),
			slog.String("routing",
				"aturan pemilihan petugas direkonstruksi dari kueri beban; tiga router "+
					"tidak ada di export (R-04)"),
		)
	} else {
		store := registrasimemory.NewStore()
		options.ClaimRepo = store
		options.TaskRepo = store.TaskRepo()
		options.NumberIssuer = registrasimemory.NewNumberIssuer()
		options.AuditRecorder = store
		options.Notifier = store
		options.UnitOfWork = store

		options.PolicyRepo = registrasimemory.NewPolicyStore(registrasimemory.SamplePolicies(clock.Now())...)
		options.Parameter = registrasimemory.NewParameter()
		options.ExchangeRateSource = registrasimemory.NewExchangeRateSource()
		options.Assigner = registrasimemory.NewAssigner(registrasimemory.SampleTeams())
		options.ClaimReportLink = registrasimemory.NewClaimReportLink()
		options.AreaDirectory = registrasimemory.NewAreaDirectory()
		options.PolicyItems = registrasimemory.NewPolicyItems(registrasimemory.SamplePolicyItems())
		options.CurrencyDirectory = registrasimemory.CurrencyDirectory{}
		options.ItemOptions = options.PolicyItems.(registrasi.ItemOptionSource)
		records := registrasimemory.SampleClaimRecords()
		options.ClaimRecords = records
		options.Attachments = records
		options.Documents = documents
		if options.Documents == nil {
			options.Documents = &registrasimemory.DocumentUploader{}
		}
		options.FaceSheet = registrasimemory.NewFaceSheet()
		options.PLA = registrasimemory.NewPLA()
		options.DLA = registrasimemory.NewDLA()
		cashierMemory := registrasimemory.NewCashier()
		options.Cashier = cashierMemory
		options.CashierGateway = cashierMemory
		options.Groups = registrasimemory.Groups{}
		options.Inbox = registrasimemory.NewInboxEntries()
		options.Accounts = registrasimemory.NewAccounts()
		options.Committees = registrasimemory.NewCommittees()
		options.CommitteeTiering = registrasimemory.NewCommitteeTiering()
		options.Acceptance = registrasimemory.NewAcceptance()
		options.Premium = registrasimemory.NewPremium()

		logger.Warn("modul registrasi berjalan ATAS DATA CONTOH — tidak ada koneksi Oracle",
			slog.String("akibat", "klaim yang dibuat tidak tersimpan dan polisnya karangan"),
		)
	}

	return registrasiusecase.NewService(options)
}

// userRoles membaca peran pemanggil dari lingkungan.
//
// # Ini sakelar sementara, dan alasannya perlu dibaca sebelum dipakai
//
// Tabel peran dan izin menu adalah `TKT-F3-004`, yang masih terhalang artefak; sampai ia
// ada, aplikasi tidak punya sumber peran sama sekali. Itu bukan sekadar kekurangan
// kosmetik: satu keputusan alur Register benar-benar bergantung pada peran pemanggil —
// rule `IsAnalystDoctor` pada `Decision7` menguji `AccessGroup.pyAccessGroup`, bukan data
// klaim.
//
// Tanpa sumber peran, cabang itu tidak akan pernah benar, dan SELURUH klaim akan berakhir
// di tahap RCLDokter. Perbedaan itu harus dapat dilihat dan diuji sekarang, bukan setelah
// `TKT-F3-004` selesai berbulan-bulan lagi.
//
// Karena itu: satu variabel lingkungan, dibaca sekali, dipakai untuk SELURUH pengguna.
// Ia bukan otorisasi — ia tidak menjaga apa pun, dan tidak boleh dikira menjaga sesuatu.
// Begitu `TKT-F3-004` selesai, fungsi ini dihapus dan peran datang dari sesi pengguna.
func userRoles() []string {
	body := strings.TrimSpace(os.Getenv("PERAN_PENGGUNA"))
	if body == "" {
		return nil
	}
	var result []string
	for _, p := range strings.Split(body, ",") {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}

// userWorkbaskets menyebut antrean bersama yang boleh dilihat pemanggil.
//
// Sama seperti peran, daftarnya kelak datang dari `TKT-F3-004`. Sampai itu ada, seluruh
// pengguna melihat ketiga antrean alur Register — pilihan yang disengaja: menyembunyikan
// antrean akan membuat klaim tampak hilang, dan klaim yang tampak hilang adalah kelas
// cacat yang paling mahal ditemukan setelah rilis.
func userWorkbaskets() []string {
	return []string{
		registrasi.WorkbasketRCLPUCL,
		registrasi.WorkbasketInvestigator,
		registrasi.WorkbasketCompliance,
	}
}
