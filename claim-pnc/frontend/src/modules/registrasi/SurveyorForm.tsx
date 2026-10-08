import { Fragment, useEffect, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate, formatPercent } from '@/components/format'

import { simpanBerkas } from '@/api/client'
import { useSession } from '@/app/session'

import { useCauseOfLossOptions, useClaimTask, useCompleteStage, useCurrencies, useFaceSheet, usePrepareSettlement, violationsFrom } from './api'
import { AcceptanceButtons } from './AcceptanceButtons'
import { LODTypeSelect } from './LODTypeSelect'
import { CommitteeStatus } from './Committee'
import { CommitteeTransferDialog } from './CommitteeTransferDialog'
import { DocumentTab, InvestigationTab, ProgressTab, SurveyTab } from './EstimateTabs'
import { EstimatePaymentTable, errorText, useEstimateEditor } from './EstimateForm'
import { ReceiverTab } from './ReceiverTab'
import { CloseClaimDialog } from './CloseClaim'
import { SendToInputorDialog } from './SendToInputor'
import { SendToRCLPUCLDialog } from './SendToRCLPUCL'
import { SettlementDetail, SettlementEditor } from './SettlementEditor'
import { isPHKCoverage, showTransferToAnalyst } from './TransferToAnalyst'
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
 * `Section/ClaimSurvey_sect.xml`. Dipakai tahap Choose Surveyor (Non-MBU, Assignment3),
 * Send To PIC Teknik (Travel, Assignment8), serta Estimation (Assignment4) dan Send To
 * Analis (Assignment5) untuk PA — keempatnya ditutup flow action InputSurveyor.
 *
 * Perbedaan PA (Group Panel 002) menurut section: Status Klaim tidak tampil (`!IsPA`), tab
 * Survey dan Kirim ke Admin tidak tampil, tab Investigasi tampil (isinya `TabInvestigasi`
 * tidak ada di export), dan Submit/Back memajukan tahap (StageSubmit).
 *
 * # Yang ditampilkan, dan dari mana
 *
 * Baris atas: Status Klaim, lalu tombol menurut kondisinya di section itu —
 * Detail Premi · Detail Polis · Riwayat Klaim · Kirim ke RCL/PUCL
 * (`isAnalystPA_PNC || isAnalisatorTravel`) · Kirim ke Inputor (`!isAnalystPA_PNC`) ·
 * Kirim ke Marketing (`IsTravel`) · Kirim ke Admin (`IsNotTravelPA`) · Tutup Klaim
 * (`IsPendingClosed` false). Tombol analis lain (Compliance, Investigator, Inputor PA) dan
 * Claim Inquiry (`IsKBGBRISurf`) belum ditampilkan: penanda BRI Surf belum ada di layar ini.
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

const TABS = ['Input Register', 'Estimasi & Adjustment', 'Survey', 'Investigasi', 'Unggah Dokumen', 'Progress Claim & Komunikasi'] as const
type Tab = (typeof TABS)[number]

/**
 * Sub-tab Estimasi & Adjustment — `Section/InputEstimasi-sect.xml`:
 *
 *   Estimasi Pembayaran       !isPA_PNC   InputEstimasiDetail
 *   Catatan Untuk Analyst     IsPA        CatatanToAnalyst_Section
 *   Catatan Untuk PIC Teknik  IsTravel    CatatanToAnalyst_Section
 *   Penerima Klaim            —           ShowReceiver (tidak di export; dari ViewShowReceiver)
 *   (pyLabel)                 IsHE        ShowObjectHEPICTeknis — tidak di export, tidak dibawa
 *   Adjustment & Akseptasi    —           ShowObjectAdj (tidak di export; dari ViewShowObjectAdj)
 */
const SUB_TABS = ['Estimasi Pembayaran', 'Catatan Untuk Analyst', 'Catatan Untuk PIC Teknik', 'Penerima Klaim', 'Adjustment & Akseptasi'] as const
type SubTab = (typeof SUB_TABS)[number]

function subTabVisible(t: SubTab, panel: string): boolean {
  if (t === 'Estimasi Pembayaran') return panel !== PANEL_PA
  if (t === 'Catatan Untuk Analyst') return panel === PANEL_PA
  if (t === 'Catatan Untuk PIC Teknik') return panel === PANEL_TRAVEL
  return true
}

/** Group Panel (`.Policy.Quotation.GroupPanel`). */
const PANEL_PA = '002'
const PANEL_MARINE_CARGO = '004'
const PANEL_TRAVEL = '005'
const PANEL_FIRE = '006'

/** When IsNonMBU: Group Panel 003, 004, 006, 009. */
const NON_MBU_PANELS = ['003', PANEL_MARINE_CARGO, PANEL_FIRE, '009']

/** Nilai Property AcceptanceStatus (dropdown section TransferKomite). Nilai lain tidak berlabel. */
const ACCEPTANCE_STATUS_LABEL: Record<string, string> = { '1': 'Akseptasi', '2': 'Belum Akseptasi' }

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
 * (PNCKomiteTeknik atau PncPICTeknik di M_LOGIN_GROUP_PNC); selebihnya server menolak (ErrNotTaskOwner) —
 * layar menyatakannya lebih dulu alih-alih membiarkan petugas mengisi lalu ditolak.
 */
/** Flow action tahap Investigator (Register_Flow Assignment11). */
const ACTION_INPUT_INVESTIGATOR = 'InputInvestigator'

/**
 * Tombol **Kirim ke RCL/PUCL** — `Section/ClaimSurvey_sect.xml` sel 6 (`pyStyleName` Strong),
 * membuka local action `KomentarRCLPUCL`. Syarat tampilnya, apa adanya dari section:
 *
 *	isAnalystPA_PNC || isAnalisatorTravel
 *
 * # Keduanya menguji KLAIM, bukan peran penekan tombol
 *
 * `When/isAnalystPA_PNC-when.xml` berlogika `A AND B`:
 *
 *	A  .Policy.Quotation.GroupPanel                    = "002"
 *	B  .ClaimData.PUCLStatus.IsKomiteTransfer_PNC      = "1"
 *
 * **`isAnalisatorTravel` tidak ada di export** (`R-16`) — ia hanya DIRUJUK di section ini.
 * Rujukannya tetap membawa satu keterangan yang menentukan, dan keterangan itu mengoreksi
 * bentuk pertama fungsi ini:
 *
 *	pxRuleClassName  ASM-FW-GCNMFW-Work-PNC
 *
 * Ia rule berkelas **objek kerja**, sehingga ia menguji DATA KLAIM. Bandingkan dengan
 * `IsAnalisator` yang berkelas `Data-Admin-Operator-ID` / `Data-Admin-WorkGroup` dan
 * memang menguji peran pemanggil. Bentuk pertama fungsi ini memakai `IsAnalisator`
 * (`tugas.analis`) untuk cabang Travel — itu jenis uji yang KELIRU: ia menyembunyikan
 * tombol dari siapa pun di luar grup Analyst, padahal kelas rule-nya menyatakan
 * perannya tidak ikut diuji.
 *
 * Yang dipakai sekarang adalah pasangan sebangun dari saudaranya: Travel menggantikan PA,
 * penanda yang sama. Nama keduanya pun sejajar — "analyst PA" dan "analisator Travel" —
 * dan keduanya di-OR dalam satu syarat. Bila rule aslinya kelak datang dan berbeda,
 * fungsi inilah yang disesuaikan.
 *
 * # Penanda B direkonstruksi, dan bedanya disadari
 *
 * Diambil dari `klaim.sudah_transfer_analis` (ANALYST_TRANSFERDATE) karena
 * `setTicketToAnalyst` mengisi keduanya pada langkah yang sama. Bedanya satu: Pega
 * MENGEMBALIKAN `IsKomiteTransfer_PNC` ke "0" saat adjustment ditransfer ke komite
 * (`KomitePost_Adjustment`, `SetListComiteeClaimPerObjAdj`), sedangkan
 * ANALYST_TRANSFERDATE tidak pernah dikosongkan — jadi pada klaim yang sudah masuk
 * komite, tombol ini masih tampil di sini padahal di Pega sudah hilang.
 */
export function showSendToRCLPUCL(klaim: Claim): boolean {
  const analystLine = klaim.polis.lini === PANEL_PA || klaim.polis.lini === PANEL_TRAVEL
  return analystLine && klaim.sudah_transfer_analis === true
}

export function ownerNotice(tugas: Task, identity: string, register: boolean): string | null {
  // Server menilai kewenangan: pemilik, atau pemegang grup tahap (M_LOGIN_GROUP_PNC).
  if (tugas.dapat_dikerjakan === true) return null
  if (tugas.dapat_dikerjakan === undefined && (tugas.pemilik === '' || tugas.pemilik === identity)) return null
  if (tugas.tindakan_keluar === ACTION_INPUT_INVESTIGATOR) {
    return `Tugas ini berada di antrean ${tugas.pemilik || 'Investigator'}. Hanya anggota grup Investigator yang dapat mengerjakannya.`
  }
  if (register) return `Tugas ini milik ${tugas.pemilik}. Hanya pemilik tugas atau anggota grup tahap ini yang dapat menyimpan isian Input Register.`
  return `Tugas ini milik ${tugas.pemilik}. Hanya pemilik tugas atau anggota grup PIC Teknik yang dapat menambah adjustment.`
}

/**
 * Bingkai ClaimSurvey_sect untuk tahap yang ditutup flow action InputSurveyor/InputInvestigator.
 * Tab Input Register di sini tampilan baca (RegisterView); tahap Input Register sendiri memakai
 * bingkai InputRegister-sect (InputRegisterFrame).
 */
export function SurveyorForm({ klaim, tugas }: { klaim: Claim; tugas: Task }) {
  const identity = useSession((state) => state.user?.identitas ?? '')
  const notice = ownerNotice(tugas, identity, false)
  // Tahap Investigator membuka tab Investigasi lebih dulu.
  const investigator = tugas.tindakan_keluar === ACTION_INPUT_INVESTIGATOR && klaim.polis.lini === PANEL_PA
  const [tab, setTab] = useState<Tab>(investigator ? 'Investigasi' : 'Estimasi & Adjustment')
  const [subTab, setSubTab] = useState<SubTab>('Adjustment & Akseptasi')
  const currencies = useCurrencies()
  const currencyList = currencies.data?.pilihan ?? []

  const panel = klaim.polis.lini
  const travel = panel === PANEL_TRAVEL
  const pa = panel === PANEL_PA
  const notTravelPA = panel !== PANEL_TRAVEL && panel !== PANEL_PA
  // Survey: (isMarineCargo || isAneka || isFire). Investigasi: GroupPanel = 002.
  const tabs = TABS.filter((t) => (t !== 'Survey' || surveyVisible(klaim)) && (t !== 'Investigasi' || pa))

  const [sendingToInputor, setSendingToInputor] = useState(false)
  const [sendingToRCLPUCL, setSendingToRCLPUCL] = useState(false)
  // Kirim ke RCL/PUCL: local action KomentarRCLPUCL. Tombolnya tampil menurut kondisi
  // section, tetapi baru dapat ditekan bila tugasnya memang dapat dikerjakan pemanggil.
  const canSendToRCLPUCL = notice === null && !tugas.dapat_diambil
  const [closing, setClosing] = useState(false)
  // Kirim ke Inputor: local action AnalystRemarks (ClaimSurvey_sect, `!isAnalystPA_PNC`).
  const canSendToInputor = notice === null && !tugas.dapat_diambil

  const buttons: { label: string; strong?: boolean; visible: boolean; onClick?: () => void }[] = [
    { label: 'Detail Premi', visible: true },
    { label: 'Detail Polis', visible: true },
    { label: 'Riwayat Klaim', visible: true },
    {
      label: 'Kirim ke RCL/PUCL',
      strong: true,
      visible: showSendToRCLPUCL(klaim),
      ...(canSendToRCLPUCL ? { onClick: () => setSendingToRCLPUCL(true) } : {}),
    },
    {
      label: 'Kirim ke Inputor',
      strong: true,
      visible: true,
      ...(canSendToInputor ? { onClick: () => setSendingToInputor(true) } : {}),
    },
    { label: 'Kirim ke Marketing', strong: true, visible: travel },
    { label: 'Kirim ke Admin', strong: true, visible: notTravelPA },
    // Tutup Klaim: local action PreventRejectClaim, tampil selama `.ClaimData.IsPendingClosed=='false'`.
    {
      label: 'Tutup Klaim',
      visible: klaim.tutup_sementara !== true,
      ...(canSendToInputor ? { onClick: () => setClosing(true) } : {}),
    },
  ]

  return (
    <section className="mt-6 rounded border border-slate-200 p-4" aria-label={tugas.tindakan_keluar || 'InputSurveyor'}>
      <h2 className="text-sm text-slate-700">{tugas.tindakan_keluar || 'InputSurveyor'}</h2>
      {tugas.dapat_diambil ? (
        <ClaimTaskNotice tugas={tugas} />
      ) : (
        notice && (
          <p className="mt-2 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900" role="status">
            {notice} Layar ini dapat dibaca, tetapi isiannya tidak dapat disimpan.
          </p>
        )
      )}

      {/* ── Bagian atas ClaimSurvey ─────────────────────────────────────────────── */}
      <div className="mt-3 flex flex-wrap items-end justify-between gap-3">
        {/* Status Klaim: layout `!IsTravel`, sel `!IsPA` — tidak tampil untuk Travel maupun PA. */}
        {!travel && !pa ? (
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
        ) : (
          <span />
        )}
        <div className="flex flex-wrap gap-2">
          {buttons
            .filter((b) => b.visible)
            .map((b) => (
              <button
                key={b.label}
                type="button"
                disabled={!b.onClick}
                title={b.onClick ? undefined : NOT_BUILT}
                onClick={b.onClick}
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

      {sendingToInputor && (
        <SendToInputorDialog claimID={klaim.id} taskID={tugas.id} onClose={() => setSendingToInputor(false)} />
      )}
      {closing && <CloseClaimDialog claimID={klaim.id} taskID={tugas.id} onClose={() => setClosing(false)} />}
      {klaim.tutup_sementara === true && (
        <p className="mt-3 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900" role="status">
          This claim is temporarily closed.
        </p>
      )}

      {sendingToRCLPUCL && (
        <SendToRCLPUCLDialog
          claimID={klaim.id}
          taskID={tugas.id}
          groupPanel={klaim.polis.lini}
          onClose={() => setSendingToRCLPUCL(false)}
        />
      )}

      <dl className="mt-4 grid gap-4 sm:grid-cols-3">
        <Field label="Catatan dari Inputor" note="Belum tersimpan: POOLDATA.T_CLAIM_PNC belum punya kolom untuk catatan ini." />
        <Field label="Status Pembayaran Premi" note="Diambil saat Detail Premi dibuka, yang belum dibangun." />
        <Field label="Aging Amount" note="Diambil saat Detail Premi dibuka, yang belum dibangun." />
      </dl>

      {/* ── Tab ─────────────────────────────────────────────────────────────────── */}
      <TabList items={tabs} current={tab} onSelect={setTab} label={`Tab ${tugas.tindakan_keluar || 'InputSurveyor'}`} />

      {tab === 'Input Register' && <RegisterView klaim={klaim} />}
      {tab === 'Survey' && <SurveyTab claimID={klaim.id} />}
      {tab === 'Investigasi' && <InvestigationTab claimID={klaim.id} />}
      {tab === 'Unggah Dokumen' && <DocumentTab claimID={klaim.id} line={klaim.polis.lini} />}
      {tab === 'Progress Claim & Komunikasi' && <ProgressTab claimID={klaim.id} />}

      {tab === 'Estimasi & Adjustment' && (
        <div>
          <TabList items={SUB_TABS.filter((t) => subTabVisible(t, panel))} current={subTab} onSelect={setSubTab} label="Sub-tab Estimasi & Adjustment" />
          {(subTab === 'Catatan Untuk Analyst' || subTab === 'Catatan Untuk PIC Teknik') && <AnalystNotes travel={travel} />}
          {subTab === 'Estimasi Pembayaran' && <EstimateView klaim={klaim} tugas={tugas} currencies={currencyList} lockedReason={notice} />}
          {subTab === 'Penerima Klaim' && <ReceiverTab klaim={klaim} tugas={tugas} lockedReason={notice} />}
          {subTab === 'Adjustment & Akseptasi' && <AdjustmentView klaim={klaim} tugas={tugas} identity={identity} currencies={currencyList} />}
        </div>
      )}

      {pa && <StageSubmit tugas={tugas} locked={notice !== null || tugas.dapat_diambil} />}
    </section>
  )
}

/**
 * Submit dan Back flow action InputSurveyor untuk tahap PA (Estimation, Send To Analis).
 *
 * Di Pega, tahap PA keluar lewat Submit flow action ini (Estimation → Investigator, Send To
 * Analis → keputusan RCLDokter/PUCL/Compliance). Tombol analis PA di baris atas
 * (Kirim ke Inputor PA, Kirim ke Investigator) bergantung When `isAnalystPA_PNC` yang tidak
 * ada di export, sehingga tanpa tombol ini klaim PA tidak dapat maju dari layar.
 */
function StageSubmit({ tugas, locked }: { tugas: Task; locked: boolean }) {
  const done = useCompleteStage()
  const go = (kembali: boolean) => done.mutate({ taskID: tugas.id, action: tugas.tindakan_keluar, kembali })
  return (
    <div className="mt-6 border-t border-slate-200 pt-4">
      {done.isError && (
        <p role="alert" className="mb-3 text-sm text-red-700">
          {done.error instanceof Error ? done.error.message : 'Tahap tidak dapat ditutup.'}
        </p>
      )}
      <div className="flex flex-wrap justify-between gap-3">
        <button
          type="button"
          disabled={done.isPending || locked}
          onClick={() => go(true)}
          className="rounded border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:opacity-60"
        >
          Back
        </button>
        <button
          type="button"
          disabled={done.isPending || locked}
          onClick={() => go(false)}
          className="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-60"
        >
          {done.isPending ? 'Memproses…' : 'Submit'}
        </button>
      </div>
    </div>
  )
}

/**
 * Tugas Workbasket (mis. Investigator, antrean InvestigatorPNC) lahir tanpa pemilik, dan server
 * menolak menutupnya sebelum diambil. Tombol Ambil di sini sama dengan tombol Ambil di inbox.
 */
function ClaimTaskNotice({ tugas }: { tugas: Task }) {
  const take = useClaimTask()
  return (
    <div className="mt-2 flex flex-wrap items-center justify-between gap-3 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900" role="status">
      <span>
        Tugas ini masih di antrean {tugas.antrean || 'bersama'}. Ambil tugas lebih dulu supaya tahap dapat ditutup.
        {take.isError && <span className="block text-red-700">{take.error instanceof Error ? take.error.message : 'Tugas tidak dapat diambil.'}</span>}
      </span>
      <button
        type="button"
        disabled={take.isPending}
        onClick={() => take.mutate(tugas.id)}
        className="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-60"
      >
        {take.isPending ? 'Mengambil…' : 'Ambil'}
      </button>
    </div>
  )
}

/**
 * `Section/CatatanToAnalyst_Section-Section.xml` — tiga grid baca dari page list work object
 * Pega (`ClaimData.ComplianceList`, `ClaimData.PUCLStatus.DateReceivedDocument`,
 * `ClaimData.AnalystDoctorList`). Ketiganya TIDAK tersimpan di tabel mana pun, dan nol dari 3.000
 * dokumen JSON_KLAIM terbaru memuatnya (2026-10-02); grid ditampilkan kosong sampai tahap
 * Compliance, PUCL, dan Analyst Doctor dibangun di aplikasi ini.
 */
function AnalystNotes({ travel }: { travel: boolean }) {
  const grids: [string, string][] = [
    ['Catatan Dari Compliance', 'ClaimData.ComplianceList'],
    ['Tanggal Terima Dokumen PUCL', 'ClaimData.PUCLStatus.DateReceivedDocument'],
    ['Komentar Dari Analyst Doctor', 'ClaimData.AnalystDoctorList'],
  ]
  return (
    <div className="mt-3 space-y-4" aria-label="Catatan Untuk Analyst">
      {grids.map(([title, source]) => (
        <table key={title} className="w-full border-collapse text-sm">
          <caption className="mb-1 text-left text-xs font-semibold text-slate-700">{title}</caption>
          <Head columns={['Tanggal', 'Komentar']} />
          <tbody>
            <tr>
              <td colSpan={2} className="p-2 text-xs text-slate-500" title={`Pega page list ${source}`}>
                Data Tidak Ada
              </td>
            </tr>
          </tbody>
        </table>
      ))}
      <dl className="grid gap-4 sm:grid-cols-2">
        <Field label="Pilihan Compliance" note="ClaimData.PilihanCompliance — set by the Compliance stage, not built yet." />
        {/* `IsPA && IsAnalisator || IsTravel`: IsAnalisator tidak ada di export, jadi hanya Travel. */}
        {travel && <Field label="Tanggal terima dokumen PUCL" note="ClaimData.PUCLStatus.TanggalTerimaDokumenPUCL — set by the PUCL stage, not built yet." />}
        <Field label="Komentar PUCL" note="ClaimData.PUCLStatus.KomentarPUCL — set by the PUCL stage, not built yet." />
        <Field label="Alasan Dokter Reject RCL" note="ClaimData.AlasanDokterRejectRCL — set by the RCL Dokter stage, not built yet." />
      </dl>
    </div>
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

export function TabList<T extends string>({
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

/** Tanggal lahir grid objek PA ditulis seperti Pega: dd/MM/yy (mis. 24/05/89). */
function shortDate(iso: string | undefined): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso ?? '')
  return m ? `${m[3]}/${m[2]}/${m[1]!.slice(2)}` : '—'
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
  // Modal "Transfer Claim ke Komite" (ClaimComitee_OC) yang sedang terbuka: jaminan berbasis 1, dan
  // baris adjustment bila dibuka dari tombol Transfer ke Komite grid Adjustment.
  const [committeeFor, setCommitteeFor] = useState<{ object: number; coverage: number; adjustment?: number } | null>(null)
  // Baris adjustment yang baru tersimpan pertama kali — dibuka di grid supaya isiannya dapat diteruskan.
  const [openFor, setOpenFor] = useState<{ i: number; j: number; n: number } | null>(null)
  // Tambah: ValidationAdjustment lebih dulu. PA pada jaminan tanpa adjustment mendapat estimasi
  // NewEstimationPA (TSI) sehingga Claim Face Sheet dapat dibuat.
  const prepare = usePrepareSettlement(klaim.id)
  const [prepareFor, setPrepareFor] = useState<{ i: number; j: number } | null>(null)
  const startAdd = (i: number, j: number) => {
    if (!pa) {
      setAddFor({ i, j })
      return
    }
    prepare.reset()
    setPrepareFor({ i, j })
    prepare.mutate({ tugas_id: tugas.id, objek: i + 1, jaminan: j + 1 }, { onSuccess: () => setAddFor({ i, j }) })
  }
  const lockedReason = ownerNotice(tugas, identity, false)
  // Download Claim Face Sheet pada baris jaminan PA.
  const faceSheet = useFaceSheet(klaim.id)
  const faceSheetViolations = violationsFrom(faceSheet.error)
  // Jaminan yang tombol Download Claim Face Sheet-nya terakhir diklik — pesannya tampil di bawah baris itu.
  const [faceSheetFor, setFaceSheetFor] = useState<{ i: number; j: number } | null>(null)
  const policyCurrency = currencyName(klaim.polis.mata_uang, currencies)
  // Grid objek ShowObjectAdj per lini: IsPA — Nama Objek, Pekerjaan, Tanggal Lahir, Currency, Nilai
  // Estimasi (.NilaiOSKalim), Nilai Akseptasi Klaim; selain itu (!IsTravelPA && !IsHE) — Nama Objek,
  // Lokasi, Currency, Nilai Akseptasi Klaim, Nilai Akseptasi Adjuster.
  const pa = klaim.polis.lini === PANEL_PA
  // Baris coverage ObjectCoverageAdj: IsPA menambah Penyebab Kerugian (nama dari master sebab kerugian).
  const causes = useCauseOfLossOptions(pa ? klaim.polis.kode_bisnis ?? '' : '')
  const causeName = (id: string) => causes.data?.pilihan.find((o) => o.id === id)?.nama ?? id
  // Baris coverage ObjectCoverageAdj kontainer IsPA, sel demi sel: CoverageNote · CauseOfLoss · Currency · SumTSI ·
  // Download Claim Face Sheet · Print PLA · include TrfKomiteButton · (kosong). Judul kolomnya apa adanya di XML —
  // termasuk "TSI" kedua di atas Print PLA — dan sel judul terakhir memuat tombol Tambah Jaminan.
  const coverageColumns = pa
    ? ['', 'Nama Coverage', 'Penyebab Kerugian', 'Mata Uang', 'TSI', '', 'TSI', '', '']
    : ['', 'Nama Coverage', 'Mata Uang', 'TSI']
  const columns = pa
    ? ['', 'Nama Objek', 'Pekerjaan', 'Tanggal Lahir', 'Currency', 'Nilai Estimasi', 'Nilai Akseptasi Klaim']
    : ['', 'Nama Objek', 'Lokasi', 'Currency', 'Nilai Akseptasi Klaim', 'Nilai Akseptasi Adjuster']

  return (
    <NoObjects klaim={klaim}>
      <table className="mt-3 w-full border-collapse text-sm">
        <caption className="sr-only">Adjustment dan akseptasi</caption>
        <Head columns={columns} />
        <tbody>
          {klaim.objek.map((o, i) => {
            const totals = acceptedTotals(o)
            return (
              <Fragment key={i}>
                <tr className="border-b border-slate-200 align-top">
                  <td className="p-2">{i + 1}</td>
                  <td className="p-2">{o.nama || o.id}</td>
                  {pa ? (
                    <>
                      <td className="p-2">{o.pekerjaan || '—'}</td>
                      <td className="p-2">{shortDate(o.tanggal_lahir)}</td>
                      <td className="p-2">{policyCurrency}</td>
                      <td className="p-2 text-right">{formatAmount(o.coverage.reduce((sum, c) => sum + claimEstimate(c), 0))}</td>
                      <td className="p-2 text-right">{formatAmount(totals.klaim)}</td>
                    </>
                  ) : (
                    <>
                      <td className="p-2">{o.lokasi || '—'}</td>
                      <td className="p-2">{policyCurrency}</td>
                      <td className="p-2 text-right">{formatAmount(totals.klaim)}</td>
                      <td className="p-2 text-right">{formatAmount(totals.adjuster)}</td>
                    </>
                  )}
                </tr>
                <tr>
                  <td colSpan={columns.length} className="p-2">
                    <div className="rounded border border-slate-300 p-2">
                      <table className="w-full border-collapse text-sm">
                        <caption className="sr-only">Jaminan {o.nama}</caption>
                        {pa ? (
                          <thead>
                            <tr className="bg-slate-100 text-left text-xs text-slate-700">
                              {coverageColumns.slice(0, -1).map((h, k) => (
                                <th key={k} className={['p-2', h === 'TSI' ? 'text-right' : ''].join(' ')}>
                                  {h}
                                </th>
                              ))}
                              <th className="p-2 text-right">
                                {/* Tambah Jaminan: addRow + ShowCoverage(StatusPA="add") — belum dibangun. */}
                                <button
                                  type="button"
                                  disabled
                                  title={NOT_BUILT}
                                  className="rounded border border-blue-300 px-2 py-1 text-xs text-blue-700 disabled:opacity-60"
                                >
                                  Tambah Jaminan
                                </button>
                              </th>
                            </tr>
                          </thead>
                        ) : (
                          <Head columns={coverageColumns} />
                        )}
                        <tbody>
                          {o.coverage.map((c, j) => (
                            <Fragment key={j}>
                              <tr className="border-b border-slate-100">
                                <td className="p-2">{j + 1}</td>
                                <td className="p-2">{c.nama || c.id}</td>
                                {pa && <td className="p-2">{c.penyebab_kerugian ? causeName(c.penyebab_kerugian) : '—'}</td>}
                                <td className="p-2">{policyCurrency}</td>
                                <td className="p-2 text-right">{formatAmount(c.tsi_sen)}</td>
                                {pa && (
                                  <>
                                    <td className="p-2">
                                      {/* Download Claim Face Sheet: `IsAnalisator || IsPHK` → DownloadClaimFaceSheet_act (rute CFS). */}
                                      {(tugas.analis === true || isPHKCoverage(c.id)) && (
                                        <button
                                          type="button"
                                          disabled={lockedReason !== null || faceSheet.isPending}
                                          title={lockedReason ?? undefined}
                                          onClick={() => {
                                            faceSheet.reset()
                                            setFaceSheetFor({ i, j })
                                            faceSheet.mutate({ tugas_id: tugas.id, objek: i + 1, jaminan: j + 1 }, { onSuccess: (file) => simpanBerkas(file) })
                                          }}
                                          className="whitespace-nowrap rounded border border-blue-300 px-2 py-1 text-xs text-blue-700 disabled:opacity-60"
                                        >
                                          {faceSheet.isPending && faceSheetFor?.i === i && faceSheetFor.j === j ? 'Mengunduh…' : 'Download Claim Face Sheet'}
                                        </button>
                                      )}
                                    </td>
                                    <td className="p-2">
                                      {/* Print PLA: local action PrintPLA_PAPHK, nonaktif bila `IsNoCoins || !isCFS` — belum dibangun. */}
                                      <button
                                        type="button"
                                        disabled
                                        title={NOT_BUILT}
                                        className="rounded bg-slate-300 px-2 py-1 text-xs text-white disabled:opacity-80"
                                      >
                                        Print PLA
                                      </button>
                                    </td>
                                    <td className="p-2">
                                      {showTransferToAnalyst(klaim, tugas.tahap, o, j) && (
                                        <button
                                          type="button"
                                          disabled={lockedReason !== null}
                                          title={lockedReason ?? undefined}
                                          onClick={() => setCommitteeFor({ object: i + 1, coverage: j + 1 })}
                                          className="whitespace-nowrap rounded bg-orange-500 px-2 py-1 text-xs text-white disabled:opacity-60"
                                        >
                                          Transfer ke Analyst
                                        </button>
                                      )}
                                    </td>
                                    <td className="p-2" />
                                  </>
                                )}
                              </tr>
                              {prepare.isError && prepareFor?.i === i && prepareFor.j === j && (
                                <tr>
                                  <td colSpan={coverageColumns.length} className="p-2">
                                    <ErrorMessage
                                      tone="penolakan"
                                      title="Adjustment belum dapat ditambahkan"
                                      description={prepare.error instanceof Error ? prepare.error.message : 'Terjadi kesalahan.'}
                                    />
                                  </td>
                                </tr>
                              )}
                              {faceSheet.isError && faceSheetFor?.i === i && faceSheetFor.j === j && (
                                <tr>
                                  <td colSpan={coverageColumns.length} className="p-2">
                                    <ErrorMessage
                                      tone="penolakan"
                                      title="Claim Face Sheet belum dapat diunduh"
                                      description={
                                        faceSheetViolations.length > 0
                                          ? faceSheetViolations.map((v) => v.pesan).join(' ')
                                          : faceSheet.error instanceof Error
                                            ? faceSheet.error.message
                                            : 'Terjadi kesalahan.'
                                      }
                                    />
                                  </td>
                                </tr>
                              )}
                              <tr>
                                <td colSpan={coverageColumns.length} className="p-2">
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
                                    exGratia={klaim.ex_gratia}
                                    onTransferCommittee={(n) => setCommitteeFor({ object: i + 1, coverage: j + 1, adjustment: n })}
                                    autoOpen={openFor?.i === i && openFor.j === j ? openFor.n : null}
                                    editable={(n, s) =>
                                      lockedReason === null && s.status_akseptasi === '' && !s.komite_id && !s.sudah_transfer_kasir ? (
                                        <SettlementEditor
                                          key={n}
                                          claimID={klaim.id}
                                          taskID={tugas.id}
                                          object={i + 1}
                                          coverage={j + 1}
                                          policyCurrency={klaim.polis.mata_uang}
                                          currencies={currencies}
                                          travel={klaim.polis.lini === PANEL_TRAVEL}
                                          nonMBU={NON_MBU_PANELS.includes(klaim.polis.lini)}
                                          pa={pa}
                                          exGratia={klaim.ex_gratia}
                                          analyst={tugas.analis === true}
                                          analystTransfer={pa && (isPHKCoverage(c.id) || c.sudah_transfer_analis === true)}
                                          existing={{ index: n, line: s }}
                                          onClose={() => {}}
                                        />
                                      ) : null
                                    }
                                    onAdd={() => startAdd(i, j)}
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
                                          pa={pa}
                                          exGratia={klaim.ex_gratia}
                                          analyst={tugas.analis === true}
                                          analystTransfer={pa && (isPHKCoverage(c.id) || c.sudah_transfer_analis === true)}
                                          onCreated={(n) => {
                                            setAddFor(null)
                                            setOpenFor({ i, j, n })
                                          }}
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
      {committeeFor && (
        <CommitteeTransferDialog
          klaim={klaim}
          taskID={tugas.id}
          stage={tugas.tahap}
          object={committeeFor.object}
          coverage={committeeFor.coverage}
          adjustment={committeeFor.adjustment}
          receivers={klaim.penerima_klaim ?? []}
          analyst={tugas.analis === true}
          onClose={() => setCommitteeFor(null)}
        />
      )}
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
  exGratia,
  onTransferCommittee,
  autoOpen = null,
  editable,
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
  /** Klaim Ex Gratia (`pyWorkPage.ClaimData.ExGratia`). */
  exGratia: boolean
  /** Tombol "Transfer ke Komite" baris adjustment (berbasis 1) — membuka modal ClaimComitee_OC. */
  onTransferCommittee: (adjustment: number) => void
  /** Baris (berbasis 0) yang dibuka otomatis — baris yang baru tersimpan pertama kali. */
  autoOpen?: number | null
  /** Form isian untuk baris yang masih dapat diubah (`.AcceptanceStatus == ''`), atau null. */
  editable?: (n: number, line: Settlement) => ReactNode | null
  onAdd: () => void
  /** Baris isian adjustment baru, bila tombol Tambah jaminan ini sedang dibuka. */
  editor: ReactNode
  /** Alasan tombol Tambah dikunci — tugas bukan milik pengguna ini. */
  lockedReason: string | null
}) {
  // Baris adjustment yang sedang dibuka untuk dilihat kembali.
  const [open, setOpen] = useState<number | null>(null)
  useEffect(() => {
    if (autoOpen !== null) setOpen(autoOpen)
  }, [autoOpen])
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
                {/* ShowAdjustment sel 28: `.IsKomiteTransfer=='' && .AcceptanceStatus=='' && IsNonMBU`,
                    membuka modal ClaimComitee_OC. Sel 29 (section TransferKomite) tidak ada di export. */}
                {!s.komite_id && s.status_akseptasi === '' && nonMBU ? (
                  <button
                    type="button"
                    disabled={lockedReason !== null}
                    title={lockedReason ?? undefined}
                    onClick={() => onTransferCommittee(n + 1)}
                    className="whitespace-nowrap rounded bg-orange-500 px-2 py-0.5 text-xs text-white disabled:opacity-60"
                  >
                    Transfer ke Komite
                  </button>
                ) : s.komite_id ? (
                  <span className="text-xs text-slate-600">Sudah ditransfer</span>
                ) : null}
              </td>
              <td className="p-2">
                {/* Section TransferKomite: `.PaymentType` · " / " bila `.AcceptanceStatus != ''` ·
                    `.AcceptanceStatus` (Property AcceptanceStatus: 1 Akseptasi, 2 Belum Akseptasi). */}
                {s.nama_tipe_pembayaran}
                {s.status_akseptasi !== '' && <> / {ACCEPTANCE_STATUS_LABEL[s.status_akseptasi] ?? ''}</>}
              </td>
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
                  {editable?.(n, s) ?? (
                  <SettlementDetail
                    line={s}
                    currencyName={currencyName(s.mata_uang, currencies)}
                    estimation={estimation}
                    spreading={spreading}
                    travel={travel}
                    nonMBU={nonMBU}
                    pa={groupPanel === PANEL_PA}
                    exGratia={exGratia}
                    address={{ claimID, taskID, object, coverage, adjustment: n + 1 }}
                  />
                  )}
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
