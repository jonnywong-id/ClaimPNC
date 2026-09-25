import { useState } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { formatDate } from '@/components/format'

import { FillingCodePicker } from './FillingCodePicker'
import type { ArchiveForm, DocumentKind, Option } from './types'

type Props = {
  form: ArchiveForm
  onChange: (change: Partial<ArchiveForm>) => void
  onSubmit: () => void
  onCancel: () => void

  documentTypes: Option[]
  documentKinds: DocumentKind[]

  busy: boolean
  fieldError: Record<string, string>

  /** Pesan berhasil dari server; kosong bila belum ada penyimpanan. */
  successMessage: string
}

/**
 * ArchiveFormPanel adalah formulir berkas arsip — bagian bawah "Input Data Archive".
 *
 * # Dua kelompok isian, dan pembedaannya terlihat
 *
 * Enam isian pertama IKUT dari klaim yang dipilih dan tidak dapat diketik; enam
 * berikutnya diketik pengguna. Yang pertama digambar sebagai keterangan baca-saja, bukan
 * sebagai isian yang dikunci — isian abu-abu yang tidak dapat diklik mengundang pengguna
 * mencoba mengubahnya berulang kali.
 *
 * Pembagiannya mengikuti `Activity/SetDataArchiveDokumentCase-Act.xml`, yang menyalin
 * tepat enam properti dari baris klaim terpilih ke formulir.
 *
 * # Jenis Dokumen bergantung pada Tipe Dokumen
 *
 * Kueri lama menggabungkan keduanya dengan INNER JOIN pada `DOC_TYPE_ID`, sehingga jenis
 * dokumen SELALU milik satu tipe. Penyaringan di sini meniru itu: mengganti tipe
 * mengosongkan jenis, karena jenis yang tertinggal dari tipe sebelumnya akan tersimpan
 * sebagai pasangan yang tidak ada di master — dan grid kemudian menampilkannya kosong
 * tanpa ada yang tahu sebabnya.
 */
export function ArchiveFormPanel({
  form,
  onChange,
  onSubmit,
  onCancel,
  documentTypes,
  documentKinds,
  busy,
  fieldError,
  successMessage,
}: Props) {
  const [pickerOpen, setPickerOpen] = useState(false)

  const kindsForType = documentKinds.filter(
    (kind) => kind.kode_tipe_dokumen === form.kode_tipe_dokumen,
  )

  const editing = form.id !== 0

  return (
    <section className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            {editing ? 'Ubah Berkas Archive' : 'Berkas Archive'}
          </h2>
          <p className="mt-1 text-sm text-slate-600">
            Klaim terpilih: <span className="font-medium">{form.nomor_klaim}</span>
          </p>
        </div>

        {editing && (
          <span className="rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-800">
            Mengubah berkas nomor {form.id}
          </span>
        )}
      </header>

      {/* Keterangan yang ikut dari klaim — baca-saja, bukan isian terkunci. */}
      <dl className="mt-4 grid gap-x-6 gap-y-3 rounded-kontrol bg-slate-50 p-4 text-sm sm:grid-cols-2 lg:grid-cols-3">
        <ReadOnly label="No Klaim" value={form.nomor_klaim} />
        <ReadOnly label="No Polis" value={form.nomor_polis} />
        <ReadOnly label="Nama Tertanggung" value={form.nama_tertanggung} />
        <ReadOnly
          label="Tgl Kejadian"
          value={form.tanggal_kejadian ? formatDate(form.tanggal_kejadian) : '—'}
        />
        <ReadOnly label="PIC Teknis" value={form.pic_teknis} />
        <ReadOnly label="Group Panel" value={form.group_panel || '—'} />
      </dl>

      <form
        className="mt-5"
        onSubmit={(event) => {
          event.preventDefault()
          onSubmit()
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Field
            id="tanggal-terima-dokumen"
            label="Tgl Terima Dokumen"
            type="date"
            value={form.tanggal_terima_dokumen}
            onChange={(event) => onChange({ tanggal_terima_dokumen: event.target.value })}
            error={fieldError['tanggal_terima_dokumen']}
          />

          <Field
            id="jumlah-lembar"
            label="Jumlah Lembar"
            type="number"
            min={1}
            inputMode="numeric"
            value={form.jumlah_lembar}
            onChange={(event) => onChange({ jumlah_lembar: event.target.value })}
            error={fieldError['jumlah_lembar']}
          />

          <SelectField
            id="kode-tipe-dokumen"
            label="Tipe Dokumen"
            value={form.kode_tipe_dokumen}
            onChange={(event) =>
              // Mengganti tipe MENGOSONGKAN jenis. Lihat penjelasan di kepala berkas.
              onChange({
                kode_tipe_dokumen: event.target.value,
                kode_jenis_dokumen: '',
              })
            }
            error={fieldError['tipe_dokumen']}
            options={documentTypes.map((type) => ({ value: type.kode, label: type.label }))}
          />

          <SelectField
            id="kode-jenis-dokumen"
            label="Jenis Dokumen"
            value={form.kode_jenis_dokumen}
            onChange={(event) => onChange({ kode_jenis_dokumen: event.target.value })}
            error={fieldError['jenis_dokumen']}
            options={kindsForType.map((kind) => ({ value: kind.kode, label: kind.label }))}
            disabled={form.kode_tipe_dokumen === ''}
            emptyText={
              form.kode_tipe_dokumen === '' ? '— pilih Tipe Dokumen dulu —' : '— pilih —'
            }
          />

          <Field
            id="nama-box"
            label="Nama BOX"
            value={form.nama_box}
            onChange={(event) => onChange({ nama_box: event.target.value })}
            error={fieldError['nama_box']}
            placeholder="BOX-A-01"
          />

          <div>
            <Field
              id="kode-filling"
              label="Kode Filling"
              value={form.kode_filling}
              onChange={(event) => onChange({ kode_filling: event.target.value })}
              error={fieldError['kode_filling']}
              placeholder="FIL-2024-001"
            />
            <div className="mt-2">
              <Button type="button" tone="kedua" onClick={() => setPickerOpen(true)}>
                Pilih Kode
              </Button>
            </div>
          </div>
        </div>

        {successMessage && (
          <p
            className="mt-4 rounded-kontrol border border-emerald-200 bg-emerald-50 px-4 py-2.5 text-sm text-emerald-900"
            role="status"
          >
            {successMessage}
          </p>
        )}

        {fieldError['__umum'] && (
          <div className="mt-4">
            <ErrorMessage
              title="Berkas tidak tersimpan"
              description={fieldError['__umum']}
              tone="penolakan"
            />
          </div>
        )}

        <div className="mt-5 flex flex-wrap justify-end gap-3">
          <Button type="button" tone="halus" onClick={onCancel} disabled={busy}>
            Batal
          </Button>
          <Button type="submit" tone="utama" disabled={busy}>
            {busy ? 'Menyimpan…' : 'Save To Archive'}
          </Button>
        </div>
      </form>

      <FillingCodePicker
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        onPick={(code) => {
          // Nama boks ikut terisi. Keduanya memang dipakai berpasangan — nomor dokumen
          // yang dikirim ke sistem Arsip merangkai ID, nama boks, dan kode filling.
          onChange({ kode_filling: code.kode, nama_box: code.nama_box || form.nama_box })
          setPickerOpen(false)
        }}
      />
    </section>
  )
}

/** ReadOnly menggambar satu keterangan yang ikut dari klaim. */
function ReadOnly({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-0.5 font-medium text-slate-900">{value || '—'}</dd>
    </div>
  )
}
