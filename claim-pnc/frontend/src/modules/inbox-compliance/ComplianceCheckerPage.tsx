import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'
import { formatDate } from '@/components/format'

import { useComplianceChecker, useSubmitComplianceDecision } from './api'
import type { CheckerResponse, Choice, WorkItem } from './types'

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
 * # Apa yang BELUM ada di layar ini, dan kenapa
 *
 * `Section/ComplianceChecker-Section.xml` memuat TIGA tab dan LIMA tombol:
 *
 *     tab Compliance         section `CompliancePNC`      ← HILANG dari export (`R-16`)
 *     tab Hasil Investigasi  section `ViewHasilSurvey`
 *     tab Dokumen            section `UploadDocument`
 *
 *     Simpan Data · Kirim ke Analyst (when IsPA) · Kirim ke PIC Teknik (when IsTravel)
 *     Unggah Dokumen · Download Dokumen Reject
 *
 * Yang dibangun di sini baru **tab Compliance dan tombol Simpan Data**. Kedua tab lain
 * menyentuh modul survei dan lampiran yang belum ada, dan dua tombol terakhir menyentuh
 * rantai cetak PDF (`DownloadPDFReject`, `PrintRejectCompliancePDF`) yang menulis ke
 * lampiran klaim — tabel yang masih dimiliki Pega.
 *
 * Tab Compliance sendiri dibangun TANPA sectionnya, karena `CompliancePNC` tidak ada di
 * export. Isinya tetap terbaca dari tempat lain dan tidak satu pun dikarang:
 *
 *     Property/PilihanCompliance_property.xml       empat nilai beserta labelnya
 *     Section/CatatanToAnalyst_Section-Section.xml  caption ketiga isiannya
 *     Activity/SetComplianceResult                  akibat tiap pilihan
 *
 * Yang BELUM dapat dipastikan: urutan isian, mana yang wajib, dan lebar kotaknya.
 */
export function ComplianceCheckerPage() {
  const { referensi = '' } = useParams<{ referensi: string }>()
  const navigate = useNavigate()

  const checker = useComplianceChecker(referensi)
  const submit = useSubmitComplianceDecision(referensi)

  return (
    <main className="mx-auto w-full max-w-5xl px-4 py-6">
      <nav className="mb-4">
        <Button tone="halus" onClick={() => navigate('/inbox-compliance')}>
          ← Kembali ke Inbox Compliance
        </Button>
      </nav>

      <h1 className="text-xl font-semibold text-slate-900">Compliance Checker</h1>

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
  )
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
  const conflict = error instanceof APIError && error.status === 409

  return (
    <div className="mt-4">
      <ErrorMessage
        tone={conflict ? 'penolakan' : 'gangguan'}
        title={
          conflict
            ? 'Klaim tidak ada di antrean Compliance'
            : 'Pekerjaan tidak dapat dibuka'
        }
        description={
          error instanceof APIError
            ? error.message
            : 'Terjadi gangguan saat membuka pekerjaan ini.'
        }
      />
    </div>
  )
}

type DecisionBody = { pilihan: string; note: string; catatan: string }

function CheckerForm({
  data,
  onSubmit,
  isSaving,
  saveError,
  saved,
}: {
  data: CheckerResponse
  onSubmit: (body: DecisionBody) => void
  isSaving: boolean
  saveError: unknown
  saved: { keputusan: { pilihan_label: string }; post_audit: { nomor_case: string } | null } | null
}) {
  // Isian disemai dari keputusan yang SUDAH tersimpan, bila ada.
  //
  // Bukan kotak kosong: petugas yang membuka form kedua kalinya harus melihat pilihan yang
  // sudah dibuatnya, bukan layar yang membuatnya mengira keputusannya hilang.
  const [choice, setChoice] = useState(data.keputusan?.pilihan ?? '')
  const [note, setNote] = useState(data.keputusan?.note ?? '')
  const [remarks, setRemarks] = useState(data.keputusan?.catatan ?? '')

  const violations = fieldViolations(saveError)

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    onSubmit({ pilihan: choice, note, catatan: remarks })
  }

  return (
    <form onSubmit={handleSubmit} className="mt-4 space-y-5">
      <ClaimSummary claim={data.klaim} />

      {/*
        Keterbatasan ditampilkan SEBELUM isian, bukan setelah tombol.
        Petugas perlu mengetahuinya sebelum memutuskan, bukan setelah menekan Simpan.
      */}
      <ErrorMessage
        tone="gangguan"
        title="Keputusan belum mengubah klaim di sistem lama"
        description={data.keterbatasan}
      />

      <section className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
        <h2 className="text-sm font-semibold text-slate-900">Pilihan Compliance</h2>

        <fieldset className="mt-3 space-y-2">
          <legend className="sr-only">Pilihan Compliance</legend>
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
            Pilihan ini menerbitkan satu baris Post Audit baru setiap kali disimpan.
          </p>
        )}
      </section>

      <TextAreaField
        id="note-pilihan-compliance"
        label="Note PilihanCompliance"
        rows={3}
        value={note}
        onChange={(event) => setNote(event.target.value)}
        error={violations.note}
      />

      <TextAreaField
        id="catatan-compliance"
        label="Catatan Dari Compliance"
        rows={4}
        value={remarks}
        onChange={(event) => setRemarks(event.target.value)}
        error={violations.catatan_compliance}
        hint="Catatan ini yang tampil pada tab Post Audit bila pilihannya Bayar/PostAudit."
      />

      {saveError !== null && Object.keys(violations).length === 0 && (
        <ErrorMessage
          tone="gangguan"
          title="Keputusan gagal disimpan"
          description={
            saveError instanceof APIError
              ? saveError.message
              : 'Terjadi gangguan saat menyimpan keputusan.'
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
                'Klaim di sistem lama belum berpindah status.'
              : 'Klaim di sistem lama belum berpindah status.'
          }
        />
      )}

      <div className="flex justify-end">
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : 'Simpan Data'}
        </Button>
      </div>
    </form>
  )
}

/** Nilai Bayar/PostAudit — satu-satunya pilihan yang menerbitkan baris baru. */
const CHOICE_POST_AUDIT = '2'

function ChoiceRadio({
  option,
  checked,
  onSelect,
}: {
  option: Choice
  checked: boolean
  onSelect: () => void
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
  )
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
              : ''
          }
        />
        <SummaryItem label="Aging" value={claim.aging} />
      </dl>
    </section>
  )
}

function SummaryItem({ label, value }: { label: string; value: string }): ReactNode {
  return (
    <div className="flex flex-col">
      <dt className="text-xs text-slate-500">{label}</dt>
      {/* Nilai kosong menjadi tanda pisah, bukan sel kosong yang tidak dapat dibedakan
          dari kolom yang gagal dimuat — sama dengan perlakuan di grid. */}
      <dd className="text-sm text-slate-900">{value || '—'}</dd>
    </div>
  )
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
  if (!(error instanceof APIError)) return {}
  return error.violations()
}
