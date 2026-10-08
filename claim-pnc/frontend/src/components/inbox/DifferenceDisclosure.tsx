/**
 * DifferenceDisclosure menggambar selisih terhadap layar Pega di kaki halaman, terlipat.
 *
 * Petugas yang membandingkan layar ini dengan Pega berdampingan AKAN menemukan selisihnya —
 * dan selisih yang tidak dinyatakan akan dilaporkan sebagai kerusakan, lalu ditelusuri
 * ulang oleh orang yang tidak tahu bahwa ia disengaja. Tidak digambar bila kosong.
 */
export function DifferenceDisclosure({ items }: Readonly<{ items: string[] }>) {
  if (items.length === 0) return null

  return (
    <details className="rounded-kartu border border-slate-200 bg-slate-50 p-4">
      <summary className="cursor-pointer text-sm font-medium text-slate-800">
        Perbedaan yang disengaja terhadap layar Pega ({items.length})
      </summary>
      <ul className="mt-3 space-y-2 text-sm text-slate-600">
        {items.map((isi) => (
          <li key={isi} className="flex gap-2">
            <span aria-hidden className="text-slate-400">
              •
            </span>
            <span>{isi}</span>
          </li>
        ))}
      </ul>
    </details>
  )
}
