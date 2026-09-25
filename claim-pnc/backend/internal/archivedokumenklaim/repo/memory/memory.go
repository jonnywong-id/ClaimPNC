// Package memory memenuhi seam archivedokumenklaim.Repo di dalam memori.
//
// Ia dipakai dua hal: pengujian aturan modul tanpa basis data, dan mode pengembangan
// lokal saat koneksi Oracle belum tersedia. Keberadaannya pula yang membuat seam Repo
// NYATA alih-alih hipotetis — dua adapter, bukan satu (`04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia meniru PERILAKU repo SQL, termasuk yang tidak nyaman:
//
//   - penomoran ID memakai nilai tertinggi ditambah satu, persis kueri `next_archive_id`;
//   - pencarian kata kunci mencocokkan PERSIS, bukan sebagian;
//   - nama tipe dan jenis dokumen dicari ke master, dan kosong bila kodenya tidak ada.
//
// Kalau ia lebih rapi daripada aslinya, uji yang lulus di sini tidak berarti apa-apa.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/archivedokumenklaim"
)

// Repo menyimpan berkas arsip di dalam memori.
//
// Seluruh operasinya terkunci mutex: satu instans dipakai bersama oleh permintaan yang
// berjalan bersamaan, dan peta Go tidak aman dibaca-tulis serentak.
type Repo struct {
	mu sync.RWMutex

	files      []archivedokumenklaim.ArchiveFile
	claims     []archivedokumenklaim.ClaimCandidate
	types      []archivedokumenklaim.DocumentTypeOption
	kinds      []archivedokumenklaim.DocumentKindOption
	nextNumber int64
}

// Options adalah bahan pembentuk Repo.
type Options struct {
	Files  []archivedokumenklaim.ArchiveFile
	Claims []archivedokumenklaim.ClaimCandidate
	Types  []archivedokumenklaim.DocumentTypeOption
	Kinds  []archivedokumenklaim.DocumentKindOption
}

// NewRepo membentuk repo memori.
func NewRepo(o Options) *Repo {
	repo := &Repo{
		files:  append([]archivedokumenklaim.ArchiveFile{}, o.Files...),
		claims: append([]archivedokumenklaim.ClaimCandidate{}, o.Claims...),
		types:  append([]archivedokumenklaim.DocumentTypeOption{}, o.Types...),
		kinds:  append([]archivedokumenklaim.DocumentKindOption{}, o.Kinds...),
	}
	repo.refreshNames()
	return repo
}

// Search membaca satu halaman grid ARCHIVE FILE KLAIM.
func (r *Repo) Search(
	_ context.Context,
	criteria archivedokumenklaim.Criteria,
	page archivedokumenklaim.Pagination,
) (archivedokumenklaim.ArchivePage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]archivedokumenklaim.ArchiveFile, 0, len(r.files))
	for _, file := range r.files {
		if matches(file, criteria) {
			matched = append(matched, file)
		}
	}

	return sliceToPage(matched, page), nil
}

// matches meniru klausa WHERE kedua mode pencarian.
func matches(file archivedokumenklaim.ArchiveFile, criteria archivedokumenklaim.Criteria) bool {
	if criteria.Mode == archivedokumenklaim.ModeInputDate {
		if file.InputDate == nil {
			return false
		}

		// Jamnya dibuang sebelum dibandingkan, persis `trunc(TGLINPUT)` pada kueri lama.
		// Tanpa itu, berkas yang diinput sore hari pada tanggal akhir akan tersaring
		// keluar — kelas cacat yang paling sering luput karena hanya muncul pada
		// sebagian baris.
		moment := file.InputDate.UTC()
		day := time.Date(moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, time.UTC)

		if criteria.From != nil && day.Before(*criteria.From) {
			return false
		}
		if criteria.To != nil && day.After(*criteria.To) {
			return false
		}
		return true
	}

	keyword := criteria.Keyword
	return strings.EqualFold(file.ClaimNumber, keyword) ||
		strings.EqualFold(file.BoxName, keyword) ||
		strings.EqualFold(file.InsuredName, keyword)
}

// PendingBranch membaca berkas yang belum dikirim ke layanan Arsip.
func (r *Repo) PendingBranch(
	_ context.Context,
	scope archivedokumenklaim.BranchScope,
	page archivedokumenklaim.Pagination,
) (archivedokumenklaim.ArchivePage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]archivedokumenklaim.ArchiveFile, 0, len(r.files))
	for _, file := range r.files {
		if file.BranchStatus == archivedokumenklaim.BranchStatusSent {
			continue
		}
		if excluded(file.GroupPanel, scope) {
			continue
		}
		matched = append(matched, file)
	}

	return sliceToPage(matched, page), nil
}

// excluded meniru saringan lini bisnis, termasuk keputusan meloloskan yang kosong.
func excluded(groupPanel string, scope archivedokumenklaim.BranchScope) bool {
	if strings.TrimSpace(groupPanel) == "" {
		return false
	}
	for _, hidden := range scope.ExcludedGroupPanels {
		if groupPanel == hidden {
			return true
		}
	}
	return false
}

// FindByID membaca satu berkas arsip.
func (r *Repo) FindByID(
	_ context.Context,
	id int64,
) (archivedokumenklaim.ArchiveFile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, file := range r.files {
		if file.ID == id {
			return file, true, nil
		}
	}
	return archivedokumenklaim.ArchiveFile{}, false, nil
}

// SearchClaims mencari calon klaim yang berkasnya hendak diarsipkan.
func (r *Repo) SearchClaims(
	_ context.Context,
	criteria archivedokumenklaim.ClaimCriteria,
) ([]archivedokumenklaim.ClaimCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]archivedokumenklaim.ClaimCandidate, 0, 8)
	for _, claim := range r.claims {
		if claimMatches(claim, criteria) {
			matched = append(matched, claim)
		}
	}
	return matched, nil
}

// claimMatches meniru kedua kueri pencarian klaim, termasuk OR ketiga kolomnya.
func claimMatches(
	claim archivedokumenklaim.ClaimCandidate,
	criteria archivedokumenklaim.ClaimCriteria,
) bool {
	if criteria.Type == archivedokumenklaim.ClaimByPolicy {
		return strings.EqualFold(claim.PolicyNumber, criteria.Value)
	}
	return strings.EqualFold(claim.Number, criteria.Value) ||
		strings.EqualFold(claim.PolicyNumber, criteria.Value) ||
		strings.EqualFold(claim.InsuredName, criteria.Value)
}

// Save menyimpan satu berkas arsip.
func (r *Repo) Save(_ context.Context, draft archivedokumenklaim.Draft) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if draft.IsNew() {
		id := r.nextNumber
		if id == 0 {
			id = 1
		}
		r.nextNumber = id + 1

		file := archivedokumenklaim.ArchiveFile{
			ID:           id,
			InputUser:    draft.InputUser,
			BranchStatus: archivedokumenklaim.BranchStatusPending,
		}
		applyDraft(&file, draft)
		r.files = append(r.files, file)
		r.fillNames(&r.files[len(r.files)-1])
		return id, nil
	}

	for index := range r.files {
		if r.files[index].ID != draft.ID {
			continue
		}
		applyDraft(&r.files[index], draft)
		r.fillNames(&r.files[index])
		return draft.ID, nil
	}

	return 0, archivedokumenklaim.ErrNotFound
}

// applyDraft menyalin isi formulir ke satu baris arsip.
//
// USERINPUT dan CABANGSTATUS TIDAK ikut tersalin — sama seperti kueri `update_archive`,
// yang sengaja tidak menyentuh keduanya.
func applyDraft(file *archivedokumenklaim.ArchiveFile, draft archivedokumenklaim.Draft) {
	file.ClaimNumber = draft.ClaimNumber
	file.PolicyNumber = draft.PolicyNumber
	file.InsuredName = draft.InsuredName
	file.LossDate = draft.LossDate
	file.TechnicalPIC = draft.TechnicalPIC
	file.DocumentReceivedDate = draft.DocumentReceivedDate
	file.SheetCount = draft.SheetCount
	file.DocumentTypeCode = draft.DocumentTypeCode
	file.DocumentKindCode = draft.DocumentKindCode
	file.BoxName = draft.BoxName
	file.FillingCode = draft.FillingCode
	file.GroupPanel = draft.GroupPanel
}

// StoreReceipt menyimpan jawaban layanan Arsip TANPA menandai barisnya terkirim.
func (r *Repo) StoreReceipt(_ context.Context, receipt archivedokumenklaim.Receipt) error {
	return r.applyReceipt(receipt, false)
}

// MarkSent menyimpan jawaban layanan Arsip dan menandai barisnya sudah dikirim.
func (r *Repo) MarkSent(_ context.Context, receipt archivedokumenklaim.Receipt) error {
	return r.applyReceipt(receipt, true)
}

// applyReceipt menyalin jawaban layanan ke satu baris.
//
// `mark` membedakan kedua jalur pengiriman sistem lama: Simpan tidak menandai, Dokument
// Cabang menandai. Lihat seam archivedokumenklaim.Repo.
func (r *Repo) applyReceipt(receipt archivedokumenklaim.Receipt, mark bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := range r.files {
		if r.files[index].ID != receipt.ID {
			continue
		}
		sentAt := receipt.SentAt
		r.files[index].ServiceCode = receipt.Code
		r.files[index].ServiceNote = receipt.Note
		r.files[index].SentDate = &sentAt
		if mark {
			r.files[index].BranchStatus = archivedokumenklaim.BranchStatusSent
		}
		return nil
	}

	return archivedokumenklaim.ErrNotFound
}

// DocumentTypes membaca pilihan Tipe Dokumen.
func (r *Repo) DocumentTypes(
	_ context.Context,
) ([]archivedokumenklaim.DocumentTypeOption, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]archivedokumenklaim.DocumentTypeOption{}, r.types...), nil
}

// DocumentKinds membaca pilihan Jenis Dokumen.
func (r *Repo) DocumentKinds(
	_ context.Context,
) ([]archivedokumenklaim.DocumentKindOption, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]archivedokumenklaim.DocumentKindOption{}, r.kinds...), nil
}

// FillingCodes mengumpulkan kode filling dari berkas yang sudah ada — rekonstruksi yang
// sama dengan kueri `filling_codes`.
func (r *Repo) FillingCodes(
	_ context.Context,
	keyword string,
) ([]archivedokumenklaim.FillingCodeOption, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type key struct{ code, box string }
	counter := map[key]int{}

	needle := strings.ToUpper(strings.TrimSpace(keyword))

	for _, file := range r.files {
		if strings.TrimSpace(file.FillingCode) == "" {
			continue
		}
		if needle != "" &&
			!strings.Contains(strings.ToUpper(file.FillingCode), needle) &&
			!strings.Contains(strings.ToUpper(file.BoxName), needle) {
			continue
		}
		counter[key{file.FillingCode, file.BoxName}]++
	}

	options := make([]archivedokumenklaim.FillingCodeOption, 0, len(counter))
	for k, count := range counter {
		options = append(options, archivedokumenklaim.FillingCodeOption{
			Code:       k.code,
			BoxName:    k.box,
			UsageCount: count,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		if options[i].Code != options[j].Code {
			return options[i].Code < options[j].Code
		}
		return options[i].BoxName < options[j].BoxName
	})

	return options, nil
}

// refreshNames mengisi nama tipe dan jenis dokumen seluruh baris.
func (r *Repo) refreshNames() {
	for index := range r.files {
		r.fillNames(&r.files[index])
	}
	r.nextNumber = 1
	for _, file := range r.files {
		if file.ID >= r.nextNumber {
			r.nextNumber = file.ID + 1
		}
	}
}

// fillNames mencari nama tipe dan jenis dokumen, dan mengosongkannya bila kodenya tidak
// ada di master — meniru LEFT JOIN pada kueri SQL.
func (r *Repo) fillNames(file *archivedokumenklaim.ArchiveFile) {
	file.DocumentTypeName = ""
	for _, option := range r.types {
		if option.Code == file.DocumentTypeCode {
			file.DocumentTypeName = option.Name
			break
		}
	}

	file.DocumentKindName = ""
	for _, option := range r.kinds {
		if option.Code == file.DocumentKindCode && option.TypeCode == file.DocumentTypeCode {
			file.DocumentKindName = option.Name
			break
		}
	}
}

// sliceToPage mengurutkan dan memotong satu halaman, meniru ORDER BY dan OFFSET/FETCH.
func sliceToPage(
	files []archivedokumenklaim.ArchiveFile,
	page archivedokumenklaim.Pagination,
) archivedokumenklaim.ArchivePage {
	page = page.Normalize()

	sort.Slice(files, func(i, j int) bool { return files[i].ID > files[j].ID })

	total := len(files)
	offset := page.Offset()
	if offset >= total {
		return archivedokumenklaim.ArchivePage{
			Files:      []archivedokumenklaim.ArchiveFile{},
			Total:      total,
			Pagination: page,
		}
	}

	end := offset + page.Size
	if end > total {
		end = total
	}

	slice := make([]archivedokumenklaim.ArchiveFile, end-offset)
	copy(slice, files[offset:end])

	return archivedokumenklaim.ArchivePage{
		Files:      slice,
		Total:      total,
		Pagination: page,
	}
}

// Repo wajib memenuhi seam modul.
var _ archivedokumenklaim.Repo = (*Repo)(nil)
