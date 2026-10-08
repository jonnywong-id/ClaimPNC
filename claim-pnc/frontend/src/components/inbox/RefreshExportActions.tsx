import type { ReactNode } from 'react'

import { Button } from '@/components/Button'

/**
 * RefreshExportActions adalah deret tombol di kepala tabel inbox PLA / DLA: "Refresh" untuk
 * mengambil ulang halaman yang terbuka, dan "Export To Excel". Tombol tambahan yang harus
 * mendahului keduanya diberikan lewat `children`.
 */
export function RefreshExportActions({
  query,
  exporting,
  onExport,
  children,
}: Readonly<{
  query: { isFetching: boolean; refetch: () => unknown }
  exporting: boolean
  onExport: () => void
  children?: ReactNode
}>) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {children}
      <Button
        type="button"
        tone="kedua"
        onClick={() => { query.refetch() }}
        disabled={query.isFetching}
      >
        Refresh
      </Button>
      <Button
        type="button"
        tone="kedua"
        disabled={exporting}
        onClick={onExport}
      >
        {exporting ? 'Menyiapkan…' : 'Export To Excel'}
      </Button>
    </div>
  )
}
