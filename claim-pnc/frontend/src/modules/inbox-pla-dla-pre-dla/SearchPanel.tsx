import { Button } from '@/components/Button'
import { Field } from '@/components/Field'
import { SearchIcon } from '@/components/Icon'

import type { FormPencarian } from './types'

type Props = {
  form: FormPencarian
  onChange: (perubahan: Partial<FormPencarian>) => void
  onSubmit: () => void
  onReset: () => void
  busy: boolean

  /**
   * Judul rentang tanggal, datang dari server.
   *
   * BERBEDA per daftar — tanggal PLA, tanggal DLA, atau tanggal Pre-DLA. Di Pega kotaknya
   * hanya berjudul "Dari" dan "Sampai", dan pengguna harus menebak tanggal apa yang
   * sedang ia batasi.
   */
  labelTanggal: string

  /** Judul kotak pencarian, datang dari server. */
  labelPencarian: string

  /** Pesan galat per isian, dikirim server sebagai `detail` pada galat validasi. */
  fieldError: Record<string, string>
}

/**
 * SearchPanel adalah panel "Dari / Sampai / No Klaim" beserta tombol "CARI DATA".
 *
 * Susunannya mengikuti ketiga section layar lama apa adanya (`D-13`): dua kotak tanggal,
 * satu kotak nomor klaim, satu tombol cari.
 *
 * # Satu tombol DITAMBAHKAN, dan alasannya
 *
 * "Bersihkan" tidak ada di Pega. Ia ditambahkan karena rentang tanggal di sini boleh
 * dikosongkan — di Pega tidak bisa, sebab isian kosong di sana dirangkai menjadi
 * `to_date('','dd/mm/yyyy')` lalu ditolak Oracle. Kemampuan mengosongkan yang tidak punya
 * tombolnya akan memaksa pengguna menghapus isi kedua kotak satu per satu.
 *
 * # Kenapa formulir, bukan pencarian saat mengetik
 *
 * Karena penyaringnya menyentuh tabel dokumen berisi puluhan juta baris. Satu perjalanan
 * ke basis data per huruf yang diketik bukan beban yang layak ditanggung — dan layar
 * lamanya pun memakai tombol.
 */
export function SearchPanel({
  form,
  onChange,
  onSubmit,
  onReset,
  busy,
  labelTanggal,
  labelPencarian,
  fieldError,
}: Readonly<Props>) {
  return (
    <form
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      onSubmit={(event) => {
        event.preventDefault()
        onSubmit()
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Field
          id="pladla-dari"
          label={`${labelTanggal} — Dari`}
          type="date"
          value={form.dari}
          onChange={(event) => onChange({ dari: event.target.value })}
          error={fieldError['dari']}
          hint="Boleh dikosongkan."
        />

        <Field
          id="pladla-sampai"
          label={`${labelTanggal} — Sampai`}
          type="date"
          value={form.sampai}
          onChange={(event) => onChange({ sampai: event.target.value })}
          error={fieldError['sampai']}
          hint="Tanggal ini ikut terhitung."
        />

        <Field
          id="pladla-cari"
          label={labelPencarian}
          value={form.cari}
          onChange={(event) => onChange({ cari: event.target.value })}
          error={fieldError['cari']}
          icon={<SearchIcon className="h-4 w-4" />}
          placeholder="Sebagian nomor klaim"
        />
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Button type="submit" disabled={busy}>
          {busy ? 'Mencari…' : 'CARI DATA'}
        </Button>
        <Button type="button" tone="kedua" onClick={onReset} disabled={busy}>
          Bersihkan
        </Button>
      </div>
    </form>
  )
}
