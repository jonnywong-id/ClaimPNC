import { useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { simpanBerkas } from '@/api/client'
import { centsToRupiah, formatRupiah, rupiahToCents, todayWIB } from '@/components/format'

import { useCurrencies, useFaceSheet, useItemOptions, useSaveEstimate, violationsFrom } from './api'
import { DocumentTab, ProgressTab, SurveyTab } from './EstimateTabs'
import { PLADialog } from './PLADialog'
import {
  EstimationType,
  type Claim,
  type CurrencyOption,
  type EstimateRequest,
  type InsuredItem,
  type ObjectItem,
  type Task,
} from './types'

/**
 * Layar tahap **Input Estimasi** — section `Section/InputEstimasiAdmin_SECT.xml`, flow
 * action `InputEstimasi` (Assignment7 Non-MBU, Assignment10 Travel).
 *
 * # Bagian atas dari section itu sendiri
 *
 * Status Klaim (`.ClaimData.StatusClaim`), tombol Detail Premi / Detail Polis / Riwayat
 * Klaim / Alasan Terlambat, Catatan ke PIC Teknis (`.ClaimData.Remark`), Aging Amount
 * (`.PaymentData.AgingAmount`), PIC Teknis (`.ClaimData.UserTeknis`), Log Transfer Klaim
 * (Tanggal, User, Catatan), lalu empat tab: Survey, Estimasi Pembayaran, Unggah Dokumen,
 * Progress Claim & Komunikasi.
 *
 * # Tab Estimasi Pembayaran — section InputEstimasiDetail
 *
 * Isinya sub-section `Section/InputEstimasiDetail_sect.xml` (kini ada di export), yang juga
 * ditanam `ClaimSurvey_sect` di layar InputSurveyor — karena itu isiannya dipegang
 * useEstimateEditor dan digambar EstimatePaymentTable, dipakai kedua layar. Susunannya
 * mengikuti tangkapan layar Pega dari Work Owner: objek (Nama Objek, Lokasi Object, Mata
 * Uang, Nilai Klaim, Nilai Adjuster) → jaminan (Jaminan, Mata Uang, TSI) → item (Objek,
 * Deskripsi Item) → baris estimasi (kolom section `Estimasi`).
 *
 * Tombol yang modulnya belum dibangun tampil tetapi mati. Download Claim Face Sheet sudah
 * berjalan: ia menyimpan isian, mengunduh PDF, lalu mengunci estimasi jaminan itu. Print PLA
 * membuka dialog PrintPLA_dtl (lihat PLADialog).
 *
 * # Kirim PIC Teknik menggantikan Next
 *
 * Section itu tidak punya tombol Next; tahapnya ditutup tombol Kirim PIC Teknik di baris
 * atas (`finishAssignment`), yang mati selama belum ada Claim Face Sheet (`!isCFS`).
 * Varian keduanya — popup `GCNMAlertSendRequest` bila `SurveyData.IsSendRequest` kosong —
 * tidak dibawa: section popup itu tidak ada di export, dan tidak satu rule pun di export
 * yang mengisi IsSendRequest.
 */

type EstimationForm = {
  tipe: string
  mata_uang: string
  tanggal: string
  nilai: string
  kurs_e4: number
  nilai_idr_sen: number
  /** Sudah dibuatkan Claim Face Sheet: baris hanya dibaca (PrintFaceClaim = 1). */
  terkunci: boolean
}

type ItemForm = {
  nama: string
  deskripsi: string
  kelompok: string
  estimasi: EstimationForm[]
  /** Jumlah estimasi yang sudah tersimpan — baris itu, dan itemnya, tidak dapat dihapus. */
  tersimpan: number
}

const TYPE_OPTIONS = [
  { value: EstimationType.Claim, label: 'Estimasi Klaim' },
  { value: EstimationType.Adjuster, label: 'Estimasi Adjuster' },
]

const TABS = ['Survey', 'Estimasi Pembayaran', 'Unggah Dokumen', 'Progress Claim & Komunikasi'] as const
type Tab = (typeof TABS)[number]

const NOT_BUILT = 'Modul ini belum dibangun.'

/** Pesan ValidateInputEstimate_act: estimasi berikutnya menunggu Claim Face Sheet. */
const WAIT_FACE_SHEET = 'Tidak Bisa Tambah Estimate, Belum Claim Face Sheet'

/** Jaminan sudah dibuatkan Claim Face Sheet — When `isCFS` pada section ObjectCoverage. */
function hasFaceSheet(items: ItemForm[]): boolean {
  return items.some((it) => it.estimasi.some((e) => e.terkunci))
}

/**
 * When `isCFS` tingkat klaim: `IsCFS_PNC` terisi sejak Claim Face Sheet pertama pada
 * jaminan mana pun. Mengatur tombol Kirim PIC Teknik.
 */
function claimHasFaceSheet(form: ItemForm[][][]): boolean {
  return form.some((coverages) => coverages.some(hasFaceSheet))
}

/** Polis tanpa koasuransi — When `IsNoCoins` (TYPEOFCOINS kosong atau 0). */
function noCoins(klaim: Claim): boolean {
  const jenis = klaim.polis.jenis_koasuransi ?? ''
  return jenis === '' || jenis === '0'
}

/** Jaminan punya estimasi yang belum dibuatkan Claim Face Sheet. */
function hasUnprinted(items: ItemForm[]): boolean {
  return items.some((it) => it.estimasi.some((e) => !e.terkunci))
}

function emptyItem(name: string): ItemForm {
  return { nama: name, deskripsi: '', kelompok: '', estimasi: [], tersimpan: 0 }
}

function fromClaim(klaim: Claim): ItemForm[][][] {
  return klaim.objek.map((o) =>
    o.coverage.map((c) => {
      const items = c.item ?? []
      if (items.length === 0) return [emptyItem('')]
      return items.map((it) => ({
        nama: it.nama,
        deskripsi: it.deskripsi,
        kelompok: it.kelompok ?? '',
        tersimpan: it.estimasi.length,
        estimasi: it.estimasi.map((e) => ({
          tipe: e.tipe || EstimationType.Claim,
          mata_uang: e.mata_uang,
          tanggal: e.tanggal,
          nilai: centsToRupiah(e.nilai_sen),
          kurs_e4: e.kurs_e4,
          nilai_idr_sen: e.nilai_idr_sen,
          terkunci: e.sudah_cfs === true,
        })),
      }))
    }),
  )
}

function toRequest(taskID: string, form: ItemForm[][][], kembali: boolean): EstimateRequest {
  return {
    tugas_id: taskID,
    kembali,
    objek: form.map((coverages) => ({
      coverage: coverages.map((items) => ({
        item: items.map(
          (it): ObjectItem => ({
            nama: it.nama,
            deskripsi: it.deskripsi,
            kelompok: it.kelompok,
            estimasi: it.estimasi.map((e) => ({
              tipe: e.tipe,
              mata_uang: e.mata_uang,
              tanggal: e.tanggal,
              nilai_sen: rupiahToCents(e.nilai) || 0,
              kurs_e4: 0,
              nilai_idr_sen: 0,
            })),
          }),
        ),
      })),
    })),
  }
}

/** Nilai Klaim dan Nilai Adjuster satu objek, dari estimasi yang sudah tersimpan (IDR). */
function objectTotals(o: InsuredItem): { klaim: number; adjuster: number } {
  let klaim = 0
  let adjuster = 0
  for (const c of o.coverage) {
    for (const it of c.item ?? []) {
      for (const e of it.estimasi) {
        if (e.tipe === EstimationType.Adjuster) adjuster += e.nilai_idr_sen
        else klaim += e.nilai_idr_sen
      }
    }
  }
  return { klaim, adjuster }
}

function currencyName(code: string, list: CurrencyOption[]): string {
  return list.find((m) => m.id === code)?.nama ?? code
}

/**
 * useEstimateEditor memegang isian section `InputEstimasiDetail`: pohon objek → jaminan →
 * item → estimasi, tombol Save (tanpa menutup tahap), Download Claim Face Sheet, dan Print
 * PLA. Section itu ditanam di DUA layar — Input Estimasi (`InputEstimasiAdmin_SECT`) dan
 * InputSurveyor (`ClaimSurvey_sect`) — sehingga isiannya hidup di sini, bukan di salah satu
 * layar.
 */
export function useEstimateEditor(klaim: Claim, tugas: Task) {
  const [form, setForm] = useState<ItemForm[][][]>(() => fromClaim(klaim))
  const save = useSaveEstimate(true)
  const faceSheet = useFaceSheet(klaim.id)
  // Jaminan yang dialog Print PLA-nya sedang terbuka.
  const [plaFor, setPLAFor] = useState<{ i: number; j: number } | null>(null)

  /**
   * Download Claim Face Sheet satu jaminan. Isian layar disimpan lebih dulu — Pega
   * membentuk CFS dari isian yang sedang tampil — lalu PDF diunduh dan estimasi jaminan
   * itu dikunci.
   */
  function downloadFaceSheet(i: number, j: number) {
    faceSheet.reset()
    save.mutate(toRequest(tugas.id, form, false), {
      onSuccess: (result) => {
        afterSave(result)
        faceSheet.mutate(
          { tugas_id: tugas.id, objek: i + 1, jaminan: j + 1 },
          {
            onSuccess: (file) => {
              simpanBerkas(file)
              updateItems(i, j, (items) =>
                items.map((it) => ({ ...it, estimasi: it.estimasi.map((e) => ({ ...e, terkunci: true })) })),
              )
            },
          },
        )
      },
    })
  }

  function updateItems(i: number, j: number, change: (items: ItemForm[]) => ItemForm[]) {
    setForm((previous) =>
      previous.map((coverages, oi) =>
        oi !== i ? coverages : coverages.map((items, ci) => (ci !== j ? items : change(items))),
      ),
    )
  }

  function updateItem(i: number, j: number, n: number, change: (item: ItemForm) => ItemForm) {
    updateItems(i, j, (items) => items.map((it, ii) => (ii === n ? change(it) : it)))
  }

  function afterSave(result: { klaim: Claim }) {
    setForm(fromClaim(result.klaim))
  }

  return {
    form,
    save,
    faceSheet,
    plaFor,
    setPLAFor,
    updateItems,
    updateItem,
    afterSave,
    downloadFaceSheet,
    /** Permintaan simpan dari isian yang sedang tampil. */
    request: (kembali: boolean) => toRequest(tugas.id, form, kembali),
    /** Save tanpa menutup tahap. */
    saveNow: () => save.mutate(toRequest(tugas.id, form, false), { onSuccess: afterSave }),
    hasFaceSheet: () => claimHasFaceSheet(form),
  }
}

export type EstimateEditor = ReturnType<typeof useEstimateEditor>

/**
 * EstimatePaymentTable menggambar isi section InputEstimasiDetail: grid objek (Nama Objek,
 * Lokasi Object, Mata Uang, Nilai Klaim, Nilai Adjuster, Informasi OS & Akseptasi), jaminan
 * di bawahnya, beserta dialog Print PLA.
 */
export function EstimatePaymentTable({
  klaim,
  tugas,
  editor,
  currencyList,
  busy,
}: {
  klaim: Claim
  tugas: Task
  editor: EstimateEditor
  currencyList: CurrencyOption[]
  busy: boolean
}) {
  const { form, updateItems, updateItem, downloadFaceSheet, plaFor, setPLAFor } = editor
  return (
    <div className="mt-3">
      {klaim.objek.length === 0 && (
        <p className="text-sm text-slate-600">Klaim ini belum punya objek. Kembali ke Input Register untuk mengisinya.</p>
      )}

      {klaim.objek.length > 0 && (
        <table className="w-full border-collapse text-sm">
          <caption className="sr-only">Objek klaim</caption>
          <thead>
            <tr className="bg-slate-100 text-left text-xs text-slate-700">
              <th className="p-2" />
              <th className="p-2">Nama Objek</th>
              <th className="p-2">Lokasi Object</th>
              <th className="p-2">Mata Uang</th>
              <th className="p-2 text-right">Nilai Klaim</th>
              <th className="p-2 text-right">Nilai Adjuster</th>
              <th className="p-2">Informasi OS &amp; Akseptasi</th>
            </tr>
          </thead>
          <tbody>
            {klaim.objek.map((o, i) => {
              const totals = objectTotals(o)
              return (
                <ObjectRows key={i}>
                  <tr className="bg-blue-100/70 align-top">
                    <td className="p-2">{i + 1}</td>
                    <td className="p-2">{o.nama || o.id}</td>
                    <td className="p-2">{o.lokasi || '—'}</td>
                    <td className="p-2">{currencyName(klaim.polis.mata_uang, currencyList)}</td>
                    <td className="p-2 text-right">{formatRupiah(totals.klaim)}</td>
                    <td className="p-2 text-right">{formatRupiah(totals.adjuster)}</td>
                    <td className="p-2">
                      <button type="button" disabled title={NOT_BUILT} className="rounded border border-blue-300 px-2 text-blue-700 disabled:opacity-60">
                        View
                      </button>
                    </td>
                  </tr>
                  <tr>
                    <td colSpan={7} className="border-l-2 border-slate-200 p-2 pl-6">
                      <CoverageTable
                        klaim={klaim}
                        objek={o}
                        objectIndex={i}
                        form={form[i] ?? []}
                        currencyList={currencyList}
                        updateItems={updateItems}
                        updateItem={updateItem}
                        busy={busy}
                        onFaceSheet={(j) => downloadFaceSheet(i, j)}
                        onPrintPLA={(j) => setPLAFor({ i, j })}
                      />
                    </td>
                  </tr>
                </ObjectRows>
              )
            })}
          </tbody>
        </table>
      )}

      <p className="mt-3 text-xs text-slate-500">
        Kurs dan nilai IDR dihitung saat disimpan, memakai kurs pada tanggal kejadian. Estimasi yang
        sudah disimpan tidak dapat dihapus, dan estimasi yang sudah dibuatkan Claim Face Sheet tidak
        dapat diubah.
      </p>

      {plaFor && (
        <PLADialog
          claimID={klaim.id}
          taskID={tugas.id}
          object={plaFor.i + 1}
          coverage={plaFor.j + 1}
          coverageName={klaim.objek[plaFor.i]?.coverage[plaFor.j]?.nama ?? ''}
          onClose={() => setPLAFor(null)}
        />
      )}
    </div>
  )
}

export function EstimateForm({ klaim, tugas }: { klaim: Claim; tugas: Task }) {
  const editor = useEstimateEditor(klaim, tugas)
  const { form, save, faceSheet, afterSave } = editor
  const [tab, setTab] = useState<Tab>('Estimasi Pembayaran')
  const complete = useSaveEstimate(false)
  const currencies = useCurrencies()
  const currencyList = currencies.data?.pilihan ?? []

  const busy = save.isPending || complete.isPending || faceSheet.isPending
  const failure = faceSheet.error ?? complete.error ?? save.error
  const violations = violationsFrom(failure)

  return (
    <section className="mt-6 rounded border border-slate-200 p-4" aria-label="Input Estimasi">
      <h2 className="text-sm text-slate-700">InputEstimasi</h2>

      {/* ── Bagian atas InputEstimasiAdmin ─────────────────────────────────────── */}
      <div className="mt-3 flex flex-wrap items-end justify-between gap-3">
        <label className="block text-xs font-semibold text-slate-800">
          Status Klaim
          <select
            disabled
            aria-label="Status Klaim"
            value={klaim.status_klaim}
            className="mt-1 block w-52 rounded border border-slate-300 bg-slate-50 px-2 py-1 text-sm font-normal"
          >
            <option value={klaim.status_klaim}>{klaim.status_klaim_nama || klaim.status_klaim || '—'}</option>
          </select>
        </label>
        <div className="flex flex-wrap items-center gap-2">
          {['Detail Premi', 'Detail Polis', 'Riwayat Klaim', 'Alasan Terlambat'].map((label) => (
            <button
              key={label}
              type="button"
              disabled
              title={NOT_BUILT}
              className="rounded border border-blue-300 px-2 py-1 text-sm text-blue-700 disabled:opacity-60"
            >
              {label}
            </button>
          ))}
          {/* Kirim PIC Teknik — finishAssignment pada InputEstimasiAdmin_SECT, mati bila !isCFS. */}
          <Button
            tone="utama"
            disabled={busy || !editor.hasFaceSheet()}
            title={editor.hasFaceSheet() ? undefined : 'Download Claim Face Sheet lebih dulu.'}
            onClick={() => complete.mutate(toRequest(tugas.id, form, false), { onSuccess: afterSave })}
          >
            {complete.isPending ? 'Memproses…' : 'Kirim PIC Teknik'}
          </Button>
        </div>
      </div>

      <label className="mt-4 block text-sm font-semibold text-slate-800">
        Catatan ke PIC Teknis
        <textarea
          disabled
          rows={4}
          className="mt-1 block w-full rounded border border-slate-300 bg-slate-50 px-2 py-1 text-sm font-normal"
          placeholder="Belum dapat disimpan: POOLDATA.T_CLAIM_PNC belum punya kolom untuk catatan ini."
        />
      </label>

      <dl className="mt-4 grid gap-4 sm:grid-cols-2">
        <div>
          <dt className="text-sm font-semibold text-slate-800">Aging Amount</dt>
          <dd className="text-sm text-slate-600">—</dd>
        </div>
        <div>
          <dt className="text-sm font-semibold text-slate-800">PIC Teknis</dt>
          <dd className="text-sm text-slate-600">{klaim.user_teknis || '—'}</dd>
        </div>
      </dl>

      <h3 className="mt-4 text-sm font-semibold text-slate-800">Log Transfer Klaim</h3>
      <table className="mt-1 w-full max-w-2xl border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-xs font-bold text-slate-700">
            <th className="py-1 pr-2">Tanggal</th>
            <th className="py-1 pr-2">User</th>
            <th className="py-1">Catatan</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td colSpan={3} className="py-2 text-xs text-slate-500">
              Data Tidak Ada
            </td>
          </tr>
        </tbody>
      </table>

      {/* ── Tab ─────────────────────────────────────────────────────────────────── */}
      <div role="tablist" className="mt-5 flex gap-1 border-b border-slate-200">
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            aria-selected={tab === t}
            onClick={() => setTab(t)}
            className={[
              'px-3 py-1.5 text-sm',
              tab === t ? 'border-b-2 border-blue-600 font-semibold text-slate-900' : 'text-slate-600',
            ].join(' ')}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === 'Survey' && <SurveyTab claimID={klaim.id} />}
      {tab === 'Unggah Dokumen' && <DocumentTab claimID={klaim.id} line={klaim.polis.lini} />}
      {tab === 'Progress Claim & Komunikasi' && <ProgressTab claimID={klaim.id} />}

      {tab === 'Estimasi Pembayaran' && (
        <EstimatePaymentTable klaim={klaim} tugas={tugas} editor={editor} currencyList={currencyList} busy={busy} />
      )}

      {failure && (
        <div className="mt-4">
          <ErrorMessage
            title={faceSheet.error ? 'Claim Face Sheet belum dapat dibuat' : complete.error ? 'Klaim belum dapat dikirim ke PIC Teknik' : 'Estimasi belum dapat disimpan'}
            description={violations.length > 0 ? violations.map((v) => v.pesan).join(' ') : errorText(failure)}
            tone="penolakan"
          />
        </div>
      )}
      {faceSheet.isSuccess && !busy && !failure && (
        <p className="mt-4 text-sm text-emerald-700" role="status">
          Claim Face Sheet diunduh. Estimasi jaminan itu kini terkunci.
        </p>
      )}
      {save.isSuccess && !faceSheet.isSuccess && !busy && !failure && (
        <p className="mt-4 text-sm text-emerald-700" role="status">
          Estimasi disimpan. Klaim tetap di tahap Input Estimasi.
        </p>
      )}

      <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
        <Button
          tone="halus"
          disabled={busy}
          onClick={() => complete.mutate(toRequest(tugas.id, form, true), { onSuccess: afterSave })}
        >
          Back
        </Button>
        <div className="flex gap-3">
          <Button
            tone="kedua"
            disabled={busy}
            onClick={editor.saveNow}
          >
            {save.isPending ? 'Menyimpan…' : 'Save'}
          </Button>
        </div>
      </div>
    </section>
  )
}

function ObjectRows({ children }: { children: ReactNode }) {
  return <>{children}</>
}

type Updaters = {
  updateItems: (i: number, j: number, change: (items: ItemForm[]) => ItemForm[]) => void
  updateItem: (i: number, j: number, n: number, change: (item: ItemForm) => ItemForm) => void
}

/** Grid jaminan satu objek: Jaminan, Mata Uang, TSI, lalu item-item di bawah setiap jaminan. */
function CoverageTable({
  klaim,
  objek,
  objectIndex: i,
  form,
  currencyList,
  updateItems,
  updateItem,
  busy,
  onFaceSheet,
  onPrintPLA,
}: {
  klaim: Claim
  objek: InsuredItem
  objectIndex: number
  form: ItemForm[][]
  currencyList: CurrencyOption[]
  busy: boolean
  onFaceSheet: (coverageIndex: number) => void
  onPrintPLA: (coverageIndex: number) => void
} & Updaters) {
  const options = useItemOptions(klaim.id, objek.id).data?.pilihan ?? []

  return (
    <table className="w-full border-collapse text-sm">
      <caption className="sr-only">Jaminan objek {objek.nama}</caption>
      <thead>
        <tr className="bg-slate-100 text-left text-xs text-slate-700">
          <th className="p-2" />
          <th className="p-2">Jaminan</th>
          <th className="p-2">Mata Uang</th>
          <th className="p-2 text-right">TSI</th>
          <th className="p-2" />
          <th className="p-2" />
        </tr>
      </thead>
      <tbody>
        {objek.coverage.map((c, j) => (
          <CoverageRows key={j}>
            <tr className="bg-blue-100/70">
              <td className="p-2">{j + 1}</td>
              <td className="p-2">{c.nama || c.id}</td>
              <td className="p-2">{currencyName(klaim.polis.mata_uang, currencyList)}</td>
              <td className="p-2 text-right">{formatRupiah(c.tsi_sen)}</td>
              <td className="p-2 text-center">
                <button
                  type="button"
                  disabled={busy || !hasUnprinted(form[j] ?? [])}
                  title={hasUnprinted(form[j] ?? []) ? undefined : 'Tidak ada estimasi baru untuk dibuatkan Claim Face Sheet.'}
                  onClick={() => onFaceSheet(j)}
                  className="rounded bg-blue-800 px-2 py-1 text-xs text-white disabled:opacity-60"
                >
                  Download Claim Face Sheet
                </button>
              </td>
              <td className="p-2 text-center">
                <button
                  type="button"
                  disabled={busy || noCoins(klaim) || !hasFaceSheet(form[j] ?? [])}
                  title={
                    noCoins(klaim)
                      ? 'Polis ini tidak berkoasuransi.'
                      : hasFaceSheet(form[j] ?? [])
                        ? undefined
                        : 'Buat Claim Face Sheet jaminan ini lebih dulu.'
                  }
                  onClick={() => onPrintPLA(j)}
                  className="rounded bg-slate-500 px-2 py-1 text-xs text-white disabled:opacity-60"
                >
                  Print PLA
                </button>
              </td>
            </tr>
            <tr>
              <td colSpan={6} className="p-2 pl-6">
                <ItemTable
                  klaim={klaim}
                  prefix={`${i}-${j}`}
                  items={form[j] ?? []}
                  options={options}
                  currencyList={currencyList}
                  onAdd={() => updateItems(i, j, (items) => [...items, emptyItem('')])}
                  onRemove={(n) => updateItems(i, j, (items) => items.filter((_, ii) => ii !== n))}
                  onChange={(n, change) => updateItem(i, j, n, change)}
                />
              </td>
            </tr>
          </CoverageRows>
        ))}
      </tbody>
    </table>
  )
}

function CoverageRows({ children }: { children: ReactNode }) {
  return <>{children}</>
}

/** Grid item satu jaminan: Objek, Deskripsi Item, Unggah Dokumen, Tambah/Hapus. */
function ItemTable({
  klaim,
  prefix,
  items,
  options,
  currencyList,
  onAdd,
  onRemove,
  onChange,
}: {
  klaim: Claim
  prefix: string
  items: ItemForm[]
  options: { nama: string; kelompok: string }[]
  currencyList: CurrencyOption[]
  onAdd: () => void
  onRemove: (n: number) => void
  onChange: (n: number, change: (item: ItemForm) => ItemForm) => void
}) {
  return (
    <table className="w-full border-collapse text-sm">
      <caption className="sr-only">Item</caption>
      <thead>
        <tr className="bg-slate-100 text-left text-xs text-slate-700">
          <th className="p-2" />
          <th className="p-2">Objek</th>
          <th className="p-2">Deskripsi Item</th>
          <th className="p-2">Unggah Dokumen</th>
          <th className="p-2">
            <button type="button" onClick={onAdd} className="rounded border border-blue-400 bg-white px-2 py-0.5 text-xs text-blue-700">
              + Tambah
            </button>
          </th>
        </tr>
      </thead>
      <tbody>
        {items.map((it, n) => {
          const id = `item-${prefix}-${n}`
          const set = (change: (item: ItemForm) => ItemForm) => onChange(n, change)
          return (
            <ItemRows key={n}>
              <tr className="bg-blue-100/70 align-top">
                <td className="p-2">{n + 1}</td>
                <td className="p-2">
                  {options.length > 0 ? (
                    <select
                      id={`${id}-objek`}
                      aria-label="Objek"
                      value={it.nama}
                      onChange={(e) => {
                        const pilihan = options.find((o) => o.nama === e.target.value)
                        set((x) => ({ ...x, nama: e.target.value, kelompok: pilihan?.kelompok ?? '' }))
                      }}
                      className="rounded border border-slate-400 bg-white px-1 py-0.5"
                    >
                      <option value="">— pilih —</option>
                      {options.map((o) => (
                        <option key={o.nama} value={o.nama}>
                          {o.nama}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <input
                      id={`${id}-objek`}
                      aria-label="Objek"
                      value={it.nama}
                      onChange={(e) => set((x) => ({ ...x, nama: e.target.value }))}
                      className="rounded border border-slate-400 bg-white px-1 py-0.5"
                    />
                  )}
                </td>
                <td className="p-2">
                  <textarea
                    aria-label="Deskripsi Item"
                    rows={3}
                    value={it.deskripsi}
                    onChange={(e) => set((x) => ({ ...x, deskripsi: e.target.value }))}
                    className="w-full rounded border border-slate-300 bg-white px-2 py-1"
                  />
                </td>
                <td className="p-2">
                  <button type="button" disabled title={NOT_BUILT} className="rounded bg-slate-400 px-2 py-1 text-xs text-white disabled:opacity-60">
                    Unggah Dokumen
                  </button>
                </td>
                <td className="p-2">
                  {it.tersimpan === 0 && (
                    <button
                      type="button"
                      onClick={() => onRemove(n)}
                      className="rounded border border-blue-400 bg-white px-2 py-0.5 text-xs text-blue-700"
                    >
                      Hapus
                    </button>
                  )}
                </td>
              </tr>
              <tr>
                <td />
                <td colSpan={4} className="p-2">
                  <EstimationTable klaim={klaim} prefix={id} item={it} currencyList={currencyList} set={set} />
                </td>
              </tr>
            </ItemRows>
          )
        })}
      </tbody>
    </table>
  )
}

function ItemRows({ children }: { children: ReactNode }) {
  return <>{children}</>
}

/** Baris estimasi satu item — kolom section `Estimasi`. */
function EstimationTable({
  klaim,
  prefix,
  item,
  currencyList,
  set,
}: {
  klaim: Claim
  prefix: string
  item: ItemForm
  currencyList: CurrencyOption[]
  set: (change: (item: ItemForm) => ItemForm) => void
}) {
  const currencyOptions = currencyList.map((m) => ({ value: m.id, label: m.nama }))
  // Estimasi berikutnya hanya boleh ditambahkan setelah estimasi terakhir dibuatkan CFS.
  const last = item.estimasi[item.estimasi.length - 1]
  const waitFaceSheet = last !== undefined && !last.terkunci
  if (klaim.polis.mata_uang && !currencyOptions.some((o) => o.value === klaim.polis.mata_uang)) {
    currencyOptions.unshift({ value: klaim.polis.mata_uang, label: klaim.polis.mata_uang })
  }

  return (
    <div>
      <table className="w-full border-collapse text-sm">
        <caption className="sr-only">Daftar estimasi</caption>
        <thead>
          <tr className="text-left text-xs text-slate-500">
            <th className="py-1 pr-2">Estimasi Ke</th>
            <th className="py-1 pr-2">Tanggal Estimasi</th>
            <th className="py-1 pr-2">Tipe Estimasi</th>
            <th className="py-1 pr-2">Mata Uang</th>
            <th className="py-1 pr-2">Nilai Estimasi</th>
            <th className="py-1 pr-2 text-right">Nilai Kurs (IDR)</th>
            <th className="py-1 pr-2 text-right">Nilai (IDR)</th>
            <th className="py-1" />
          </tr>
        </thead>
        <tbody>
          {item.estimasi.map((e, m) => {
            const patch = (value: Partial<EstimationForm>) =>
              set((x) => ({ ...x, estimasi: x.estimasi.map((row, r) => (r === m ? { ...row, ...value } : row)) }))
            return (
              <tr key={m}>
                <td className="py-1 pr-2">{m + 1}</td>
                <td className="py-1 pr-2">
                  <input
                    type="date"
                    aria-label="Tanggal Estimasi"
                    value={e.tanggal}
                    disabled={e.terkunci}
                    onChange={(ev) => patch({ tanggal: ev.target.value })}
                    className="w-36 rounded border border-slate-300 px-2 py-0.5"
                  />
                </td>
                <td className="py-1 pr-2">
                  <select
                    aria-label="Tipe Estimasi"
                    id={`${prefix}-${m}-tipe`}
                    value={e.tipe}
                    disabled={e.terkunci}
                    onChange={(ev) => patch({ tipe: ev.target.value })}
                    className="rounded border border-slate-300 px-1 py-0.5"
                  >
                    {TYPE_OPTIONS.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </td>
                <td className="py-1 pr-2">
                  <select
                    aria-label="Mata Uang"
                    value={e.mata_uang}
                    disabled={e.terkunci}
                    onChange={(ev) => patch({ mata_uang: ev.target.value })}
                    className="rounded border border-slate-300 px-1 py-0.5"
                  >
                    {currencyOptions.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </td>
                <td className="py-1 pr-2">
                  <input
                    aria-label="Nilai Estimasi"
                    inputMode="decimal"
                    value={e.nilai}
                    disabled={e.terkunci}
                    onChange={(ev) => patch({ nilai: ev.target.value })}
                    className="w-36 rounded border border-slate-300 px-2 py-0.5 text-right"
                  />
                </td>
                <td className="py-1 pr-2 text-right text-slate-600">
                  {e.kurs_e4 ? (e.kurs_e4 / 10_000).toLocaleString('id-ID') : '—'}
                </td>
                <td className="py-1 pr-2 text-right text-slate-600">{e.kurs_e4 ? formatRupiah(e.nilai_idr_sen) : '—'}</td>
                <td className="py-1">
                  {e.terkunci && <span className="text-xs text-slate-500">Sudah CFS</span>}
                  {m >= item.tersimpan && (
                    <button
                      type="button"
                      onClick={() => set((x) => ({ ...x, estimasi: x.estimasi.filter((_, r) => r !== m) }))}
                      className="text-xs text-red-700 underline"
                    >
                      Hapus estimasi
                    </button>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      <button
        type="button"
        disabled={waitFaceSheet}
        title={waitFaceSheet ? WAIT_FACE_SHEET : undefined}
        onClick={() =>
          set((x) => ({
            ...x,
            estimasi: [
              ...x.estimasi,
              { tipe: EstimationType.Claim, mata_uang: klaim.polis.mata_uang, tanggal: todayWIB(), nilai: '', kurs_e4: 0, nilai_idr_sen: 0, terkunci: false },
            ],
          }))
        }
        className="mt-1 rounded border border-slate-300 bg-white px-2 py-0.5 text-xs text-slate-700 hover:bg-slate-100 disabled:opacity-60"
      >
        Tambah estimasi
      </button>
    </div>
  )
}

export function errorText(failure: unknown): string {
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
