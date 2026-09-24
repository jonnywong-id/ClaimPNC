import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DetailDocumentType, type DetailDocumentTypeChoice } from '@/api/types'
import { Button } from '@/components/Button'
import { ComboField } from '@/components/ComboField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

/**
 * Nilai dropdown Status Wajib.
 *
 * Teks "Ya"/"Tidak" — sama dengan yang DISIMPAN server (keputusan Work Owner 2026-09-23),
 * bukan angka. Kontrak API mengirimnya sebagai boolean; kode di bawah yang menjembatani
 * keduanya.
 *
 * Nilainya dibuat sama persis dengan yang tersimpan supaya apa yang tampil di layar dan
 * apa yang ada di tabel dapat dibandingkan langsung saat menelusuri masalah.
 */
const MANDATORY_YES = 'Ya'
const MANDATORY_NO = 'Tidak'

/**
 * Batas panjang isian, disamakan dengan konstanta di lapisan domain.
 *
 * Ini BUKAN aturan bisnis melainkan bentuk kolom — layar Pega tidak memuat satu pun
 * `pyMaxLength` pada isian mana pun. Angkanya ada di sini supaya pesannya muncul sebelum
 * permintaan dikirim, bukan sesudah server menolaknya; server tetap memeriksanya sendiri.
 */
const MAX_DETAIL = 200
const MAX_INSURED_STATUS = 100
const MAX_RISK = 50
const MAX_REFERENCE = 50
const MAX_DESCRIPTION = 200

/**
 * Yang diperiksa di layar hanyalah panjang isian dan bentuk angka pada Resiko dan
 * Minimum Dokumen.
 *
 * Layar Pega tidak memuat satu pun `pyRequired` bernilai true maupun Rule-Obj-Validate
 * pada `Section/BrowseListDetailTypeDocument-Section.xml`, sehingga Detail Dokumen kosong,
 * rujukan kosong, dan kode yang tidak ada di master semuanya sah (`P-5`).
 *
 * Resiko dijaga berbentuk angka karena `Activity/SetTypePDFAdjustment-Act.xml` membacanya
 * dengan `@toDecimal(.RISK)` — teks yang bukan angka akan terbaca nol di sana tanpa satu
 * pun tanda. KOSONG tetap diterima, karena `toDecimal("")` pun menghasilkan nol.
 */
const schema = z.object({
  id_tipe_dokumen: z.string().trim().max(MAX_REFERENCE, `Paling panjang ${MAX_REFERENCE} karakter.`),
  detail_dokumen: z.string().trim().max(MAX_DETAIL, `Paling panjang ${MAX_DETAIL} karakter.`),
  status_tertanggung: z
    .string()
    .trim()
    .max(MAX_INSURED_STATUS, `Paling panjang ${MAX_INSURED_STATUS} karakter.`),
  // Yang ada di form adalah KETERANGANNYA, bukan kodenya — itulah yang dilihat dan
  // diketik petugas. Kodenya bukan isian: ia dicarikan layar saat permintaan disusun,
  // meniru `pyPropertyTarget` pada autocomplete Pega.
  keterangan_penyebab_kerugian: z
    .string()
    .trim()
    .max(MAX_DESCRIPTION, `Paling panjang ${MAX_DESCRIPTION} karakter.`),
  keterangan_objek_dokumen: z
    .string()
    .trim()
    .max(MAX_DESCRIPTION, `Paling panjang ${MAX_DESCRIPTION} karakter.`),
  // TEKS, bukan angka, meski isinya dibaca sebagai bilangan.
  //
  // Memaksanya menjadi angka di dalam skema membuat isian KOSONG berubah menjadi NaN —
  // lalu dilaporkan sebagai "harus berupa angka" kepada pengguna yang sebenarnya tidak
  // mengetik apa pun. Pada isian yang kosongnya sah, itu penolakan yang salah.
  resiko: z
    .string()
    .trim()
    .max(MAX_RISK, `Paling panjang ${MAX_RISK} karakter.`)
    .regex(/^-?\d*([.,]\d+)?$/, 'Resiko hanya boleh berisi angka.'),
  // Disimpan sebagai senarai objek karena useFieldArray menuntut setiap barisnya berupa
  // objek agar dapat memberinya kunci yang stabil.
  bisnis: z.array(
    z.object({
      id_bisnis: z.string().trim().max(MAX_REFERENCE, `Paling panjang ${MAX_REFERENCE} karakter.`),
      status_wajib: z.enum([MANDATORY_YES, MANDATORY_NO]),
      minimum_dokumen: z
        .string()
        .trim()
        .regex(/^\d*$/, 'Minimum Dokumen hanya boleh berisi angka.'),
    }),
  ),
})

export type DetailDocumentTypeFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: DetailDocumentType | null
  /** Pilihan ID Tipe Dokumen, dibaca dari POOLDATA.V_LST_DOC_TYPE. */
  documentTypes: DetailDocumentTypeChoice[]
  /** Pilihan Dokumen kolom ID, dibaca dari POOLDATA.M_CAUSE_OF_LOSS. */
  causesOfLoss: DetailDocumentTypeChoice[]
  /** Pilihan Objek Dokumen, dibaca dari POOLDATA.V_LST_DOC_OBJ. */
  objectDocuments: DetailDocumentTypeChoice[]
  /** Pilihan ID Bisnis, dibaca dari POOLDATA.BUSINESS milik GISFW. */
  businesses: DetailDocumentTypeChoice[]
  /** Daftar bisnis baris yang disunting masih dimuat. */
  isLoadingBusinessRules: boolean
  isSaving: boolean
  error: unknown
  onSave: (values: DetailDocumentTypeFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti. */
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
      case ErrorCode.validationFailed:
        return {
          title: 'Ada isian yang belum benar',
          description:
            'Periksa keterangan di bawah setiap isian, perbaiki, lalu simpan lagi. Isian Anda belum tersimpan.',
          tone: 'penolakan',
        }
      case ErrorCode.notFound:
        return {
          title: 'Baris ini sudah tidak ada',
          description:
            'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
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
function valuesOf(edited: DetailDocumentType | null): DetailDocumentTypeFields {
  return {
    id_tipe_dokumen: edited?.id_tipe_dokumen ?? '',
    detail_dokumen: edited?.detail_dokumen ?? '',
    status_tertanggung: edited?.status_tertanggung ?? '',
    keterangan_penyebab_kerugian: edited?.keterangan_penyebab_kerugian ?? '',
    keterangan_objek_dokumen: edited?.keterangan_objek_dokumen ?? '',
    resiko: edited?.resiko ?? '',
    bisnis: (edited?.bisnis ?? []).map((row) => ({
      id_bisnis: row.id_bisnis,
      status_wajib: row.status_wajib ? MANDATORY_YES : MANDATORY_NO,
      minimum_dokumen: String(row.minimum_dokumen ?? 0),
    })),
  }
}

/** Mencari keterangan sebuah kode pada daftar pilihannya. */
function describe(choices: DetailDocumentTypeChoice[], id: string | undefined): string | undefined {
  const key = id?.trim()
  if (!key) {
    return undefined
  }
  return choices.find((row) => row.id === key)?.nama
}

/**
 * Mencari kode sebuah keterangan pada daftar pilihannya — arah kebalikan `describe`.
 *
 * Dipakai kedua isian yang bekerja atas keterangan. `undefined` berarti keterangan itu
 * tidak ada di master, dan itu keadaan yang SAH: `pyAllowFreeFormInput=true` di layar lama
 * mengizinkannya, dan yang tersimpan hanyalah keterangannya tanpa kode.
 *
 * Diekspor supaya layar dapat memakainya saat menyusun badan permintaan — kode dan
 * keterangan dikirim berpasangan.
 */
export function codeOf(
  choices: DetailDocumentTypeChoice[],
  name: string | undefined,
): string | undefined {
  const key = name?.trim()
  if (!key) {
    return undefined
  }
  return choices.find((row) => row.nama === key)?.id
}

/**
 * DetailDocumentTypeForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama:
 * `Section/BrowseListDetailTypeDocument-Section.xml` memakai satu halaman `TempDTDoc`
 * untuk keduanya dan membedakan modusnya hanya lewat `TempDTDoc.pyLabel` yang berisi
 * `"tambah"` atau `"update"`. Isian yang dibandingkan pengguna karena itu berada di
 * tempat yang sama pada kedua mode.
 *
 * # Susunan isiannya disamakan dengan Pega
 *
 * Diambil dari urutan kemunculan isiannya di section itu (`D-13`: tata letak ditiru
 * supaya pengguna tidak perlu belajar ulang):
 *
 *	ID                 TempDTDoc.ID               hanya ditampilkan
 *	ID Tipe Dokumen    TempDTDoc.DOC_TYPE_ID      daftar V_LST_DOC_TYPE, isinya KODE
 *	Detail Dokumen     TempDTDoc.DETAIL_DOCUMENT  teks
 *	Status Tertanggung TempDTDoc.STS_INSURED      teks
 *	Dokumen kolom ID   TempDTDoc.DOC_COL_INFO     autocomplete, isinya KETERANGAN
 *	                   -> target TempDTDoc.DOC_COL_ID  (.M_COL_ID)
 *	Objek Dokumen      TempDTDoc.OBJ_DOC_DESC     autocomplete, isinya KETERANGAN
 *	                   -> target TempDTDoc.OBJ_DOC     (.ID)
 *	Resiko             TempDTDoc.RISK             teks
 *	ID Bisnis          DFT_BISNIS_ID[].Note       grid berulang, isinya NAMA
 *	                   -> target .ID
 *	Status Wajib       DFT_BISNIS_ID[].STS_WAJIB  grid berulang, dropdown
 *	Minimum Dokumen    DFT_BISNIS_ID[].MIN_DOC    grid berulang, teks
 *
 * # Isian mana yang berisi kode, dan mana yang berisi keterangan
 *
 * Dibaca dari section, bukan ditebak dari labelnya:
 *
 *	ID Tipe Dokumen   isian = KODE          labelnya pun menyebutnya "ID"
 *	Dokumen kolom ID  isian = KETERANGAN    kodenya target tersembunyi
 *	Objek Dokumen     isian = KETERANGAN    kodenya target tersembunyi
 *	ID Bisnis         isian = NAMA          kodenya target tersembunyi
 *
 * Label "Dokumen kolom ID" menyesatkan — isiannya bukan ID melainkan keterangan. Labelnya
 * ditiru apa adanya (`D-13`); yang tidak ditiru adalah kekeliruannya.
 *
 * Ketiga autocomplete ber-`pyAllowFreeFormInput=true`, sehingga keterangan di luar master
 * tetap boleh diketik. Untuk kedua isian tengah, keterangannya BENAR-BENAR TERSIMPAN di
 * kolomnya sendiri (DOC_COL_INFO, OBJ_DOC_DESC) sehingga ketikan bebas selamat. Untuk ID
 * Bisnis tidak: view anaknya hanya menyimpan kodenya, dan namanya dijoin — sehingga di
 * sana isiannya bekerja atas KODE, lihat grid di bawah.
 *
 * Kode yang menyertai kedua keterangan ditampilkan sebagai `hint` begitu keterangannya
 * cocok dengan master — menggantikan peran daftar autocomplete Pega setelah pilihannya
 * ditutup.
 *
 * # Kenapa ComboField, bukan SelectField
 *
 * Keempatnya autocomplete di Pega, bukan dropdown — kode di luar daftar TETAP boleh
 * diketik dan tersimpan. Dropdown akan menutup kemungkinan itu, dan itu perubahan
 * perilaku, bukan perbaikan. Perlakuan yang sama sudah dipakai isian Bisnis pada Master
 * COL Simas Online dan isian ID Dokumen pada Daftar Detail Dokumen Travel.
 *
 * Yang TETAP dropdown hanyalah Status Wajib: nilainya memang hanya dua, dan di Pega pun
 * ia daftar tertutup.
 */
export function DetailDocumentTypeForm({
  edited,
  documentTypes,
  causesOfLoss,
  objectDocuments,
  businesses,
  isLoadingBusinessRules,
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
  } = useForm<DetailDocumentTypeFields>({
    resolver: zodResolver(schema),
    defaultValues: valuesOf(edited),
  })

  // `bisnis` adalah grid, bukan satu isian. useFieldArray yang memegang barisnya supaya
  // setiap baris punya kunci stabil — tanpa itu, menghapus baris tengah akan membuat
  // React memakai ulang elemen input dan nilai baris berikutnya ikut bergeser.
  const { fields, append, remove } = useFieldArray({ control, name: 'bisnis' })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu
  // — misalnya pengguna menekan "Ubah" pada baris lain, atau saat daftar bisnisnya baru
  // selesai dimuat dari server.
  useEffect(() => {
    reset(valuesOf(edited))
  }, [edited, reset])

  const message = messageFor(error)

  // Judul yang sama untuk kedua modus, mengikuti layar lama. Beda modus tetap terlihat
  // dari ada-tidaknya baris "ID" di bawahnya dan dari label tombol simpannya.
  const title = 'Detail Tipe Dokumen'

  // Isian "ID Tipe Dokumen" memang berisi KODE — labelnya pun menyebutnya demikian —
  // sehingga keterangannya yang ditampilkan sebagai petunjuk.
  const chosenDocumentType = describe(documentTypes, watch('id_tipe_dokumen'))
  // Kedua isian berikut berisi KETERANGAN, sehingga yang ditampilkan sebagai petunjuk
  // adalah kodenya — kebalikan dari yang di atas. Kosong berarti keterangannya tidak ada
  // di master, dan itu sah.
  const chosenCauseOfLoss = codeOf(causesOfLoss, watch('keterangan_penyebab_kerugian'))
  const chosenObjectDocument = codeOf(objectDocuments, watch('keterangan_objek_dokumen'))
  const rows = watch('bisnis')

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
          isiannya `pyEditOptions=Read-only`. */}
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
        id="id_tipe_dokumen"
        label="ID Tipe Dokumen"
        options={documentTypes.map((row) => row.id)}
        autoFocus
        hint={
          chosenDocumentType
            ? `Tipe dokumen: ${chosenDocumentType}`
            : 'Pilih dari daftar, atau ketik kode lain bila memang dikehendaki.'
        }
        error={errors.id_tipe_dokumen?.message}
        {...register('id_tipe_dokumen')}
      />

      <Field
        id="detail_dokumen"
        label="Detail Dokumen"
        type="text"
        hint="Nama rincian dokumen yang dibaca petugas, dan yang tampil sebagai pilihan di layar Daftar Tipe Dokumen Bisnis."
        error={errors.detail_dokumen?.message}
        {...register('detail_dokumen')}
      />

      <Field
        id="status_tertanggung"
        label="Status Tertanggung"
        type="text"
        hint="Isian teks bebas, sama seperti di layar lama — bukan daftar pilihan."
        error={errors.status_tertanggung?.message}
        {...register('status_tertanggung')}
      />

      <ComboField
        id="keterangan_penyebab_kerugian"
        label="Dokumen kolom ID"
        options={causesOfLoss.map((row) => row.nama)}
        hint={
          chosenCauseOfLoss
            ? `Kode penyebab kerugian: ${chosenCauseOfLoss}`
            : 'Pilih dari daftar, atau ketik keterangan lain bila memang dikehendaki.'
        }
        error={errors.keterangan_penyebab_kerugian?.message}
        {...register('keterangan_penyebab_kerugian')}
      />

      <ComboField
        id="keterangan_objek_dokumen"
        label="Objek Dokumen"
        options={objectDocuments.map((row) => row.nama)}
        hint={
          chosenObjectDocument
            ? `Kode objek dokumen: ${chosenObjectDocument}`
            : 'Pilih dari daftar, atau ketik keterangan lain bila memang dikehendaki.'
        }
        error={errors.keterangan_objek_dokumen?.message}
        {...register('keterangan_objek_dokumen')}
      />

      {/* `type="text"`, bukan `type="number"`, dan itu bukan kelalaian.

          Isiannya di Pega adalah teks biasa. Memakai `type="number"` akan membuat
          peramban MENOLAK ketikan yang tidak valid sebelum sampai ke kode, sehingga
          penjaga di skema tidak pernah berjalan dan pesannya tidak pernah terlihat
          siapa pun. Yang lebih buruk: peramban mengosongkan nilainya diam-diam saat
          ketikan sementara tidak valid.

          `inputMode="decimal"` tetap dipasang supaya papan ketik angka yang muncul di
          tablet surveyor (`D-12`), tanpa mengubah apa yang boleh diketik. */}
      <Field
        id="resiko"
        label="Resiko"
        type="text"
        inputMode="decimal"
        hint="Kosong dibaca sebagai nol, sama seperti di layar lama."
        error={errors.resiko?.message}
        {...register('resiko')}
      />

      {/* Lini bisnis adalah GRID, bukan satu isian. Di Pega ia
          `pyPageListProperty = TempDTDoc.DFT_BISNIS_ID` — satu rincian dokumen dapat
          punya aturan berbeda di tiap lini bisnis. */}
      <fieldset className="rounded-kontrol border border-slate-200 p-4">
        <legend className="px-1 text-sm font-medium text-slate-700">Lini Bisnis</legend>

        {isLoadingBusinessRules ? (
          <p className="text-sm text-slate-500">Memuat lini bisnis yang sudah dipilih…</p>
        ) : fields.length === 0 ? (
          <p className="text-sm text-slate-500">
            Belum ada lini bisnis. Dibiarkan kosong berarti rincian dokumen ini belum diminta
            pada lini bisnis mana pun.
          </p>
        ) : (
          <ul className="space-y-3">
            {fields.map((row, index) => {
              const businessName = describe(businesses, rows?.[index]?.id_bisnis)

              return (
                <li
                  key={row.id}
                  className="grid gap-2 sm:grid-cols-[1.5fr_1fr_1fr_auto] sm:items-end"
                >
                  <ComboField
                    id={`bisnis-${index}-id`}
                    label={`ID Bisnis baris ${index + 1}`}
                    options={businesses.map((item) => item.id)}
                    hint={businessName}
                    error={errors.bisnis?.[index]?.id_bisnis?.message}
                    {...register(`bisnis.${index}.id_bisnis` as const)}
                  />
                  <SelectField
                    id={`bisnis-${index}-wajib`}
                    label={`Status Wajib baris ${index + 1}`}
                    options={[
                      { value: MANDATORY_YES, label: 'Ya' },
                      { value: MANDATORY_NO, label: 'Tidak' },
                    ]}
                    error={errors.bisnis?.[index]?.status_wajib?.message}
                    {...register(`bisnis.${index}.status_wajib` as const)}
                  />
                  <Field
                    id={`bisnis-${index}-minimum`}
                    label={`Minimum Dokumen baris ${index + 1}`}
                    type="text"
                    inputMode="numeric"
                    error={errors.bisnis?.[index]?.minimum_dokumen?.message}
                    {...register(`bisnis.${index}.minimum_dokumen` as const)}
                  />
                  <Button
                    tone="halus"
                    onClick={() => remove(index)}
                    aria-label={`Hapus lini bisnis baris ${index + 1}`}
                  >
                    Hapus
                  </Button>
                </li>
              )
            })}
          </ul>
        )}

        <div className="mt-3">
          <Button
            tone="kedua"
            onClick={() =>
              append({ id_bisnis: '', status_wajib: MANDATORY_YES, minimum_dokumen: '1' })
            }
          >
            Tambah Bisnis
          </Button>
          {/* Daftar saran yang gagal dimuat TIDAK menghalangi apa pun: kodenya memang
              boleh diketik sendiri. Yang hilang hanya kenyamanan memilih. */}
          {businesses.length === 0 && (
            <p className="mt-2 text-sm text-slate-500">
              Daftar bisnis belum dapat dimuat, jadi tidak ada saran. Kode bisnis tetap dapat
              diketik sendiri.
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
        <Button type="submit" tone="utama" disabled={isSaving || isLoadingBusinessRules}>
          {isSaving ? 'Menyimpan…' : editMode ? 'Ubah' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}

/** Diekspor supaya layar dapat mengubah isian form menjadi boolean kontrak. */
export { MANDATORY_YES }
