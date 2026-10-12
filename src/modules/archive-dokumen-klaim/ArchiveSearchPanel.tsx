import { Field } from '@/components/Field'
import { SearchIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'

import type { Option, SearchForm } from './types'

type Props = {
  /** Isi dropdown "Tipe Pencarian Archive", dikirim server. */
  columns: Option[]

  form: SearchForm
  onChange: (change: Partial<SearchForm>) => void
  onSubmit: () => void

  /** Pesan galat per isian, dikirim server sebagai `detail` pada galat validasi. */
  fieldError: Record<string, string>
}

/**
 * ArchiveSearchPanel adalah kedua penyaring grid ARCHIVE FILE KLAIM.
 *
 * # Keduanya tampil BERSAMAAN, dan itu koreksi atas rancangan pertama
 *
 * Rancangan pertama menjadikannya satu dropdown mode — Keyword *atau* Tgl Input — yang
 * saling menggantikan. Bentuk itu tidak pernah ada di Pega.
 *
 * `Section/SecArchiveDokumen-Section.xml` memberi KEDUA blok ini `pyContainerVisibleWhen`
 * yang sama, `FalgArchiveData.FlagASO==2`:
 *
 *	posisi 164179   "Tipe Pencarian Archive" + "Keyword"
 *	posisi 707930   "Tgl Input Dari" + "Tgl Input Sampai"
 *
 * Keduanya karena itu digambar berdampingan, masing-masing boleh diisi atau dikosongkan.
 *
 * # Tombol Cari tidak ada di sini
 *
 * Ia berada di toolbar bersama Tambah, Dokumen Cabang, dan Refresh — urutan yang
 * disebutkan Work Owner dari layar Pega (2026-10-03). Panel ini hanya isiannya, dan
 * menekan Enter pada salah satu isian menjalankan pencarian yang sama.
 */
export function ArchiveSearchPanel({ columns, form, onChange, onSubmit, fieldError }: Props) {
  return (
    <form
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      onSubmit={(event) => {
        event.preventDefault()
        onSubmit()
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <SelectField
          id="tipe-pencarian-archive"
          label="Tipe Pencarian Archive"
          value={form.tipe_pencarian}
          onChange={(event) => onChange({ tipe_pencarian: event.target.value })}
          error={fieldError['tipe_pencarian']}
          options={columns.map((column) => ({ value: column.kode, label: column.label }))}
          // Tanpa pilihan kosong: salah satu kolom harus aktif, dan pilihan kosong hanya
          // menciptakan keadaan keempat yang tidak berarti apa-apa.
          emptyText=""
        />

        <Field
          id="kata-kunci-arsip"
          label="Keyword"
          value={form.kata_kunci}
          onChange={(event) => onChange({ kata_kunci: event.target.value })}
          error={fieldError['kata_kunci']}
          hint="Cocok PERSIS, bukan sebagian."
          icon={<SearchIcon className="h-4 w-4" />}
          placeholder="PNC-100001"
        />

        <Field
          id="tanggal-dari-arsip"
          label="Tgl Input Dari"
          type="date"
          value={form.tanggal_dari}
          onChange={(event) => onChange({ tanggal_dari: event.target.value })}
          error={fieldError['tanggal_dari']}
        />

        <Field
          id="tanggal-sampai-arsip"
          label="Tgl Input Sampai"
          type="date"
          value={form.tanggal_sampai}
          onChange={(event) => onChange({ tanggal_sampai: event.target.value })}
          error={fieldError['tanggal_sampai']}
          hint="Rentangnya termasuk tanggal ini."
        />
      </div>

      {/*
        Tombol pengirim tersembunyi. Ia ada supaya menekan Enter di dalam isian
        menjalankan pencarian — perilaku yang diharapkan setiap formulir.

        Ia DIKELUARKAN dari pohon aksesibilitas dan dari urutan fokus: tombol "Cari" yang
        sesungguhnya ada di toolbar, dan dua tombol bernama sama membuat pembaca layar
        mengumumkan dua pilihan untuk satu tindakan.
      */}
      <button type="submit" aria-hidden="true" tabIndex={-1} className="sr-only">
        Cari
      </button>
    </form>
  )
}
