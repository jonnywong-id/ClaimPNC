import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { Field } from '@/components/Field'
import { SearchIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'
import { formatDate } from '@/components/format'

import type { ClaimCandidate, ClaimSearchForm, Option } from './types'

type Props = {
  types: Option[]
  form: ClaimSearchForm
  onChange: (change: Partial<ClaimSearchForm>) => void
  onSubmit: () => void

  claims: ClaimCandidate[]
  isLoading: boolean
  error: string

  /** Dipanggil saat pengguna menekan Detail pada satu baris klaim. */
  onPick: (claim: ClaimCandidate) => void

  /** Nomor klaim yang sedang terpilih, ditandai di grid. */
  pickedNumber: string

  fieldError: Record<string, string>

  /** True selama pencarian pernah dijalankan; grid disembunyikan sebelum itu. */
  searched: boolean
}

/**
 * ClaimPicker adalah bagian atas "Input Data Archive": cari klaim, lalu pilih satu.
 *
 * # Satu hal yang perlu diketahui pengguna, dan layar menyebutkannya
 *
 * Dropdown "Tipe Input Archive" PRAKTIS TIDAK MEMPERSEMPIT apa pun kecuali pada pilihan
 * "No Polis". `RDB List/SearchArchiveInsert-SQL.xml` mencocokkan nilai yang diketik ke
 * nomor klaim, nomor polis, DAN nama tertanggung sekaligus — apa pun tipe yang dipilih.
 *
 * Itu perilaku sistem lama, bukan kelalaian pembacaan, dan ia direplikasi. Menyebutkannya
 * di layar lebih baik daripada membiarkan pengguna menyimpulkan sendiri bahwa dropdownnya
 * rusak.
 */
export function ClaimPicker({
  types,
  form,
  onChange,
  onSubmit,
  claims,
  isLoading,
  error,
  onPick,
  pickedNumber,
  fieldError,
  searched,
}: Props) {
  const columns: Column<ClaimCandidate>[] = [
    { key: 'nomor_klaim', title: 'No Klaim', value: (row) => row.nomor_klaim },
    { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis },
    { key: 'nama_tertanggung', title: 'Nama Tertanggung', value: (row) => row.nama_tertanggung },
    {
      key: 'tanggal_kejadian',
      title: 'Tgl Kejadian',
      value: (row) => dateText(row.tanggal_kejadian),
    },
    { key: 'bisnis', title: 'Bisnis', value: (row) => row.bisnis },
    { key: 'cabang', title: 'Cabang', value: (row) => row.cabang },
    { key: 'status', title: 'Status', value: (row) => row.status },
    { key: 'posisi_klaim', title: 'Posisi Klaim', value: (row) => row.posisi_klaim || '—' },
    { key: 'tanggal_close', title: 'Tanggal Close', value: (row) => dateText(row.tanggal_close) },
    { key: 'catatan_close', title: 'Catatan Close', value: (row) => row.catatan_close || '—' },
    { key: 'pic_teknis', title: 'PIC Teknis', value: (row) => row.pic_teknis },
    {
      key: 'aksi',
      title: '',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          type="button"
          tone={row.nomor_klaim === pickedNumber ? 'utama' : 'kedua'}
          onClick={() => onPick(row)}
        >
          {row.nomor_klaim === pickedNumber ? 'Terpilih' : 'Detail'}
        </Button>
      ),
    },
  ]

  return (
    <section>
      <form
        className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
        onSubmit={(event) => {
          event.preventDefault()
          onSubmit()
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <SelectField
            id="tipe-input-archive"
            label="Tipe Input Archive"
            value={form.tipe}
            onChange={(event) => onChange({ tipe: event.target.value })}
            error={fieldError['tipe_input']}
            options={types.map((type) => ({ value: type.kode, label: type.label }))}
            emptyText=""
          />

          <Field
            id="keyword-klaim"
            label="Keyword"
            value={form.nilai}
            onChange={(event) => onChange({ nilai: event.target.value })}
            error={fieldError['keyword_klaim']}
            icon={<SearchIcon className="h-4 w-4" />}
            hint={
              form.tipe === 'no_polis'
                ? 'Hanya No Polis yang dicocokkan.'
                : 'Dicocokkan ke No Klaim, No Polis, dan Nama Tertanggung sekaligus — sama seperti sistem lama.'
            }
            placeholder="PNC-100001"
          />
        </div>

        <div className="mt-4 flex justify-end">
          <Button type="submit" tone="utama" disabled={isLoading}>
            {isLoading ? 'Mencari…' : 'Cari Klaim'}
          </Button>
        </div>
      </form>

      {searched && (
        <div className="mt-4">
          <DataTable
            columns={columns}
            rows={claims}
            rowKey={(row) => row.nomor_klaim}
            title="Input Data Archive"
            label="Daftar klaim yang berkasnya dapat diarsipkan"
            isLoading={isLoading}
            error={error}
            emptyMessage="Tidak ada klaim yang cocok. Periksa kembali nilai yang dicari — pencocokannya PERSIS, bukan sebagian."
            hideSearch
          />
        </div>
      )}
    </section>
  )
}

function dateText(iso: string | null): string {
  return iso ? formatDate(iso) : '—'
}
