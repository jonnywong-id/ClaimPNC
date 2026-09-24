import { useMemo } from 'react'
import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'

import { NetworkError } from '@/api/client'
import type { AutoClaimCompanySummary } from '@/api/types'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useAutoClaimSummary } from './api'

type Props = {
  /** Tab yang sedang terbuka; ringkasannya dibaca dari tabel tab itu. */
  source: string

  /** Kode perusahaan yang sedang dipilih; teks kosong berarti "All". */
  selected: string
  onSelect: (code: string) => void
}

/**
 * Panel ringkasan Inbox Auto Claim — donut di kiri, tabel jumlah di kanan.
 *
 * # Apa yang menggantikan apa
 *
 * Bentuknya mengikuti contoh yang disetujui Work Owner 2026-09-19: layar Pega menampilkan
 * ringkasan ini lewat komponen bawaan platform yang meringkas report definition-nya
 * sendiri. Komponen itu TIDAK ADA di export dan tidak punya rule yang dapat dibaca —
 * jadi panel ini **kemampuan baru**, bukan pemindahan, dan tidak ada yang dapat
 * dibandingkan dengannya pada gerbang 1.
 *
 * Yang menjaganya tetap jujur adalah satuan hitungnya: **jumlah BATCH**, sama dengan satu
 * baris grid. Angka di sini dan total paginasi grid setelah disaring selalu cocok, dan
 * itu dapat diperiksa pengguna langsung dari layar.
 *
 * # Ia SATU-SATUNYA penyaring perusahaan
 *
 * Mengeklik irisan, baris, atau nama di legenda menyaring grid di bawahnya; baris **All**
 * membatalkannya. Dropdown "Nama Perusahaan" yang dulu duduk di bilah judul grid sudah
 * dibuang atas keputusan Work Owner 2026-09-20, mengikuti layar Pega yang juga tidak
 * punya dropdown — di sana penyaringnya memang panel ringkasan ini.
 *
 * Konsekuensinya diterima sadar: pada portal dengan puluhan perusahaan, menemukan satu
 * nama berarti menelusuri tabel ini, bukan mengetik di daftar pilihan. Karena itu tabel
 * di sebelah kanan memuat SELURUH perusahaan master — termasuk yang jumlah batch-nya nol
 * dan karenanya tidak digambar di donut. Tanpa itu, perusahaan yang belum pernah
 * mengirim batch tidak akan punya cara dipilih sama sekali.
 */
export function CompanySummary({ source, selected, onSelect }: Props) {
  const summary = useAutoClaimSummary(source)

  const company = useMemo(() => summary.data?.perusahaan ?? [], [summary.data])
  const total = summary.data?.total ?? 0

  if (summary.isError) {
    return (
      <div className="mt-5">
        <ErrorMessage
          title="Ringkasan tidak dapat dimuat"
          description={
            summary.error instanceof NetworkError
              ? 'Server Claim PNC tidak dapat dihubungi.'
              : 'Daftar batch di bawah tetap dapat dipakai. Coba muat ulang halaman ini.'
          }
          tone="gangguan"
        />
      </div>
    )
  }

  if (summary.isPending) {
    return (
      <div
        className="mt-5 h-64 animate-pulse rounded-kartu border border-slate-200 bg-slate-50"
        aria-label="Memuat ringkasan"
      />
    )
  }

  if (company.length === 0) {
    return null
  }

  return (
    <section
      className="mt-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      aria-label="Ringkasan batch per perusahaan"
    >
      <div className="grid gap-6 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]">
        <SummaryChart company={company} selected={selected} onSelect={onSelect} />
        <SummaryTable company={company} total={total} selected={selected} onSelect={onSelect} />
      </div>
    </section>
  )
}

/**
 * Donut jumlah batch per perusahaan.
 *
 * Grafiknya disembunyikan dari pembaca layar (`aria-hidden`) dan tabel di sebelahnya yang
 * menjadi sumber resminya. Alasannya: seluruh angka yang ada di grafik ADA di tabel, dan
 * tabel dapat dibaca berurutan sementara SVG tidak. Membiarkan keduanya terbaca hanya
 * membuat setiap angka dibacakan dua kali.
 */
function SummaryChart({
  company,
  selected,
  onSelect,
}: {
  company: AutoClaimCompanySummary[]
  selected: string
  onSelect: (code: string) => void
}) {
  // Yang berjumlah NOL tidak digambar — irisan bernilai nol tidak punya sudut, dan
  // legendanya akan penuh nama yang tidak menunjuk apa pun.
  //
  // Penyaringan ini ada di GRAFIK saja, bukan di data: tabel di sebelahnya justru harus
  // memuat seluruh perusahaan master, termasuk yang belum pernah mengirim — itu yang
  // membuatnya dapat dipilih sebagai penyaring.
  const data = company
    .filter((c) => c.jumlah_batch > 0)
    .map((c) => ({
      kode: c.kode,
      nama: c.nama === '' ? c.kode : c.nama,
      jumlah: c.jumlah_batch,
    }))

  return (
    <div className="h-64" aria-hidden="true">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            dataKey="jumlah"
            nameKey="nama"
            innerRadius="55%"
            outerRadius="80%"
            paddingAngle={1}
            // Animasi dimatikan: ia membuat uji komponen harus menunggu tanpa alasan,
            // dan pada data yang jarang berubah tidak menambah kejelasan apa pun.
            isAnimationActive={false}
            // Irisan yang diklik dikenali lewat INDEKS, bukan lewat isi objek yang
            // dikirim Recharts. Bentuk objek itu milik pustaka dan dapat berubah antar
            // versi; indeksnya menunjuk ke `data` yang kita susun sendiri, dan itu tetap
            // benar tanpa perlu menebak bentuk apa pun.
            onClick={(_slice, index) => {
              const clicked = data[index]
              if (clicked === undefined) return
              onSelect(clicked.kode === selected ? '' : clicked.kode)
            }}
          >
            {data.map((slice, index) => (
              <Cell
                key={slice.kode}
                fill={sliceColour(index)}
                // Irisan yang sedang dipilih dipertegas GARIS TEPI, bukan hanya warna —
                // pembedaan penting tidak pernah hanya warna (sistem desain).
                stroke={slice.kode === selected ? '#0f172a' : '#ffffff'}
                strokeWidth={slice.kode === selected ? 3 : 1}
                className="cursor-pointer outline-none"
              />
            ))}
          </Pie>
          <Tooltip
            // Nilainya datang bertipe longgar dari Recharts, jadi bentuknya diperiksa di
            // sini alih-alih dipaksa dengan `as`. Yang bukan angka ditampilkan apa adanya
            // — lebih baik daripada "NaN batch".
            formatter={(value) => (typeof value === 'number' ? `${value} batch` : String(value))}
            contentStyle={{ fontSize: '0.8125rem', borderRadius: '0.5rem' }}
          />
          <Legend
            verticalAlign="bottom"
            height={48}
            wrapperStyle={{ fontSize: '0.75rem', lineHeight: '1.25rem', cursor: 'pointer' }}
            // Legenda ikut menyaring. Mengekliknya adalah hal yang wajar dicoba — dan
            // irisan kecil jauh lebih sulit dikenai daripada namanya di legenda.
            onClick={(entry: { value?: unknown }) => {
              const nama = typeof entry.value === 'string' ? entry.value : ''
              const cocok = data.find((d) => d.nama === nama)
              if (cocok === undefined) return
              onSelect(cocok.kode === selected ? '' : cocok.kode)
            }}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  )
}

/** Tabel "Nama Perusahaan | Count" beserta baris All. */
function SummaryTable({
  company,
  total,
  selected,
  onSelect,
}: {
  company: AutoClaimCompanySummary[]
  total: number
  selected: string
  onSelect: (code: string) => void
}) {
  return (
    <div className="overflow-x-auto">
      <table aria-label="Jumlah batch per perusahaan" className="w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-300 text-left">
            <th scope="col" className="px-3 py-2 font-semibold text-slate-700">
              Nama Perusahaan
            </th>
            <th scope="col" className="px-3 py-2 text-right font-semibold text-slate-700">
              Jumlah Batch
            </th>
          </tr>
        </thead>
        <tbody>
          <SummaryRow
            label="All"
            count={total}
            active={selected === ''}
            onSelect={() => onSelect('')}
          />
          {company.map((c) => (
            <SummaryRow
              key={c.kode}
              label={c.nama === '' ? c.kode : c.nama}
              // Kode disebut terpisah hanya bila namanya ada, supaya baris tanpa master
              // tidak menampilkan kode dua kali.
              hint={c.nama === '' ? 'tidak terdaftar di Master Auto Claim' : c.kode}
              count={c.jumlah_batch}
              active={selected === c.kode}
              onSelect={() => onSelect(selected === c.kode ? '' : c.kode)}
            />
          ))}
        </tbody>
      </table>
    </div>
  )
}

function SummaryRow({
  label,
  hint,
  count,
  active,
  onSelect,
}: {
  label: string
  hint?: string
  count: number
  active: boolean
  onSelect: () => void
}) {
  return (
    // SELURUH BARIS dapat diklik, bukan hanya teks namanya.
    //
    // Versi pertama hanya memasang penangan pada tombol nama, sementara barisnya menyala
    // saat disentuh kursor — jadi baris itu TERLIHAT dapat diklik seluruhnya. Pengguna
    // yang mengeklik angkanya, atau ruang kosong di sebelahnya, tidak mendapat apa pun
    // dan tidak ada tanda mengapa. Itu cacat yang benar-benar dilaporkan, dan sorotan
    // hover-lah yang menjanjikan sesuatu yang tidak ditepati.
    //
    // Tombolnya TETAP ADA dan tetap menjadi satu-satunya hal yang dapat difokus: pengguna
    // papan ketik dan pembaca layar memerlukan target bernama beserta keadaannya, dan
    // `<tr onClick>` saja tidak memberi keduanya. Klik pada barisnya hanyalah kemudahan
    // bagi pengguna tetikus.
    <tr
      onClick={onSelect}
      className={['cursor-pointer', active ? 'bg-blue-50' : 'hover:bg-slate-50'].join(' ')}
    >
      <td className="border-b border-slate-200 px-3 py-2">
        <button
          type="button"
          // Klik pada tombol dihentikan di sini supaya penangan barisnya tidak ikut
          // berjalan — tanpa itu, satu klik memanggil onSelect DUA KALI dan penyaringnya
          // langsung menyala lalu padam kembali.
          onClick={(event) => {
            event.stopPropagation()
            onSelect()
          }}
          // aria-pressed, bukan sekadar warna latar: pembaca layar tidak melihat warna,
          // dan "sedang menyaring perusahaan ini" adalah keadaan, bukan hiasan.
          aria-pressed={active}
          className={[
            'rounded-kontrol px-1 text-left transition-colors duration-150 ease-halus',
            'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20',
            active ? 'font-semibold text-blue-900' : 'text-slate-800 hover:text-blue-800',
          ].join(' ')}
        >
          {label}
          {hint !== undefined && hint !== '' && (
            <span className="ml-2 text-xs font-normal text-slate-500">{hint}</span>
          )}
        </button>
      </td>
      <td
        className={[
          'border-b border-slate-200 px-3 py-2 text-right tabular-nums',
          active ? 'font-semibold text-blue-900' : 'text-slate-700',
        ].join(' ')}
      >
        {count}
      </td>
    </tr>
  )
}

/**
 * Warna irisan.
 *
 * Daftarnya tetap dan berulang, bukan diacak atau diturunkan dari nama: warna yang
 * berpindah setiap kali data berubah membuat pengguna kehilangan jejak perusahaan yang
 * sedang diamatinya. Urutannya mengikuti urutan data, yang sendirinya menurun menurut
 * jumlah — jadi irisan terbesar selalu mendapat warna pertama.
 */
const PALETTE = [
  '#2563eb', // blue-600
  '#0d9488', // teal-600
  '#d97706', // amber-600
  '#7c3aed', // violet-600
  '#dc2626', // red-600
  '#0891b2', // cyan-600
  '#65a30d', // lime-600
  '#db2777', // pink-600
  '#475569', // slate-600
] as const

function sliceColour(index: number): string {
  // Modulo menjamin indeksnya selalu di dalam jangkauan, tetapi TypeScript tidak dapat
  // membuktikannya. Yang dipakai `?? PALETTE[0]`, bukan `as string`: kalau suatu saat
  // daftarnya dikosongkan, warnanya jelas salah — bukan `undefined` yang diam-diam
  // membuat irisannya tidak tergambar.
  return PALETTE[index % PALETTE.length] ?? PALETTE[0]
}
