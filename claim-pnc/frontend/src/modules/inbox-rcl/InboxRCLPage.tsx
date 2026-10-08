import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { useSelectedPortal } from '@/app/portal'
import { portalGate } from '@/components/inbox/InboxNotices'
import { TaskQueueTable } from '@/components/inbox/TaskQueueTable'
import { apiMessageOf } from '@/components/inbox/messages'
import { InboxPageFrame } from '@/components/inbox/InboxPageFrame'
import {
  PlainText,
  buildTaskColumns,
  commonTaskRenderers,
  type TaskRenderer,
} from '@/components/inbox/taskColumns'

import { PAGE_SIZE, useDaftarRCL, useKeteranganRCL } from './api'
import type { KolomLayar, TugasRCL } from './types'

/**
 * Inbox RCL — menu `MENU_ID 62`, pengganti harness `RCL_Harness`.
 *
 * Isinya antrean **penolakan yang memerlukan pertimbangan medis** milik satu dokter RCL:
 * klaim Personal Accident yang dikirim analis ke tahap `RCLDokter` (`Flow/Register_Flow.xml`,
 * `Assignment12`). Per `D-79` ia benar-benar Inbox.
 *
 * JANGAN tertukar dengan Inbox RCL/PUCL (`MENU_ID 61`): yang itu antrean BERSAMA, yang ini
 * antrean PER ORANG.
 *
 * # Susunan layar
 *
 * Kelima judul kolomnya diambil apa adanya dari rule `pyCaption …` di harness, urutannya dari
 * kelima sel berkepala pada `Section/InboxRCLDokter_Section-Section.xml`, dan judulnya
 * datang dari SERVER.
 *
 * # Tampilan sama dengan Pega — tanpa catatan di layar
 *
 * Antrean disaring dengan identitas LAMA pemanggil. Bila identitas lama tidak ditemukan,
 * atau kolom tabelnya belum terisi, layar hanya menampilkan grid kosong — persis seperti
 * layar lama. Work Owner memutuskan 2026-09-27: tidak ada peringatan merah maupun catatan
 * "selisih terencana/keterbatasan" di layar; penjelasannya hanya di kode dan
 * `docs/catatan-pengembangan.md` §64.
 *
 * # Layar ini hanya MEMBACA
 *
 * Menyelesaikan tugas RCL Dokter berarti menjalankan Flow Action `SendToRCLDokter`, yang
 * memindahkan penugasan — milik Pega selama masa paralel (`P-1`).
 */
export function InboxRCLPage() {
  const navigate = useNavigate()
  const portal = useSelectedPortal((state) => state.alias)

  const [cari, setCari] = useState('')
  const [lewati, setLewati] = useState(0)

  const keterangan = useKeteranganRCL()
  const daftar = useDaftarRCL(cari, lewati)

  /** Mengubah kata kunci SEKALIGUS kembali ke halaman pertama. */
  function ubahPencarian(nilai: string) {
    setCari(nilai)
    setLewati(0)
  }

  /**
   * Halaman yang ternyata kosong di luar halaman pertama dikembalikan ke halaman pertama.
   *
   * Server menghitung total lewat `COUNT(*) OVER ()`, sehingga halaman kosong tidak membawa
   * total — tanpa ini, pengguna yang halamannya mendadak kosong (tugasnya diselesaikan di
   * Pega) kehilangan bilah halaman dan jalan kembali.
   */
  const halamanKosongDiLuarAwal =
    lewati > 0 && daftar.isSuccess && !daftar.isPlaceholderData && daftar.data.data.length === 0
  useEffect(() => {
    if (halamanKosongDiLuarAwal) setLewati(0)
  }, [halamanKosongDiLuarAwal])

  /**
   * Tujuan tautan baris. Di sistem lama sel pertama membuka penugasannya lewat
   * `SetAssignmentInboxRCLDoctor_act` (`ASSIGN-WORKLIST <pzInsKey>!Register_Flow`). Yang
   * dibuka di sini tampilan klaimnya lewat nomor case, sama seperti enam inbox lain (`D-22`).
   */
  function bukaKlaim(nomorCase: string) {
    navigate(`/view-claim/${encodeURIComponent(nomorCase)}`)
  }

  const gate = portalGate(portal, 'Antrean RCL')
  if (gate) return <PageFrame>{gate}</PageFrame>

  const kolom = keterangan.data?.kolom ?? []

  return (
    <PageFrame>

      <div className="mt-6">
        <TaskQueueTable<TugasRCL>
          columns={buildTaskColumns(kolom, renderer, bukaKlaim)}
          // Gabungan `INNER JOIN` ke worklist dapat memunculkan satu klaim dua kali (`P-5`),
          // sehingga kuncinya gabungan klaim_id dan nomor case.
          rowKey={(row) => `${row.klaim_id}|${row.nomor_case}`}
          title="Antrean RCL Dokter"
          label="Antrean Inbox RCL"
          description={(total) => `${total} klaim menunggu pertimbangan Anda.`}
          searchLabel="Cari Nomor Case / No Polis"
          emptyMessage="Tidak ada klaim RCL untuk Anda saat ini."
          search={cari}
          onSearch={ubahPencarian}
          offset={lewati}
          onOffset={setLewati}
          pageSize={PAGE_SIZE}
          query={daftar}
          describe={apiMessageOf}
        />
      </div>
    </PageFrame>
  )
}

type Renderer = TaskRenderer<TugasRCL, KolomLayar>

const renderer: Record<string, Renderer | undefined> = {
  ...commonTaskRenderers<TugasRCL, KolomLayar>(),

  tanggal_masuk_inbox: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '10rem',
    value: (row) => row.tanggal_masuk_inbox,
    render: (row) => (
      <span className="tabular-nums" title={k.keterangan ?? ''}>
        <PlainText value={row.tanggal_masuk_inbox} />
      </span>
    ),
  }),

  deskripsi_analyst: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '22rem',
    value: (row) => row.deskripsi_analyst,
    render: (row) =>
      row.deskripsi_analyst ? (
        <span className="block whitespace-pre-line text-slate-700">{row.deskripsi_analyst}</span>
      ) : (
        <span className="text-slate-400">—</span>
      ),
  }),
}

function PageFrame({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <InboxPageFrame
      title="Inbox RCL"
      intro={
        'Klaim yang dikirim analis untuk pertimbangan penolakan medis Anda. Barisnya ' +
        'hilang begitu tugasnya diselesaikan.'
      }
    >
      {children}
    </InboxPageFrame>
  )
}

