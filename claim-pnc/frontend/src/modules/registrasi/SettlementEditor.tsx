import { useEffect, useRef, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { centsToRupiah, rupiahToCents } from '@/components/format'

import { AcceptanceNumber } from './AcceptanceNumber'
import { SettlementHistory } from './SettlementHistory'
import { useAddSettlement, usePreviewSettlement, useUpdateSettlement, violationsFrom } from './api'
import {
  PaymentType,
  RiskType,
  type CurrencyOption,
  type Settlement,
  type SettlementPreviewResponse,
  type SettlementRequest,
  type Spreading,
} from './types'

/**
 * Baris isian Adjustment BARU di dalam grid Adjustment & Akseptasi — muncul di bawah baris
 * yang ada saat tombol Tambah ditekan, seperti tambah-baris grid Pega (bukan jendela baru).
 *
 * # Susunan dari `Section/InputAdjustment_sect.xml`
 *
 * Dua kolom. Kiri: Mata Uang · Tipe Pembayaran* · Total Klaim* · Nilai Pengajuan
 * Tertanggung* · Tipe Resiko Sendiri* · Persen Resiko Sendiri (%) · Nilai Resiko Sendiri* ·
 * Nilai Nett Pembayaran · Status Persetujuan LOD · Tanggal Transfer LOD. Kanan: Nilai Dalam
 * IDR · Nilai Estimasi · Lack Of Document (%) · Nilai Salvage A · Nilai Salvage B · Nilai
 * Interim · Share ASM (%) · Nilai Yang Dibayarkan ASM · Status Lunas · Tanggal Bayar Kasir.
 * Di bawahnya tabel Tipe Treaty / Pembagian Persentase (`.SpreadingList`). Fee adjuster (tipe
 * 4) memakai Professional Fee · Survey Expenses · Tipe VAT · VAT (%) · Nilai Adjuster Fee.
 *
 * Kondisi tampilnya mengikuti section: Total Klaim sampai Nilai Resiko Sendiri untuk tipe
 * selain 3/4/7; LOC dan Salvage A/B hanya di luar PA/Travel (`!IsPATRAVEL`); Nilai Interim,
 * Status Lunas, dan kedua tanggal hanya Non-MBU; Status Persetujuan LOD di luar Travel; Mata
 * Uang hanya-baca untuk Travel.
 *
 * # Hitungan
 *
 * Seluruh nilai tampilan (Nilai Dalam IDR, Nilai Resiko Sendiri tipe 1/2, Nilai Interim,
 * Nilai Nett Pembayaran, Nilai Yang Dibayarkan ASM) dihitung SERVER lewat rute pratinjau
 * setiap isian berubah — padanan Pega yang menjalankan `SetNilaiResikoSendiri` pada perubahan
 * field. Layar tidak menghitung sendiri.
 */

const PAYMENT_OPTIONS: { value: string; label: string; disabled?: boolean }[] = [
  { value: PaymentType.Final, label: 'Final' },
  { value: PaymentType.Interim, label: 'Interim' },
  { value: PaymentType.Salvage, label: 'Salvage (dari modul Salvage)', disabled: true },
  { value: PaymentType.AdjusterFee, label: 'Adjuster Fee' },
  { value: PaymentType.Adjustment, label: 'Adjustment' },
  { value: PaymentType.Reject, label: 'Tolak Klaim (belum tersedia)', disabled: true },
]

// Label pilihan 1 dan 2 tidak ada di export (daftarnya dari definisi properti); "Lainnya"
// dari tangkapan layar Pega.
const RISK_OPTIONS = [
  { value: RiskType.OfClaim, label: '% dari Nilai Klaim' },
  { value: RiskType.OfTSI, label: '% dari TSI' },
  { value: RiskType.Other, label: 'Lainnya' },
]

const VAT_OPTIONS = [
  { value: '1', label: 'Dari Professional Fee' },
  { value: '2', label: 'Dari Professional Fee + Survey Expenses' },
]

const EMPTY = '——'

/** Hapus baris tersimpan butuh soft delete, sedangkan T_CLAIM_ADJUSTMENT belum punya DIHAPUS_PADA. */
const DELETE_NEEDS_COLUMN = 'Deleting a saved line needs column DIHAPUS_PADA on POOLDATA.T_CLAIM_ADJUSTMENT (DBA).'
const PREVIEW_DELAY_MS = 300

/** Persen × 10.000 menjadi teks yang dapat disunting kembali (`105000` → `10,5`). */
/**
 * Lama Hari Rawat Inap (`.InpatientDay`) — `ValidationAdjustment` langkah 95:
 * `@DateTimeDifference(DateOfLoss, TanggalSelesaiRawatInap, 'D')`, yaitu selisih hari kalender
 * Tanggal Kejadian / Tanggal Masuk Rawat Inap sampai Tanggal Keluar Rawat Inap. Null bila salah
 * satunya kosong atau tanggal keluar mendahului tanggal masuk.
 *
 * Batas `.InpatientDayMax` jaminan (langkah 94–95: hari di atas batas diganti batasnya) tidak
 * diterapkan — batas per jaminan itu (SumOfDay) belum ada di data jaminan aplikasi ini.
 */
export function inpatientDays(admission: string | undefined, discharge: string | undefined): number | null {
  const day = (iso: string | undefined) => {
    const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso ?? '')
    return m ? Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) / 86_400_000 : null
  }
  const from = day(admission)
  const to = day(discharge)
  if (from === null || to === null || to < from) return null
  return to - from
}

function percentText(e4: number): string {
  return e4 ? String(e4 / 10_000).replace('.', ',') : ''
}

/** Persen yang diketik (`10,5`) menjadi persen × 10.000 (`105000`). */
function percentE4(text: string): number {
  const clean = text.trim().replace(',', '.')
  if (clean === '') return 0
  const n = Number(clean)
  return Number.isNaN(n) ? Number.NaN : Math.round(n * 10_000)
}

const amount = new Intl.NumberFormat('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const rate = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 4 })
const percent = new Intl.NumberFormat('id-ID', { minimumFractionDigits: 4, maximumFractionDigits: 4 })

type Values = {
  tipe_pembayaran: string
  mata_uang: string
  total_klaim: string
  nilai_pengajuan: string
  /** PA: .ProposeValueTertanggung — tampil, tetapi tidak punya kolom T_CLAIM_ADJUSTMENT. */
  nilai_pengajuan_tertanggung: string
  /** .NoInvoice (tipe 4) — tampil, tetapi tidak punya kolom T_CLAIM_ADJUSTMENT. */
  no_invoice: string
  loc: string
  salvage_a: string
  salvage_b: string
  tipe_resiko: string
  persen_resiko: string
  nilai_resiko: string
  professional_fee: string
  survey_expenses: string
  vat: string
  tipe_vat: string
}

export function SettlementEditor({
  claimID,
  taskID,
  object,
  coverage,
  policyCurrency,
  currencies,
  travel,
  nonMBU,
  pa = false,
  exGratia = false,
  inpatientDays = null,
  analyst = false,
  analystTransfer = false,
  existing,
  onCreated,
  onClose,
}: {
  claimID: string
  taskID: string
  object: number
  coverage: number
  policyCurrency: string
  currencies: CurrencyOption[]
  /** When IsTravel. */
  travel: boolean
  /** When IsNonMbu. */
  nonMBU: boolean
  /** When IsPA. */
  pa?: boolean
  /** `pyWorkPage.ClaimData.ExGratia = 1` — baris adjustment mewarisinya (`.ExGratia`). */
  exGratia?: boolean
  /** Lama Hari Rawat Inap klaim PA (lihat inpatientDays); null bila tidak rawat inap. */
  inpatientDays?: number | null
  /** When `IsAnalisator` — pemanggil anggota grup Analyst. */
  analyst?: boolean
  /**
   * When `isAnalistorTransfer`: lini PA dan jaminan PHK (`ValidationAdjustment` step 30) atau jaminan yang sudah
   * ditransfer ke Analyst (`setTicketToAnalyst`). Memunculkan "Total Klaim" (`.ProposeAdjustmentValue`).
   */
  analystTransfer?: boolean
  /** Baris yang sudah tersimpan (berbasis 0) — isiannya dimuat dan perubahannya disimpan ulang. */
  existing?: { index: number; line: Settlement }
  /** Baris baru pertama kali tersimpan; indeksnya (berbasis 0) di jaminan ini. */
  onCreated?: (index: number) => void
  /** Hapus pada baris yang belum tersimpan. */
  onClose: () => void
}) {
  const add = useAddSettlement(claimID)
  const update = useUpdateSettlement(claimID)
  const preview = usePreviewSettlement(claimID)
  const old = existing?.line
  const [computed, setComputed] = useState<SettlementPreviewResponse | null>(null)
  const [values, setValues] = useState<Values>({
    // PA: baris baru bertipe Interim (`ValidationAdjustment` step 19: PaymentType := 2).
    tipe_pembayaran: old ? old.tipe_pembayaran : pa ? PaymentType.Interim : '',
    mata_uang: old ? old.mata_uang : policyCurrency,
    total_klaim: old ? centsToRupiah(old.nilai_propose_sen) : '',
    nilai_pengajuan: old ? centsToRupiah(old.nilai_pengajuan_sen) : '',
    nilai_pengajuan_tertanggung: old ? centsToRupiah(old.nilai_pengajuan_tertanggung_sen ?? 0) : '',
    no_invoice: '',
    loc: old ? percentText(old.loc) : '',
    salvage_a: old ? centsToRupiah(old.nilai_salvage_sen) : '',
    salvage_b: old ? centsToRupiah(old.nilai_salvage_b_sen) : '',
    tipe_resiko: old ? old.tipe_resiko : '',
    persen_resiko: old ? percentText(old.persen_resiko) : '',
    nilai_resiko: old && old.tipe_resiko === RiskType.Other ? centsToRupiah(old.nilai_resiko_sen) : '',
    professional_fee: '',
    survey_expenses: '',
    vat: '',
    tipe_vat: '1',
  })
  const set = (key: keyof Values) => (value: string) => setValues((v) => ({ ...v, [key]: value }))
  // Pega menjalankan SetNilaiResikoSendiri (hitung, periksa, simpan) pada perubahan isian: di sini saat
  // pilihan berubah dan saat isian teks ditinggalkan.
  const [commitPending, setCommitPending] = useState(false)
  const commitSoon = () => setCommitPending(true)
  const choose = (key: keyof Values) => (value: string) => {
    set(key)(value)
    commitSoon()
  }

  const pt = values.tipe_pembayaran
  const proposeBased = pt === PaymentType.Final || pt === PaymentType.Interim || pt === PaymentType.Adjustment
  const fee = pt === PaymentType.AdjusterFee
  const withPercent = values.tipe_resiko === RiskType.OfClaim || values.tipe_resiko === RiskType.OfTSI
  const manualRisk = values.tipe_resiko === RiskType.Other

  const numbers = {
    // PA: Total Klaim (.ProposeAdjustmentValue) hanya ada pada baris isAnalistorTransfer.
    nilai_propose_sen: proposeBased && (!pa || analystTransfer) ? rupiahToCents(values.total_klaim) : 0,
    nilai_pengajuan_sen: proposeBased ? rupiahToCents(values.nilai_pengajuan) : 0,
    // Nilai Pengajuan Tertanggung PA — PROPOSE_VALUE_TERTANGGUNG (Work Owner 2026-10-09).
    nilai_pengajuan_tertanggung_sen: proposeBased && pa ? rupiahToCents(values.nilai_pengajuan_tertanggung) : 0,
    loc: proposeBased && !travel && !pa ? percentE4(values.loc) : 0,
    nilai_salvage_sen: proposeBased && !travel && !pa ? rupiahToCents(values.salvage_a) : 0,
    nilai_salvage_b_sen: proposeBased && !travel && !pa ? rupiahToCents(values.salvage_b) : 0,
    persen_resiko: proposeBased && withPercent ? percentE4(values.persen_resiko) : 0,
    nilai_resiko_sen: proposeBased && manualRisk ? rupiahToCents(values.nilai_resiko) : 0,
    professional_fee_sen: fee ? rupiahToCents(values.professional_fee) : 0,
    survey_expenses_sen: fee ? rupiahToCents(values.survey_expenses) : 0,
    vat: fee ? percentE4(values.vat) : 0,
  }
  const malformed = Object.values(numbers).some((n) => Number.isNaN(n))

  const request: SettlementRequest = {
    tugas_id: taskID,
    objek: object,
    jaminan: coverage,
    tipe_pembayaran: pt,
    mata_uang: values.mata_uang,
    tipe_resiko: proposeBased ? values.tipe_resiko : '',
    tipe_vat: fee ? values.tipe_vat : '',
    kronologi: '',
    catatan: '',
    ...numbers,
  }
  const requestKey = JSON.stringify(request)

  // Pratinjau dihitung ulang setiap isian berubah, ditunda sebentar supaya tidak satu
  // permintaan per ketukan. Jawaban yang datang untuk isian lama diabaikan.
  const latest = useRef('')
  const { mutate: runPreview } = preview
  useEffect(() => {
    if (malformed) return
    latest.current = requestKey
    const timer = window.setTimeout(() => {
      runPreview(JSON.parse(requestKey) as SettlementRequest, {
        onSuccess: (result) => {
          if (latest.current === requestKey) setComputed(result)
        },
      })
    }, PREVIEW_DELAY_MS)
    return () => window.clearTimeout(timer)
  }, [requestKey, malformed, runPreview])

  // Baris tersimpan: indeksnya (berbasis 0). Baris baru: null sampai simpan pertama berhasil.
  const savedIndex = existing?.index ?? null
  const lastSaved = useRef(existing ? requestKey : '')
  const { mutate: runAdd } = add
  const { mutate: runUpdate } = update
  useEffect(() => {
    if (!commitPending) return
    setCommitPending(false)
    if (malformed || requestKey === lastSaved.current) return
    const body = JSON.parse(requestKey) as SettlementRequest
    if (savedIndex === null) {
      runAdd(body, {
        onSuccess: (result) => {
          lastSaved.current = requestKey
          const count = result?.klaim?.objek?.[object - 1]?.coverage[coverage - 1]?.adjustment?.length ?? 0
          if (count > 0) onCreated?.(count - 1)
        },
      })
    } else {
      runUpdate({ ...body, adjustment: savedIndex + 1 }, { onSuccess: () => (lastSaved.current = requestKey) })
    }
  }, [commitPending, requestKey, malformed, savedIndex, runAdd, runUpdate, object, coverage, onCreated])

  const line = computed?.adjustment
  const saveError = savedIndex === null ? add.error : update.error
  const saving = add.isPending || update.isPending
  const violations = violationsFrom(saveError)
  const failure = saveError ?? preview.error
  const money = (sen: number | undefined) => (line && sen !== undefined ? amount.format(sen / 100) : EMPTY)

  return (
    <div
      role="group"
      aria-label={existing ? `Ubah adjustment ${existing.index + 1}` : 'Adjustment baru'}
      className="rounded border border-blue-200 bg-white p-4 text-sm"
    >
      <div className="grid gap-x-8 gap-y-3 md:grid-cols-2">
        {/* ── Kolom kiri ─────────────────────────────────────────────────────── */}
        <div className="space-y-3">
          <Select
            label="Mata Uang"
            value={values.mata_uang}
            onChange={choose('mata_uang')}
            disabled={travel}
            options={currencies.map((c) => ({ value: c.id, label: c.nama }))}
          />
          {/* Tipe Pembayaran IsPA: baca saja bila !IsAnalisator. */}
          {pa && !analyst ? (
            <Display label="Tipe Pembayaran" required>
              {PAYMENT_OPTIONS.find((o) => o.value === pt)?.label ?? EMPTY}
            </Display>
          ) : (
            <Select label="Tipe Pembayaran" required value={pt} onChange={choose('tipe_pembayaran')} options={PAYMENT_OPTIONS} placeholder="--- PILIH ---" />
          )}
          {/* NoInvoice: tipe 4. */}
          {fee && (
            <Text onBlur={commitSoon}
              label="NoInvoice"
              value={values.no_invoice}
              onChange={set('no_invoice')}
              note="Belum tersimpan: POOLDATA.T_CLAIM_ADJUSTMENT belum punya kolom No Invoice."
            />
          )}

          {proposeBased && pa && (
            <>
              {/* Kontainer IsPA, tipe selain 3/4. */}
              <Money onBlur={commitSoon}
                label="Nilai Pengajuan Tertanggung"
                value={values.nilai_pengajuan_tertanggung}
                onChange={set('nilai_pengajuan_tertanggung')}
              />
              <Money onBlur={commitSoon} label="Nilai Pengajuan" required value={values.nilai_pengajuan} onChange={set('nilai_pengajuan')} disabled={analyst} />
              {analystTransfer && (
                <Text onBlur={commitSoon} label="Total Klaim" required value={values.total_klaim} onChange={set('total_klaim')} />
              )}
            </>
          )}
          {proposeBased && !pa && (
            <>
              <Text onBlur={commitSoon} label="Total Klaim" required value={values.total_klaim} onChange={set('total_klaim')} />
              <Money onBlur={commitSoon} label="Nilai Pengajuan Tertanggung" required value={values.nilai_pengajuan} onChange={set('nilai_pengajuan')} />
            </>
          )}

          {proposeBased && (
            <>
              <Select label="Tipe Resiko Sendiri" required value={values.tipe_resiko} onChange={choose('tipe_resiko')} options={RISK_OPTIONS} placeholder="--- PILIH ---" />
              {withPercent ? (
                <Text onBlur={commitSoon} label="Persen Resiko Sendiri (%)" value={values.persen_resiko} onChange={set('persen_resiko')} />
              ) : (
                <Display label="Persen Resiko Sendiri (%)">{EMPTY}</Display>
              )}
              {manualRisk ? (
                <Money onBlur={commitSoon} label="Nilai Resiko Sendiri" required value={values.nilai_resiko} onChange={set('nilai_resiko')} />
              ) : (
                <Display label="Nilai Resiko Sendiri">{money(line?.nilai_resiko_sen)}</Display>
              )}
            </>
          )}

          {fee && (
            <>
              <Text onBlur={commitSoon} label="Professional Fee" required value={values.professional_fee} onChange={set('professional_fee')} />
              <Text onBlur={commitSoon} label="Survey Expenses" required value={values.survey_expenses} onChange={set('survey_expenses')} />
              <Select label="Tipe VAT" required value={values.tipe_vat} onChange={choose('tipe_vat')} options={VAT_OPTIONS} />
              <Text onBlur={commitSoon} label="VAT (%)" required value={values.vat} onChange={set('vat')} />
              <Display label="Nilai Adjuster Fee">{money(line?.nilai_asm_sen)}</Display>
            </>
          )}

          {pt !== '' && <Display label="Nilai Nett Pembayaran">{money(line?.nilai_gross_sen)}</Display>}
          {!travel && <Display label="Status Persetujuan LOD">{EMPTY}</Display>}
          {/* Status Aksep Analysator (.AcceptanceAnalystStatus): IsPA, baca saja. */}
          {pa && <Display label="Status Aksep Analysator">{EMPTY}</Display>}
          {/* Ex Gratia (.ExGratia): tampil bila klaim Ex Gratia, baca saja. */}
          {exGratia && <Display label="Ex Gratia">Ya</Display>}
          {nonMBU && (
            <Display label="Tanggal Transfer LOD" small>
              {EMPTY}
            </Display>
          )}
        </div>

        {/* ── Kolom kanan ────────────────────────────────────────────────────── */}
        <div className="space-y-3">
          <Display label="Nilai Dalam IDR">{line ? rate.format(line.kurs_e4 / 10_000) : EMPTY}</Display>
          <Display label="Nilai Estimasi">{money(line?.nilai_estimasi_sen)}</Display>
          {/* Lama hari rawat inap (.InpatientDay): IsPA, baca saja kecuali isAnalistorTransfer. */}
          {pa && <Display label="Lama Hari Rawat Inap">{inpatientDays ?? EMPTY}</Display>}

          {/* LOC, Salvage A, Salvage B: kontainer `!IsPATRAVEL` — tidak untuk PA dan Travel. */}
          {proposeBased && !travel && !pa && (
            <>
              <Text onBlur={commitSoon} label="Lack Of Document (%)" value={values.loc} onChange={set('loc')} />
              <Text onBlur={commitSoon} label="Nilai Salvage A" value={values.salvage_a} onChange={set('salvage_a')} />
              <Text onBlur={commitSoon} label="Nilai Salvage B" value={values.salvage_b} onChange={set('salvage_b')} />
            </>
          )}
          {proposeBased && nonMBU && <Display label="Nilai Interim">{money(line?.nilai_interim_sen)}</Display>}

          <Display label="Share ASM (%)">{line ? percent.format(line.share_asm / 10_000) : EMPTY}</Display>
          {proposeBased && <Display label="Nilai Yang Dibayarkan ASM">{money(line?.nilai_asm_sen)}</Display>}
          {nonMBU && (
            <Display label="Status Lunas" small>
              {EMPTY}
            </Display>
          )}
          {nonMBU && (
            <Display label="Tanggal Bayar Kasir" small>
              {EMPTY}
            </Display>
          )}
        </div>
      </div>

      {/* ── Spreading jaminan (.SpreadingList) ──────────────────────────────────── */}
      {/* Tipe Treaty / Pembagian Persentase: kontainer `.ExGratia = 1`. */}
      {exGratia && computed && computed.spreading.length > 0 && <SpreadingTable spreading={computed.spreading} />}

      {malformed && <p className="mt-3 text-sm text-red-700">Angka tidak valid.</p>}
      {failure && (
        <div className="mt-4">
          <ErrorMessage
            title={saveError ? 'Adjustment belum tersimpan' : 'Nilai belum dapat dihitung'}
            description={
              violations.length > 0
                ? violations.map((v) => v.pesan).join(' ')
                : failure instanceof Error
                  ? failure.message
                  : 'Terjadi kesalahan pada sistem.'
            }
            tone="penolakan"
          />
        </div>
      )}

      {/* Pega tidak punya Simpan/Batal: setiap perubahan isian langsung disimpan. Hapus menghapus baris. */}
      <div className="mt-4 flex items-center justify-end gap-3">
        {saving && <span className="text-xs text-slate-500">Menyimpan…</span>}
        <Button
          tone="halus"
          disabled={saving || savedIndex !== null}
          title={savedIndex !== null ? DELETE_NEEDS_COLUMN : undefined}
          onClick={onClose}
        >
          Hapus
        </Button>
      </div>
    </div>
  )
}

function Caption({ label, required, small }: { label: string; required?: boolean; small?: boolean }) {
  return (
    <span className={small ? 'text-xs font-semibold text-slate-800' : 'font-semibold text-slate-800'}>
      {label}
      {required && <span className="text-orange-600">*</span>}
    </span>
  )
}

function Display({ label, small, required, children }: { label: string; small?: boolean; required?: boolean; children: ReactNode }) {
  return (
    <div>
      <Caption label={label} small={small ?? false} required={required ?? false} />
      <p className="text-slate-700">{children}</p>
    </div>
  )
}

function Select({
  label,
  value,
  onChange,
  options,
  placeholder,
  required,
  disabled,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  options: { value: string; label: string; disabled?: boolean }[]
  placeholder?: string
  required?: boolean
  disabled?: boolean
}) {
  return (
    <label className="block">
      <Caption label={label} required={required ?? false} />
      <select
        aria-label={label}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 block rounded border border-slate-300 px-2 py-1 text-sm disabled:bg-slate-50"
      >
        {placeholder !== undefined && <option value="">{placeholder}</option>}
        {options.map((o) => (
          <option key={o.value} value={o.value} disabled={o.disabled}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  )
}

function Text({
  label,
  value,
  onChange,
  required,
  disabled,
  note,
  onBlur,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  onBlur?: () => void
  required?: boolean
  disabled?: boolean
  /** Keterangan di bawah isian, mis. bila nilainya belum tersimpan. */
  note?: string
}) {
  return (
    <label className="block">
      <Caption label={label} required={required ?? false} />
      <input
        aria-label={label}
        value={value}
        inputMode="decimal"
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        onBlur={onBlur}
        className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm disabled:bg-slate-50"
      />
      {note && <span className="mt-1 block text-xs text-slate-500">{note}</span>}
    </label>
  )
}

/**
 * Isian uang yang memasang pemisah ribuan saat diketik — 3000000 tampil 3.000.000 — dan
 * melengkapi dua desimal saat ditinggalkan: 3.000.000,00 (Work Owner 2026-10-09). Nilainya
 * tetap teks berformat Indonesia yang dibaca rupiahToCents.
 */
function Money(props: Parameters<typeof Text>[0]) {
  return (
    <Text
      {...props}
      value={groupMoney(props.value)}
      onChange={(v) => props.onChange(groupMoney(v))}
      onBlur={() => {
        props.onChange(finishMoney(props.value))
        props.onBlur?.()
      }}
    />
  )
}

/**
 * groupMoney merapikan ketikan uang: hanya angka dan satu koma desimal (paling banyak dua
 * angka di belakangnya), dengan titik pemisah ribuan. Titik yang diketik pengguna dibuang —
 * ia pemisah ribuan, bukan desimal.
 */
export function groupMoney(text: string): string {
  const clean = text.replace(/[^\d,]/g, '')
  if (clean === '') return ''
  const comma = clean.indexOf(',')
  const whole = (comma < 0 ? clean : clean.slice(0, comma)).replace(/^0+(?=\d)/, '')
  const grouped = (whole === '' ? '0' : whole).replace(/\B(?=(\d{3})+(?!\d))/g, '.')
  if (comma < 0) return grouped
  return `${grouped},${clean.slice(comma + 1).replace(/,/g, '').slice(0, 2)}`
}

/** finishMoney melengkapi dua desimal: 3.000.000 menjadi 3.000.000,00. Kosong tetap kosong. */
export function finishMoney(text: string): string {
  const grouped = groupMoney(text)
  if (grouped === '') return ''
  const [whole, fraction = ''] = grouped.split(',')
  return `${whole},${fraction.padEnd(2, '0')}`
}

/** Label pilihan Persetujuan Tertanggung pada form AcceptationLOD. */
const LOD_STATUS: Record<string, string> = { '1': 'Setuju', '0': 'Tidak Setuju' }

const RISK_NAMES: Record<string, string> = Object.fromEntries(RISK_OPTIONS.map((o) => [o.value, o.label]))

/**
 * Tampilan baca satu baris Adjustment yang sudah tersimpan — dibuka dengan mengklik
 * barisnya di grid. Susunannya sama dengan isian (InputAdjustment_sect).
 *
 * Nilai Salvage B, Nilai Interim, dan komponen fee adjuster tidak punya kolom di
 * T_CLAIM_ADJUSTMENT; yang tersimpan hanya Nilai Nett Pembayaran hasilnya, sehingga
 * ketiganya tampil —— pada baris yang dimuat ulang.
 *
 * Nomor Akseptasi beserta tombol PRINT dan Transfer Kasir tampil di kolom kanan, seperti
 * InputAdjustment_sect (lihat AcceptanceNumber). Status Persetujuan LOD dan Case ID Kasir dibaca
 * dari STATUSAKSEPTASILOD dan IDCHASIER.
 */
export function SettlementDetail({
  line,
  currencyName,
  estimation,
  spreading,
  travel,
  nonMBU,
  pa = false,
  exGratia = false,
  inpatientDays = null,
  address,
}: {
  line: Settlement
  currencyName: string
  /** Nilai Estimasi jaminan (estimasi klaim), dalam sen. */
  estimation: number
  spreading: Spreading[]
  travel: boolean
  nonMBU: boolean
  /** When IsPA. */
  pa?: boolean
  /** Klaim Ex Gratia — tabel treaty hanya tampil di kontainer `.ExGratia = 1`. */
  exGratia?: boolean
  /** Lama Hari Rawat Inap klaim PA (lihat inpatientDays); null bila tidak rawat inap. */
  inpatientDays?: number | null
  /** Tugas dan letak baris ini (berbasis 1) — untuk tombol PRINT dan Transfer Kasir. */
  address?: { claimID: string; taskID: string; object: number; coverage: number; adjustment: number }
}) {
  const pt = line.tipe_pembayaran
  const proposeBased = pt === PaymentType.Final || pt === PaymentType.Interim || pt === PaymentType.Adjustment
  const fee = pt === PaymentType.AdjusterFee
  const money = (sen: number) => amount.format(sen / 100)
  const optional = (sen: number) => (sen ? money(sen) : EMPTY)

  return (
    <div role="group" aria-label={`Detail adjustment ${line.nama_tipe_pembayaran}`} className="rounded border border-slate-200 bg-slate-50 p-4 text-sm">
      <div className="grid gap-x-8 gap-y-3 md:grid-cols-2">
        <div className="space-y-3">
          <Display label="Mata Uang">{currencyName}</Display>
          <Display label="Tipe Pembayaran">{line.nama_tipe_pembayaran}</Display>
          {proposeBased && (
            <>
              {pa ? (
                <>
                  {/* Kontainer IsPA: ProposeValueTertanggung (PROPOSE_VALUE_TERTANGGUNG); ProposeValue; ProposeAdjustmentValue bila diisi. */}
                  <Display label="Nilai Pengajuan Tertanggung">{optional(line.nilai_pengajuan_tertanggung_sen ?? 0)}</Display>
                  <Display label="Nilai Pengajuan">{money(line.nilai_pengajuan_sen)}</Display>
                  {line.nilai_propose_sen > 0 && <Display label="Total Klaim">{money(line.nilai_propose_sen)}</Display>}
                </>
              ) : (
                <>
                  <Display label="Total Klaim">{money(line.nilai_propose_sen)}</Display>
                  <Display label="Nilai Pengajuan Tertanggung">{money(line.nilai_pengajuan_sen)}</Display>
                </>
              )}
              <Display label="Tipe Resiko Sendiri">{RISK_NAMES[line.tipe_resiko] ?? EMPTY}</Display>
              <Display label="Persen Resiko Sendiri (%)">{line.persen_resiko ? percent.format(line.persen_resiko / 10_000) : EMPTY}</Display>
              <Display label="Nilai Resiko Sendiri">{money(line.nilai_resiko_sen)}</Display>
            </>
          )}
          {fee && <Display label="Nilai Adjuster Fee">{money(line.nilai_asm_sen)}</Display>}
          <Display label="Nilai Nett Pembayaran">{money(line.nilai_gross_sen)}</Display>
          {!travel && <Display label="Status Persetujuan LOD">{LOD_STATUS[line.status_akseptasi_lod ?? ''] ?? EMPTY}</Display>}
          {pa && <Display label="Status Aksep Analysator">{EMPTY}</Display>}
          {exGratia && <Display label="Ex Gratia">Ya</Display>}
          {nonMBU && (
            <Display label="Tanggal Transfer LOD" small>
              {EMPTY}
            </Display>
          )}
        </div>
        <div className="space-y-3">
          <Display label="Nilai Dalam IDR">{rate.format(line.kurs_e4 / 10_000)}</Display>
          <Display label="Nilai Estimasi">{money(line.nilai_estimasi_sen || estimation)}</Display>
          {pa && <Display label="Lama Hari Rawat Inap">{inpatientDays ?? EMPTY}</Display>}
          {proposeBased && !travel && !pa && (
            <>
              <Display label="Lack Of Document (%)">{line.loc ? percent.format(line.loc / 10_000) : EMPTY}</Display>
              <Display label="Nilai Salvage A">{optional(line.nilai_salvage_sen)}</Display>
              <Display label="Nilai Salvage B">{optional(line.nilai_salvage_b_sen)}</Display>
            </>
          )}
          {proposeBased && nonMBU && <Display label="Nilai Interim">{optional(line.nilai_interim_sen)}</Display>}
          <Display label="Share ASM (%)">{percent.format(line.share_asm / 10_000)}</Display>
          {proposeBased && <Display label="Nilai Yang Dibayarkan ASM">{money(line.nilai_asm_sen)}</Display>}
          {line.case_id_kasir && <Display label="Case ID Kasir">{line.case_id_kasir}</Display>}
          {nonMBU && (
            <Display label="Status Lunas" small>
              {EMPTY}
            </Display>
          )}
          {nonMBU && (
            <Display label="Tanggal Bayar Kasir" small>
              {EMPTY}
            </Display>
          )}
          {address && (
            <AcceptanceNumber
              claimID={address.claimID}
              taskID={address.taskID}
              object={address.object}
              coverage={address.coverage}
              adjustment={address.adjustment}
              line={line}
            />
          )}
        </div>
      </div>
      {exGratia && spreading.length > 0 && <SpreadingTable spreading={spreading} />}
      {address && (
        <SettlementHistory
          claimID={address.claimID}
          object={address.object}
          coverage={address.coverage}
          adjustment={address.adjustment}
        />
      )}
    </div>
  )
}

function SpreadingTable({ spreading }: { spreading: Spreading[] }) {
  return (
    <table className="mt-5 border-collapse border border-slate-200 bg-white text-xs">
      <caption className="sr-only">Spreading jaminan</caption>
      <thead>
        <tr className="bg-slate-100 text-left text-slate-700">
          <th className="p-2" />
          <th className="p-2">Tipe Treaty</th>
          <th className="p-2 text-right">Pembagian Persentase (%)</th>
        </tr>
      </thead>
      <tbody>
        {spreading.map((s, n) => (
          <tr key={n} className="border-t border-slate-200">
            <td className="p-2">{n + 1}</td>
            <td className="p-2">{s.nama || s.jenis_treaty}</td>
            <td className="p-2 text-right">{rate.format(s.share / 10_000)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
