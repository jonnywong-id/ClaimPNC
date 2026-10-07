import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import { formatTanggal, pesanGalat } from './pesan'
import type { BarisXOL, Kolom } from './types'

type Props = {
  kolom: Kolom[]
  rows: BarisXOL[]
  isLoading: boolean
  isError: boolean
  error: unknown
}

/**
 * XOLPanel adalah grid **"DATA PLA DLA XOL KLAIM"**.
 *
 * Isinya ringkasan pemberitahuan XOL yang SUDAH dikirimkan kepada reasuradur pemanggil,
 * dikelompokkan per tahun dan per penyebab kerugian.
 *
 * # Isinya kini milik Anda sendiri, dan di Pega TIDAK
 *
 * Kueri lamanya memakai nama reasuradur yang ditulis TETAP di dalam activity
 * (`Activity/SetDataPLADLA-Act.xml` menetapkan `Local.loginreas` satu kali dan tidak
 * pernah menimpanya). Akibatnya setiap mitra yang membuka layar itu melihat ringkasan
 * XOL milik satu mitra tertentu — bukan miliknya.
 *
 * Itu kebocoran data antar pihak ketiga, bukan keanehan yang layak ditiru, dan `D-15`
 * melarang nilai bisnis ditulis tetap. Keterangan di kaki panel menyebutkannya supaya
 * mitra yang terbiasa melihat angka lama tahu mengapa angkanya berubah.
 */
export function XOLPanel({ kolom, rows, isLoading, isError, error }: Readonly<Props>) {
  return (
    <section
      className="space-y-2"
      aria-label="Ringkasan pemberitahuan XOL yang sudah terkirim"
    >
      <DataTable<BarisXOL>
        columns={kolomXOL(kolom)}
        rows={rows}
        rowKey={(row) => `${row.jenis}-${row.tahun}-${row.penyebab_kerugian}`}
        title="DATA PLA DLA XOL KLAIM"
        description="Pemberitahuan XOL yang sudah dikirimkan kepada Anda, dikelompokkan per tahun dan penyebab kerugian."
        label="Ringkasan XOL"
        isLoading={isLoading}
        error={
          isError ? (
            <ErrorMessage
              title="Ringkasan XOL tidak dapat dimuat"
              description={pesanGalat(error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage="Belum ada pemberitahuan XOL yang dikirimkan kepada Anda."
      />

      <p className="text-xs text-slate-500">
        Isi tabel ini mengikuti login Anda. Di Pega ia ditulis tetap ke satu mitra,
        sehingga angkanya di sana belum tentu milik Anda.
      </p>
    </section>
  )
}

/** kolomXOL menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable. */
function kolomXOL(kolom: Kolom[]): Column<BarisXOL>[] {
  return kolom.map((item) => ({
    key: item.kunci,
    title: item.judul,
    value: (row) => nilaiSel(row, item.kunci),
    render: (row) => {
      const isi = nilaiSel(row, item.kunci)
      if (item.tanggal) return formatTanggal(isi)
      if (isi === '') return <span className="text-slate-400">—</span>
      return isi
    },
  }))
}

/** nilaiSel mengambil isi satu sel sebagai TEKS. */
function nilaiSel(row: BarisXOL, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}
