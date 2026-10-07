import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'

import { SelectField } from '@/components/SelectField'

import type { BusinessLine, Tab, TabCount, ViewerResponse } from './types'

type Props = {
  tabs: Tab[]
  /** Jumlah baris per tab; kosong selama dimuat. */
  counts: TabCount[] | undefined
  countsLoading: boolean
  countsFailed: boolean
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
  lines: BusinessLine[]
  business: string
  onBusiness: (code: string) => void
  /** Batas data pemanggil; kosong selama dimuat. */
  viewer: ViewerResponse | undefined
  region: string
  onRegion: (code: string) => void
}

/** Warna irisan donat, berurutan mengikuti urutan tab. */
const SLICE_COLORS = ['#2f9fd0', '#f59e0b', '#10b981', '#8b5cf6', '#ef4444', '#64748b', '#ec4899', '#0ea5e9']

/**
 * StatusRegister — panel atas layar Inbox Admin, pengganti daftar "Status Register" Pega.
 *
 * # Apa yang digantikan
 *
 * Di Pega, antrean TIDAK dipilih lewat bilah tab, melainkan lewat tabel dua kolom
 * "Status Register | Count" yang disusun activity `GetReportClaimRegistList`, dengan
 * diagram donat jumlahnya di sebelah kiri dan dropdown "Bisnis" di atasnya (tangkapan
 * layar Pega 2026-10-07). Mengeklik nama baris membuka antreannya.
 *
 * Label barisnya mengikuti activity itu — "Outstanding" untuk kode 3, "LOD" untuk 12,
 * "PUCL" untuk 13 (`D-13`).
 *
 * # Yang sengaja TIDAK ditiru
 *
 * - Baris "Unregistered RCV" tampil DUA KALI di Pega, karena dua langkah activity
 *   (32 dan 34) sama-sama menambahkannya dari dua sumber hitungan yang berbeda. Di sini ia
 *   satu baris, dengan jumlah yang sama dengan isi tabnya.
 * - Dropdown "Pilih Kanwil" hanya tampil bagi manajer, karena hanya bagi merekalah
 *   pilihan kanwil berlaku di Pega. Pilihannya dibaca dari basis data (Kanwil 1-6),
 *   bukan dari daftar tetap Pega yang berhenti di Kanwil 5.
 *
 * Baris dipasang dengan `role="tab"` — ia memang berfungsi sebagai pemilih antrean, dan
 * pembaca layar karenanya mengumumkan antrean mana yang sedang terbuka.
 */
export function StatusRegister({
  tabs,
  counts,
  countsLoading,
  countsFailed,
  active,
  onSelect,
  lines,
  business,
  onBusiness,
  viewer,
  region,
  onRegion,
}: Props) {
  const countOf = new Map((counts ?? []).map((c) => [c.kode, c]))
  const chartData = tabs
    .map((tab, index) => ({
      kode: tab.kode,
      nama: tab.nama,
      jumlah: countOf.get(tab.kode)?.jumlah ?? 0,
      warna: SLICE_COLORS[index % SLICE_COLORS.length] ?? '#2f9fd0',
    }))
    .filter((slice) => slice.jumlah > 0)

  return (
    <section className="mt-4 grid gap-6 rounded-kartu border border-slate-200 bg-white p-5 lg:grid-cols-2">
      <div className="flex min-h-56 items-center justify-center">
        {countsLoading ? (
          <p className="text-sm text-slate-500">Menghitung antrean…</p>
        ) : chartData.length === 0 ? (
          <p className="text-sm text-slate-500">Tidak ada antrean yang berisi.</p>
        ) : (
          <div className="h-64 w-full" aria-hidden="true">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={chartData}
                  dataKey="jumlah"
                  nameKey="nama"
                  innerRadius="50%"
                  outerRadius="80%"
                  paddingAngle={chartData.length > 1 ? 1 : 0}
                  label={({ value }) => String(value ?? '')}
                  isAnimationActive={false}
                  onClick={(_slice, index) => {
                    const clicked = chartData[index]
                    if (clicked) onSelect(clicked.kode)
                  }}
                >
                  {chartData.map((slice) => (
                    <Cell
                      key={slice.kode}
                      fill={slice.warna}
                      stroke={slice.kode === active ? '#0f172a' : '#ffffff'}
                      strokeWidth={slice.kode === active ? 3 : 1}
                      className="cursor-pointer"
                    />
                  ))}
                </Pie>
                <Tooltip contentStyle={{ fontSize: '0.8125rem', borderRadius: '0.5rem' }} />
                <Legend verticalAlign="bottom" height={36} wrapperStyle={{ fontSize: '0.75rem' }} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      <div>
        <div className="flex flex-wrap items-end gap-3">
          {/*
            "Pilih Kanwil" hanya bagi manajer: langkah 2 activity Pega memasang penyaring
            kanwil HANYA untuk CaseManager dan PncManagerAdmin. Petugas lain dibatasi
            cabangnya sendiri, dan dropdown yang tidak mengubah apa pun lebih menyesatkan
            daripada tidak ada.
          */}
          {viewer?.manajer && (
            <div className="w-full sm:w-44">
              <SelectField
                id="inbox-admin-kanwil"
                label="Pilih Kanwil"
                options={viewer.kanwil.map((option) => ({ value: option.kode, label: option.label }))}
                emptyText="--Pilih--"
                value={region}
                onChange={(event) => onRegion(event.target.value)}
              />
            </div>
          )}
          <div className="w-full sm:w-56">
            <SelectField
              id="inbox-admin-bisnis"
              label="Bisnis"
              options={lines.map((line) => ({ value: line.kode, label: line.label }))}
              emptyText="Semua Lini Bisnis"
              value={business}
              onChange={(event) => onBusiness(event.target.value)}
            />
          </div>
        </div>
        {viewer?.cabang && (
          <p className="mt-2 text-xs text-slate-600">
            Antrean dibatasi cabang Anda ({viewer.cabang}), sama seperti di sistem lama.
          </p>
        )}

        <table className="mt-4 w-full text-sm">
          <thead>
            <tr className="border-b border-slate-300 text-left text-xs font-semibold text-slate-700">
              <th className="py-1.5 pl-2">Status Register</th>
              <th className="w-28 py-1.5">Count</th>
            </tr>
          </thead>
          <tbody role="tablist" aria-label="Antrean Inbox Admin">
            {tabs.map((tab) => {
              const selected = tab.kode === active
              const count = countOf.get(tab.kode)
              return (
                <tr key={tab.kode} className={['border-b border-slate-200', selected ? 'bg-blue-50' : ''].join(' ')}>
                  <td className="py-1.5 pl-2">
                    <button
                      type="button"
                      role="tab"
                      aria-selected={selected}
                      title={tab.keterangan}
                      onClick={() => onSelect(tab.kode)}
                      className={[
                        'inline-flex items-center gap-2 text-left',
                        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                        selected ? 'font-semibold text-blue-700 underline' : 'text-slate-800 hover:underline',
                      ].join(' ')}
                    >
                      <span aria-hidden="true">📁</span>
                      {tab.nama}
                    </button>
                  </td>
                  <td className="py-1.5">
                    <button
                      type="button"
                      tabIndex={-1}
                      onClick={() => onSelect(tab.kode)}
                      className="text-blue-600 hover:underline"
                      aria-label={`Buka ${tab.nama}`}
                    >
                      {countsLoading ? '…' : count?.gagal || countsFailed ? '!' : (count?.jumlah ?? 0)}
                    </button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
        {(countsFailed || (counts ?? []).some((c) => c.gagal)) && (
          <p className="mt-2 text-xs text-amber-700">
            Tanda ! berarti jumlah antrean itu tidak dapat dihitung; isinya tetap dapat dibuka.
          </p>
        )}
      </div>
    </section>
  )
}
