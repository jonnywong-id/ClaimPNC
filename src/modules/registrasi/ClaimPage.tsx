import { forwardRef, useEffect, useState, type InputHTMLAttributes, type ReactNode } from 'react'
import {
  Controller,
  useFieldArray,
  useForm,
  useWatch,
  type Control,
  type UseFormRegister,
  type UseFormSetValue,
  type UseFormWatch,
} from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DateField } from '@/components/DateField'
import { Paginator } from '@/components/DataTable'
import { FormField } from '@/components/FormField'
import { SelectField } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatPercent, formatRupiah, formatDate, rupiahToCents, centsToRupiah, todayWIB } from '@/components/format'

import {
  useFlow,
  useClaim,
  useCompleteStage,
  useSaveRegister,
  useSaveDraft,
  useAreaOptions,
  useCauseOfLossOptions,
  useCoverageOptions,
  useCurrencies,
  useInsuredProfile,
  violationsFrom,
  messagesByField,
} from './api'
import { EstimateForm } from './EstimateForm'
import { StagePath } from './StagePath'
import { SurveyorForm } from './SurveyorForm'
import { InputRegisterFrame } from './InputRegisterFrame'
import { AnalystNoteNotice } from './SendToInputor'
import { InsuredDataSection } from './InsuredData'
import {
  AreaLevel,
  COUNTRY_INDONESIA,
  CustomerPrinciple,
  type Area,
  type AreaOption,
  type CauseOfLossOption,
  type Claim,
  type InsuredItem,
  type Violation,
  type RegisterRequest,
  type Task,
} from './types'

/** Pengenal tahap Input Register, satu-satunya tahap yang isiannya dimiliki modul ini. */
const TAHAP_INPUT_REGISTER = 'input-register'

/** Group Panel (`.Policy.Quotation.GroupPanel`) — penentu kondisi tampil InputRegisterDetail2_sect. */
const PANEL_PA = '002'
const PANEL_TRAVEL = '005'

/**
 * Pilihan Jenis Laporan (`.ClaimData.ReportType`, `pxDropdown` di InputRegisterDetail-sect).
 * Sumber pilihannya "associated" — rule property `ReportType` yang tidak ada di export —
 * sehingga kode dan labelnya diambil dari Data Transform `ReportTypetoText`. Bawaannya `1`
 * (`pyDefaultValue` di section). Pega menyimpannya di T_CLAIM_PNC.REPORTTYPE.
 */
const REPORT_TYPE_DIRECT = '1'
const REPORT_TYPES = [
  { value: '1', label: 'Direct' },
  { value: '2', label: 'Via Email' },
  { value: '3', label: 'Via Fax' },
  { value: '4', label: 'Via Pos / Kurir' },
  { value: '5', label: 'Via Telephone' },
  { value: '6', label: 'Via Portal' },
]

/** Pilihan Jenis Laporan, ditambah kode tersimpan yang tidak ada di daftar. */
function reportTypeOptions(current: string | undefined) {
  return withStoredCode(REPORT_TYPES, current)
}

/**
 * Daftar pilihan ditambah kode tersimpan yang tidak ada di daftar (mis. Status Pelapor 0 pada
 * klaim lama), supaya nilainya tidak diam-diam berubah saat form dibuka.
 */
function withStoredCode(options: { value: string; label: string }[], current: string | undefined) {
  const code = (current ?? '').trim()
  if (code === '' || options.some((t) => t.value === code)) return options
  return [...options, { value: code, label: code }]
}

/** Pesan isian Lokasi kosong — sama dengan pesan server (ViolationLocationEmpty). */
const LOCATION_REQUIRED = 'Lokasi Kerugian/Kejadian is required.'

/** Kode IDR di master POOLDATA.CURRENCY. */
const CURRENCY_IDR = '10026'

/**
 * ObjectGrid adalah empat grid `.ClaimData.ObjectList` di Section/InputRegisterDetail-sect.xml,
 * dipilih menurut kondisi tampil kontainernya: `IsTravel` (GroupPanel 005), `IsPA`
 * (GroupPanel 002), `IsHE` (BusinessType "HE" atau "ContractorsPM" — `When/IsHE-When.xml`, logika
 * `A OR B`), dan `!IsTravelPA && !IsHE` untuk lini lain.
 */
type ObjectGrid = 'travel' | 'pa' | 'he' | 'umum'

/**
 * Judul kolom tiap grid, urut seperti di XML. Seluruhnya read-only di XML (`pyEditOptions
 * Read-only`) kecuali dua yang sengaja tidak dibawa:
 *
 * - "Type Object" (`.TypeObjectKlaim`) di grid umum dan HE: kondisi tampilnya `1==2`, jadi
 *   tidak pernah tampil di Pega.
 * - "Pilih Peserta" (`.Selected`, kotak centang) di grid Travel: satu-satunya isian yang
 *   dapat diubah, tetapi T_CLAIM_OBJECTLIST tidak punya kolom untuk menyimpannya.
 *
 * Ejaan "Pekerjaan" dibetulkan dari "Perkerjaan" Pega — lihat catatan grid di bawah.
 */
const OBJECT_COLUMNS: Record<ObjectGrid, string[]> = {
  travel: ['Nama Peserta', 'Status', 'KTP/Paspor', 'Tanggal Lahir'],
  pa: ['Nama', 'Pekerjaan', 'Tanggal Lahir'],
  he: ['Object', 'Model', 'Merk', 'Nama Tipe', 'Nomor Chasis', 'Location'],
  umum: ['Object', 'Location'],
}

/** BusinessType yang memenuhi When `IsHE`: "HE" OR "ContractorsPM" (dibandingkan tanpa beda huruf). */
const HE_BUSINESS_TYPES = ['HE', 'CONTRACTORSPM']

function objectGrid(panel: string, businessType: string): ObjectGrid {
  if (panel === PANEL_TRAVEL) return 'travel'
  if (panel === PANEL_PA) return 'pa'
  if (HE_BUSINESS_TYPES.includes(businessType.trim().toUpperCase())) return 'he'
  return 'umum'
}

/** nextObjectID memberi kode objek baru: satu di atas kode angka terbesar yang dipakai. */
function nextObjectID(ids: string[]): string {
  const max = ids.reduce((m, id) => (/^\d+$/.test(id.trim()) ? Math.max(m, Number(id.trim())) : m), 0)
  return String(max + 1)
}

/** Tahap Input Estimasi (Non-MBU dan Travel), yang isiannya dimiliki EstimateForm. */
const TAHAP_INPUT_ESTIMASI = ['estimasi-admin', 'estimasi-travel']

/**
 * Tahap yang ditutup flow action InputSurveyor — layar SurveyorForm:
 *
 *   pilih-surveyor    Choose Surveyor (Non-MBU, Assignment3)
 *   kirim-pic-teknik  Send To PIC Teknik (Travel, Assignment8)
 *   estimasi-pa       Estimation (PA, Assignment4)
 *   kirim-analis      Send To Analis (PA, Assignment5)
 *
 * Ditambah tahap Investigator (PA, Assignment11, workbasket InvestigatorPNC) yang ditutup flow
 * action InputInvestigator. Section flow action itu (`InputInvestigator`) tidak ada di export,
 * sehingga ia memakai bingkai ClaimSurvey_sect yang sama dan keluar lewat Submit/Back:
 * Register_Flow Estimation → Investigator → Send To Analis.
 */
const TAHAP_INPUT_SURVEYOR = ['pilih-surveyor', 'kirim-pic-teknik', 'estimasi-pa', 'investigator', 'kirim-analis']

/**
 * Layar kerja satu klaim.
 *
 * Ia menampilkan di mana klaim berada, apa yang menantinya, dan tindakan yang tersedia
 * pada tahap sekarang. Hanya satu tahap yang punya formulir di sini — Input Register.
 * Tahap lain ditutup dengan satu tindakan, karena isiannya milik modul lain (`B-5`,
 * `B-8`, `B-11`) yang belum dibangun.
 */
export function ClaimPage() {
  const { claimID } = useParams<{ claimID: string }>()
  const klaim = useClaim(claimID)
  const flow = useFlow()

  if (klaim.isPending) {
    return <Frame><p className="text-sm text-slate-500">Memuat klaim…</p></Frame>
  }

  if (klaim.isError) {
    return (
      <Frame>
        <ErrorMessage title="Klaim tidak dapat dimuat" description={errorMessage(klaim.error)} tone="gangguan" />
      </Frame>
    )
  }

  const content = klaim.data
  const atInputRegister = content.klaim.tahap_kini === TAHAP_INPUT_REGISTER
  const atInputEstimate = TAHAP_INPUT_ESTIMASI.includes(content.klaim.tahap_kini)
  const atInputSurveyor = TAHAP_INPUT_SURVEYOR.includes(content.klaim.tahap_kini)

  return (
    <Frame>
      <ClaimHeader klaim={content.klaim} />

      {flow.data && content.jalur && content.jalur.length > 0 && (
        <div className="mt-6">
          <StagePath tahap={flow.data.tahap} jalur={content.jalur} currentStage={content.klaim.tahap_kini} />
        </div>
      )}

      {content.klaim.tahap_kini === '' && (
        <p className="mt-6 rounded border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700">
          Klaim ini sudah selesai pada alur Register. Tidak ada tugas yang menunggu.
        </p>
      )}

      {content.tugas && !atInputRegister && !atInputEstimate && !atInputSurveyor && (
        <StageActions tugas={content.tugas} />
      )}
      {content.tugas && atInputSurveyor && (
        <SurveyorForm key={content.tugas.id} klaim={content.klaim} tugas={content.tugas} />
      )}
      {content.tugas && atInputEstimate && (
        <EstimateForm key={content.tugas.id} klaim={content.klaim} tugas={content.tugas} />
      )}
      {content.tugas && atInputRegister && (
        <InputRegisterFrame
          key={content.tugas.id}
          klaim={content.klaim}
          tugas={content.tugas}
          register={<FormRegister klaim={content.klaim} tugas={content.tugas} />}
        />
      )}

      {!content.tugas && content.klaim.tahap_kini !== '' && (
        <p className="mt-6 rounded border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          Klaim berada di tahap {content.klaim.tahap_kini}, tetapi tidak ada tugas terbuka yang
          dapat Anda kerjakan. Tugasnya mungkin milik rekan kerja, atau berada di antrean
          bersama yang belum Anda ambil.
        </p>
      )}
    </Frame>
  )
}

function Frame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <Link to="/registrasi" className="text-sm text-slate-600 underline">
        ← Kembali ke inbox
      </Link>
      <div className="mt-4">{children}</div>
    </div>
  )
}

function ClaimHeader({ klaim }: { klaim: Claim }) {
  return (
    <header className="border-b border-slate-200 pb-4">
      <h1 className="text-xl font-semibold text-slate-900">
        {klaim.nomor || 'Klaim belum bernomor'}
      </h1>
      <dl className="mt-3 grid gap-x-8 gap-y-3 sm:grid-cols-3">
        <Row label="Nomor polis" value={klaim.polis.nomor} />
        <Row label="Lini bisnis" value={`${klaim.polis.nama_lini} (${klaim.polis.lini})`} />
        <Row label="Tertanggung" value={klaim.polis.nama_tertanggung || '—'} />
        <Row
          label="Periode polis"
          value={`${formatDate(klaim.polis.mulai_pertanggungan)} – ${formatDate(klaim.polis.akhir_pertanggungan)}`}
        />
        <Row label="Status Proses" value={klaim.status_proses} />
        <Row label="Status Klaim" value={klaim.status_klaim || '—'} />
      </dl>
    </header>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="text-sm text-slate-900">{value}</dd>
    </div>
  )
}

/**
 * Tindakan penutup untuk tahap yang isiannya milik modul lain.
 *
 * Nama tindakannya datang dari SERVER — ia adalah `tindakan_keluar` tahap yang
 * bersangkutan, bukan teks yang ditebak layar. Bila layar mengirim tindakan yang bukan
 * penutup tahap itu, server menolaknya (`ADR-0023`).
 */
function StageActions({ tugas }: { tugas: Task }) {
  const done = useCompleteStage()

  return (
    <section className="mt-6 rounded border border-slate-200 p-4">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">
        Tahap {tugas.nama_tahap}
      </h2>
      <p className="mt-2 text-sm text-slate-600">
        Isian tahap ini dimiliki modul lain yang belum dibangun. Yang tersedia di sini
        adalah memindahkan klaim ke tahap berikutnya sesuai alur Register.
      </p>

      <div className="mt-4 flex flex-wrap gap-3">
        <button
          type="button"
          disabled={done.isPending}
          onClick={() => done.mutate({ taskID: tugas.id, action: tugas.tindakan_keluar })}
          className="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {done.isPending ? 'Memproses…' : `Selesaikan (${tugas.tindakan_keluar})`}
        </button>
        <button
          type="button"
          disabled={done.isPending}
          onClick={() =>
            done.mutate({ taskID: tugas.id, action: tugas.tindakan_keluar, kembali: true })
          }
          className="rounded border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
        >
          Kembali (Back)
        </button>
      </div>

      {done.isError && (
        <div className="mt-3">
          <ErrorMessage title="Tahap tidak dapat ditutup" description={errorMessage(done.error)} tone="penolakan" />
        </div>
      )}

      {done.isSuccess && done.data.jejak_keputusan && done.data.jejak_keputusan.length > 0 && (
        <DecisionTrace trace={done.data.jejak_keputusan} />
      )}
    </section>
  )
}

/**
 * Jejak keputusan alur yang baru saja dilewati.
 *
 * Di sistem lama, klaim berpindah tahap tanpa penjelasan apa pun yang terlihat petugas —
 * dan pada satu keputusan, tujuannya bahkan bergantung pada peran orang yang menekan
 * tombol. Menampilkan cabang yang dipilih membuat perpindahan itu dapat dijelaskan tanpa
 * membaca kode.
 */
function DecisionTrace({ trace }: { trace: string[] }) {
  return (
    <div className="mt-4 rounded border border-slate-200 bg-slate-50 p-3">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
        Percabangan yang dilewati
      </p>
      <ul className="mt-2 space-y-1 text-sm text-slate-700">
        {trace.map((j) => (
          <li key={j}>{j}</li>
        ))}
      </ul>
    </div>
  )
}

// ── Formulir Input Register ────────────────────────────────────────────────────────

type SpreadingInput = {
  jenis_treaty: string
  nama: string
  share: string
  objek_fac_offer: string
  dihapus: boolean
}

type CoverageInput = {
  id: string
  nama: string
  penyebab_kerugian: string
  tsi: string
  spreading: SpreadingInput[]
}

type InsuredItemInput = {
  id: string
  nama: string
  lokasi: string
  coverage: CoverageInput[]
}

type RegisterFormValues = {
  /** Jenis Laporan — T_CLAIM_PNC.REPORTTYPE. */
  jenis_laporan: string
  tanggal_kejadian: string
  tanggal_lapor: string
  tanggal_terima_dokumen: string
  /** Apakah Melakukan Rawat Inap — tidak punya kolom; tercentang bila tanggal keluar terisi. */
  rawat_inap: boolean
  /** Tanggal Keluar Rawat Inap — T_CLAIM_PNC.TANGGALSELESAIRAWATINAP. */
  tanggal_keluar_rawat_inap: string
  lokasi: string
  kronologi: string
  pelapor_nama: string
  pelapor_telepon: string
  pelapor_email: string
  pelapor_alamat: string
  pelapor_hubungan: string
  pelapor_hubungan_lainnya: string
  nilai_estimasi: string
  mata_uang: string
  nomor_slik: string
  user_teknis: string
  rcv_id: string
  /** Radio YES / NO seperti layar Pega; dikirim ke server sebagai boolean. */
  ex_gratia: 'YES' | 'NO'
  wilayah: Area
  prinsip_mengenal_nasabah: string
  komentar_suspicious: string
  status_pucl: string
  transfer_compliance: boolean
  objek: InsuredItemInput[]
  // Isian InputRegisterDetail2_sect yang tersimpan (T_CLAIM_PNC).
  email_lod: string
  rekomendasi: string
  subjek_email: string
  status_salvage: string
}

/** Kode hubungan pelapor yang menuntut keterangan tambahan (langkah 28 sistem lama). */
const HUBUNGAN_LAIN_LAIN = '7'

/** Pemisah beberapa alamat pada Email Pelapor. */
const EMAIL_SEPARATOR = /[;,]/

/** Bentuk alamat email yang diterima: satu "@", tanpa spasi, domain bertitik. */
const EMAIL_PATTERN = /^[^\s@;,]+@[^\s@;,]+\.[^\s@;,]+$/

/**
 * Alamat Email Pelapor yang tidak berbentuk email. Isian boleh kosong, dan boleh memuat
 * beberapa alamat dipisah ";" atau ","; pemisah ganda atau di ujung diabaikan.
 */
export function invalidEmails(text: string | undefined): string[] {
  return (text ?? '')
    .split(EMAIL_SEPARATOR)
    .map((part) => part.trim())
    .filter((part) => part !== '' && !EMAIL_PATTERN.test(part))
}

function emailMessage(bad: string[]): string {
  return `Email Pelapor tidak valid: ${bad.join(', ')}. Pisahkan beberapa email dengan ";" atau ",".`
}

/** Baris grid Objek pertanggungan per halaman (Work Owner 2026-10-09). */
const ITEM_PAGE_SIZE = 10

/** Status Pelapor — Property InsuredRelationship (`.ClaimData.InsuredRelationship`). */
const RELATIONSHIPS = [
  { value: '1', label: 'Tertanggung' },
  { value: '2', label: 'Suami/Istri' },
  { value: '3', label: 'Anak' },
  { value: '4', label: 'Orang Tua' },
  { value: '5', label: 'Famili' },
  { value: '6', label: 'Teman' },
  { value: '7', label: 'Lainnya' },
]

/** Status Pelapor Tertanggung — SetTertanggungRegister. */
const RELATIONSHIP_INSURED = '1'

/** Jenis Laporan pelaporan online — SetTertanggungRegister (`.ClaimData.ReportType == 7`). */
const REPORT_TYPE_ONLINE = '7'

/**
 * Formulir tahap Input Register.
 *
 * # Kenapa tidak ada aturan bisnis di sini
 *
 * Seluruh aturan — urutan tanggal, periode polis, duplikasi, kelengkapan, total
 * spreading — ditegakkan SERVER, dan layar ini hanya menampilkan jawabannya. Itu bukan
 * kemalasan: `ADR-0023` menetapkan aturan yang berada di layar dapat dipintas lewat
 * pemanggilan langsung ke API, dan aturan yang ditulis di dua tempat pasti berbeda cepat
 * atau lambat.
 *
 * Yang dilakukan layar adalah memastikan setiap pesan dari server menempel pada KOLOM
 * yang benar, supaya petugas tahu apa yang harus ia perbaiki.
 */
function FormRegister({ klaim, tugas }: { klaim: Claim; tugas: Task }) {
  const save = useSaveRegister()
  const draft = useSaveDraft()
  const navigate = useNavigate()
  const [violations, setViolations] = useState<Violation[]>([])

  const { register, control, handleSubmit, watch, reset, setValue } = useForm<RegisterFormValues>({
    defaultValues: fromClaim(klaim),
  })

  // Klaim yang dimuat ulang — misalnya setelah tombol Back — mengisi ulang formulir.
  useEffect(() => {
    reset(fromClaim(klaim))
  }, [klaim, reset])

  const objek = useFieldArray({ control, name: 'objek' })
  // Objek yang baru ditambahkan: barisnya digambar sudah terbuka. Menambah objek lalu
  // mendapati tidak ada yang terjadi adalah tombol yang tampak rusak.
  const [addedItem, setAddedItem] = useState<number | null>(null)
  // Grid objek dipaginasi di layar, ITEM_PAGE_SIZE baris per halaman (Work Owner 2026-10-09):
  // klaim PA dapat memuat ratusan peserta. Baris di halaman lain tetap ada di form dan ikut
  // tersimpan; hanya tidak digambar.
  const [itemPage, setItemPage] = useState(1)
  const fieldErrors = messagesByField(violations)
  const hubungan = watch('pelapor_hubungan')

  // SetTertanggungRegister — dijalankan saat Status Pelapor dipilih:
  //   Tertanggung, bukan pelaporan online: Nama Pelapor = QQName polis, No. Telepon Pelapor =
  //     telepon pertama alamat pertama tertanggung (AddressList(1).ASMTelfax(1)), Alamat Pelapor
  //     = alamat pertama (AddressList(1).ASMAddress).
  //   Tertanggung, pelaporan online (Jenis Laporan 7): data pengirim berkas RCV — sudah terisi
  //     dari berkas saat klaim dibuka, sehingga dibiarkan.
  //   Selain Tertanggung: Nama, No. Telepon, dan Alamat Pelapor dikosongkan.
  const insuredProfile = useInsuredProfile(klaim.id)
  const applyRelationship = (value: string) => {
    const dirty = { shouldDirty: true }
    if (value !== RELATIONSHIP_INSURED) {
      setValue('pelapor_nama', '', dirty)
      setValue('pelapor_telepon', '', dirty)
      setValue('pelapor_alamat', '', dirty)
      return
    }
    if (watch('jenis_laporan') === REPORT_TYPE_ONLINE) return
    const address = insuredProfile.data?.alamat?.[0]
    setValue('pelapor_nama', klaim.polis.nama_tertanggung ?? '', dirty)
    setValue('pelapor_telepon', address?.telepon?.[0]?.nomor ?? '', dirty)
    setValue('pelapor_alamat', address?.alamat ?? '', dirty)
  }

  // Kondisi tampil Section/InputRegisterDetail.
  const panel = klaim.polis.lini
  const pa = panel === PANEL_PA
  const travel = panel === PANEL_TRAVEL
  const grid = objectGrid(panel, klaim.polis.jenis_bisnis ?? '')
  const gridColumns = OBJECT_COLUMNS[grid]
  const formItems = watch('objek')

  // Mata Uang (.ClaimData.Currency) bernilai KODE master POOLDATA.CURRENCY (10026 = IDR),
  // bukan simbol. Isiannya tidak ditampilkan di Input Register, tetapi tetap dikirim; klaim
  // lama yang telanjur menyimpan simbol diterjemahkan ke kodenya begitu daftar termuat.
  const currencies = useCurrencies()
  const currencyList = currencies.data?.pilihan ?? []
  const currencyValue = watch('mata_uang') ?? ''
  useEffect(() => {
    const value = currencyValue.trim()
    if (value === '' || currencyList.some((c) => c.id === value)) return
    const bySymbol = currencyList.find((c) => c.nama.trim().toUpperCase() === value.toUpperCase())
    if (bySymbol) setValue('mata_uang', bySymbol.id, { shouldDirty: true })
  }, [currencyValue, currencyList, setValue])
  // Tanggal Terima Dokumen: IsTravelPA. Tetap ditampilkan bila server menolaknya, supaya
  // petugas dapat memperbaikinya.
  const showDateReceived = pa || travel || Boolean(fieldErrors['tanggal_terima_dokumen'])

  // PA: Tanggal Terima Dokumen terisi otomatis tanggal input (hari ini, WIB) bila masih kosong —
  // permintaan Work Owner 2026-10-08. Tetap dapat diubah petugas.
  const dateReceived = watch('tanggal_terima_dokumen')
  useEffect(() => {
    if (pa && !dateReceived) setValue('tanggal_terima_dokumen', todayWIB(), { shouldDirty: true })
  }, [pa, dateReceived, setValue])

  // No KTP dan Pengkinian Data — tersimpan ke T_CLAIM_PNC.PENGKINIAN_NO_KTP / _NO_HP / _EMAIL
  // (Work Owner 2026-10-09). Dimuat ulang dari klaim setiap kali klaim dimuat.
  const [insuredUpdate, setInsuredUpdate] = useState(() => insuredUpdateOf(klaim))
  useEffect(() => {
    setInsuredUpdate(insuredUpdateOf(klaim))
  }, [klaim])

  const visible = (content: RegisterFormValues): RegisterFormValues => ({
    ...content,
    email_lod: pa ? content.email_lod : '',
  })
  // Hanya PA yang menampilkan isiannya; lini lain mengirim nilai tersimpan apa adanya.
  const withInsuredUpdate = (request: RegisterRequest): RegisterRequest => ({
    ...request,
    ...(pa ? insuredUpdate : insuredUpdateOf(klaim)),
  })

  // Halaman dijepit ke jumlah halaman yang ada: menghapus objek terakhir di halaman terakhir
  // tidak meninggalkan halaman kosong.
  const itemPages = Math.max(1, Math.ceil(objek.fields.length / ITEM_PAGE_SIZE))
  const itemCurrentPage = Math.min(itemPage, itemPages)
  const itemFirst = (itemCurrentPage - 1) * ITEM_PAGE_SIZE

  const submit = (content: RegisterFormValues, kembali: boolean) => {
    // Lokasi Kerugian/Kejadian wajib (`pyRequired=always` di InputRegisterDetail-sect). Next
    // ditahan di layar lebih dulu; server menolaknya juga (ViolationLocationEmpty), sehingga
    // klaim tidak lagi lolos ke akseptasi dengan lokasi kosong ("Location kosong !!").
    // Back tidak diperiksa: ia hanya mengembalikan tahap.
    if (!kembali && !pa && !(content.lokasi ?? '').trim()) {
      save.reset()
      draft.reset()
      setViolations([{ kode: 'lokasi_kosong', field: 'lokasi', pesan: LOCATION_REQUIRED }])
      document.getElementById('lokasi')?.focus()
      return
    }
    const badEmail = invalidEmails(content.pelapor_email)
    if (badEmail.length > 0) {
      save.reset()
      draft.reset()
      setViolations([{ kode: 'email_pelapor_tidak_valid', field: 'pelapor_email', pesan: emailMessage(badEmail) }])
      document.getElementById('pelapor_email')?.focus()
      return
    }
    setViolations([])
    draft.reset()
    save.mutate(withInsuredUpdate(toRequest(visible(content), tugas.id, kembali)), {
      onError: (failure) => setViolations(violationsFrom(failure)),
    })
  }

  // Save menyimpan TANPA menutup tahap dan tanpa gerbang validasi (usecase.SaveDraft).
  const saveOnly = (content: RegisterFormValues) => {
    setViolations([])
    save.reset()
    draft.mutate(withInsuredUpdate(toRequest(visible(content), tugas.id, false)))
  }

  const busy = save.isPending || draft.isPending

  return (
    <form className="mt-6 space-y-6" onSubmit={handleSubmit((content) => submit(content, false))}>
      {save.isError && (
        <ErrorMessage
          title={violations.length > 0 ? 'Klaim belum dapat disimpan' : 'Penyimpanan gagal'}
          description={errorMessage(save.error)}
          tone={violations.length > 0 ? 'penolakan' : 'gangguan'}
        />
      )}

      {violations.length > 1 && (
        <div className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          <p className="font-medium">{violations.length} ketentuan belum terpenuhi:</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            {violations.map((p, i) => (
              <li key={`${p.kode}-${i}`}>{p.pesan}</li>
            ))}
          </ul>
        </div>
      )}

      {save.isSuccess && save.data.large_loss && (
        <div className="rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          <p className="font-medium">Notice of Large Losses diterbitkan.</p>
          <p className="mt-1">
            Nilai estimasi melampaui ambang, sehingga pemberitahuan ke Underwriting dan
            pimpinan ikut dicatat.
          </p>
        </div>
      )}

      {draft.isSuccess && (
        <div className="rounded border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-900">
          Isian tersimpan. Klaim tetap di tahap Input Register sampai Anda menekan Next.
        </div>
      )}
      {draft.isError && (
        <ErrorMessage title="Isian belum tersimpan" description={errorMessage(draft.error)} tone="gangguan" />
      )}

      {/*
        Isian tab Register = Section/InputRegisterDetail (bingkai InputRegister-sect), berurutan
        dengan kondisi tampilnya. Isian tanpa kolom di T_CLAIM_PNC tampil bergaris putus-putus.
      */}
      {/* "Catatan dari Analyst" — .ClaimData.AnaylstRemarks, syarat StatusAnalystRemarks == 1. */}
      <AnalystNoteNotice claimID={klaim.id} />

      {/* Kontainer IsPA: DATA TERTANGGUNG KLAIM dan InputAddress_PNC_Klaim. */}
      {pa && (
        <InsuredDataSection
          klaim={klaim}
          idCard={insuredUpdate.pengkinian_no_ktp}
          onIDCard={(v) => setInsuredUpdate((u) => ({ ...u, pengkinian_no_ktp: v }))}
          phone={insuredUpdate.pengkinian_no_hp}
          onPhone={(v) => {
            // No. Telepon Pelapor mengikuti No. HP Pengkinian Data — permintaan Work Owner
            // 2026-10-08. Tetap dapat diubah sesudahnya.
            setInsuredUpdate((u) => ({ ...u, pengkinian_no_hp: v }))
            setValue('pelapor_telepon', v, { shouldDirty: true })
          }}
          email={insuredUpdate.pengkinian_email}
          onEmail={(v) => {
            // Email Pelapor dan Email Tertanggung mengikuti Email Pengkinian Data — permintaan
            // Work Owner 2026-10-08. Keduanya tetap dapat diubah sesudahnya.
            setInsuredUpdate((u) => ({ ...u, pengkinian_email: v }))
            setValue('pelapor_email', v, { shouldDirty: true })
            setValue('email_lod', v, { shouldDirty: true })
          }}
        />
      )}

      <Section title="Input Register">
        <div className="grid gap-4 sm:grid-cols-2">
          {!travel && (
            <FormField id="rcv_id" label="RCV_ID" readOnly {...register('rcv_id')} />
          )}
          {showDateReceived && (
            <Controller control={control} name="tanggal_terima_dokumen" render={({ field }) => (
              <DateField id="tanggal_terima_dokumen" label="Tanggal Terima Dokumen" value={field.value}
                onChange={field.onChange} error={fieldErrors['tanggal_terima_dokumen']} />
            )} />
          )}
          <Controller control={control} name="tanggal_kejadian" render={({ field }) => (
            <DateField id="tanggal_kejadian" label="Tanggal Kejadian / Tanggal Masuk Rawat Inap" value={field.value}
              onChange={field.onChange} error={fieldErrors['tanggal_kejadian']} />
          )} />
          {pa && (
            <div className="flex items-end pb-2">
              <label className="flex items-center gap-2 text-sm text-slate-700">
                <input type="checkbox" className="h-4 w-4 rounded border-slate-300" {...register('rawat_inap')} />
                Apakah Melakukan Rawat Inap ?
              </label>
            </div>
          )}
          {/* Tanggal Keluar Rawat Inap — tersimpan ke T_CLAIM_PNC.TANGGALSELESAIRAWATINAP
              (Work Owner 2026-10-09). Lama Hari Rawat Inap dihitung darinya di layar Adjustment. */}
          {pa && watch('rawat_inap') && (
            <Controller control={control} name="tanggal_keluar_rawat_inap" render={({ field }) => (
              <DateField id="tanggal_keluar_rawat_inap" label="Tanggal Keluar Rawat Inap" value={field.value}
                onChange={field.onChange} error={fieldErrors['tanggal_keluar_rawat_inap']} />
            )} />
          )}
          <Controller control={control} name="tanggal_lapor" render={({ field }) => (
            <DateField id="tanggal_lapor" label="Tanggal Lapor" value={field.value} onChange={field.onChange}
              error={fieldErrors['tanggal_lapor']} />
          )} />
          {/* Tgl Terima HCDKP disembunyikan — tidak dipakai (Work Owner 2026-10-08). */}
          <FormField id="pelapor_nama" label="Nama Pelapor" {...register('pelapor_nama')} />
          {/* Jenis Laporan — tersimpan ke T_CLAIM_PNC.REPORTTYPE (Work Owner 2026-10-09). Kode
              tersimpan di luar daftar (mis. 7 dari unggahan) tetap ditampilkan apa adanya. */}
          <SelectField id="jenis_laporan" label="Jenis Laporan" emptyText="— pilih —"
            options={reportTypeOptions(watch('jenis_laporan'))} {...register('jenis_laporan')} />
          <SelectField id="pelapor_hubungan" label="Status Pelapor" emptyText="— pilih —"
            options={withStoredCode(RELATIONSHIPS, hubungan)}
            {...register('pelapor_hubungan', { onChange: (e) => applyRelationship(e.target.value) })} />
          {hubungan === HUBUNGAN_LAIN_LAIN && (
            <FormField id="hubungan_lainnya" label="Sebutkan..."
              failure={fieldErrors['hubungan_lainnya']} {...register('pelapor_hubungan_lainnya')} />
          )}
          <FormField id="pelapor_telepon" label="No. Telepon Pelapor" {...register('pelapor_telepon')} />
          {/* Lebih dari satu alamat boleh, dipisah ";" atau "," (Work Owner 2026-10-09) — karena itu
              teks biasa, bukan type="email" yang menolak pemisah. Setiap alamat diperiksa saat simpan. */}
          <FormField id="pelapor_email" label="Email Pelapor" placeholder="nama@contoh.co.id; nama2@contoh.co.id"
            failure={fieldErrors['pelapor_email']} {...register('pelapor_email')} />
          {pa && (
            <TextAreaField id="email_lod" label="Email Tertanggung" rows={2} {...register('email_lod')} />
          )}
          <TextAreaField id="pelapor_alamat" label="Alamat Pelapor" rows={2} {...register('pelapor_alamat')} />
        </div>
      </Section>

      {/* Detail Ekspedisi (kontainer GroupPanel == 002) disembunyikan — tidak digunakan (Work Owner 2026-10-08). */}

      <Section title="Deksripsi Laporan">
        <TextAreaField id="kronologi" label="Deksripsi Laporan" rows={3} {...register('kronologi')} />
      </Section>

      <Section title="Objek pertanggungan">
        {fieldErrors['objek'] && <p className="mb-3 text-sm text-red-700">{fieldErrors['objek']}</p>}
        {fieldErrors['penyebab_kerugian'] && (
          <p className="mb-3 text-sm text-red-700">{fieldErrors['penyebab_kerugian']}</p>
        )}
        {fieldErrors['spreading'] && <p className="mb-3 text-sm text-red-700">{fieldErrors['spreading']}</p>}

        {/*
          Grid objek Section/InputRegisterDetail-sect.xml — EMPAT grid berbeda menurut lini
          (lihat ObjectGrid dan OBJECT_COLUMNS). Kolomnya berasal dari polis dan read-only,
          seperti di XML. Objek yang ditambahkan petugas sendiri tidak punya pasangan di
          polis, sehingga nama dan lokasinya tetap dapat diisi.

          Ejaan judulnya "Pekerjaan", bukan "Perkerjaan" seperti di Pega. Salah ketik itu
          tidak dibawa karena layar Surveyor di aplikasi ini sudah menulisnya benar, dan
          dua ejaan berbeda untuk kolom yang sama di dua layar lebih membingungkan
          daripada selisih satu huruf terhadap Pega.
        */}
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <caption className="sr-only">Objek pertanggungan beserta jaminannya</caption>
            <thead>
              <tr className="bg-slate-100 text-left text-xs text-slate-700">
                {/* Kolom tombol buka-tutup. Judulnya kosong: tombolnya sudah ber-aria-label. */}
                <th scope="col" className="w-8 border border-slate-300 p-2" />
                <th scope="col" className="w-10 border border-slate-300 p-2 text-right">#</th>
                {gridColumns.map((title) => (
                  <th key={title} scope="col" className="border border-slate-300 p-2">
                    {title}
                  </th>
                ))}
                <th scope="col" className="w-20 border border-slate-300 p-2">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {objek.fields.length === 0 && (
                <tr>
                  <td colSpan={gridColumns.length + 3} className="border border-slate-300 p-2 text-xs text-slate-500">
                    Data Tidak Ada
                  </td>
                </tr>
              )}
              {objek.fields.slice(itemFirst, itemFirst + ITEM_PAGE_SIZE).map((f, n) => {
                const i = itemFirst + n
                return (
                <InsuredItemEditor
                  key={f.id}
                  claimID={klaim.id}
                  index={i}
                  grid={grid}
                  defaultOpen={i === addedItem}
                  currency={klaim.polis.mata_uang ?? ''}
                  control={control}
                  register={register}
                  setValue={setValue}
                  businessCode={klaim.polis.kode_bisnis ?? ''}
                  saved={klaim.objek.find((o) => o.id.trim() === (formItems?.[i]?.id ?? '').trim())}
                  pa={pa}
                  onRemove={() => objek.remove(i)}
                />
                )
              })}
            </tbody>
          </table>
        </div>
        {objek.fields.length > ITEM_PAGE_SIZE && (
          <Paginator
            firstRow={itemFirst + 1}
            lastRow={Math.min(itemFirst + ITEM_PAGE_SIZE, objek.fields.length)}
            totalRows={objek.fields.length}
            currentPage={itemCurrentPage}
            totalPages={itemPages}
            onPick={setItemPage}
          />
        )}

        {/* PA: objek mengikuti peserta polis — Tambah objek dan Hapus objek disembunyikan
            (Work Owner 2026-10-08). */}
        {!pa && (
        <button
          type="button"
          onClick={() => {
            setAddedItem(objek.fields.length)
            // Objek baru ada di baris terakhir: buka halaman terakhir supaya terlihat.
            setItemPage(Math.floor(objek.fields.length / ITEM_PAGE_SIZE) + 1)
            // Kode objek tidak punya kolom di grid Pega; objek tambahan diberi kode berikutnya.
            const used = [...(formItems ?? []).map((o) => o.id), ...klaim.objek.map((o) => o.id)]
            objek.append({ id: nextObjectID(used), nama: '', lokasi: '', coverage: [] })
          }}
          className="mt-4 rounded border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
        >
          Tambah objek
        </button>
        )}
      </Section>

      <SpreadingSummary values={watch('objek')} />

      {/* PA: Lokasi Kerugian/Kejadian ditampilkan tanpa Negara, Provinsi, dan wilayah di bawahnya
          (Work Owner 2026-10-09); Data registrasi lainnya tetap disembunyikan (2026-10-08). Lokasi
          tidak wajib untuk PA, di layar maupun di server. Wilayah tersimpan tetap terkirim apa adanya. */}
      <LossLocationSection register={register} watch={watch} setValue={setValue} fieldErrors={fieldErrors}
        withArea={!pa} required={!pa} />

      {/*
        Bagian Estimasi (Mata Uang, Estimasi Klaim, Prinsip Mengenal Nasabah, Akan Dikirim ke
        User Teknis, beserta isian PA/Travel-nya) TIDAK ditampilkan pada Input Register —
        keputusan Work Owner 2026-10-08, berbeda dari InputRegisterDetail-sect.xml. Nilainya
        tetap dikirim apa adanya dari klaim yang tersimpan (react-hook-form menahan nilai
        bawaan), sehingga menyimpan Input Register tidak mengosongkannya. Estimasi diisi di
        tahap Input Estimasi.
      */}
      {(!pa || fieldErrors['nomor_slik'] || fieldErrors['nomor_polis']) && (
      <Section title="Data registrasi lainnya">
        <p className="mb-3 text-xs text-slate-500">
          These fields are not on the Pega Register tab (section InputRegisterDetail) but are needed to register the claim.
        </p>
        <div className="grid gap-4 sm:grid-cols-3">
          <FormField id="nomor_slik" label="Nomor SLIK"
            failure={fieldErrors['nomor_slik']} {...register('nomor_slik')} />
          <FormField id="status_pucl" label="Status RCL/PUCL" inputMode="numeric" {...register('status_pucl')} />
          <div className="flex items-end gap-6 pb-2">
            <Toggle label="Transfer Compliance" {...register('transfer_compliance')} />
          </div>
        </div>

        {fieldErrors['nomor_polis'] && (
          <p className="mt-3 text-sm text-red-700">{fieldErrors['nomor_polis']}</p>
        )}
      </Section>
      )}


      {/*
        Susunan tombol mengikuti layar tahap Pega: Cancel dan Back di kiri, Save dan Next
        di kanan.

          Cancel  keluar tanpa menyimpan
          Back    kembali ke View Polis (pyNote "Back")
          Save    menyimpan tanpa menutup tahap dan tanpa gerbang validasi
          Next    menutup Input Register lewat seluruh gerbang validasi, lalu klaim
                  berpindah ke tahap berikutnya menurut Register_Flow — Input Estimasi
                  untuk Non-MBU dan Travel, Estimation untuk PA
      */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 pt-4">
        <div className="flex gap-2">
          <Button type="button" tone="halus" disabled={busy} onClick={() => navigate('/registrasi')}>
            Cancel
          </Button>
          <Button
            type="button"
            tone="kedua"
            disabled={busy}
            onClick={handleSubmit((content) => submit(content, true))}
          >
            Back
          </Button>
        </div>
        <div className="flex gap-2">
          <Button type="button" tone="kedua" disabled={busy} onClick={handleSubmit(saveOnly)}>
            {draft.isPending ? 'Menyimpan…' : 'Save'}
          </Button>
          <Button type="submit" tone="utama" disabled={busy}>
            {save.isPending ? 'Memproses…' : 'Next'}
          </Button>
        </div>
      </div>
    </form>
  )
}

// ── Lokasi kejadian ────────────────────────────────────────────────────────────────

/**
 * Lokasi Kerugian/Kejadian beserta wilayahnya — kontainer `!IsHE` pada Section/InputRegisterDetail.
 * Prinsip Mengenal Nasabah dan Ex Gratia berada di bagian Estimasi, yang tidak ditampilkan di
 * Input Register.
 *
 * # Daftar pilihan bertingkat
 *
 * Negara → Provinsi → Kota → Kabupaten → Kelurahan. Setiap tingkat baru dimuat setelah
 * induknya dipilih, dan mengganti induk mengosongkan seluruh tingkat di bawahnya —
 * kabupaten dari kota yang lama tidak boleh tertinggal di bawah kota yang baru.
 *
 * Memilih kelurahan mengisi Kode Pos dari master (ZipCode baris RW), dan Kode Pos tetap
 * dapat diubah — di Pega ia isian teks biasa.
 *
 * Kota sampai Kode Pos hanya tampil bila Negara INDONESIA, mengikuti kondisi
 * .ClaimData.Country = 'INDONESIA' pada section Pega.
 */
function LossLocationSection({
  register,
  watch,
  setValue,
  fieldErrors,
  withArea,
  required,
}: {
  register: UseFormRegister<RegisterFormValues>
  watch: UseFormWatch<RegisterFormValues>
  setValue: UseFormSetValue<RegisterFormValues>
  fieldErrors: Record<string, string>
  /** Tampilkan Negara, Provinsi, dan tingkat di bawahnya — tidak untuk PA. */
  withArea: boolean
  /** Lokasi wajib diisi — tidak untuk PA. */
  required: boolean
}) {
  const w = watch('wilayah')
  const indonesia = (w.negara ?? '').toUpperCase() === COUNTRY_INDONESIA
  const countries = useAreaOptions(AreaLevel.Country, '')
  // Provinsi disaring menurut NAMA negara — lihat catatan AreaDirectory di backend.
  const provinces = useAreaOptions(AreaLevel.Province, w.negara ?? '')
  const cities = useAreaOptions(AreaLevel.City, indonesia ? w.provinsi_id ?? '' : '')
  const districts = useAreaOptions(AreaLevel.District, indonesia ? w.kota_id ?? '' : '')
  const villages = useAreaOptions(AreaLevel.Village, indonesia ? w.kabupaten_id ?? '' : '')

  const set = (field: keyof Area, value: string) =>
    setValue(`wilayah.${field}`, value, { shouldDirty: true })

  // Urutan tingkat, dari atas ke bawah. Memilih satu tingkat mengosongkan semua yang di bawahnya.
  const levels: [keyof Area, keyof Area][] = [
    ['negara', 'negara_id'],
    ['provinsi', 'provinsi_id'],
    ['kota', 'kota_id'],
    ['kabupaten', 'kabupaten_id'],
    ['kelurahan', 'kelurahan_id'],
  ]
  const pick = (index: number, option: AreaOption | undefined) => {
    const [name, id] = levels[index]!
    set(name, option?.nama ?? '')
    set(id, option?.id ?? '')
    for (const [childName, childID] of levels.slice(index + 1)) {
      set(childName, '')
      set(childID, '')
    }
    if (index >= levels.length - 1) {
      set('kode_pos', option?.kode_pos ?? '')
    } else {
      set('kode_pos', '')
    }
  }

  return (
    <Section title="Lokasi kerugian/kejadian">
      <TextAreaField
        id="lokasi"
        label={required ? 'Lokasi Kerugian/Kejadian *' : 'Lokasi Kerugian/Kejadian'}
        rows={3}
        error={fieldErrors['lokasi']}
        {...register('lokasi')}
      />

      {withArea && (
      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <AreaSelect id="negara" label="Negara" query={countries} value={w.negara_id} currentName={w.negara}
          onPick={(o) => pick(0, o)} />
        <AreaSelect id="provinsi" label="Provinsi" query={provinces} value={w.provinsi_id} currentName={w.provinsi}
          disabled={!w.negara} onPick={(o) => pick(1, o)} />

        {indonesia && (
          <>
            <AreaSelect id="kota" label="Kota" query={cities} value={w.kota_id} currentName={w.kota}
              disabled={!w.provinsi_id} onPick={(o) => pick(2, o)} />
            <AreaSelect id="kabupaten" label="Kabupaten" query={districts} value={w.kabupaten_id}
              currentName={w.kabupaten} disabled={!w.kota_id} onPick={(o) => pick(3, o)} />
            <AreaSelect id="kelurahan" label="Kelurahan" query={villages} value={w.kelurahan_id}
              currentName={w.kelurahan} disabled={!w.kabupaten_id} onPick={(o) => pick(4, o)} />
            <FormField id="kode_pos" label="Kode Pos" inputMode="numeric" {...register('wilayah.kode_pos')} />
          </>
        )}
      </div>
      )}

    </Section>
  )
}

/**
 * Satu tingkat daftar wilayah. Nilai yang tersimpan tetapi tidak lagi ada di master
 * tetap ditampilkan dengan namanya, supaya membuka ulang klaim lama tidak diam-diam
 * mengosongkan isiannya.
 */
function AreaSelect({
  id,
  label,
  query,
  value,
  currentName,
  disabled,
  onPick,
}: {
  id: string
  label: string
  query: { data?: { pilihan: AreaOption[] } | undefined; isFetching: boolean }
  value: string | undefined
  currentName: string | undefined
  disabled?: boolean
  onPick: (option: AreaOption | undefined) => void
}) {
  const option = query.data?.pilihan ?? []
  const known = option.some((o) => o.id === value)
  const list = [
    ...(value && !known && currentName ? [{ value, label: currentName }] : []),
    ...option.map((o) => ({ value: o.id, label: o.nama })),
  ]
  return (
    <SelectField
      id={id}
      label={label}
      options={list}
      value={value ?? ''}
      disabled={disabled}
      emptyText={query.isFetching ? 'Memuat…' : '— pilih —'}
      onChange={(e) => onPick(option.find((o) => o.id === e.target.value))}
    />
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded border border-slate-200 p-4">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">{title}</h2>
      <div className="mt-3">{children}</div>
    </section>
  )
}

/**
 * Sakelar adalah satu kotak centang berlabel.
 *
 * Ref diteruskan supaya `register` dari React Hook Form dapat memegang elemennya —
 * tanpa itu, nilainya tidak pernah ikut terkirim.
 */
const Toggle = forwardRef<
  HTMLInputElement,
  { label: string } & InputHTMLAttributes<HTMLInputElement>
>(function Toggle({ label, ...remaining }, ref) {
  return (
    <label className="flex items-center gap-2 text-sm text-slate-700">
      <input ref={ref} type="checkbox" className="h-4 w-4 rounded border-slate-300" {...remaining} />
      {label}
    </label>
  )
})

/**
 * InsuredItemEditor menggambar SATU objek sebagai dua baris tabel: baris datanya, dan
 * baris jaminannya yang membentang seluruh kolom.
 *
 * # Kenapa dua baris, bukan satu kartu
 *
 * Karena objek berjajar dalam kolom yang sama dapat dibandingkan sekilas — nama di bawah
 * nama, tanggal di bawah tanggal. Pada kartu bertumpuk, isian yang sama berpindah tempat
 * di setiap objek, dan mata harus mencarinya satu per satu. Itulah bentuk grid di
 * `Section/InputRegisterDetail-sect.xml`, dan alasannya sama.
 *
 * Jaminan tetap berada di baris terpisah di bawah objeknya: ia punya tabel spreading
 * sendiri, dan menyelipkannya ke dalam satu sel akan merusak perjajaran yang baru saja
 * dibangun.
 */
/**
 * Disclosure adalah tombol buka-tutup satu baris grid.
 *
 * Ia `aria-expanded`, bukan sekadar tanda panah yang berputar: pembaca layar harus tahu
 * baris itu sedang terbuka atau tertutup, dan panah hanyalah gambar.
 */
function Disclosure({ open, label, onClick }: { open: boolean; label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      aria-expanded={open}
      aria-label={label}
      onClick={onClick}
      className="flex h-6 w-6 items-center justify-center rounded border border-slate-300 bg-white text-xs text-slate-600 hover:bg-slate-100"
    >
      <span aria-hidden="true">{open ? '▾' : '▸'}</span>
    </button>
  )
}

/**
 * InsuredItemEditor menggambar SATU objek sebagai baris grid yang dapat dibuka.
 *
 * # Kenapa jaminannya disembunyikan sampai dibuka
 *
 * Karena satu klaim Fire dapat memuat belasan objek, dan setiap objek memuat beberapa
 * jaminan yang masing-masing punya tabel spreading sendiri. Menggambar seluruhnya
 * sekaligus menghasilkan halaman yang harus digulung berlayar-layar hanya untuk
 * menemukan objek kedua — dan perjajaran kolom yang menjadi alasan grid ini dibuat
 * justru hilang karenanya.
 *
 * Baris yang BARU DITAMBAHKAN petugas terbuka dengan sendirinya. Menambah objek lalu
 * mendapati tidak ada yang terjadi adalah tombol yang tampak rusak.
 */
function InsuredItemEditor({
  claimID,
  index,
  grid,
  defaultOpen,
  currency,
  control,
  register,
  setValue,
  businessCode,
  saved,
  pa = false,
  onRemove,
}: {
  /** Lini PA: tombol Hapus objek dan Tambah spreading disembunyikan. */
  pa?: boolean
  /** ID klaim — dipakai membaca pilihan coverage polis per objek. */
  claimID: string
  index: number
  /** Grid objek lini polis — menentukan kolom yang digambar. */
  grid: ObjectGrid
  /** Baris yang baru ditambahkan terbuka saat digambar pertama kali. */
  defaultOpen: boolean
  /** Mata uang polis — `pyWorkPage.Policy.Currency` pada grid jaminan Pega. */
  currency: string
  control: Control<RegisterFormValues>
  register: UseFormRegister<RegisterFormValues>
  setValue: UseFormSetValue<RegisterFormValues>
  businessCode: string
  /**
   * Objek yang sama dari klaim tersimpan, dicocokkan lewat kode objek. Kolom polis —
   * Pekerjaan, Tanggal Lahir, KTP/Paspor, Status, Model, Merk, Nama Tipe, Nomor Chasis —
   * dibaca dari sini dan tidak pernah disunting, sehingga tidak masuk keadaan formulir.
   * Objek yang baru ditambahkan petugas belum punya pasangannya: nama dan lokasinya
   * diisi petugas, kolom polis lainnya memang kosong.
   */
  saved: InsuredItem | undefined
  onRemove: () => void
}) {
  const coverage = useFieldArray({ control, name: `objek.${index}.coverage` })
  const [open, setOpen] = useState(defaultOpen)
  // Jaminan yang baru ditambahkan: barisnya digambar dengan spreading sudah terbuka.
  const [addedCoverage, setAddedCoverage] = useState<number | null>(null)
  const columns = OBJECT_COLUMNS[grid].length + 3
  const formName = useWatch({ control, name: `objek.${index}.nama` })
  const formLocation = useWatch({ control, name: `objek.${index}.lokasi` })
  const name = formName?.trim() || `baris ${index + 1}`

  // Kolom polis digambar sebagai teks, bukan isian yang dimatikan: isian berwarna abu-abu
  // tampak seperti sesuatu yang seharusnya dapat diisi tetapi sedang terkunci, padahal ia
  // memang bukan milik layar ini.
  const text = (value: string | undefined) => (
    <td className="border border-slate-300 p-2">{value?.trim() || '—'}</td>
  )
  // Nama dan lokasi: teks untuk objek polis, isian untuk objek tambahan petugas.
  const nameCell = saved ? (
    text(formName)
  ) : (
    <td className="border border-slate-300 p-1">
      <input
        aria-label={`Nama objek baris ${index + 1}`}
        className="w-full rounded border border-slate-300 px-2 py-1"
        {...register(`objek.${index}.nama`)}
      />
    </td>
  )
  const locationCell = saved ? (
    text(formLocation)
  ) : (
    <td className="border border-slate-300 p-1">
      <input
        aria-label={`Lokasi objek baris ${index + 1}`}
        className="w-full rounded border border-slate-300 px-2 py-1"
        {...register(`objek.${index}.lokasi`)}
      />
    </td>
  )

  return (
    <>
      <tr className="border border-slate-300 align-top">
        <td className="border border-slate-300 p-1 text-center">
          <Disclosure
            open={open}
            label={`${open ? 'Tutup' : 'Buka'} jaminan objek ${name}`}
            onClick={() => setOpen((v) => !v)}
          />
        </td>
        <td className="border border-slate-300 p-2 text-right text-slate-500">{index + 1}</td>
        {nameCell}
        {grid === 'travel' && (
          <>
            {text(saved?.status_peserta)}
            {text(saved?.ktp_paspor)}
            <td className="border border-slate-300 p-2">{birthDate(saved?.tanggal_lahir)}</td>
          </>
        )}
        {grid === 'pa' && (
          <>
            {text(saved?.pekerjaan)}
            <td className="border border-slate-300 p-2">{birthDate(saved?.tanggal_lahir)}</td>
          </>
        )}
        {grid === 'he' && (
          <>
            {text(saved?.model)}
            {text(saved?.merk)}
            {text(saved?.nama_tipe)}
            {text(saved?.nomor_chasis)}
            {locationCell}
          </>
        )}
        {grid === 'umum' && locationCell}
        <td className="border border-slate-300 p-1 text-center">
          {/*
            Label terbacanya "Hapus" supaya kolom Aksi tetap sempit, tetapi aria-label-nya
            menyebut objek dan nomor barisnya. Tanpa itu, satu layar memuat belasan tombol
            bernama "Hapus" yang semuanya terdengar sama bagi pembaca layar — dan tiga di
            antaranya membuang hal yang berbeda: objek, jaminan, dan baris spreading.
          */}
          {!pa && (
            <button
              type="button"
              aria-label={`Hapus objek baris ${index + 1}`}
              onClick={onRemove}
              className="rounded border border-red-200 bg-white px-2 py-1 text-xs text-red-700 hover:bg-red-50"
            >
              Hapus
            </button>
          )}
        </td>
      </tr>

      {open && (
        <tr>
          <td colSpan={columns} className="border border-slate-300 bg-slate-50 p-2">
            {/*
              Grid jaminan `Section/ObjectCoverageAdj-sect.xml`: Nama Coverage
              (`.CoverageNote`), Penyebab Kerugian (`.CauseOfLoss`), Mata Uang
              (`pyWorkPage.Policy.Currency`), TSI (`.SumTSI`). Kode Coverage ditambahkan
              karena ia ikut terkirim saat menyimpan dan tidak punya tempat lain.
            */}
            <table className="w-full border-collapse text-sm">
              <caption className="sr-only">Jaminan objek {name}</caption>
              <thead>
                <tr className="bg-slate-100 text-left text-xs text-slate-700">
                  <th scope="col" className="w-8 border border-slate-300 p-2" />
                  <th scope="col" className="border border-slate-300 p-2">Kode Coverage</th>
                  <th scope="col" className="border border-slate-300 p-2">Coverage</th>
                  <th scope="col" className="border border-slate-300 p-2">Penyebab Kerugian</th>
                  <th scope="col" className="w-24 border border-slate-300 p-2">Mata Uang</th>
                  <th scope="col" className="w-32 border border-slate-300 p-2">TSI</th>
                  <th scope="col" className="w-24 border border-slate-300 p-2 text-center">
                    <button
                      type="button"
                      onClick={() => {
                        setAddedCoverage(coverage.fields.length)
                        coverage.append({ id: '', nama: '', penyebab_kerugian: '', tsi: '', spreading: [] })
                      }}
                      className="rounded border border-blue-300 bg-white px-2 py-1 text-xs text-blue-700 hover:bg-blue-50"
                    >
                      Tambah coverage
                    </button>
                  </th>
                </tr>
              </thead>
              <tbody>
                {coverage.fields.length === 0 && (
                  <tr>
                    <td colSpan={7} className="border border-slate-300 bg-white p-2 text-xs text-slate-500">
                      Objek ini belum punya jaminan. Registrasi menolak objek tanpa jaminan.
                    </td>
                  </tr>
                )}
                {coverage.fields.map((f, j) => (
                  <CoverageEditor
                    key={f.id}
                    claimID={claimID}
                    itemIndex={index}
                    index={j}
                    defaultOpen={j === addedCoverage}
                    currency={currency}
                    control={control}
                    register={register}
                    setValue={setValue}
                    businessCode={businessCode}
                    pa={pa}
                    onRemove={() => coverage.remove(j)}
                  />
                ))}
              </tbody>
            </table>
          </td>
        </tr>
      )}
    </>
  )
}

/** Tanggal lahir grid objek PA ditulis seperti Pega: dd/MM/yyyy (mis. 03/02/1996). */
function birthDate(iso: string | undefined): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso ?? '')
  return m ? `${m[3]}/${m[2]}/${m[1]}` : '—'
}

/**
 * CoverageEditor menggambar SATU jaminan sebagai baris grid yang dapat dibuka.
 *
 * Yang tersembunyi di dalamnya adalah tabel spreading — bagian terbesar dari layar ini,
 * dan bagian yang paling jarang disentuh: spreading datang dari polis dan biasanya sudah
 * benar. Total share tetap terlihat tanpa membuka satu baris pun, lewat Ringkasan di
 * bawah daftar objek.
 */
function CoverageEditor({
  claimID,
  itemIndex,
  index,
  defaultOpen,
  currency,
  control,
  register,
  setValue,
  businessCode,
  pa = false,
  onRemove,
}: {
  /** Lini PA: Tambah spreading disembunyikan (spreading mengikuti polis). */
  pa?: boolean
  claimID: string
  itemIndex: number
  index: number
  defaultOpen: boolean
  currency: string
  control: Control<RegisterFormValues>
  register: UseFormRegister<RegisterFormValues>
  setValue: UseFormSetValue<RegisterFormValues>
  businessCode: string
  onRemove: () => void
}) {
  const nama = `objek.${itemIndex}.coverage.${index}` as const
  const spreading = useFieldArray({ control, name: `${nama}.spreading` })
  const cause = useWatch({ control, name: `${nama}.penyebab_kerugian` })
  const causes = useCauseOfLossOptions(businessCode)
  const causeOptions = causes.data?.pilihan ?? []
  const [open, setOpen] = useState(defaultOpen)

  // Dropdown kode coverage: coverage polis milik objek INI saja, dibaca dari
  // T_COVERAGELIST_CARGO/ANEKA/FIRE/PERSON sesuai lini bisnis. Memilih satu coverage
  // mengisi nama, TSI, dan spreading-nya dari polis; petugas tetap boleh mengubahnya.
  // Bila polis tidak punya coverage untuk objek ini (mis. T_COVERAGELIST_PERSON kosong),
  // isian kembali bebas seperti sebelumnya supaya klaim tetap dapat diisi.
  const objectID = useWatch({ control, name: `objek.${itemIndex}.id` }) ?? ''
  const coverageID = useWatch({ control, name: `${nama}.id` }) ?? ''
  const coverageOptions = useCoverageOptions(claimID, objectID)
  const policyCoverage = coverageOptions.data?.pilihan ?? []
  const useDropdown = policyCoverage.length > 0

  function applyCoverage(id: string) {
    const chosen = policyCoverage.find((c) => c.id === id)
    if (!chosen) return
    setValue(`${nama}.nama`, chosen.nama ?? '', { shouldDirty: true })
    setValue(`${nama}.tsi`, centsToRupiah(chosen.tsi_sen), { shouldDirty: true })
    spreading.replace(
      chosen.spreading.map((s) => ({
        jenis_treaty: s.jenis_treaty,
        nama: s.nama,
        share: (s.share / 10_000).toString(),
        objek_fac_offer: s.objek_fac_offer,
        dihapus: s.dihapus,
      })),
    )
  }

  // Kode bisnis yang pilihannya tepat satu tidak perlu dipilih petugas — pilihan itu
  // langsung diisi. Lebih dari satu pilihan tetap menunggu petugas.
  const only = causeOptions.length === 1 ? causeOptions[0]!.id : ''
  useEffect(() => {
    if (only !== '' && !cause) setValue(`${nama}.penyebab_kerugian`, only, { shouldDirty: true })
  }, [only, cause, nama, setValue])

  const label = `jaminan ${index + 1} objek ${itemIndex + 1}`

  return (
    <>
      <tr className="border border-slate-300 bg-white align-top">
        <td className="border border-slate-300 p-1 text-center">
          <Disclosure
            open={open}
            label={`${open ? 'Tutup' : 'Buka'} spreading ${label}`}
            onClick={() => setOpen((v) => !v)}
          />
        </td>
        <td className="border border-slate-300 p-1">
          {useDropdown ? (
            <select
              aria-label={`Kode coverage ${label}`}
              className="w-full rounded border border-slate-300 px-2 py-1"
              {...register(`${nama}.id`, { onChange: (event) => applyCoverage(event.target.value) })}
            >
              <option value="">— pilih coverage —</option>
              {coverageID !== '' && !policyCoverage.some((c) => c.id === coverageID) && (
                <option value={coverageID}>{coverageID}</option>
              )}
              {policyCoverage.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.nama ? `${c.id} — ${c.nama}` : c.id}
                </option>
              ))}
            </select>
          ) : (
            <input
              aria-label={`Kode coverage ${label}`}
              className="w-full rounded border border-slate-300 px-2 py-1"
              {...register(`${nama}.id`)}
            />
          )}
        </td>
        <td className="border border-slate-300 p-1">
          <input
            aria-label={`Nama coverage ${label}`}
            className="w-full rounded border border-slate-300 px-2 py-1"
            {...register(`${nama}.nama`)}
          />
        </td>
        <td className="border border-slate-300 p-1">
          <select
            aria-label={`Penyebab kerugian ${label}`}
            className="w-full rounded border border-slate-300 px-2 py-1"
            {...register(`${nama}.penyebab_kerugian`)}
          >
            <option value="">{causes.isFetching ? 'Memuat…' : '— pilih —'}</option>
            {causeSelectOptions(causeOptions, cause).map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </td>
        {/*
          Mata Uang mengikat `pyWorkPage.Policy.Currency` di Pega — mata uang POLIS, dan
          bukan sesuatu yang diisi per jaminan. Ia teks, bukan isian.
        */}
        <td className="border border-slate-300 p-2 text-slate-600">{currency || '—'}</td>
        <td className="border border-slate-300 p-1">
          <input
            aria-label={`TSI ${label}`}
            inputMode="decimal"
            className="w-full rounded border border-slate-300 px-2 py-1 text-right"
            {...register(`${nama}.tsi`)}
          />
        </td>
        <td className="border border-slate-300 p-1 text-center">
          <button
            type="button"
            aria-label={`Hapus ${label}`}
            onClick={onRemove}
            className="rounded border border-red-200 bg-white px-2 py-1 text-xs text-red-700 hover:bg-red-50"
          >
            Hapus
          </button>
        </td>
      </tr>

      {open && (
        <tr>
          <td colSpan={7} className="border border-slate-300 bg-slate-50 p-2">
            <table className="w-full border-collapse text-sm">
              <caption className="sr-only">Spreading reasuransi {label}</caption>
              <thead>
                <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-500">
                  <th scope="col" className="py-1 pr-2 font-medium">Treaty</th>
                  <th scope="col" className="py-1 pr-2 font-medium">Nama</th>
                  <th scope="col" className="py-1 pr-2 font-medium">Share %</th>
                  {/* PA: spreading tidak dapat dihapus maupun ditambah (Work Owner 2026-10-09). */}
                  {!pa && <th scope="col" className="py-1 font-medium">Hapus</th>}
                </tr>
              </thead>
              <tbody>
                {spreading.fields.map((f, n) => (
                  <tr key={f.id} className="border-b border-slate-100">
                    <td className="py-1 pr-2">
                      <input className="w-24 rounded border border-slate-300 px-2 py-1"
                        aria-label="Jenis treaty" {...register(`${nama}.spreading.${n}.jenis_treaty`)} />
                    </td>
                    <td className="py-1 pr-2">
                      <input className="w-full rounded border border-slate-300 px-2 py-1"
                        aria-label="Nama treaty" {...register(`${nama}.spreading.${n}.nama`)} />
                    </td>
                    <td className="py-1 pr-2">
                      <input className="w-28 rounded border border-slate-300 px-2 py-1" inputMode="decimal"
                        aria-label="Share persen" {...register(`${nama}.spreading.${n}.share`)} />
                    </td>
                    {/* Objek Fac Offer tidak ditampilkan — tidak digunakan (Work Owner
                        2026-10-08). Nilai tersimpannya tetap terkirim apa adanya. */}
                    {!pa && (
                      <td className="py-1">
                        <button type="button" onClick={() => spreading.remove(n)}
                          className="rounded border border-red-200 px-2 py-1 text-xs text-red-700 hover:bg-red-50">
                          Hapus
                        </button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>

            {!pa && (
              <button
                type="button"
                onClick={() => spreading.append({ jenis_treaty: '', nama: '', share: '', objek_fac_offer: '', dihapus: false })}
                className="mt-2 rounded border border-slate-300 bg-white px-3 py-1 text-sm text-slate-700 hover:bg-slate-100"
              >
                Tambah spreading
              </button>
            )}
          </td>
        </tr>
      )}
    </>
  )
}

export function causeSelectOptions(options: CauseOfLossOption[], current: string | undefined) {
  const list = options.map((o) => ({ value: o.id, label: o.nama }))
  if (current && !options.some((o) => o.id === current)) list.unshift({ value: current, label: current })
  return list
}

/**
 * Ringkasan total share, ditampilkan sebelum petugas menekan Simpan.
 *
 * Ia BUKAN validasi — penolakan tetap milik server. Ia perhitungan yang sama yang sudah
 * ada di layar, ditunjukkan lebih awal, supaya petugas tidak perlu menekan Simpan untuk
 * tahu totalnya belum 100%.
 *
 * Totalnya PER JAMINAN, sama dengan server (`InputRegister_act` 37.3.1 mereset total di
 * dalam loop coverage): dua jaminan masing-masing 100% sudah benar.
 *
 * Cukup SATU objek terisi — punya coverage dan setiap coverage-nya 100% (Work Owner,
 * 2026-10-07). Jaminan objek lain yang belum 100% karena itu ditampilkan sebagai keterangan,
 * bukan galat; merah hanya bila belum ada satu objek pun yang terisi.
 */
function SpreadingSummary({ values }: { values: InsuredItemInput[] | undefined }) {
  if (!values || values.length === 0) return null

  let count = 0
  let totalTSI = 0
  let filled = 0
  const offside: string[] = []

  values.forEach((o, i) => {
    const coverages = o.coverage ?? []
    let complete = coverages.length > 0
    coverages.forEach((c, j) => {
      totalTSI += rupiahToCents(c.tsi || '0') || 0
      const totalE4 = coverageShareE4(c)
      count += (c.spreading ?? []).filter((s) => !s.dihapus).length
      if (totalE4 < 999_999 || totalE4 > 1_000_001) {
        complete = false
        offside.push(`Objek ${i + 1} · ${c.id || `jaminan ${j + 1}`}: ${formatPercent(totalE4)}`)
      }
    })
    if (complete) filled++
  })

  return (
    <section className="rounded border border-slate-200 p-4 text-sm">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">Ringkasan</h2>
      <dl className="mt-3 grid gap-x-8 gap-y-2 sm:grid-cols-3">
        <Row label="Baris spreading" value={String(count)} />
        <Row label="Total TSI" value={formatRupiah(totalTSI)} />
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Objek terisi
          </dt>
          <dd className={filled === 0 ? 'text-sm font-medium text-red-700' : 'text-sm text-slate-900'}>
            {filled} dari {values.length}
            {filled === 0 ? ' — minimal satu objek harus terisi' : ''}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Total share per jaminan
          </dt>
          {offside.length === 0 ? (
            <dd className="text-sm text-slate-900">100% di setiap jaminan</dd>
          ) : (
            offside.map((line) => (
              <dd
                key={line}
                className={filled === 0 ? 'text-sm font-medium text-red-700' : 'text-sm text-amber-700'}
              >
                {line} — belum 100%{filled > 0 ? ' (objek ini tidak wajib)' : ''}
              </dd>
            ))
          )}
        </div>
      </dl>
    </section>
  )
}

/** Total share satu jaminan dalam persen × 10.000, baris bertanda hapus tidak dihitung. */
export function coverageShareE4(c: CoverageInput): number {
  let total = 0
  for (const s of c.spreading ?? []) {
    if (s.dihapus) continue
    const num = Number((s.share || '0').replace(',', '.'))
    if (!Number.isNaN(num)) total += Math.round(num * 10_000)
  }
  return total
}

// ── Terjemahan antara bentuk layar dan bentuk API ──────────────────────────────────

const EMPTY_AREA: Area = {
  negara: '',
  negara_id: '',
  provinsi: '',
  provinsi_id: '',
  kota: '',
  kota_id: '',
  kabupaten: '',
  kabupaten_id: '',
  kelurahan: '',
  kelurahan_id: '',
  kode_pos: '',
}

function fromClaim(klaim: Claim): RegisterFormValues {
  return {
    // Klaim yang belum punya Jenis Laporan dibuka dengan Direct, pilihan pertama dropdown.
    jenis_laporan: klaim.jenis_laporan || REPORT_TYPE_DIRECT,
    tanggal_kejadian: klaim.tanggal_kejadian,
    tanggal_lapor: klaim.tanggal_lapor,
    tanggal_terima_dokumen: klaim.tanggal_terima_dokumen,
    rawat_inap: !!klaim.tanggal_keluar_rawat_inap,
    tanggal_keluar_rawat_inap: klaim.tanggal_keluar_rawat_inap ?? '',
    lokasi: klaim.lokasi,
    kronologi: klaim.kronologi,
    pelapor_nama: klaim.pelapor.nama,
    pelapor_telepon: klaim.pelapor.telepon,
    pelapor_email: klaim.pelapor.email,
    pelapor_alamat: klaim.pelapor.alamat,
    pelapor_hubungan: klaim.pelapor.hubungan ? String(klaim.pelapor.hubungan) : '',
    pelapor_hubungan_lainnya: klaim.pelapor.hubungan_lainnya,
    nilai_estimasi: centsToRupiah(klaim.nilai_estimasi_sen),
    // Bawaan = mata uang polis (`.ClaimData.Currency` default `pyWorkPage.Policy.Currency`).
    // Polis yang tidak membawa mata uang jatuh ke IDR dalam bentuk KODE (10026): kurs di
    // M_CURRENCYSTANDARD dikunci kode, dan simbol "IDR" dulu membuat kursnya tidak ditemukan.
    mata_uang: klaim.mata_uang || klaim.polis.mata_uang || CURRENCY_IDR,
    nomor_slik: klaim.nomor_slik,
    user_teknis: klaim.user_teknis,
    rcv_id: klaim.rcv_id,
    ex_gratia: klaim.ex_gratia ? 'YES' : 'NO',
    wilayah: { ...EMPTY_AREA, ...(klaim.wilayah ?? {}) },
    // NORMAL adalah bawaan layar Pega (pyDefaultValue 1).
    prinsip_mengenal_nasabah: klaim.prinsip_mengenal_nasabah || CustomerPrinciple.Normal,
    komentar_suspicious: klaim.komentar_suspicious ?? '',
    email_lod: klaim.email_lod ?? '',
    rekomendasi: klaim.rekomendasi ?? '',
    subjek_email: klaim.subjek_email ?? '',
    status_salvage: klaim.status_salvage ?? '',
    status_pucl: klaim.status_pucl ? String(klaim.status_pucl) : '0',
    transfer_compliance: klaim.transfer_compliance,
    objek: klaim.objek.map((o) => ({
      id: o.id,
      nama: o.nama,
      lokasi: o.lokasi,
      coverage: o.coverage.map((c) => ({
        id: c.id,
        nama: c.nama ?? '',
        penyebab_kerugian: c.penyebab_kerugian,
        tsi: centsToRupiah(c.tsi_sen),
        spreading: c.spreading.map((s) => ({
          jenis_treaty: s.jenis_treaty,
          nama: s.nama,
          share: (s.share / 10_000).toString(),
          objek_fac_offer: s.objek_fac_offer,
          dihapus: s.dihapus,
        })),
      })),
    })),
  }
}

/** No KTP dan Pengkinian Data yang tersimpan pada klaim. */
function insuredUpdateOf(klaim: Claim) {
  return {
    pengkinian_no_ktp: klaim.pengkinian_no_ktp ?? '',
    pengkinian_no_hp: klaim.pengkinian_no_hp ?? '',
    pengkinian_email: klaim.pengkinian_email ?? '',
  }
}

function toRequest(content: RegisterFormValues, taskID: string, kembali: boolean): RegisterRequest {
  return {
    tugas_id: taskID,
    tanggal_kejadian: content.tanggal_kejadian,
    tanggal_lapor: content.tanggal_lapor,
    tanggal_terima_dokumen: content.tanggal_terima_dokumen,
    // Rawat inap tidak dicentang: tanggal keluar dikosongkan.
    tanggal_keluar_rawat_inap: content.rawat_inap ? content.tanggal_keluar_rawat_inap : '',
    lokasi: content.lokasi,
    kronologi: content.kronologi,
    pelapor: {
      nama: content.pelapor_nama,
      telepon: content.pelapor_telepon,
      email: content.pelapor_email,
      alamat: content.pelapor_alamat,
      hubungan: Number(content.pelapor_hubungan) || 0,
      hubungan_lainnya: content.pelapor_hubungan_lainnya,
    },
    nilai_estimasi_sen: rupiahToCents(content.nilai_estimasi) || 0,
    mata_uang: content.mata_uang,
    nomor_slik: content.nomor_slik,
    ex_gratia: content.ex_gratia === 'YES',
    wilayah: content.wilayah,
    prinsip_mengenal_nasabah: content.prinsip_mengenal_nasabah,
    // Komentar hanya bermakna bila SUSPICIOUS; isian yang tersembunyi tidak dikirim.
    komentar_suspicious:
      content.prinsip_mengenal_nasabah === CustomerPrinciple.Suspicious ? content.komentar_suspicious : '',
    // Isian yang tersembunyi menurut lini tidak dikirim: EmailLOD hanya PA, Remarks
    // Recommendation dan Subject Email hanya selain PA (kondisi InputRegisterDetail2_sect).
    email_lod: content.email_lod,
    jenis_laporan: content.jenis_laporan,
    rekomendasi: content.rekomendasi,
    subjek_email: content.subjek_email,
    status_salvage: content.status_salvage,
    user_teknis: content.user_teknis,
    rcv_id: content.rcv_id,
    status_pucl: Number(content.status_pucl) || 0,
    transfer_compliance: content.transfer_compliance,
    kembali,
    objek: (content.objek ?? []).map((o) => ({
      id: o.id,
      nama: o.nama,
      lokasi: o.lokasi,
      coverage: (o.coverage ?? []).map((c) => ({
        id: c.id,
        nama: c.nama,
        penyebab_kerugian: c.penyebab_kerugian,
        tsi_sen: rupiahToCents(c.tsi) || 0,
        spreading: (c.spreading ?? []).map((s) => ({
          jenis_treaty: s.jenis_treaty,
          nama: s.nama,
          // Share dikirim sebagai persen dikali 10.000 — presisi yang dipakai server
          // untuk memvalidasi totalnya.
          share: Math.round((Number((s.share || '0').replace(',', '.')) || 0) * 10_000),
          dihapus: s.dihapus,
          objek_fac_offer: s.objek_fac_offer,
        })),
      })),
    })),
  }
}

function errorMessage(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
