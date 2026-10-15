import { Button } from '@/components/Button'
import { Field } from '@/components/Field'
import { SearchIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'

import type { SearchForm, SearchType } from './types'

type Props = {
  types: SearchType[]
  form: SearchForm
  onChange: (change: Partial<SearchForm>) => void
  onSubmit: () => void
  busy: boolean

  /** Pesan galat per isian, dikirim server sebagai `detail` pada galat validasi. */
  fieldError: Record<string, string>
}

/**
 * Panel pencarian — tiga isian dan satu tombol, persis formulir layar lama.
 *
 * # Kenapa isiannya tiga, bukan satu
 *
 * Karena begitulah bentuk formulir `Section/PNCSearchKlaim-Section.xml`, dan bentuk itu
 * yang melahirkan salah satu cacat yang Work Owner putuskan untuk direplikasi:
 *
 *	"Nama Pencarian"    teks     tipe 1,2,3,4,5,7,8,11,12,13
 *	"Tanggal Pencarian" tanggal  tipe 6 dan 12
 *	"Tanggal Lahir"     tanggal  tipe 9
 *
 * Menyederhanakannya menjadi satu isian yang berganti tipe akan menghapus perbedaan
 * antara "Tanggal Pencarian" dan "Tanggal Lahir" — dan bersamanya cacat yang membuat
 * pencarian Tanggal Lahir tidak pernah membuahkan hasil.
 *
 * Isian mana yang tampak DITENTUKAN SERVER lewat ketiga penanda pada tipe terpilih, bukan
 * ditebak layar. Dengan begitu bentuk formulir punya satu sumber kebenaran.
 */
export function SearchPanel({ types, form, onChange, onSubmit, busy, fieldError }: Props) {
  const selected = types.find((t) => t.kode === form.tipe)

  return (
    <form
      className="mt-4 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      onSubmit={(e) => {
        e.preventDefault()
        onSubmit()
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <SelectField
          id="tipe-pencarian"
          label="Tipe Pencarian"
          value={form.tipe}
          onChange={(e) => onChange({ tipe: e.target.value })}
          error={fieldError['tipe_pencarian']}
          emptyText="----- PILIH -----"
          // Pilihan yang belum tersedia TETAP tampil, ditandai pada labelnya.
          // Menghapusnya akan membuat pengguna melaporkan pilihan yang "hilang";
          // membiarkannya membuat kemajuan migrasi terbaca langsung dari layar.
          //
          // Ia tidak dinonaktifkan pada tingkat pilihan — `SelectOption` milik pustaka
          // komponen baku tidak mengenal penanda itu, dan menambahkannya berarti
          // mengubah komponen yang dipakai seluruh modul demi satu layar. Yang
          // dinonaktifkan adalah tombol Cari, ditambah keterangan alasannya di bawah.
          options={types.map((t) => ({
            value: t.kode,
            label: t.tersedia ? t.label : `${t.label} — belum tersedia`,
          }))}
        />

        {selected?.pakai_teks && (
          <Field
            id="nilai-pencarian"
            label="Nama Pencarian"
            value={form.nilai}
            onChange={(e) => onChange({ nilai: e.target.value })}
            error={fieldError['nilai_pencarian']}
            placeholder={placeholderFor(selected.kode)}
            autoComplete="off"
            icon={<SearchIcon className="h-4 w-4" />}
            hint={hintFor(selected.kode)}
          />
        )}

        {selected?.pakai_tanggal_pencarian && (
          <Field
            id="tanggal-pencarian"
            label="Tanggal Pencarian"
            type="date"
            value={form.tanggal_pencarian}
            onChange={(e) => onChange({ tanggal_pencarian: e.target.value })}
            error={fieldError['tanggal_pencarian']}
          />
        )}

        {selected?.pakai_tanggal_lahir && (
          <Field
            id="tanggal-lahir"
            label="Tanggal Lahir"
            type="date"
            value={form.tanggal_lahir}
            onChange={(e) => onChange({ tanggal_lahir: e.target.value })}
            error={fieldError['tanggal_lahir']}
          />
        )}

        <div className="flex items-end">
          <Button type="submit" tone="utama" disabled={busy || !selected?.tersedia}>
            <SearchIcon className="mr-1.5 h-4 w-4" />
            {busy ? 'Mencari…' : 'Cari'}
          </Button>
        </div>
      </div>

      {selected && !selected.tersedia && selected.alasan_belum_tersedia && (
        <p className="mt-4 rounded-kartu border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900">
          {selected.alasan_belum_tersedia}
        </p>
      )}
    </form>
  )
}

/**
 * placeholderFor memberi contoh bentuk isian yang diharapkan.
 *
 * Contohnya karangan, dan sengaja berpola jelas: nomor polis dan nomor klaim punya bentuk
 * yang berbeda, dan pengguna yang salah menempelkan salah satunya akan menyadarinya dari
 * bentuk contoh — bukan dari hasil pencarian yang kosong.
 */
function placeholderFor(code: string): string {
  switch (code) {
    case '1':
      return 'POL-0000-0000'
    case '2':
      return 'nama tertanggung'
    case '3':
      return 'nama objek pertanggungan'
    case '4':
      return 'PLA-0000'
    case '5':
      return 'DLA-0000'
    case '7':
      return 'PNC-0000 atau PNCN.26.0000'
    case '8':
      return 'AKS-0000'
    case '11':
      return 'SRV-0000'
    case '13':
      return 'BL-00'
    default:
      return ''
  }
}

/**
 * hintFor menyatakan cara isian dicocokkan.
 *
 * Perbedaan antara "dicocokkan sebagian" dan "harus persis" menentukan apakah hasil
 * kosong berarti datanya tidak ada atau kata kuncinya kurang lengkap — dan itu satu-satunya
 * hal yang tidak dapat disimpulkan pengguna dari layar yang kosong.
 *
 * Keduanya diturunkan dari kueri sistem lama: yang memakai `LIKE '%…%'` dicocokkan
 * sebagian, sisanya dengan `=`.
 */
function hintFor(code: string): string {
  const partial = ['2', '3', '4', '5']
  return partial.includes(code)
    ? 'Dicocokkan sebagian, tidak membedakan huruf besar-kecil.'
    : 'Dicocokkan persis, tidak membedakan huruf besar-kecil.'
}
