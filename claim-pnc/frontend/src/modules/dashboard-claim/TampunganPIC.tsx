import { APIError } from '@/api/client'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useState } from 'react'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { Button } from '@/components/Button'

import { PAGE_SIZE, unduhTampungan, useTampunganPIC } from './api'
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
 * # Tombol Transfer tidak dibawa
 *
 * Layar lama menggambar tombol "Transfer" pada setiap baris — ia MENULIS, memindahkan tugas
 * ke operator lain. `P-1` menetapkan `DATAPEGA.PC_ASSIGN_WORKLIST` masih ditulis Pega selama
 * masa paralel, sehingga tab ini membaca saja.
 *
 * Dinyatakan di layar, bukan dihilangkan diam-diam: pengguna yang mencari tombolnya perlu
 * tahu ia belum ada, bukan mengira layarnya rusak.
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

  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const [transfer, setTransfer] = useState<BarisTampungan | null>(null)
  const [galatUnduh, setGalatUnduh] = useState<string | null>(null)

  async function unduh() {
    setGalatUnduh(null)
    try {
      await unduhTampungan(cari, token, portal)
    } catch (failure) {
      // Unduhan gagal TIDAK boleh diam: pengguna menekan tombol, tidak ada berkas yang
      // muncul, dan tanpa pesan ia tidak tahu apakah berkasnya kosong atau permintaannya
      // yang gagal.
      setGalatUnduh(pesanGalat(failure))
    }
  }

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
        <em>Transfer</em> mencatat permintaan pemindahan — penugasannya dipindahkan Pega
        selama masa paralel, jadi barisnya masih akan tampil di sini sesudahnya.
      </p>

      {galatUnduh !== null ? (
        <div className="mb-4">
          <ErrorMessage
            tone="gangguan"
            title="Unduhan gagal"
            description={galatUnduh}
          />
        </div>
      ) : null}

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
        actions={
          <Button tone="kedua" onClick={() => void unduh()}>
            Export to Excel
          </Button>
        }
      />

      {transfer !== null ? (
        <DialogTransfer
          lingkup="baris"
          nomorKlaim={transfer.nomor_klaim}
          klaimID={transfer.klaim_id}
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
