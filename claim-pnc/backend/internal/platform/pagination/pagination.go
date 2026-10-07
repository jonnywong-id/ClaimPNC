// Package pagination memuat hitungan halaman yang dipakai setiap inbox dan laporan.
//
// Setiap modul tetap memiliki tipe Pagination dan Page-nya sendiri — keduanya bagian dari
// API domain modul dan ukuran halaman bawaannya berbeda per layar. Yang tinggal di sini
// hanyalah aritmetikanya, yang sebelumnya tersalin di dua puluh paket.
package pagination

// Normalize merapikan nomor halaman dan ukurannya: halaman di bawah 1 menjadi 1, ukuran di
// bawah 1 menjadi defaultSize, dan ukuran di atas maxSize dipotong ke maxSize.
func Normalize(page, size, defaultSize, maxSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}
	return page, size
}

// Offset adalah jumlah baris yang dilewati sebelum halaman page yang berukuran size.
func Offset(page, size int) int { return (page - 1) * size }

// Pages adalah jumlah halaman untuk total baris; minimal 1, termasuk saat tidak ada baris.
func Pages(total, size int) int {
	if total <= 0 {
		return 1
	}
	pages := total / size
	if total%size != 0 {
		pages++
	}
	return pages
}

// Window memotong satu halaman dari all. Di luar jangkauan hasilnya senarai KOSONG yang
// bukan nil, supaya responsnya tertulis `[]`, bukan `null`.
func Window[T any](all []T, offset, size int) []T {
	if offset >= len(all) {
		return []T{}
	}
	end := offset + size
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}

// Sizes memberi ukuran halaman bawaan dan maksimum milik satu layar. Modul membuatnya dari
// konstantanya sendiri, sehingga angka ukuran tetap berasal dari satu tempat.
type Sizes interface {
	Default() int
	Max() int
}

// Request adalah permintaan satu halaman: nomor halaman (mulai dari 1) dan ukurannya.
type Request[Z Sizes] struct {
	Page int
	Size int
}

// Normalize merapikan permintaan dengan ukuran bawaan dan maksimum milik Z.
func (p Request[Z]) Normalize() Request[Z] {
	var z Z
	p.Page, p.Size = Normalize(p.Page, p.Size, z.Default(), z.Max())
	return p
}

// Offset adalah jumlah baris yang dilewati sebelum halaman yang diminta.
func (p Request[Z]) Offset() int {
	clean := p.Normalize()
	return Offset(clean.Page, clean.Size)
}

// Page adalah satu halaman hasil beserta jumlah seluruh barisnya.
type Page[T any, Z Sizes] struct {
	Items      []T
	Total      int
	Pagination Request[Z]
}

// TotalPages adalah jumlah halaman; minimal 1.
func (p Page[T, Z]) TotalPages() int { return Pages(p.Total, p.Pagination.Normalize().Size) }

// Slice memotong satu halaman dari seluruh baris yang sudah termuat.
func Slice[T any, Z Sizes](all []T, page Request[Z]) Page[T, Z] {
	clean := page.Normalize()
	return Page[T, Z]{Total: len(all), Pagination: clean, Items: Window(all, clean.Offset(), clean.Size)}
}

// LimitOffset merapikan batas dan geseran baris: batas di bawah 1 menjadi defaultLimit, di
// atas maxLimit dipotong ke maxLimit, dan geseran negatif menjadi nol.
func LimitOffset(limit, offset, defaultLimit, maxLimit int) (int, int) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
