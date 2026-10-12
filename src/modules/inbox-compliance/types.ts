/**
 * Bentuk data layar Inbox Compliance — menu `MENU_ID 47`, pengganti harness
 * `inboxCompliance_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega.
 */

/** Satu baris pekerjaan. */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol buka detail klaim.
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom.
   */
  referensi: string;

  nomor_case: string;

  /**
   * Kolom "No Klaim" pada tab Post Audit.
   *
   * Namanya menyesatkan dan itu bentuk sistem lama: isinya BUKAN nomor klaim melainkan
   * kunci teknis Pega, `ASM-FW-GCNMFW-WORK PNC-2114`. Layar Pega menampilkannya apa
   * adanya, dan `D-13` menetapkan tampilan ditiru.
   */
  no_klaim: string;

  no_polis: string;
  nama_tertanggung: string;
  nama_bisnis: string;
  nama_cabang: string;
  nama_admin: string;

  /** Tanggal saja: `YYYY-MM-DD`. */
  tanggal_kirim_compliance: string | null;

  /**
   * Tanggal DAN jam: `YYYY-MM-DD HH:MM`.
   *
   * Berbeda dari kolom tanggal lain, dan itu disengaja — layar Pega menampilkannya sebagai
   * `22/04/25 13:46`.
   */
  tanggal_kirim_post_audit: string | null;

  catatan_compliance: string;

  /**
   * Teks kolom Aging, mengikuti bentuk sistem lama — "5 hours ago",
   * "2 days 3 hours ago". Kosong bila tidak dapat dihitung.
   */
  aging: string;

  /**
   * Angka mentah Aging dalam jam, sudah dipotong akhir pekan.
   *
   * `null` berarti tidak dapat dihitung — dan itu BERBEDA dari `0`, yang berarti baris
   * baru saja masuk antrean. Layar memakai angka ini untuk menandai baris yang terlalu
   * lama menunggu, tanpa harus mengurai teksnya kembali menjadi angka.
   */
  aging_jam: number | null;

  /**
   * Kolom "OutStanding" pada tab Post Audit — `1 year 5 months ago`.
   *
   * BUKAN kolom Aging dengan nama lain: dasarnya waktu kalender apa adanya, sedangkan
   * Aging memotong akhir pekan.
   */
  outstanding: string;
};

/** Nama isian pada satu baris — dipakai memilih sel yang digambar sebuah kolom. */
export type WorkItemField = keyof WorkItem;

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type TabColumn = {
  kunci: WorkItemField;
  judul: string;
};

/** Satu tab beserta bentuk gridnya. */
export type Tab = {
  kode: string;
  nama: string;
  keterangan: string;
  kolom: TabColumn[];

  /**
   * Tab ini sudah dapat menampilkan data.
   *
   * Tab yang belum dapat dilayani TETAP dikirim beserta penghalangnya — bukan
   * disembunyikan. Menyembunyikannya membuat pengguna yang mencarinya menduga modulnya
   * belum selesai.
   */
  tersedia: boolean;

  /** Apa yang kurang dan siapa pemiliknya. Kosong bila tersedia. */
  penghalang?: string;
};

/** Jawaban `GET /api/inbox-compliance/tab`. */
export type MetadataResponse = {
  tab: Tab[];
  tab_bawaan: string;
  portal: string;

  /**
   * Hal yang belum berjalan penuh beserta alasannya, siap ditampilkan apa adanya.
   *
   * Ia datang dari server supaya hilang dengan sendirinya begitu penghalangnya hilang —
   * tanpa menyunting layar.
   */
  keterbatasan: string[];
};

/** Keterangan halaman. */
export type PageInfo = {
  halaman: number;
  ukuran: number;
  total: number;
  total_halaman: number;
};

/** Jawaban `GET /api/inbox-compliance`. */
export type ListResponse = {
  tab: Tab;
  baris: WorkItem[];
  paginasi: PageInfo;
  portal: string;
};

/** Kode galat modul ini, sebagaimana dikirim backend. */
export const InboxComplianceError = {
  validationFail: "validasi_gagal",
  tabNotReady: "tab_belum_tersedia",
} as const;

/** Badan permintaan `POST /api/inbox-compliance/post-audit`. */
export type SendPostAuditRequest = {
  /** Kunci klaim yang dikirim — isian `referensi` pada baris tab Compliance. */
  referensi: string;

  /** Catatan. Boleh kosong: kolomnya nullable dan tidak ada bukti bahwa ia wajib. */
  catatan: string;
};

/** Jawaban pengiriman yang berhasil. */
export type SendPostAuditResponse = {
  nomor_case: string;
  no_klaim: string;
  nama_tertanggung: string;
  no_polis: string;
  catatan: string;
  tanggal_kirim_post_audit: string | null;
  portal: string;
};

// ── Form Compliance Checker ──────────────────────────────────────────────────

/**
 * Satu Pilihan Compliance beserta labelnya.
 *
 * Keempatnya datang dari server, bukan ditulis di sini: nilainya adalah hasil pembacaan
 * `Property/PilihanCompliance_property.xml`, dan tempat pembacaan itu tercatat adalah
 * backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang
 * satu akan tertinggal saat yang lain diperbaiki.
 */
/** Satu baris grid komentar yang sudah tersimpan. */
export type Comment = {
  /** Nomor baris yang dilihat petugas — grid Pega bernomor. */
  urutan: number;
  tanggal: string | null;
  komentar: string;
};

/** Satu baris grid komentar yang dikirim layar. */
export type CommentRequest = {
  /** Kosong berarti pakai nilai bawaan Pega: waktu saat keputusan disimpan. */
  tanggal: string;
  komentar: string;
};

export type Choice = {
  /** `"0"` Fraud/Tolak · `"1"` Bayar/Valid · `"2"` Bayar/PostAudit · `"3"` Lain-Lain. */
  nilai: string;
  label: string;
};

/** Keputusan Compliance yang sudah tersimpan atas satu klaim. */
export type Decision = {
  pilihan: string;
  pilihan_label: string;
  note: string;

  /** Grid komentar — tempat petugas Compliance menulis. */
  komentar: Comment[];

  /**
   * Catatan **Investigator** — `.ClaimData.ComplianceRemark`.
   *
   * READ-ONLY pada form Pega (`pyEditOptions=Read-only`). Jangan tertukar dengan
   * `komentar` di atas, yang justru isian petugas.
   */
  catatan_investigator: string;

  diputuskan_oleh: string;
  diputuskan_pada: string | null;

  /** Terisi HANYA pada Bayar/Valid. */
  tanggal_valid: string | null;

  /** Terisi HANYA pada Bayar/PostAudit. */
  tanggal_kirim_post_audit: string | null;
};

/**
 * Tombol mana yang boleh digambar pada form.
 *
 * Nilainya datang dari SERVER, tidak disimpulkan di sini dari lini bisnis. Syaratnya
 * dibaca dari rule Pega (`IsTravel`, `IsPA`) dan tempat pembacaan itu tercatat adalah
 * backend; menyimpulkannya di layar berarti aturan yang sama hidup di dua tempat.
 */
export type Actions = {
  unggah_dokumen: boolean;
  unduh_dokumen_reject: boolean;
  simpan: boolean;

  /** Tampil pada lini **PA**. */
  kirim_ke_analyst: boolean;

  /**
   * Tampil pada lini **Travel**.
   *
   * Tombolnya digambar tetapi belum berfungsi: Data Transform `SendToPIC` yang
   * dipanggilnya tidak ada di export (`R-16`).
   */
  kirim_ke_pic_teknik: boolean;
};

/** Form Compliance Checker yang terbuka. */
export type CheckerResponse = {
  klaim: WorkItem;
  pilihan: Choice[];

  /** `null` berarti klaimnya belum pernah diputuskan. */
  keputusan: Decision | null;

  /** Tombol yang boleh digambar untuk klaim ini. */
  tombol: Actions;

  /**
   * Kalimat yang WAJIB ditampilkan, bukan disembunyikan.
   *
   * Form ini mencatat keputusan; ia belum menjalankan alurnya. Klaimnya di Pega tidak
   * berpindah status dan tidak keluar dari antrean, karena tabel klaim masih dimiliki
   * Pega selama masa paralel (`P-1`).
   */
  keterbatasan: string;

  // Medan `dokumen` dan `dokumen_gagal_dibaca` DICABUT 2026-10-08 bersama gridnya —
  // Pega tidak pernah memuat lampiran dari basis data saat form dibuka.

  /**
   * Blok **"Hasil Investigasi"** digambar, yakni `IsPA`.
   *
   * Terpisah dari panjang `hasil_investigasi`: klaim PA yang belum pernah disurvei
   * tetap menggambar bloknya dengan tabel kosong, persis seperti grid Pega.
   */
  tampilkan_hasil_investigasi: boolean;

  /** Isi blok "Hasil Investigasi" — kosong pada lini selain PA. */
  hasil_investigasi: SurveyResult[];

  /**
   * `true` ketika hasil investigasi GAGAL dibaca, sementara form tetap terbuka.
   *
   * Dibedakan dari senarai kosong: keduanya menggambar tabel tanpa baris, tetapi yang
   * satu berarti "belum pernah disurvei" dan yang lain "tidak terbaca".
   */
  hasil_investigasi_gagal_dibaca: boolean;

  /**
   * Isian form Surat Penolakan yang terbawa dari klaim.
   *
   * Dikirim bersama form, bukan lewat permintaan tersendiri saat dialognya dibuka —
   * sehingga dialognya terbuka tanpa menunggu.
   */
  pra_isi_surat_penolakan: RejectPrefill;

  /**
   * Tab "Dokumen" DIGAMBAR, yakni `IsTravel`.
   *
   * Terpisah dari panjang `daftar_dokumen`: lini Travel yang masternya belum diisi
   * tetap menggambar tabnya dengan tabel kosong.
   */
  tampilkan_daftar_dokumen: boolean;

  /** Isi tab "Dokumen" — kosong pada lini selain Travel. */
  daftar_dokumen: DocumentChecklistRow[];

  /** `true` ketika daftar periksa GAGAL dibaca, sementara form tetap terbuka. */
  daftar_dokumen_gagal_dibaca: boolean;

  portal: string;
};

/**
 * Satu baris grid dokumen.
 *
 * Ketiga kolom yang digambar adalah sel 55–57 `Section/CompliancePNC-Section.xml`; kolom
 * keempat di Pega adalah ikon tanpa judul.
 */
export type ClaimDocument = {
  id: string;
  nama_file: string;
  tipe_file: string;
  kategori: string;

  /** Kunci berkas di penyimpanan luar — dipakai tombol lihat, bukan digambar. */
  id_penyimpanan: string;

  tanggal_unggah: string | null;
};

/**
 * Satu baris blok "Hasil Investigasi".
 *
 * Keempat isiannya persis keempat kolom grid `isPA_PNC` pada
 * `Section/ViewHasilSurvey-Section.xml` — tidak lebih.
 */
export type SurveyResult = {
  /** `null` berarti kolom tanggalnya kosong. */
  tanggal_investigasi: string | null;

  /** Pada lini PA objek pertanggungannya ORANG, sehingga nama objek = nama peserta. */
  nama_peserta: string;

  lokasi_objek: string;
  status: string;
};

/** Badan permintaan penyimpanan keputusan. */
export type SubmitDecisionRequest = {
  pilihan: string;

  /** Hanya tampil di layar saat `pilihan` bernilai `"3"`. */
  note: string;

  komentar: CommentRequest[];

  /**
   * Tombol mana yang ditekan.
   *
   * Di Pega ketiganya mengerjakan hal yang berbeda: "Simpan Data" TIDAK memanggil
   * `SetComplianceResult`, sehingga ia tidak memindahkan klaim keluar dari antrean.
   * Kedua tombol "Kirim" memanggilnya — dan tujuannya ditentukan lini bisnis, bukan
   * tombolnya, sehingga keduanya mengirim nilai yang sama.
   */
  aksi: "simpan" | "kirim";
};

/** Hasil penyimpanan keputusan. */
export type SubmitDecisionResponse = {
  keputusan: Decision;

  /** Terisi HANYA pada Bayar/PostAudit, ketika baris Post Audit ikut terbit. */
  post_audit: SendPostAuditResponse | null;

  keterbatasan: string;
  portal: string;
};

/** Tautan siap buka untuk satu dokumen. */
export type OpenDocumentResponse = {
  nama_file: string;

  /**
   * Tautan yang DIBUKA pengguna — tautan bertanda tangan yang sudah dibungkus penampil
   * Office Online, persis seperti `GetLinkViewDoc_Act` langkah 22.
   */
  tautan: string;

  portal: string;
};

/**
 * Isian form Surat Penolakan yang terbawa dari klaim.
 *
 * Hanya TIGA dari empat isian `AutoFillFormReject_Pre` — yang keempat, Tanggal Keluar
 * Rawat Inap, tidak punya kolom basis data sehingga selalu diketik petugas.
 */
export type RejectPrefill = {
  nama_pasien: string;
  tempat_kejadian: string;

  /** `null` berarti kolomnya kosong. */
  tanggal_kejadian: string | null;
};

/**
 * Badan permintaan penerbitan Surat Penolakan.
 *
 * Seluruh tanggalnya TEKS, bukan tanggal. Satu-satunya tujuan nilai ini adalah digambar
 * ke surat, dan isian "Tanggal Keluar Rawat Inap" diketik bebas karena tidak punya sumber
 * basis data.
 */
export type RejectLetterRequest = {
  up: string;
  jabatan: string;
  nama_pasien: string;
  tempat_kejadian: string;
  tanggal_kejadian: string;
  tanggal_keluar_rawat_inap: string;
  nilai_klaim_dibayarkan: string;
  tanggal_pembayaran: string;

  /** Grid "Alasan" apa adanya, termasuk baris kosongnya. */
  alasan: string[];
};

/** Jawaban penerbitan Surat Penolakan. */
export type RejectLetterResponse = {
  /** Baris dokumen yang baru tercatat — bentuknya sama dengan baris grid Dokumen. */
  dokumen: ClaimDocument;

  /**
   * `true` berarti surat sebelumnya DIGANTI, bukan ditambahkan.
   *
   * Dipakai layar untuk mengatakan "surat diperbarui" alih-alih "surat dibuat". Tanpa
   * ini, perilaku ganti-bukan-tambah tidak terlihat petugas sama sekali.
   */
  mengganti_surat_sebelumnya: boolean;

  portal: string;
};

/**
 * Satu baris tab **Dokumen** — daftar periksa kelengkapan dokumen.
 *
 * Barisnya **kategori yang seharusnya ada**, bukan berkas yang sudah diunggah. Kategori
 * tanpa berkas tetap muncul; justru itu gunanya.
 */
export type DocumentChecklistRow = {
  /** `DOC_TYPE_DT_ID` — tidak digambar; dipakai tombol Unggah untuk menempelkan berkas. */
  id_kategori: string;

  kategori: string;

  /**
   * `"Ya"`, `"Tidak"`, atau KOSONG — tiga nilai, bukan dua.
   *
   * Kosong memang terjadi di Pega ketika `STS_WAJIB` tidak terisi, dan di layar Travel
   * yang diperlihatkan Work Owner seluruh barisnya kosong. Boolean akan memaksa kosong
   * menjadi "Tidak", dan itu mengubah arti.
   */
  wajib_unggah: string;

  /** `null` berarti tidak ditentukan — berbeda dari `0`. */
  minimal_unggah: number | null;

  total_sudah_diunggah: number;
};

/** Jawaban daftar lampiran klaim pada satu kategori. */
export type ListDocumentsResponse = {
  dokumen: ClaimDocument[];
  portal: string;
};

/** Badan permintaan pemindahan kategori lampiran. */
export type ChangeDocumentCategoryRequest = {
  /** `DOC_TYPE_DT_ID` kategori TUJUAN. */
  kategori: string;
};
