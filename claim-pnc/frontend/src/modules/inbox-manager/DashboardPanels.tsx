import { DataTable, type Column } from '@/components/DataTable'

import type { Cell, Panel } from './types'

type Props = {
  panels: Panel[]
  isLoading: boolean

  /** Waktu cuplikan sumbernya terakhir disegarkan; kosong berarti tidak berlaku. */
  refreshedAt?: string | undefined
}

/**
 * Grid-grid sebuah tab dashboard.
 *
 * # Dua panel, dan angka dua itu dibaca dari section
 *
 * `Section/PNCDashboardOS` mengikat TEPAT DUA page list, meski activity pemasoknya
 * `PNCGetDashboardOSInbox_Act` menjalankan LIMA kueri. Tiga sisanya mengisi penyaring dan
 * daftar pilihan, bukan grid. `DisplayInboxProduktivitas_Sect` dan `DashboardKlaim_sec` pun
 * mengikat dua.
 *
 * Membaca activity saja akan menggambar tiga tabel yang tidak pernah ada di layar lama.
 *
 * # Barisnya TIDAK dapat dipilih, dan itu disengaja
 *
 * Baris dashboard tidak punya kunci, tidak dapat diputuskan, dan tidak hilang setelah
 * ditindaklanjuti. Menggambar kotak pilih di sini akan menyarankan tindakan yang tidak ada.
 */
export function DashboardPanels({ panels, isLoading, refreshedAt }: Props) {
  return (
    <div className="space-y-6">
      {refreshedAt ? <RefreshNote at={refreshedAt} /> : null}

      {panels.map((panel) => (
        <DataTable<Record<string, Cell>>
          key={panel.kunci}
          title={panel.judul}
          label={panel.judul}
          columns={panel.kolom.map(toColumn)}
          rows={panel.baris}
          rowKey={(row) => rowKeyOf(panel, row)}
          isLoading={isLoading}
          hideSearch
          emptyMessage="Tidak ada angka untuk penyaring ini."
        />
      ))}
    </div>
  )
}

/**
 * Keterangan waktu penyegaran.
 *
 * # Kenapa ia ditampilkan, bukan disembunyikan sebagai detail
 *
 * Karena dua dari tiga dashboard membaca TABEL CUPLIKAN (`POOLDATA.PEGA_DASHBOARDPNC`), bukan
 * data klaim langsung. Angkanya berumur sampai penyegaran berikutnya, dan penyelia yang
 * membandingkannya dengan layar lain berhak tahu itu.
 *
 * Layar lama pun menampilkannya — `Section/DisplayInboxProduktivitas_Sect` punya
 * `<pyCaption Last Refresh>`, diisi `Activity/LastRefresh_act` dari kolom `refreshdate`.
 */
function RefreshNote({ at }: { at: string }) {
  const when = new Date(at)
  const readable = Number.isNaN(when.getTime())
    ? at
    : when.toLocaleString('id-ID', { dateStyle: 'long', timeStyle: 'short' })

  return (
    <p className="text-sm text-slate-600">
      Angka di bawah berasal dari cuplikan yang terakhir disegarkan{' '}
      <span className="font-medium text-slate-800">{readable}</span>, bukan dari data klaim
      saat ini.
    </p>
  )
}

/**
 * toColumn mengubah kolom yang ditetapkan server menjadi kolom DataTable.
 *
 * # Nilai uang diformat di sini, dan nilainya tetap TEKS
 *
 * Ia datang sebagai teks presisi penuh dari basis data dan tidak pernah melewati `Number` —
 * kecuali untuk ditampilkan. Mengubahnya menjadi angka JavaScript berarti jumlah rupiah
 * melewati bilangan pecahan biner, yang `I-12` larang.
 */
function toColumn(column: { kunci: string; judul: string; uang?: boolean }): Column<
  Record<string, Cell>
> {
  return {
    key: column.kunci,
    title: column.judul,
    alignRight: column.uang === true,
    value: (row) => cellText(row[column.kunci], column.uang === true),
  }
}

/** cellText menyusun teks sebuah sel sesuai jenis kolomnya. */
function cellText(cell: Cell | undefined, money: boolean): string {
  if (!cell) return ''

  if (money) return formatAmount(cell.nilai ?? '')
  if (cell.teks !== undefined && cell.teks !== '') return cell.teks
  if (cell.jumlah !== undefined) return String(cell.jumlah)

  return ''
}

/**
 * formatAmount menyisipkan pemisah ribuan tanpa mengubah nilainya.
 *
 * Ia bekerja pada TEKS, bukan pada angka: pembulatan yang terjadi saat teks presisi penuh
 * diubah menjadi `number` tidak dapat dibatalkan, dan pada jumlah rupiah yang besar ia
 * mengubah digit terakhir.
 */
function formatAmount(value: string): string {
  const clean = value.trim()
  if (clean === '') return ''

  const negative = clean.startsWith('-')
  const unsigned = negative ? clean.slice(1) : clean

  const [whole, fraction] = unsigned.split('.')
  if (whole === undefined || !/^\d+$/.test(whole)) return clean

  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, '.')
  const tail = fraction ? `,${fraction}` : ''

  return `${negative ? '-' : ''}${grouped}${tail}`
}

/**
 * rowKeyOf menyusun kunci baris untuk React.
 *
 * Baris dashboard TIDAK punya kunci dari server — ia memang tidak butuh satu, karena tidak
 * dapat diputuskan. Yang dipakai di sini gabungan seluruh sel dimensinya, sehingga dua baris
 * yang benar-benar berbeda tidak pernah berbagi kunci.
 */
function rowKeyOf(panel: Panel, row: Record<string, Cell>): string {
  return panel.kolom
    .map((column) => {
      const cell = row[column.kunci]
      return cell?.teks ?? ''
    })
    .filter((text) => text !== '')
    .join('|')
}
