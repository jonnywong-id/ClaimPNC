import { useEffect, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { formatRupiah } from '@/lib/money'

import { useOSClaimPerCabangDetail } from './api'
import type {
  DetailHeader,
  DetailMessage,
  DetailObject,
  DetailProgress,
} from './types'

/**
 * Popup Detail — pengganti harness `View_DetailKlaimCabang_Harness`.
 *
 * Dibuka tombol "Detail" pada grid, persis seperti di layar lama. Isinya empat bagian dengan
 * urutan yang sama dengan `Section/DetailKlaimCabang_Sect-Section.xml`:
 *
 *   1. ringkasan             delapan nilai berlabel
 *   2. objek pertanggungan   grid, kolomnya berbeda menurut lini bisnis
 *   3. riwayat progres       grid 8 kolom
 *   4. komunikasi adjuster   grid 5 kolom
 *
 * # Hanya NOMOR klaim yang dikirim
 *
 * Tombol Detail di Pega mengirim lima parameter dari baris yang diklik — nomor, nilai
 * cadangan, umur, lini bisnis, dan catatan PIC. Di sini hanya nomornya. Empat sisanya dibaca
 * ulang peladen, karena nilai uang yang ditentukan peramban dapat diubah lewat alat
 * pengembang biasa.
 *
 * Akibat yang disadari: bila datanya berubah antara daftar dimuat dan popup dibuka, angka di
 * popup berbeda dari angka di barisnya. Yang benar adalah yang di popup.
 */
export function DetailDialog({
  nomorKlaim,
  onTutup,
}: {
  nomorKlaim: string
  onTutup: () => void
}) {
  const detail = useOSClaimPerCabangDetail(nomorKlaim)

  // Escape menutup popup, seperti dialog mana pun yang dikenal pengguna.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onTutup()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onTutup])

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-detail-os-cabang"
    >
      <div className="w-full max-w-5xl rounded-kartu bg-white p-6 shadow-angkat">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2
              id="judul-detail-os-cabang"
              className="text-lg font-semibold text-slate-900"
            >
              Detail Klaim
            </h2>
            <p className="mt-1 font-mono text-sm text-slate-600">{nomorKlaim}</p>
          </div>
          <Button tone="kedua" onClick={onTutup}>
            Tutup
          </Button>
        </div>

        {detail.isLoading ? (
          <p className="mt-6 text-sm text-slate-500">Memuat detail klaim…</p>
        ) : null}

        {detail.error ? (
          <div className="mt-6">
            {/*
              Nadanya PENOLAKAN ketika klaimnya tidak ditemukan, dan GANGGUAN untuk
              selebihnya. Keduanya menuntut tindakan yang berbeda: yang pertama menyuruh
              pengguna memuat ulang daftarnya, yang kedua menyuruhnya mencoba lagi nanti.
            */}
            <ErrorMessage
              tone={isNotFound(detail.error) ? 'penolakan' : 'gangguan'}
              title={
                isNotFound(detail.error)
                  ? 'Klaim tidak ditemukan'
                  : 'Detail klaim tidak dapat dimuat'
              }
              description={messageOf(detail.error)}
            />
          </div>
        ) : null}

        {detail.data ? (
          <div className="mt-6 space-y-8">
            <Summary header={detail.data.ringkasan} />

            <Section title="Objek Pertanggungan">
              <DataTable
                columns={objectColumnsFor(detail.data.ringkasan.cob)}
                rows={detail.data.objek}
                // Nama objek DAPAT berulang pada satu klaim — dua unit di lokasi berbeda
                // kerap bernama sama. Kuncinya karena itu menggabungkan seluruh kolom yang
                // membedakan baris, bukan namanya saja.
                rowKey={(row) => `${row.nama}|${row.lokasi}|${row.ktp_paspor}`}
                hideSearch
                emptyMessage="Tidak ada objek pertanggungan pada klaim ini."
              />
            </Section>

            <Section title="Riwayat Progress">
              <DataTable
                columns={PROGRESS_COLUMNS}
                rows={detail.data.riwayat_progres}
                rowKey={(row) =>
                  `${row.tanggal_input}|${row.status_progres_1}|${row.keterangan}`
                }
                hideSearch
                emptyMessage="Belum ada catatan progres untuk klaim ini."
              />
            </Section>

            <Section title="KOMUNIKASI DENGAN LOSS ADJUSTER">
              <DataTable
                columns={MESSAGE_COLUMNS}
                rows={detail.data.komunikasi_adjuster}
                rowKey={(row) => `${row.tanggal_proses}|${row.nama_user}|${row.pesan}`}
                hideSearch
                emptyMessage="Belum ada komunikasi dengan loss adjuster."
              />
            </Section>

            <DetailPlannedDifferences lines={detail.data.selisih_terencana} />
          </div>
        ) : null}
      </div>
    </div>
  )
}

/**
 * Summary menggambar delapan nilai berlabel pada kepala popup.
 *
 * Label ditulis PERSIS seperti di layar lama, termasuk bahasa campurnya — "Occupation :",
 * "Kronologi :", "Note dari PIC :". `D-13` menetapkan teks yang dilihat pengguna mengikuti
 * Pega, dan menerjemahkan sebagiannya justru membuat layar tidak dikenali lagi.
 */
function Summary({ header }: { header: DetailHeader }) {
  return (
    <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2">
      <Entry label="COB">{header.cob || '—'}</Entry>
      <Entry label="Occupation :">{header.occupation || '—'}</Entry>
      <Entry label="Total Sum Insured :">{formatRupiah(header.total_sum_insured)}</Entry>
      <Entry label="Total Reserve :">{formatRupiah(header.total_reserve)}</Entry>

      <Entry label="Aging :">
        {/*
          Ditandai dengan cara yang SAMA dengan barisnya di grid, dan keterangannya ikut
          terbawa lewat `title` — bukan hanya lewat warna, yang tidak sampai ke semua orang.
        */}
        <span
          className={header.perlu_perhatian ? 'font-semibold text-red-700' : undefined}
          title={
            header.perlu_perhatian
              ? 'Umur klaim melewati ambang yang ditetapkan.'
              : undefined
          }
        >
          {header.aging_hari} hari
        </span>
      </Entry>

      <Entry label="Dominant Factor :">{header.dominant_factor || '—'}</Entry>

      <Entry label="Kronologi :" wide>
        {header.kronologi || '—'}
      </Entry>
      <Entry label="Claim Recommendation :" wide>
        {header.claim_recommendation || '—'}
      </Entry>
      <Entry label="Note dari PIC :" wide>
        {header.note_pic || '—'}
      </Entry>
    </dl>
  )
}

/** Entry menggambar satu pasang label dan nilai. */
function Entry({
  label,
  children,
  wide = false,
}: {
  label: string
  children: ReactNode
  wide?: boolean
}) {
  return (
    <div className={wide ? 'sm:col-span-2' : undefined}>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </dt>
      <dd className="mt-1 whitespace-pre-wrap text-sm text-slate-900">{children}</dd>
    </div>
  )
}

/** Section membungkus satu grid beserta judulnya. */
function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h3 className="mb-2 text-sm font-semibold text-slate-900">{title}</h3>
      {children}
    </section>
  )
}

/**
 * objectColumnsFor memilih varian kolom grid objek menurut lini bisnis.
 *
 * # TIGA varian, dan urutannya terbaca dari section
 *
 * Section lama memuat tiga susunan kolom berbeda, masing-masing didahului rujukan when rule.
 * Posisi keduanya di dalam berkas menunjukkan pasangannya:
 *
 *	IsAneka · IsMarineCargo · IsFire  @136k  ->  "Lokasi Object"  @167k   varian A
 *	IsPA                             @210k  ->  "Pekerjaan"      @247k   varian B
 *	IsTravel                         @292k  ->  "Nama Peserta"   @328k   varian C
 *
 * # Kelima when rule ADA di export, dan isinya sudah dibaca
 *
 *	IsPA            pyWorkPage.Policy.Quotation.GroupPanel = "002"
 *	IsTravel        Kode Bisnis = "77"
 *	IsAneka         Kode Bisnis = 24 | 18 | 17 | 10 | 07 | 06
 *	IsMarineCargo   Kode Bisnis = 24 | 18 | 17 | 10 | 07 | 06   (IDENTIK dengan IsAneka)
 *	IsFire          kosong — "Double click to add condition"
 *
 * Hanya `IsPA` yang dapat dipetakan tepat: `GroupPanel = "002"` sama persis dengan `cob`
 * bernilai `"PA"` pada kueri daftar.
 *
 * # Dua hal yang masih tebakan, dan dinyatakan sebagai tebakan
 *
 * `IsTravel` menyaring **Kode Bisnis** `"77"`, bukan `GroupPanel`. Layar ini tidak membawa
 * kode bisnis, sehingga yang dipakai `cob === 'Travel'` — hasil terjemahan `grouppanel = 005`.
 * Keduanya BELUM dibuktikan menunjuk himpunan klaim yang sama.
 *
 * `IsAneka` dan `IsMarineCargo` bersyarat IDENTIK, dan `IsFire` tanpa syarat sama sekali.
 * Ketiganya karena itu tidak dapat saling dibedakan — dan memang tidak perlu: ketiganya
 * mengarah ke varian yang sama.
 *
 * # Kenapa pemilihannya di sini, bukan di peladen
 *
 * Peladen mengirim SELURUH kolom ketiga varian. Menaruh pemilihan di layar membuat kedua
 * tebakan di atas dapat diperbaiki tanpa menyentuh penyimpanan; menaruhnya di kueri akan
 * menguncinya ke dalam SQL.
 */
function objectColumnsFor(cob: string): Column<DetailObject>[] {
  // Varian B — PA. `IsPA` berpasangan dengan susunan berkolom "Pekerjaan", BUKAN dengan
  // susunan peserta. Keduanya sempat tertukar di sini, dan tertukarnya tidak menghasilkan
  // galat apa pun: grid tetap terisi, hanya kolomnya yang milik lini lain.
  if (cob === 'PA') {
    return [
      { key: 'nama', title: 'Nama Objek', value: (row) => row.nama || '—' },
      { key: 'pekerjaan', title: 'Pekerjaan', value: (row) => row.pekerjaan || '—' },
      {
        key: 'tanggal_lahir',
        title: 'Tanggal lahir',
        value: (row) => formatDate(row.tanggal_lahir) || '—',
      },
    ]
  }

  // Varian C — Travel. Ejaan "Tanggal Lahir" di sini memang berbeda dari "Tanggal lahir"
  // pada varian PA; keduanya dibawa apa adanya (`D-13`).
  if (cob === 'Travel') {
    return [
      { key: 'nama', title: 'Nama Peserta', value: (row) => row.nama || '—' },
      {
        key: 'status_peserta',
        title: 'Status',
        value: (row) => row.status_peserta || '—',
      },
      { key: 'ktp_paspor', title: 'KTP/Paspor', value: (row) => row.ktp_paspor || '—' },
      {
        key: 'tanggal_lahir',
        title: 'Tanggal Lahir',
        value: (row) => formatDate(row.tanggal_lahir) || '—',
      },
    ]
  }

  // Varian A — Aneka, Marine Cargo, Fire, dan selebihnya.
  return [
    { key: 'nama', title: 'Nama Objek', value: (row) => row.nama || '—' },
    { key: 'lokasi', title: 'Lokasi Object', value: (row) => row.lokasi || '—' },
  ]
}

/**
 * PROGRESS_COLUMNS adalah delapan kolom riwayat progres.
 *
 * Judulnya HURUF BESAR seperti di layar lama, dan ejaan "Keterangan" yang berhuruf kecil di
 * antara tujuh yang kapital juga dibawa apa adanya (`D-13`). Di Pega judul terakhir itu
 * bahkan ditulis `<b>Keterangan<b>` dengan tag penutup yang salah ketik; yang dibawa
 * teksnya, bukan salah ketiknya.
 */
const PROGRESS_COLUMNS: Column<DetailProgress>[] = [
  {
    key: 'tanggal_input',
    title: 'TANGGAL INPUT',
    value: (row) => row.tanggal_input || '—',
  },
  { key: 'no_klaim', title: 'NOMOR KLAIM', value: (row) => row.no_klaim || '—' },
  {
    key: 'status_progres_1',
    title: 'STATUS PROGRESS 1',
    value: (row) => row.status_progres_1 || '—',
  },
  {
    key: 'status_progres_2',
    title: 'STATUS PROGRESS 2',
    value: (row) => row.status_progres_2 || '—',
  },
  { key: 'user_input', title: 'USER INPUT', value: (row) => row.user_input || '—' },
  {
    key: 'tanggal_next_followup',
    title: 'TANGGAL NEXT FOLLOWUP',
    value: (row) => formatDate(row.tanggal_next_followup) || '—',
  },
  { key: 'status', title: 'STATUS', value: (row) => row.status || '—' },
  { key: 'keterangan', title: 'Keterangan', value: (row) => row.keterangan || '—' },
]

/**
 * MESSAGE_COLUMNS adalah lima kolom komunikasi dengan loss adjuster.
 *
 * Kolom pertama menandai pengirim INTERNAL. Penandanya tidak digambar sebagai kolom
 * tersendiri — ia melekat pada nama, karena tanpa itu percakapan terbaca seolah satu pihak
 * saja dan pembacanya tidak dapat tahu mana pesan keluar.
 */
const MESSAGE_COLUMNS: Column<DetailMessage>[] = [
  {
    key: 'nama_user',
    title: 'Nama User',
    value: (row) => row.nama_user || '—',
    render: (row) => (
      <span>
        {row.nama_user || '—'}
        {row.internal ? (
          <span className="ml-2 rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-600">
            internal
          </span>
        ) : null}
      </span>
    ),
  },
  {
    key: 'tanggal_proses',
    title: 'Tanggal Proses',
    value: (row) => row.tanggal_proses || '—',
  },
  { key: 'pesan', title: 'Pesan', value: (row) => row.pesan || '—' },
  {
    key: 'tanggal_balas',
    title: 'Tanggal Balas',
    value: (row) => row.tanggal_balas || '—',
  },
  { key: 'jawaban', title: 'Jawaban', value: (row) => row.jawaban || '—' },
]

/**
 * DetailPlannedDifferences menampilkan selisih POPUP terhadap layar Pega.
 *
 * Daftarnya berbeda dari milik layar daftar, dan itu disengaja: catatan tentang
 * "Total Sum Insured" tidak berlaku di grid, dan catatan tentang paginasi tidak berlaku di
 * sini (`D-54`).
 */
function DetailPlannedDifferences({ lines }: { lines: string[] }) {
  if (lines.length === 0) return null

  return (
    <details className="rounded-kartu border border-slate-200 bg-slate-50 p-4 text-sm">
      <summary className="cursor-pointer font-medium text-slate-700">
        Perbedaan yang disengaja terhadap layar lama
      </summary>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-slate-600">
        {lines.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </details>
  )
}

/**
 * isNotFound membedakan "klaim tidak ditemukan" dari gangguan lain.
 *
 * Dicocokkan lewat KODE, bukan lewat teks pesan: teks dapat berubah kapan saja tanpa
 * mengubah artinya, dan pencocokan teks akan diam-diam berhenti bekerja ketika itu terjadi.
 */
function isNotFound(error: unknown): boolean {
  return error instanceof APIError && error.kode === 'klaim_tidak_ditemukan'
}

/** messageOf membaca pesan galat yang layak dibaca pengguna. */
function messageOf(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  return 'Detail klaim tidak dapat dimuat. Coba lagi beberapa saat lagi.'
}
