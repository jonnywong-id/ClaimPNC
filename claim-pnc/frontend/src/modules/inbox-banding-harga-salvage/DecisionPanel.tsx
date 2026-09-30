import type { ReactNode } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useBandingHargaSalvageDecisions } from './api'
import { DecisionCode, type Decision, type DecisionColumn } from './types'

type Props = {
  /** Nomor klaim yang dibuka. Kosong berarti panel tertutup. */
  claimNo: string
  columns: DecisionColumn[]
  onClose: () => void
}

/**
 * Panel rincian pada grid **History Cheker** — seluruh keputusan banding harga satu klaim.
 *
 * # Apa yang digantikan
 *
 *   Flow Action/DetailHistoryRequestSalvage   pembungkusnya; dikunci nomor klaim
 *   Section/DetailHistReqSalvage              ketujuh kolomnya
 *   Activity/ShowDtlHistoryReqSalvage_Act     pemasoknya
 *   RDB List/DetailHistReqSalvage_SQL         kuerinya
 *
 * # Kenapa ia DAFTAR, bukan satu baris
 *
 * Karena satu klaim dapat punya beberapa barang yang dibanding harganya, dan tiap barang
 * punya keputusannya sendiri. Grid History menampilkan pengajuan salvage; panel ini
 * menampilkan setiap keputusan di bawah klaim yang sama.
 *
 * # Kenapa ia panel, bukan halaman tersendiri
 *
 * Karena di Pega pun ia flow action — dibuka di atas gridnya, lalu ditutup. Menjadikannya
 * halaman berarti pengguna kehilangan tempatnya di daftar setiap kali membuka satu klaim,
 * dan pada antrean yang dibaca berurutan itu terasa setiap kali.
 */
export function DecisionPanel({ claimNo, columns, onClose }: Props) {
  const decisions = useBandingHargaSalvageDecisions(claimNo)

  if (claimNo === '') return null

  return (
    <section
      className="mt-4 rounded-kartu border border-slate-200 bg-white"
      aria-label={`Riwayat keputusan banding klaim ${claimNo}`}
    >
      <header className="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold text-slate-900">
            Riwayat Keputusan Banding
          </h2>
          <p className="mt-0.5 text-xs text-slate-600">
            Klaim <span className="font-medium text-slate-800">{claimNo}</span>
          </p>
        </div>

        <Button tone="halus" onClick={onClose} aria-label="Tutup riwayat keputusan">
          Tutup
        </Button>
      </header>

      <div className="px-4 pb-4">
        <DataTable<Decision>
          columns={columnsFor(columns)}
          rows={decisions.data?.baris ?? []}
          rowKey={(row) => `${row.detail_object}|${row.tanggal_approve ?? ''}`}
          label={`Keputusan banding klaim ${claimNo}`}
          hideSearch
          isLoading={decisions.isPending}
          error={
            decisions.isError ? (
              <ErrorMessage
                title="Riwayat tidak dapat dimuat"
                description={messageOf(decisions.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          emptyMessage={
            'Belum ada keputusan yang tercatat untuk klaim ini. Itu keadaan yang sah — ' +
            'baris pada History menampilkan pengajuan salvage, dan keputusannya dicatat ' +
            'per barang.'
          }
        />
      </div>
    </section>
  )
}

/** columnsFor menyusun kolom panel dari bentuk yang ditetapkan server. */
function columnsFor(columns: DecisionColumn[]): Column<Decision>[] {
  return columns.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => valueOf(row, column),
    render: (row) => renderCell(row, column),
    alignRight: column.angka,
  }))
}

function valueOf(row: Decision, column: DecisionColumn): string {
  const raw = row[column.kunci]
  if (raw === null || raw === undefined) return ''
  return String(raw)
}

/**
 * renderCell menggambar satu sel.
 *
 * Kolom "Jawaban Checker" digambar sebagai LENCANA berwarna, bukan teks biasa. Alasannya
 * bukan hiasan: panel ini dibaca untuk satu hal — barang mana yang harganya diterima dan mana
 * yang tidak — dan dua kalimat yang hanya berbeda satu kata ("Setuju" dan "Tidak setuju")
 * sulit dibedakan sekilas pada daftar yang panjang.
 */
function renderCell(row: Decision, column: DecisionColumn): ReactNode {
  const raw = valueOf(row, column)
  if (raw === '') return <span className="text-slate-400">—</span>

  if (column.kunci === 'tanggal_approve') {
    return formatDate(raw)
  }

  if (column.kunci === 'jawaban_checker') {
    return <DecisionBadge code={row.jawaban_checker_kode} label={raw} />
  }

  if (column.angka) {
    return <span className="tabular-nums">{formatMoney(raw)}</span>
  }

  return raw
}

/**
 * Lencana keputusan.
 *
 * Warnanya dipilih dari KODE, bukan dari teksnya — teks dapat berubah tanpa mengubah artinya,
 * dan mencocokkan kalimat berarti lencana menjadi kelabu diam-diam begitu kalimatnya disunting.
 *
 * Kode yang tidak dikenal digambar netral beserta teksnya apa adanya. Itu disengaja:
 * `DecisionLabel` di server mengembalikan "PROSES" untuk kode asing, dan menyembunyikannya di
 * balik warna "ditolak" akan menyatakan sesuatu yang belum tentu benar.
 */
function DecisionBadge({ code, label }: { code: string; label: string }) {
  const tone =
    code === DecisionCode.approved
      ? 'bg-emerald-50 text-emerald-800 ring-emerald-200'
      : code === DecisionCode.rejected
        ? 'bg-rose-50 text-rose-800 ring-rose-200'
        : 'bg-slate-100 text-slate-700 ring-slate-200'

  return (
    <span
      className={[
        'inline-flex items-center rounded-full px-2 py-0.5',
        'text-xs font-medium ring-1 ring-inset whitespace-nowrap',
        tone,
      ].join(' ')}
    >
      {label}
    </span>
  )
}

/**
 * formatMoney menggambar nilai uang dengan pemisah ribuan.
 *
 * Nilainya datang sebagai TEKS dan tetap teks sampai di sini — nilai uang disimpan presisi
 * penuh dan hanya dibulatkan saat ditampilkan. Nilai yang tidak terbaca sebagai angka
 * digambar apa adanya, bukan diganti nol.
 */
function formatMoney(raw: string): string {
  const parsed = Number(raw.replace(',', '.'))
  if (!Number.isFinite(parsed)) return raw
  return parsed.toLocaleString('id-ID', { maximumFractionDigits: 2 })
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
