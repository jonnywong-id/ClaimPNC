import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'

import { NetworkError } from '@/api/client'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useOutstandingSummary } from './api'
import type { DocumentStatusCode, DocumentStatusCount, OutstandingFilter } from './types'

type Props = {
  /** Penyaring layar yang sedang berlaku; ringkasan menghormati semuanya kecuali status. */
  filter: OutstandingFilter

  /** Status yang sedang dipilih; teks kosong berarti seluruh status. */
  selected: DocumentStatusCode | ''
  onSelect: (status: DocumentStatusCode | '') => void
}

/**
 * Panel status dokumen My Inbox — deret tab beserta lencananya, dan donut di bawahnya.
 *
 * # Apa yang menggantikan apa
 *
 * `Section/InboxRegister_Section-Section.xml` memuat SEMBILAN tab bertuliskan tebal,
 * berurutan `:11438`, `:11626`, `:11841`, `:12023`, `:12170`, `:12632`, `:12825`,
 * `:12971`, `:13184` — masing-masing menyetel `TempVisibility.Email`. Donutnya ada di
 * `:4151` (`pyType=pie`, `pySubType=doughnut`, agregasi `count`). Keduanya **pemindahan**,
 * bukan kemampuan baru.
 *
 * Bentuk tab dan lencananya mengikuti Inbox Laporan Klaim, atas permintaan Work Owner —
 * yang ditiru POLANYA, modulnya tidak disentuh.
 *
 * # Tab tanpa lencana bukan tab bernilai nol
 *
 * Enam dari sembilan tab belum dapat dihitung dari `T_CLAIMLIST_ADMIN`:
 *
 *     Loss Adjuster · Internal Surveyor   Work-SurveyClaim — case type LAIN
 *     Temporary Close · Deadline          Resolved-Completed — di LUAR himpunan ini
 *     Communication                       tabel komunikasi
 *     TKA                                 kolom TKA_1 TIDAK ADA di tabel ini
 *
 * Keenamnya tetap DIGAMBAR supaya petugas Pega mengenali layarnya, tetapi tanpa lencana
 * dan tidak dapat ditekan. Memberinya angka nol akan menyatakan "tidak ada satu pun",
 * padahal yang benar "belum dihitung" — alasan yang sama persis dengan tab "Data rejected"
 * pada Inbox Laporan Klaim.
 */
export function DocumentStatusSummary({ filter, selected, onSelect }: Props) {
  const summary = useOutstandingSummary(filter)

  if (summary.isError) {
    return (
      <div className="mt-5">
        <ErrorMessage
          title="Ringkasan tidak dapat dimuat"
          description={
            summary.error instanceof NetworkError
              ? 'Server Claim PNC tidak dapat dihubungi.'
              : 'Daftar klaim di bawah tetap dapat dipakai. Coba muat ulang halaman ini.'
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

  // `?? []` bukan kerapian: respons yang cacat atau versi backend yang lebih tua akan
  // menjatuhkan SELURUH layar bila dibaca membabi buta — dan yang hilang hanyalah panel
  // ini, sementara daftar klaim di bawahnya tetap berguna.
  const status = summary.data.status ?? []
  const total = summary.data.total ?? 0

  return (
    <section
      className="mt-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      aria-label="Ringkasan status dokumen"
    >
      <StatusTabs status={status} total={total} selected={selected} onSelect={onSelect} />

      {/*
        Panel TETAP TAMPIL meski inbox kosong, dan itu koreksi atas versi pertama.

        Versi pertama mengembalikan `null` saat total nol, dengan alasan donut tanpa irisan
        hanya ruang kosong. Alasannya masuk akal dan akibatnya buruk: petugas yang tidak
        sedang memegang satu pun tugas tidak melihat panelnya SAMA SEKALI, sehingga tidak
        ada cara membedakan "saya tidak punya pekerjaan" dari "fiturnya tidak ada".
      */}
      <div className="mt-5">
        {total === 0 ? (
          <div className="flex h-56 items-center justify-center rounded-kartu bg-slate-50 px-4 text-center text-sm text-slate-500">
            Tidak ada klaim berjalan untuk diringkas.
          </div>
        ) : (
          <SummaryChart status={status} selected={selected} onSelect={onSelect} />
        )}
      </div>
    </section>
  )
}

/**
 * Deret tab beserta lencananya.
 *
 * # Kenapa `<nav>` berisi tombol, bukan tautan
 *
 * Karena berpindah tab TIDAK mengubah alamat halaman. Menjadikannya tautan berarti
 * berjanji alamatnya dapat disalin dan dibuka kembali — janji yang tidak dipenuhi selama
 * penyaringnya hidup di state komponen.
 */
function StatusTabs({
  status,
  total,
  selected,
  onSelect,
}: {
  status: DocumentStatusCount[]
  total: number
  selected: DocumentStatusCode | ''
  onSelect: (status: DocumentStatusCode | '') => void
}) {
  if (status.length === 0) return null

  return (
    <nav aria-label="Status dokumen" className="flex flex-wrap gap-2">
      {status.map((s) => {
        // "ALL Case" adalah keadaan TANPA penyaring, bukan satu status tersendiri —
        // memilihnya berarti membatalkan pilihan.
        const isAll = s.kode === 'semua'
        const active = isAll ? selected === '' : s.kode === selected
        const jumlah = isAll ? total : s.jumlah

        return (
          <button
            key={s.kode}
            type="button"
            disabled={!s.dapat_dipilih}
            aria-current={active ? 'true' : undefined}
            /*
              Lencananya diberi nama, bukan dibiarkan terbaca sebagai angka telanjang.
              Tanpa ini pembaca layar menyebut "Complete documents0" — angkanya menempel
              tanpa jeda, dan "0" sendirian tidak menyatakan nol apa.
            */
            aria-label={
              !s.dapat_dipilih
                ? `${s.judul}, belum tersedia`
                : jumlah === null
                  ? s.judul
                  : `${s.judul}, ${jumlah} klaim`
            }
            title={
              s.dapat_dipilih ? undefined : 'Sumber datanya belum dimigrasikan dari Pega.'
            }
            onClick={() => onSelect(isAll || active ? '' : s.kode)}
            className={[
              'inline-flex items-center gap-2 rounded-kontrol border px-3 py-1.5 text-sm',
              'transition-[background-color,border-color,box-shadow] duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
              s.dapat_dipilih
                ? active
                  ? 'border-blue-600 bg-blue-600 font-medium text-white shadow-aksen'
                  : 'border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:bg-slate-50'
                : 'cursor-not-allowed border-slate-200 bg-slate-50 text-slate-400',
            ].join(' ')}
          >
            {s.judul}
            {/*
              Lencana hanya digambar bila jumlahnya BENAR-BENAR dihitung — konvensi yang
              sama dengan tab "Data rejected" pada Inbox Laporan Klaim.
            */}
            {jumlah !== null && (
              <span
                aria-hidden="true"
                className={[
                  'rounded-full px-1.5 py-0.5 text-xs font-medium tabular-nums',
                  active ? 'bg-white/20 text-white' : 'bg-slate-100 text-slate-600',
                ].join(' ')}
              >
                {jumlah}
              </span>
            )}
          </button>
        )
      })}
    </nav>
  )
}

/**
 * Donut jumlah klaim per status dokumen.
 *
 * Grafiknya disembunyikan dari pembaca layar (`aria-hidden`) dan deret tab di atasnya yang
 * menjadi sumber resminya: seluruh angka yang ada di grafik ADA di lencana tab, dan tab
 * dapat dibaca berurutan sementara SVG tidak.
 */
function SummaryChart({
  status,
  selected,
  onSelect,
}: {
  status: DocumentStatusCount[]
  selected: DocumentStatusCode | ''
  onSelect: (status: DocumentStatusCode | '') => void
}) {
  // Yang digambar hanyalah tab yang benar-benar MEMBAGI HABIS inbox.
  //
  // "ALL Case" dikeluarkan karena ia totalnya sendiri — memasukkannya membuat donut
  // separuhnya satu irisan bernama "semua". Yang jumlahnya nol tidak digambar: irisan nol
  // tidak punya sudut, dan legendanya akan penuh nama yang tidak menunjuk apa pun.
  const data = status.filter(
    (s) => s.dapat_dipilih && s.kode !== 'semua' && s.jumlah !== null && s.jumlah > 0,
  )

  if (data.length === 0) {
    return (
      <div className="flex h-56 items-center justify-center rounded-kartu bg-slate-50 px-4 text-center text-sm text-slate-500">
        Belum ada status dokumen yang dapat digambarkan.
      </div>
    )
  }

  return (
    <div className="h-56" aria-hidden="true">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            dataKey="jumlah"
            nameKey="judul"
            innerRadius="55%"
            outerRadius="80%"
            paddingAngle={1}
            // Animasi dimatikan: ia membuat uji komponen harus menunggu tanpa alasan.
            isAnimationActive={false}
            // Irisan dikenali lewat INDEKS, bukan lewat isi objek yang dikirim Recharts —
            // bentuk objek itu milik pustaka dan dapat berubah antar versi.
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
                // Irisan terpilih dipertegas GARIS TEPI, bukan hanya warna — pembedaan
                // penting tidak pernah hanya warna.
                stroke={slice.kode === selected ? '#0f172a' : '#ffffff'}
                strokeWidth={slice.kode === selected ? 3 : 1}
                className="cursor-pointer outline-none"
              />
            ))}
          </Pie>
          <Tooltip
            formatter={(value) => (typeof value === 'number' ? `${value} klaim` : String(value))}
            contentStyle={{ fontSize: '0.8125rem', borderRadius: '0.5rem' }}
          />
          <Legend
            verticalAlign="bottom"
            height={36}
            wrapperStyle={{ fontSize: '0.75rem', lineHeight: '1.25rem', cursor: 'pointer' }}
            onClick={(entry: { value?: unknown }) => {
              const judul = typeof entry.value === 'string' ? entry.value : ''
              const cocok = data.find((d) => d.judul === judul)
              if (cocok === undefined) return
              onSelect(cocok.kode === selected ? '' : cocok.kode)
            }}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  )
}

/**
 * Warna irisan.
 *
 * Daftarnya tetap dan berulang, bukan diacak: warna yang berpindah setiap kali data
 * berubah membuat pengguna kehilangan jejak status yang sedang diamatinya.
 */
const PALETTE = [
  '#d97706', // amber-600
  '#0d9488', // teal-600
  '#2563eb', // blue-600
  '#7c3aed', // violet-600
] as const

function sliceColour(index: number): string {
  // Modulo menjamin indeksnya di dalam jangkauan, tetapi TypeScript tidak dapat
  // membuktikannya. Yang dipakai `?? PALETTE[0]`, bukan `as string`.
  return PALETTE[index % PALETTE.length] ?? PALETTE[0]
}
