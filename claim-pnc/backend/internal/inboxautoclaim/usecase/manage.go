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

	"claim-pnc/internal/inboxautoclaim"
)

// Service adalah pintu masuk seluruh perkara Inbox Auto Claim.
type Service struct {
	repoSelector inboxautoclaim.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector inboxautoclaim.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxautoclaim/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
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
// # Dua pemeriksaan yang BELUM dapat dijalankan
//
// "Tanggal kejadian tidak dalam range polis" menuntut periode polis dari snapshot (B-1),
// dan "Premi belum lunas" menuntut `CekPremiAutoKlaim` yang belum ada di export. Baris
// yang seharusnya gagal karena keduanya akan LOLOS di sini dan diteruskan ke pemrosesan.
// Konsekuensinya dicatat di docs/keputusan-implementasi.md §18.
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

	if err := inboxautoclaim.CheckUploadShape(row); err != nil {
		return inboxautoclaim.UploadResult{}, err
	}

	var (
		line     []inboxautoclaim.UploadLine
		rejected []inboxautoclaim.RejectedRow
	)

	for _, r := range row {
		resolved, err := s.resolve(ctx, repo, r)
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

		line = append(line, inboxautoclaim.UploadLine{
			Row:         r,
			CompanyCode: resolved.CompanyCode,
			CompanyName: resolved.CompanyName,
			ProductSeq:  resolved.ProductSeq,
			Message:     resolved.Message,
		})
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

	if message := inboxautoclaim.CheckDateOrder(row.DateOfLoss, row.ReportDate); message != "" {
		resolution.Message = message
	}
	return resolution, nil
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
