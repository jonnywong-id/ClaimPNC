import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type BusinessDocumentRule, type MasterChoice } from '@/api/types'
import { Button } from '@/components/Button'
import { ComboField } from '@/components/ComboField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'

/**
 * labelOf menyusun teks yang dilihat petugas pada isian autocomplete.
 *
 * Nama diikuti kodenya, supaya dua master yang kebetulan bernama sama tetap dapat
 * dibedakan — dan supaya kode yang tersimpan terbaca tanpa membuka basis data.
 */
export function labelOf(item: MasterChoice): string {
  return `${item.nama} (${item.id})`
}

/**
 * idFromLabel mencari kembali kode dari teks yang diketik petugas.
 *
 * Mengembalikan teks kosong bila tidak ada yang cocok, dan itu MENIRU Pega: autocomplete
 * di sana ber-`pyAllowFreeFormInput=true`, sehingga teks di luar daftar tetap boleh
 * diketik — yang terjadi hanyalah properti kode di belakangnya tidak terisi
 * (`Section/InputListDetailTypeDocumentBusiness_sect-Section.xml:1574`, `:6461`, `:6745`,
 * `:6987`).
 *
 * Pencocokan juga menerima KODE yang diketik langsung. Petugas lama hafal kodenya, dan
 * memaksanya mengetik nama lengkap hanya untuk memilih baris yang sudah ia kenal adalah
 * hambatan yang tidak ada di layar lama.
 */
export function idFromLabel(list: MasterChoice[], typed: string): string {
  const clean = typed.trim()
  if (clean === '') return ''

  const matched = list.find(
    (item) => labelOf(item) === clean || item.nama === clean || item.id === clean,
  )
  return matched?.id ?? ''
}

/** labelForID menyusun teks awal isian dari kode yang sudah tersimpan. */
function labelForID(list: MasterChoice[], id: string, fallback: string): string {
  if (id === '') return ''
  const matched = list.find((item) => item.id === id)
  // Baris yang kodenya tidak lagi ada di master tetap menampilkan SESUATU — nama yang
  // ikut terbaca dari join, atau kodenya. Mengosongkannya akan membuat penyimpanan
  // berikutnya diam-diam menghapus kode yang sebenarnya masih terpakai.
  return matched ? labelOf(matched) : fallback || id
}

/**
 * Skema sengaja nyaris tanpa aturan.
 *
 * Layar lama tidak memuat satu pun isian ber-`pyRequired` bernilai true dan tidak
 * memasang Validate rule; satu-satunya validasinya menjaga pilihan BISNIS, dan itu hanya
 * berlaku pada layar Tambah. Menambahkan aturan di sini berarti menolak penyimpanan yang
 * di sistem lama berhasil (`P-5`).
 *
 * Ketiga isian rujukan disimpan sebagai TEKS yang diketik, bukan sebagai kode. Kodenya
 * dicarikan saat menyusun badan permintaan — persis seperti autocomplete Pega yang
 * menyalin `.ID` ke properti tersembunyi saat sebuah saran dipilih.
 *
 * `minimum_dokumen` juga teks, bukan angka: `z.coerce.number()` membuat masukannya
 * bertipe `unknown` sehingga React Hook Form menolak resolver-nya, dan isian kosong yang
 * berubah menjadi `NaN` akan dilaporkan sebagai "harus berupa angka" kepada pengguna yang
 * sebenarnya tidak mengetik apa pun.
 */
const schema = z.object({
  tipe_dokumen: z.string(),
  object_dokumen: z.string(),
  detail_dokumen_pilihan: z.string(),
  detail_dokumen: z.string(),
  status_wajib: z.boolean(),
  minimum_dokumen: z.string(),
})

type FormFields = z.infer<typeof schema>

/** Nilai yang dikirim ke pemanggil — sudah berupa kode, bukan teks yang diketik. */
export type BusinessDocumentRuleFields = {
  id_tipe_dokumen: string
  id_object_dokumen: string
  id_detail_dokumen: string
  detail_dokumen: string
  status_wajib: boolean
  minimum_dokumen: string
}

type Props = {
  edited: BusinessDocumentRule
  documentTypes: MasterChoice[]
  detailDocuments: MasterChoice[]
  objectDocuments: MasterChoice[]
  isSaving: boolean
  error: unknown
  onSave: (values: BusinessDocumentRuleFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa sambungan jaringan, lalu simpan sekali lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.businessDocumentRuleNotFound:
        return {
          title: 'Aturan dokumen sudah tidak ada',
          description:
            'Baris ini mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih entitas di bagian atas layar, lalu simpan sekali lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description: 'Hubungi tim infrastruktur bila keadaan ini berlanjut.',
          tone: 'gangguan',
        }
      default:
        return { title: 'Penyimpanan gagal', description: error.message, tone: 'gangguan' }
    }
  }
  return null
}

/**
 * Form penyuntingan satu aturan dokumen.
 *
 * # Ketiga isian rujukan adalah autocomplete, bukan dropdown
 *
 * Di Pega ketiganya `pyAutoComplete` ber-`pyAllowFreeFormInput=true` yang menampilkan
 * NAMA dan menyalin KODE ke properti tersembunyi — petugas mengetik untuk mencari, bukan
 * menggulir daftar.
 *
 * Versi pertama modul ini memakai dropdown, dengan alasan "yang tersimpan hanya kode,
 * sehingga isian bebas tidak bermakna". Itu penilaian saya sendiri, dan ia menghapus
 * kemampuan ketik-cari yang di layar lama justru satu-satunya cara memakai master
 * sepanjang ratusan baris. Dicabut atas keputusan Work Owner 2026-09-23 — layar ini
 * mengikuti Pega apa adanya.
 *
 * Nama dokumen — DETAIL_DOKUMEN — TETAP isian bebas biasa, karena ia memang teks yang
 * tersimpan apa adanya, bukan rujukan ke master.
 */
export function BusinessDocumentRuleForm({
  edited,
  documentTypes,
  detailDocuments,
  objectDocuments,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  function seed(): FormFields {
    return {
      tipe_dokumen: labelForID(documentTypes, edited.id_tipe_dokumen, edited.tipe_dokumen),
      object_dokumen: labelForID(objectDocuments, edited.id_object_dokumen, edited.object_dokumen),
      detail_dokumen_pilihan: labelForID(detailDocuments, edited.id_detail_dokumen, ''),
      detail_dokumen: edited.detail_dokumen,
      status_wajib: edited.status_wajib,
      minimum_dokumen: String(edited.minimum_dokumen),
    }
  }

  const { register, handleSubmit, reset, watch } = useForm<FormFields>({
    resolver: zodResolver(schema),
    defaultValues: seed(),
  })

  useEffect(() => {
    reset(seed())
    // Daftar pilihan ikut menjadi ketergantungan: ia dimuat terpisah dan dapat tiba
    // SESUDAH form terbuka. Tanpa ini, isian akan tetap menampilkan kode mentah meski
    // namanya sudah di tangan.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [edited, documentTypes, detailDocuments, objectDocuments, reset])

  // Pilihan Detail Dokumen menyempit mengikuti Tipe Dokumen yang sedang dipilih — persis
  // parameter `idDocument` pada autocomplete `BrowseVLstDetTypeDoc_RD`.
  const selectedDocumentTypeID = idFromLabel(documentTypes, watch('tipe_dokumen'))
  const narrowedDetails = selectedDocumentTypeID
    ? detailDocuments.filter((item) => item.id_induk === selectedDocumentTypeID)
    : detailDocuments

  const message = messageFor(error)

  function submit(values: FormFields) {
    onSave({
      id_tipe_dokumen: idFromLabel(documentTypes, values.tipe_dokumen),
      id_object_dokumen: idFromLabel(objectDocuments, values.object_dokumen),
      id_detail_dokumen: idFromLabel(detailDocuments, values.detail_dokumen_pilihan),
      detail_dokumen: values.detail_dokumen,
      status_wajib: values.status_wajib,
      minimum_dokumen: values.minimum_dokumen,
    })
  }

  return (
    <form
      onSubmit={handleSubmit(submit)}
      noValidate
      aria-label="Update Data"
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      {/* Judulnya ditiru dari layar lama apa adanya (`D-13`). */}
      <h2 className="text-base font-semibold text-slate-900">Update Data</h2>

      {message && (
        <ErrorMessage
          title={message.title}
          description={message.description}
          tone={message.tone}
        />
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <span className="block text-sm font-medium text-slate-700">ID</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-slate-600">
            {edited.id}
            <span className="ml-2 font-sans text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>

        {/*
          Nama Bisnis ditampilkan tetapi TIDAK dapat disunting, dan itu bukan pembatasan
          yang ditambahkan: procedure lama pun tidak mengubah BUSINESSID saat memperbarui
          baris. Aturan dokumen disalin ke lini bisnis lain, bukan dipindahkan.
        */}
        <div>
          <span className="block text-sm font-medium text-slate-700">Nama Bisnis</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
            {edited.nama_bisnis || <span className="text-slate-400">—</span>}
            <span className="ml-2 text-xs text-slate-500">(tidak dapat dipindah)</span>
          </p>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <ComboField
          id="tipe_dokumen"
          label="Tipe Dokumen"
          options={documentTypes.map(labelOf)}
          autoComplete="off"
          hint="Tahap klaim tempat dokumen ini diminta."
          {...register('tipe_dokumen')}
        />
        <ComboField
          id="object_dokumen"
          label="Object Dokumen"
          options={objectDocuments.map(labelOf)}
          autoComplete="off"
          hint="Boleh dikosongkan."
          {...register('object_dokumen')}
        />
      </div>

      <ComboField
        id="detail_dokumen_pilihan"
        label="Detail Dokumen"
        options={narrowedDetails.map(labelOf)}
        autoComplete="off"
        hint="Daftarnya menyempit mengikuti Tipe Dokumen yang dipilih di atas."
        {...register('detail_dokumen_pilihan')}
      />

      <Field
        id="detail_dokumen"
        label="Nama Dokumen"
        type="text"
        autoComplete="off"
        hint={
          'Teks yang dibaca petugas di layar unggah. Isi tepat "-" untuk menyembunyikan ' +
          'dokumen ini dari seluruh layar unggah tanpa menghapus barisnya.'
        }
        {...register('detail_dokumen')}
      />

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex items-center gap-2 pt-6">
          <input
            id="status_wajib"
            type="checkbox"
            className="h-4 w-4 rounded border-slate-300"
            {...register('status_wajib')}
          />
          <label htmlFor="status_wajib" className="text-sm font-medium text-slate-700">
            Status Wajib
          </label>
        </div>

        <Field
          id="minimum_dokumen"
          label="Minimum Dokumen"
          type="text"
          inputMode="numeric"
          autoComplete="off"
          hint="Berkas paling sedikit yang harus diunggah. Kosong atau 0 berarti tanpa tuntutan jumlah."
          {...register('minimum_dokumen')}
        />
      </div>

      {/*
        Peringatan ini ada karena "Status Wajib" di layar ini TIDAK cukup membuat dokumen
        menjadi wajib di klaim — kueri klaim menghitung jaminannya lebih dulu, dan nol
        jaminan berarti "Tidak". Ia keterangan atas aturan yang SUDAH ADA, bukan aturan
        baru: tidak satu pun nilai yang tersimpan berubah karenanya.
      */}
      {watch('status_wajib') && edited.jenis_klaim.length === 0 && (
        <ErrorMessage
          title="Wajib di sini belum berarti wajib di klaim"
          description={
            'Baris ini ditandai wajib tetapi belum punya satu Jenis Klaim pun. Selama ' +
            'daftar Jenis Klaim kosong, dokumen ini tetap terbaca TIDAK wajib pada setiap ' +
            'klaim. Tambahkan Jenis Klaim di bawah setelah menyimpan.'
          }
          tone="gangguan"
        />
      )}

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}
