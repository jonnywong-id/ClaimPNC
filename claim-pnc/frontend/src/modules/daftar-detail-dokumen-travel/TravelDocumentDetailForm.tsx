import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type TravelCoverage,
  type TravelDocumentChoice,
  type TravelDocumentDetail,
  type TravelPlan,
} from '@/api/types'
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
 * `Section/BrowseDocumentTravel-Section.xml`, sehingga ID Dokumen kosong, nama dokumen
 * kosong, dan baris jaminan yang hanya terisi sebagian semuanya sah. Work Owner
 * menetapkan 2026-09-21 layar disamakan dengan Pega.
 *
 * Minimal Unggah tetap dijaga berupa angka bulat tak negatif karena itu bukan aturan
 * bisnis melainkan bentuk kolomnya: teks yang bukan angka akan ditolak Oracle sebagai
 * galat yang tidak dapat dibaca pengguna, dan "paling sedikit minus satu berkas" bukan
 * aturan yang dapat dipenuhi maupun dilanggar.
 */
const schema = z.object({
  id_dokumen: z.string().trim(),
  nama_dokumen: z.string().trim(),
  status_wajib: z.enum([MANDATORY_YES, MANDATORY_NO]),
  // TEKS, bukan angka, meski isiannya `type="number"`.
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
  // Disimpan sebagai senarai objek karena useFieldArray menuntut setiap barisnya berupa
  // objek agar dapat memberinya kunci yang stabil.
  jaminan: z.array(
    z.object({
      nama_plan: z.string().trim(),
      nama_jaminan: z.string().trim(),
    }),
  ),
})

export type TravelDocumentDetailFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: TravelDocumentDetail | null
  /** Pilihan ID Dokumen, dibaca dari POOLDATA.M_DOCTRAVEL milik entitas yang aktif. */
  documents: TravelDocumentChoice[]
  /** Pilihan Nama Plan, dibaca dari POOLDATA.M_PLANTRAVEL milik GISFW. */
  plans: TravelPlan[]
  /** Pilihan Nama Jaminan beserta plan pemiliknya. */
  coverages: TravelCoverage[]
  /** Pembatasan plan baris yang disunting masih dimuat. */
  isLoadingCoverageMapping: boolean
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
    jaminan: (edited?.jaminan ?? []).map((row) => ({
      nama_plan: row.nama_plan,
      nama_jaminan: row.nama_jaminan,
    })),
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
 *	Nama Plan       COVERAGELIST.PLANNAME         grid berulang, autocomplete
 *	Nama Jaminan    COVERAGELIST.COVERAGENAME     grid berulang, autocomplete disaring plan
 *
 * Perhatikan Minimal Unggah berada DI ATAS Status Wajib — urutan itu diambil dari
 * posisinya di section, bukan disusun ulang menurut selera.
 *
 * # Kenapa isiannya ComboField, bukan SelectField
 *
 * Ketiga isian berdaftar di Pega adalah autocomplete, dan `SearchCoverageTravel_RD`
 * bahkan ber-`pyAllowFreeFormInput=true` — nama di luar daftar TETAP boleh diketik dan
 * tersimpan. Dropdown akan menutup kemungkinan itu, dan itu perubahan perilaku, bukan
 * perbaikan. Perlakuan yang sama sudah dipakai isian Bisnis pada Master COL Simas Online.
 *
 * Yang TETAP dropdown hanyalah Status Wajib: nilainya memang hanya dua, dan di Pega pun
 * ia `pxDropdown`.
 */
export function TravelDocumentDetailForm({
  edited,
  documents,
  plans,
  coverages,
  isLoadingCoverageMapping,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const editMode = edited !== null

  const {
    control,
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<TravelDocumentDetailFields>({
    resolver: zodResolver(schema),
    defaultValues: valuesOf(edited),
  })

  // `jaminan` adalah grid, bukan satu isian. useFieldArray yang memegang barisnya supaya
  // setiap baris punya kunci stabil — tanpa itu, menghapus baris tengah akan membuat
  // React memakai ulang elemen input dan nilai baris berikutnya ikut bergeser.
  const { fields, append, remove } = useFieldArray({ control, name: 'jaminan' })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu
  // — misalnya pengguna menekan "Ubah" pada baris lain, atau saat pembatasan plan-nya
  // baru selesai dimuat dari server.
  useEffect(() => {
    reset(valuesOf(edited))
  }, [edited, reset])

  const message = messageFor(error)

  // Judul yang sama untuk kedua modus, mengikuti layar lama. Beda modus tetap terlihat
  // dari ada-tidaknya baris "ID" di bawahnya dan dari label tombol simpannya.
  const title = 'Detail Dokumen Travel'

  const documentOptions = documents.map((row) => row.id)
  const planOptions = plans.map((row) => row.nama)

  // Nama dokumen menurut master, ditampilkan sebagai keterangan di bawah isian ID
  // Dokumen. Ini TAMBAHAN terhadap Pega — autocomplete di sana menampilkan NAMADOKUMEN
  // di dalam daftarnya, dan keterangan ini menggantikan peran itu setelah pilihannya
  // ditutup. Ia tidak mengisi apa pun; isian Nama Dokumen tetap diketik sendiri, persis
  // seperti Pega yang menandai NAMADOKUMEN `pySetValueOnSelect=false`.
  const chosenDocument = watch('id_dokumen')
  const chosenDocumentName = documents.find((row) => row.id === chosenDocument?.trim())?.nama

  const rows = watch('jaminan')

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

      {/* Jaminan adalah GRID, bukan satu isian. Di Pega ia
          `pyPageListProperty = TempDTDocTravel.COVERAGELIST` — satu aturan dokumen dapat
          dibatasi pada banyak kombinasi plan dan jaminan. */}
      <fieldset className="rounded-kontrol border border-slate-200 p-4">
        <legend className="px-1 text-sm font-medium text-slate-700">Plan dan Jaminan</legend>

        {isLoadingCoverageMapping ? (
          <p className="text-sm text-slate-500">Memuat plan dan jaminan yang sudah dipilih…</p>
        ) : fields.length === 0 ? (
          <p className="text-sm text-slate-500">
            Belum ada pembatasan. Dibiarkan kosong berarti aturan dokumen ini berlaku untuk
            seluruh plan dan jaminan.
          </p>
        ) : (
          <ul className="space-y-3">
            {fields.map((row, index) => {
              // Jaminan disaring menurut plan yang dipilih pada BARIS INI — persis
              // penyaringan yang dilakukan `SearchCoverageTravel_RD` lewat parameter
              // `plan`. Bedanya penyaringan dikerjakan di sini, atas daftar yang sudah di
              // tangan, bukan dengan menembak server sekali per baris.
              //
              // Plan yang diketik bebas tidak cocok dengan satu pun baris master, dan
              // pada keadaan itu SELURUH jaminan ditawarkan — bukan daftar kosong. Daftar
              // kosong akan terbaca sebagai "tidak ada jaminan untuk plan ini", padahal
              // yang benar adalah "plan ini tidak dikenali master".
              const planName = rows?.[index]?.nama_plan?.trim() ?? ''
              const planID = plans.find((plan) => plan.nama === planName)?.id
              const coverageOptions = (
                planID ? coverages.filter((item) => item.id_plan === planID) : coverages
              ).map((item) => item.nama)

              return (
                <li key={row.id} className="grid gap-2 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
                  <ComboField
                    id={`jaminan-${index}-plan`}
                    label={`Nama Plan baris ${index + 1}`}
                    options={planOptions}
                    error={errors.jaminan?.[index]?.nama_plan?.message}
                    {...register(`jaminan.${index}.nama_plan` as const)}
                  />
                  <ComboField
                    id={`jaminan-${index}-jaminan`}
                    label={`Nama Jaminan baris ${index + 1}`}
                    options={coverageOptions}
                    error={errors.jaminan?.[index]?.nama_jaminan?.message}
                    {...register(`jaminan.${index}.nama_jaminan` as const)}
                  />
                  <Button
                    tone="halus"
                    onClick={() => remove(index)}
                    aria-label={`Hapus pembatasan baris ${index + 1}`}
                  >
                    Hapus
                  </Button>
                </li>
              )
            })}
          </ul>
        )}

        <div className="mt-3">
          <Button tone="kedua" onClick={() => append({ nama_plan: '', nama_jaminan: '' })}>
            Tambah Plan
          </Button>
          {/* Daftar saran yang gagal dimuat TIDAK menghalangi apa pun: namanya memang
              boleh diketik sendiri. Yang hilang hanya kenyamanan memilih. */}
          {planOptions.length === 0 && (
            <p className="mt-2 text-sm text-slate-500">
              Daftar plan belum dapat dimuat, jadi tidak ada saran. Nama plan dan jaminan tetap
              dapat diketik sendiri.
            </p>
          )}
        </div>
      </fieldset>

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        {/* Label tombolnya mengikuti layar Pega, yang memakai "Simpan" dan "Ubah" untuk
            kedua modusnya. */}
        <Button type="submit" tone="utama" disabled={isSaving || isLoadingCoverageMapping}>
          {isSaving ? 'Menyimpan…' : editMode ? 'Ubah' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}

/** MANDATORY_YES diekspor supaya layar dapat mengubah isian form menjadi boolean kontrak. */
export { MANDATORY_YES }
