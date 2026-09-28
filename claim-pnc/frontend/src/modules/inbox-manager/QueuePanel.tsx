import { useEffect, useState } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import type { Pagination, QueueRow, Tab, Verdict } from './types'

type Props = {
  tab: Tab
  rows: QueueRow[]
  pagination?: Pagination | undefined
  isLoading: boolean

  page: number
  onPageChange: (page: number) => void

  onDecide: (verdict: Verdict, keys: string[], reason: string) => void
  isDeciding: boolean

  /** Pesan hasil keputusan terakhir; kosong berarti belum ada. */
  resultMessage?: string | undefined
  decideError?: string | undefined
}

/**
 * Satu antrean persetujuan beserta bilah keputusannya.
 *
 * # Kenapa pemilihan baris ada di sini, bukan di DataTable
 *
 * Karena `DataTable` dipakai lebih dari enam puluh layar, dan hampir seluruhnya tidak
 * memutuskan apa pun. Menambahkan kolom pilih ke komponen bersama akan menyentuh semuanya
 * demi satu layar — dan `U-2` justru ada supaya satu perubahan tidak menyentuh ratusan layar.
 *
 * Kotak pilihnya karena itu digambar sebagai kolom biasa lewat `render`, dan "pilih semua"
 * berada di bilah keputusan — bukan di kepala tabel, yang komponennya memang tidak punya
 * tempat untuk itu.
 */
export function QueuePanel({
  tab,
  rows,
  pagination,
  isLoading,
  page,
  onPageChange,
  onDecide,
  isDeciding,
  resultMessage,
  decideError,
}: Props) {
  const [selected, setSelected] = useState<string[]>([])
  const [reason, setReason] = useState('')

  // Pilihan DIBUANG setiap kali antrean atau halamannya berganti.
  //
  // Tanpa ini, kunci yang dipilih di halaman 1 tetap ikut terkirim saat penyelia menekan
  // Setujui di halaman 2 — dan ia tidak akan melihat baris yang ikut terputuskan.
  useEffect(() => {
    setSelected([])
    setReason('')
  }, [tab.kode, page])

  const rule = tab.keputusan
  const approveBlocked = Boolean(rule.alasan_setuju_ditahan)
  const allSelected = rows.length > 0 && selected.length === rows.length

  const columns: Column<QueueRow>[] = [
    {
      key: 'pilih',
      title: 'Pilih',
      width: '3.5rem',
      noSort: true,
      value: (row) => (selected.includes(row.kunci) ? 'dipilih' : ''),
      render: (row) => (
        <input
          type="checkbox"
          aria-label={`Pilih baris ${row.kunci}`}
          checked={selected.includes(row.kunci)}
          onChange={() => toggle(row.kunci)}
          className="size-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
        />
      ),
    },
    ...(tab.kolom ?? []).map((column) => ({
      key: column.kunci,
      title: column.judul,
      value: (row: QueueRow) => row.sel[column.kunci] ?? '',
    })),
  ]

  function toggle(key: string) {
    setSelected((current) =>
      current.includes(key) ? current.filter((item) => item !== key) : [...current, key],
    )
  }

  function toggleAll() {
    setSelected(allSelected ? [] : rows.map((row) => row.kunci))
  }

  return (
    <div className="space-y-4">
      <DataTable<QueueRow>
        title={tab.nama}
        description={tab.keterangan}
        label={tab.nama}
        columns={columns}
        rows={rows}
        rowKey={(row) => row.kunci}
        isLoading={isLoading}
        hideSearch
        emptyMessage="Tidak ada pengajuan yang menunggu persetujuan di antrean ini."
        {...(pagination
          ? {
              pagination: {
                page: pagination.halaman,
                size: pagination.ukuran,
                total: pagination.total,
                totalPage: pagination.total_halaman,
                onPageChange,
                isLoading,
              },
            }
          : {})}
      />

      {rule.dapat_diputuskan ? (
        <div className="space-y-3 rounded-kontrol border border-slate-200 bg-slate-50 p-4">
          <div className="flex flex-wrap items-center gap-3">
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input
                type="checkbox"
                aria-label="Pilih semua baris di halaman ini"
                checked={allSelected}
                onChange={toggleAll}
                disabled={rows.length === 0}
                className="size-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              Pilih semua di halaman ini
            </label>

            <span className="text-sm text-slate-500" aria-live="polite">
              {selected.length} baris dipilih
            </span>
          </div>

          {/*
            Isian alasan hanya digambar bila tabelnya PUNYA kolom alasan. Pada antrean yang
            tidak punya, ia tidak digambar sama sekali — bukan digambar lalu isinya dibuang
            diam-diam oleh server.
          */}
          {rule.label_alasan ? (
            <label className="block text-sm">
              <span className="mb-1 block font-medium text-slate-700">
                {rule.label_alasan}
                {rule.alasan_wajib_saat_menolak ? (
                  <span className="ml-1 text-slate-500">(wajib saat menolak)</span>
                ) : null}
              </span>
              <textarea
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                rows={2}
                className="w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                placeholder="Tuliskan alasannya supaya pengaju tahu apa yang harus diperbaiki."
              />
            </label>
          ) : null}

          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              disabled={selected.length === 0 || isDeciding || approveBlocked}
              title={rule.alasan_setuju_ditahan}
              onClick={() => onDecide('setujui', selected, reason)}
            >
              Setujui
            </Button>

            <Button
              type="button"
              tone="kedua"
              disabled={selected.length === 0 || isDeciding}
              onClick={() => onDecide('tolak', selected, reason)}
            >
              Tolak
            </Button>
          </div>

          {/*
            Alasan penahanan ditampilkan DI TEMPAT tombolnya, bukan disembunyikan di tooltip
            saja. Tombol nonaktif tanpa keterangan akan dilaporkan sebagai kerusakan.
          */}
          {approveBlocked ? (
            <p className="rounded-kontrol border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
              <span className="font-medium">Persetujuan sedang ditahan. </span>
              {rule.alasan_setuju_ditahan}
            </p>
          ) : null}

          {decideError ? (
            <ErrorMessage
              tone="penolakan"
              title="Keputusan tidak dapat disimpan"
              description={decideError}
            />
          ) : null}

          {resultMessage ? (
            <p className="text-sm text-slate-700" role="status">
              {resultMessage}
            </p>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
