import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { ReloadIcon } from '@/components/Icon'

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

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean RCL milik satu badan hukum, dan aplikasi ini melayani empat. Pilih ' +
            'portal di bilah atas untuk membukanya.'
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
        <DataTable<TugasRCL>
          columns={buildColumns(kolom, bukaKlaim)}
          rows={baris}
          // Gabungan `INNER JOIN` ke worklist dapat memunculkan satu klaim dua kali (`P-5`),
          // sehingga kuncinya gabungan klaim_id dan nomor case.
          rowKey={(row) => `${row.klaim_id}|${row.nomor_case}`}
          title="Antrean RCL Dokter"
          label="Antrean Inbox RCL"
          {...(total > 0 ? { description: `${total} klaim menunggu pertimbangan Anda.` } : {})}
          isLoading={daftar.isPending}
          searchLabel="Cari Nomor Case / No Polis"
          emptyMessage="Tidak ada klaim RCL untuk Anda saat ini."
          serverSearch={{ value: cari, onChange: ubahPencarian, matchCount: total }}
          actions={
            <Button
              tone="halus"
              onClick={() => { daftar.refetch() }}
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
    </PageFrame>
  )
}

/**
 * Menyusun kolom tabel dari judul yang dikirim server. Kunci yang tidak dikenal dilewati,
 * bukan digambar kosong.
 */
function buildColumns(
  kolom: KolomLayar[],
  bukaKlaim: (nomorCase: string) => void,
): Column<TugasRCL>[] {
  const hasil: Column<TugasRCL>[] = []
  for (const k of kolom) {
    const dibangun = renderer[k.kunci]?.(k, bukaKlaim)
    if (dibangun) hasil.push(dibangun)
  }
  return hasil
}

/** Teks sel yang kosong digambar sebagai em dash, bukan dibiarkan hampa. */
function Teks({ nilai }: Readonly<{ nilai: string }>) {
  if (!nilai) return <span className="text-slate-400">—</span>
  return <span className="truncate">{nilai}</span>
}

type Renderer = (k: KolomLayar, bukaKlaim: (nomorCase: string) => void) => Column<TugasRCL>

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

  tanggal_masuk_inbox: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '10rem',
    value: (row) => row.tanggal_masuk_inbox,
    render: (row) => (
      <span className="tabular-nums" title={k.keterangan ?? ''}>
        <Teks nilai={row.tanggal_masuk_inbox} />
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
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox RCL</h1>
        <p className="mt-1 text-sm text-slate-600">
          Klaim yang dikirim analis untuk pertimbangan penolakan medis Anda. Barisnya hilang
          begitu tugasnya diselesaikan.
        </p>
      </header>
      {children}
    </div>
  )
}

function pesanGalat(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
