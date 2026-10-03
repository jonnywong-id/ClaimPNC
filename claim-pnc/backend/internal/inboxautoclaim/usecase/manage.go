// Package usecase mengorkestrasi modul Inbox Auto Claim.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, memanggil Repo, dan menyusun berkas ekspor. Ia tidak
// tahu apa pun tentang HTTP maupun SQL.
package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"sync"

	"claim-pnc/internal/inboxautoclaim"
)

// Service adalah pintu masuk seluruh perkara Inbox Auto Claim.
type Service struct {
	repoSelector inboxautoclaim.RepoSelector
	premium      inboxautoclaim.PremiumChecker
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector inboxautoclaim.RepoSelector

	// Premium memeriksa status premi setiap baris unggahan. Wajib: Work Owner menetapkan
	// (2026-09-29) SEMUA baris di ketiga tab dicek preminya, sehingga rakitan tanpa
	// pemeriksa premi harus gagal saat start — bukan diam-diam meloloskan setiap baris.
	Premium inboxautoclaim.PremiumChecker
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxautoclaim/usecase: RepoSelector wajib diisi")
	}
	if o.Premium == nil {
		return nil, errors.New("inboxautoclaim/usecase: pemeriksa premi wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, premium: o.Premium}, nil
}

// requireSource menolak tab yang tidak dikenal, termasuk NILAI KOSONG.
//
// Tanpa ini, BatchFilter{} yang lupa mengisi tab akan membaca senarai kosong pada
// penyimpanan memori dan menghasilkan "tidak ada data" yang tampak wajar — padahal
// sebabnya bukan data melainkan tab yang tidak disebut. Di sqlstore akibatnya lebih
// keras (panic saat mencari kueri), dan dua penyimpanan yang berbeda perilakunya pada
// masukan yang sama adalah cacat tersendiri.
//
// Ini pagar yang sama dengan penolakan portal pada R-20: yang tidak disebut ditolak,
// bukan ditebak.
func requireSource(source inboxautoclaim.Source) error {
	if _, exists := source.Info(); !exists {
		return fmt.Errorf("%w: %q", inboxautoclaim.ErrUnknownSource, source)
	}
	return nil
}

// ListBatch mengembalikan satu halaman daftar batch milik satu portal.
func (s *Service) ListBatch(
	ctx context.Context,
	portalAlias string,
	filter inboxautoclaim.BatchFilter,
) (inboxautoclaim.BatchPage, error) {
	if err := requireSource(filter.Source); err != nil {
		return inboxautoclaim.BatchPage{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxautoclaim.BatchPage{}, err
	}
	return repo.ListBatch(ctx, filter)
}

// ListCompany mengembalikan pilihan penyaring Nama Perusahaan.
func (s *Service) ListCompany(ctx context.Context, portalAlias string) ([]inboxautoclaim.Company, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListCompany(ctx)
}

// PremiumCheckChoices mengembalikan isi kedua isian tab Cek Premi.
func (s *Service) PremiumCheckChoices(ctx context.Context, portalAlias string) (inboxautoclaim.PremiumCheckChoices, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxautoclaim.PremiumCheckChoices{}, err
	}
	return repo.PremiumCheckChoices(ctx)
}

// CheckPremiumTotal menjalankan tombol Cek Premi: total premi terbayar dari layanan REST,
// lalu total klaim Kredit yang sudah Sukses Klaim dari basis data.
//
// Urutannya mengikuti `CekPremi-Act` (layanan dulu, kueri kemudian). Layanan yang gagal
// menghentikan seluruhnya dengan ErrPremiumServiceUnavailable — menampilkan Total Klaim
// tanpa Total Premi mengundang petugas membandingkan angka dengan nol.
func (s *Service) CheckPremiumTotal(
	ctx context.Context,
	portalAlias string,
	query inboxautoclaim.PremiumCheckQuery,
) (inboxautoclaim.PremiumCheckResult, error) {
	clean, err := query.Clean()
	if err != nil {
		return inboxautoclaim.PremiumCheckResult{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxautoclaim.PremiumCheckResult{}, err
	}

	paid, err := s.premium.PremiumPaidBySource(ctx, portalAlias, clean)
	if err != nil {
		return inboxautoclaim.PremiumCheckResult{}, fmt.Errorf("%w: %v", inboxautoclaim.ErrPremiumServiceUnavailable, err)
	}

	claim, err := repo.SucceededClaimTotal(ctx, clean)
	if err != nil {
		return inboxautoclaim.PremiumCheckResult{}, err
	}
	return inboxautoclaim.PremiumCheckResult{PremiumPaid: paid, ClaimTotal: claim}, nil
}

// SummarizeCompany mengembalikan ringkasan jumlah batch per perusahaan.
func (s *Service) SummarizeCompany(ctx context.Context, portalAlias string, source inboxautoclaim.Source) (inboxautoclaim.Summary, error) {
	if err := requireSource(source); err != nil {
		return inboxautoclaim.Summary{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxautoclaim.Summary{}, err
	}
	return repo.SummarizeCompany(ctx, source)
}

// ListLine mengembalikan satu halaman rincian batch.
//
// Batch yang tidak ada menghasilkan ErrBatchNotFound, BUKAN halaman kosong. Keduanya
// terlihat sama di layar — tabel tanpa baris — tetapi menuntut tindakan yang berbeda:
// yang pertama berarti tautannya salah, yang kedua berarti penyaringnya terlalu sempit.
func (s *Service) ListLine(
	ctx context.Context,
	portalAlias string,
	query inboxautoclaim.LineQuery,
) (inboxautoclaim.LinePage, error) {
	if err := requireSource(query.Source); err != nil {
		return inboxautoclaim.LinePage{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxautoclaim.LinePage{}, err
	}

	exists, err := repo.BatchExists(ctx, query.Source, query.CompanyCode, query.BatchNumber)
	if err != nil {
		return inboxautoclaim.LinePage{}, err
	}
	if !exists {
		return inboxautoclaim.LinePage{}, inboxautoclaim.ErrBatchNotFound
	}
	return repo.ListLine(ctx, query)
}

// ExportFile adalah berkas unduhan yang sudah jadi.
type ExportFile struct {
	// FileName sudah berakhiran .csv dan sudah memuat penanda batch-nya.
	FileName string

	// Content adalah isi berkas.
	Content []byte

	// Rows adalah jumlah baris data di dalamnya, di luar baris judul. Dicatat supaya
	// lapisan transport dapat menuliskannya ke log tanpa membuka isi berkas.
	Rows int
}

// Export menyusun berkas CSV hasil satu batch.
//
// # Kenapa berkasnya disusun di sini, bukan di lapisan transport
//
// Judul kolom dan urutannya adalah KETETAPAN BISNIS yang disalin dari
// Activity/REPORT_AUTO_CLAIM_ACT-Act.xml — bukan urusan HTTP. Menyusunnya di transport
// berarti aturan itu hidup di lapisan yang tidak dapat diuji tanpa menyalakan server.
//
// Yang tetap menjadi urusan transport hanyalah header Content-Type dan
// Content-Disposition.
func (s *Service) Export(
	ctx context.Context,
	portalAlias string,
	query inboxautoclaim.LineQuery,
) (ExportFile, error) {
	spec, err := inboxautoclaim.ExportSpecFor(query.Result)
	if err != nil {
		return ExportFile{}, err
	}

	if err := requireSource(query.Source); err != nil {
		return ExportFile{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return ExportFile{}, err
	}

	exists, err := repo.BatchExists(ctx, query.Source, query.CompanyCode, query.BatchNumber)
	if err != nil {
		return ExportFile{}, err
	}
	if !exists {
		return ExportFile{}, inboxautoclaim.ErrBatchNotFound
	}

	line, err := repo.ExportLine(ctx, query)
	if err != nil {
		return ExportFile{}, err
	}

	var buffer bytes.Buffer
	// BOM UTF-8 ditulis lebih dulu. Tanpa itu, Excel membaca berkas sebagai ANSI dan
	// setiap huruf beraksen pada nama rusak. Berkas ini dibuka di Excel oleh petugas,
	// bukan diproses mesin, jadi yang menentukan bukan kemurnian melainkan keterbacaan.
	buffer.WriteString("\ufeff")

	writer := csv.NewWriter(&buffer)
	if err := writer.Write(spec.Header); err != nil {
		return ExportFile{}, fmt.Errorf("inboxautoclaim/usecase: menulis judul kolom: %w", err)
	}
	for _, l := range line {
		if err := writer.Write(spec.Row(l)); err != nil {
			return ExportFile{}, fmt.Errorf("inboxautoclaim/usecase: menulis baris ekspor: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return ExportFile{}, fmt.Errorf("inboxautoclaim/usecase: menutup berkas ekspor: %w", err)
	}

	// Nama berkas menyebut perusahaan dan batch-nya. Sistem lama tidak menyebutkannya —
	// FileName-nya tetap "Laporan Hasil Klaim" untuk setiap batch — sehingga mengunduh
	// dua batch berturut-turut menghasilkan dua berkas bernama sama di folder Unduhan,
	// dan yang kedua menimpa atau bernama "(1)". Penambahan ini tidak mengubah ISI
	// berkas sedikit pun, jadi ia tidak mempengaruhi uji kesetaraan.
	return ExportFile{
		FileName: fmt.Sprintf("%s - %s %s.csv", spec.FileName, query.CompanyCode, query.BatchNumber),
		Content:  buffer.Bytes(),
		Rows:     len(line),
	}, nil
}

// Upload menjalankan rantai pemeriksaan unggahan lalu menyimpannya.
//
// # Dua jenis kegagalan yang berakibat berbeda
//
// Urutan dan akibatnya mengikuti `Activity/InsertKlaimToTable_Other-Act.xml`:
//
//	BENTUK BERKAS    judul kolom, isian wajib, bentuk tanggal, bentuk nilai
//	                 -> SELURUH berkas ditolak, tidak satu baris pun tersimpan
//	ISI TERHADAP POLIS  perusahaan, prodke, urutan tanggal
//	                 -> baris TETAP DISIMPAN, dengan alasannya pada TMP_MESSAGE
//
// Pembedaan itu bukan selera. Berkas yang judul kolomnya salah belum berbentuk berkas
// klaim sama sekali — menyimpan sebagiannya hanya membuat batch yang isinya tidak dapat
// dipertanggungjawabkan. Sedangkan baris yang polisnya tidak ketemu adalah data yang sah
// bentuknya dan memang perlu terlihat petugas sebagai "Gagal", persis seperti di Pega.
//
// # Satu kegagalan yang membuat baris TIDAK disimpan
//
// Bila perusahaan tidak dapat diturunkan dari polis, barisnya tidak punya kode perusahaan
// — dan tanpa itu ia tidak masuk ke batch mana pun, sehingga tidak akan pernah terlihat di
// grid. Baris seperti itu dikembalikan sebagai Rejected supaya pengunggah tahu, bukan
// disimpan ke tempat yang tidak dapat dilihat siapa pun.
//
// # Periode polis dan premi (2026-09-29)
//
// Periode, status, dan mata uang polis dibaca dari T_GENERAL; status premi dari layanan
// `CekPremiAutoKlaim` untuk SEMUA baris (lihat checkPremium). Yang masih BELUM dibawa dari
// tab Kredit: "Tidak bisa input adjustment" dan "Objek belum ada Outstanding" (alur
// akseptasi) — lihat docs/keputusan-implementasi.md §64.
func (s *Service) Upload(
	ctx context.Context,
	portalAlias string,
	source inboxautoclaim.Source,
	row []inboxautoclaim.UploadRow,
	uploadedBy string,
) (inboxautoclaim.UploadResult, error) {
	if err := requireSource(source); err != nil {
		return inboxautoclaim.UploadResult{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxautoclaim.UploadResult{}, err
	}

	if err := inboxautoclaim.CheckUploadShape(source, row); err != nil {
		return inboxautoclaim.UploadResult{}, err
	}

	var (
		line     []inboxautoclaim.UploadLine
		policy   []inboxautoclaim.PolicyDetail
		rejected []inboxautoclaim.RejectedRow

		// Kontrak Kredit yang sudah muncul di berkas ini. Pega menyisipkan per baris, sehingga
		// baris kedua berkontrak sama tertangkap `CekObjekNotDouble`; di sini seluruh berkas
		// disimpan sekaligus di akhir, jadi baris kembar di dalam berkas dicatat sendiri.
		seenContract = map[string]bool{}
	)

	for _, r := range row {
		// Pemeriksaan khusus tab yang MENOLAK baris dijalankan sebelum pencarian polis,
		// mengikuti urutan Pega: keterangan laporan Travel diperiksa sebelum penerima
		// klaim dicari (InsertKlaimToTable_Travel :3976).
		if message, reject := inboxautoclaim.CheckRowForSource(source, r); reject {
			rejected = append(rejected, inboxautoclaim.RejectedRow{
				LineNumber: r.LineNumber,
				PolicyNo:   r.PolicyNo,
				Message:    message,
			})
			continue
		}

		resolved, err := s.resolve(ctx, repo, source, r)
		if err != nil {
			return inboxautoclaim.UploadResult{}, err
		}

		if resolved.CompanyCode == "" {
			rejected = append(rejected, inboxautoclaim.RejectedRow{
				LineNumber: r.LineNumber,
				PolicyNo:   r.PolicyNo,
				Message:    resolved.Message,
			})
			continue
		}

		if source == inboxautoclaim.SourceKredit && resolved.Message == "" {
			key := resolved.CompanyCode + "\x00" + strings.ToUpper(strings.TrimSpace(r.ContractNo))
			if seenContract[key] {
				resolved.Message = inboxautoclaim.MessageAlreadyClaimed
			}
			seenContract[key] = true
		}

		line = append(line, inboxautoclaim.UploadLine{
			Row:         r,
			CompanyCode: resolved.CompanyCode,
			CompanyName: resolved.CompanyName,
			ProductSeq:  resolved.ProductSeq,
			CurrencyID:  resolved.CurrencyID,
			Message:     resolved.Message,
		})
		policy = append(policy, resolved.Policy)
	}

	if err := s.checkPremium(ctx, repo, portalAlias, source, line, policy); err != nil {
		return inboxautoclaim.UploadResult{}, err
	}

	if len(line) == 0 {
		// Seluruh baris ditolak. Hasilnya tetap dikembalikan, bukan sebagai galat:
		// pengunggah perlu melihat alasan per barisnya, dan "tidak ada yang tersimpan"
		// sudah terbaca dari JumlahBaris nol.
		return inboxautoclaim.UploadResult{Rejected: rejected}, nil
	}

	result, err := repo.InsertUpload(ctx, source, line, uploadedBy)
	if err != nil {
		return inboxautoclaim.UploadResult{}, err
	}
	result.Rejected = rejected

	// Nama perusahaan dilengkapi dari hasil pencarian supaya ringkasan menyebut nama,
	// bukan kode. Penyimpanan SQL tidak mengisinya; yang memori mengisinya. Melengkapinya
	// di sini membuat keduanya menghasilkan ringkasan yang sama bentuknya.
	name := map[string]string{}
	for _, l := range line {
		if l.CompanyName != "" {
			name[l.CompanyCode] = l.CompanyName
		}
	}
	for index := range result.Batch {
		if result.Batch[index].CompanyName == "" {
			result.Batch[index].CompanyName = name[result.Batch[index].CompanyCode]
		}
	}
	return result, nil
}

// resolve menjalankan ketiga pemeriksaan terhadap polis untuk satu baris.
//
// Urutannya mengikuti activity aslinya, dan urutan itu berarti: pencarian perusahaan lebih
// dulu, karena kegagalannya satu-satunya yang membuat baris tidak dapat disimpan.
//
// Pemeriksaan BERHENTI pada kegagalan pertama. Itu mengikuti Pega, yang menetapkan satu
// pesan per baris — kolom TMP_MESSAGE hanya memuat satu, dan menggabungkan beberapa alasan
// akan menghasilkan teks yang tidak pernah ada di data lama.
func (s *Service) resolve(
	ctx context.Context,
	repo inboxautoclaim.Repo,
	source inboxautoclaim.Source,
	row inboxautoclaim.UploadRow,
) (inboxautoclaim.Resolution, error) {
	company, found, err := repo.ResolveReceiver(ctx, row.PolicyNo)
	if err != nil {
		return inboxautoclaim.Resolution{}, err
	}
	if !found {
		return inboxautoclaim.Resolution{Message: inboxautoclaim.MessageReceiverNotFound}, nil
	}

	resolution := inboxautoclaim.Resolution{
		CompanyCode: company.Code,
		CompanyName: company.Name,
	}

	productSeq, found, err := repo.FindPolicyProductSeq(ctx, row.PolicyNo)
	if err != nil {
		return inboxautoclaim.Resolution{}, err
	}
	if !found {
		// Polis ada di T_GENERAL — pencarian perusahaan berhasil — tetapi tidak ada di
		// JSON_POLIS. Itu keadaan yang dibedakan activity aslinya, dan pesannya berbeda.
		resolution.Message = inboxautoclaim.MessagePolicyNotFound
		return resolution, nil
	}
	resolution.ProductSeq = productSeq

	// Data polis dari T_GENERAL — pengganti snapshot JSON yang sudah tidak dipakai.
	detail, found, err := repo.FindPolicyDetail(ctx, row.PolicyNo, productSeq)
	if err != nil {
		return inboxautoclaim.Resolution{}, err
	}
	if !found {
		resolution.Message = inboxautoclaim.MessagePolicyNotFound
		return resolution, nil
	}
	resolution.Policy = detail

	// Mata uang tidak pernah menggagalkan baris: kode yang tidak dikenal hanya membuat
	// kolom CURRENCY kosong, sama seperti sebelum pencarian ini ada.
	if currencyID, known, err := repo.CurrencyID(ctx, detail.Currency); err != nil {
		return inboxautoclaim.Resolution{}, err
	} else if known {
		resolution.CurrencyID = currencyID
	}

	switch source {
	case inboxautoclaim.SourceKredit:
		// "Sudah Klaim" — kontrak yang sama sudah pernah diunggah dan belum gagal.
		claimed, err := repo.ContractClaimed(ctx, company.Code, strings.ToUpper(row.ContractNo))
		if err != nil {
			return inboxautoclaim.Resolution{}, err
		}
		if claimed {
			resolution.Message = inboxautoclaim.MessageAlreadyClaimed
		}
	case inboxautoclaim.SourceAneka:
		// Urutan tanggal lebih dulu (Other :4936), lalu periode polis KHUSUS produk hewan
		// (Other :5005).
		if message, _ := inboxautoclaim.CheckRowForSource(source, row); message != "" {
			resolution.Message = message
		} else if detail.BusinessCode == inboxautoclaim.PetBusinessCode && !detail.Covers(row.DateOfLoss) {
			resolution.Message = inboxautoclaim.MessageLossOutsidePolicy
		}
	case inboxautoclaim.SourceTravel:
		// Travel tidak punya tanggal lapor; yang diperiksa hanya periode polis (:5647).
		if !detail.Covers(row.DateOfLoss) {
			resolution.Message = inboxautoclaim.MessageLossOutsidePolicyTravel
		}
	}
	return resolution, nil
}

// premiumWorkers membatasi pemanggilan layanan premi yang berjalan bersamaan.
//
// Satu berkas dapat memuat ribuan baris. Memanggil satu per satu membuat unggahan menunggu
// menit-an; memanggil semuanya sekaligus membanjiri layanan milik sistem lain. Delapan
// adalah kompromi yang menjaga keduanya.
const premiumWorkers = 8

// checkPremium menjalankan cek premi untuk setiap baris yang masih lolos, lalu pemeriksaan
// yang dalam urutan Pega berada SESUDAH premi.
//
// # Aturannya (keputusan Work Owner 2026-09-29)
//
//   - SEMUA baris di ketiga tab dicek — tidak dibatasi Source of Business seperti Pega.
//   - Layanan mati, belum terdaftar, atau jawabannya tak terbaca → MessagePremiumCheckFailed.
//   - Belum lunas (`AgingAmount` kosong atau > 1) → MessagePremiumUnpaid; KECUALI di tab
//     Travel dan ANEKA bila polis punya Open Protection tipe 3 di T_CLAIM_OPENPROTECTION.
//   - Tab Kredit: sesudah premi lolos, polis yang dibatalkan → MessagePolicyCancelled
//     (urutan InsertKlaimToTable_Kredit: premi :8471 lalu batal :9301).
//
// Layanan dipanggil SEKALI per pasangan (polis, prodke): satu berkas lazim memuat polis
// yang sama berulang, dan jawabannya tidak berubah dalam satu unggahan.
func (s *Service) checkPremium(
	ctx context.Context,
	repo inboxautoclaim.Repo,
	portalAlias string,
	source inboxautoclaim.Source,
	line []inboxautoclaim.UploadLine,
	policy []inboxautoclaim.PolicyDetail,
) error {
	type outcome struct {
		answer inboxautoclaim.PremiumAnswer
		err    error
	}

	unique := map[inboxautoclaim.PremiumQuery]*outcome{}
	for _, l := range line {
		if l.Accepted() {
			unique[inboxautoclaim.PremiumQuery{PolicyNo: l.Row.PolicyNo, ProductSeq: l.ProductSeq}] = &outcome{}
		}
	}

	var (
		wait  sync.WaitGroup
		slots = make(chan struct{}, premiumWorkers)
	)
	for query, result := range unique {
		wait.Add(1)
		slots <- struct{}{}
		go func(query inboxautoclaim.PremiumQuery, result *outcome) {
			defer wait.Done()
			defer func() { <-slots }()
			result.answer, result.err = s.premium.CheckPremium(ctx, portalAlias, query)
		}(query, result)
	}
	wait.Wait()

	for index := range line {
		l := &line[index]
		if !l.Accepted() {
			continue
		}
		result := unique[inboxautoclaim.PremiumQuery{PolicyNo: l.Row.PolicyNo, ProductSeq: l.ProductSeq}]

		switch {
		case result.err != nil:
			l.Message = inboxautoclaim.MessagePremiumCheckFailed
			continue
		case result.answer.Unpaid():
			exempt := false
			if source != inboxautoclaim.SourceKredit {
				protected, err := repo.HasOpenProtection(ctx, l.Row.PolicyNo, inboxautoclaim.OpenProtectionPremiumType)
				if err != nil {
					return err
				}
				exempt = protected
			}
			if !exempt {
				l.Message = inboxautoclaim.MessagePremiumUnpaid
				continue
			}
		}

		if source == inboxautoclaim.SourceKredit && policy[index].Cancelled() {
			l.Message = inboxautoclaim.MessagePolicyCancelled
		}
	}
	return nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca — dan pada
// unggahan itu berarti sebelum berkas berukuran megabyte ditarik dari jaringan.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("inboxautoclaim/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}
