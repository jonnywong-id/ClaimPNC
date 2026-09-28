import { useEffect, useRef, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { rupiahToCents } from '@/components/format'

import { useAddSettlement, usePreviewSettlement, violationsFrom } from './api'
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
const PREVIEW_DELAY_MS = 300

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
  onClose: () => void
}) {
  const add = useAddSettlement(claimID)
  const preview = usePreviewSettlement(claimID)
  const [computed, setComputed] = useState<SettlementPreviewResponse | null>(null)
  const [values, setValues] = useState<Values>({
    tipe_pembayaran: '',
    mata_uang: policyCurrency,
    total_klaim: '',
    nilai_pengajuan: '',
    loc: '',
    salvage_a: '',
    salvage_b: '',
    tipe_resiko: '',
    persen_resiko: '',
    nilai_resiko: '',
    professional_fee: '',
    survey_expenses: '',
    vat: '',
    tipe_vat: '1',
  })
  const set = (key: keyof Values) => (value: string) => setValues((v) => ({ ...v, [key]: value }))

  const pt = values.tipe_pembayaran
  const proposeBased = pt === PaymentType.Final || pt === PaymentType.Interim || pt === PaymentType.Adjustment
  const fee = pt === PaymentType.AdjusterFee
  const withPercent = values.tipe_resiko === RiskType.OfClaim || values.tipe_resiko === RiskType.OfTSI
  const manualRisk = values.tipe_resiko === RiskType.Other

  const numbers = {
    nilai_propose_sen: proposeBased ? rupiahToCents(values.total_klaim) : 0,
    nilai_pengajuan_sen: proposeBased ? rupiahToCents(values.nilai_pengajuan) : 0,
    loc: proposeBased && !travel ? percentE4(values.loc) : 0,
    nilai_salvage_sen: proposeBased && !travel ? rupiahToCents(values.salvage_a) : 0,
    nilai_salvage_b_sen: proposeBased && !travel ? rupiahToCents(values.salvage_b) : 0,
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

  function submit() {
    add.mutate(request, { onSuccess: onClose })
  }

  const line = computed?.adjustment
  const violations = violationsFrom(add.error)
  const failure = add.error ?? preview.error
  const money = (sen: number | undefined) => (line && sen !== undefined ? amount.format(sen / 100) : EMPTY)

  return (
    <div role="group" aria-label="Adjustment baru" className="rounded border border-blue-200 bg-white p-4 text-sm">
      <div className="grid gap-x-8 gap-y-3 md:grid-cols-2">
        {/* ── Kolom kiri ─────────────────────────────────────────────────────── */}
        <div className="space-y-3">
          <Select
            label="Mata Uang"
            value={values.mata_uang}
            onChange={set('mata_uang')}
            disabled={travel}
            options={currencies.map((c) => ({ value: c.id, label: c.nama }))}
          />
          <Select label="Tipe Pembayaran" required value={pt} onChange={set('tipe_pembayaran')} options={PAYMENT_OPTIONS} placeholder="--- PILIH ---" />

          {proposeBased && (
            <>
              <Text label="Total Klaim" required value={values.total_klaim} onChange={set('total_klaim')} />
              <Text label="Nilai Pengajuan Tertanggung" required value={values.nilai_pengajuan} onChange={set('nilai_pengajuan')} />
              <Select label="Tipe Resiko Sendiri" required value={values.tipe_resiko} onChange={set('tipe_resiko')} options={RISK_OPTIONS} placeholder="--- PILIH ---" />
              {withPercent ? (
                <Text label="Persen Resiko Sendiri (%)" value={values.persen_resiko} onChange={set('persen_resiko')} />
              ) : (
                <Display label="Persen Resiko Sendiri (%)">{EMPTY}</Display>
              )}
              {manualRisk ? (
                <Text label="Nilai Resiko Sendiri" required value={values.nilai_resiko} onChange={set('nilai_resiko')} />
              ) : (
                <Display label="Nilai Resiko Sendiri">{money(line?.nilai_resiko_sen)}</Display>
              )}
            </>
          )}

          {fee && (
            <>
              <Text label="Professional Fee" required value={values.professional_fee} onChange={set('professional_fee')} />
              <Text label="Survey Expenses" required value={values.survey_expenses} onChange={set('survey_expenses')} />
              <Select label="Tipe VAT" required value={values.tipe_vat} onChange={set('tipe_vat')} options={VAT_OPTIONS} />
              <Text label="VAT (%)" required value={values.vat} onChange={set('vat')} />
              <Display label="Nilai Adjuster Fee">{money(line?.nilai_asm_sen)}</Display>
            </>
          )}

          {pt !== '' && <Display label="Nilai Nett Pembayaran">{money(line?.nilai_gross_sen)}</Display>}
          {!travel && <Display label="Status Persetujuan LOD">{EMPTY}</Display>}
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

          {proposeBased && !travel && (
            <>
              <Text label="Lack Of Document (%)" value={values.loc} onChange={set('loc')} />
              <Text label="Nilai Salvage A" value={values.salvage_a} onChange={set('salvage_a')} />
              <Text label="Nilai Salvage B" value={values.salvage_b} onChange={set('salvage_b')} />
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
      {computed && computed.spreading.length > 0 && <SpreadingTable spreading={computed.spreading} />}

      {malformed && <p className="mt-3 text-sm text-red-700">Angka tidak valid.</p>}
      {failure && (
        <div className="mt-4">
          <ErrorMessage
            title={add.error ? 'Adjustment belum dapat ditambahkan' : 'Nilai belum dapat dihitung'}
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

      <div className="mt-4 flex justify-end gap-3">
        <Button tone="halus" disabled={add.isPending} onClick={onClose}>
          Batal
        </Button>
        <Button tone="utama" disabled={add.isPending || malformed} onClick={submit}>
          {add.isPending ? 'Menyimpan…' : 'Simpan'}
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

function Display({ label, small, children }: { label: string; small?: boolean; children: ReactNode }) {
  return (
    <div>
      <Caption label={label} small={small ?? false} />
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
}: {
  label: string
  value: string
  onChange: (value: string) => void
  required?: boolean
}) {
  return (
    <label className="block">
      <Caption label={label} required={required ?? false} />
      <input
        aria-label={label}
        value={value}
        inputMode="decimal"
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm"
      />
    </label>
  )
}

const RISK_NAMES: Record<string, string> = Object.fromEntries(RISK_OPTIONS.map((o) => [o.value, o.label]))

/**
 * Tampilan baca satu baris Adjustment yang sudah tersimpan — dibuka dengan mengklik
 * barisnya di grid. Susunannya sama dengan isian (InputAdjustment_sect).
 *
 * Nilai Salvage B, Nilai Interim, dan komponen fee adjuster tidak punya kolom di
 * T_CLAIM_ADJUSTMENT; yang tersimpan hanya Nilai Nett Pembayaran hasilnya, sehingga
 * ketiganya tampil —— pada baris yang dimuat ulang.
 */
export function SettlementDetail({
  line,
  currencyName,
  estimation,
  spreading,
  travel,
  nonMBU,
}: {
  line: Settlement
  currencyName: string
  /** Nilai Estimasi jaminan (estimasi klaim), dalam sen. */
  estimation: number
  spreading: Spreading[]
  travel: boolean
  nonMBU: boolean
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
              <Display label="Total Klaim">{money(line.nilai_propose_sen)}</Display>
              <Display label="Nilai Pengajuan Tertanggung">{money(line.nilai_pengajuan_sen)}</Display>
              <Display label="Tipe Resiko Sendiri">{RISK_NAMES[line.tipe_resiko] ?? EMPTY}</Display>
              <Display label="Persen Resiko Sendiri (%)">{line.persen_resiko ? percent.format(line.persen_resiko / 10_000) : EMPTY}</Display>
              <Display label="Nilai Resiko Sendiri">{money(line.nilai_resiko_sen)}</Display>
            </>
          )}
          {fee && <Display label="Nilai Adjuster Fee">{money(line.nilai_asm_sen)}</Display>}
          <Display label="Nilai Nett Pembayaran">{money(line.nilai_gross_sen)}</Display>
          {!travel && <Display label="Status Persetujuan LOD">{EMPTY}</Display>}
          {nonMBU && (
            <Display label="Tanggal Transfer LOD" small>
              {EMPTY}
            </Display>
          )}
        </div>
        <div className="space-y-3">
          <Display label="Nilai Dalam IDR">{rate.format(line.kurs_e4 / 10_000)}</Display>
          <Display label="Nilai Estimasi">{money(line.nilai_estimasi_sen || estimation)}</Display>
          {proposeBased && !travel && (
            <>
              <Display label="Lack Of Document (%)">{line.loc ? percent.format(line.loc / 10_000) : EMPTY}</Display>
              <Display label="Nilai Salvage A">{optional(line.nilai_salvage_sen)}</Display>
              <Display label="Nilai Salvage B">{optional(line.nilai_salvage_b_sen)}</Display>
            </>
          )}
          {proposeBased && nonMBU && <Display label="Nilai Interim">{optional(line.nilai_interim_sen)}</Display>}
          <Display label="Share ASM (%)">{percent.format(line.share_asm / 10_000)}</Display>
          {proposeBased && <Display label="Nilai Yang Dibayarkan ASM">{money(line.nilai_asm_sen)}</Display>}
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
      {spreading.length > 0 && <SpreadingTable spreading={spreading} />}
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
