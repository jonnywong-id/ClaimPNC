package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"claim-pnc/internal/masterpanel"
)

// Dua jalur unggah CSV — padanan `PNCUploadMasterPanelCSV` dan `PNCUploadLokasiPanelCSV`.
//
// # Bentuknya upsert, dan kuncinya NAME
//
// Berbeda dari Master Sparepart yang berkunci `NO_SPART`, modul ini tidak punya kolom nomor.
// Kunci alaminya adalah NAMA, dan itu bukan pilihan di sini melainkan bacaan dari
// `RDB List/ValidationMasterPanel-SQL.xml`, yang mencocokkan `upper(trim(name))` — satu-satunya
// kueri pencarian baris yang dimiliki modul ini selain lewat ID.
//
// Ketemu -> barisnya di-UPDATE lewat Save. Tidak ketemu -> baris baru lewat Create, dengan
// ID diterbitkan `PANEL_HE_SEQ`.
//
// # Kenapa memanggil Create dan Save, bukan menulis jalur sendiri
//
// Supaya satu berkas CSV tidak dapat memasukkan data yang form sendiri tolak. Keduanya sudah
// memegang pembersihan, pemeriksaan kesepuluh isian wajib, penolakan nama ganda, dan
// penyetelan status kembali ke menunggu — menyalinnya ke sini berarti dua tempat memutuskan
// hal yang sama, dan yang kedua pasti menyimpang pada perbaikan pertama yang hanya
// diterapkan di salah satunya.
//
// # Setiap baris berdiri sendiri
//
// Satu baris yang gagal TIDAK membatalkan yang lain, mengikuti pola `PNCUploadMasterSparepart_Act`
// yang memutar baris satu per satu tanpa transaksi yang membungkus keseluruhan.
//
// Akibatnya harus disadari: unggahan yang separuh gagal meninggalkan separuh perubahan. Yang
// membuatnya dapat diterima adalah laporannya — setiap baris dinyatakan hasilnya, sehingga
// pengguna tahu persis apa yang masuk dan dapat mengunggah ulang bagian yang gagal saja.
// Membungkus semuanya dalam satu transaksi justru membuat satu salah ketik membatalkan 299
// baris yang benar.

// ImportOutcome menyatakan apa yang terjadi pada satu baris.
type ImportOutcome string

const (
	// ImportCreated: barisnya baru, dan ID-nya baru diterbitkan.
	ImportCreated ImportOutcome = "baru"
	// ImportUpdated: namanya sudah ada, barisnya diperbarui.
	ImportUpdated ImportOutcome = "diperbarui"
	// ImportFailed: barisnya ditolak, dan Message menyebutkan sebabnya.
	ImportFailed ImportOutcome = "gagal"
)

// ImportRow adalah hasil satu baris berkas.
type ImportRow struct {
	// Line adalah nomor baris di BERKAS, header terhitung sebagai baris 1.
	Line int

	// Name adalah nama panelnya — kunci upsert, dan satu-satunya penanda yang pasti ada
	// bahkan pada baris yang gagal.
	Name string

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

// ImportPanelCSV membaca satu berkas master panel dan meng-upsert seluruh barisnya.
// fileName kosong berarti pemanggil tidak menyertakan nama berkasnya; penautan dokumen
// dilewati, dan itu dinyatakan di log alih-alih menggagalkan seluruh unggahan.
func (l *Service) ImportPanelCSV(
	ctx context.Context,
	portalAlias string,
	content []byte,
	fileName string,
	by Actor,
	logger *slog.Logger,
) (ImportReport, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return ImportReport{}, err
	}

	rows, err := masterpanel.ParsePanelCSV(bytes.NewReader(content))
	if err != nil {
		return ImportReport{}, err
	}

	report := ImportReport{Total: len(rows), Rows: make([]ImportRow, 0, len(rows))}

	// Berkas CSV-nya SENDIRI disimpan sebagai dokumen, lalu ditautkan ke setiap baris yang
	// disentuhnya. Itu perilaku Pega, bukan tambahan:
	//
	//	PNCUploadMasterPanel_Act → PNCSaveAttachmentToDB      sekali, di luar perulangan
	//	                         → TempPanel.CoverID := …     pada SETIAP baris
	//
	// AKIBAT YANG HARUS DISADARI: baris yang sudah punya dokumen KEHILANGAN tautannya,
	// diganti berkas CSV ini. Jalur unggah lokasi justru mempertahankannya — asimetri itu
	// ada di sistem lama, dan ditiru apa adanya (`P-5`).
	//
	// Kegagalan menyimpan dokumennya TIDAK membatalkan unggahan: barisnya tetap masuk,
	// hanya tanpa tautan. Membatalkan seluruh berkas karena lampirannya gagal akan menukar
	// kerugian kecil dengan kerugian besar.
	// Berkasnya diunggah lebih dulu; BARIS dokumennya baru dicatat setelah panel pertama
	// berhasil disimpan, karena pencatatan itu sekaligus menautkannya ke sebuah panel.
	imageID := l.uploadCSVFile(ctx, portalAlias, content, fileName, by, logger)
	dataID := ""

	for _, row := range rows {
		// Kata berkas diterjemahkan menjadi sandi LEBIH DULU — lihat masterpanel.ApplyCSVWords.
		// Tanpa ini, teks "TIDAK" tersimpan ke kolom yang seharusnya berisi "0".
		translated := masterpanel.ApplyCSVWords(row.Input)
		clean := translated.Clean()
		result := ImportRow{Line: row.Line, Name: clean.Name}

		id, lookupErr := existingIDOf(ctx, store, clean.Name)
		if lookupErr != nil {
			result.Outcome = ImportFailed
			result.Message = "Baris tidak dapat diperiksa terhadap data yang ada."
			report.Failed++
			report.Rows = append(report.Rows, result)
			// Kegagalan membaca bukan kesalahan barisnya; ia menandakan basis datanya
			// sedang bermasalah. Dicatat supaya tidak hilang di antara kegagalan validasi.
			if logger != nil {
				logger.ErrorContext(ctx, "gagal mencari panel saat unggah CSV",
					slog.String("modul", "masterpanel"),
					slog.Int("baris", row.Line),
					slog.String("galat", lookupErr.Error()))
			}
			continue
		}

		var saved masterpanel.Panel
		var writeErr error
		if id == "" {
			saved, writeErr = l.Create(ctx, portalAlias, translated, by, nil)
			result.Outcome = ImportCreated
		} else {
			saved, writeErr = l.Save(ctx, portalAlias, id, translated, by, nil)
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
		if imageID != "" {
			dataID = l.attachToPanel(ctx, store, imageID, dataID, saved.ID, fileName, by, logger)
		}
		if result.Outcome == ImportCreated {
			report.Created++
		} else {
			report.Updated++
		}
		report.Rows = append(report.Rows, result)
	}

	logImport(ctx, logger, "unggah CSV master panel selesai", portalAlias, by, report)
	return report, nil
}

// ImportLocationCSV membaca satu berkas lokasi panel dan menambahkannya ke panelnya.
//
// # Ia MENAMBAH, persis seperti Pega
//
// `Activity/PNCUploadLokasiSisiPanel_Act` memakai `TempLokasiPanel.LOKASI(<APPEND>)` dan
// menyalin seluruh isian induk dari panel yang ditemukan — sehingga lokasi lama tetap utuh
// dan tautan dokumennya pun dipertahankan (`CoverID` disalin apa adanya).
//
// Tidak ada satu pun pemeriksaan lokasi ganda di sana. Mengunggah berkas yang sama dua kali
// karena itu MENGGANDAKAN barisnya, dan itu ditiru apa adanya (`P-5`).
//
// AKIBAT YANG HARUS DISADARI. Panel yang lokasinya telanjur ganda tidak dapat disimpan
// ulang dari form sampai salah satunya dihapus — form memakai aturan yang penuh. Pengguna
// dapat memperbaikinya sendiri dengan membuang baris yang kelebihan.
//
// # Kenapa SaveWith, bukan menulis langsung ke tabel lokasi
//
// Supaya satu berkas CSV tidak dapat memasukkan data yang form sendiri tolak — kecuali satu
// aturan yang Pega memang longgarkan, dan kelonggaran itu dinyatakan eksplisit di
// pemanggilannya. Satu panel disimpan SEKALI meski berkasnya menyebut beberapa barisnya.
func (l *Service) ImportLocationCSV(
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

	rows, err := masterpanel.ParseLocationCSV(bytes.NewReader(content))
	if err != nil {
		return ImportReport{}, err
	}

	report := ImportReport{Total: len(rows), Rows: make([]ImportRow, 0, len(rows))}

	// Dikelompokkan menurut URUTAN KEMUNCULAN panelnya, bukan menurut peta — supaya
	// laporannya terbaca mengikuti berkasnya, dan supaya hasilnya tidak berubah-ubah
	// antar-jalan karena urutan peta Go memang acak.
	order := make([]string, 0, len(rows))
	grouped := map[string][]masterpanel.LocationCSVRow{}
	for _, row := range rows {
		key := strings.ToUpper(strings.TrimSpace(row.PanelID))
		if key == "" {
			key = "NAMA:" + strings.ToUpper(strings.TrimSpace(row.PanelName))
		}
		if _, seen := grouped[key]; !seen {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], row)
	}

	for _, key := range order {
		group := grouped[key]
		panel, found, lookupErr := panelOfLocationRow(ctx, store, group[0])

		if lookupErr != nil || !found {
			message := "Panel tidak ditemukan. Unggah master panelnya lebih dulu."
			if lookupErr != nil {
				message = "Panel tidak dapat diperiksa terhadap data yang ada."
				if logger != nil {
					logger.ErrorContext(ctx, "gagal mencari panel saat unggah CSV lokasi",
						slog.String("modul", "masterpanel"),
						slog.Int("baris", group[0].Line),
						slog.String("galat", lookupErr.Error()))
				}
			}
			for _, row := range group {
				report.Failed++
				report.Rows = append(report.Rows, ImportRow{
					Line:    row.Line,
					Name:    labelOf(row),
					Outcome: ImportFailed,
					Message: message,
				})
			}
			continue
		}

		input := inputOf(panel)
		added := make([]masterpanel.ImportLocationResult, 0, len(group))
		for _, row := range group {
			location := masterpanel.PanelLocation{
				Name: strings.ToUpper(strings.TrimSpace(row.Location.Name)),
				Side: row.Location.Side,
			}
			// Lokasi yang SUDAH ADA tetap ditambahkan lagi, dan itu mengikuti Pega:
			// `LOKASI(<APPEND>)` tanpa satu pun pemeriksaan. Mengunggah berkas yang sama
			// dua kali karena itu menggandakan barisnya — di sini maupun di sistem lama.
			//
			// Dilaporkan `diperbarui`, bukan `baru`, supaya pengguna melihat bahwa barisnya
			// memang sudah ada sebelumnya.
			sudahAda := containsLocation(input.Location, location)
			input.Location = append(input.Location, location)
			added = append(added, masterpanel.ImportLocationResult{
				Line: row.Line, Existing: sudahAda,
			})
		}

		saved, writeErr := l.SaveWith(ctx, portalAlias, panel.ID, input, by, nil,
			masterpanel.CheckOption{AllowDuplicateLocation: true})
		if writeErr != nil {
			message := reasonOf(writeErr)
			for _, row := range group {
				report.Failed++
				report.Rows = append(report.Rows, ImportRow{
					Line:    row.Line,
					Name:    labelOf(row),
					Outcome: ImportFailed,
					Message: message,
				})
			}
			continue
		}

		for index, row := range group {
			outcome := ImportCreated
			if added[index].Existing {
				// Lokasi yang sudah ada BUKAN kegagalan: mengunggah ulang berkas yang sama
				// harus aman, dan melaporkannya gagal akan membuat pengguna mencari
				// kesalahan yang tidak ada.
				outcome = ImportUpdated
				report.Updated++
			} else {
				report.Created++
			}
			report.Rows = append(report.Rows, ImportRow{
				Line:    row.Line,
				Name:    labelOf(row),
				ID:      saved.ID,
				Outcome: outcome,
			})
		}
	}

	logImport(ctx, logger, "unggah CSV lokasi panel selesai", portalAlias, by, report)
	return report, nil
}

// panelOfLocationRow mencari panel yang dirujuk satu baris lokasi.
//
// ID didahulukan atas nama: bila berkasnya membawa keduanya, ID yang lebih pasti.
func panelOfLocationRow(
	ctx context.Context, store masterpanel.Store, row masterpanel.LocationCSVRow,
) (masterpanel.Panel, bool, error) {
	if id := strings.TrimSpace(row.PanelID); id != "" {
		return panelWithLocation(ctx, store, id)
	}

	name := strings.TrimSpace(row.PanelName)
	if name == "" {
		return masterpanel.Panel{}, false, nil
	}
	found, err := store.FindByName(ctx, name)
	if errors.Is(err, masterpanel.ErrNotFound) {
		return masterpanel.Panel{}, false, nil
	}
	if err != nil {
		return masterpanel.Panel{}, false, fmt.Errorf(
			"masterpanel/usecase: mencari panel %q: %w", name, err)
	}

	// Dibaca ULANG lewat Get, dan itu BUKAN perjalanan yang mubazir.
	//
	// `FindByName` sengaja TIDAK membaca daftar lokasinya — pemanggil aslinya hanya perlu
	// tahu barisnya ada dan apa kuncinya. Memakai hasilnya apa adanya di sini membuat
	// `inputOf` menyusun Input berlokasi KOSONG, dan Save kemudian MENGGANTI seluruh
	// lokasi panel itu dengan baris CSV saja.
	//
	// Akibatnya persis yang jalur ini dirancang untuk dicegah: berkas berisi satu baris
	// memusnahkan lokasi lain panel itu, tanpa satu pun galat. Ditangkap uji
	// `TestImporLokasiMenambahBukanMengganti`, bukan oleh pembacaan ulang kode.
	return panelWithLocation(ctx, store, found.ID)
}

// panelWithLocation membaca panel LENGKAP dengan daftar lokasinya.
func panelWithLocation(
	ctx context.Context, store masterpanel.Store, id string,
) (masterpanel.Panel, bool, error) {
	full, err := store.Get(ctx, strings.TrimSpace(id))
	if errors.Is(err, masterpanel.ErrNotFound) {
		return masterpanel.Panel{}, false, nil
	}
	if err != nil {
		return masterpanel.Panel{}, false, fmt.Errorf(
			"masterpanel/usecase: membaca panel %q: %w", id, err)
	}
	return full, true, nil
}

// labelOf memberi penanda baris lokasi untuk laporan.
func labelOf(row masterpanel.LocationCSVRow) string {
	if name := strings.TrimSpace(row.PanelName); name != "" {
		return name
	}
	return strings.TrimSpace(row.PanelID)
}

// containsLocation menyatakan lokasi itu sudah ada di daftar.
//
// Pembandingannya tidak peka huruf besar-kecil pada nama, sejalan dengan Clean() yang
// menghurufbesarkannya saat disimpan.
func containsLocation(list []masterpanel.PanelLocation, one masterpanel.PanelLocation) bool {
	for _, existing := range list {
		if strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(one.Name)) &&
			existing.Side == one.Side {
			return true
		}
	}
	return false
}

// inputOf menyalin panel yang sudah ada menjadi Input untuk disimpan ulang.
//
// Seluruh isian induk disalin apa adanya: berkas lokasi TIDAK menyentuh satu pun di
// antaranya, dan mengosongkannya akan membuat unggahan lokasi diam-diam menghapus isian
// panelnya.
func inputOf(p masterpanel.Panel) masterpanel.Input {
	location := make([]masterpanel.PanelLocation, len(p.Location))
	copy(location, p.Location)
	return masterpanel.Input{
		Name:                p.Name,
		RepairStatus:        p.RepairStatus,
		EditQuantityStatus:  p.EditQuantityStatus,
		PremiumRepairStatus: p.PremiumRepairStatus,
		ShatterStatus:       p.ShatterStatus,
		StickerStatus:       p.StickerStatus,
		SideStatus:          p.SideStatus,
		SevereDamageStatus:  p.SevereDamageStatus,
		ActiveStatus:        p.ActiveStatus,
		ExclusionC:          p.ExclusionC,
		Location:            location,
	}
}

// existingIDOf mengembalikan ID baris bernama itu, atau "" bila belum ada.
//
// Padanan `RDB List/ValidationMasterPanel-SQL.xml`. Nama kosong TIDAK dicari: ia pasti
// ditolak pemeriksaan isian sesaat kemudian, dan mencarinya lebih dulu hanya menambah satu
// perjalanan ke basis data untuk baris yang sudah pasti gagal.
func existingIDOf(ctx context.Context, store masterpanel.Store, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil
	}
	found, err := store.FindByName(ctx, name)
	if errors.Is(err, masterpanel.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("masterpanel/usecase: mencari nama %q: %w", name, err)
	}
	return strings.TrimSpace(found.ID), nil
}

// reasonOf mengubah galat tulis menjadi kalimat untuk laporan.
//
// Pelanggaran validasi digabung menjadi satu kalimat, bukan dibiarkan sebagai daftar:
// laporan per baris hanya punya satu kolom sebab, dan daftar bersarang di dalam tabel
// membuatnya tidak terbaca.
func reasonOf(err error) string {
	var invalid *masterpanel.ValidationError
	if errors.As(err, &invalid) {
		parts := make([]string, 0, len(invalid.Violation))
		for _, one := range invalid.Violation {
			parts = append(parts, one.Message)
		}
		return strings.Join(parts, " ")
	}
	if errors.Is(err, masterpanel.ErrNameTaken) {
		return "Nama panel tersebut telah digunakan panel lain."
	}
	return "Baris tidak dapat disimpan."
}

// logImport mencatat ringkasan satu unggahan.
func logImport(
	ctx context.Context, logger *slog.Logger, message, portalAlias string,
	by Actor, report ImportReport,
) {
	if logger == nil {
		return
	}
	logger.InfoContext(ctx, message,
		slog.String("modul", "masterpanel"),
		slog.String("portal", portalAlias),
		slog.String("oleh", strings.TrimSpace(by.Login)),
		slog.Int("total", report.Total),
		slog.Int("baru", report.Created),
		slog.Int("diperbarui", report.Updated),
		slog.Int("gagal", report.Failed))
}

// uploadCSVFile mengirim berkas CSV-nya sendiri ke layanan penyimpanan.
//
// Mengembalikan "" bila tidak dapat dikerjakan — tanpa pengunggah, tanpa nama berkas, atau
// gagal di tengah jalan. Unggahan barisnya TETAP diteruskan: membatalkan seluruh berkas
// karena lampirannya gagal menukar kerugian kecil dengan kerugian besar.
func (l *Service) uploadCSVFile(
	ctx context.Context, portalAlias string, content []byte, fileName string,
	by Actor, logger *slog.Logger,
) string {
	if l.uploader == nil || strings.TrimSpace(fileName) == "" || len(content) == 0 {
		return ""
	}

	imageID, err := l.uploader.Upload(ctx, masterpanel.DocumentFile{
		Portal:   portalAlias,
		FileName: fileName,
		Content:  content,
		By:       by.Login,
	})
	if err != nil {
		if logger != nil {
			logger.ErrorContext(ctx, "berkas CSV gagal diunggah sebagai dokumen",
				slog.String("modul", "masterpanel"),
				slog.String("galat", err.Error()))
		}
		return ""
	}
	return imageID
}

// attachToPanel menautkan berkas CSV ke satu panel, dan mengembalikan DATAID-nya.
//
// Baris `DATA_ATTACHFILE`-nya dicatat SEKALI — pada panel pertama yang berhasil disimpan —
// lalu panel berikutnya hanya ditautkan ke baris yang sama. Mencatatnya per panel akan
// menerbitkan satu baris lampiran per panel untuk berkas yang sama persis.
//
// Kegagalan di sini hanya dicatat; ia tidak menggagalkan barisnya, yang sudah tersimpan.
func (l *Service) attachToPanel(
	ctx context.Context, store masterpanel.Store,
	imageID, dataID, panelID, fileName string, by Actor, logger *slog.Logger,
) string {
	catat := func(pesan string, err error) {
		if logger != nil {
			logger.ErrorContext(ctx, pesan,
				slog.String("modul", "masterpanel"),
				slog.String("panel", panelID),
				slog.String("galat", err.Error()))
		}
	}

	if dataID != "" {
		if err := store.LinkDocument(ctx, panelID, dataID); err != nil {
			catat("gagal menautkan berkas CSV sebagai dokumen panel", err)
		}
		return dataID
	}

	now := time.Now()
	baru, err := store.SaveDocument(ctx, masterpanel.PanelDocument{
		ImageID:    imageID,
		Name:       fileName,
		UploadedBy: by.Login,
		UploadedAt: &now,
		PanelID:    panelID,
	})
	if err != nil {
		catat("gagal mencatat berkas CSV sebagai dokumen panel", err)
		return ""
	}
	return baru
}
