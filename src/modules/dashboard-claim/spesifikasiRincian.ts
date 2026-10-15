/*
  ============================================================================
  KUNCI KOLOM BERPINDAH DARI JALUR JSON KE FIELD KONTRAK — 2026-10-12
  ============================================================================

  Sampai 2026-10-12, `kunci` di berkas ini adalah JALUR di dalam dokumen
  `POOLDATA.JSON_KLAIM.DATA_JSON`. Setelah Work Owner memutuskan migrasi tidak memakai JSON
  lagi, isinya dibaca dari tabel relasional dan dikirim sebagai bagian bertipe — sehingga
  `kunci` kini adalah NAMA FIELD pada kontrak.

  `label` TIDAK berubah sama sekali: ia yang dilihat pengguna, dan `D-13` menetapkan teks
  layar mengikuti Pega apa adanya. Yang berubah hanya dari mana nilainya diambil.

  Kunci berawalan `BELUM_TERSAMBUNG_` adalah kolom yang BELUM punya padanan kolom basis
  data. Ia sengaja tidak dihapus dari daftar: menghapusnya membuat kolomnya hilang dari layar
  tanpa jejak, sedangkan membiarkannya kosong menunjukkan ada yang belum selesai. Kolom-kolom
  itu berasal dari grid HE, checklist dokumen, dan grid komite.
*/
import type { Isian, KolomGrid } from "./BagianRincian";

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
  { label: "Tanggal Terima Dokumen", kunci: "tanggal_terima_dokumen" },
  {
    label: "Tanggal Kejadian / Tanggal Masuk Rawat Inap",
    kunci: "tanggal_kejadian",
  },
  {
    label: "Tanggal Keluar Rawat Inap",
    kunci: "tanggal_keluar_rawat_inap",
  },
  { label: "Tanggal Lapor", kunci: "tanggal_lapor" },
  { label: "No Telephone Pelapor", kunci: "telepon_pelapor" },
  { label: "Email", kunci: "email_pelapor" },
  { label: "Status Pelapor", kunci: "status_pelapor" },
  { label: "Sebutkan...", kunci: "status_pelapor_lain" },
  {
    label: "Klaim Estimasi",
    kunci: "klaim_estimasi",
    absen:
      "Nilai estimasi hidup di POOLDATA.T_CLAIM_ESTIMASI — per objek dan per coverage, bukan " +
      "satu nilai per klaim. Menjumlahkannya adalah keputusan bisnis yang belum diambil.",
  },
  {
    label: "Nilai Dalam IDR",
    kunci: "nilai_idr",
    absen:
      "Sumbernya sama dengan Klaim Estimasi (kolom CONVERTVALUE), dan terhalang pertanyaan " +
      "yang sama.",
  },
  { label: "Lokasi Kerugian/Kejadian", kunci: "lokasi_kejadian" },
  { label: "Kelurahan", kunci: "kelurahan" },
  { label: "Apakah tertanggung seorang TKI?", kunci: "tertanggung_tki" },
  {
    label: "Catatan dari Analyst",
    kunci: "catatan_dari_analyst",
    absen:
      "Catatan analis tersimpan sebagai PERCAKAPAN di POOLDATA.M_KOMUNIKASI_PNC, bukan satu " +
      "nilai. Menampilkan pesan terakhirnya saja akan menyembunyikan sisanya.",
  },
  { label: "Catatan Ke Analyst", kunci: "catatan_ke_analyst" },
  { label: "Catatan Ke PIC Teknis", kunci: "catatan_ke_pic_teknis" },
];

/**
 * Grid objek pertanggungan pada bagian Register.
 *
 * Kolomnya dari ikatan sel `ASM-FW-GCNMFW-Data-Object` di section yang sama. Labelnya memakai
 * istilah `CONTEXT.md` — section Pega-nya tidak membawa `pyLabelFieldValue` untuk kolom grid,
 * dan nama propertinya sendiri (`ObjectName`, `ObjectIDCard`) sudah terbaca apa adanya.
 */
export const KOLOM_OBJEK: KolomGrid[] = [
  { label: "Nama Objek", kunci: "nama_objek" },
  { label: "Lokasi Objek", kunci: "lokasi_objek" },
  { label: "Status Peserta", kunci: "status_peserta" },
  { label: "NIK", kunci: "nik" },
  { label: "Tanggal Lahir", kunci: "tanggal_lahir" },
  { label: "Pekerjaan", kunci: "pekerjaan" },
  { label: "Tipe Kendaraan", kunci: "tipe_kendaraan" },
  { label: "No Rangka", kunci: "no_rangka" },
];

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
  { label: "Nama", kunci: "nama" },
  { label: "Alamat", kunci: "alamat" },
];

/**
 * Bagian **Hasil Survey** — `Section/ViewHasilSurvey-Section.xml`.
 *
 * Jalurnya bersarang di `ClaimData.SurveyData`, bukan di akar `ClaimData` — dibaca apa adanya
 * dari ikatan selnya.
 */
export const ISIAN_SURVEY: Isian[] = [
  {
    label: "Survey Atas Permintaan",
    kunci: "atas_permintaan",
  },
  { label: "Lokasi Survey", kunci: "lokasi" },
  { label: "No. Telp", kunci: "telepon" },
  {
    label: "Tanggal Request Survey",
    kunci: "tanggal_request",
  },
  { label: "Cabang", kunci: "cabang" },
  { label: "Surveyor", kunci: "surveyor" },
  { label: "Email Surveyor", kunci: "email_surveyor" },
  { label: "Nama Object", kunci: "nama_objek" },
];

/**
 * Bagian **Catatan ke Analyst** — `Section/CatatanToAnalyst_Section-Section.xml`.
 *
 * Label "Note PilihanCompliance" dibawa APA ADANYA, termasuk kata tersambungnya — itu yang
 * tertulis di `pyLabelFieldValue`, dan `D-13` menetapkan teks layar mengikuti Pega.
 */
export const ISIAN_CATATAN: Isian[] = [
  {
    label: "Pilihan Compliance",
    kunci: "pilihan_compliance",
    absen:
      "Dicari menyeluruh ke ALL_TAB_COLUMNS seluruh skema POOLDATA — tidak ada satu pun kolom " +
      "yang sepadan. Bukan hanya tidak ada di T_CLAIM_PNC.",
  },
  {
    label: "Note PilihanCompliance",
    kunci: "note_pilihan_compliance",
    absen:
      "Sama seperti Pilihan Compliance: tidak ada kolomnya di mana pun pada skema POOLDATA.",
  },
  {
    label: "Tanggal Kelengkapan Dokumen Akhir PUCL",
    kunci: "tanggal_dokumen_lengkap",
  },
  {
    label: "Alasan Dokter Tolak RCL",
    kunci: "alasan_dokter_tolak_rcl",
  },
];

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
  { label: "Nama Coverage", kunci: "nama_coverage" },
  { label: "Nama Plan", kunci: "kode_plan" },
  { label: "TSI", kunci: "tsi" },
  { label: "Mata Uang", kunci: "mata_uang" },
  { label: "Penyebab Kerugian", kunci: "penyebab_kerugian" },
];

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
  {
    label: "Kategori",
    kunci: "detail_document",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Wajib Unggah",
    kunci: "sts_wajib",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Minimal Unggah",
    kunci: "min_doc",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Total Sudah Diunggah",
    kunci: "risk",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
];

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
  { label: "Category", kunci: "kategori" },
  { label: "Name", kunci: "nama" },
  { label: "Created date", kunci: "ditambahkan_pada" },
];

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
  { label: "Case ID", kunci: "kode_tahap" },
  { label: "POSISI", kunci: "nama_tahap" },
  { label: "Tanggal", kunci: "tanggal" },
  { label: "STATUS", kunci: "status" },
];

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
  { label: "Object", kunci: "nama_objek" },
  {
    label: "Model",
    kunci: "objectvehiclemodel",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Merk",
    kunci: "objectvehiclebrand",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "Nama Tipe", kunci: "tipe_kendaraan" },
  { label: "Nomor Chasis", kunci: "no_rangka" },
  { label: "Location", kunci: "lokasi_objek" },
];

/**
 * Isian kepala popup — label yang dipakai BERSAMA oleh beberapa section.
 *
 * "Aging Amount", "Status Klaim", dan "Catatan dari Inputor" muncul sebagai
 * `pyLabelFieldValue` di `ViewInputRegisterDetail`, `ViewInputEstimasiDetail`, dan
 * `ViewShowReceiver` sekaligus — tanda ia milik kerangka popup, bukan satu bagian tertentu.
 */
export const ISIAN_KEPALA: Isian[] = [
  { label: "Status Klaim", kunci: "status_klaim" },
  {
    label: "Aging Amount",
    kunci: "aging_amount",
    absen:
      "Milik data polis (Policy.AgingAmount), dan polis bukan milik bounded context ini (D-04).",
  },
  { label: "Catatan dari Inputor", kunci: "catatan_inputor" },
];

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
  {
    label: "SurveyName",
    kunci: "surveyname",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "SurveyDate", kunci: "tanggal_survey" },
  { label: "SurveyorName", kunci: "nama_surveyor" },
  { label: "LocationSurvey", kunci: "lokasi_survey" },
  {
    label: "InsuredPIC",
    kunci: "insuredpic",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "LossType", kunci: "jenis_kerugian" },
  {
    label: "LossDescription",
    kunci: "lossdescription",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "EstimateDescription",
    kunci: "estimatedescription",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "SurveyorDescription",
    kunci: "surveyordescription",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "PolicyTerm",
    kunci: "policyterm",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "PoliceReport",
    kunci: "policereport",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "WitnessName",
    kunci: "witnessname",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "WitnessAddress",
    kunci: "witnessaddress",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Salvage",
    kunci: "salvage",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "SalvageDescription",
    kunci: "salvagedescription",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "ClientSignature",
    kunci: "clientsignature",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "OthersDescription",
    kunci: "othersdescription",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
];

/**
 * Sub-popup **Hasil Survey** — `ShowSurveyResults`, dibuka dari baris grid Hasil Survey.
 *
 * Dua puluh caption terbaca di section itu; yang di bawah adalah yang berpasangan dengan
 * properti. Yang tidak dipakai ("Button", "Dokumen", "Remark", "Komentar") melekat pada
 * kendali yang tidak membawa nilai.
 */
export const ISIAN_HASIL_SURVEY: Isian[] = [
  { label: "Reference Number", kunci: "no_referensi" },
  {
    label: "Appointed No",
    kunci: "idsurvey",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Appointment Date",
    kunci: "appointmentdate",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Request Date Appointment",
    kunci: "appointmentdatereq",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "Tanggal Survey", kunci: "tanggal_survey" },
  { label: "Lokasi Survey", kunci: "lokasi_survey" },
  { label: "Nama Surveyor", kunci: "nama_surveyor" },
  { label: "Status Survey", kunci: "status" },
  { label: "Catatan Survey", kunci: "catatan" },
  {
    label: "Komite ID",
    kunci: "komiteid",
    absen:
      "Milik T_CLAIM_KOMITE_LIST — belum dibaca modul ini (±40 juta baris, NO_KLAIM tanpa index).",
  },
  {
    label: "Status Komite",
    kunci: "komiteaproval",
    absen: "Milik T_CLAIM_KOMITE_LIST — belum dibaca modul ini.",
  },
  {
    label: "Komentar",
    kunci: "komitecomment",
    absen: "Milik T_CLAIM_KOMITE_LIST — belum dibaca modul ini.",
  },
  {
    label: "Note Terlambat",
    kunci: "keteranganlain",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
];

/**
 * Sub-popup **Detail Progress** — `ShowDetailProgress_sect`, dibuka dari baris grid Posisi.
 *
 * Caption "---" dan "YA" tidak dipakai: keduanya isi dropdown, bukan label isian.
 */
export const ISIAN_DETAIL_PROGRESS: Isian[] = [
  { label: "Tgl Progress", kunci: "tanggal" },
  { label: "Keterangan", kunci: "keterangan" },
  { label: "PIC", kunci: "oleh" },
  {
    label: "Next Follow Up",
    kunci: "nextfollowup",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "Tgl Next Follow Up", kunci: "tindak_lanjut" },
  {
    label: "Catatan Delivery",
    kunci: "comment",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
];

/**
 * Sub-popup **Coverage HE** — `CoverageGridClaimHE_Komite`.
 *
 * Kesepuluh judul disalin dari rujukan `pyCaption`-nya.
 */
export const KOLOM_COVERAGE_HE: KolomGrid[] = [
  { label: "Coverage", kunci: "nama_coverage" },
  { label: "Penyebab Kerugian", kunci: "penyebab_kerugian" },
  {
    label: "Merk",
    kunci: "objectvehiclebrand",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Model",
    kunci: "objectvehiclemodel",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "Type", kunci: "tipe_kendaraan" },
  {
    label: "Tahun Kendaraan",
    kunci: "objectvehicleyear",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "No Rangka", kunci: "no_rangka" },
  {
    label: "No Mesin",
    kunci: "objectvehicleenginenumber",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "No Serial",
    kunci: "serialno",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
];

/**
 * Sub-popup **Detail Coverage** — `ViewObjectCoverageAdj`, dibuka Flow Action
 * `ViewCoverageAdj` dari baris grid Coverage.
 *
 * Kelima judul sama dengan judul kolom gridnya — section itu memang menggambar isian yang
 * sama secara vertikal. Yang bertambah hanya `DESCRIPTION`, yang di grid tidak digambar.
 */
export const ISIAN_COVERAGE_DETAIL: Isian[] = [
  { label: "Nama Coverage", kunci: "nama_coverage" },
  { label: "Nama Plan", kunci: "kode_plan" },
  { label: "TSI", kunci: "tsi" },
  { label: "Mata Uang", kunci: "mata_uang" },
  { label: "Penyebab Kerugian", kunci: "penyebab_kerugian" },
];

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
  { label: "Tanggal Pengajuan Survey", kunci: "tanggal_survey" },
  { label: "Nama Surveyor", kunci: "nama_surveyor" },
  { label: "Nama Objek", kunci: "nama_objek" },
  { label: "Lokasi Objek", kunci: "lokasi_objek" },
  { label: "Status", kunci: "status" },
];

/**
 * Grid **Detail Surveyor** — barisnya membuka sub-popup `DetailHasilSurveyor_Sect`.
 *
 * Section itu nol caption, sehingga kolom DAN isian sub-popup-nya memakai nama properti apa
 * adanya. Lihat catatan pada `ISIAN_DETAIL_SURVEYOR`.
 */
export const KOLOM_DETAIL_SURVEYOR: KolomGrid[] = [
  {
    label: "SurveyName",
    kunci: "surveyname",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  { label: "SurveyDate", kunci: "tanggal_survey" },
  { label: "SurveyorName", kunci: "nama_surveyor" },
  { label: "LocationSurvey", kunci: "lokasi_survey" },
];

/**
 * Sub-popup **Coverage HE** — `CoverageGridClaimHE_Komite`, dibuka Flow Action
 * `KomiteCoverageGrid_ClaimHE` dari baris grid Objek HE.
 *
 * Kesembilan judul disalin dari rujukan `pyCaption` section itu.
 */
export const ISIAN_COVERAGE_HE: Isian[] = [
  { label: "Coverage", kunci: "nama_coverage" },
  { label: "Penyebab Kerugian", kunci: "penyebab_kerugian" },
  {
    label: "Merk",
    kunci: "objectvehiclebrand",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Model",
    kunci: "objectvehiclemodel",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Type",
    kunci: "objectvehicletype",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "Tahun Kendaraan",
    kunci: "objectvehicleyear",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "No Rangka",
    kunci: "objectvehiclechasis",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "No Mesin",
    kunci: "objectvehicleenginenumber",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
  {
    label: "No Serial",
    kunci: "serialno",
    absen: 'Kolom padanannya BELUM ditelusuri — berbeda dari "tidak ada".',
  },
];

/**
 * Sub-popup **Penerima Klaim** — `ViewInputReceiver`, dibuka Flow Action bernama sama dari
 * baris grid Penerima.
 *
 * Isinya LEBIH LENGKAP daripada gridnya: tiga caption yang di grid tidak muat — "Nama
 * Rekening" di antaranya — digambar di sini. Itu bentuk Pega: grid menunjukkan ringkasan,
 * `pyEditAction` membuka seluruhnya.
 */
export const ISIAN_PENERIMA_DETAIL: Isian[] = [
  { label: "Nama", kunci: "nama" },
  { label: "Alamat", kunci: "alamat" },
  {
    label: "Telepon",
    kunci: "telephone",
    absen: "T_CLAIM_RECEIVER hanya punya 6 kolom; telepon bukan salah satunya.",
  },
  {
    label: "Nomor Kontrak",
    kunci: "nomorkontrak",
    absen:
      "T_CLAIM_RECEIVER hanya punya 6 kolom; nomor kontrak bukan salah satunya.",
  },
  { label: "No Rekening", kunci: "no_rekening" },
  { label: "Nama Bank", kunci: "nama_bank" },
  {
    label: "Nama Cabang Bank",
    kunci: "branchofbank",
    absen: "T_CLAIM_RECEIVER menyimpan NAMEOFBANK, tanpa cabangnya.",
  },
];

/**
 * Sub-popup **Coverage (grid)** — `ViewCoverageGridContent`, dibuka Flow Action
 * `ViewCoverageGrid_fa`.
 *
 * Captionnya sama dengan `ViewObjectCoverageAdj` ditambah **Coverage** — section berbeda,
 * isian hampir sama. Dibawa apa adanya, bukan disatukan: keduanya dipanggil dari tempat yang
 * berbeda di Pega, dan menyatukannya menghapus perbedaan yang mungkin bermakna.
 */
export const ISIAN_COVERAGE_GRID: Isian[] = [
  { label: "Coverage", kunci: "nama_coverage" },
  { label: "Nama Plan", kunci: "kode_plan" },
  { label: "TSI", kunci: "tsi" },
  { label: "Mata Uang", kunci: "mata_uang" },
  { label: "Penyebab Kerugian", kunci: "penyebab_kerugian" },
];

/**
 * Grid **komite** pada tab Heavy Equipment — grid kedua
 * `Section/ShowObjectHEPICTeknis_komite-Section.xml`.
 *
 * Dipisahkan dari KOLOM_OBJEK_HE pada 2026-10-08. Menggabungkannya membuat satu baris objek
 * seolah membawa satu keputusan komite — padahal keduanya daftar yang berbeda panjang.
 */
export const KOLOM_KOMITE_HE: KolomGrid[] = [
  {
    label: "Nama Komite",
    kunci: "komiteid",
    absen:
      "Milik T_CLAIM_KOMITE_LIST — belum dibaca modul ini (±40 juta baris, NO_KLAIM tanpa index).",
  },
  {
    label: "Status",
    kunci: "komiteaproval",
    absen: "Milik T_CLAIM_KOMITE_LIST — belum dibaca modul ini.",
  },
  {
    label: "Komentar",
    kunci: "komitecomment",
    absen: "Milik T_CLAIM_KOMITE_LIST — belum dibaca modul ini.",
  },
];

/** Grid kedua tab **Hasil Survey** — varian investigasi di section yang sama. */
export const KOLOM_INVESTIGASI: KolomGrid[] = [
  { label: "Tanggal Investigasi", kunci: "tanggal_survey" },
  { label: "Nama Peserta", kunci: "nama_objek" },
  { label: "Lokasi Objek", kunci: "lokasi_objek" },
  { label: "Status", kunci: "status" },
];

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
  penjaga: string;
  kolom: KolomGrid[];
};

/** Varian grid objek tab **Registrasi** — `ViewInputRegisterDetail`. */
export const VARIAN_OBJEK_REGISTER: VarianObjek[] = [
  {
    penjaga: "IsTravel",
    kolom: [
      { label: "Nama Peserta", kunci: "nama_objek" },
      { label: "Status", kunci: "status_peserta" },
      { label: "KTP/Paspor", kunci: "nik" },
      { label: "Tanggal Lahir", kunci: "tanggal_lahir" },
    ],
  },
  {
    // "Perkerjaan" — salah ketik di layar lama, dibawa apa adanya (`D-13`).
    penjaga: "IsPA",
    kolom: [
      { label: "Nama", kunci: "nama_objek" },
      { label: "Perkerjaan", kunci: "pekerjaan" },
      { label: "Tanggal Lahir", kunci: "tanggal_lahir" },
    ],
  },
  {
    penjaga: "IsAneka",
    kolom: [
      { label: "Object", kunci: "nama_objek" },
      { label: "Nama Tipe", kunci: "tipe_kendaraan" },
      { label: "Nomor Chasis", kunci: "no_rangka" },
      { label: "Location", kunci: "lokasi_objek" },
    ],
  },
  {
    penjaga: "IsFire || IsMarineCargo",
    kolom: [
      { label: "Object", kunci: "nama_objek" },
      { label: "Location", kunci: "lokasi_objek" },
    ],
  },
];

/** Varian grid objek tab **Estimasi** — `ViewInputEstimasiDetail`. */
export const VARIAN_OBJEK_ESTIMASI: VarianObjek[] = [
  {
    // "Lokasi Object" — campur Indonesia/Inggris, apa adanya.
    penjaga: "IsAneka || IsMarineCargo || IsFire",
    kolom: [
      { label: "Nama Objek", kunci: "nama_objek" },
      { label: "Lokasi Object", kunci: "lokasi_objek" },
    ],
  },
  {
    // "Tanggal lahir" — huruf kecil pada "lahir", berbeda dari grid lain. Apa adanya.
    penjaga: "IsPA",
    kolom: [
      { label: "Nama Objek", kunci: "nama_objek" },
      { label: "Pekerjaan", kunci: "pekerjaan" },
      { label: "Tanggal lahir", kunci: "tanggal_lahir" },
    ],
  },
  {
    penjaga: "IsTravel",
    kolom: [
      { label: "Nama Peserta", kunci: "nama_objek" },
      { label: "Status", kunci: "status_peserta" },
      { label: "KTP/Paspor", kunci: "nik" },
      { label: "Tanggal Lahir", kunci: "tanggal_lahir" },
    ],
  },
];

/** Varian grid objek tab **Adjustment & Akseptasi** — `ViewShowObjectAdj`. */
export const VARIAN_OBJEK_ADJ: VarianObjek[] = [
  {
    penjaga: "IsAneka || IsFire || IsMarineCargo",
    kolom: [
      { label: "Nama Objek", kunci: "nama_objek" },
      { label: "Lokasi", kunci: "lokasi_objek" },
    ],
  },
  {
    penjaga: "IsPA",
    kolom: [
      { label: "Nama Objek", kunci: "nama_objek" },
      { label: "Pekerjaan", kunci: "pekerjaan" },
      { label: "Tanggal Lahir", kunci: "tanggal_lahir" },
    ],
  },
  {
    penjaga: "IsTravel",
    kolom: [
      { label: "Nama Peserta", kunci: "nama_objek" },
      { label: "Status", kunci: "status_peserta" },
      { label: "KTP/Paspor", kunci: "nik" },
      { label: "Tanggal Lahir", kunci: "tanggal_lahir" },
    ],
  },
];
