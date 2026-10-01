import { Fragment, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate, formatPercent } from '@/components/format'

import { useSession } from '@/app/session'

import { useCurrencies, violationsFrom } from './api'
import { AcceptanceButtons } from './AcceptanceButtons'
import { LODTypeSelect } from './LODTypeSelect'
import { CommitteeStatus, TransferCommitteeButton } from './Committee'
import { DocumentTab, ProgressTab, SurveyTab } from './EstimateTabs'
import { EstimatePaymentTable, errorText, useEstimateEditor } from './EstimateForm'
import { ReceiverTab } from './ReceiverTab'
import { SettlementDetail, SettlementEditor } from './SettlementEditor'
import {
  EstimationType,
  PaymentType,
  type Claim,
  type Coverage,
  type CurrencyOption,
  type Receiver,
  type Settlement,
  type Spreading,
  type Task,
} from './types'

/** Nilai Estimasi jaminan: jumlah estimasi klaim (bukan adjuster), dalam mata uangnya. */
function claimEstimate(c: Coverage): number {
  let total = 0
  for (const it of c.item ?? []) for (const e of it.estimasi) if (e.tipe !== EstimationType.Adjuster) total += e.nilai_sen
  return total
}

/**
 * Layar tahap **InputSurveyor** — flow action `InputSurveyor` di atas section
 * `Section/ClaimSurvey_sect.xml`. Dipakai tahap Choose Surveyor (Non-MBU, Assignment3) dan
 * Send To PIC Teknik (Travel, Assignment8), yang keduanya menerima klaim dari tombol
 * Kirim PIC Teknik pada Input Estimasi.
 *
 * # Yang ditampilkan, dan dari mana
 *
 * Baris atas: Status Klaim, lalu tombol menurut kondisinya di section itu —
 * Detail Premi · Detail Polis · Riwayat Klaim · Kirim ke Inputor (`!isAnalystPA_PNC`) ·
 * Kirim ke Marketing (`IsTravel`) · Kirim ke Admin (`IsNotTravelPA`) · Tutup Klaim
 * (`IsPendingClosed` false). Tombol berbasis peran analis (Kirim ke RCL/PUCL, Compliance,
 * Investigator, Inputor PA) dan Claim Inquiry (`IsKBGBRISurf`) tidak ditampilkan: data
 * peran analis dan penanda BRI Surf belum ada di layar ini.
 *
 * Tab untuk klaim yang belum ditutup sementara (`!IsPendingClose && !IsPNCReceive`):
 * Input Register (`InputRegisterDetail2`) · Estimasi & Adjustment (`InputEstimasi`) ·
 * Survey (`(isMarineCargo || isAneka || isFire)`, `TabSurvey`) · Unggah Dokumen ·
 * Progress Claim & Komunikasi.
 *
 * # Yang direkonstruksi
 *
 * Section tab Estimasi & Adjustment (`InputEstimasi`) TIDAK ada di export. Sub-tabnya —
 * Estimasi Pembayaran, Penerima Klaim, Adjustment & Akseptasi — mengikuti tangkapan layar
 * Pega dari Work Owner dan section tampilan yang ada (`ViewInputEstimasiDetail`,
 * `ViewShowReceiver`).
 *
 * Isian yang berjalan: sub-tab Estimasi Pembayaran (section InputEstimasiDetail, sama
 * dengan layar Input Estimasi — tambah dan simpan estimasi, Claim Face Sheet, Print PLA),
 * Tambah pada grid Adjustment (SettlementEditor), dan grid Penerima Klaim yang barisnya
 * dibuka menjadi panel InputReceiver (ReceiverTab). Tombol lain yang memproses klaim tampil
 * tetapi mati sampai prosesnya dibangun.
 */

const NOT_BUILT = 'Proses tombol ini belum dibangun.'

const TABS = ['Input Register', 'Estimasi & Adjustment', 'Survey', 'Unggah Dokumen', 'Progress Claim & Komunikasi'] as const
type Tab = (typeof TABS)[number]

const SUB_TABS = ['Estimasi Pembayaran', 'Penerima Klaim', 'Adjustment & Akseptasi'] as const
type SubTab = (typeof SUB_TABS)[number]

/** Group Panel (`.Policy.Quotation.GroupPanel`). */
const PANEL_PA = '002'
const PANEL_MARINE_CARGO = '004'
const PANEL_TRAVEL = '005'
const PANEL_FIRE = '006'

/** When IsNonMBU: Group Panel 003, 004, 006, 009. */
const NON_MBU_PANELS = ['003', PANEL_MARINE_CARGO, PANEL_FIRE, '009']

/** When IsAneka: `Quotation.BusinessCode = "10140"`. */
const BUSINESS_CODE_ANEKA = '10140'

const amountFormatter = new Intl.NumberFormat('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** Nilai uang tanpa lambang mata uang, seperti grid Pega: `1.000.000,00`. */
function formatAmount(sen: number): string {
  return amountFormatter.format(sen / 100)
}

function currencyName(code: string, list: CurrencyOption[]): string {
  return list.find((m) => m.id === code)?.nama ?? code
}

/** Tab Survey: `(isMarineCargo || isAneka || isFire)`. */
function surveyVisible(klaim: Claim): boolean {
  const panel = klaim.polis.lini
  const type = klaim.polis.jenis_bisnis
  const marineCargo = type === 'MarineCargo' || panel === PANEL_MARINE_CARGO
  const fire = type === 'Fire' || panel === PANEL_FIRE
  const aneka = (klaim.polis.kode_bisnis ?? '') === BUSINESS_CODE_ANEKA
  return marineCargo || fire || aneka
}

/**
 * Tugas InputSurveyor dirutekan PNCTeknikRouter ke PIC Teknik (beban paling ringan di
 * POOLDATA.MST_USER_TEKNIK). Ia boleh dikerjakan pemiliknya atau pemegang grup PIC Teknik
 * (PNCKomiteTeknik di M_LOGIN_GROUP_PNC); selebihnya server menolak (ErrNotTaskOwner) —
 * layar menyatakannya lebih dulu alih-alih membiarkan petugas mengisi lalu ditolak.
 */
function ownerNotice(tugas: Task, identity: string): string | null {
  // Server menilai kewenangan: pemilik, atau pemegang grup tahap (M_LOGIN_GROUP_PNC).
  if (tugas.dapat_dikerjakan === true) return null
  if (tugas.dapat_dikerjakan === undefined && (tugas.pemilik === '' || tugas.pemilik === identity)) return null
  return `Tugas ini milik ${tugas.pemilik}. Hanya pemilik tugas atau anggota grup PIC Teknik yang dapat menambah adjustment.`
}

export function SurveyorForm({ klaim, tugas }: { klaim: Claim; tugas: Task }) {
  const identity = useSession((state) => state.user?.identitas ?? '')
  const notice = ownerNotice(tugas, identity)
  const [tab, setTab] = useState<Tab>('Estimasi & Adjustment')
  const [subTab, setSubTab] = useState<SubTab>('Adjustment & Akseptasi')
  const currencies = useCurrencies()
  const currencyList = currencies.data?.pilihan ?? []

  const panel = klaim.polis.lini
  const travel = panel === PANEL_TRAVEL
  const notTravelPA = panel !== PANEL_TRAVEL && panel !== PANEL_PA
  const tabs = TABS.filter((t) => t !== 'Survey' || surveyVisible(klaim))

  const buttons: { label: string; strong?: boolean; visible: boolean }[] = [
    { label: 'Detail Premi', visible: true },
    { label: 'Detail Polis', visible: true },
    { label: 'Riwayat Klaim', visible: true },
    { label: 'Kirim ke Inputor', strong: true, visible: true },
    { label: 'Kirim ke Marketing', strong: true, visible: travel },
    { label: 'Kirim ke Admin', strong: true, visible: notTravelPA },
    { label: 'Tutup Klaim', visible: true },
  ]

  return (
    <section className="mt-6 rounded border border-slate-200 p-4" aria-label="InputSurveyor">
      <h2 className="text-sm text-slate-700">{tugas.tindakan_keluar || 'InputSurveyor'}</h2>
      {notice && (
        <p className="mt-2 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900" role="status">
          {notice} Layar ini dapat dibaca, tetapi isiannya tidak dapat disimpan.
        </p>
      )}

      {/* ── Bagian atas ClaimSurvey ─────────────────────────────────────────────── */}
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
        <div className="flex flex-wrap gap-2">
          {buttons
            .filter((b) => b.visible)
            .map((b) => (
              <button
                key={b.label}
                type="button"
                disabled
                title={NOT_BUILT}
                className={[
                  'rounded px-2 py-1 text-sm disabled:opacity-60',
                  b.strong ? 'bg-orange-500 text-white' : 'border border-blue-300 text-blue-700',
                ].join(' ')}
              >
                {b.label}
              </button>
            ))}
        </div>
      </div>

      <dl className="mt-4 grid gap-4 sm:grid-cols-3">
        <Field label="Catatan dari Inputor" note="Belum tersimpan: POOLDATA.T_CLAIM_PNC belum punya kolom untuk catatan ini." />
        <Field label="Status Pembayaran Premi" note="Diambil saat Detail Premi dibuka, yang belum dibangun." />
        <Field label="Aging Amount" note="Diambil saat Detail Premi dibuka, yang belum dibangun." />
      </dl>

      {/* ── Tab ─────────────────────────────────────────────────────────────────── */}
      <TabList items={tabs} current={tab} onSelect={setTab} label="Tab InputSurveyor" />

      {tab === 'Input Register' && <RegisterView klaim={klaim} />}
      {tab === 'Survey' && <SurveyTab claimID={klaim.id} />}
      {tab === 'Unggah Dokumen' && <DocumentTab claimID={klaim.id} />}
      {tab === 'Progress Claim & Komunikasi' && <ProgressTab claimID={klaim.id} />}

      {tab === 'Estimasi & Adjustment' && (
        <div>
          <TabList items={SUB_TABS} current={subTab} onSelect={setSubTab} label="Sub-tab Estimasi & Adjustment" />
          {subTab === 'Estimasi Pembayaran' && <EstimateView klaim={klaim} tugas={tugas} currencies={currencyList} lockedReason={notice} />}
          {subTab === 'Penerima Klaim' && <ReceiverTab klaim={klaim} tugas={tugas} lockedReason={notice} />}
          {subTab === 'Adjustment & Akseptasi' && <AdjustmentView klaim={klaim} tugas={tugas} identity={identity} currencies={currencyList} />}
        </div>
      )}
    </section>
  )
}

function Field({ label, note }: { label: string; note: string }) {
  return (
    <div>
      <dt className="text-sm font-semibold text-slate-800">{label}</dt>
      <dd className="text-sm text-slate-600" title={note}>
        —
      </dd>
    </div>
  )
}

function TabList<T extends string>({
  items,
  current,
  onSelect,
  label,
}: {
  items: readonly T[]
  current: T
  onSelect: (t: T) => void
  label: string
}) {
  return (
    <div role="tablist" aria-label={label} className="mt-5 flex flex-wrap gap-1 border-b border-slate-200">
      {items.map((t) => (
        <button
          key={t}
          type="button"
          role="tab"
          aria-selected={current === t}
          onClick={() => onSelect(t)}
          className={[
            'px-3 py-1.5 text-sm',
            current === t ? 'border-b-2 border-blue-600 font-semibold text-slate-900' : 'text-slate-600',
          ].join(' ')}
        >
          {t}
        </button>
      ))}
    </div>
  )
}

function Head({ columns }: { columns: string[] }) {
  return (
    <thead>
      <tr className="bg-slate-100 text-left text-xs text-slate-700">
        {columns.map((c, i) => (
          <th key={`${c}-${i}`} className="p-2">
            {c}
          </th>
        ))}
      </tr>
    </thead>
  )
}

function Empty({ span }: { span: number }) {
  return (
    <tr>
      <td colSpan={span} className="p-2 text-xs text-slate-500">
        Data Tidak Ada
      </td>
    </tr>
  )
}

function NoObjects({ klaim, children }: { klaim: Claim; children: ReactNode }) {
  if (klaim.objek.length === 0) {
    return <p className="mt-3 text-sm text-slate-600">Klaim ini belum punya objek.</p>
  }
  return <>{children}</>
}

// ── Tab Input Register — dibaca saja ───────────────────────────────────────────────

function RegisterView({ klaim }: { klaim: Claim }) {
  const rows: [string, string][] = [
    ['Nomor Polis', klaim.polis.nomor],
    ['Tertanggung', klaim.polis.nama_tertanggung || '—'],
    ['Lini Bisnis', `${klaim.polis.nama_lini} (${klaim.polis.lini})`],
    ['Tanggal Kejadian', formatDate(klaim.tanggal_kejadian)],
    ['Tanggal Lapor', formatDate(klaim.tanggal_lapor)],
    ['Tanggal Terima Dokumen', formatDate(klaim.tanggal_terima_dokumen)],
    ['Lokasi Kejadian', klaim.lokasi || '—'],
    ['Nama Pelapor', klaim.pelapor.nama || '—'],
    ['Telepon Pelapor', klaim.pelapor.telepon || '—'],
    ['PIC Teknis', klaim.user_teknis || '—'],
    ['Ex Gratia', klaim.ex_gratia ? 'YES' : 'NO'],
  ]
  return (
    <div className="mt-3">
      <p className="text-xs text-slate-500">
        Isian Input Register tampil untuk dibaca. Mengubahnya dari tahap ini belum dibangun.
      </p>
      <dl className="mt-3 grid gap-x-8 gap-y-3 sm:grid-cols-3">
        {rows.map(([label, value]) => (
          <div key={label}>
            <dt className="text-xs font-semibold text-slate-800">{label}</dt>
            <dd className="text-sm text-slate-700">{value}</dd>
          </div>
        ))}
      </dl>
      <div className="mt-3">
        <p className="text-xs font-semibold text-slate-800">Kronologi</p>
        <p className="whitespace-pre-line text-sm text-slate-700">{klaim.kronologi || '—'}</p>
      </div>

      <NoObjects klaim={klaim}>
        <table className="mt-4 w-full border-collapse text-sm">
          <caption className="sr-only">Objek, jaminan, dan spreading</caption>
          <Head columns={['', 'Nama Objek', 'Lokasi', 'Jaminan', 'TSI', 'Spreading']} />
          <tbody>
            {klaim.objek.map((o, i) =>
              o.coverage.map((c, j) => (
                <tr key={`${i}-${j}`} className="border-b border-slate-100 align-top">
                  <td className="p-2">{j === 0 ? i + 1 : ''}</td>
                  <td className="p-2">{j === 0 ? o.nama || o.id : ''}</td>
                  <td className="p-2">{j === 0 ? o.lokasi || '—' : ''}</td>
                  <td className="p-2">{c.nama || c.id}</td>
                  <td className="p-2 text-right">{formatAmount(c.tsi_sen)}</td>
                  <td className="p-2 text-xs">
                    {c.spreading
                      .filter((s) => !s.dihapus)
                      .map((s) => `${s.nama || s.jenis_treaty} ${formatPercent(s.share)}`)
                      .join(' · ') || '—'}
                  </td>
                </tr>
              )),
            )}
          </tbody>
        </table>
      </NoObjects>
    </div>
  )
}

// ── Sub-tab Estimasi Pembayaran — ViewInputEstimasiDetail ──────────────────────────

function EstimateView({
  klaim,
  tugas,
  currencies,
  lockedReason,
}: {
  klaim: Claim
  tugas: Task
  currencies: CurrencyOption[]
  lockedReason: string | null
}) {
  // Section InputEstimasiDetail yang sama dengan layar Input Estimasi — ClaimSurvey_sect
  // menanamnya, sehingga estimasi dapat ditambah dan disimpan di tahap ini tanpa memindahkan
  // tahap. Kirim PIC Teknik tidak ada di sini.
  const editor = useEstimateEditor(klaim, tugas)
  const { save, faceSheet } = editor
  const busy = save.isPending || faceSheet.isPending
  const failure = faceSheet.error ?? save.error
  const violations = violationsFrom(failure)

  return (
    <div>
      <EstimatePaymentTable klaim={klaim} tugas={tugas} editor={editor} currencyList={currencies} busy={busy || lockedReason !== null} />
      {failure && (
        <div className="mt-4">
          <ErrorMessage
            title={faceSheet.error ? 'Claim Face Sheet belum dapat dibuat' : 'Estimasi belum dapat disimpan'}
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
          Estimasi disimpan.
        </p>
      )}
      <div className="mt-4 flex justify-end">
        <Button tone="kedua" disabled={busy || lockedReason !== null} title={lockedReason ?? undefined} onClick={editor.saveNow}>
          {save.isPending ? 'Menyimpan…' : 'Save'}
        </Button>
      </div>
    </div>
  )
}
// ── Sub-tab Adjustment & Akseptasi ─────────────────────────────────────────────────

/** Adjustment sudah diakseptasi komite (STATUSAKSEPTASI 1). */
function accepted(s: Settlement): boolean {
  return s.status_akseptasi === '1'
}

/**
 * Nilai Akseptasi Klaim dan Adjuster objek: jumlah nilai akseptasi baris yang sudah
 * diakseptasi — fee adjuster (tipe 4) di kolom Adjuster, selebihnya di kolom Klaim. Baris
 * yang baru ditambahkan belum ditransfer ke komite, sehingga belum ikut terjumlah.
 */
function acceptedTotals(o: Claim['objek'][number]): { klaim: number; adjuster: number } {
  let klaim = 0
  let adjuster = 0
  for (const c of o.coverage) {
    for (const s of c.adjustment ?? []) {
      if (!accepted(s)) continue
      if (s.tipe_pembayaran === PaymentType.AdjusterFee) adjuster += s.nilai_akseptasi_sen
      else klaim += s.nilai_akseptasi_sen
    }
  }
  return { klaim, adjuster }
}

function AdjustmentView({
  klaim,
  tugas,
  identity,
  currencies,
}: {
  klaim: Claim
  tugas: Task
  identity: string
  currencies: CurrencyOption[]
}) {
  // Jaminan yang baris isian Tambah-nya sedang terbuka.
  const [addFor, setAddFor] = useState<{ i: number; j: number } | null>(null)
  const lockedReason = ownerNotice(tugas, identity)
  const policyCurrency = currencyName(klaim.polis.mata_uang, currencies)

  return (
    <NoObjects klaim={klaim}>
      <table className="mt-3 w-full border-collapse text-sm">
        <caption className="sr-only">Adjustment dan akseptasi</caption>
        <Head columns={['', 'Nama Objek', 'Lokasi', 'Currency', 'Nilai Akseptasi Klaim', 'Nilai Akseptasi Adjuster']} />
        <tbody>
          {klaim.objek.map((o, i) => {
            const totals = acceptedTotals(o)
            return (
              <Fragment key={i}>
                <tr className="border-b border-slate-200 align-top">
                  <td className="p-2">{i + 1}</td>
                  <td className="p-2">{o.nama || o.id}</td>
                  <td className="p-2">{o.lokasi || '—'}</td>
                  <td className="p-2">{policyCurrency}</td>
                  <td className="p-2 text-right">{formatAmount(totals.klaim)}</td>
                  <td className="p-2 text-right">{formatAmount(totals.adjuster)}</td>
                </tr>
                <tr>
                  <td colSpan={6} className="p-2">
                    <div className="rounded border border-slate-300 p-2">
                      <table className="w-full border-collapse text-sm">
                        <caption className="sr-only">Jaminan {o.nama}</caption>
                        <Head columns={['', 'Nama Coverage', 'Mata Uang', 'TSI']} />
                        <tbody>
                          {o.coverage.map((c, j) => (
                            <Fragment key={j}>
                              <tr className="border-b border-slate-100">
                                <td className="p-2">{j + 1}</td>
                                <td className="p-2">{c.nama || c.id}</td>
                                <td className="p-2">{policyCurrency}</td>
                                <td className="p-2 text-right">{formatAmount(c.tsi_sen)}</td>
                              </tr>
                              <tr>
                                <td colSpan={4} className="p-2">
                                  <SettlementGrid
                                    claimID={klaim.id}
                                    taskID={tugas.id}
                                    object={i + 1}
                                    coverage={j + 1}
                                    name={c.nama}
                                    lines={c.adjustment ?? []}
                                    currencies={currencies}
                                    spreading={c.spreading.filter((s) => !s.dihapus)}
                                    estimation={claimEstimate(c)}
                                    travel={klaim.polis.lini === PANEL_TRAVEL}
                                    nonMBU={NON_MBU_PANELS.includes(klaim.polis.lini)}
                                    groupPanel={klaim.polis.lini}
                                    businessType={klaim.polis.jenis_bisnis}
                                    receivers={klaim.penerima_klaim ?? []}
                                    onAdd={() => setAddFor({ i, j })}
                                    lockedReason={lockedReason}
                                    editor={
                                      addFor?.i === i && addFor.j === j ? (
                                        <SettlementEditor
                                          claimID={klaim.id}
                                          taskID={tugas.id}
                                          object={i + 1}
                                          coverage={j + 1}
                                          policyCurrency={klaim.polis.mata_uang}
                                          currencies={currencies}
                                          travel={klaim.polis.lini === PANEL_TRAVEL}
                                          nonMBU={NON_MBU_PANELS.includes(klaim.polis.lini)}
                                          onClose={() => setAddFor(null)}
                                        />
                                      ) : null
                                    }
                                  />
                                </td>
                              </tr>
                            </Fragment>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </td>
                </tr>
              </Fragment>
            )
          })}
        </tbody>
      </table>
    </NoObjects>
  )
}

function SettlementGrid({
  claimID,
  taskID,
  object,
  coverage,
  name,
  lines,
  currencies,
  spreading,
  estimation,
  travel,
  nonMBU,
  groupPanel,
  businessType,
  receivers,
  onAdd,
  editor,
  lockedReason,
}: {
  claimID: string
  taskID: string
  /** Objek dan jaminan berbasis 1 — alamat baris untuk Transfer Komite. */
  object: number
  coverage: number
  name: string
  lines: Settlement[]
  currencies: CurrencyOption[]
  spreading: Spreading[]
  estimation: number
  travel: boolean
  nonMBU: boolean
  /** Group Panel dan jenis bisnis polis — aturan tombol Print LOD. */
  groupPanel: string
  businessType: string
  /** Penerima klaim — pilihan Penerima Klaim form Persetujuan / Akseptasi. */
  receivers: Receiver[]
  onAdd: () => void
  /** Baris isian adjustment baru, bila tombol Tambah jaminan ini sedang dibuka. */
  editor: ReactNode
  /** Alasan tombol Tambah dikunci — tugas bukan milik pengguna ini. */
  lockedReason: string | null
}) {
  // Baris adjustment yang sedang dibuka untuk dilihat kembali.
  const [open, setOpen] = useState<number | null>(null)
  return (
    <table className="w-full border-collapse border border-slate-200 text-sm">
      <caption className="sr-only">Adjustment {name}</caption>
      <thead>
        <tr className="bg-slate-100 text-left text-xs text-slate-700">
          <th className="p-2">Adjustment</th>
          <th className="p-2">Transfer Komite</th>
          <th className="p-2">Tipe Pembayaran/ Akseptasi Komite</th>
          <th className="p-2">Akseptasi</th>
          <th className="p-2">
            <button
              type="button"
              onClick={onAdd}
              disabled={editor !== null || lockedReason !== null}
              title={lockedReason ?? undefined}
              className="rounded border border-blue-300 px-2 py-0.5 text-xs text-blue-700 hover:bg-blue-50 disabled:opacity-60"
            >
              Tambah
            </button>
          </th>
        </tr>
      </thead>
      <tbody>
        {lines.length === 0 && editor === null ? (
          <Empty span={5} />
        ) : (
          lines.map((s, n) => (
            <Fragment key={n}>
            <tr className="border-b border-slate-100 align-top">
              <td className="p-2">
                {/* `.PDFType` (ShowAdjustment_sect, sel pertama kolom Adjustment): dropdown Tipe LOD. */}
                <LODTypeSelect
                  claimID={claimID}
                  taskID={taskID}
                  object={object}
                  coverage={coverage}
                  adjustment={n + 1}
                  line={s}
                  groupPanel={groupPanel}
                  businessType={businessType}
                  lockedReason={lockedReason}
                />
                <button
                  type="button"
                  aria-expanded={open === n}
                  onClick={() => setOpen(open === n ? null : n)}
                  className="text-left hover:underline"
                >
                  <span className="block font-medium text-blue-800">
                    {open === n ? '▾' : '▸'} Adjustment {n + 1}
                  </span>
                  <span className="block text-xs text-slate-600">
                    {currencyName(s.mata_uang, currencies)} {formatAmount(s.nilai_asm_sen)}
                  </span>
                  <span className="block text-xs text-slate-500">Gross {formatAmount(s.nilai_gross_sen)}</span>
                </button>
              </td>
              <td className="p-2">
                <TransferCommitteeButton
                  claimID={claimID}
                  taskID={taskID}
                  object={object}
                  coverage={coverage}
                  adjustment={n + 1}
                  line={s}
                  lockedReason={lockedReason}
                />
              </td>
              <td className="p-2">{s.nama_tipe_pembayaran}</td>
              <td className="p-2 text-xs">
                <CommitteeStatus line={s} />
                <AcceptanceButtons
                  claimID={claimID}
                  taskID={taskID}
                  object={object}
                  coverage={coverage}
                  adjustment={n + 1}
                  line={s}
                  groupPanel={groupPanel}
                  businessType={businessType}
                  receivers={receivers}
                />
              </td>
              <td className="p-2" />
            </tr>
            {open === n && (
              <tr>
                <td colSpan={5} className="p-2">
                  <SettlementDetail
                    line={s}
                    currencyName={currencyName(s.mata_uang, currencies)}
                    estimation={estimation}
                    spreading={spreading}
                    travel={travel}
                    nonMBU={nonMBU}
                    address={{ claimID, taskID, object, coverage, adjustment: n + 1 }}
                  />
                </td>
              </tr>
            )}
            </Fragment>
          ))
        )}
        {editor !== null && (
          <tr>
            <td colSpan={5} className="p-2">
              {editor}
            </td>
          </tr>
        )}
      </tbody>
    </table>
  )
}
