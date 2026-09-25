import { Button } from '@/components/Button'
import { Field } from '@/components/Field'
import { SearchIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'

import { SearchMode, type SearchForm, type SearchModeValue } from './types'

type Props = {
  form: SearchForm
  onChange: (change: Partial<SearchForm>) => void
  onSubmit: () => void
  busy: boolean

  /** Pesan galat per isian, dikirim server sebagai `detail` pada galat validasi. */
  fieldError: Record<string, string>
}

/**
 * Kedua mode pencarian, persis bentuk layar lama.
 *
 * `Activity/SearchDataArchiveFilling-Act.xml` menyusun dua klausa WHERE yang SALING
 * MENGGANTIKAN — yang kedua menimpa yang pertama, bukan menambahinya. Karena itu modenya
 * dropdown, bukan empat isian yang boleh diisi bersamaan: bentuk kedua akan menjanjikan
 * penyaringan gabungan yang tidak pernah terjadi.
 */
const MODES = [
  { value: SearchMode.keyword, label: 'Keyword' },
  { value: SearchMode.inputDate, label: 'Tgl Input' },
] as const

/**
 * ArchiveSearchPanel adalah panel pencarian grid ARCHIVE FILE KLAIM.
 *
 * Isiannya mengikuti `Harness/PNCArchiveDokumen-Harness.xml`: "Tipe Pencarian Archive"
 * dengan satu isian Keyword, atau rentang "Tgl Input Dari"–"Tgl Input Sampai".
 *
 * Nama isiannya TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega supaya
 * pengguna tidak perlu belajar ulang, dan itulah teks yang selama ini mereka baca.
 */
export function ArchiveSearchPanel({ form, onChange, onSubmit, busy, fieldError }: Props) {
  const byKeyword = form.mode === SearchMode.keyword

  return (
    <form
      className="mt-4 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      onSubmit={(event) => {
        event.preventDefault()
        onSubmit()
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <SelectField
          id="mode-pencarian-arsip"
          label="Tipe Pencarian Archive"
          value={form.mode}
          onChange={(event) =>
            onChange({ mode: event.target.value as SearchModeValue })
          }
          error={fieldError['tipe_pencarian']}
          options={MODES.map((mode) => ({ value: mode.value, label: mode.label }))}
          // Tanpa pilihan kosong: kedua mode sama-sama sah, dan salah satunya harus
          // aktif. Pilihan kosong di sini hanya menciptakan keadaan ketiga yang tidak
          // berarti apa-apa.
          emptyText=""
        />

        {byKeyword ? (
          <Field
            id="kata-kunci-arsip"
            label="Keyword"
            value={form.kata_kunci}
            onChange={(event) => onChange({ kata_kunci: event.target.value })}
            error={fieldError['kata_kunci']}
            hint="Cocok PERSIS dengan No Klaim, Nama BOX, atau Nama Tertanggung."
            icon={<SearchIcon className="h-4 w-4" />}
            placeholder="PNC-100001"
          />
        ) : (
          <>
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
          </>
        )}
      </div>

      <div className="mt-4 flex justify-end">
        <Button type="submit" tone="utama" disabled={busy}>
          {busy ? 'Mencari…' : 'Cari'}
        </Button>
      </div>
    </form>
  )
}
