import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'

import type { CoverageRow } from './types'

/**
 * Panel **"Detail Perubahan Cause Of Loss"** — pemilih baris coverage.
 *
 * Migrasi dari panel bersyarat pada `Section/AcceptProtectionSection-Section.xml`, yang
 * muncul ketika `.TypeProtection == 8`. Bentuknya disalin dari layar Pega yang diperlihatkan
 * Work Owner (`OPC-221`, klaim `PNC-1452`): tabel berkolom **Object Name · Coverage Name ·
 * Cause of Loss**, dengan tombol **Pilih** pada setiap baris.
 *
 * # Kenapa pemilih, bukan satu nilai
 *
 * Satu klaim dapat punya banyak objek, dan tiap objek banyak coverage — masing-masing dengan
 * Penyebab Kerugiannya sendiri. Pada klaim `PNC-1452`, kuerinya mengembalikan:
 *
 *     JackHugh  Resiko A    ILLNESS
 *     JackHugh  Resiko A    STORM
 *     JackHugh  Katastropi  WINDSTORM
 *     JackHugh  Resiko A    HURRICANE
 *
 * `JackHugh / Resiko A` muncul TIGA KALI. Jadi nama tidak menunjuk baris, dan tanpa panel ini
 * permintaan perubahan tidak punya cara menyatakan baris mana yang dimaksud.
 *
 * Versi sebelumnya menampilkan Penyebab Kerugian coverage PERTAMA sebagai satu field
 * hanya-baca — yang pada klaim seperti ini menampilkan baris yang belum tentu hendak diubah.
 */
export function CoveragePicker({
  rows,
  selected,
  onSelect,
  memuat,
  gagal,
  failure,
  pilihanHilang,
}: Readonly<{
  rows: CoverageRow[]
  selected: { objek: string; coverage: string }
  onSelect: (row: CoverageRow) => void
  memuat: boolean
  gagal: boolean
  failure: string | null
  pilihanHilang: boolean
}>) {
  function terpilih(row: CoverageRow): boolean {
    return row.id_objek === selected.objek && row.id_coverage === selected.coverage
  }

  const columns: Column<CoverageRow>[] = [
    {
      key: 'nama_objek',
      title: 'Object Name',
      value: (row) => row.nama_objek,
    },
    {
      key: 'nama_coverage',
      title: 'Coverage Name',
      value: (row) => row.nama_coverage,
    },
    {
      key: 'penyebab_kerugian',
      title: 'Cause of Loss',
      // Kode ditampilkan berdampingan dengan deskripsinya.
      //
      // Dua baris dapat bernama sama persis sementara kodenya berbeda, dan akseptasi
      // mengubah KEDUANYA. Menyembunyikan kode membuat pemohon memilih di antara dua baris
      // yang terlihat identik.
      value: (row) =>
        row.penyebab_kerugian_id
          ? `${row.penyebab_kerugian} (${row.penyebab_kerugian_id})`
          : row.penyebab_kerugian,
    },
    {
      key: 'pilih',
      title: '',
      value: (row) => (terpilih(row) ? 'Terpilih' : 'Pilih'),
      noSort: true,
      alignRight: true,
      render: (row) =>
        terpilih(row) ? (
          <span className="text-sm font-medium text-emerald-700">Terpilih</span>
        ) : (
          <Button tone="halus" type="button" onClick={() => { onSelect(row) }}>
            Pilih
          </Button>
        ),
    },
  ]

  return (
    <div className="space-y-2">
      <DataTable
        columns={columns}
        rows={rows}
        // Kunci baris adalah pasangan objek + coverage, BUKAN nama.
        //
        // Nama berulang pada klaim nyata, dan kunci yang berulang membuat React menukar baris
        // saat daftarnya berubah — pengguna menekan Pilih pada satu baris dan yang tersorot
        // baris lain.
        rowKey={(row) => `${row.id_objek}-${row.id_coverage}`}
        isLoading={memuat}
        hideSearch
        dense
        emptyMessage={
          gagal
            ? 'Daftar coverage gagal dimuat.'
            : 'Klaim ini belum punya coverage, sehingga perubahan Cause Of Loss belum dapat diajukan.'
        }
      />

      {pilihanHilang && (
        <p className="text-sm text-amber-700">
          Baris yang dipilih sebelumnya sudah tidak ada pada klaim ini. Pilih kembali salah
          satu baris di atas.
        </p>
      )}

      {failure && <p className="text-sm text-rose-700">{failure}</p>}
    </div>
  )
}
