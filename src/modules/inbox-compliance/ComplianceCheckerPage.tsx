import { useRef, useState, type FormEvent, type ReactNode } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { APIError } from "@/api/client";
import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";
import { TextAreaField } from "@/components/TextAreaField";
import { formatDate } from "@/components/format";

import { RejectLetterDialog } from "./RejectLetterDialog";

import {
  useComplianceChecker,
  useChangeComplianceDocumentCategory,
  useComplianceDocumentsInCategory,
  useOpenComplianceDocument,
  useGenerateRejectLetter,
  useUploadComplianceDocument,
  useSubmitComplianceDecision,
} from "./api";
import type {
  CheckerResponse,
  Choice,
  SubmitDecisionRequest,
  DocumentChecklistRow,
  SurveyResult,
  WorkItem,
} from "./types";

/**
 * Form Compliance Checker — layar yang terbuka ketika petugas menekan Nomor Case pada tab
 * Compliance.
 *
 * # Jalur Pega yang ditiru
 *
 *   1. `Section/InputComplianceDtl_Section-Section.xml` — sel Nomor Case menjalankan
 *      `SetAssignmentInboxPUCL_act(inskey=.pzInsKey)`, yang membuka ASSIGNMENT.
 *   2. `Flow/Register_Flow.xml` — Assignment9 "Compliance", `pyWorkBasket=CompliancePNC`,
 *      satu transisi: flow action `ComplianceChecker` → `End1`.
 *   3. `Flow Action/ComplianceChecker-FA.xml` — `pySectionReference = ComplianceChecker`,
 *      `pyPreDataTransform = SendToPIC`.
 *   4. Tombol "Simpan Data" menjalankan `Activity/SetComplianceResult`.
 *
 * # Empat blok, BUKAN tab
 *
 * `Section/ComplianceChecker-Section.xml` memuat empat layout bertipe `SUBHEADER` —
 * blok berjudul yang bertumpuk di satu halaman, bukan tab yang dipilih bergantian:
 *
 *     S1  "Compliance"         selalu     section `CompliancePNC`
 *     S2  "Hasil Investigasi"  IsPA       section `ViewHasilSurvey`
 *     S3  "Dokumen"            IsTravel   section `UploadDocument`
 *     S4  (tanpa judul)        IsTravel   bilah tombol
 *
 * # Dua bilah tombol yang saling melengkapi
 *
 * Kelima tombol yang sama muncul di dua tempat, dan syarat WADAH-nya berpasangan:
 *
 *     Section/CompliancePNC-Section.xml      S14, sel 76–80, wadah `!IsTravel`
 *     Section/ComplianceChecker-Section.xml  S4,  sel 15–19, wadah `IsTravel`
 *
 * Jadi setiap klaim selalu mendapat tepat satu bilah. Syarat per tombolnya identik di
 * keduanya:
 *
 *     Unggah Dokumen            pyVisible=ALWAYS
 *     Download Dokumen Reject   pyVisible=ALWAYS
 *     Simpan Data               pyVisible=ALWAYS
 *     Kirim ke Analyst          pyCondition=IsPA
 *     Kirim ke PIC Teknik       pyCondition=IsTravel
 *
 * Versi sebelumnya membaca S14 saja dan menyimpulkan ketiga tombol umum ber-`!IsTravel`,
 * sehingga klaim Travel digambar tanpa tombol apa pun. Itu **salah** — yang `!IsTravel`
 * adalah wadahnya, bukan tombolnya.
 *
 * Tombol mana yang tampil ditentukan SERVER lewat `data.tombol`, bukan disimpulkan di
 * sini dari lini bisnis (`D-59`).
 *
 * # Apa yang BELUM ada di layar ini
 *
 * Blok "Hasil Investigasi" dan blok "Dokumen". Keduanya menyentuh modul survei dan
 * lampiran yang belum ada. Tiga tombol pun belum berfungsi: rantai cetaknya
 * (`DownloadPDFReject`, `PrintRejectCompliancePDF`) menulis ke lampiran klaim, dan
 * "Kirim ke PIC Teknik" memanggil Data Transform `SendToPIC` yang hilang dari export.
 */
export function ComplianceCheckerPage() {
  const { referensi = "" } = useParams<{ referensi: string }>();
  const navigate = useNavigate();

  const checker = useComplianceChecker(referensi);
  const submit = useSubmitComplianceDecision(referensi);

  return (
    <main className="mx-auto w-full max-w-5xl px-4 py-6">
      <nav className="mb-4">
        <Button tone="halus" onClick={() => navigate("/inbox-compliance")}>
          ← Kembali ke Inbox Compliance
        </Button>
      </nav>

      <h1 className="text-xl font-semibold text-slate-900">
        Compliance Checker
      </h1>

      {checker.isPending && (
        <p className="mt-4 text-sm text-slate-600">Memuat pekerjaan…</p>
      )}

      {checker.isError && <OpenFailure error={checker.error} />}

      {checker.data && (
        <CheckerForm
          // `key` memaksa React membuang keadaan form ketika pekerjaan yang dibuka
          // BERGANTI tanpa halamannya dibongkar — terjadi bila petugas berpindah lewat
          // alamat. Tanpa ini, isian pekerjaan sebelumnya terbawa ke pekerjaan
          // berikutnya, dan petugas dapat menyimpannya tanpa sadar.
          //
          // Ia menggantikan useEffect yang menyemai ulang isian. Versi useEffect itu
          // punya dua cacat yang `key` tidak punya: ia MENIMPA ketikan petugas setiap
          // kali TanStack Query mengambil ulang datanya, dan ia membaca `data.klaim`
          // yang akan meledak bila badan responsnya tidak berbentuk seperti dugaan.
          //
          // Kuncinya kunci klaim dari ALAMAT, bukan dari badan respons: alamatlah yang
          // menyatakan pekerjaan mana yang sedang dibuka, dan ia sudah ada sebelum
          // responsnya tiba.
          key={referensi}
          data={checker.data}
          onSubmit={(body) => submit.mutate(body)}
          isSaving={submit.isPending}
          saveError={submit.isError ? submit.error : null}
          saved={submit.data ?? null}
        />
      )}
    </main>
  );
}

/**
 * Kegagalan membuka form.
 *
 * `ErrClaimNotInQueue` dipisahkan dari galat lain, dan pemisahan itu yang penting: ia
 * bukan "tidak ditemukan". Klaimnya boleh jadi ada dan sehat, hanya sudah diputuskan
 * petugas lain beberapa detik sebelumnya. Pesan "tidak ditemukan" akan membuat petugas
 * mencari klaimnya, padahal yang perlu dilakukan hanyalah kembali dan menyegarkan daftar.
 */
function OpenFailure({ error }: { error: unknown }) {
  const conflict = error instanceof APIError && error.status === 409;

  return (
    <div className="mt-4">
      <ErrorMessage
        tone={conflict ? "penolakan" : "gangguan"}
        title={
          conflict
            ? "Klaim tidak ada di antrean Compliance"
            : "Pekerjaan tidak dapat dibuka"
        }
        description={
          error instanceof APIError
            ? error.message
            : "Terjadi gangguan saat membuka pekerjaan ini."
        }
      />
    </div>
  );
}

/**
 * Badan permintaan penyimpanan keputusan.
 *
 * Diturunkan dari kontraknya, bukan diketik ulang: bentuk kedua yang ditulis tangan akan
 * diam-diam berbeda dari `SubmitDecisionRequest` begitu kontraknya bertambah — dan itu
 * persis yang terjadi pada `aksi`.
 */
type DecisionBody = SubmitDecisionRequest;

/** Nilai Lain-Lain — satu-satunya pilihan yang memunculkan "Note Lainya". */
const CHOICE_OTHER = "3";

/** Satu baris grid komentar, sebagaimana disunting di layar. */
type CommentRow = { tanggal: string; komentar: string };

// Konstanta BELUM_BERFUNGSI DICABUT 2026-10-08.
//
// Ia menjadi keterangan tombol yang tampil tetapi belum bekerja. Ketiga tombol yang
// memakainya — Unggah Dokumen, Download Dokumen Reject, dan grid dokumen — kini
// seluruhnya berfungsi, sehingga keterangannya tidak punya pemakai lagi.

function CheckerForm({
  data,
  onSubmit,
  isSaving,
  saveError,
  saved,
}: {
  data: CheckerResponse;
  onSubmit: (body: DecisionBody) => void;
  isSaving: boolean;
  saveError: unknown;
  saved: {
    keputusan: { pilihan_label: string };
    post_audit: { nomor_case: string } | null;
  } | null;
}) {
  // Isian disemai dari keputusan yang SUDAH tersimpan, bila ada.
  //
  // Bukan kotak kosong: petugas yang membuka form kedua kalinya harus melihat pilihan yang
  // sudah dibuatnya, bukan layar yang membuatnya mengira keputusannya hilang.
  const [choice, setChoice] = useState(data.keputusan?.pilihan ?? "");
  const [note, setNote] = useState(data.keputusan?.note ?? "");

  // Tab yang sedang terbuka. "compliance" lebih dulu, karena ia tab pertama di
  // `Section/ComplianceChecker-Section.xml` (S1) dan satu-satunya yang selalu ada.
  const [tab, setTab] = useState<"compliance" | "hasil-investigasi" | "dokumen">("compliance");

  const unggah = useUploadComplianceDocument(data.klaim.referensi);
  const pemilihBerkas = useRef<HTMLInputElement>(null);

  // Dialog Surat Penolakan.
  //
  // Isiannya TIDAK disimpan sebagai data klaim — ia bahan surat, berumur satu
  // permintaan. Karena itu dialognya tidak menyimpan keadaan apa pun di sini; ia
  // memulai dari pra-isi setiap kali dibuka.
  const [dialogSuratTerbuka, setDialogSuratTerbuka] = useState(false);
  const suratPenolakan = useGenerateRejectLetter(data.klaim.referensi);

  // Grid komentar — satu baris per komentar, masing-masing bertanggal sendiri.
  //
  // Kosong ketika klaimnya belum pernah dikomentari, dan gridnya menampilkan
  // "Data Tidak Ada" — meniru Pega. Versi sebelumnya memaksa satu baris kosong selalu
  // ada, yang di Pega tidak terjadi.
  const [comments, setComments] = useState<CommentRow[]>(() =>
    (data.keputusan?.komentar ?? []).map((row) => ({
      tanggal: row.tanggal ?? "",
      komentar: row.komentar,
    })),
  );

  const violations = fieldViolations(saveError);

  function updateComment(index: number, patch: Partial<CommentRow>) {
    setComments((rows) =>
      rows.map((row, i) => (i === index ? { ...row, ...patch } : row)),
    );
  }

  /**
   * Mengirim isian form beserta TOMBOL yang ditekan.
   *
   * Keduanya tidak boleh tertukar: "simpan" meninggalkan klaim di antrean, "kirim"
   * memindahkannya dan membuat form ini tidak dapat dibuka lagi.
   */
  function kirimForm(aksi: "simpan" | "kirim") {
    onSubmit({
      pilihan: choice,
      note,
      // Tanggal dikirim apa adanya. Kosong berarti server memakai waktu keputusan —
      // padanan nilai bawaan `CurrentDateTime()` pada sel Pega, yang di sana pun masih
      // dapat diubah petugas.
      komentar: comments,
      aksi,
    });
  }

  // `onSubmit` form hanya dipicu tombol "Simpan Data" — satu-satunya ber-`type="submit"`.
  // Kedua tombol Kirim memanggil kirimForm('kirim') langsung, supaya Enter di dalam kotak
  // teks tidak pernah memindahkan klaim.
  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    kirimForm("simpan");
  }

  return (
    <form onSubmit={handleSubmit} className="mt-4 space-y-5">
      {/*
        Ringkasan klaim — padanan tab **Information** pada harness Pega.

        Ia BUKAN tambahan: Pega menampilkan isian yang sama, hanya di tab terpisah,
        ditambah nomor kasusnya di kepala harness (`PA (PNC-2114) New`). Layar ini belum
        bertab, sehingga ringkasannya digambar di atas.
      */}
      <ClaimSummary claim={data.klaim} />

      {/*
        Bilah tab — `pyNewTabGroup = true` pada layout S1
        `Section/ComplianceChecker-Section.xml`, dengan `pyTabAlignment = Top`.

        KOREKSI: pembacaan pertama menyimpulkan keempat layout itu blok `SUBHEADER` yang
        bertumpuk, karena `pyContainerType`-nya memang SUBHEADER. Itu salah — yang
        menentukan bentuknya adalah `pyNewTabGroup`, dan layar Pega yang berjalan memang
        menggambarnya sebagai TAB.

        Ketiga tabnya tidak pernah muncul bersamaan — syaratnya saling meniadakan:

          Compliance         selalu
          Hasil Investigasi  IsPA      GroupPanel "002"
          Dokumen            IsTravel  GroupPanel "005"

        Jadi klaim PA melihat dua tab, klaim Travel melihat dua tab yang berbeda, dan
        lini lain hanya melihat satu.
      */}
      {/*
        Judul "Compliance Checker" — `pyLabel` pada
        `Flow Action/ComplianceChecker-FA.xml` — sudah digambar sebagai `<h1>` di kepala
        halaman ini, tepat di atas bilah tab seperti di Pega. Tidak diulang di sini.
      */}
      <div className="flex gap-1 border-b border-slate-200" role="tablist">
        <TabButton
          active={tab === "compliance"}
          onClick={() => setTab("compliance")}
        >
          Compliance
        </TabButton>

        {data.tampilkan_hasil_investigasi && (
          <TabButton
            active={tab === "hasil-investigasi"}
            onClick={() => setTab("hasil-investigasi")}
          >
            Hasil Investigasi
          </TabButton>
        )}

        {data.tampilkan_daftar_dokumen && (
          <TabButton
            active={tab === "dokumen"}
            onClick={() => setTab("dokumen")}
          >
            Dokumen
          </TabButton>
        )}
      </div>

      {/*
        Kotak peringatan "Keputusan belum mengubah klaim di sistem lama" DIBUANG
        2026-10-07, atas keputusan Work Owner: layar disamakan dengan Pega, tanpa ditambah
        maupun dikurangi — dan Pega tidak punya kotak itu.

        Ia juga sudah separuh tidak benar: sejak Ticket `SendtoAnalysator` dibangun, klaim
        BERPINDAH — status berubah, riwayat tertulis, dan ia keluar dari antrean
        Compliance kita.

        Yang MASIH benar dan karena itu tidak hilang begitu saja: baris antrean di Pega
        sendiri tidak ikut dihapus, sehingga klaimnya tetap tampak di Inbox Compliance
        Pega. Keterangan itu pindah ke catatan lingkup penguji — tempat yang membacanya
        orang yang perlu tahu, bukan petugas yang setiap hari memakai form ini.

        Server TETAP mengirim `keterbatasan`; yang dicabut hanya penggambarannya.
      */}

      {/*
        Keterangan dari Investigator — READ-ONLY, dan urutannya paling atas.

        Keduanya mengikuti `Section/CompliancePNC-Section.xml`: selnya bertanda
        `pyEditOptions=Read-only` dan berlabel persis begini. Ia MENAMPILKAN catatan
        Investigator; ia bukan isian petugas Compliance. Versi sebelumnya form ini keliru
        menjadikannya kotak isian berlabel "Catatan Dari Compliance".
      */}
      {/*
        SATU kotak untuk seluruh isi tab Compliance — bukan tiga kartu terpisah.

        Begitulah `CompliancePNC` tergambar di Pega: satu bingkai yang memuat Keterangan,
        grid komentar, Pilihan, dan bilah tombol berurutan. Versi sebelumnya memecahnya
        menjadi tiga kartu bersekat, yang membuat layar terlihat jelas berbeda meski
        isiannya sama.
      */}
      {tab === "compliance" && (
        <div className="space-y-5 rounded-kartu border border-slate-300 bg-white p-5">
          <section>
            <h2 className="text-sm font-semibold text-slate-900">
              Keterangan dari Investigator
            </h2>
            {data.keputusan?.catatan_investigator ? (
              <p className="mt-2 whitespace-pre-wrap text-sm text-slate-700">
                {data.keputusan.catatan_investigator}
              </p>
            ) : (
              /*
            Catatan Investigator TIDAK PUNYA KOLOM di basis data — dan ini sudah
            ditelusuri sampai habis, bukan dugaan:

              · `ComplianceRemark` dialiaskan di 3 kueri, dan ketiganya MENYESATKAN —
                `TO_CHAR(a.tgl_aksep,'yyyy')` pada ExportDataDetailKlaim, dan
                `SUM(TOTAL_CLAIM*CURRENCYVALUE)` pada BrowseClaimStudy.
              · `Report Definition/InboxCompliance_RD-RD.xml` menandai `ComplianceRemarks`
                sebagai properti yang "may result in poor performance" — penanda khas
                properti yang hidup di BLOB, bukan di kolom.
              · `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc` hanya membawa
                `TanggalBuatCompliance` → `COMPLIANCE_CREATEDATE`. Tidak ada teksnya.

            Jadi nilainya hidup di BLOB objek kerja Pega (`pzPVStream`), yang tidak dapat
            dibaca Go tanpa membongkar format milik Pega. Itu yang harus diminta ke Tim
            Pega — bukan Report Definition baru.
          */
              <p className="mt-2 text-sm text-slate-500">
                Belum tersedia. Catatan ini hanya tersimpan di dalam objek kerja
                Pega, tanpa kolom basis data — menunggu Tim Pega mengeluarkannya
                ke kolom atau layanan.
              </p>
            )}
          </section>

          {/*
        Grid komentar — `.ClaimData.ComplianceList`. INILAH isian petugas.

        Bentuknya disamakan dengan layar Pega setelah dibandingkan berdampingan
        (2026-10-07): TABEL dua kolom berjudul "Tanggal Komentar" dan "Komentar", dengan
        tautan Tambah dan Hapus di atasnya, dan "Data Tidak Ada" ketika kosong.

        Versi sebelumnya menumpuk kotak teks berlabel "Komentar 1", "Komentar 2", tanpa
        kolom tanggal sama sekali — dan memaksa satu baris kosong selalu ada.

        Tanggal dapat disunting, meniru Pega: di sana ia bernilai bawaan waktu sekarang
        yang MASIH dapat diubah petugas. Dibiarkan kosong berarti server memakai waktu
        keputusan.
      */}
          <section>
            {/*
          TANPA judul "Komentar", dan tautannya di KIRI — bukan di kanan.

          Di Pega, "✚ Tambah  Hapus" berdiri sendiri tepat di bawah Keterangan dari
          Investigator, langsung di atas tabelnya. Tidak ada judul di antaranya. Judul
          "Komentar" adalah tambahan kami, dan kolom tabelnya sudah menyebutkan isinya.
        */}
            <div className="flex flex-wrap items-center gap-4">
              <div className="flex gap-4 text-sm">
                <button
                  type="button"
                  onClick={() =>
                    setComments((rows) => [
                      ...rows,
                      { tanggal: "", komentar: "" },
                    ])
                  }
                  className="font-medium text-sky-700 hover:text-sky-900"
                >
                  ✚ Tambah
                </button>
                <button
                  type="button"
                  disabled={comments.length === 0}
                  onClick={() => setComments((rows) => rows.slice(0, -1))}
                  className="font-medium text-sky-700 hover:text-sky-900 disabled:text-slate-400"
                >
                  Hapus
                </button>
              </div>
            </div>

            <div className="mt-2 overflow-x-auto">
              <table className="w-full border-collapse text-sm">
                <thead>
                  <tr className="border-b border-slate-200 text-left text-slate-600">
                    <th className="w-56 py-2 pr-3 font-medium">
                      Tanggal Komentar
                    </th>
                    <th className="py-2 font-medium">Komentar</th>
                  </tr>
                </thead>
                <tbody>
                  {comments.length === 0 && (
                    <tr>
                      <td colSpan={2} className="py-3 text-slate-500">
                        Data Tidak Ada
                      </td>
                    </tr>
                  )}

                  {comments.map((row, index) => (
                    // Indeks sebagai kunci aman di sini: baris hanya ditambah di ujung dan
                    // dihapus dari ujung — tidak pernah disisipkan di tengah.
                    <tr
                      key={index}
                      className="border-b border-slate-100 align-top"
                    >
                      <td className="py-2 pr-3">
                        <input
                          type="datetime-local"
                          aria-label={`Tanggal Komentar ${index + 1}`}
                          value={row.tanggal}
                          onChange={(event) =>
                            updateComment(index, {
                              tanggal: event.target.value,
                            })
                          }
                          className="w-full rounded-kontrol border border-slate-300 px-2 py-1"
                        />
                      </td>
                      <td className="py-2">
                        <textarea
                          aria-label={`Komentar ${index + 1}`}
                          rows={2}
                          value={row.komentar}
                          onChange={(event) =>
                            updateComment(index, {
                              komentar: event.target.value,
                            })
                          }
                          className="w-full rounded-kontrol border border-slate-300 px-2 py-1"
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {violations.komentar && (
              <p role="alert" className="mt-2 text-sm text-rose-700">
                {violations.komentar}
              </p>
            )}
          </section>

          <section>
            {/*
          Judulnya "Pilihan", bukan "Pilihan Compliance".

          Diambil dari `pyLabelFieldValue` sel 32 `Section/CompliancePNC-Section.xml`,
          yang `pyLabelFormat = Heading 2` — karena itu digambar sebagai h2. Versi
          sebelumnya memakai "Pilihan Compliance", nama karangan sendiri (`D-13`).
        */}
            <h2 className="text-sm font-semibold text-slate-900">Pilihan</h2>

            {/* Mendatar, 3 per baris — `pyOrientation=horizontal`, `pyWrapBefore=3`. */}
            <fieldset className="mt-3 grid gap-2 sm:grid-cols-3">
              <legend className="sr-only">Pilihan</legend>
              {data.pilihan.map((option) => (
                <ChoiceRadio
                  key={option.nilai}
                  option={option}
                  checked={choice === option.nilai}
                  onSelect={() => setChoice(option.nilai)}
                />
              ))}
            </fieldset>

            {violations.pilihan && (
              <p role="alert" className="mt-2 text-sm text-rose-700">
                {violations.pilihan}
              </p>
            )}

            {/*
          Peringatan khusus Bayar/PostAudit.

          Pilihan ini SATU-SATUNYA yang menerbitkan baris baru — `SetComplianceResult`
          langkah 10, `pxAddChildWork` kelas `Work-Compliance`. Tiga pilihan lain hanya
          mengubah keputusan yang tersimpan. Menekan Simpan dua kali pada pilihan ini
          menerbitkan DUA baris Post Audit, dan tabelnya tidak punya constraint yang akan
          menolaknya.
        */}
            {choice === CHOICE_POST_AUDIT && (
              <p className="mt-3 rounded-kontrol bg-amber-50 p-3 text-sm text-amber-900">
                Pilihan ini menerbitkan satu baris Post Audit baru setiap kali
                disimpan.
              </p>
            )}
          </section>

          {/*
        "Note Lainya" muncul HANYA pada pilihan Lain-Lain.

        Syaratnya dibaca apa adanya dari selnya: `.ClaimData.PilihanCompliance==3`,
        `pyVisible=OTHER`, `pyIsClientWhen=true`. Isian ini ditambahkan ke Pega pada
        10 Januari 2025 — catatan pengembangnya masih tertinggal di rule-nya.

        Nilainya TETAP dikirim saat pilihan berganti, meniru Pega: kondisinya di sana sisi
        klien, sehingga isian yang tersembunyi tetap ikut tersimpan saat Obj-Save.
      */}
          {choice === CHOICE_OTHER && (
            <TextAreaField
              id="note-pilihan-compliance"
              label="Note Lainya"
              rows={3}
              value={note}
              onChange={(event) => setNote(event.target.value)}
              error={violations.note}
            />
          )}

          {/*
            Grid dokumen DICABUT 2026-10-08, atas pengamatan Work Owner pada layar Pega.

            Grid "Nama File / Type File / Category" memang ADA di
            `Section/CompliancePNC-Section.xml`, dan itu yang membuat saya menggambarnya.
            Yang saya lewatkan adalah SUMBER DATANYA: `TempDocumentAttach` berkelas
            `Code-Pega-List` — halaman klipboard, bukan tabel — dan satu-satunya yang
            mengisinya adalah `SaveAllDataattachfilecompilance` serta
            `InsertHistoryClaimPNC_compilance`, keduanya berjalan saat MENYIMPAN.

            Artinya Pega tidak pernah memuat grid ini dari basis data saat form dibuka. Ia
            kosong sampai ada unggahan pada sesi yang sama. Kami memuatnya dari
            `DATA_ATTACHFILE` setiap kali form dibuka — itu menampilkan berkas yang di
            Pega tidak pernah terlihat di layar ini, dan terbukti pada `PNC-2114` yang
            punya tiga lampiran di basis data tetapi kosong di Pega.

            Lampiran klaim di Pega dibaca lewat panel **Lampiran** di bawah layar, bukan
            lewat form ini.
          */}

          {saveError !== null && Object.keys(violations).length === 0 && (
            <ErrorMessage
              tone="gangguan"
              title="Keputusan gagal disimpan"
              description={
                saveError instanceof APIError
                  ? saveError.message
                  : "Terjadi gangguan saat menyimpan keputusan."
              }
            />
          )}

          {saved && (
            <ErrorMessage
              tone="gangguan"
              title={`Keputusan tersimpan: ${saved.keputusan.pilihan_label}`}
              description={
                saved.post_audit
                  ? `Baris Post Audit terbit dengan nomor ${saved.post_audit.nomor_case}. ` +
                    "Klaim di sistem lama belum berpindah status."
                  : "Klaim di sistem lama belum berpindah status."
              }
            />
          )}

          {/*
        Deretan KELIMA tombol, berurutan seperti selnya di Pega:

          76/15  Unggah Dokumen            selalu
          77/16  Download Dokumen Reject   selalu
          78/17  Simpan Data               selalu
          79/18  Kirim ke Analyst          IsPA
          80/19  Kirim ke PIC Teknik       IsTravel

        Dua nomor sel karena Pega punya DUA bilah tombol yang saling melengkapi —
        `CompliancePNC` S14 (`!IsTravel`) dan `ComplianceChecker` S4 (`IsTravel`).

        Versi sebelumnya membaca S14 saja, menaruh "Simpan Data" di urutan terakhir, dan
        menyimpulkan klaim Travel tidak mendapat tombol apa pun. Ketiganya salah.

        Tiap tombol muncul hanya bila SERVER membolehkannya; tidak satu pun syarat
        disimpulkan di sini dari lini bisnis (`D-59`).
      */}
          <div className="flex flex-wrap justify-end gap-2">
            {data.tombol.unggah_dokumen && (
              <>
                {/*
                  Pemilih berkas disembunyikan, tombolnya yang terlihat.

                  Pega memakai `pzMultiFilePath` di dalam dialog tersendiri; di sini
                  pemilih bawaan peramban dipakai langsung — satu langkah lebih sedikit
                  untuk hasil yang sama, dan dialog itu sendiri tidak membawa isian lain
                  yang perlu diisi.
                */}
                <input
                  ref={pemilihBerkas}
                  type="file"
                  className="hidden"
                  onChange={(event) => {
                    const berkas = event.target.files?.[0];
                    // Nilainya dikosongkan supaya memilih BERKAS YANG SAMA dua kali
                    // tetap memicu onChange.
                    event.target.value = "";
                    if (berkas) unggah.mutate(berkas);
                  }}
                />
                <Button
                  type="button"
                  tone="kedua"
                  disabled={unggah.isPending}
                  onClick={() => pemilihBerkas.current?.click()}
                >
                  {unggah.isPending ? "Mengunggah…" : "Unggah Dokumen"}
                </Button>
              </>
            )}

            {data.tombol.unduh_dokumen_reject && (
              <Button
                type="button"
                tone="kedua"
                disabled={suratPenolakan.isPending}
                onClick={() => setDialogSuratTerbuka(true)}
              >
                Download Dokumen Reject
              </Button>
            )}

            {data.tombol.simpan && (
              <Button type="submit" tone="utama" disabled={isSaving}>
                {isSaving ? "Menyimpan…" : "Simpan Data"}
              </Button>
            )}

            {/*
              Kedua tombol Kirim menjalankan `SetComplianceResult` yang SAMA, dan
              tujuannya ditentukan lini bisnis — bukan tombolnya. Karena itu keduanya
              mengirim `aksi: "kirim"` yang sama, dan tidak ada tombol yang memilih
              tujuan sendiri.

              Keduanya `type="button"`, bukan submit: submit milik "Simpan Data".
            */}
            {data.tombol.kirim_ke_analyst && (
              <Button
                type="button"
                tone="kedua"
                disabled={isSaving}
                onClick={() => kirimForm("kirim")}
              >
                {isSaving ? "Mengirim…" : "Kirim ke Analyst"}
              </Button>
            )}

            {data.tombol.kirim_ke_pic_teknik && (
              <Button
                type="button"
                tone="kedua"
                disabled={isSaving}
                onClick={() => kirimForm("kirim")}
              >
                {isSaving ? "Mengirim…" : "Kirim ke PIC Teknik"}
              </Button>
            )}
          </div>
        </div>
      )}

      {/*
        Tab "Hasil Investigasi" — layout S2, bersyarat `IsPA`.

        Isinya tabel baca-saja; tidak ada tombol di tab ini. Bilah tombol milik tab
        Compliance (S14, di dalam `CompliancePNC`), sehingga ia ikut tersembunyi di sini —
        persis seperti layar Pega.
      */}
      {tab === "hasil-investigasi" && data.tampilkan_hasil_investigasi && (
        <SurveyResultsBlock
          rows={data.hasil_investigasi}
          failed={data.hasil_investigasi_gagal_dibaca}
        />
      )}

      {tab === "dokumen" && data.tampilkan_daftar_dokumen && (
        <DocumentChecklistBlock
          rows={data.daftar_dokumen}
          failed={data.daftar_dokumen_gagal_dibaca}
          reference={data.klaim.referensi}
        />
      )}

      {/*
        Kabar setelah surat terbit.

        Ia menyebut secara tegas bahwa suratnya DILAMPIRKAN, bukan diunduh — tanpa itu
        petugas menunggu berkas yang tidak akan datang, lalu menekan tombolnya lagi.
      */}
      {suratPenolakan.isSuccess && (
        <ErrorMessage
          tone="gangguan"
          title={
            suratPenolakan.data.mengganti_surat_sebelumnya
              ? "Surat penolakan diperbarui"
              : "Surat penolakan dibuat"
          }
          description={
            `${suratPenolakan.data.dokumen.nama_file} tersimpan sebagai dokumen klaim ` +
            "ini. Buka lewat grid Dokumen di tab Compliance."
          }
        />
      )}

      {dialogSuratTerbuka && (
        <RejectLetterDialog
          prefill={{
            nama_pasien: data.pra_isi_surat_penolakan.nama_pasien,
            tempat_kejadian: data.pra_isi_surat_penolakan.tempat_kejadian,
            tanggal_kejadian:
              data.pra_isi_surat_penolakan.tanggal_kejadian ?? undefined,
          }}
          sedangMenerbitkan={suratPenolakan.isPending}
          galat={
            suratPenolakan.error instanceof APIError
              ? suratPenolakan.error.message
              : suratPenolakan.error
                ? "Surat penolakan gagal dibuat. Coba lagi."
                : null
          }
          onBatal={() => {
            suratPenolakan.reset();
            setDialogSuratTerbuka(false);
          }}
          onTerbitkan={(isian) => {
            suratPenolakan.mutate(isian, {
              // Dialognya ditutup hanya ketika BERHASIL. Pada kegagalan ia tetap
              // terbuka beserta isiannya — isian itu tidak tersimpan di mana pun,
              // sehingga menutup dialog berarti petugas mengetik ulang delapan isian
              // dan seluruh baris alasan.
              onSuccess: () => setDialogSuratTerbuka(false),
            });
          }}
        />
      )}
    </form>
  );
}

/**
 * Satu tab pada bilah tab form.
 *
 * `type="button"` WAJIB: tombol tanpa tipe di dalam `<form>` bertipe `submit`, sehingga
 * berpindah tab akan menyimpan keputusan — akibat yang jauh lebih besar daripada salah
 * ketik biasa, karena keputusan ini memindahkan klaim.
 */
function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={
        active
          ? "border-b-2 border-biru-600 px-4 py-2 text-sm font-semibold text-biru-700"
          : "border-b-2 border-transparent px-4 py-2 text-sm text-slate-600 hover:text-slate-900"
      }
    >
      {children}
    </button>
  );
}

/** Nilai Bayar/PostAudit — satu-satunya pilihan yang menerbitkan baris baru. */
const CHOICE_POST_AUDIT = "2";

function ChoiceRadio({
  option,
  checked,
  onSelect,
}: {
  option: Choice;
  checked: boolean;
  onSelect: () => void;
}) {
  return (
    <label className="flex cursor-pointer items-center gap-2.5 text-sm text-slate-800">
      <input
        type="radio"
        name="pilihan-compliance"
        value={option.nilai}
        checked={checked}
        onChange={onSelect}
        className="h-4 w-4 accent-sky-700"
      />
      <span>{option.label}</span>
    </label>
  );
}

/**
 * Blok "Hasil Investigasi" — layout S2 `Section/ComplianceChecker-Section.xml`, `IsPA`.
 *
 * # Empat kolom, seluruhnya baca-saja
 *
 * Keempatnya diambil dari grid `isPA_PNC` pada `Section/ViewHasilSurvey-Section.xml`,
 * sel header 149–152 dan sel isi 154–157. Section itu punya bentuk KEDUA — delapan isian
 * kepala ditambah grid lima kolom — tetapi bentuk itu bersyarat `!isPA_PNC`, sehingga ia
 * tidak pernah tampil di sini. Blok ini hanya muncul ketika `IsPA`, dan `IsPA` sama
 * dengan `isPA_PNC`: keduanya `GroupPanel = "002"`.
 *
 * Tidak ada tombol, tidak ada isian yang dapat diubah. Blok ini menampilkan hasil kerja
 * Investigator kepada petugas Compliance.
 */
// Komponen DocumentGrid DICABUT 2026-10-08 bersama gridnya — lihat catatan di tempat ia
// dulu digambar. Hook `useOpenComplianceDocument` dan `useDeleteComplianceDocument`
// SENGAJA dibiarkan di `api.ts`: endpointnya tetap ada dan teruji, dan keduanya yang akan
// dipakai panel Lampiran — tempat Pega sebenarnya menampilkan lampiran klaim.


function SurveyResultsBlock({
  rows,
  failed,
}: {
  rows: SurveyResult[];
  failed: boolean;
}) {
  return (
    <section className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
      <h2 className="text-sm font-semibold text-slate-900">
        Hasil Investigasi
      </h2>

      {/*
        Kegagalan baca DINYATAKAN, bukan digambar sebagai tabel kosong.

        Tabel kosong terbaca sebagai "belum pernah disurvei". Pada klaim PA yang sedang
        diputuskan, selisih antara "belum ada hasil" dan "hasilnya tidak terbaca"
        menentukan apakah petugas boleh memutuskan sekarang.
      */}
      {failed ? (
        <div className="mt-3">
          <ErrorMessage
            tone="gangguan"
            title="Hasil investigasi tidak dapat dibaca"
            description={
              "Keputusan tetap dapat disimpan, tetapi hasil investigasi klaim ini " +
              "sedang tidak terbaca — kosongnya tabel di bawah BUKAN berarti klaim " +
              "ini belum disurvei."
            }
          />
        </div>
      ) : null}

      <div className="mt-3 overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-left text-slate-600">
              <th className="w-48 py-2 pr-3 font-medium">
                Tanggal Investigasi
              </th>
              <th className="py-2 pr-3 font-medium">Nama Peserta</th>
              <th className="py-2 pr-3 font-medium">Lokasi Objek</th>
              <th className="w-40 py-2 font-medium">Status</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && (
              <tr>
                <td colSpan={4} className="py-3 text-slate-500">
                  Data Tidak Ada
                </td>
              </tr>
            )}

            {rows.map((row, index) => (
              // Indeks sebagai kunci: barisnya baca-saja dan urutannya tetap
              // (`ORDER BY INDEX_SURVEY`), sehingga tidak pernah disisipkan maupun
              // ditukar di layar.
              <tr key={index} className="border-b border-slate-100 align-top">
                <td className="py-2 pr-3">{row.tanggal_investigasi ?? ""}</td>
                <td className="py-2 pr-3">{row.nama_peserta}</td>
                <td className="py-2 pr-3">{row.lokasi_objek}</td>
                <td className="py-2">{row.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

/**
 * Ringkasan klaim yang sedang diperiksa.
 *
 * Kolomnya sama dengan yang dikirim server untuk baris daftar — bukan daftar kedua yang
 * disusun di sini. Kolom Nama Bisnis dan Nama Cabang yang disembunyikan pada grid tetap
 * ditampilkan di sini: alasan menyembunyikannya di grid adalah layar Pega yang hanya
 * menampilkan satu kolom, dan itu tidak berlaku pada form.
 */
function ClaimSummary({ claim }: { claim: WorkItem }) {
  return (
    <section className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
      <h2 className="text-sm font-semibold text-slate-900">Klaim</h2>
      <dl className="mt-3 grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2">
        <SummaryItem label="Nomor Case" value={claim.nomor_case} />
        <SummaryItem label="No Polis" value={claim.no_polis} />
        <SummaryItem label="Nama Tertanggung" value={claim.nama_tertanggung} />
        <SummaryItem label="Nama Bisnis" value={claim.nama_bisnis} />
        <SummaryItem label="Nama Cabang" value={claim.nama_cabang} />
        <SummaryItem label="Nama Admin" value={claim.nama_admin} />
        <SummaryItem
          label="Tanggal Kirim Compliance"
          value={
            claim.tanggal_kirim_compliance
              ? formatDate(claim.tanggal_kirim_compliance)
              : ""
          }
        />
        {/* "Lama Waktu Klaim" — judul kolom kedelapan di layar Pega, bukan "Aging". */}
        <SummaryItem label="Lama Waktu Klaim" value={claim.aging} />
      </dl>
    </section>
  );
}

function SummaryItem({
  label,
  value,
}: {
  label: string;
  value: string;
}): ReactNode {
  return (
    <div className="flex flex-col">
      <dt className="text-xs text-slate-500">{label}</dt>
      {/* Nilai kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan
          dari kolom yang gagal dimuat — sama dengan perlakuan di grid. */}
      <dd className="text-sm text-slate-900">{value || "—"}</dd>
    </div>
  );
}

/**
 * fieldViolations memetakan pelanggaran validasi 422 ke isiannya.
 *
 * Ia memanggil `APIError.violations()`, bukan membaca `detail` atau `field` sendiri:
 * ketiga modul master mengirim pelanggaran dengan bentuk yang berbeda hari ini, dan
 * helper itu ada supaya tidak satu layar pun perlu tahu modul mana memakai bentuk yang
 * mana. Menyalin logikanya ke sini berarti layar ini tertinggal ketika `TKT-F1-004`
 * menyeragamkan kontraknya.
 *
 * Server mengembalikan SELURUH pelanggaran sekaligus, bukan yang pertama saja (`P-5`), dan
 * ketiganya ditampilkan bersamaan. Mengambil yang pertama saja akan membuat petugas
 * memperbaiki satu isian, menekan Simpan, lalu menemukan isian berikutnya — persis
 * perilaku yang `P-5` ada untuk mencegahnya.
 */
function fieldViolations(error: unknown): Record<string, string> {
  if (!(error instanceof APIError)) return {};
  return error.violations();
}

/**
 * Tab **Dokumen** — layout S3 `Section/ComplianceChecker-Section.xml`, bersyarat
 * `IsTravel`, menyisipkan `Section/UploadDocument_sect.xml`.
 *
 * # Ia daftar periksa, BUKAN daftar berkas
 *
 * Barisnya adalah **kategori dokumen yang seharusnya ada** untuk lini bisnis klaim itu.
 * Kategori yang belum punya berkas tetap muncul — justru itu gunanya. Jangan tertukar
 * dengan grid dokumen pada tab Compliance yang dicabut 2026-10-08.
 *
 * # Tiga tombol Pega yang BELUM digambar di sini
 *
 *     Unggah Dokumen     UploadDokumenKlaim          InputParamUpload_act
 *     Lihat dokumen      GCNMViewAttachment2         -
 *     Ubah Kategori Dok  GCNMChangeViewAttachment    SetCategoryAttachment
 *
 * Ketiganya ada di export dan dapat dibangun; yang belum adalah permukaannya di sini.
 * Tabel ini dulu supaya kelengkapan dokumen sudah terbaca, dan tombolnya menyusul.
 */
function DocumentChecklistBlock({
  rows,
  failed,
  reference,
}: {
  rows: DocumentChecklistRow[];
  failed: boolean;
  reference: string;
}) {
  return (
    <section className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
      <h2 className="text-sm font-semibold text-slate-900">Dokumen</h2>

      {/*
        Kegagalan baca DINYATAKAN, dan di sini taruhannya lebih besar daripada pada Hasil
        Investigasi: tabel kosong terbaca sebagai "tidak ada dokumen yang wajib", dan
        petugas dapat meneruskan klaim yang dokumennya belum lengkap.
      */}
      {failed ? (
        <div className="mt-3">
          <ErrorMessage
            tone="gangguan"
            title="Daftar dokumen tidak dapat dibaca"
            description={
              "Keputusan tetap dapat disimpan, tetapi daftar kelengkapan dokumen klaim " +
              "ini sedang tidak terbaca — kosongnya tabel di bawah BUKAN berarti tidak " +
              "ada dokumen yang wajib diunggah."
            }
          />
        </div>
      ) : null}

      <div className="mt-3 overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-left text-slate-600">
              <th className="py-2 pr-3 font-medium">Kategori</th>
              <th className="w-32 py-2 pr-3 font-medium">Wajib Unggah</th>
              <th className="w-32 py-2 pr-3 font-medium">Minimal Unggah</th>
              <th className="w-44 py-2 font-medium">Total Sudah Diunggah</th>
              <th className="w-56 py-2 font-medium">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && (
              <tr>
                <td colSpan={4} className="py-3 text-slate-500">
                  Data Tidak Ada
                </td>
              </tr>
            )}

            {rows.map((row) => (
              <tr
                key={row.id_kategori}
                className="border-b border-slate-100 align-top"
              >
                <td className="py-2 pr-3 text-slate-900">{row.kategori}</td>

                {/*
                  Kosong dibiarkan KOSONG, tidak diganti "Tidak".

                  `STS_WAJIB` yang NULL menghasilkan sel kosong di Pega — dan memang
                  begitu yang terlihat pada layar Travel. Menggantinya dengan "Tidak"
                  adalah menjawab pertanyaan yang datanya tidak menjawab.
                */}
                <td className="py-2 pr-3 text-slate-700">{row.wajib_unggah}</td>

                <td className="py-2 pr-3 text-slate-700">
                  {row.minimal_unggah ?? ""}
                </td>
                <td className="py-2 text-slate-700">
                  {row.total_sudah_diunggah}
                </td>
                <td className="py-2">
                  <ChecklistRowActions row={row} reference={reference} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

/**
 * Ketiga tombol pada satu baris daftar periksa.
 *
 * Urutannya mengikuti kolom Pega: **Unggah Dokumen** · **Lihat dokumen** ·
 * **Ubah Kategori Dok**.
 *
 * # Satu perbedaan bentuk yang disengaja
 *
 * Di Pega ketiganya ikon, dan masing-masing membuka modal tersendiri
 * (`UploadDokumenKlaim`, `GCNMViewAttachment2`, `GCNMChangeViewAttachment`). Di sini
 * Unggah memakai pemilih berkas bawaan peramban — satu langkah lebih sedikit untuk hasil
 * yang sama — sedangkan Lihat dan Ubah Kategori membuka daftar di bawah barisnya alih-alih
 * modal bertumpuk di atas modal.
 *
 * Alasannya praktis: form ini sendiri sudah sebuah layar penuh, dan modal di atas modal
 * menyembunyikan baris yang sedang dikerjakan petugas.
 */
function ChecklistRowActions({
  row,
  reference,
}: {
  row: DocumentChecklistRow;
  reference: string;
}) {
  // `null` berarti tertutup; string kosong berarti "seluruh dokumen klaim" — dipakai
  // Ubah Kategori, karena dokumen yang kategorinya belum terdaftar pun harus terlihat.
  const [dibuka, setDibuka] = useState<"lihat" | "ubah" | null>(null);

  const pemilihBerkas = useRef<HTMLInputElement>(null);
  const unggah = useUploadComplianceDocument(reference);
  const buka = useOpenComplianceDocument(reference);
  const pindah = useChangeComplianceDocumentCategory(reference);

  const daftar = useComplianceDocumentsInCategory(
    reference,
    dibuka === "lihat" ? row.id_kategori : dibuka === "ubah" ? "" : null,
  );

  function tutup() {
    setDibuka(null);
  }

  return (
    <>
      <div className="flex flex-wrap gap-1">
        <input
          ref={pemilihBerkas}
          type="file"
          className="hidden"
          onChange={(event) => {
            const berkas = event.target.files?.[0];
            // Dikosongkan supaya memilih berkas YANG SAMA dua kali tetap memicu onChange.
            event.target.value = "";
            // Kategori baris inilah yang membedakan unggahan ini dari tombol Unggah
            // Dokumen pada tab Compliance — berkasnya mendarat di kategori yang ditekan.
            if (berkas) unggah.mutate(berkas);
          }}
        />
        <Button
          type="button"
          tone="kedua"
          disabled={unggah.isPending}
          aria-label={`Unggah Dokumen ${row.kategori}`}
          onClick={() => pemilihBerkas.current?.click()}
        >
          Unggah
        </Button>

        <Button
          type="button"
          tone="kedua"
          aria-label={`Lihat dokumen ${row.kategori}`}
          onClick={() => setDibuka(dibuka === "lihat" ? null : "lihat")}
        >
          Lihat
        </Button>

        <Button
          type="button"
          tone="kedua"
          aria-label={`Ubah Kategori Dok ${row.kategori}`}
          onClick={() => setDibuka(dibuka === "ubah" ? null : "ubah")}
        >
          Ubah Kategori
        </Button>
      </div>

      {dibuka !== null && (
        <div className="mt-2 rounded-kontrol border border-slate-200 bg-slate-50 p-2">
          {daftar.isPending && (
            <p className="text-xs text-slate-500">Memuat dokumen…</p>
          )}

          {daftar.isError && (
            <p className="text-xs text-merah-700">
              Dokumen tidak dapat dibaca. Coba lagi.
            </p>
          )}

          {daftar.data?.dokumen.length === 0 && (
            <p className="text-xs text-slate-500">Data Tidak Ada</p>
          )}

          <ul className="space-y-1">
            {daftar.data?.dokumen.map((d) => (
              <li key={d.id} className="flex items-center gap-2 text-xs">
                <span className="flex-1 text-slate-800">{d.nama_file}</span>

                {dibuka === "lihat" ? (
                  <button
                    type="button"
                    className="text-biru-700 hover:underline"
                    aria-label={`Buka ${d.nama_file}`}
                    onClick={() =>
                      buka.mutate(d.id, {
                        onSuccess: (hasil) => {
                          window.open(hasil.tautan, "_blank", "noopener");
                        },
                      })
                    }
                  >
                    Buka
                  </button>
                ) : (
                  <button
                    type="button"
                    className="text-biru-700 hover:underline disabled:text-slate-400"
                    // Dokumen yang SUDAH di kategori ini tidak perlu dipindahkan ke
                    // sini. Server pun menjawabnya berhasil tanpa menulis apa pun, tetapi
                    // tombol yang tidak melakukan apa-apa membingungkan.
                    disabled={d.kategori === row.id_kategori || pindah.isPending}
                    aria-label={`Pindahkan ${d.nama_file} ke ${row.kategori}`}
                    onClick={() =>
                      pindah.mutate(
                        { dokumen: d.id, kategori: row.id_kategori },
                        { onSuccess: tutup },
                      )
                    }
                  >
                    {d.kategori === row.id_kategori ? "Sudah di sini" : "Pindahkan ke sini"}
                  </button>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}
