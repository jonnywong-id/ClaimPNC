import { useState } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { downloadURL, useAdviceSearch, useCauseOfLoss } from './api'
import { isValidationError, messageOf, violationsOf } from './errors'
import { EMPTY_ADVICE_FORM, type Advice, type AdviceForm, type AdviceType } from './types'

/**
 * Panel "Generated / Cari Data DLA PLA XOL".
 *
 * # Kenapa kedua tombol lama menjadi SATU panel
 *
 * Karena keduanya menjalankan hal yang sama. `Generated DLA PLA XOL` dan `Cari Data DLA
 * PLA XOL` sama-sama memanggil `BrowseDataXOLPLADLAGenerated` dengan tiga parameter yang
 * sama — tahun, penyebab kerugian, dan tipe. Yang membedakannya di sistem lama hanyalah
 * dari mana ketiga nilai itu diambil: yang satu dari baris yang sedang disorot, yang lain
 * dari isian pencarian.
 *
 * Menggambar dua tombol yang membuka dua panel berisi tabel yang sama akan mengulang
 * kekeliruan asalnya, bukan meniru perilaku yang berbeda.
 *
 * # Kolomnya
 *
 * Diambil apa adanya dari `Section/InboxClaimXOL-Section.xml`: NO PLA / DLA · Nama
 * Insurance · Nama Layer · Tahun · Kurs (IDR) · Share Percent · Email · Remark.
 */
export function AdvicePanel() {
  const [form, setForm] = useState<AdviceForm>(EMPTY_ADVICE_FORM)
  const [submitted, setSubmitted] = useState<AdviceForm | null>(null)

  const causes = useCauseOfLoss()
  const search = useAdviceSearch(submitted ?? EMPTY_ADVICE_FORM, submitted !== null)

  const violations = violationsOf(search.error)
  const rows = search.data?.pemberitahuan ?? []

  const causeOptions = (causes.data?.sebab_kerugian ?? []).map((cause) => ({
    // Nilainya DESKRIPSI, bukan ID: itulah yang tersimpan di kolom CAUSEOFLOSS pada
    // tabel PLA/DLA. Mengirim ID akan menghasilkan daftar kosong tanpa pesan galat.
    value: cause.deskripsi,
    label: cause.deskripsi,
  }))

  return (
    <div className="mt-4 space-y-4">
      <form
        className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut"
        onSubmit={(event) => {
          event.preventDefault()
          setSubmitted(form)
        }}
      >
        <div className="grid gap-4 sm:grid-cols-3">
          <Field
            id="xol-tahun"
            label="Tahun XOL"
            inputMode="numeric"
            placeholder="2024"
            value={form.tahun}
            error={violations['tahun']}
            onChange={(event) => setForm({ ...form, tahun: event.target.value })}
          />

          <SelectField
            id="xol-sebab"
            label="Penyebab Kerugian"
            options={causeOptions}
            emptyText={causes.isPending ? '— memuat —' : '— pilih penyebab —'}
            value={form.sebab_kerugian}
            error={violations['sebab_kerugian']}
            onChange={(event) => setForm({ ...form, sebab_kerugian: event.target.value })}
          />

          <SelectField
            id="xol-tipe"
            label="Tipe Pemberitahuan"
            options={[
              { value: 'PLA', label: 'PLA — Preliminary Loss Advice' },
              { value: 'DLA', label: 'DLA — Definite Loss Advice' },
            ]}
            emptyText="— pilih tipe —"
            value={form.tipe}
            error={violations['tipe']}
            onChange={(event) =>
              setForm({ ...form, tipe: event.target.value as AdviceType | '' })
            }
          />
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-2">
          <Button type="submit" tone="utama" disabled={search.isFetching}>
            {search.isFetching ? 'Mencari…' : 'Cari Data DLA PLA XOL'}
          </Button>

          {submitted !== null && rows.length > 0 && (
            <DownloadButton form={submitted} />
          )}
        </div>

        {causes.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Daftar penyebab kerugian tidak dapat dimuat"
              description={messageOf(causes.error)}
              tone="gangguan"
            />
          </div>
        )}
      </form>

      {submitted !== null && (
        <DataTable<Advice>
          columns={adviceColumns}
          rows={rows}
          rowKey={(row) => `${row.nomor_asli}|${row.revisi}|${row.nama_layer}`}
          title={`Pemberitahuan ${submitted.tipe} — ${submitted.tahun} · ${submitted.sebab_kerugian}`}
          // Kotak cari bawaan disembunyikan: panel ini sudah punya formulir pencariannya
          // sendiri di atas, dan dua kotak yang mencari hal berbeda tidak dapat
          // dibedakan pengguna.
          hideSearch
          isLoading={search.isPending}
          error={
            search.isError && !isValidationError(search.error) ? (
              <ErrorMessage
                title="Pencarian tidak dapat dijalankan"
                description={messageOf(search.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          emptyMessage="Belum ada pemberitahuan yang diterbitkan untuk tahun dan penyebab kerugian ini."
        />
      )}
    </div>
  )
}

/**
 * DownloadButton menggantikan tombol "Print Perhitungan" — sementara, dan dinyatakan
 * demikian.
 *
 * Tombol lama menyusun dokumen PLA/DLA lewat engine cetak Pega. `D-11` menetapkan
 * dokumen dibuat sendiri di Go, dan Work Owner memutuskan 2026-09-20 bahwa tahap ini
 * cukup mengeluarkan berkas tabel lebih dulu.
 *
 * Labelnya karena itu TIDAK berbunyi "Print Perhitungan". Berkas yang tampak seperti
 * dokumen resmi padahal bukan adalah kekeliruan yang berakibat ke luar perusahaan —
 * PLA dan DLA dikirim kepada reasuradur.
 *
 * Ia tautan, bukan tombol ber-onClick: unduhan adalah navigasi, dan peramban sudah
 * menanganinya termasuk saat pengguna membukanya di tab baru.
 */
function DownloadButton({ form }: { form: AdviceForm }) {
  return (
    <a
      href={downloadURL(form)}
      className="inline-flex items-center rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 shadow-lembut transition hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-4"
    >
      Unduh perhitungan (CSV)
    </a>
  )
}

const adviceColumns: Column<Advice>[] = [
  {
    key: 'nomor',
    title: 'NO PLA / DLA',
    value: (row) => row.nomor,
    width: '12rem',
  },
  {
    key: 'nama_insurance',
    title: 'Nama Insurance',
    value: (row) => row.nama_insurance,
  },
  {
    key: 'nama_layer',
    title: 'Nama Layer',
    value: (row) => row.nama_layer,
    width: '8rem',
  },
  {
    key: 'tahun',
    title: 'Tahun',
    value: (row) => row.tahun,
    width: '6rem',
  },
  {
    key: 'kurs',
    title: 'Kurs (IDR)',
    value: (row) => String(row.kurs),
    render: (row) => new Intl.NumberFormat('id-ID').format(row.kurs),
    alignRight: true,
    width: '8rem',
  },
  {
    key: 'share_percent',
    title: 'Share Percent',
    value: (row) => String(row.share_percent),
    // Tanda persen ditambahkan DI SINI, bukan di SQL. Nilai bersatuan tidak dapat
    // dijumlahkan maupun diurutkan — kueri lama merangkainya sebagai `PERCENT || ' %'`.
    render: (row) =>
      `${new Intl.NumberFormat('id-ID', { maximumFractionDigits: 4 }).format(row.share_percent)} %`,
    alignRight: true,
    width: '8rem',
  },
  {
    key: 'email',
    title: 'Email',
    value: (row) => row.email,
  },
  {
    key: 'remark',
    title: 'Remark',
    value: (row) => row.remark,
  },
  {
    key: 'status',
    title: 'Status',
    noSort: true,
    width: '9rem',
    value: (row) => (row.sudah_disetujui ? 'Disetujui' : 'Menunggu'),
    render: (row) =>
      row.sudah_disetujui ? (
        <span className="inline-flex items-center rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 ring-1 ring-emerald-200">
          Disetujui
        </span>
      ) : (
        <span className="inline-flex items-center rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600 ring-1 ring-slate-200">
          Menunggu komite
        </span>
      ),
  },
]
