import { useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { useSelectedPortal } from '@/app/portal'
import { NoteList, portalGate } from '@/components/inbox/InboxNotices'
import { TaskQueueTable } from '@/components/inbox/TaskQueueTable'
import { apiMessageOf } from '@/components/inbox/messages'
import { InboxPageFrame } from '@/components/inbox/InboxPageFrame'
import {
  buildTaskColumns,
  commonTaskRenderers,
  textColumn,
  type TaskRenderer,
} from '@/components/inbox/taskColumns'

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

  const gate = portalGate(portal, 'Antrean penilaian medis')
  if (gate) return <PageFrame>{gate}</PageFrame>

  const kolom = keterangan.data?.kolom ?? []

  return (
    <PageFrame>
      <div className="mt-6">
        <TaskQueueTable<TugasAnalystDoctor>
          columns={buildTaskColumns(kolom, renderer, bukaKlaim)}
          // Kunci baris memakai klaim_id DAN nomor case sekaligus.
          //
          // Gabungan `INNER JOIN` ke worklist membuat satu klaim yang punya DUA penugasan
          // terbuka muncul DUA KALI — perilaku Pega yang sengaja dibawa (`P-5`). Memakai
          // klaim_id saja akan membuat React menemukan kunci ganda pada baris yang memang
          // seharusnya kembar.
          rowKey={(row) => `${row.klaim_id}|${row.nomor_case}`}
          title="Antrean penilaian medis"
          label="Antrean Inbox Analyst Doctor"
          description={(total) => `${total} tugas menunggu dinilai.`}
          searchLabel="Cari Nomor Case / No Polis"
          // Hanya keadaan "memang tidak ada" yang dinyatakan di sini.
          //
          // Keadaan "pencarian tidak cocok" TIDAK diurus layar ini: `DataTable` sudah
          // menggantinya sendiri menjadi `Tidak ada baris yang cocok dengan “…”` begitu
          // `serverSearch` terisi.
          emptyMessage="Tidak ada tugas penilaian medis untuk Anda saat ini."
          search={cari}
          onSearch={ubahPencarian}
          offset={lewati}
          onOffset={setLewati}
          pageSize={PAGE_SIZE}
          query={daftar}
          describe={apiMessageOf}
        />
      </div>

      <NoteList
        title="Perbedaan yang disengaja terhadap layar lama"
        lines={keterangan.data?.selisih_terencana ?? []}
      />
      <NoteList title="Yang perlu diketahui" lines={keterangan.data?.keterbatasan ?? []} />
    </PageFrame>
  )
}

/**
 * Cara menggambar tiap kolom, dikunci dengan `kunci` yang dikirim server.
 *
 * Judulnya TIDAK diketik di sini — ia diambil dari `k.judul`, supaya satu-satunya sumber
 * judul tetap backend.
 */
type Renderer = TaskRenderer<TugasAnalystDoctor, KolomLayar>

const renderer: Record<string, Renderer | undefined> = {
  ...commonTaskRenderers<TugasAnalystDoctor, KolomLayar>(),

  nama_cabang: (k) => textColumn(k, '10rem', (row) => row.nama_cabang),

  tanggal_pendaftaran: (k) => textColumn(k, '9rem', (row) => row.tanggal_pendaftaran),

  nama_admin: (k) => textColumn(k, '10rem', (row) => row.nama_admin),

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

function PageFrame({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <InboxPageFrame
      title="Inbox Analyst Doctor"
      intro={
        'Klaim yang menunggu penilaian medis Anda. Barisnya hilang begitu tugasnya ' +
        'diselesaikan.'
      }
    >
      {children}
    </InboxPageFrame>
  )
}

