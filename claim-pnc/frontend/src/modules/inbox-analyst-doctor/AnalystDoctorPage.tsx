import { useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { ReloadIcon } from '@/components/Icon'

import { PAGE_SIZE, useDaftarAnalystDoctor, useKeteranganAnalystDoctor } from './api'
import type { KolomLayar, TugasAnalystDoctor } from './types'

/**
 * Inbox Analyst Doctor — menu `MENU_ID 60`, pengganti harness `inboxAnalystDoctor_Harness`.
 *
 * Isinya antrean **penilaian medis** milik satu petugas: klaim yang menunggu dinilai tenaga
 * medis sebelum jalur analis ditutup. Per `D-79` ia benar-benar Inbox — barisnya tugas milik
 * pemanggil, hilang begitu tugasnya tuntas, dan "hanya milik saya" adalah aturan kewenangan,
 * bukan sekadar penyaring.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Berbeda dari Inbox Close Claim dan Inbox Outstanding yang harness-nya hilang, layar ini
 * LENGKAP di export. Kedelapan judul kolomnya diambil apa adanya dari rule `pyCaption …` di
 * dalam harness, dan urutannya dari kedelapan sel berkepala pada
 * `Section/InboxAnalystDoctor_Section-Section.xml`.
 *
 * Judul kolomnya karena itu datang dari SERVER, bukan diketik di sini: ia hasil pembacaan
 * export yang tercatat di backend, dan menyalinnya ke layar berarti daftar yang sama hidup
 * di dua tempat.
 *
 * # Layar lama tidak punya satu pun penyaring
 *
 * Report Definition-nya tidak menyalakan panel penyaring apa pun. Kotak cari di sini adalah
 * TAMBAHAN yang disadari, dan ia dinyatakan kepada pengguna sebagai selisih terencana —
 * bukan disamarkan sebagai fitur yang memang selalu ada.
 *
 * # Layar ini hanya MEMBACA
 *
 * Menyelesaikan tugas Analyst Doctor berarti menjalankan Flow Action `SendAnalystDoctor`,
 * yang memindahkan penugasan — dan penugasan masih dimiliki Pega selama masa paralel
 * (`P-1`). Tidak ada tombol yang mengubah apa pun di sini, dan ketiadaannya disengaja.
 */
export function AnalystDoctorPage() {
  const navigate = useNavigate()
  const portal = useSelectedPortal((state) => state.alias)

  const [cari, setCari] = useState('')
  const [lewati, setLewati] = useState(0)

  const keterangan = useKeteranganAnalystDoctor()
  const daftar = useDaftarAnalystDoctor(cari, lewati)

  /**
   * Mengubah kata kunci SEKALIGUS kembali ke halaman pertama.
   *
   * Tanpa penyetelan ulang itu, mengetik kata kunci saat berada di halaman lima akan
   * menampilkan halaman lima dari hasil yang baru — yang hampir selalu kosong, dan terbaca
   * sebagai "tidak ditemukan".
   */
  function ubahPencarian(nilai: string) {
    setCari(nilai)
    setLewati(0)
  }

  /**
   * Tujuan tautan baris.
   *
   * Di sistem lama, sel pertama grid adalah `Link` yang membuka klaimnya lewat kunci teknis
   * Pega. Yang dikirim di sini adalah nomor case-nya saja: `D-22` menetapkan sistem baru
   * tidak pernah menuliskan awalan kelas Pega lagi, dan penyusunan kunci itu — bila memang
   * masih dibutuhkan — adalah urusan modul View Claim, bukan urusan layar ini.
   */
  function bukaKlaim(nomorCase: string) {
    navigate(`/view-claim/${encodeURIComponent(nomorCase)}`)
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean penilaian medis milik satu badan hukum, dan aplikasi ini melayani ' +
            'empat. Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  const kolom = keterangan.data?.kolom ?? []
  const baris = daftar.data?.data ?? []
  const total = daftar.data?.total ?? 0
  const halaman = Math.floor(lewati / PAGE_SIZE) + 1
  const totalHalaman = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <PageFrame>
      <div className="mt-6">
        <DataTable<TugasAnalystDoctor>
          columns={buildColumns(kolom, bukaKlaim)}
          rows={baris}
          // Kunci baris memakai klaim_id DAN nomor case sekaligus.
          //
          // Gabungan `INNER JOIN` ke worklist membuat satu klaim yang punya DUA penugasan
          // terbuka muncul DUA KALI — perilaku Pega yang sengaja dibawa (`P-5`). Memakai
          // klaim_id saja akan membuat React menemukan kunci ganda pada baris yang memang
          // seharusnya kembar.
          rowKey={(row) => `${row.klaim_id}|${row.nomor_case}`}
          title="Antrean penilaian medis"
          label="Antrean Inbox Analyst Doctor"
          // Prop tidak dikirim sama sekali saat kosong, bukan dikirim bernilai undefined:
          // tsconfig memakai exactOptionalPropertyTypes, yang membedakan keduanya.
          {...(total > 0 ? { description: `${total} tugas menunggu dinilai.` } : {})}
          isLoading={daftar.isPending}
          searchLabel="Cari Nomor Case / No Polis"
          // Hanya keadaan "memang tidak ada" yang dinyatakan di sini.
          //
          // Keadaan "pencarian tidak cocok" TIDAK diurus layar ini: `DataTable` sudah
          // menggantinya sendiri menjadi `Tidak ada baris yang cocok dengan “…”` begitu
          // `serverSearch` terisi. Mencabangkannya di sini akan menghasilkan kode yang tidak
          // pernah berjalan — dan pesan yang berbeda dari layar lain untuk keadaan yang sama.
          emptyMessage="Tidak ada tugas penilaian medis untuk Anda saat ini."
          serverSearch={{ value: cari, onChange: ubahPencarian, matchCount: total }}
          actions={
            <Button
              tone="halus"
              onClick={() => void daftar.refetch()}
              disabled={daftar.isFetching}
            >
              <ReloadIcon className="h-4 w-4" />
              {daftar.isFetching ? 'Memuat…' : 'Muat ulang'}
            </Button>
          }
          error={
            daftar.isError ? (
              <ErrorMessage
                title="Antrean tidak dapat dimuat"
                description={pesanGalat(daftar.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          pagination={{
            page: halaman,
            size: PAGE_SIZE,
            total,
            totalPage: totalHalaman,
            onPageChange: (nomor) => setLewati((nomor - 1) * PAGE_SIZE),
            isLoading: daftar.isFetching,
          }}
        />
      </div>

      <Catatan
        judul="Perbedaan yang disengaja terhadap layar lama"
        baris={keterangan.data?.selisih_terencana ?? []}
      />
      <Catatan
        judul="Yang perlu diketahui"
        baris={keterangan.data?.keterbatasan ?? []}
      />
    </PageFrame>
  )
}

/**
 * Menyusun kolom tabel dari judul yang dikirim server.
 *
 * Yang datang dari server hanyalah KUNCI dan JUDULNYA; cara menggambarnya tetap milik layar.
 * Pemisahan itu disengaja: judul adalah hasil pembacaan export yang boleh berubah bila
 * bacaannya dikoreksi, sedangkan lebar kolom dan bentuk selnya adalah keputusan tampilan.
 *
 * Kunci yang tidak dikenal dilewati, bukan digambar kosong. Kolom baru menuntut keputusan
 * tampilan yang belum diambil, dan kolom kosong tanpa isi hanya menambah lebar tabel.
 */
function buildColumns(
  kolom: KolomLayar[],
  bukaKlaim: (nomorCase: string) => void,
): Column<TugasAnalystDoctor>[] {
  const hasil: Column<TugasAnalystDoctor>[] = []

  for (const k of kolom) {
    const dibangun = renderer[k.kunci]?.(k, bukaKlaim)
    if (dibangun) hasil.push(dibangun)
  }
  return hasil
}

/** Teks sel yang kosong digambar sebagai em dash, bukan dibiarkan hampa. */
function Teks({ nilai }: { nilai: string }) {
  if (!nilai) return <span className="text-slate-400">—</span>
  return <span className="truncate">{nilai}</span>
}

/**
 * Cara menggambar tiap kolom, dikunci dengan `kunci` yang dikirim server.
 *
 * Judulnya TIDAK diketik di sini — ia diambil dari `k.judul`, supaya satu-satunya sumber
 * judul tetap backend.
 */
type Renderer = (
  k: KolomLayar,
  bukaKlaim: (nomorCase: string) => void,
) => Column<TugasAnalystDoctor>

const renderer: Record<string, Renderer | undefined> = {
  nomor_case: (k, bukaKlaim) => ({
    key: k.kunci,
    title: k.judul,
    width: '11rem',
    value: (row) => row.nomor_case,
    render: (row) =>
      row.nomor_case ? (
        <button
          type="button"
          className="rounded-md font-mono text-xs font-medium text-blue-700 underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          onClick={() => bukaKlaim(row.nomor_case)}
        >
          {row.nomor_case}
        </button>
      ) : (
        <span className="text-slate-400">belum bernomor</span>
      ),
  }),

  nomor_polis: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '11rem',
    value: (row) => row.nomor_polis,
    render: (row) => <Teks nilai={row.nomor_polis} />,
  }),

  nama_tertanggung: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '14rem',
    value: (row) => row.nama_tertanggung,
    render: (row) => <Teks nilai={row.nama_tertanggung} />,
  }),

  nama_cabang: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '10rem',
    value: (row) => row.nama_cabang,
    render: (row) => <Teks nilai={row.nama_cabang} />,
  }),

  tanggal_pendaftaran: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '9rem',
    value: (row) => row.tanggal_pendaftaran,
    render: (row) => <Teks nilai={row.tanggal_pendaftaran} />,
  }),

  nama_admin: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '10rem',
    value: (row) => row.nama_admin,
    render: (row) => <Teks nilai={row.nama_admin} />,
  }),

  komentar_pic_teknis: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '18rem',
    value: (row) => row.komentar_pic_teknis,
    render: (row) =>
      row.komentar_pic_teknis ? (
        <span className="block whitespace-pre-line text-slate-700">
          {row.komentar_pic_teknis}
          {row.pic_teknis && (
            <span className="mt-0.5 block text-xs text-slate-500">— {row.pic_teknis}</span>
          )}
        </span>
      ) : (
        // Sel yang kosong MENJELASKAN dirinya, tidak dibiarkan hampa.
        //
        // Kolom ini masih kosong terhadap Oracle karena properti Pega-nya tidak terekspos.
        // Tanpa keterangan, sel kosong terbaca sebagai "PIC Teknis memang tidak menulis
        // apa-apa" — dan tidak ada seorang pun yang menanyakannya.
        <span className="text-xs text-slate-400" title={k.keterangan ?? ''}>
          belum terbawa
        </span>
      ),
  }),

  lama_hari: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '8rem',
    value: (row) => String(row.lama_hari),
    render: (row) => (
      <span className="tabular-nums text-slate-700">
        {row.lama_hari} hari
      </span>
    ),
  }),
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Analyst Doctor</h1>
        <p className="mt-1 text-sm text-slate-600">
          Klaim yang menunggu penilaian medis Anda. Barisnya hilang begitu tugasnya
          diselesaikan.
        </p>
      </header>
      {children}
    </div>
  )
}

/**
 * Catatan di bawah layar.
 *
 * Datang dari SERVER, bukan ditulis tetap di sini, supaya hilang dengan sendirinya begitu
 * penghalangnya hilang. Tanpa catatan ini, kolom yang belum terbawa dan kotak cari yang tidak
 * ada di Pega akan dilaporkan berulang kali sebagai kerusakan.
 */
function Catatan({ judul, baris }: { judul: string; baris: string[] }) {
  if (baris.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">{judul}</h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
        {baris.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </section>
  )
}

/** pesanGalat mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function pesanGalat(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
