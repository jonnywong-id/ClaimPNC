import { Button } from '@/components/Button'

/** Bentuk paginasi yang dikirim server pada layar-layar inbox. */
export type PrevNextPageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

type Props = {
  info: PrevNextPageInfo
  /** Jumlah baris yang benar-benar tergambar di halaman ini. */
  visible: number
  onMove: (page: number) => void
  loading: boolean
  /** Kata benda satuan baris pada ringkasan, mis. "baris" atau "klaim". */
  unit?: string | undefined
}

/**
 * Paginasi "sebelumnya / berikutnya", bukan nomor halaman.
 *
 * Bentuknya sama di seluruh layar inbox (Inbox Admin, Pelaporan Klaim, Claim Treaty, dan
 * saudaranya) supaya layar-layar itu tidak terasa dirakit dari aplikasi yang berbeda.
 */
export function InboxPagination({ info, visible, onMove, loading, unit = 'baris' }: Readonly<Props>) {
  const first = visible === 0 ? 0 : (info.halaman - 1) * info.ukuran + 1
  const last = (info.halaman - 1) * info.ukuran + visible

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <output className="block text-sm text-slate-600">
        Menampilkan {first}–{last} dari {info.total} {unit}.
      </output>
      <div className="flex gap-2">
        <Button
          tone="kedua"
          onClick={() => onMove(Math.max(1, info.halaman - 1))}
          disabled={info.halaman <= 1 || loading}
        >
          Sebelumnya
        </Button>
        <Button
          tone="kedua"
          onClick={() => onMove(info.halaman + 1)}
          disabled={info.halaman >= info.total_halaman || loading}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}
