import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef, useState, type FocusEvent } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Technician } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField, type SelectOption } from '@/components/SelectField'

import { useLookupEmployee, useSaveTechnician, useTechnicianList } from './api'

/**
 * Batas panjang isian; sama dengan konstanta di internal/masterpicteknik.
 *
 * Seluruhnya KEBUTUHAN BARU — layar Pega tidak membatasi panjang sama sekali. Yang di sini
 * menjawab pengguna tanpa perjalanan jaringan; yang di server adalah yang menegakkan.
 * Bila salah satu berubah, KEDUANYA wajib ikut berubah.
 */
const MAX_OPERATOR_ID_LENGTH = 64
const MAX_EMAIL_LENGTH = 100
const MAX_SUPERVISOR_LENGTH = 64
const MAX_CLAIM_COUNTER = 9999

/**
 * Pilihan Kelompok dan Bisnis, DISALIN dari Property rule Pega — bukan diturunkan dari
 * data yang kebetulan tampil.
 *
 *	Property/TEAM_GROUP_property.xml     pyStandardValue A · B · C
 *	Property/TYPE_BUSINESS_property.xml  pyStandardValue NONMBU · TRAVEL · PA · BONDING
 *
 * Urutannya mengikuti urutan di berkas itu, bukan diurutkan ulang menurut abjad, supaya
 * dropdown tampil sama dengan layar lama.
 *
 * Daftar yang sama ditegakkan server di `masterpicteknik.GroupCodes` dan `BusinessCodes`.
 * Bila Property rule berubah, KEDUA tempat wajib ikut berubah.
 */
const GROUP_CODES = ['A', 'B', 'C'] as const
const BUSINESS_CODES = ['NONMBU', 'TRAVEL', 'PA', 'BONDING'] as const

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Itu duplikasi yang DISENGAJA. Yang di sini menjawab pengguna tanpa perjalanan jaringan;
 * yang di sana adalah yang menegakkan — karena pemanggilan langsung ke API tidak melewati
 * layar ini sama sekali.
 *
 * Nama isiannya mengikuti LABEL layar Pega, bukan nama kolomnya:
 * `TYPE_BUSINESS` → Bisnis, `TEAM_GROUP` → Kelompok, `COUNTER_QUOTA` → Counter Klaim <1M.
 */
const schema = z.object({
  id_operator: z
    .string()
    .trim()
    .min(1, 'Username wajib diisi.')
    .max(MAX_OPERATOR_ID_LENGTH, `Username paling panjang ${MAX_OPERATOR_ID_LENGTH} karakter.`),
  // Email TIDAK wajib — Section Pega tidak memuat satu pun `pyRequired=true`, dan
  // kolomnya NULLABLE. Mewajibkannya membuat baris lama yang surelnya kosong tidak dapat
  // disunting sama sekali. Formatnya tetap diperiksa bila diisi.
  email: z
    .string()
    .trim()
    .max(MAX_EMAIL_LENGTH, `Email paling panjang ${MAX_EMAIL_LENGTH} karakter.`)
    // Sengaja longgar, sama dengan EmailPlausible di domain: satu-satunya cara membuktikan
    // alamat benar adalah mengirim surel ke sana, dan validasi yang terlalu ketat justru
    // menolak alamat yang sah.
    .refine(
      (value) => value === '' || /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(value),
      'Format email tidak benar.',
    ),
  // Kelompok dan Bisnis adalah PILIHAN, bukan teks bebas. Kosong tetap diterima: layar
  // Pega menyediakan `--Pilih--`, dan kolomnya NULLABLE.
  kelompok: z
    .string()
    .trim()
    .refine(
      (value) => value === '' || GROUP_CODES.some((code) => code === value.toUpperCase()),
      `Kelompok harus salah satu dari ${GROUP_CODES.join(', ')}.`,
    ),
  aktif: z.enum(['1', '0']),
  atasan: z
    .string()
    .trim()
    .max(MAX_SUPERVISOR_LENGTH, `Atasan paling panjang ${MAX_SUPERVISOR_LENGTH} karakter.`),
  bisnis: z
    .string()
    .trim()
    .refine(
      (value) => value === '' || BUSINESS_CODES.some((code) => code === value.toUpperCase()),
      `Bisnis harus salah satu dari ${BUSINESS_CODES.join(', ')}.`,
    ),
  // Angka diubah React Hook Form lewat `valueAsNumber`, BUKAN oleh `z.coerce`: pada Zod 4
  // tipe masukan `z.coerce.number()` adalah `unknown`, dan itu merusak keterkaitan tipe
  // antara skema dan form. Isian kosong menjadi NaN, dan `z.number()` menolaknya.
  counter_klaim_kurang_1m: z
    .number({ error: 'Counter Klaim <1M harus berupa angka.' })
    .int('Counter Klaim <1M harus bilangan bulat.')
    .min(0, 'Counter Klaim <1M tidak boleh negatif.')
    .max(MAX_CLAIM_COUNTER, `Counter Klaim <1M paling besar ${MAX_CLAIM_COUNTER}.`),
  counter_klaim_lebih_1m: z
    .number({ error: 'Counter Klaim >1M harus berupa angka.' })
    .int('Counter Klaim >1M harus bilangan bulat.')
    .min(0, 'Counter Klaim >1M tidak boleh negatif.')
    .max(MAX_CLAIM_COUNTER, `Counter Klaim >1M paling besar ${MAX_CLAIM_COUNTER}.`),
})

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  technician: Technician | null
  onClose: () => void
}

/**
 * Form tambah dan ubah Master PIC Teknik.
 *
 * # Isiannya SAMA PERSIS dengan Pega
 *
 * Urutan dan labelnya dibaca dari `Section/BrowseUserTeknis-Section.xml`, bukan dikarang:
 *
 *	16284 Username              → TempDcol.OPERATOR_ID   (16390)
 *	      Input Nama            → TempDcol.MCL_NAME      (17068) — hanya dibaca
 *	17839 Email                 → TempDcol.EMAIL         (17878)
 *	18319 Kelompok              → TempDcol.TEAM_GROUP    (18329) — dropdown
 *	19882 Status Aktif          → TempDcol.STS_AKTIF     (19800) — dropdown Ya/Tidak
 *	20426 Atasan                → TempDcol.ATASAN        (20503) — dropdown
 *	21180 Bisnis                → TempDcol.TYPE_BUSINESS (21059) — dropdown, --Pilih--
 *	21683 Counter Klaim <1M     → TempDcol.COUNTER_QUOTA (21807)
 *	22436 Counter Klaim >1M     → TempDcol.OLD_OPERATOR_ID (22351)
 *
 * Judulnya `pyTitle` pada baris 15873: **"Memperbaharui Data"**.
 *
 * # Yang TIDAK ada, karena tidak ada pula di Pega
 *
 * TOTAL_JOB tidak diisikan ke form mana pun, dan tidak ada tombol Cari tersendiri —
 * pencariannya berjalan saat isian Username ditinggalkan, persis seperti
 * `SetMstUserTeknisMstUser_act` yang terpicu tanpa tombol.
 */
export function TechnicianForm({ technician, onClose }: Props) {
  const save = useSaveTechnician()
  const lookup = useLookupEmployee()
  const list = useTechnicianList()
  const editing = technician !== null
  const firstField = useRef<HTMLInputElement | null>(null)

  // Nama dan nama atasan TIDAK ikut di dalam form state: keduanya tidak pernah dikirim ke
  // server. Mereka hanya ditampilkan, dan sumbernya pencarian direktori.
  const [name, setName] = useState(technician?.nama ?? '')
  const [supervisorName, setSupervisorName] = useState('')

  const rows = list.data?.pic_teknik ?? []

  const {
    register,
    handleSubmit,
    getValues,
    setValue,
    setError,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      id_operator: technician?.id_operator ?? '',
      email: technician?.email ?? '',
      kelompok: technician?.kelompok ?? '',
      aktif: technician?.aktif === false ? '0' : '1',
      atasan: technician?.atasan ?? '',
      bisnis: technician?.bisnis ?? '',
      counter_klaim_kurang_1m: technician?.counter_klaim_kurang_1m ?? 0,
      counter_klaim_lebih_1m: technician?.counter_klaim_lebih_1m ?? 0,
    },
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan ketik
  // harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    firstField.current?.focus()
  }, [])

  // Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di
  // kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya
  // berguna bila layar menyorotnya di tempat isiannya.
  useEffect(() => {
    if (!(save.error instanceof APIError)) return
    for (const [field, message] of Object.entries(save.error.violations())) {
      if (field in schema.shape) {
        setError(field as keyof FieldValues, { type: 'server', message })
      }
    }
  }, [save.error, setError])

  // ID yang tidak ketemu TIDAK lagi ditandai sebagai isian salah.
  //
  // Dulu ia dipasang lewat setError, sehingga isian memerah dan form tampak tidak dapat
  // disimpan. Itu menyesatkan: server kini menerimanya — sama seperti Pega, yang hanya
  // mengisi MCL_NAME apa adanya tanpa memeriksa hasil pencarian. Yang tersisa cukup
  // sebuah keterangan di bawah isian, bukan penolakan.
  const notFound =
    lookup.error instanceof APIError && lookup.error.violations()['id_operator'] !== undefined

  /**
   * Mencari pegawai lalu mengisikan hasilnya ke form.
   *
   * Atasan dan surel hanya diisi bila pengguna mengosongkannya. Atasan dari direktori
   * adalah atasan menurut struktur organisasi — belum tentu atasan penanganan klaim — dan
   * surel memang DIKETIK di sistem lama, bukan diturunkan. Server memakai aturan yang sama.
   */
  function searchDirectory() {
    const operatorID = getValues('id_operator').trim()
    if (operatorID === '') return

    lookup.mutate(operatorID, {
      onSuccess: (answer) => {
        setName(answer.pegawai.nama)
        setSupervisorName(answer.pegawai.nama_atasan)
        if (getValues('atasan').trim() === '') {
          setValue('atasan', answer.pegawai.atasan)
        }
        if (getValues('email').trim() === '' && answer.pegawai.email !== '') {
          setValue('email', answer.pegawai.email)
        }
      },
    })
  }

  // onBlur ikut dipisahkan, bukan hanya ref: isian Username memicu pencarian saat
  // ditinggalkan, dan menimpa onBlur milik React Hook Form akan mematikan penandaan
  // "sudah disentuh" yang dipakainya. Keduanya dirantai, tidak saling menggantikan.
  const {
    ref: refOperatorID,
    onBlur: onBlurOperatorID,
    ...remainingOperatorID
  } = register('id_operator')

  function handleOperatorIDBlur(event: FocusEvent<HTMLInputElement>) {
    void onBlurOperatorID(event)
    searchDirectory()
  }

  function send(values: FieldValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal.
    save.mutate({ ...values, aktif: values.aktif === '1', ubah: editing }, { onSuccess: onClose })
  }

  const busy = save.isPending || lookup.isPending

  /*
    Kelompok dan Bisnis memakai daftar TETAP dari Property rule Pega.

    Nilai baris yang sedang disunting tetap ikut dimasukkan walau di luar daftar — tanpa
    itu, membuka baris lama yang menyimpan nilai warisan akan mengosongkan isiannya
    diam-diam, dan menyimpan kembali justru menghapus nilai yang sebenarnya ada.

    ATASAN berbeda: `Property/ATASAN_property.xml` adalah Text biasa tanpa daftar nilai.
    Yang membatasinya di layar Pega adalah autocomplete atas daftar operator, dan di sini
    daftar itu diambil dari isi master yang sedang tampil.
  */
  const groupOptions = optionsFrom(GROUP_CODES, technician?.kelompok)
  const businessOptions = optionsFrom(BUSINESS_CODES, technician?.bisnis)
  const supervisorOptions = optionsFrom(
    [...new Set(rows.map((t) => t.id_operator.trim()).filter((id) => id !== ''))]
      .filter((id) => id !== technician?.id_operator)
      .sort(),
    technician?.atasan,
  )

  return (
    /*
      Panel ini muncul di atas tabel, bukan sebagai dialog melayang: pengguna sering perlu
      melihat petugas lain yang sudah ada untuk memastikan kelompok dan atasan yang
      diisinya masuk akal — dan dialog yang menutup layar menyembunyikan jawabannya.
    */
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={editing ? 'Memperbaharui Data' : 'Memperbaharui Data'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        {/* Judulnya diambil dari pyTitle layar Pega apa adanya. */}
        <h3 className="text-base font-semibold text-slate-900">
          {editing ? 'Memperbaharui Data' : 'Memperbaharui Data'}
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          {editing
            ? 'Username tetap, karena data klaim menyimpannya. Input Nama disegarkan dari direktori pegawai setiap kali disimpan.'
            : 'Isi Username lalu pindah ke isian berikutnya. Input Nama diambil dari direktori pegawai, tidak diketik.'}
        </p>
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}
        {lookup.isError && !notFound && <LookupErrorMessage error={lookup.error} />}

        <div className="grid gap-5 sm:grid-cols-2">
          {/* 1. Username */}
          {editing ? (
            <div>
              <span className="block text-sm font-medium text-slate-700">Username</span>
              {/*
                Digambar sebagai kotak mati, bukan input ber-`disabled`. Input yang
                dinonaktifkan tetap terlihat seperti isian dan mengundang pengguna
                mengkliknya; kotak ini jelas bukan tempat mengetik.
              */}
              <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 text-sm text-slate-500">
                {technician.id_operator}
              </p>
              <p className="mt-1.5 text-xs text-slate-500">
                Tidak dapat diubah — setiap klaim menyimpan Username ini.
              </p>
              <input type="hidden" {...remainingOperatorID} ref={refOperatorID} />
            </div>
          ) : (
            <Field
              id="id_operator"
              label="Username"
              placeholder="Contoh: PICTEKNIK05"
              maxLength={MAX_OPERATOR_ID_LENGTH}
              autoComplete="off"
              hint={
                notFound
                  ? 'Namanya tidak ketemu di master maupun direktori pegawai. Tetap dapat disimpan; kolom Input Nama akan kosong.'
                  : 'Nama petugas dicari setelah isian ini ditinggalkan.'
              }
              error={errors.id_operator?.message}
              disabled={busy}
              {...remainingOperatorID}
              ref={(element) => {
                refOperatorID(element)
                firstField.current = element
              }}
              onBlur={handleOperatorIDBlur}
            />
          )}

          {/* 2. Input Nama — hanya dibaca, hasil pencarian direktori */}
          <div>
            <span className="block text-sm font-medium text-slate-700">Input Nama</span>
            <p className="mt-1.5 flex min-h-[2.75rem] items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 text-sm text-slate-700">
              {lookup.isPending ? (
                <span className="text-slate-400">Mencari…</span>
              ) : name === '' ? (
                <span className="text-slate-400">——</span>
              ) : (
                <span className="font-medium text-slate-900">{name}</span>
              )}
            </p>
            <p className="mt-1.5 text-xs text-slate-500">
              Berasal dari direktori pegawai; tidak dapat diketik.
            </p>
          </div>

          {/* 3. Email */}
          <Field
            id="email"
            label="Email"
            type="email"
            placeholder="nama@sinarmas.co.id"
            maxLength={MAX_EMAIL_LENGTH}
            autoComplete="off"
            error={errors.email?.message}
            disabled={busy}
            {...register('email')}
          />

          {/* 4. Kelompok */}
          <SelectField
            id="kelompok"
            label="Kelompok"
            options={groupOptions}
            emptyText="--Pilih--"
            error={errors.kelompok?.message}
            disabled={busy}
            {...register('kelompok')}
          />

          {/* 5. Status Aktif */}
          <SelectField
            id="aktif"
            label="Status Aktif"
            options={[
              { value: '1', label: 'Ya' },
              { value: '0', label: 'Tidak' },
            ]}
            emptyText="--Pilih--"
            error={errors.aktif?.message}
            disabled={busy}
            {...register('aktif')}
          />

          {/* 6. Atasan */}
          <div>
            <SelectField
              id="atasan"
              label="Atasan"
              options={supervisorOptions}
              emptyText="--Pilih--"
              error={errors.atasan?.message}
              disabled={busy}
              {...register('atasan')}
            />
            {supervisorName !== '' && (
              <p className="mt-1.5 text-xs text-slate-500">
                Menurut direktori:{' '}
                <span className="font-medium text-slate-700">{supervisorName}</span>
              </p>
            )}
          </div>

          {/* 7. Bisnis */}
          <SelectField
            id="bisnis"
            label="Bisnis"
            options={businessOptions}
            emptyText="--Pilih--"
            error={errors.bisnis?.message}
            disabled={busy}
            {...register('bisnis')}
          />

          {/* 8. Counter Klaim <1M */}
          <Field
            id="counter_klaim_kurang_1m"
            label="Counter Klaim <1M"
            type="number"
            min={0}
            max={MAX_CLAIM_COUNTER}
            hint="Banyaknya klaim bernilai di bawah Rp 1 Miliar yang sudah ditangani."
            error={errors.counter_klaim_kurang_1m?.message}
            disabled={busy}
            {...register('counter_klaim_kurang_1m', { valueAsNumber: true })}
          />

          {/* 9. Counter Klaim >1M */}
          <Field
            id="counter_klaim_lebih_1m"
            label="Counter Klaim >1M"
            type="number"
            min={0}
            max={MAX_CLAIM_COUNTER}
            hint="Banyaknya klaim bernilai di atas Rp 1 Miliar yang sudah ditangani."
            error={errors.counter_klaim_lebih_1m?.message}
            disabled={busy}
            {...register('counter_klaim_lebih_1m', { valueAsNumber: true })}
          />
        </div>

        <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
          <Button type="submit" tone="utama" disabled={busy}>
            {save.isPending ? 'Menyimpan…' : 'Simpan'}
          </Button>
          <Button tone="halus" onClick={onClose} disabled={busy}>
            Batal
          </Button>
        </div>
      </div>
    </form>
  )
}

/**
 * optionsFrom menyusun pilihan dropdown, dengan nilai baris yang sedang disunting selalu
 * ikut walau berada di luar daftar.
 *
 * Urutan daftarnya DIPERTAHANKAN apa adanya — untuk Kelompok dan Bisnis ia urutan
 * `pyStandardValue` di Property rule, dan mengurutkannya ulang membuat dropdown tampil
 * berbeda dari layar lama. Nilai warisan yang di luar daftar ditaruh di belakang.
 */
function optionsFrom(values: readonly string[], current?: string): SelectOption[] {
  const option = values.map((value) => ({ value, label: value }))
  const currentClean = (current ?? '').trim()

  if (currentClean !== '' && !values.some((value) => value === currentClean)) {
    option.push({ value: currentClean, label: currentClean })
  }
  return option
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Galat pencarian dipisahkan dari galat simpan.
 *
 * Keduanya terjadi pada saat yang berbeda dan menuntut tindakan yang berbeda: yang satu
 * saat mengisi Username, yang lain saat menekan Simpan.
 */
function LookupErrorMessage({ error }: { error: unknown }) {
  const message = parseDirectory(error)
  if (message === null) return null
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

function SaveErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Perubahan belum tersimpan. Periksa koneksi lalu coba lagi."
        tone="gangguan"
      />
    )
  }

  if (!(error instanceof APIError)) {
    return (
      <ErrorMessage
        title="Gagal menyimpan"
        description="Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi."
        tone="gangguan"
      />
    )
  }

  const message = parseSave(error)
  if (message === null) return null
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

/**
 * Galat pencarian direktori.
 *
 * Yang TIDAK dikenali jatuh ke "Pencarian gagal" — dan itu hanya benar di jalur ini.
 * Jalur simpan memakai `parseSave`, yang jatuh ke "Gagal menyimpan": galat penyimpanan
 * yang dijawab "pencarian gagal" akan menyuruh pengguna memperbaiki hal yang salah.
 */
function parseDirectory(error: unknown): MessageContent | null {
  return parseShared(error) ?? fallbackFor(error, 'Pencarian gagal')
}

/**
 * Kode galat yang berarti sama di kedua jalur: gangguan direktori, portal, dan validasi.
 *
 * Mengembalikan null bila kodenya bukan salah satu di antaranya, supaya pemanggil
 * menentukan sendiri pesan penutupnya.
 */
function parseShared(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Permintaan belum dapat dijalankan. Periksa koneksi lalu coba lagi.',
      tone: 'gangguan',
    }
  }
  if (!(error instanceof APIError)) return null

  switch (error.kode) {
    case ErrorCode.directoryUnconfigured:
      return {
        title: 'Layanan pencarian pegawai belum terdaftar',
        description:
          'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk mendaftarkan alamat layanannya bagi entitas ini.',
        tone: 'gangguan',
      }

    case ErrorCode.directoryUnreachable:
      return {
        title: 'Direktori pegawai sedang tidak dapat dihubungi',
        description: 'Ini gangguan sementara di luar aplikasi. Coba beberapa saat lagi.',
        tone: 'gangguan',
      }

    case ErrorCode.validationFailed:
      // Bila detailnya ada, isiannya sudah disorot di tempatnya; kotak pesan hanya akan
      // mengulang hal yang sama.
      return Object.keys(error.violations()).length > 0
        ? null
        : { title: 'Isian belum benar', description: error.message, tone: 'penolakan' }

    case ErrorCode.portalNotStated:
    case ErrorCode.portalUnknown:
      return {
        title: 'Portal entitas belum dipilih',
        description: 'Pilih portal entitas di bilah atas halaman, lalu coba lagi.',
        tone: 'penolakan',
      }

    default:
      return null
  }
}

/** Pesan penutup bila kode galatnya tidak dikenali sama sekali. */
function fallbackFor(error: unknown, title: string): MessageContent | null {
  if (!(error instanceof APIError)) return null
  return { title, description: error.message, tone: 'gangguan' }
}

function parseSave(error: APIError): MessageContent | null {
  switch (error.kode) {
    case ErrorCode.technicianExists:
      return {
        title: 'Username sudah terdaftar',
        description:
          'Petugas dengan Username itu sudah ada di entitas ini. Buka datanya lalu ubah di sana.',
        tone: 'penolakan',
      }

    case ErrorCode.technicianNotFound:
      return {
        title: 'PIC teknik tidak ditemukan',
        description: 'Baris ini mungkin sudah diubah petugas lain. Muat ulang daftarnya.',
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
      // Kode bersama — gangguan direktori, portal, validasi — dipetakan sama di kedua
      // jalur; sisanya ditutup pesan SIMPAN, bukan pesan pencarian.
      return parseShared(error) ?? fallbackFor(error, 'Gagal menyimpan')
  }
}
