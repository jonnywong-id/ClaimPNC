import { Button } from '@/components/Button'
import { Field } from '@/components/Field'

import type { FormPencarian } from './types'

type Props = {
  form: FormPencarian
  onChange: (perubahan: Partial<FormPencarian>) => void
  onSubmit: () => void
  onReset: () => void
  busy: boolean

  /**
   * Jenis tanggal yang sedang disaring — "Tanggal PLA", "Tanggal DLA", atau
   * "Tanggal Pre DLA". Datang dari server.
   *
   * Ia TIDAK digambar sebagai judul kotak: di Pega kotaknya hanya berjudul "Dari" dan
   * "Sampai", dan `D-13` menetapkan tata letaknya ditiru. Yang dilakukan dengannya adalah
   * memasangnya sebagai NAMA bagi pembaca layar dan sebagai keterangan saat disentuh
   * penunjuk — sehingga keterangan yang di Pega harus ditebak tetap tersedia tanpa
   * mengubah apa yang terlihat.
   *
   * Nama bagi pembaca layar MEMUAT judul yang terlihat ("… — Dari"), sebagaimana dituntut
   * WCAG 2.5.3: pengguna perintah suara yang mengucapkan "Dari" tetap mengenai kotak ini.
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
 * # Susunannya mengikuti layar lama baris per baris (`D-13`)
 *
 *	Dari  [tanggal]        Sampai  [tanggal]
 *	No Klaim  [teks]  [CARI DATA]
 *
 * Itu persis susunan ketiga section Pega: dua kotak tanggal berdampingan di baris
 * pertama, lalu kotak nomor klaim dengan tombolnya DI SAMPINGNYA — bukan di baris
 * tersendiri di bawah. Sebelum 2026-10-10 ketiganya digambar sebagai tiga kolom sejajar
 * dengan tombol di bawahnya, dan itu tata letak yang dikarang.
 *
 * # Kenapa formulir, bukan pencarian saat mengetik
 *
 * Karena penyaringnya menyentuh tabel dokumen berisi puluhan juta baris. Satu perjalanan
 * ke basis data per huruf yang diketik bukan beban yang layak ditanggung — dan layar
 * lamanya pun memakai tombol.
 *
 * # Satu tombol DITAMBAHKAN, dan alasannya
 *
 * "Bersihkan" tidak ada di Pega. Ia ditambahkan karena rentang tanggal di sini boleh
 * dikosongkan — di Pega tidak bisa, sebab isian kosong di sana dirangkai menjadi
 * `to_date('','dd/mm/yyyy')` lalu ditolak Oracle. Kemampuan mengosongkan yang tidak punya
 * tombolnya akan memaksa pengguna menghapus isi kedua kotak satu per satu.
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
}: Props) {
  return (
    <form
      className="rounded-kartu border border-slate-200 bg-white px-5 py-4 shadow-lembut"
      onSubmit={(event) => {
        event.preventDefault()
        onSubmit()
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          id="pladla-dari"
          label="Dari"
          aria-label={`${labelTanggal} — Dari`}
          title={`${labelTanggal} — Dari`}
          type="date"
          value={form.dari}
          onChange={(event) => onChange({ dari: event.target.value })}
          error={fieldError['dari']}
        />

        <Field
          id="pladla-sampai"
          label="Sampai"
          aria-label={`${labelTanggal} — Sampai`}
          title={`${labelTanggal} — Sampai`}
          type="date"
          value={form.sampai}
          onChange={(event) => onChange({ sampai: event.target.value })}
          error={fieldError['sampai']}
        />
      </div>

      {/*
        Tombol "CARI DATA" berdampingan dengan kotak No Klaim, seperti di Pega.

        `items-end` meratakan tombolnya dengan DASAR kotak isian, bukan dengan judulnya —
        tanpa itu tombol melayang sejajar tulisan "No Klaim".
      */}
      <div className="mt-4 flex flex-wrap items-end gap-2">
        <div className="min-w-[16rem] flex-1 sm:max-w-md">
          <Field
            id="pladla-cari"
            label={labelPencarian}
            value={form.cari}
            onChange={(event) => onChange({ cari: event.target.value })}
            error={fieldError['cari']}
            placeholder="Sebagian nomor klaim"
          />
        </div>

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
