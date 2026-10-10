import { APIError } from '@/api/client'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useState } from 'react'

import { Button } from '@/components/Button'

import { PAGE_SIZE, useTampunganPIC } from './api'
import { DialogTransfer } from './DialogTransfer'
import type { BarisTampungan } from './types'

/**
 * Tab **Inbox Tampungan PIC** — tab kedua layar lama.
 *
 * Isinya klaim yang SUDAH terdaftar tetapi BELUM punya PIC Teknik: tugasnya masih diparkir
 * di akun penampung `ServicePNC`. Menggantikan `RDB List/BrowseCaseNotAssigned-SQL.xml` dan
 * `RDB List/CountCaseNotAssigned-SQL.xml`.
 *
 * # Tanpa penyaring lini bisnis
 *
 * Kueri lamanya tidak punya penandanya, dan layar lamanya tidak menggambar dropdown Bisnis
 * pada tab ini. Satu-satunya penyaringnya kotak cari No Klaim.
 *
 * # Tombol Transfer MEMINDAHKAN
 *
 * KOREKSI (2026-10-06): sebelumnya tombol ini hanya mencatat permintaan, dengan alasan `P-1`
 * — `DATAPEGA.PC_ASSIGN_WORKLIST` ditulis Pega selama masa paralel. Alasan itu salah sasaran:
 * yang dipindahkan BUKAN penugasan Pega melainkan **PIC Teknik klaim** (`USERTEKNIS_1`), dan
 * `PNC_ReassignPNCTeknik` — activity di balik tombol yang sama di Pega — juga tidak menyentuh
 * `PC_ASSIGN_WORKLIST`.
 *
 * Antreannya karena itu dicabut: ia tidak punya pelaksana, dan baris di tab ini akan menumpuk
 * selamanya. Tab inilah yang paling dirugikan oleh itu — isinya justru klaim yang menunggu
 * diberi PIC.
 *
 * # TANPA tombol unduh — berbeda dari keempat tile
 *
 * Tab ini pernah punya tombol "Export to Excel"; ia **dicabut 2026-10-07** karena layar lama
 * tidak punya padanannya. Dibuktikan dua cara: `DashboardClaim_Section2-Section.xml` memuat
 * **nol** `openUrlInWindow` — cara keempat ekspor tile dipanggil, yang muncul 4× di
 * `DashboardClaim_Section1` — dan tidak ada satu pun activity `Export*` di export yang
 * menyentuh klaim ber-`USERTEKNIS_1 IS NULL`.
 *
 * Yang ada di tab ini hanya dua: kotak cari (`SearchCaseNotAssigned_act`) dan tombol Transfer
 * per baris (`GCNMTransferAssignmentManager_act`).
 */
export function TampunganPIC({
  cari,
  onCari,
  halaman,
  onHalaman,
}: {
  cari: string
  onCari: (nilai: string) => void
  halaman: number
  onHalaman: (halaman: number) => void
}) {
  const daftar = useTampunganPIC(true, { cari, halaman })

  const [transfer, setTransfer] = useState<BarisTampungan | null>(null)

  const keterangan = daftar.data?.halaman
  const pagination = {
    page: keterangan?.halaman ?? halaman,
    size: keterangan?.ukuran ?? PAGE_SIZE,
    total: keterangan?.total ?? 0,
    totalPage: keterangan?.total_halaman ?? 1,
    onPageChange: onHalaman,
    isLoading: daftar.isFetching,
  }

  return (
    <>
      <p className="mb-4 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
        Klaim yang sudah terdaftar tetapi <strong>belum dipegang PIC Teknik</strong>. Tombol{' '}
        <em>Transfer</em> menyerahkannya kepada petugas yang dipilih, dan barisnya hilang dari
        sini setelah daftarnya dimuat ulang.
      </p>

      <DataTable<BarisTampungan>
        title="Inbox Tampungan PIC"
        label="Daftar klaim di tampungan PIC"
        columns={kolomTampungan(setTransfer)}
        rows={daftar.data?.klaim ?? []}
        rowKey={(row) => row.klaim_id || row.nomor_klaim}
        isLoading={daftar.isPending}
        error={
          daftar.isError ? (
            <ErrorMessage
              tone="gangguan"
              title="Daftar tidak dapat dibaca"
              description={pesanGalat(daftar.error)}
            />
          ) : undefined
        }
        serverSearch={{ value: cari, onChange: onCari }}
        searchLabel="Cari No Klaim"
        emptyMessage="Tidak ada klaim di tampungan yang cocok dengan pencarian."
        showHeaderWhenEmpty
        pagination={pagination}
      />

      {transfer !== null ? (
        <DialogTransfer
          lingkup="baris"
          nomorKlaim={transfer.nomor_klaim}
          klaimID={transfer.klaim_id}
          // Tab Tampungan tidak punya penyaring lini bisnis — kuerinya pun tidak (`D-73`).
          // Daftar PIC karena itu tidak dapat disaring dari sini, dan dialog menyatakannya.
          onTutup={() => setTransfer(null)}
        />
      ) : null}
    </>
  )
}

/**
 * Kolom tab Inbox Tampungan PIC, mengikuti grid layar lama apa adanya:
 *
 *	No Klaim · No Polis · Nama Tertanggung · Nama Bisnis · Sumber Bisnis · Nama Cabang ·
 *	Tanggal Pendaftaran · Admin PNC
 *
 * PIC Teknik TIDAK ada di sini, dan itu bukan kelalaian: menurut definisi tab ini, klaimnya
 * memang belum punya PIC Teknik. Menggambar kolomnya berarti menggambar kolom yang setiap
 * barisnya pasti kosong.
 */
function kolomTampungan(
  onTransfer: (row: BarisTampungan) => void,
): Column<BarisTampungan>[] {
  return [
  { key: 'nomor_klaim', title: 'No Klaim', value: (row) => row.nomor_klaim, width: '9rem' },
  { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis, width: '11rem' },
  { key: 'nama_tertanggung', title: 'Nama Tertanggung', value: (row) => row.nama_tertanggung },
  { key: 'nama_bisnis', title: 'Nama Bisnis', value: (row) => row.nama_bisnis },
  { key: 'sumber_bisnis', title: 'Sumber Bisnis', value: (row) => row.sumber_bisnis },
  { key: 'nama_cabang', title: 'Nama Cabang', value: (row) => row.nama_cabang },
  {
    key: 'tanggal_pendaftaran',
    title: 'Tanggal Pendaftaran',
    value: (row) => row.tanggal_pendaftaran,
    render: (row) => formatDate(row.tanggal_pendaftaran),
    width: '10rem',
  },
  { key: 'admin_pnc', title: 'Admin PNC', value: (row) => row.admin_pnc },
  {
    key: 'aksi',
    title: 'Aksi',
    // Kolom aksi tidak diurutkan dan tidak dicari: isinya tombol, bukan data.
    noSort: true,
    value: () => '',
    width: '7rem',
    render: (row) => (
      <Button tone="kedua" onClick={() => onTransfer(row)}>
        Transfer
      </Button>
    ),
  },
  ]
}

/** Membaca pesan galat yang layak dibaca pengguna. */
function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
