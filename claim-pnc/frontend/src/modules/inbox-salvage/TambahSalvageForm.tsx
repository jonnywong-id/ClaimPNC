import { useRef, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'

import { useCreateSalvage, useUploadSalvageDetail } from './api'
import type { CreateRequest, DetailItem, StatusOption } from './types'

type Props = {
  statusOptions: StatusOption[]
  uploadColumns: string[]
  onClose: () => void
  onSaved: (message: string) => void
}

/** Isian form dalam bentuk yang dipegang komponen ini. */
type FormState = Omit<CreateRequest, 'mode' | 'id_salvage' | 'detail_item_salvage'>

const emptyForm: FormState = {
  nomor_klaim: '',
  id_object: '',
  nama_object: '',
  id_coverage: '',
  nama_coverage: '',
  tanggal_input: today(),
  jenis_salvage: '',
  status_salvage: '',
  lokasi_salvage: '',
  lokasi_salvage_di_jabodetabek: false,
  mata_uang: '',
  minimum_salvage: '',
  quantity_salvage: '',
  nilai_penawaran: '',
  share_tertanggung: '',
  remark: '',
  email: '',
  nama_pic_survey: '',
  no_telp_pic_survey: '',
  email_pic_survey: '',
}

/**
 * Form **"Menambahkan Data Salvage"** — di balik tombol Tambah.
 *
 * # Apa yang digantikan
 *
 * `Section/TambahData_Salvage-Section.xml`, yang dibuka setelah
 * `Data Transform/CNMShowInsertSalvage_dt-DT.xml` membersihkan halaman formnya.
 *
 * Data transform itu ternyata TIDAK menyimpan apa pun — kedelapan langkahnya hanya
 * membuang halaman klipboard lama dan menandai mode `"Insert"`. Ia pembersih form, bukan
 * penyimpan. Padanannya di sini adalah keadaan awal `emptyForm`.
 *
 * # Tombol "Upload File" TIDAK menyimpan apa pun
 *
 * Itu yang paling mudah disalahpahami, dan penelusuran ke
 * `Activity/UploadDetailSalvage-Act.xml` membuktikannya: ketiga langkahnya hanya menyalin
 * isi berkas CSV ke grid DI DALAM form. Penyimpanan baru terjadi saat Submit ditekan.
 *
 * Karena itu keterangan di bawah tombolnya menyebutkan hal itu apa adanya — pengguna yang
 * mengunggah lalu menutup layar akan kehilangan isinya, sama seperti di Pega.
 *
 * # Isian yang TIDAK digambar, dan itu disengaja
 *
 * Procedure `INSERT_SALVAGE` menerima enam parameter yang form Tambah tidak pernah isi:
 * tanggal transfer ke bagian umum, tanggal dan nomor akseptasi, nama pemenang lelang,
 * tanggal lelang, dan tanggal terima. Keenamnya berasal dari isian yang hanya tergambar
 * pada mode UBAH — bukan saat pengajuan dibuat — dan menggambarnya di sini akan meminta
 * pengguna mengisi hal yang belum terjadi.
 */
export function TambahSalvageForm({
  statusOptions,
  uploadColumns,
  onClose,
  onSaved,
}: Props) {
  const [form, setForm] = useState<FormState>(emptyForm)
  const [items, setItems] = useState<DetailItem[]>([])
  const [uploadNote, setUploadNote] = useState('')

  const fileInput = useRef<HTMLInputElement>(null)

  const create = useCreateSalvage()
  const upload = useUploadSalvageDetail()

  // Pelanggaran per isian datang dari server sebagai SENARAI dan dipakai menandai
  // isiannya di tempatnya — bukan sebagai satu pesan di atas form. Pada form berisi tujuh
  // belas isian, satu pesan umum memaksa pengguna menebak isian mana yang dimaksud.
  const violations =
    create.error instanceof APIError ? create.error.violations() : {}

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((current) => ({ ...current, [key]: value }))

  function submit(event: React.FormEvent) {
    event.preventDefault()

    create.mutate(
      {
        ...form,
        mode: 'insert',
        id_salvage: '',
        detail_item_salvage: items,
      },
      {
        onSuccess: (result) => {
          onSaved(result.pesan)
          onClose()
        },
      },
    )
  }

  function chooseFile(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return

    upload.mutate(file, {
      onSuccess: (result) => {
        // Baris yang diunggah DITAMBAHKAN ke yang sudah ada, bukan menggantikannya.
        // `UploadDetailSalvage` memakai `<APPEND>` pada setiap barisnya, sehingga dua
        // berkas yang diunggah berturut-turut menghasilkan gabungan keduanya.
        setItems((current) => [...current, ...result.detail_item_salvage])
        setUploadNote(result.pesan)
      },
    })

    // Nilai input dikosongkan supaya berkas yang SAMA dapat diunggah lagi. Tanpa ini,
    // memilih berkas yang sama dua kali tidak memicu perubahan apa pun.
    event.target.value = ''
  }

  return (
    <form
      onSubmit={submit}
      className="space-y-6 rounded-kartu border border-slate-200 bg-white p-5"
      aria-label="Menambahkan Data Salvage"
    >
      <div>
        <h2 className="text-base font-semibold text-slate-900">
          Menambahkan Data Salvage
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          Pengajuan yang disimpan langsung masuk antrean <strong>Checker</strong>.
        </p>
      </div>

      {create.error != null && (
        <ErrorMessage
          title="Pengajuan tidak tersimpan"
          description={messageOf(create.error)}
          tone="penolakan"
        />
      )}

      <fieldset className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <legend className="sr-only">Data Salvage</legend>

        <FormField
          id="salvage-nomor-klaim"
          label="Nomor Klaim"
          value={form.nomor_klaim}
          onChange={(event) => set('nomor_klaim', event.target.value)}
          failure={violations['nomor_klaim']}
          required
        />
        <FormField
          id="salvage-nama-object"
          label="Nama Object"
          value={form.nama_object}
          onChange={(event) => set('nama_object', event.target.value)}
          failure={violations['nama_object']}
          required
        />
        <FormField
          id="salvage-nama-coverage"
          label="Nama Coverage"
          value={form.nama_coverage}
          onChange={(event) => set('nama_coverage', event.target.value)}
          failure={violations['nama_coverage']}
          required
        />

        <FormField
          id="salvage-tanggal-input"
          label="Tanggal Input"
          type="date"
          value={form.tanggal_input}
          onChange={(event) => set('tanggal_input', event.target.value)}
          failure={violations['tanggal_input']}
        />
        <FormField
          id="salvage-jenis"
          label="Jenis Salvage"
          value={form.jenis_salvage}
          onChange={(event) => set('jenis_salvage', event.target.value)}
          failure={violations['jenis_salvage']}
          required
        />

        {/* "--Pilih--" ada di layar lama apa adanya (`pyCaption --Pilih--`). */}
        <SelectField
          id="salvage-status"
          label="Status Salvage"
          emptyText="--Pilih--"
          options={statusOptions.map((option) => ({
            value: option.kode,
            label: option.label,
          }))}
          value={form.status_salvage}
          onChange={(event) => set('status_salvage', event.target.value)}
          error={violations['status_salvage']}
        />

        <FormField
          id="salvage-lokasi"
          label="Lokasi Salvage"
          value={form.lokasi_salvage}
          onChange={(event) => set('lokasi_salvage', event.target.value)}
          failure={violations['lokasi_salvage']}
        />
        <FormField
          id="salvage-mata-uang"
          label="Mata Uang"
          value={form.mata_uang}
          onChange={(event) => set('mata_uang', event.target.value)}
          failure={violations['mata_uang']}
        />
        <FormField
          id="salvage-quantity"
          label="Quantity Salvage"
          inputMode="decimal"
          value={form.quantity_salvage}
          onChange={(event) => set('quantity_salvage', event.target.value)}
          failure={violations['quantity_salvage']}
        />

        {/*
          Nilai uang diketik sebagai TEKS, bukan `type="number"`.

          `type="number"` menyerahkan pembacaannya ke peramban, dan peramban membulatkannya
          menjadi bilangan pecahan biner — persis yang `D-51` larang. Pemeriksaan bentuknya
          dikerjakan server, yang menolak isian yang bukan angka beserta nama isiannya.
        */}
        <FormField
          id="salvage-minimum"
          label="Minimum Salvage"
          inputMode="decimal"
          value={form.minimum_salvage}
          onChange={(event) => set('minimum_salvage', event.target.value)}
          failure={violations['minimum_salvage']}
        />
        <FormField
          id="salvage-penawaran"
          label="Nilai Penawaran"
          inputMode="decimal"
          value={form.nilai_penawaran}
          onChange={(event) => set('nilai_penawaran', event.target.value)}
          failure={violations['nilai_penawaran']}
        />
        <FormField
          id="salvage-share"
          label="Share Tertanggung"
          inputMode="decimal"
          value={form.share_tertanggung}
          onChange={(event) => set('share_tertanggung', event.target.value)}
          failure={violations['share_tertanggung']}
        />

        <FormField
          id="salvage-email"
          label="Email"
          type="email"
          value={form.email}
          onChange={(event) => set('email', event.target.value)}
          failure={violations['email']}
        />
        <FormField
          id="salvage-pic-survey"
          label="Nama PIC Survey"
          value={form.nama_pic_survey}
          onChange={(event) => set('nama_pic_survey', event.target.value)}
        />
        <FormField
          id="salvage-telp-survey"
          label="No Telp PIC Survey"
          value={form.no_telp_pic_survey}
          onChange={(event) => set('no_telp_pic_survey', event.target.value)}
        />
        <FormField
          id="salvage-email-survey"
          label="Email PIC Survey"
          type="email"
          value={form.email_pic_survey}
          onChange={(event) => set('email_pic_survey', event.target.value)}
        />
      </fieldset>

      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input
          type="checkbox"
          checked={form.lokasi_salvage_di_jabodetabek}
          onChange={(event) =>
            set('lokasi_salvage_di_jabodetabek', event.target.checked)
          }
          className="size-4 rounded border-slate-300"
        />
        Lokasi Salvage Di Jabodatabek
      </label>

      <TextAreaField
        id="salvage-remark"
        label="Remark"
        rows={3}
        value={form.remark}
        onChange={(event) => set('remark', event.target.value)}
        error={violations['remark']}
      />

      <DetailItemTable
        items={items}
        uploadColumns={uploadColumns}
        uploadNote={uploadNote}
        uploadError={upload.error}
        isUploading={upload.isPending}
        fileInput={fileInput}
        onChooseFile={chooseFile}
        onRemove={(index) =>
          setItems((current) => current.filter((_, position) => position !== index))
        }
        failure={violations['detail_item_salvage']}
      />

      <div className="flex flex-wrap items-center gap-2 border-t border-slate-200 pt-4">
        <Button type="submit" disabled={create.isPending}>
          {create.isPending ? 'Menyimpan…' : 'Submit'}
        </Button>
        <Button type="button" tone="halus" onClick={onClose}>
          Batal
        </Button>
      </div>
    </form>
  )
}

type DetailProps = {
  items: DetailItem[]
  uploadColumns: string[]
  uploadNote: string
  uploadError: unknown
  isUploading: boolean
  fileInput: React.RefObject<HTMLInputElement | null>
  onChooseFile: (event: React.ChangeEvent<HTMLInputElement>) => void
  onRemove: (index: number) => void
  failure: string | undefined
}

/**
 * Grid **"Detail Item Salvage"** beserta tombol unggahnya.
 *
 * Di Pega barisnya dapat ditambahkan satu per satu MAUPUN diunggah sekaligus lewat CSV.
 * Yang dibangun di sini adalah jalur unggahnya, karena itulah jalur yang tombolnya
 * ditunjuk Work Owner — dan baris yang sudah masuk tetap dapat dihapus satu per satu.
 */
function DetailItemTable({
  items,
  uploadColumns,
  uploadNote,
  uploadError,
  isUploading,
  fileInput,
  onChooseFile,
  onRemove,
  failure,
}: DetailProps) {
  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-semibold text-slate-900">Detail Item Salvage</h3>

        <div className="flex items-center gap-2">
          <input
            ref={fileInput}
            type="file"
            accept=".csv,text/csv"
            onChange={onChooseFile}
            className="hidden"
            aria-hidden="true"
            tabIndex={-1}
          />
          <Button
            type="button"
            tone="kedua"
            disabled={isUploading}
            onClick={() => fileInput.current?.click()}
          >
            {isUploading ? 'Membaca berkas…' : 'Upload Detail Salvage'}
          </Button>
        </div>
      </div>

      <p className="text-sm text-slate-600">
        Berkas CSV berkolom{' '}
        <span className="font-mono text-xs">{uploadColumns.join(', ')}</span>. Pemisah
        antarkolom adalah <strong>koma</strong> — berkas yang disimpan Excel dengan setelan
        Indonesia memakai titik koma dan tidak akan terbaca.
      </p>

      {uploadError != null && (
        <ErrorMessage
          title="Berkas tidak dapat dibaca"
          description={messageOf(uploadError)}
          tone="penolakan"
        />
      )}
      {uploadNote !== '' && (
        <p className="rounded-kontrol bg-amber-50 px-3 py-2 text-sm text-amber-900">
          {uploadNote}
        </p>
      )}
      {failure !== undefined && (
        <p className="text-sm text-red-700" role="alert">
          {failure}
        </p>
      )}

      {items.length === 0 ? (
        <p className="rounded-kontrol border border-dashed border-slate-300 px-3 py-4 text-sm text-slate-500">
          Belum ada item. Pengajuan tanpa detail item tetap dapat disimpan.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-kartu border border-slate-200">
          <table className="w-full min-w-max text-sm" aria-label="Detail Item Salvage">
            <thead className="bg-slate-50 text-left text-xs font-semibold tracking-wide text-slate-600 uppercase">
              <tr>
                <th scope="col" className="px-4 py-2.5">
                  Nama Item
                </th>
                <th scope="col" className="px-4 py-2.5 text-right">
                  Jumlah Item / Qty
                </th>
                <th scope="col" className="px-4 py-2.5">
                  Satuan
                </th>
                <th scope="col" className="px-4 py-2.5">
                  Remark
                </th>
                <th scope="col" className="px-4 py-2.5 text-right">
                  Aksi
                </th>
              </tr>
            </thead>

            <tbody className="divide-y divide-slate-100">
              {items.map((item, index) => (
                <tr key={`${item.nama_item}-${index}`}>
                  <td className="px-4 py-2 text-slate-900">{item.nama_item}</td>
                  <td className="px-4 py-2 text-right tabular-nums text-slate-900">
                    {item.jumlah_item}
                  </td>
                  <td className="px-4 py-2 text-slate-700">{item.satuan}</td>
                  <td className="px-4 py-2 text-slate-700">{item.remark}</td>
                  <td className="px-4 py-2 text-right">
                    <button
                      type="button"
                      onClick={() => onRemove(index)}
                      className="rounded-kontrol px-2 py-1 text-sm text-red-700 hover:bg-red-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500/50"
                    >
                      Hapus
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

/**
 * today mengembalikan tanggal hari ini berbentuk `YYYY-MM-DD`.
 *
 * Ia dipakai sebagai nilai awal isian "Tanggal Input" saja. Tanggal yang benar-benar
 * tersimpan tetap yang dikirim form ini, dan server tidak menggantinya — berbeda dari
 * `INSERT_PLADLA` yang membuang tanggal pilihan pengguna dan memakai waktu sistem
 * (`D-49` butir 7).
 */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}

function today(): string {
  const now = new Date()
  const bulan = String(now.getMonth() + 1).padStart(2, '0')
  const tanggal = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${bulan}-${tanggal}`
}
