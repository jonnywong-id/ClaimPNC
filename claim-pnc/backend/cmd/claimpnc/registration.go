package main

import (
	"database/sql"
	"log/slog"
	"os"
	"strings"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
	registrasimemory "claim-pnc/internal/registrasi/repo/memory"
	registrasisql "claim-pnc/internal/registrasi/repo/sqlstore"
	registrasiusecase "claim-pnc/internal/registrasi/usecase"
)

// assembleRegistration menyusun modul Registrasi Klaim di balik seam-nya masing-masing.
//
// # Kenapa sebagian seam SELALU memakai penyimpanan memori
//
// Enam seam modul ini punya pengisi SQL: klaim, tugas, nomor klaim, jejak audit,
// pemberitahuan, dan batas transaksi. Keenamnya menulis ke tabel yang migrasi `0002`
// buat — tabel milik aplikasi ini sendiri.
//
// Lima seam sisanya TIDAK punya pengisi SQL, dan ketiadaannya bukan kelalaian:
//
//   - PolicyRepo   → sumbernya GISFW lewat modul `B-1` yang belum ada (`ADR-0006`).
//   - Parameter   → sumbernya master `F-4` yang belum ada (`TKT-F4-003`, `TKT-F4-005`).
//   - ExchangeRateSource  → sumbernya master kurs `TKT-F4-004` yang belum ada (`ADR-0015`).
//   - Penugasan   → tiga aturan routing tidak ada di export (`R-04`), dan daftar
//     petugasnya adalah data yang belum diberikan.
//   - IDGenerator   → tidak pernah butuh basis data.
//
// Menambal kelimanya dengan tabel karangan akan menyembunyikan bahwa modul ini belum
// lengkap. Yang dilakukan sebagai gantinya: aplikasi MEMPERINGATKAN saat start, dengan
// menyebut persis apa yang belum nyata.
func assembleRegistration(db *sql.DB, logger *slog.Logger) (*registrasiusecase.Service, error) {
	idGenerator := registrasimemory.IDGenerator{}
	clock := clock.System{}

	options := registrasiusecase.Options{
		PolicyRepo:         registrasimemory.NewPolicyStore(registrasimemory.SamplePolicies(clock.Now())...),
		Parameter:          registrasimemory.NewParameter(),
		ExchangeRateSource: registrasimemory.NewExchangeRateSource(),
		Assigner:           registrasimemory.NewAssigner(registrasimemory.SampleTeams()),
		IDGenerator:        idGenerator,
		Clock:              clock,
	}

	if db != nil {
		options.ClaimRepo = registrasisql.NewClaimStore(db)
		options.TaskRepo = registrasisql.NewTaskStore(db)
		options.NumberIssuer = registrasisql.NewNumberIssuer(db)
		options.AuditRecorder = registrasisql.NewAuditRecorder(db, idGenerator)
		options.Notifier = registrasisql.NewNotifier(db, idGenerator)
		options.UnitOfWork = registrasisql.NewUnitOfWork(db)
	} else {
		store := registrasimemory.NewStore()
		options.ClaimRepo = store
		options.TaskRepo = store.TaskRepo()
		options.NumberIssuer = registrasimemory.NewNumberIssuer()
		options.AuditRecorder = store
		options.Notifier = store
		options.UnitOfWork = store
	}

	logger.Warn("modul registrasi berjalan dengan sumber data yang belum lengkap",
		slog.String("polis", "contoh di memori — B-1 belum ada"),
		slog.String("ambang_dan_penerima", "nilai bawaan — master F-4 belum ada"),
		slog.String("kurs", "hanya IDR — master kurs belum ada"),
		slog.String("routing", "tim contoh — tiga aturan routing tidak ada di export (R-04)"),
	)

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
