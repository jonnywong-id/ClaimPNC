import { Button } from '@/components/Button'
import { FormField } from '@/components/FormField'

/** Rentang tanggal yang sedang diketik pada bilah penyaring laporan. */
export type PeriodDraft = {
  dari: string
  sampai: string
}

/**
 * PeriodFields menggambar dua isian tanggal "dari" dan "sampai" pada bilah penyaring
 * laporan. id isian dibentuk `<idPrefix>-dari` dan `<idPrefix>-sampai`; pelanggaran dibaca
 * dari kunci `dari` dan `sampai` yang dikirim server.
 */
export function PeriodFields({
  idPrefix,
  fromLabel,
  toLabel,
  value,
  violations,
  onChange,
  className,
}: Readonly<{
  idPrefix: string
  fromLabel: string
  toLabel: string
  value: PeriodDraft
  violations: Record<string, string>
  onChange: (patch: Partial<PeriodDraft>) => void
  /** Kelas tambahan pada kedua isian, mis. pembatas lebar. */
  className?: string | undefined
}>) {
  return (
    <>
      <FormField
        id={`${idPrefix}-dari`}
        label={fromLabel}
        type="date"
        className={className}
        value={value.dari}
        failure={violations['dari']}
        onChange={(event) => onChange({ dari: event.target.value })}
      />

      <FormField
        id={`${idPrefix}-sampai`}
        label={toLabel}
        type="date"
        className={className}
        value={value.sampai}
        failure={violations['sampai']}
        onChange={(event) => onChange({ sampai: event.target.value })}
      />
    </>
  )
}

/**
 * ReportActions menggambar baris tombol bilah penyaring laporan: tombol kirim pencarian
 * dan tombol ekspor. Ekspor memakai penyaring yang SUDAH DIKIRIM — itu urusan pemanggil
 * lewat `onExport`.
 */
export function ReportActions({
  searchLabel,
  exportLabel,
  onExport,
  exportDisabled,
}: Readonly<{
  searchLabel: string
  exportLabel: string
  onExport: () => void
  exportDisabled: boolean
}>) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button type="submit" tone="utama">
        {searchLabel}
      </Button>
      <Button onClick={onExport} disabled={exportDisabled}>
        {exportLabel}
      </Button>
    </div>
  )
}
