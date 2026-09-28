// Package usecase mengorkestrasi modul Archive Dokumen Klaim.
//
// Tujuh operasi, mengikuti ketiga bagian layar lama:
//
//	Open           menyiapkan isi dropdown sekali per kunjungan
//	Search         grid ARCHIVE FILE KLAIM
//	SearchClaims   calon klaim pada bagian Input Data Archive
//	FillingCodes   isi pemilih "Pilih Kode"
//	Save           simpan berkas arsip — sisip atau ubah
//	PendingBranch  daftar berkas yang belum dikirim ke layanan Arsip
//	SendToBranch   kirim satu berkas lalu tandai
//
// # Kenapa kepemilikan transaksi ada di sini, bukan di repo
//
// `08-TECHNICAL-STRATEGY.md` §4.5 menetapkan transaksi dimulai dan diakhiri di lapisan
// aplikasi. Pada modul ini aturannya menggigit di satu tempat: SendToBranch memanggil
// sistem luar, dan panggilan itu TIDAK BOLEH berada di dalam transaksi basis data.
// Urutannya karena itu dipisah dengan sengaja — baca, kirim, baru tulis.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/archivedokumenklaim"
)

// Service melayani modul Archive Dokumen Klaim.
type Service struct {
	repoSelector archivedokumenklaim.RepoSelector
	gateway      archivedokumenklaim.Gateway
	clock        archivedokumenklaim.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector archivedokumenklaim.RepoSelector
	Gateway      archivedokumenklaim.Gateway
	Clock        archivedokumenklaim.Clock
}

// NewService membentuk layanan modul Archive Dokumen Klaim.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("archivedokumenklaim/usecase: RepoSelector wajib diisi")
	}
	if o.Gateway == nil {
		return nil, errors.New("archivedokumenklaim/usecase: Gateway wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("archivedokumenklaim/usecase: Clock wajib diisi")
	}
	return &Service{
		repoSelector: o.RepoSelector,
		gateway:      o.Gateway,
		clock:        o.Clock,
	}, nil
}

// Opened adalah keadaan layar tepat setelah dibuka.
//
// Seluruh isi dropdown dikirim sekaligus, bukan satu permintaan per dropdown. Ketiganya
// kecil dan jarang berubah, dan satu perjalanan jaringan lebih baik daripada tiga yang
// berlomba.
type Opened struct {
	ClaimSearchTypes []archivedokumenklaim.ClaimSearchTypeOption
	DocumentTypes    []archivedokumenklaim.DocumentTypeOption
	DocumentKinds    []archivedokumenklaim.DocumentKindOption

	// BranchScope adalah lini bisnis yang boleh dilihat pemanggil pada daftar kirim ke
	// cabang. Ia dikirim supaya layar dapat menerangkannya kepada pengguna alih-alih
	// membiarkannya menduga mengapa sebagian barisnya tidak muncul.
	BranchScope archivedokumenklaim.BranchScope
}

// Open menyiapkan isi dropdown dan cakupan lini bisnis pemanggil.
func (s *Service) Open(
	ctx context.Context,
	portalAlias string,
	caller archivedokumenklaim.Caller,
) (Opened, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Opened{}, archivedokumenklaim.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Opened{}, err
	}

	types, err := repo.DocumentTypes(ctx)
	if err != nil {
		return Opened{}, fmt.Errorf("membaca tipe dokumen: %w", err)
	}

	kinds, err := repo.DocumentKinds(ctx)
	if err != nil {
		return Opened{}, fmt.Errorf("membaca jenis dokumen: %w", err)
	}

	return Opened{
		ClaimSearchTypes: archivedokumenklaim.ClaimSearchTypes(),
		DocumentTypes:    types,
		DocumentKinds:    kinds,
		BranchScope:      archivedokumenklaim.BranchScopeFor(clean.Position),
	}, nil
}

// Found adalah hasil satu pencarian arsip.
type Found struct {
	Page archivedokumenklaim.ArchivePage

	// Criteria adalah kriteria yang BENAR-BENAR dipakai setelah divalidasi.
	Criteria archivedokumenklaim.Criteria
}

// Search menjalankan pencarian pada grid ARCHIVE FILE KLAIM.
func (s *Service) Search(
	ctx context.Context,
	portalAlias string,
	input archivedokumenklaim.CriteriaInput,
	page archivedokumenklaim.Pagination,
) (Found, error) {
	criteria, err := archivedokumenklaim.NewCriteria(input)
	if err != nil {
		return Found{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Found{}, err
	}

	result, err := repo.Search(ctx, criteria, page)
	if err != nil {
		return Found{}, fmt.Errorf("membaca berkas arsip: %w", err)
	}

	return Found{Page: result, Criteria: criteria}, nil
}

// SearchClaims mencari calon klaim yang berkasnya hendak diarsipkan.
//
// Ia TIDAK dipaginasi, berbeda dari pencarian arsip. Alasannya bukan kelalaian: ketiga
// tipe pencariannya mencocokkan nomor klaim, nomor polis, atau nama tertanggung secara
// PERSIS — bukan sebagian — sehingga hasilnya beberapa baris, bukan ribuan.
func (s *Service) SearchClaims(
	ctx context.Context,
	portalAlias string,
	searchType string,
	value string,
) ([]archivedokumenklaim.ClaimCandidate, error) {
	criteria, err := archivedokumenklaim.NewClaimCriteria(searchType, value)
	if err != nil {
		return nil, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	candidates, err := repo.SearchClaims(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("mencari klaim: %w", err)
	}

	return candidates, nil
}

// FillingCodes membaca isi pemilih "Pilih Kode".
//
// Asal daftarnya, dan apa yang belum diketahui tentangnya, dijelaskan di
// archivedokumenklaim.FillingCodeOption.
func (s *Service) FillingCodes(
	ctx context.Context,
	portalAlias string,
	keyword string,
) ([]archivedokumenklaim.FillingCodeOption, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	codes, err := repo.FillingCodes(ctx, keyword)
	if err != nil {
		return nil, fmt.Errorf("membaca kode filling: %w", err)
	}

	return codes, nil
}

// Saved adalah hasil satu penyimpanan berkas arsip.
type Saved struct {
	// ID adalah ID_ARCHIVE baris yang tersimpan — baru diterbitkan bila menyisipkan.
	ID int64

	// Created menyatakan barisnya baru, bukan hasil pengubahan.
	Created bool

	// Sent menyatakan berkasnya berhasil dikirim ke layanan Arsip pada penyimpanan ini.
	Sent bool

	// ServiceCode dan ServiceNote adalah jawaban layanan Arsip; kosong bila tidak dikirim
	// atau pengirimannya gagal.
	ServiceCode string
	ServiceNote string

	// SendError menjelaskan MENGAPA pengirimannya gagal, bila gagal.
	//
	// Ia dibawa sebagai teks, bukan sebagai galat yang dikembalikan, karena kegagalan
	// mengirim TIDAK menggagalkan penyimpanan — lihat Save.
	SendError string
}

// Save menyimpan satu berkas arsip.
//
// # Kenapa penanda "insert"/"update" sistem lama tidak dibawa
//
// Prosedur lama menerima parameter `flags` berisi teks "insert" atau "update", TERPISAH
// dari ID yang menyertainya. Keduanya dapat bertentangan — dan bila `flags` bukan salah
// satu dari kedua teks itu, prosedurnya diam-diam tidak melakukan apa pun lalu
// mengembalikan galat kosong.
//
// Di sini yang menentukan hanyalah ID: nol berarti sisip, selain itu berarti ubah. Dua
// nilai tidak dapat bertentangan bila hanya ada satu.
func (s *Service) Save(
	ctx context.Context,
	portalAlias string,
	caller archivedokumenklaim.Caller,
	branchCode string,
	input archivedokumenklaim.DraftInput,
) (Saved, error) {
	draft, err := archivedokumenklaim.NewDraft(input, caller, branchCode)
	if err != nil {
		return Saved{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Saved{}, err
	}

	// Pengubahan diperiksa keberadaannya lebih dulu. Prosedur lama menjalankan UPDATE
	// tanpa memeriksanya, dan UPDATE yang tidak mengenai satu baris pun TIDAK gagal —
	// ia mengembalikan "1" yang berarti berhasil. Pengguna karena itu melihat pesan
	// sukses untuk penyimpanan yang tidak menyimpan apa pun.
	if !draft.IsNew() {
		if _, exists, err := repo.FindByID(ctx, draft.ID); err != nil {
			return Saved{}, fmt.Errorf("memeriksa berkas arsip: %w", err)
		} else if !exists {
			return Saved{}, archivedokumenklaim.ErrNotFound
		}
	}

	id, err := repo.Save(ctx, draft)
	if err != nil {
		return Saved{}, fmt.Errorf("menyimpan berkas arsip: %w", err)
	}

	saved := Saved{ID: id, Created: draft.IsNew()}

	// Berkasnya langsung dikirim ke layanan Arsip, persis sistem lama:
	// `Activity/SaveAttachArchiveToDatabase-Act.xml` langkah 5 memanggil
	// `SendDataArchiveDOcumentByService` tepat setelah prosedur penyisipannya selesai.
	//
	// Jawabannya disimpan TANPA menandai CABANGSTATUS, sehingga berkasnya tetap muncul di
	// daftar Dokument Cabang dan akan terkirim untuk kedua kalinya dari sana. Pengiriman
	// ganda itu perilaku sistem lama, dan Work Owner memutuskan 2026-09-25 ia direplikasi
	// apa adanya.
	s.sendOnSave(ctx, portalAlias, repo, id, draft, &saved)

	return saved, nil
}

// sendOnSave mengirim berkas yang baru disimpan ke layanan Arsip.
//
// # Kenapa kegagalannya TIDAK menggagalkan penyimpanan
//
// Karena barisnya sudah tersimpan, dan di sistem lama pun begitu: prosedurnya melakukan
// COMMIT sendiri sebelum langkah pengiriman dijalankan. Mengembalikan galat dari Save
// akan membuat layar melaporkan "gagal menyimpan" atas berkas yang sebenarnya ADA di
// basis data — dan pengguna akan menyimpannya lagi, menghasilkan baris ganda yang tidak
// dapat dihapus dari layar ini.
//
// Yang dikerjakan sebagai gantinya: kegagalannya dilaporkan sebagai keterangan pada
// jawaban yang berhasil, dan berkasnya tetap berada di daftar Dokument Cabang sehingga
// pengirimannya dapat diulang dari sana.
func (s *Service) sendOnSave(
	ctx context.Context,
	portalAlias string,
	repo archivedokumenklaim.Repo,
	id int64,
	draft archivedokumenklaim.Draft,
	saved *Saved,
) {
	shipment := archivedokumenklaim.Shipment{
		ID:          id,
		BoxName:     draft.BoxName,
		FillingCode: draft.FillingCode,
		RequestedAt: s.clock.Now().UTC(),
	}

	receipt, err := s.gateway.Send(ctx, portalAlias, shipment)
	if err != nil {
		saved.SendError = err.Error()
		return
	}

	receipt.ID = id
	if receipt.SentAt.IsZero() {
		receipt.SentAt = shipment.RequestedAt
	}

	if err := repo.StoreReceipt(ctx, receipt); err != nil {
		// Berkasnya SUDAH sampai ke sistem Arsip; yang gagal hanyalah mencatat
		// jawabannya. Menyembunyikannya akan membuat jejak pengiriman hilang tanpa satu
		// pun tanda.
		saved.SendError = err.Error()
		return
	}

	saved.Sent = true
	saved.ServiceCode = receipt.Code
	saved.ServiceNote = receipt.Note
}

// Pending adalah satu halaman daftar kirim ke cabang.
type Pending struct {
	Page archivedokumenklaim.ArchivePage

	// Scope adalah cakupan lini bisnis yang dipakai menyaring halaman ini.
	Scope archivedokumenklaim.BranchScope
}

// PendingBranch membaca berkas yang belum dikirim ke layanan Arsip.
func (s *Service) PendingBranch(
	ctx context.Context,
	portalAlias string,
	caller archivedokumenklaim.Caller,
	page archivedokumenklaim.Pagination,
) (Pending, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Pending{}, archivedokumenklaim.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Pending{}, err
	}

	scope := archivedokumenklaim.BranchScopeFor(clean.Position)

	result, err := repo.PendingBranch(ctx, scope, page)
	if err != nil {
		return Pending{}, fmt.Errorf("membaca berkas yang belum dikirim: %w", err)
	}

	return Pending{Page: result, Scope: scope}, nil
}

// Sent adalah hasil satu pengiriman ke layanan Arsip.
type Sent struct {
	ID int64

	// Code dan Note adalah jawaban layanan Arsip, apa adanya.
	Code string
	Note string
}

// SendToBranch mengirim satu berkas arsip ke layanan Arsip lalu menandainya.
//
// # Urutannya, dan kenapa persis begini
//
//	1 baca barisnya         — memastikan ia ada DAN belum pernah dikirim
//	2 kirim ke layanan luar — DI LUAR transaksi basis data
//	3 simpan jawabannya     — sekaligus menandai CABANGSTATUS='1'
//
// Langkah 1 tidak ada di sistem lama: `SENDDATACABANGKEARCHIVE` menerima ID dari tombol
// lalu langsung mengirim, sehingga menekan tombolnya dua kali mengirim berkas yang sama
// dua kali ke sistem Arsip. Pemeriksaan di sini menutupnya, dan ia ditegakkan di SERVER —
// menonaktifkan tombol di layar hanyalah kenyamanan tampilan (`D-59`).
//
// Langkah 2 berada di luar transaksi karena pemanggilan sistem eksternal tidak boleh
// menahan kunci baris (`10-API-STRATEGY.md` §8.2). Konsekuensinya diterima secara sadar:
// bila aplikasi berhenti tepat antara langkah 2 dan 3, berkasnya sudah sampai ke sistem
// Arsip tetapi belum tertandai, sehingga ia akan muncul lagi di daftar dan dapat terkirim
// dua kali. Menukar urutannya hanya memindahkan masalahnya — menandai lebih dulu berarti
// berkas yang pengirimannya gagal hilang dari daftar tanpa pernah sampai.
func (s *Service) SendToBranch(
	ctx context.Context,
	portalAlias string,
	caller archivedokumenklaim.Caller,
	id int64,
) (Sent, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Sent{}, archivedokumenklaim.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Sent{}, err
	}

	file, exists, err := repo.FindByID(ctx, id)
	if err != nil {
		return Sent{}, fmt.Errorf("membaca berkas arsip: %w", err)
	}
	if !exists {
		return Sent{}, archivedokumenklaim.ErrNotFound
	}
	if file.BranchStatus == archivedokumenklaim.BranchStatusSent {
		return Sent{}, archivedokumenklaim.ErrAlreadySent
	}

	shipment := archivedokumenklaim.Shipment{
		ID:          file.ID,
		BoxName:     file.BoxName,
		FillingCode: file.FillingCode,
		RequestedAt: s.clock.Now().UTC(),
	}

	receipt, err := s.gateway.Send(ctx, portalAlias, shipment)
	if err != nil {
		// Galat layanan diteruskan apa adanya supaya transport dapat membedakan
		// "alamatnya belum terdaftar" dari "layanannya menolak" — keduanya ditangani
		// orang yang berbeda.
		return Sent{}, err
	}

	receipt.ID = file.ID
	if receipt.SentAt.IsZero() {
		receipt.SentAt = shipment.RequestedAt
	}

	if err := repo.MarkSent(ctx, receipt); err != nil {
		return Sent{}, fmt.Errorf("menyimpan jawaban layanan Arsip: %w", err)
	}

	return Sent{ID: file.ID, Code: receipt.Code, Note: receipt.Note}, nil
}
