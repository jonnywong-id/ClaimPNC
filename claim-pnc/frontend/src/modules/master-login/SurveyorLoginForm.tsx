import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, SurveyorLoginErrorCode, type SurveyorLogin } from '@/api/types'
import { Field } from '@/components/Field'
import { TextAreaField } from '@/components/TextAreaField'
import { MasterFormActions, MasterFormFrame } from '@/components/masterform/BoxForm'
import { type CodeMessages, notFoundMessage, portalMessages, saveErrorMessage, validationMessage, violationsOf } from '@/components/masterform/saveErrorMessage'

/**
 * Batas panjang harus sama dengan konstanta di backend.
 *
 * Keempatnya ASUMSI, bukan angka dari DDL: `POOLDATA.MST_LOGIN_SURVEYOR` tidak ada DDL-nya
 * di export (`R-08`), dan layar lamanya tidak memasang satu pun `pyMaxLength` pada kelima
 * isiannya.
 *
 * Diperiksa di dua tempat dengan sengaja: di sini supaya pengguna tahu sebelum mengirim,
 * dan di server karena API dapat ditembak tanpa melewati layar ini. Server tetap yang
 * berwenang — pemeriksaan di sini hanya kenyamanan.
 *
 * Bila MAX_NAME_LENGTH berubah, `internal/masterlogin/masterlogin.go` harus ikut berubah.
 * Uji `TestMaxNameLengthMatchesFrontendForm` yang menjaganya.
 */
const MAX_NAME_LENGTH = 100
const MAX_EMAIL_LENGTH = 100
const MAX_PHONE_LENGTH = 50
const MAX_ADDRESS_LENGTH = 250

/**
 * deriveLogin menghitung LOGIN dari Nama — SAMA PERSIS dengan server.
 *
 * Rujukannya `Activity/SetLoginSurveyor_act-Act.xml`:
 *
 *	local.login := @replaceAll(@replaceAll(@replaceAll(@replaceAll(
 *	                 TempLoginSurvey.SurveyName," ",""),".",""),",",""),"-","")
 *
 * # Ia dihitung di sini untuk DITAMPILKAN, bukan untuk DIKIRIM
 *
 * Perbedaan itu menentukan, dan ia sengaja berbeda dari Pega. Di sana nilainya dihitung di
 * layar lalu dikirim kembali sebagai isian biasa, sehingga permintaan yang tidak datang
 * dari layar dapat mengirim LOGIN apa pun — dan LOGIN adalah kunci barisnya.
 *
 * Di sini badan permintaan TIDAK memuatnya sama sekali; server menurunkannya sendiri. Yang
 * dikerjakan fungsi ini hanyalah memperlihatkan hasilnya sementara pengguna mengetik,
 * persis seperti layar lama yang memperbaruinya pada setiap perubahan isian Nama.
 *
 * Karena keduanya menghitung hal yang sama, keduanya harus berubah bersamaan. Bila salah
 * satu berubah sendirian, pengguna melihat login yang bukan yang tersimpan.
 */
export function deriveLogin(name: string): string {
  return name.trim().replaceAll(/[ .,-]/g, '')
}

const schema = z.object({
  nama: z
    .string()
    .trim()
    .min(1, 'Nama wajib diisi.')
    .max(MAX_NAME_LENGTH, `Nama paling panjang ${MAX_NAME_LENGTH} karakter.`)
    .refine(
      (value) => deriveLogin(value) !== '',
      'Nama harus memuat setidaknya satu huruf atau angka. ' +
        'Spasi, titik, koma, dan tanda hubung dibuang saat Login dibentuk.',
    ),
  email: z
    .string()
    .trim()
    .min(1, 'Email wajib diisi.')
    .max(MAX_EMAIL_LENGTH, `Email paling panjang ${MAX_EMAIL_LENGTH} karakter.`),
  telp: z
    .string()
    .trim()
    .min(1, 'Telp wajib diisi.')
    .max(MAX_PHONE_LENGTH, `Telp paling panjang ${MAX_PHONE_LENGTH} karakter.`),
  alamat: z
    .string()
    .trim()
    .max(MAX_ADDRESS_LENGTH, `Alamat paling panjang ${MAX_ADDRESS_LENGTH} karakter.`),
})

export type SurveyorLoginFormValues = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  editing: SurveyorLogin | null
  isSaving: boolean
  error: unknown
  onSave: (values: SurveyorLoginFormValues) => void
  onCancel: () => void
}

/** Pesan galat penyimpanan per kode; yang tidak dikenal jatuh ke pesan galat sistem. */
const saveMessages: CodeMessages = {
  // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya akan
  // mengulang hal yang sama.
  [ErrorCode.validationFailed]: validationMessage,
  // Ia sudah disorot pada isiannya, tetapi TETAP ditampilkan sebagai kotak pesan —
  // dan itu berbeda dari galat validasi biasa.
  //
  // Alasannya: yang bentrok bukan Nama melainkan LOGIN yang DITURUNKAN darinya,
  // sehingga pengguna yang mencari nama itu di daftar bisa saja tidak menemukannya.
  // Sorotan di bawah isian tidak cukup menjelaskan ke mana ia harus mencari.
  [SurveyorLoginErrorCode.loginTaken]: {
    title: 'Login itu sudah terdaftar',
    description:
      'Login dibentuk dari Nama dengan membuang spasi, titik, koma, dan tanda ' +
      'hubung — sehingga dua nama yang terlihat berbeda dapat menghasilkan login ' +
      'yang sama. Cari login tersebut di daftar, atau pakai nama lain.',
    tone: 'penolakan',
  },
  [SurveyorLoginErrorCode.nameLocked]: {
    title: 'Nama tidak dapat diubah',
    description:
      'Login dibentuk dari Nama, dan mengubahnya akan memindahkan kunci baris ini. ' +
      'Tutup form ini dan muat ulang daftarnya.',
    tone: 'penolakan',
  },
  [ErrorCode.notFound]: notFoundMessage,
  ...portalMessages,
}

/** Isian yang dapat disorot server. Dipakai menyaring pelanggaran yang tidak dikenal. */
const FIELDS = ['nama', 'email', 'telp', 'alamat'] as const

/**
 * SurveyorLoginForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `Activity/SetLoginSurveyorValue_act` memuat
 * baris ke form yang sama lalu menandainya "Update"
 * (`TempLoginSurvey.pyLabel := "Update"`).
 *
 * # LIMA isian di layar Pega, EMPAT yang dapat diketik
 *
 * Yang kelima — Login — diturunkan dari Nama dan ditampilkan baca-saja. Di Pega ia memang
 * kontrol tersendiri yang nilainya ditimpa pada setiap perubahan Nama; di sini ia
 * keterangan, karena badan permintaan tidak memuatnya sama sekali.
 *
 * # Nama TERKUNCI saat menyunting
 *
 * Dibaca dari `pyDisabledWhen = TempLoginSurvey.pyLabel='Update'` pada kontrol Nama.
 * Alasannya bukan kerapian: Login diturunkan dari Nama dan Login adalah kunci baris, jadi
 * Nama yang berubah akan memindahkan kunci sementara pernyataan simpannya masih menyaring
 * kunci lama.
 *
 * Penguncian di sini adalah kenyamanan tampilan; **server ikut menolaknya**, karena
 * permintaan yang tidak datang dari layar ini tidak tersentuh olehnya.
 *
 * # Dua keterangan baca-saja yang TIDAK ada di layar Pega
 *
 * Status Login dan Login Leader ditulis sistem lama tetapi tidak pernah digambar. Keduanya
 * ditampilkan di sini karena nilainya menentukan peran dan tim seseorang — dan
 * menyembunyikan hal yang tersimpan tidak membuatnya tidak tersimpan. Keduanya tidak dapat
 * diubah dari layar mana pun, dan itu dinyatakan apa adanya.
 */
export function SurveyorLoginForm({ editing, isSaving, error, onSave, onCancel }: Readonly<Props>) {
  const editMode = editing !== null

  const {
    register,
    handleSubmit,
    reset,
    setError,
    control,
    formState: { errors },
  } = useForm<SurveyorLoginFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama: editing?.nama ?? '',
      email: editing?.email ?? '',
      telp: editing?.telp ?? '',
      alamat: editing?.alamat ?? '',
    },
  })

  // Login diperlihatkan sementara pengguna mengetik, meniru aksi `refresh` bereven `change`
  // pada kontrol Nama di layar lama. Yang dipantau hanya isian Nama — memantau seluruh form
  // akan menggambar ulang keterangan ini pada setiap ketikan di isian mana pun.
  const typedName = useWatch({ control, name: 'nama' })
  const previewLogin = deriveLogin(typedName ?? '')

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu —
  // misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset({
      nama: editing?.nama ?? '',
      email: editing?.email ?? '',
      telp: editing?.telp ?? '',
      alamat: editing?.alamat ?? '',
    })
  }, [editing, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan
  // itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if ((FIELDS as readonly string[]).includes(column)) {
        setError(column as (typeof FIELDS)[number], { type: 'server', message })
      }
    }
  }, [error, setError])

  const message = saveErrorMessage(error, saveMessages)
  const title = editMode ? 'Ubah Login Surveyor' : 'Tambah Login Surveyor'

  // Isi keterangan Login: kunci yang tersimpan, pratinjau, atau petunjuk.
  let loginContent: ReactNode
  if (editMode) {
    loginContent = (
      <>
        <span className="font-medium">{editing.login}</span>
        <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
      </>
    )
  } else if (previewLogin === '') {
    loginContent = <span className="text-slate-500">Terisi otomatis setelah Nama diketik.</span>
  } else {
    loginContent = <span className="font-medium">{previewLogin}</span>
  }

  return (
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={message}>
      <Field
        id="nama"
        label="Nama"
        type="text"
        autoFocus={!editMode}
        maxLength={MAX_NAME_LENGTH}
        disabled={editMode}
        hint={
          editMode
            ? 'Tidak dapat diubah — Login dibentuk dari Nama, dan Login adalah kunci baris ini.'
            : 'Login akan dibentuk otomatis dari Nama.'
        }
        error={errors.nama?.message}
        {...register('nama')}
      />

      {/* Login BACA-SAJA.

          Di Pega ia kontrol tersendiri yang nilainya ditimpa pada setiap perubahan Nama; di
          sini ia keterangan, karena badan permintaan tidak memuatnya sama sekali — server
          yang menurunkannya. Yang ditampilkan adalah hasil perhitungan yang sama persis. */}
      <div>
        <span className="block text-sm font-medium text-slate-700">Login</span>
        <p className="mt-1.5 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">
          {loginContent}
        </p>
        {!editMode && (
          <p className="mt-1 text-xs text-slate-500">
            Dibentuk dari Nama dengan membuang spasi, titik, koma, dan tanda hubung.
          </p>
        )}
      </div>

      <Field
        id="email"
        label="Email"
        type="text"
        autoFocus={editMode}
        maxLength={MAX_EMAIL_LENGTH}
        error={errors.email?.message}
        {...register('email')}
      />

      <Field
        id="telp"
        label="Telp"
        type="text"
        maxLength={MAX_PHONE_LENGTH}
        error={errors.telp?.message}
        {...register('telp')}
      />

      <TextAreaField
        id="alamat"
        label="Alamat"
        rows={3}
        maxLength={MAX_ADDRESS_LENGTH}
        hint="Tidak wajib diisi."
        error={errors.alamat?.message}
        {...register('alamat')}
      />

      {/* Dua kolom yang tersimpan tetapi tidak dapat diubah dari layar mana pun.

          Keduanya tidak pernah digambar di layar Pega. Ditampilkan di sini karena nilainya
          menentukan peran dan tim seseorang — dan pengguna yang tidak melihatnya tidak punya
          cara mengetahui mengapa sebuah baris tampak tidak bertim. */}
      {editMode && (
        <div className="grid gap-4 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-3 sm:grid-cols-2">
          <div>
            <span className="block text-xs font-medium text-slate-500">Status Login</span>
            <p className="mt-0.5 text-sm text-slate-700">{editing.status_login || '—'}</p>
          </div>
          <div>
            <span className="block text-xs font-medium text-slate-500">Login Leader</span>
            <p className="mt-0.5 text-sm text-slate-700">
              {editing.login_leader || (
                <span className="text-slate-500">tidak bertaut ke tim mana pun</span>
              )}
            </p>
          </div>
          <p className="text-xs text-slate-500 sm:col-span-2">
            Keduanya diisi sistem dan tidak dapat diubah dari layar ini — sama seperti
            aplikasi lama, yang juga tidak pernah menampilkannya.
          </p>
        </div>
      )}

      {/* Akibat menyimpan dinyatakan di muka, bukan ditemukan setelah tombol ditekan. */}
      {!editMode && (
        <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          Menyimpan mencatat login surveyor di master ini. <strong>Akun aplikasinya belum
          diterbitkan</strong> — surveyor yang baru ditambahkan belum dapat masuk sampai
          sistem identitas tersedia.
        </p>
      )}

      <MasterFormActions isSaving={isSaving} onCancel={onCancel} />
    </MasterFormFrame>
  )
}
