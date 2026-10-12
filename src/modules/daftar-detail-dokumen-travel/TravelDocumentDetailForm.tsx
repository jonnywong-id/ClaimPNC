import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type TravelDocumentChoice, type TravelDocumentDetail } from '@/api/types'
import { Button } from '@/components/Button'
import { ComboField } from '@/components/ComboField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

/**
 * Nilai STSWAJIB yang benar-benar tersimpan di basis data.
 *
 * Angka, bukan teks. `Activity/BrowseDocTravel-Act.xml` membuktikannya lewat precondition
 * langkahnya — `.STSWAJIB==1` dan `.STSWAJIB==0` — dan teks "Ya"/"Tidak" yang dilihat
 * petugas hanya dibentuk untuk ditampilkan.
 *
 * Kontrak API mengirimnya sebagai boolean; kode di bawah yang menjembatani keduanya.
 * Angkanya tetap dipakai sebagai nilai dropdown supaya yang tampil di layar dan yang
 * tersimpan di tabel dapat dibandingkan langsung saat menelusuri masalah.
 */
const MANDATORY_YES = '1'
const MANDATORY_NO = '0'

/**
 * Yang diperiksa di layar hanyalah bentuk angka pada Minimal Unggah, dan itu disengaja.
 *
 * Layar Pega tidak memuat satu pun `pyRequired` bernilai true maupun Validate rule pada
 * `Section/BrowseDocumentTravel-Section.xml`, sehingga ID Dokumen kosong dan nama dokumen
 * kosong keduanya sah. Work Owner menetapkan 2026-09-21 layar disamakan dengan Pega.
 *
 * Minimal Unggah tetap dijaga berupa angka bulat tak negatif karena itu bukan aturan
 * bisnis melainkan bentuk kolomnya: teks yang bukan angka akan ditolak Oracle sebagai
 * galat yang tidak dapat dibaca pengguna.
 */
const schema = z.object({
  id_dokumen: z.string().trim(),
  nama_dokumen: z.string().trim(),
  status_wajib: z.enum([MANDATORY_YES, MANDATORY_NO]),
  // TEKS, bukan angka, meski isiannya tampak angka.
  //
  // Sebabnya bentuk isian HTML: elemen input selalu mengembalikan teks, dan memaksanya
  // menjadi angka di dalam skema membuat isian KOSONG berubah menjadi NaN — lalu
  // dilaporkan sebagai "harus berupa angka" kepada pengguna yang sebenarnya tidak
  // mengetik apa pun. Pada layar tanpa validasi, kosong justru harus diterima.
  //
  // Pengubahannya menjadi angka dikerjakan layar saat menyusun badan permintaan; kosong
  // dibaca sebagai nol.
  minimal_unggah: z
    .string()
    .trim()
    .regex(/^\d*$/, 'Minimal Unggah hanya boleh berisi angka.'),
})

export type TravelDocumentDetailFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: TravelDocumentDetail | null
  /** Pilihan ID Dokumen, dibaca dari POOLDATA.M_DOCTRAVEL milik entitas yang aktif. */
  documents: TravelDocumentChoice[]
  isSaving: boolean
  error: unknown
  onSave: (values: TravelDocumentDetailFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Tidak ada cabang `validationFailed` di sini, dan itu bukan kelalaian: modul ini tanpa
 * validasi server, sehingga tidak ada pelanggaran per isian yang dapat dilaporkan.
 */
function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      // Kode MILIK MODUL INI, bukan `notFound` yang umum — server mengirim
      // `detail_dokumen_travel_tidak_ditemukan`. Penyeragamannya masuk TKT-F1-004.
      case ErrorCode.travelDocumentDetailNotFound:
      case ErrorCode.notFound:
        return {
          title: 'Baris ini sudah tidak ada',
          description: 'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih portal entitas di bagian atas halaman, lalu simpan lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Terjadi kesalahan pada sistem',
          description: 'Isian Anda belum tersimpan. Coba beberapa saat lagi.',
          tone: 'gangguan',
        }
    }
  }
  return null
}

/** Menyusun nilai awal form dari baris yang disunting. */
function valuesOf(edited: TravelDocumentDetail | null): TravelDocumentDetailFields {
  return {
    id_dokumen: edited?.id_dokumen ?? '',
    nama_dokumen: edited?.nama_dokumen ?? '',
    status_wajib: edited?.status_wajib ? MANDATORY_YES : MANDATORY_NO,
    minimal_unggah: String(edited?.minimal_unggah ?? 0),
  }
}

/**
 * TravelDocumentDetailForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `Section/BrowseDocumentTravel-Section.xml`
 * memakai satu halaman `TempDTDocTravel` untuk keduanya dan membedakan modusnya hanya
 * lewat tombol — "Simpan" pada penambahan, "Ubah" pada penyuntingan. Isian yang
 * dibandingkan pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * # Susunan isiannya disamakan dengan Pega
 *
 * Diambil dari urutan kemunculan isiannya di section itu (`D-13`: tata letak ditiru
 * supaya pengguna tidak perlu belajar ulang):
 *
 *	ID              TempDTDocTravel.ID            hanya ditampilkan
 *	ID Dokumen      TempDTDocTravel.DOCID         autocomplete BrowseMstDocTravel_RD
 *	Nama Dokumen    TempDTDocTravel.DOCUMENTNAME  teks
 *	Minimal Unggah  TempDTDocTravel.MINUNGGAH     teks
 *	Status Wajib    TempDTDocTravel.STSWAJIB      dropdown
 *
 * Perhatikan Minimal Unggah berada DI ATAS Status Wajib — urutan itu diambil dari
 * posisinya di section, bukan disusun ulang menurut selera.
 *
 * # Grid Plan dan Jaminan TIDAK ada di sini
 *
 * Dicatat karena ia sempat dibangun lalu dicabut, dan karena export rule menyesatkan di
 * titik ini.
 *
 * `Section/BrowseDocumentTravel-Section.xml:3731` memuat grid berulang
 * `TempDTDocTravel.COVERAGELIST` tanpa kondisi yang menyembunyikannya, sehingga dibaca
 * dari XML saja ia tampak bagian form ini. **Di aplikasi Pega yang berjalan grid itu
 * tidak ada** — Work Owner memeriksa layarnya langsung dan menetapkannya 2026-10-03.
 *
 * Menambahkannya kembali menuntut keputusan Work Owner lebih dulu, bukan sekadar
 * menyalin dari riwayat berkas ini.
 *
 * # Kenapa ID Dokumen ComboField, bukan SelectField
 *
 * Isiannya di Pega adalah autocomplete, dan layar ini tanpa validasi: kode yang tidak ada
 * di master pun diterima. Dropdown akan menutup kemungkinan itu, dan itu perubahan
 * perilaku, bukan perbaikan. Perlakuan yang sama dipakai isian Bisnis pada Master COL
 * Simas Online.
 *
 * Yang TETAP dropdown hanyalah Status Wajib: nilainya memang hanya dua, dan di Pega pun
 * ia `pxDropdown`.
 */
export function TravelDocumentDetailForm({
  edited,
  documents,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const editMode = edited !== null

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<TravelDocumentDetailFields>({
    resolver: zodResolver(schema),
    defaultValues: valuesOf(edited),
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu
  // — misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset(valuesOf(edited))
  }, [edited, reset])

  const message = messageFor(error)

  // Judul yang sama untuk kedua modus, mengikuti layar lama. Beda modus tetap terlihat
  // dari ada-tidaknya baris "ID" di bawahnya dan dari label tombol simpannya.
  const title = 'Detail Dokumen Travel'

  const documentOptions = documents.map((row) => row.id)

  // Nama dokumen menurut master, ditampilkan sebagai keterangan di bawah isian ID
  // Dokumen. Ini TAMBAHAN terhadap Pega — autocomplete di sana menampilkan NAMADOKUMEN
  // di dalam daftarnya, dan keterangan ini menggantikan peran itu setelah pilihannya
  // ditutup. Ia tidak mengisi apa pun; isian Nama Dokumen tetap diketik sendiri, persis
  // seperti Pega yang menandai NAMADOKUMEN `pySetValueOnSelect=false`.
  const chosenDocument = watch('id_dokumen')
  const chosenDocumentName = documents.find((row) => row.id === chosenDocument?.trim())?.nama

  return (
    <form
      onSubmit={handleSubmit(onSave)}
      noValidate
      aria-label={title}
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">{title}</h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — kodenya diterbitkan server dari urutan basis data. Di Pega pun
          isiannya hanya ditampilkan. */}
      {editMode && (
        <div>
          <span className="block text-sm font-medium text-slate-700">ID</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
            {edited.id}
            <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

      <ComboField
        id="id_dokumen"
        label="ID Dokumen"
        options={documentOptions}
        autoFocus
        hint={
          chosenDocumentName
            ? `Master dokumen travel: ${chosenDocumentName}`
            : 'Pilih dari daftar, atau ketik kode lain bila memang dikehendaki.'
        }
        error={errors.id_dokumen?.message}
        {...register('id_dokumen')}
      />

      <Field
        id="nama_dokumen"
        label="Nama Dokumen"
        type="text"
        error={errors.nama_dokumen?.message}
        {...register('nama_dokumen')}
      />

      {/* `type="text"`, bukan `type="number"`, dan itu bukan kelalaian.

          Isiannya di Pega adalah `pxTextInput`, sama seperti Nama Dokumen — bukan isian
          angka. Memakai `type="number"` akan membuat peramban MENOLAK ketikan yang tidak
          valid sebelum sampai ke kode, sehingga penjaga di skema tidak pernah berjalan
          dan pesannya tidak pernah terlihat siapa pun. Yang lebih buruk: peramban
          mengosongkan nilainya diam-diam saat ketikan sementara tidak valid, sehingga
          "-2" tersimpan sebagai kosong tanpa satu pun tanda bagi pengguna.

          `inputMode="numeric"` tetap dipasang supaya papan ketik angka yang muncul di
          tablet surveyor (`D-12`), tanpa mengubah apa yang boleh diketik. */}
      <Field
        id="minimal_unggah"
        label="Minimal Unggah"
        type="text"
        inputMode="numeric"
        error={errors.minimal_unggah?.message}
        {...register('minimal_unggah')}
      />

      <SelectField
        id="status_wajib"
        label="Status Wajib"
        options={[
          { value: MANDATORY_YES, label: 'Ya' },
          { value: MANDATORY_NO, label: 'Tidak' },
        ]}
        error={errors.status_wajib?.message}
        {...register('status_wajib')}
      />

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        {/* Label tombolnya mengikuti layar Pega, yang memakai "Simpan" dan "Ubah" untuk
            kedua modusnya. */}
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : editMode ? 'Ubah' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}

/** MANDATORY_YES diekspor supaya layar dapat mengubah isian form menjadi boolean kontrak. */
export { MANDATORY_YES }
