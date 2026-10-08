import type { ReactNode } from 'react'

/** Penanda sedang menghitung ringkasan status. */
export function SummaryLoading({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <output
      className="block rounded-kartu border border-slate-200 bg-white p-4 text-sm text-slate-500"
    >
      {children}
    </output>
  )
}

/**
 * StatusSummaryTable adalah tabel dua kolom "Status Salvage / Jumlah" di atas daftar
 * salvage. Baris-barisnya disusun pemanggil; `label` menamai tabelnya bagi pembaca layar.
 */
export function StatusSummaryTable({
  label,
  children,
}: Readonly<{ label: string; children: ReactNode }>) {
  return (
    <div className="overflow-x-auto rounded-kartu border border-slate-200 bg-white">
      <table className="w-full min-w-max text-sm" aria-label={label}>
        <caption className="sr-only">
          {label}. Pilih satu baris untuk membuka daftarnya.
        </caption>

        <thead className="bg-slate-50 text-left text-xs font-semibold tracking-wide text-slate-600 uppercase">
          <tr>
            <th scope="col" className="px-4 py-2.5">
              Status Salvage
            </th>
            <th scope="col" className="px-4 py-2.5 text-right">
              Jumlah
            </th>
          </tr>
        </thead>

        <tbody className="divide-y divide-slate-100">{children}</tbody>
      </table>
    </div>
  )
}

/**
 * SummaryLinkButton adalah nama status yang membuka daftarnya bila diklik. Menggambarnya
 * sebagai angka mati akan menghilangkan satu cara berpindah yang sudah ada di layar lama.
 */
export function SummaryLinkButton({
  selected,
  onClick,
  children,
}: Readonly<{ selected: boolean; onClick: () => void; children: ReactNode }>) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={selected ? 'true' : undefined}
      className={[
        'rounded-kontrol px-1 text-left',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
        selected
          ? 'font-semibold text-blue-700'
          : 'text-blue-700 hover:underline',
      ].join(' ')}
    >
      {children}
    </button>
  )
}
