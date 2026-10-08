import { Button } from '@/components/Button'

type Props = {
  /** Keadaan mutasi ekspor (`useMutation`). */
  state: { isPending: boolean; isError: boolean; error: unknown }
  onExport: () => void
  enabled: boolean
  describe: (error: unknown) => string
  /** Teks tombol saat diam. */
  label?: string | undefined
}

/**
 * ColumnsExportButton adalah tombol ekspor yang menyebutkan kolom isi berkasnya di bawah
 * tombol — dipakai layar yang berkasnya berisi kolom berbeda dari tabel di layar.
 */
export function ColumnsExportButton({
  state,
  onExport,
  disabled,
  label,
  columns,
  describe,
}: Readonly<{
  state: { isPending: boolean; isError: boolean; error: unknown }
  onExport: () => void
  disabled: boolean
  /** Teks tombol, termasuk saat sedang menyiapkan berkas. */
  label: string
  /** Judul kolom isi berkas; tidak digambar bila kosong. */
  columns: { judul: string }[]
  describe: (error: unknown) => string
}>) {
  return (
    <div className="flex max-w-sm flex-col items-end gap-1">
      <Button tone="kedua" disabled={disabled} onClick={onExport}>
        {label}
      </Button>

      {columns.length > 0 && (
        <p className="text-right text-xs text-slate-500">
          Berisi: {columns.map((column) => column.judul).join(' · ')}
        </p>
      )}

      {state.isError && (
        <p className="text-right text-xs text-red-700" role="alert">
          {describe(state.error)}
        </p>
      )}
    </div>
  )
}

/**
 * Tombol ekspor antrean inbox.
 *
 * Ekspor hanya membaca, dan membaca tidak melanggar kepemilikan tabel (`P-1`), karena itu
 * tombol ini berfungsi penuh. Ia dimatikan saat tidak ada yang dapat diekspor: berkas
 * kosong yang tetap terunduh adalah jawaban yang membingungkan — pengguna tidak dapat
 * membedakannya dari ekspor yang gagal diam-diam.
 */
export function ExportDataButton({ state, onExport, enabled, describe, label = 'Export Data' }: Readonly<Props>) {
  return (
    <div className="flex flex-col items-end gap-1">
      <Button tone="kedua" disabled={!enabled || state.isPending} onClick={onExport}>
        {state.isPending ? 'Menyiapkan berkas…' : label}
      </Button>
      {state.isError && (
        <p className="max-w-md text-right text-xs text-red-700" role="alert">
          {describe(state.error)}
        </p>
      )}
    </div>
  )
}
