import type { RingkasStatus } from './types'

type Props = {
  rows: RingkasStatus[]
  isLoading: boolean
}

/**
 * StatusSummary adalah tabel ringkas **"Status / Jumlah"** di atas daftar.
 *
 * # Apa yang digantikan
 *
 * Grid `TempPLADLA.pxResults` beserta bagan di sebelahnya pada
 * `Section/InboxDLAReas_sect-Section.xml`.
 *
 * # Kenapa TANPA bagan
 *
 * Pega menggambar bagan batang di samping tabelnya. Ia tidak dibawa, dan alasannya bukan
 * kemalasan: bagannya menggambarkan hal yang SAMA dengan tabel di sebelahnya — jumlah
 * klaim per status — pada daftar yang biasanya berisi kurang dari sepuluh status. Bagan
 * atas sepuluh angka tidak menjelaskan apa pun yang tidak sudah terbaca dari angkanya,
 * dan ia menambah satu pustaka bagan ke dalam bundel.
 *
 * Bila kelak jumlah statusnya bertambah banyak sehingga tabelnya sulit dibaca sekilas,
 * bagannya layak ditambahkan — dan itu keputusan tersendiri.
 *
 * # Kenapa barisnya TIDAK dapat diklik
 *
 * Berbeda dari tabel ringkas Inbox Salvage, baris di sini tidak menuju daftar mana pun:
 * status bukan tab, dan tidak ada daftar terpisah per status di Pega. Membuatnya dapat
 * diklik akan menjanjikan penyaringan yang tidak ada.
 */
export function StatusSummary({ rows, isLoading }: Readonly<Props>) {
  if (isLoading) {
    return (
      <p className="text-sm text-slate-600">Menghitung ringkasan status…</p>
    )
  }

  if (rows.length === 0) return null

  const total = rows.reduce((jumlah, baris) => jumlah + baris.jumlah, 0)

  return (
    <section
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      aria-label="Ringkasan jumlah klaim per status"
    >
      <h2 className="text-sm font-semibold text-slate-900">Status / Jumlah</h2>

      <ul className="mt-3 flex flex-wrap gap-2">
        {rows.map((baris) => (
          <li
            key={baris.kode_status}
            className="rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2"
          >
            <span className="block text-xs text-slate-600">
              {/*
                Label yang kosong digantikan KODENYA, bukan dibiarkan kosong.

                Kode status yang tidak ada di master memang terjadi — domainnya 33 kode
                dan data lama memuat kode di luar itu. Sel kosong akan membuat barisnya
                tampak rusak; kodenya masih dapat ditelusuri.
              */}
              {baris.status || `Kode ${baris.kode_status || '—'}`}
            </span>
            <span className="block text-base font-semibold text-slate-900">
              {baris.jumlah}
            </span>
          </li>
        ))}
      </ul>

      <p className="mt-3 text-xs text-slate-500">
        Jumlah seluruhnya {total} klaim, mengikuti pencarian yang sedang aktif.
      </p>
    </section>
  )
}
