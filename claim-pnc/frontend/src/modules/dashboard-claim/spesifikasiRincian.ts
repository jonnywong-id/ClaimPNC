import type { Isian, KolomGrid } from './BagianRincian'

/**
 * Spesifikasi isi popup rincian klaim.
 *
 * # Setiap label DISALIN, bukan dikarang
 *
 * Seluruh teks `label` di berkas ini diambil apa adanya dari elemen `pyLabelFieldValue` pada
 * section Pega yang bersangkutan. Itu termasuk ejaannya yang tidak konsisten
 * ("No Telephone Pelapor") dan judul gabungannya
 * ("Tanggal Kejadian / Tanggal Masuk Rawat Inap") — keduanya dibawa apa adanya (`D-13`).
 *
 * # Kenapa label dan jalur berpasangan di SATU tempat
 *
 * Nama propertinya menyesatkan — `.City` memuat nomor klaim, `.Currency` memuat nomor polis
 * (utang teknis §4.2). Memisahkan label dari jalurnya akan membuat keduanya menyimpang diam-
 * diam, dan hasilnya label yang benar di atas nilai yang salah: cacat yang tidak terlihat
 * sebagai cacat.
 */

/**
 * Bagian **Register** — `Section/ViewInputRegisterDetail-Section.xml`.
 *
 * Dua puluh label nyata ada di section itu; yang di bawah adalah yang berpasangan dengan
 * jalur dokumen. Sisanya melekat pada kendali yang tidak membawa nilai (tombol, penanda
 * pengiriman) dan karena itu tidak digambar sebagai isian.
 */
export const ISIAN_REGISTER: Isian[] = [
  { label: 'Tanggal Terima Dokumen', jalur: 'ClaimData.DateReceived' },
  { label: 'Tanggal Kejadian / Tanggal Masuk Rawat Inap', jalur: 'ClaimData.DateOfLoss' },
  { label: 'Tanggal Keluar Rawat Inap', jalur: 'ClaimData.TanggalSelesaiRawatInap' },
  { label: 'Tanggal Lapor', jalur: 'ClaimData.ReportDate' },
  { label: 'No Telephone Pelapor', jalur: 'ClaimData.ReporterTelp' },
  { label: 'Email', jalur: 'ClaimData.Email' },
  { label: 'Status Pelapor', jalur: 'ClaimData.InsuredRelationship' },
  { label: 'Sebutkan...', jalur: 'ClaimData.InsuredRelationshipOthers' },
  { label: 'Klaim Estimasi', jalur: 'ClaimData.ClaimEstimate' },
  { label: 'Nilai Dalam IDR', jalur: 'ClaimData.DollarCurrencyVal' },
  { label: 'Lokasi Kerugian/Kejadian', jalur: 'ClaimData.Location' },
  { label: 'Kelurahan', jalur: 'ClaimData.District' },
  { label: 'Apakah tertanggung seorang TKI?', jalur: 'ClaimData.TKI' },
  { label: 'Catatan dari Analyst', jalur: 'ClaimData.AnaylstRemarks' },
  { label: 'Catatan Ke Analyst', jalur: 'ClaimData.Remark' },
  { label: 'Catatan Ke PIC Teknis', jalur: 'ClaimData.RemarkRecommendation' },
]

/**
 * Grid objek pertanggungan pada bagian Register.
 *
 * Kolomnya dari ikatan sel `ASM-FW-GCNMFW-Data-Object` di section yang sama. Labelnya memakai
 * istilah `CONTEXT.md` — section Pega-nya tidak membawa `pyLabelFieldValue` untuk kolom grid,
 * dan nama propertinya sendiri (`ObjectName`, `ObjectIDCard`) sudah terbaca apa adanya.
 */
export const KOLOM_OBJEK: KolomGrid[] = [
  { label: 'Nama Objek', kunci: 'ObjectName' },
  { label: 'Lokasi Objek', kunci: 'ObjectLocation' },
  { label: 'Status Peserta', kunci: 'ObjectParticipantStatus' },
  { label: 'NIK', kunci: 'ObjectIDCard' },
  { label: 'Tanggal Lahir', kunci: 'ObjectDateOfBirth' },
  { label: 'Pekerjaan', kunci: 'ObjectJob' },
  { label: 'Tipe Kendaraan', kunci: 'ObjectVehicleType' },
  { label: 'No Rangka', kunci: 'ObjectVehicleChasis' },
]

/**
 * Grid **Penerima Klaim** — `Section/ViewInputReceiver-Section.xml`.
 *
 * Seluruh judul DISALIN dari rujukan `pyCaption` section itu — sepuluh caption terbaca di
 * sana, dan ketujuh yang berpasangan dengan properti dipakai di bawah.
 *
 * Pasangannya langsung, bukan tafsiran: `Address` → Alamat, `NoAccount` → No Rekening,
 * `NameOfBank` → Nama Bank, `BranchOfBank` → Nama Cabang Bank, `NomorKontrak` → Nomor Kontrak.
 *
 * TIGA caption TIDAK dipakai, dan sebabnya dicatat supaya tidak dikira terlewat:
 *
 *	"Nama Rekening"           tidak ada properti yang berpasangan dengannya di section ini
 *	"No Telepon Pelapor"      ejaan kedua untuk kolom yang sama dengan "Telepon"
 *	"No Telephone Pelapor"    ejaan ketiga, idem — ketiganya menunjuk `.Telephone`
 */
/**
 * Grid tab **Penerima Pembayaran Klaim** — `Section/ViewShowReceiver-Section.xml`.
 *
 * KOREKSI (2026-10-08): grid ini sebelumnya menggambar TUJUH kolom — Telepon, Nomor
 * Kontrak, No Rekening, Nama Bank, Nama Cabang Bank ikut digambar. Kelimanya **karangan**.
 *
 * Section Pega-nya hanya punya DUA kolom, dan tangkapan layar layar lama memperlihatkan hal
 * yang sama. Kelima yang lain memang ada datanya — tetapi di layar lama ia muncul pada
 * sub-popup baris, bukan di grid.
 */
export const KOLOM_PENERIMA: KolomGrid[] = [
  { label: 'Nama', kunci: 'Name' },
  { label: 'Alamat', kunci: 'Address' },
]

/**
 * Bagian **Hasil Survey** — `Section/ViewHasilSurvey-Section.xml`.
 *
 * Jalurnya bersarang di `ClaimData.SurveyData`, bukan di akar `ClaimData` — dibaca apa adanya
 * dari ikatan selnya.
 */
export const ISIAN_SURVEY: Isian[] = [
  { label: 'Survey Atas Permintaan', jalur: 'ClaimData.SurveyData.RequestorName' },
  { label: 'Lokasi Survey', jalur: 'ClaimData.SurveyData.LocationSurvey' },
  { label: 'No. Telp', jalur: 'ClaimData.SurveyData.NoTelpDiHubungi' },
  { label: 'Tanggal Request Survey', jalur: 'ClaimData.SurveyData.RequestSurveyDate' },
  { label: 'Cabang', jalur: 'ClaimData.SurveyData.Cabang' },
  { label: 'Surveyor', jalur: 'ClaimData.SurveyData.SurveyorName' },
  { label: 'Email Surveyor', jalur: 'ClaimData.SurveyData.EmailSurvey' },
  { label: 'Nama Object', jalur: 'ClaimData.SurveyData.ObjectName' },
]

/**
 * Bagian **Catatan ke Analyst** — `Section/CatatanToAnalyst_Section-Section.xml`.
 *
 * Label "Note PilihanCompliance" dibawa APA ADANYA, termasuk kata tersambungnya — itu yang
 * tertulis di `pyLabelFieldValue`, dan `D-13` menetapkan teks layar mengikuti Pega.
 */
export const ISIAN_CATATAN: Isian[] = [
  { label: 'Pilihan Compliance', jalur: 'ClaimData.PilihanCompliance' },
  { label: 'Note PilihanCompliance', jalur: 'ClaimData.NotePilihanCompliance' },
  {
    label: 'Tanggal Kelengkapan Dokumen Akhir PUCL',
    jalur: 'ClaimData.PUCLStatus.TanggalTerimaDokumenPUCL',
  },
  { label: 'Alasan Dokter Tolak RCL', jalur: 'ClaimData.AlasanDokterRejectRCL' },
]

/**
 * Grid **Coverage** — `Section/ViewObjectCoverageAdj-Section.xml`.
 *
 * # Judulnya DISALIN dari Pega, bukan dari glossary
 *
 * Judul kolom grid tidak tersimpan sebagai `pyLabelFieldValue` melainkan sebagai rujukan
 * Field Value, yang nama rule-nya memuat teksnya sendiri:
 *
 *	<pyRuleName>pyCaption Nama Coverage</pyRuleName>
 *
 * Kelimanya diambil dari sana apa adanya. Bentuk pertama memakai padanan `CONTEXT.md`
 * ("Kode Coverage", "Keterangan Coverage") — itu glossary yang disepakati, tetapi BUKAN
 * salinan Pega, dan Work Owner menegaskan layar mengikuti Pega apa adanya (`D-13`).
 *
 * Satu akibatnya langsung terlihat: Pega menggambar **Mata Uang** dan **Nama Plan**, dua kolom
 * yang tidak ada di bentuk karangan itu — sementara "Kode Coverage" ternyata tidak pernah ada.
 */
export const KOLOM_COVERAGE: KolomGrid[] = [
  { label: 'Nama Coverage', kunci: 'CoverageNote' },
  { label: 'Nama Plan', kunci: 'CoverageID' },
  { label: 'TSI', kunci: 'SumTSI' },
  { label: 'Mata Uang', kunci: 'Currency' },
  { label: 'Penyebab Kerugian', kunci: 'CauseOfLoss' },
]

/**
 * Grid **Dokumen Klaim** — `Section/GCNMViewAttachment-Section.xml`.
 *
 * Ketiga judul disalin dari rujukan `pyCaption` section itu: "Nama", "Kategori Dokumen",
 * "Jenis Dokumen". Kuncinya dari ikatan selnya — `ATTACHNAME`, `ATTACHNOTE`, `DATAID`.
 */
/**
 * Grid tab **Dokumen** — `Section/ViewUploadDocument-Section.xml`.
 *
 * KOREKSI (2026-10-08): ketiga kolom sebelumnya (Nama, Kategori Dokumen, Jenis Dokumen)
 * **karangan** — tidak satu pun ada di section itu.
 *
 * Section-nya memuat satu kolom lagi, "Unggah Dokumen", yang isinya tombol unggah. Tidak
 * digambar: modul ini hanya membaca selama masa paralel (`P-1`).
 *
 * `.RISK` memuat jumlah dokumen yang sudah diunggah — alias menyesatkan, dipakai apa adanya.
 */
export const KOLOM_DOKUMEN: KolomGrid[] = [
  { label: 'Kategori', kunci: 'DETAIL_DOCUMENT' },
  { label: 'Wajib Unggah', kunci: 'STS_WAJIB' },
  { label: 'Minimal Unggah', kunci: 'MIN_DOC' },
  { label: 'Total Sudah Diunggah', kunci: 'RISK' },
]

/**
 * Grid **Riwayat Lampiran** — `Section/GCNMViewAttachmentHistory-Section.xml`.
 *
 * Keempat judulnya BERBAHASA INGGRIS di Pega — "Attachment", "Category", "Created date",
 * "Name" — dan dibawa apa adanya (`D-13`). Menerjemahkannya akan membuat layar ini berbeda
 * dari yang dikenal pengguna, dan tiga section bersebelahan memakai bahasa Indonesia; campuran
 * itu memang ada di sistem lama.
 */
/**
 * Grid **Riwayat Lampiran** — `Section/GCNMViewAttachmentHistory-Section.xml`.
 *
 * KOREKSI (2026-10-08): urutannya menjadi Category -> Name -> Created date mengikuti Pega,
 * dan kolom keempat ("Attachment") DIBUANG — ia tidak ada di section itu.
 *
 * Satu penyimpangan yang disengaja: Pega mengikat kolom Name ke `.pyTemplateInputBox`,
 * yaitu KENDALI yang merender tautan, bukan field data. Kita memakai `pyAttachName` karena
 * itulah kunci yang membawa namanya di dokumen JSON; memakai nama kendali akan
 * menghasilkan kolom kosong.
 */
export const KOLOM_RIWAYAT_LAMPIRAN: KolomGrid[] = [
  { label: 'Category', kunci: 'pyCategory' },
  { label: 'Name', kunci: 'pyAttachName' },
  { label: 'Created date', kunci: 'pxCreateDateTime' },
]

/**
 * Grid **Posisi Klaim** — `Section/InputPosisiClaim-Section.xml`.
 *
 * Keempat judul disalin apa adanya, TERMASUK huruf besarnya: Pega menulis "POSISI" dan
 * "TANGGAL INPUT" kapital penuh sementara "Case ID" dan "Status" tidak. Menyeragamkannya
 * adalah perubahan tampilan yang tidak diminta siapa pun.
 */
/**
 * Grid tab **Progress Klaim** — `Section/ProgressCloseClaim-Section.xml`.
 *
 * Judulnya disalin apa adanya, termasuk `POSISI` dan `STATUS` yang huruf besar semua
 * sementara `Case ID` dan `Tanggal` tidak (`D-13`).
 *
 * Nama propertinya MENYESATKAN seperti di tempat lain: posisi disimpan pada
 * `.AlasanTerlambat`, status pada `.CountryID`, tanggal pada `.KomiteApproveDate`.
 * Ketiganya dipakai apa adanya karena itulah kunci di dokumen JSON-nya.
 */
export const KOLOM_POSISI: KolomGrid[] = [
  { label: 'Case ID', kunci: 'CaseID' },
  { label: 'POSISI', kunci: 'AlasanTerlambat' },
  { label: 'Tanggal', kunci: 'KomiteApproveDate' },
  { label: 'STATUS', kunci: 'CountryID' },
]

/**
 * Grid **Objek HE** — `Section/ShowObjectHEPICTeknis_komite-Section.xml`.
 *
 * Section ini hanya digambar untuk lini **Heavy Equipment** (When `IsHE`). Kesembilan judul
 * disalin dari rujukan `pyCaption`-nya; pasangan propertinya langsung — `ObjectVehicleBrand`
 * → Merk, `ObjectVehicleModel` → Model, `ObjectVehicleType` → Nama Tipe,
 * `ObjectVehicleChasis` → Nomor Chasis.
 *
 * Tiga kolom terakhir menyangkut komite — `KomiteID`, `KomiteAproval`, `KomiteComment` —
 * dan ejaan "Aproval" dibawa apa adanya dari nama propertinya.
 */
/**
 * Grid objek tab Heavy Equipment — `Section/ShowObjectHEPICTeknis_komite-Section.xml`.
 *
 * KOREKSI (2026-10-08): tiga kolom komite (Nama Komite, Status, Komentar) dikeluarkan.
 * Di Pega ketiganya **grid tersendiri** di section yang sama, bukan sambungan kolom grid
 * objek — lihat KOLOM_KOMITE_HE. Urutan kolomnya juga dikoreksi mengikuti Pega.
 */
export const KOLOM_OBJEK_HE: KolomGrid[] = [
  { label: 'Object', kunci: 'ObjectName' },
  { label: 'Model', kunci: 'ObjectVehicleModel' },
  { label: 'Merk', kunci: 'ObjectVehicleBrand' },
  { label: 'Nama Tipe', kunci: 'ObjectVehicleType' },
  { label: 'Nomor Chasis', kunci: 'ObjectVehicleChasis' },
  { label: 'Location', kunci: 'ObjectLocation' },
]

/**
 * Isian kepala popup — label yang dipakai BERSAMA oleh beberapa section.
 *
 * "Aging Amount", "Status Klaim", dan "Catatan dari Inputor" muncul sebagai
 * `pyLabelFieldValue` di `ViewInputRegisterDetail`, `ViewInputEstimasiDetail`, dan
 * `ViewShowReceiver` sekaligus — tanda ia milik kerangka popup, bukan satu bagian tertentu.
 */
export const ISIAN_KEPALA: Isian[] = [
  { label: 'Status Klaim', jalur: 'ClaimData.StatusClaim' },
  { label: 'Aging Amount', jalur: 'Policy.AgingAmount' },
  { label: 'Catatan dari Inputor', jalur: 'ClaimData.Remark' },
]

/**
 * Sub-popup **Detail Hasil Surveyor** — `DetailHasilSurveyor_Sect`, dibuka Flow Action
 * `DetailHasilSurveyor` dari baris grid Hasil Survey.
 *
 * Section ini **nol** rujukan `pyCaption` dan nol `pyLabelFieldValue` — sudah diperiksa
 * keduanya. Labelnya karena itu memakai nama propertinya apa adanya, BUKAN padanan karangan:
 * nama-nama di sini deskriptif (`LossDescription`, `WitnessName`, `PoliceReport`) dan tidak
 * termasuk alias menyesatkan yang menjangkiti modul ini.
 *
 * Bila judul aslinya kelak ditemukan, yang berubah hanya berkas ini.
 */
export const ISIAN_DETAIL_SURVEYOR: Isian[] = [
  { label: 'SurveyName', jalur: 'SurveyName' },
  { label: 'SurveyDate', jalur: 'SurveyDate' },
  { label: 'SurveyorName', jalur: 'SurveyorName' },
  { label: 'LocationSurvey', jalur: 'LocationSurvey' },
  { label: 'InsuredPIC', jalur: 'InsuredPIC' },
  { label: 'LossType', jalur: 'LossType' },
  { label: 'LossDescription', jalur: 'LossDescription' },
  { label: 'EstimateDescription', jalur: 'EstimateDescription' },
  { label: 'SurveyorDescription', jalur: 'SurveyorDescription' },
  { label: 'PolicyTerm', jalur: 'PolicyTerm' },
  { label: 'PoliceReport', jalur: 'PoliceReport' },
  { label: 'WitnessName', jalur: 'WitnessName' },
  { label: 'WitnessAddress', jalur: 'WitnessAddress' },
  { label: 'Salvage', jalur: 'Salvage' },
  { label: 'SalvageDescription', jalur: 'SalvageDescription' },
  { label: 'ClientSignature', jalur: 'ClientSignature' },
  { label: 'OthersDescription', jalur: 'OthersDescription' },
]

/**
 * Sub-popup **Hasil Survey** — `ShowSurveyResults`, dibuka dari baris grid Hasil Survey.
 *
 * Dua puluh caption terbaca di section itu; yang di bawah adalah yang berpasangan dengan
 * properti. Yang tidak dipakai ("Button", "Dokumen", "Remark", "Komentar") melekat pada
 * kendali yang tidak membawa nilai.
 */
export const ISIAN_HASIL_SURVEY: Isian[] = [
  { label: 'Reference Number', jalur: 'ReferenceNo' },
  { label: 'Appointed No', jalur: 'IDSurvey' },
  { label: 'Appointment Date', jalur: 'AppointmentDate' },
  { label: 'Request Date Appointment', jalur: 'AppointmentDateReq' },
  { label: 'Tanggal Survey', jalur: 'SurveyDate' },
  { label: 'Lokasi Survey', jalur: 'LocationSurvey' },
  { label: 'Nama Surveyor', jalur: 'AdjusterPIC' },
  { label: 'Status Survey', jalur: 'AdjusterStatus' },
  { label: 'Catatan Survey', jalur: 'AdjusterAcceptNote' },
  { label: 'Komite ID', jalur: 'KomiteID' },
  { label: 'Status Komite', jalur: 'KomiteAproval' },
  { label: 'Komentar', jalur: 'KomiteComment' },
  { label: 'Note Terlambat', jalur: 'KeteranganLain' },
]

/**
 * Sub-popup **Detail Progress** — `ShowDetailProgress_sect`, dibuka dari baris grid Posisi.
 *
 * Caption "---" dan "YA" tidak dipakai: keduanya isi dropdown, bukan label isian.
 */
export const ISIAN_DETAIL_PROGRESS: Isian[] = [
  { label: 'Tgl Progress', jalur: 'TglProgress' },
  { label: 'Keterangan', jalur: 'KeteranganProgress' },
  { label: 'PIC', jalur: 'PIC' },
  { label: 'Next Follow Up', jalur: 'NextFollowUp' },
  { label: 'Tgl Next Follow Up', jalur: 'ObjekTanggal' },
  { label: 'Catatan Delivery', jalur: 'Comment' },
]

/**
 * Sub-popup **Coverage HE** — `CoverageGridClaimHE_Komite`.
 *
 * Kesepuluh judul disalin dari rujukan `pyCaption`-nya.
 */
export const KOLOM_COVERAGE_HE: KolomGrid[] = [
  { label: 'Coverage', kunci: 'CoverageNote' },
  { label: 'Penyebab Kerugian', kunci: 'CauseOfLoss' },
  { label: 'Merk', kunci: 'ObjectVehicleBrand' },
  { label: 'Model', kunci: 'ObjectVehicleModel' },
  { label: 'Type', kunci: 'ObjectVehicleType' },
  { label: 'Tahun Kendaraan', kunci: 'ObjectVehicleYear' },
  { label: 'No Rangka', kunci: 'ObjectVehicleChasis' },
  { label: 'No Mesin', kunci: 'ObjectVehicleEngineNumber' },
  { label: 'No Serial', kunci: 'SerialNo' },
]

/**
 * Sub-popup **Detail Coverage** — `ViewObjectCoverageAdj`, dibuka Flow Action
 * `ViewCoverageAdj` dari baris grid Coverage.
 *
 * Kelima judul sama dengan judul kolom gridnya — section itu memang menggambar isian yang
 * sama secara vertikal. Yang bertambah hanya `DESCRIPTION`, yang di grid tidak digambar.
 */
export const ISIAN_COVERAGE_DETAIL: Isian[] = [
  { label: 'Nama Coverage', jalur: 'CoverageNote' },
  { label: 'Nama Plan', jalur: 'CoverageID' },
  { label: 'TSI', jalur: 'SumTSI' },
  { label: 'Mata Uang', jalur: 'Currency' },
  { label: 'Penyebab Kerugian', jalur: 'CauseOfLoss' },
]

/**
 * Grid **Hasil Survey** — barisnya membuka sub-popup `ShowSurveyResults`.
 *
 * Kolomnya diambil dari caption section itu; sisanya digambar di sub-popup-nya, persis seperti
 * Pega yang menaruh sebagian besar isian di balik `pyEditAction`.
 */
/**
 * Grid pertama tab **Hasil Survey** — `Section/ViewHasilSurvey-Section.xml`.
 *
 * KOREKSI (2026-10-08): "Reference Number" dan "Appointed No" dibuang — keduanya tidak ada
 * di section itu. "Tanggal Survey" dikoreksi menjadi **"Tanggal Pengajuan Survey"**, dan dua
 * kolom yang hilang (Nama Objek, Lokasi Objek) ditambahkan.
 *
 * Lokasi objek terikat `.RescheduleLocation` — sekali lagi alias yang menyesatkan.
 */
export const KOLOM_HASIL_SURVEY: KolomGrid[] = [
  { label: 'Tanggal Pengajuan Survey', kunci: 'SurveyDate' },
  { label: 'Nama Surveyor', kunci: 'SurveyorName' },
  { label: 'Nama Objek', kunci: 'ObjectName' },
  { label: 'Lokasi Objek', kunci: 'RescheduleLocation' },
  { label: 'Status', kunci: 'SurveyStatus' },
]

/**
 * Grid **Detail Surveyor** — barisnya membuka sub-popup `DetailHasilSurveyor_Sect`.
 *
 * Section itu nol caption, sehingga kolom DAN isian sub-popup-nya memakai nama properti apa
 * adanya. Lihat catatan pada `ISIAN_DETAIL_SURVEYOR`.
 */
export const KOLOM_DETAIL_SURVEYOR: KolomGrid[] = [
  { label: 'SurveyName', kunci: 'SurveyName' },
  { label: 'SurveyDate', kunci: 'SurveyDate' },
  { label: 'SurveyorName', kunci: 'SurveyorName' },
  { label: 'LocationSurvey', kunci: 'LocationSurvey' },
]

/**
 * Sub-popup **Coverage HE** — `CoverageGridClaimHE_Komite`, dibuka Flow Action
 * `KomiteCoverageGrid_ClaimHE` dari baris grid Objek HE.
 *
 * Kesembilan judul disalin dari rujukan `pyCaption` section itu.
 */
export const ISIAN_COVERAGE_HE: Isian[] = [
  { label: 'Coverage', jalur: 'CoverageNote' },
  { label: 'Penyebab Kerugian', jalur: 'CauseOfLoss' },
  { label: 'Merk', jalur: 'ObjectVehicleBrand' },
  { label: 'Model', jalur: 'ObjectVehicleModel' },
  { label: 'Type', jalur: 'ObjectVehicleType' },
  { label: 'Tahun Kendaraan', jalur: 'ObjectVehicleYear' },
  { label: 'No Rangka', jalur: 'ObjectVehicleChasis' },
  { label: 'No Mesin', jalur: 'ObjectVehicleEngineNumber' },
  { label: 'No Serial', jalur: 'SerialNo' },
]

/**
 * Sub-popup **Penerima Klaim** — `ViewInputReceiver`, dibuka Flow Action bernama sama dari
 * baris grid Penerima.
 *
 * Isinya LEBIH LENGKAP daripada gridnya: tiga caption yang di grid tidak muat — "Nama
 * Rekening" di antaranya — digambar di sini. Itu bentuk Pega: grid menunjukkan ringkasan,
 * `pyEditAction` membuka seluruhnya.
 */
export const ISIAN_PENERIMA_DETAIL: Isian[] = [
  { label: 'Nama', jalur: 'Name' },
  { label: 'Alamat', jalur: 'Address' },
  { label: 'Telepon', jalur: 'Telephone' },
  { label: 'Nomor Kontrak', jalur: 'NomorKontrak' },
  { label: 'No Rekening', jalur: 'NoAccount' },
  { label: 'Nama Bank', jalur: 'NameOfBank' },
  { label: 'Nama Cabang Bank', jalur: 'BranchOfBank' },
]

/**
 * Sub-popup **Coverage (grid)** — `ViewCoverageGridContent`, dibuka Flow Action
 * `ViewCoverageGrid_fa`.
 *
 * Captionnya sama dengan `ViewObjectCoverageAdj` ditambah **Coverage** — section berbeda,
 * isian hampir sama. Dibawa apa adanya, bukan disatukan: keduanya dipanggil dari tempat yang
 * berbeda di Pega, dan menyatukannya menghapus perbedaan yang mungkin bermakna.
 */
export const ISIAN_COVERAGE_GRID: Isian[] = [
  { label: 'Coverage', jalur: 'CoverageNote' },
  { label: 'Nama Plan', jalur: 'CoverageID' },
  { label: 'TSI', jalur: 'SumTSI' },
  { label: 'Mata Uang', jalur: 'Currency' },
  { label: 'Penyebab Kerugian', jalur: 'CauseOfLoss' },
]


/**
 * Grid **komite** pada tab Heavy Equipment — grid kedua
 * `Section/ShowObjectHEPICTeknis_komite-Section.xml`.
 *
 * Dipisahkan dari KOLOM_OBJEK_HE pada 2026-10-08. Menggabungkannya membuat satu baris objek
 * seolah membawa satu keputusan komite — padahal keduanya daftar yang berbeda panjang.
 */
export const KOLOM_KOMITE_HE: KolomGrid[] = [
  { label: 'Nama Komite', kunci: 'KomiteID' },
  { label: 'Status', kunci: 'KomiteAproval' },
  { label: 'Komentar', kunci: 'KomiteComment' },
]

/** Grid kedua tab **Hasil Survey** — varian investigasi di section yang sama. */
export const KOLOM_INVESTIGASI: KolomGrid[] = [
  { label: 'Tanggal Investigasi', kunci: 'SurveyDate' },
  { label: 'Nama Peserta', kunci: 'ObjectName' },
  { label: 'Lokasi Objek', kunci: 'ObjectLocation' },
  { label: 'Status', kunci: 'SurveyStatus' },
]

/**
 * Varian grid objek — Pega menggambar SATU dari beberapa, menurut lini bisnis.
 *
 * Tiga tab memakai daftar objek yang sama dengan kolom berbeda:
 *
 *	Registrasi              `ViewInputRegisterDetail`   4 varian
 *	Estimasi                `ViewInputEstimasiDetail`   3 varian
 *	Adjustment & Akseptasi  `ViewShowObjectAdj`         3 varian
 *
 * # Penjaganya DISALIN, bukan didekati
 *
 * KOREKSI (2026-10-08): sebelumnya varian dipilih dari data — kolom pembeda yang terisi —
 * dengan alasan "dua dari delapan When rule kosong di export". **Alasan itu salah.**
 *
 * Yang saya baca adalah `pyConditionString`, yaitu label tampilan yang usang: enam When
 * membawa teks yang sama persis, dan `IsFire` hanya berisi "[Double click to add condition]".
 * Kondisi sebenarnya ada di `pyParametersParamValue`, dan keenamnya jelas.
 *
 * Penjaga tiap grid pun terbaca — bukan di `pyVisible` melainkan di
 * **`pyContainerVisibleWhen`** pada wadah grid.
 *
 * Tebakan lama bahkan TERBALIK: saya mengira kolom Peserta milik PA; yang benar milik
 * **Travel**, sedangkan PA memakai Nama/Perkerjaan/Tanggal Lahir.
 */
export type VarianObjek = {
  /** Ungkapan `pyContainerVisibleWhen` apa adanya — satu nama When, atau beberapa dengan `||`. */
  penjaga: string
  kolom: KolomGrid[]
}

/** Varian grid objek tab **Registrasi** — `ViewInputRegisterDetail`. */
export const VARIAN_OBJEK_REGISTER: VarianObjek[] = [
  {
    penjaga: 'IsTravel',
    kolom: [
      { label: 'Nama Peserta', kunci: 'ObjectName' },
      { label: 'Status', kunci: 'ObjectParticipantStatus' },
      { label: 'KTP/Paspor', kunci: 'ObjectIDCard' },
      { label: 'Tanggal Lahir', kunci: 'ObjectDateOfBirth' },
    ],
  },
  {
    // "Perkerjaan" — salah ketik di layar lama, dibawa apa adanya (`D-13`).
    penjaga: 'IsPA',
    kolom: [
      { label: 'Nama', kunci: 'ObjectName' },
      { label: 'Perkerjaan', kunci: 'ObjectJob' },
      { label: 'Tanggal Lahir', kunci: 'ObjectDateOfBirth' },
    ],
  },
  {
    penjaga: 'IsAneka',
    kolom: [
      { label: 'Object', kunci: 'ObjectName' },
      { label: 'Nama Tipe', kunci: 'ObjectVehicleType' },
      { label: 'Nomor Chasis', kunci: 'ObjectVehicleChasis' },
      { label: 'Location', kunci: 'ObjectLocation' },
    ],
  },
  {
    penjaga: 'IsFire || IsMarineCargo',
    kolom: [
      { label: 'Object', kunci: 'ObjectName' },
      { label: 'Location', kunci: 'ObjectLocation' },
    ],
  },
]

/** Varian grid objek tab **Estimasi** — `ViewInputEstimasiDetail`. */
export const VARIAN_OBJEK_ESTIMASI: VarianObjek[] = [
  {
    // "Lokasi Object" — campur Indonesia/Inggris, apa adanya.
    penjaga: 'IsAneka || IsMarineCargo || IsFire',
    kolom: [
      { label: 'Nama Objek', kunci: 'ObjectName' },
      { label: 'Lokasi Object', kunci: 'ObjectLocation' },
    ],
  },
  {
    // "Tanggal lahir" — huruf kecil pada "lahir", berbeda dari grid lain. Apa adanya.
    penjaga: 'IsPA',
    kolom: [
      { label: 'Nama Objek', kunci: 'ObjectName' },
      { label: 'Pekerjaan', kunci: 'ObjectJob' },
      { label: 'Tanggal lahir', kunci: 'ObjectDateOfBirth' },
    ],
  },
  {
    penjaga: 'IsTravel',
    kolom: [
      { label: 'Nama Peserta', kunci: 'ObjectName' },
      { label: 'Status', kunci: 'ObjectParticipantStatus' },
      { label: 'KTP/Paspor', kunci: 'ObjectIDCard' },
      { label: 'Tanggal Lahir', kunci: 'ObjectDateOfBirth' },
    ],
  },
]

/** Varian grid objek tab **Adjustment & Akseptasi** — `ViewShowObjectAdj`. */
export const VARIAN_OBJEK_ADJ: VarianObjek[] = [
  {
    penjaga: 'IsAneka || IsFire || IsMarineCargo',
    kolom: [
      { label: 'Nama Objek', kunci: 'ObjectName' },
      { label: 'Lokasi', kunci: 'ObjectLocation' },
    ],
  },
  {
    penjaga: 'IsPA',
    kolom: [
      { label: 'Nama Objek', kunci: 'ObjectName' },
      { label: 'Pekerjaan', kunci: 'ObjectJob' },
      { label: 'Tanggal Lahir', kunci: 'ObjectDateOfBirth' },
    ],
  },
  {
    penjaga: 'IsTravel',
    kolom: [
      { label: 'Nama Peserta', kunci: 'ObjectName' },
      { label: 'Status', kunci: 'ObjectParticipantStatus' },
      { label: 'KTP/Paspor', kunci: 'ObjectIDCard' },
      { label: 'Tanggal Lahir', kunci: 'ObjectDateOfBirth' },
    ],
  },
]
