// Package memory memenuhi kelima seam modul Daftar Tipe Dokumen Bisnis tanpa basis data.
//
// Dipakai pengujian dan pengembangan lokal. Ia BUKAN cache dan bukan lapisan di depan
// Oracle: isinya hilang bersama prosesnya.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/daftartipedokumenbisnis"
)

// defaultSite adalah awalan ID yang dipakai adapter ini.
//
// Di Oracle ia dibaca dari M_SITE_DATABASE dan berbeda-beda per entitas. Di sini ia tetap,
// karena adapter ini hanya pernah melayani satu portal.
const defaultSite = "1"

// Repo menyimpan aturan dokumen di memori.
type Repo struct {
	mutex     sync.Mutex
	rules     map[string]daftartipedokumenbisnis.DocumentRule
	coverages map[string][]string
	site      string
	sequence  int64
	failure   error
}

// NewRepo membentuk repo berisi baris awal yang diberikan.
func NewRepo(rows ...daftartipedokumenbisnis.DocumentRule) *Repo {
	repo := &Repo{
		rules:     make(map[string]daftartipedokumenbisnis.DocumentRule, len(rows)),
		coverages: map[string][]string{},
		site:      defaultSite,
	}
	for _, row := range rows {
		clean := daftartipedokumenbisnis.DocumentRule{
			ID:               strings.TrimSpace(row.ID),
			BusinessID:       strings.TrimSpace(row.BusinessID),
			BusinessName:     strings.TrimSpace(row.BusinessName),
			DocumentTypeID:   strings.TrimSpace(row.DocumentTypeID),
			DocumentTypeName: strings.TrimSpace(row.DocumentTypeName),
			ObjectDocID:      strings.TrimSpace(row.ObjectDocID),
			ObjectDocName:    strings.TrimSpace(row.ObjectDocName),
			DetailTypeDocID:  strings.TrimSpace(row.DetailTypeDocID),
			DetailDocument:   strings.TrimSpace(row.DetailDocument),
			Mandatory:        row.Mandatory,
			MinDocument:      row.MinDocument,
		}
		repo.rules[clean.ID] = clean
		for _, coverage := range row.Coverages {
			if id := strings.TrimSpace(coverage.ID); id != "" {
				repo.coverages[clean.ID] = append(repo.coverages[clean.ID], id)
			}
		}
		if number := sequenceFromID(clean.ID, repo.site); number > repo.sequence {
			repo.sequence = number
		}
	}
	return repo
}

// SetError membuat seluruh operasi berikutnya gagal dengan galat itu.
//
// Ada supaya jalur galat repo dapat diuji tanpa basis data — tanpa ini, satu-satunya cara
// membuktikan layar menangani kegagalan penyimpanan adalah mematikan Oracle.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// ListBusinesses mengembalikan lini bisnis yang sudah punya aturan dokumen.
func (r *Repo) ListBusinesses(_ context.Context) ([]daftartipedokumenbisnis.Business, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	seen := map[string]daftartipedokumenbisnis.Business{}
	for _, rule := range r.rules {
		if rule.BusinessID == "" {
			continue
		}
		seen[rule.BusinessID] = daftartipedokumenbisnis.Business{
			ID:   rule.BusinessID,
			Name: rule.BusinessName,
		}
	}

	result := make([]daftartipedokumenbisnis.Business, 0, len(seen))
	for _, business := range seen {
		result = append(result, business)
	}
	// Menurut NAMA, sama dengan kueri Oracle-nya. Mengurutkannya berbeda akan membuat uji
	// layar lulus terhadap urutan yang tidak pernah terjadi di produksi.
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// ListByBusiness mengembalikan aturan dokumen milik satu lini bisnis.
func (r *Repo) ListByBusiness(_ context.Context, businessID string) ([]daftartipedokumenbisnis.DocumentRule, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	key := strings.TrimSpace(businessID)
	var result []daftartipedokumenbisnis.DocumentRule
	for _, rule := range r.rules {
		if rule.BusinessID == key {
			result = append(result, rule)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Get mengembalikan satu aturan lengkap dengan jaminannya.
func (r *Repo) Get(_ context.Context, id string) (daftartipedokumenbisnis.DocumentRule, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftartipedokumenbisnis.DocumentRule{}, r.failure
	}
	return r.get(strings.TrimSpace(id))
}

// get membaca satu baris; pemanggilnya sudah memegang kunci.
func (r *Repo) get(id string) (daftartipedokumenbisnis.DocumentRule, error) {
	rule, exists := r.rules[id]
	if !exists {
		return daftartipedokumenbisnis.DocumentRule{}, daftartipedokumenbisnis.ErrNotFound
	}
	for _, coverage := range r.coverages[id] {
		rule.Coverages = append(rule.Coverages, daftartipedokumenbisnis.Coverage{ID: coverage})
	}
	return rule, nil
}

// InsertBatch menyisipkan perkalian bisnis kali baris aturan.
func (r *Repo) InsertBatch(
	_ context.Context,
	input daftartipedokumenbisnis.BatchInput,
	_ daftartipedokumenbisnis.Editor,
) ([]daftartipedokumenbisnis.DocumentRule, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	var saved []daftartipedokumenbisnis.DocumentRule
	for _, businessID := range input.BusinessIDs {
		for _, rule := range input.Rules {
			id := r.issueID()
			row := daftartipedokumenbisnis.DocumentRule{
				ID:              id,
				BusinessID:      businessID,
				DocumentTypeID:  rule.DocumentTypeID,
				ObjectDocID:     rule.ObjectDocID,
				DetailTypeDocID: rule.DetailTypeDocID,
				DetailDocument:  rule.DetailDocument,
				Mandatory:       rule.Mandatory,
				MinDocument:     rule.MinDocument,
			}
			r.rules[id] = row
			saved = append(saved, row)
		}
	}
	return saved, nil
}

// Update mengubah satu baris aturan.
func (r *Repo) Update(
	_ context.Context,
	id string,
	input daftartipedokumenbisnis.Input,
	_ daftartipedokumenbisnis.Editor,
) (daftartipedokumenbisnis.DocumentRule, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftartipedokumenbisnis.DocumentRule{}, r.failure
	}

	key := strings.TrimSpace(id)
	existing, exists := r.rules[key]
	if !exists {
		return daftartipedokumenbisnis.DocumentRule{}, daftartipedokumenbisnis.ErrNotFound
	}

	// BusinessID dan kedua nama hasil join dipertahankan apa adanya — yang pertama karena
	// UPDATE di Oracle pun tidak menyentuhnya, kedua sisanya karena keduanya tidak pernah
	// dikirim pengguna.
	existing.DocumentTypeID = input.DocumentTypeID
	existing.ObjectDocID = input.ObjectDocID
	existing.DetailTypeDocID = input.DetailTypeDocID
	existing.DetailDocument = input.DetailDocument
	existing.Mandatory = input.Mandatory
	existing.MinDocument = input.MinDocument
	existing.Coverages = nil
	r.rules[key] = existing

	return r.get(key)
}

// AddCoverage menambahkan satu jaminan, dan diam bila jaminan itu sudah ada.
func (r *Repo) AddCoverage(_ context.Context, id string, coverageID string) (daftartipedokumenbisnis.DocumentRule, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftartipedokumenbisnis.DocumentRule{}, r.failure
	}

	key := strings.TrimSpace(id)
	if _, exists := r.rules[key]; !exists {
		return daftartipedokumenbisnis.DocumentRule{}, daftartipedokumenbisnis.ErrNotFound
	}

	coverage := strings.TrimSpace(coverageID)
	for _, existing := range r.coverages[key] {
		if existing == coverage {
			return r.get(key)
		}
	}
	r.coverages[key] = append(r.coverages[key], coverage)
	sort.Strings(r.coverages[key])
	return r.get(key)
}

// issueID menerbitkan ID yang belum terpakai; pemanggilnya sudah memegang kunci.
func (r *Repo) issueID() string {
	for {
		r.sequence++
		id := daftartipedokumenbisnis.FormatID(r.site, r.sequence)
		if _, taken := r.rules[id]; !taken {
			return id
		}
	}
}

// sequenceFromID membaca kembali nomor urut dari sebuah ID.
//
// ID yang tidak berbentuk seperti itu menghasilkan nol, bukan galat: contoh data boleh
// memuat ID berbentuk lain, dan menolaknya akan membuat adapter ini tidak dapat diisi
// data yang datang dari produksi.
func sequenceFromID(id, site string) int64 {
	if !strings.HasPrefix(id, site) {
		return 0
	}
	number, err := strconv.ParseInt(strings.TrimPrefix(id, site), 10, 64)
	if err != nil {
		return 0
	}
	return number
}

// BusinessRepo menyimpan master lini bisnis di memori.
type BusinessRepo struct {
	mutex sync.Mutex
	rows  []daftartipedokumenbisnis.Business
}

// NewBusinessRepo membentuk pembaca master lini bisnis.
func NewBusinessRepo(rows ...daftartipedokumenbisnis.Business) *BusinessRepo {
	return &BusinessRepo{rows: rows}
}

// List mengembalikan seluruh lini bisnis.
func (r *BusinessRepo) List(_ context.Context) ([]daftartipedokumenbisnis.Business, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	result := make([]daftartipedokumenbisnis.Business, len(r.rows))
	copy(result, r.rows)
	return result, nil
}

// ReferenceRepo menyimpan satu master daftar pilihan di memori.
//
// Satu tipe untuk ketiga seam rujukan, sejalan dengan tipe Reference di lapisan domain:
// ketiganya benar-benar berbentuk sama, dan tiga tipe kembar hanya akan menuntut tiga
// konstruktor yang isinya identik.
type ReferenceRepo struct {
	mutex sync.Mutex
	rows  []daftartipedokumenbisnis.Reference
}

// NewReferenceRepo membentuk pembaca daftar pilihan.
func NewReferenceRepo(rows ...daftartipedokumenbisnis.Reference) *ReferenceRepo {
	return &ReferenceRepo{rows: rows}
}

// List mengembalikan seluruh pilihan.
func (r *ReferenceRepo) List(_ context.Context) ([]daftartipedokumenbisnis.Reference, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	result := make([]daftartipedokumenbisnis.Reference, len(r.rows))
	copy(result, r.rows)
	return result, nil
}

var (
	_ daftartipedokumenbisnis.Repo              = (*Repo)(nil)
	_ daftartipedokumenbisnis.BusinessRepo      = (*BusinessRepo)(nil)
	_ daftartipedokumenbisnis.DocumentTypeRepo  = (*ReferenceRepo)(nil)
	_ daftartipedokumenbisnis.DetailTypeDocRepo = (*ReferenceRepo)(nil)
	_ daftartipedokumenbisnis.ObjectDocRepo     = (*ReferenceRepo)(nil)
)
