package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/mastersparepart"
)

// Unggah CSV master sparepart — padanan `PNCUploadMasterSparepartCSV`.
//
// # Bentuknya upsert, dan kuncinya NO_SPART
//
// `Activity/PNCUploadMasterSparepart_Act` mencari baris lama lewat
// `GCNM GetSparepartFromNoSparepart`, yang kuerinya utuh berbunyi:
//
//	select ID as "ID" from pooldata.sparepart_he where no_spart = {InputSparepart.NO_SPART}
//
// Ketemu -> ID lama dipakai ulang dan barisnya di-UPDATE. Tidak ketemu -> baris baru, dengan
// ID diterbitkan sentinel `"UnknownID"` di `PEGA_M_SPAREPART_HE.prc`. Di sini keduanya
// dipetakan ke Save dan Create yang sudah ada.
//
// **Nol penyaring tambahan pada kuerinya.** Tidak ada `STS_AKTIF`, tidak ada `APPROVAL` —
// sehingga baris yang sudah DITOLAK pun ikut tertimpa dan kembali ke antrean persetujuan.
// Itu perilaku sistem lama, ditiru apa adanya (`P-5`), dan dinyatakan di layar supaya tidak
// mengejutkan.
//
// # Kenapa memanggil Create dan Save, bukan menulis jalur sendiri
//
// Supaya satu berkas CSV tidak dapat memasukkan data yang form sendiri tolak. Keduanya sudah
// memegang pembersihan, pemeriksaan isian, penolakan kunci ganda, stempel harga, dan
// penyetelan `APPROVAL := "0"` — menyalinnya ke sini berarti dua tempat memutuskan hal yang
// sama, dan yang kedua pasti menyimpang pada perbaikan pertama yang hanya diterapkan di
// salah satunya.
//
// Satu akibatnya disadari: pemeriksaan kunci ganda **lebih ketat daripada Pega**, yang tidak
// memeriksanya sama sekali pada jalur unggah. Baris CSV yang namanya bentrok dengan baris
// lain akan gagal di sini dan lolos di sana. Itu dipilih sadar — longgarnya Pega menghasilkan
// master yang kemudian tidak dapat disunting lewat layar.

// ImportOutcome menyatakan apa yang terjadi pada satu baris.
type ImportOutcome string

const (
	// ImportCreated: barisnya baru, dan ID-nya baru diterbitkan.
	ImportCreated ImportOutcome = "baru"
	// ImportUpdated: NO_SPART-nya sudah ada, barisnya diperbarui.
	ImportUpdated ImportOutcome = "diperbarui"
	// ImportFailed: barisnya ditolak, dan Message menyebutkan sebabnya.
	ImportFailed ImportOutcome = "gagal"
)

// ImportRow adalah hasil satu baris berkas.
type ImportRow struct {
	// Line adalah nomor baris di BERKAS, header terhitung sebagai baris 1.
	Line int

	// Number adalah NO_SPART-nya — kunci upsert, dan satu-satunya penanda yang pasti ada
	// bahkan pada baris yang gagal.
	Number string

	// ID terisi hanya pada baris yang berhasil.
	ID string

	Outcome ImportOutcome

	// Message terisi hanya pada baris yang gagal.
	Message string
}

// ImportReport meringkas satu unggahan.
type ImportReport struct {
	Total   int
	Created int
	Updated int
	Failed  int

	// Rows memuat SELURUH baris, bukan yang gagal saja.
	//
	// Pengguna yang mengunggah 300 baris perlu dapat memastikan ketiga ratusnya terbaca —
	// laporan yang hanya memuat kegagalan tidak dapat membedakan "semua berhasil" dari
	// "separuh berkas tidak terbaca".
	Rows []ImportRow
}

// ImportCSV membaca satu berkas CSV dan meng-upsert seluruh barisnya.
//
// # Setiap baris berdiri sendiri
//
// Satu baris yang gagal TIDAK membatalkan yang lain, dan itu mengikuti Pega: activity-nya
// memutar baris satu per satu tanpa transaksi yang membungkus keseluruhan.
//
// Pilihan itu punya akibat yang harus disadari — unggahan yang separuh gagal meninggalkan
// separuh perubahan. Yang membuatnya dapat diterima adalah laporannya: setiap baris
// dinyatakan hasilnya, sehingga pengguna tahu persis apa yang masuk dan apa yang tidak, lalu
// dapat mengunggah ulang bagian yang gagal saja. Membungkus semuanya dalam satu transaksi
// justru akan membuat satu salah ketik membatalkan 299 baris yang benar.
func (l *Service) ImportCSV(
	ctx context.Context,
	portalAlias string,
	content []byte,
	by Actor,
	logger *slog.Logger,
) (ImportReport, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return ImportReport{}, err
	}

	rows, err := mastersparepart.ParseCSV(bytes.NewReader(content))
	if err != nil {
		return ImportReport{}, err
	}

	report := ImportReport{Total: len(rows), Rows: make([]ImportRow, 0, len(rows))}

	for _, row := range rows {
		clean := row.Input.Clean()
		result := ImportRow{Line: row.Line, Number: clean.Number}

		id, lookupErr := existingIDOf(ctx, store, clean.Number)
		if lookupErr != nil {
			result.Outcome = ImportFailed
			result.Message = "Baris tidak dapat diperiksa terhadap data yang ada."
			report.Failed++
			report.Rows = append(report.Rows, result)
			// Kegagalan membaca bukan kesalahan barisnya; ia menandakan basis datanya
			// sedang bermasalah. Dicatat supaya tidak hilang di antara kegagalan validasi.
			if logger != nil {
				logger.ErrorContext(ctx, "gagal mencari sparepart saat unggah CSV",
					slog.String("modul", "mastersparepart"),
					slog.Int("baris", row.Line),
					slog.String("galat", lookupErr.Error()))
			}
			continue
		}

		var saved mastersparepart.Sparepart
		var writeErr error
		if id == "" {
			saved, writeErr = l.Create(ctx, portalAlias, row.Input, by, nil)
			result.Outcome = ImportCreated
		} else {
			saved, writeErr = l.Save(ctx, portalAlias, id, row.Input, by, nil)
			result.Outcome = ImportUpdated
		}

		if writeErr != nil {
			result.Outcome = ImportFailed
			result.Message = reasonOf(writeErr)
			report.Failed++
			report.Rows = append(report.Rows, result)
			continue
		}

		result.ID = saved.ID
		if result.Outcome == ImportCreated {
			report.Created++
		} else {
			report.Updated++
		}
		report.Rows = append(report.Rows, result)
	}

	if logger != nil {
		logger.InfoContext(ctx, "unggah CSV master sparepart selesai",
			slog.String("modul", "mastersparepart"),
			slog.String("portal", portalAlias),
			slog.String("oleh", strings.TrimSpace(by.Login)),
			slog.Int("total", report.Total),
			slog.Int("baru", report.Created),
			slog.Int("diperbarui", report.Updated),
			slog.Int("gagal", report.Failed))
	}
	return report, nil
}

// existingIDOf mengembalikan ID baris ber-NO_SPART itu, atau "" bila belum ada.
//
// Padanan `GCNM GetSparepartFromNoSparepart`. Nomor kosong TIDAK dicari: ia pasti ditolak
// pemeriksaan isian sesaat kemudian, dan mencarinya lebih dulu hanya menambah satu
// perjalanan ke basis data untuk baris yang sudah pasti gagal.
func existingIDOf(
	ctx context.Context, store mastersparepart.Store, number string,
) (string, error) {
	if strings.TrimSpace(number) == "" {
		return "", nil
	}
	found, err := store.FindByNumber(ctx, number)
	if errors.Is(err, mastersparepart.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("mastersparepart/usecase: mencari nomor %q: %w", number, err)
	}
	return strings.TrimSpace(found.ID), nil
}

// reasonOf mengubah galat satu baris menjadi kalimat yang dapat ditindaklanjuti.
//
// Galat validasi dirangkai SELURUHNYA, bukan yang pertama saja — alasannya sama dengan pada
// form: petugas yang salah pada dua isian perlu tahu keduanya dalam satu kali unggah, bukan
// dua.
func reasonOf(err error) string {
	var validation *mastersparepart.ValidationError
	if errors.As(err, &validation) {
		pesan := make([]string, 0, len(validation.Violation))
		for _, v := range validation.Violation {
			pesan = append(pesan, v.Message)
		}
		return strings.Join(pesan, " ")
	}
	switch {
	case errors.Is(err, mastersparepart.ErrNumberTaken):
		return "Nomor sparepart sudah dipakai baris lain."
	case errors.Is(err, mastersparepart.ErrNameTaken):
		return "Nama sparepart sudah dipakai baris lain."
	case errors.Is(err, mastersparepart.ErrCodeTaken):
		return "Kode sparepart sudah dipakai baris lain."
	case errors.Is(err, mastersparepart.ErrNotFound):
		// Barisnya ketemu saat dicari, lalu hilang saat disimpan — petugas lain
		// menghapusnya di antara keduanya.
		return "Baris yang hendak diperbarui sudah tidak ada. Unggah ulang baris ini."
	}
	// Galat yang tidak dikenali TIDAK ditampilkan apa adanya: ia dapat memuat nama tabel
	// atau pesan driver (`11-CROSSCUTTING.md` §1.2 butir 5).
	return "Baris gagal disimpan karena kesalahan tak terduga."
}
