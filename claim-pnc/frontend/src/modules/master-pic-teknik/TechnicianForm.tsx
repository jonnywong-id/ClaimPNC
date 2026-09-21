import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef, useState, type FocusEvent } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Technician } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SearchIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'

import { useLookupEmployee, useSaveTechnician } from './api'

/**
 * Batas panjang isian; sama dengan konstanta di internal/masterpicteknik.
 *
 * Seluruhnya KEBUTUHAN BARU — layar Pega tidak membatasi panjang sama sekali. Yang di sini
 * menjawab pengguna tanpa perjalanan jaringan; yang di server adalah yang menegakkan.
 * Bila salah satu berubah, KEDUANYA wajib ikut berubah.
 */
const MAX_OPERATOR_ID_LENGTH = 64
const MAX_EMAIL_LENGTH = 100
const MAX_GROUP_LENGTH = 50
const MAX_SUPERVISOR_LENGTH = 64
const MAX_BUSINESS_LINE_LENGTH = 50
const MAX_QUOTA = 9999

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Itu duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan langsung
 * ke API tidak melewati layar ini sama sekali.
 *
 * Keberadaan pegawai di direktori TIDAK diperiksa di sini: ia menuntut memanggil layanan
 * luar, dan jawabannya dapat berubah antara saat form dibuka dan saat Simpan ditekan.
 * Penegakannya ada di server, dan galatnya disorot pada kolom id_operator.
 */
const schema = z.object({
  id_operator: z
    .string()
    .trim()
    .min(1, 'ID operator wajib diisi.')
    .max(MAX_OPERATOR_ID_LENGTH, `ID operator paling panjang ${MAX_OPERATOR_ID_LENGTH} karakter.`),
  email: z
    .string()
    .trim()
    .min(1, 'Email wajib diisi.')
    .max(MAX_EMAIL_LENGTH, `Email paling panjang ${MAX_EMAIL_LENGTH} karakter.`)
    // Sengaja longgar, sama dengan EmailPlausible di domain: satu-satunya cara
    // membuktikan alamat benar adalah mengirim surel ke sana, dan validasi yang terlalu
    // ketat justru menolak alamat yang sah.
    .regex(/^[^@\s]+@[^@\s]+\.[^@\s]+$/, 'Format email tidak benar.'),
  lini_bisnis: z
    .string()
    .trim()
    .max(MAX_BUSINESS_LINE_LENGTH, `Lini bisnis paling panjang ${MAX_BUSINESS_LINE_LENGTH} karakter.`),
  grup: z.string().trim().max(MAX_GROUP_LENGTH, `Grup paling panjang ${MAX_GROUP_LENGTH} karakter.`),
  atasan: z
    .string()
    .trim()
    .max(MAX_SUPERVISOR_LENGTH, `Atasan paling panjang ${MAX_SUPERVISOR_LENGTH} karakter.`),
  // Angka diubah React Hook Form lewat `valueAsNumber` saat register, BUKAN oleh
  // `z.coerce` di sini.
  //
  // Alasannya bukan selera: pada Zod 4 tipe masukan `z.coerce.number()` adalah `unknown`,
  // dan itu merusak keterkaitan tipe antara skema dan form — kesalahan nama isian tidak
  // lagi tertangkap pemeriksa tipe. Membiarkan form yang mengubahnya menjaga jaring
  // pengaman itu tetap utuh.
  //
  // Isian kosong menjadi NaN, dan `z.number()` menolaknya — itulah pesan "harus berupa
  // angka" di bawah.
  kuota: z
    .number({ error: 'Kuota harus berupa angka.' })
    .int('Kuota harus bilangan bulat.')
    .min(0, 'Kuota tidak boleh negatif.')
    .max(MAX_QUOTA, `Kuota paling besar ${MAX_QUOTA}.`),
  kuota_luar: z
    .number({ error: 'Kuota sistem lain harus berupa angka.' })
    .int('Kuota sistem lain harus bilangan bulat.')
    .min(0, 'Kuota sistem lain tidak boleh negatif.')
    .max(MAX_QUOTA, `Kuota sistem lain paling besar ${MAX_QUOTA}.`),
  aktif: z.enum(['1', '0']),
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
 * Meniru `Section/BrowseUserTeknis-Section.xml` beserta pembagian isiannya:
 *
 * | Isian | Di Pega | Di sini |
 * |---|---|---|
 * | ID Operator | diketik, memicu pencarian | sama, dengan tombol Cari yang terlihat |
 * | Nama | hasil pencarian, tidak diketik | sama, digambar sebagai kotak mati |
 * | Beban kerja | `pyReadOnly=true` | sama, hanya tampil saat mengubah |
 * | Kuota, Kuota sistem lain | dapat diisi | sama |
 * | Email, Lini bisnis, Grup, Atasan, Status | dapat diisi | sama |
 * | Grup panel | tidak ada di form | tidak ada |
 *
 * Yang sengaja dibuat berbeda: pencariannya punya tombol dan hasilnya terlihat sebelum
 * Simpan ditekan. Di Pega ia berjalan diam-diam, dan kegagalannya baru muncul sebagai
 * penolakan setelah seluruh form diisi.
 */
export function TechnicianForm({ technician, onClose }: Props) {
  const save = useSaveTechnician()
  const lookup = useLookupEmployee()
  const editing = technician !== null
  const firstField = useRef<HTMLInputElement | null>(null)

  // Nama dan nama atasan TIDAK ikut di dalam form state: keduanya tidak pernah dikirim ke
  // server. Mereka hanya ditampilkan, dan sumbernya pencarian direktori.
  const [name, setName] = useState(technician?.nama ?? '')
  const [supervisorName, setSupervisorName] = useState('')

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
      lini_bisnis: technician?.lini_bisnis ?? '',
      grup: technician?.grup ?? '',
      atasan: technician?.atasan ?? '',
      kuota: technician?.kuota ?? 0,
      kuota_luar: technician?.kuota_luar ?? 0,
      aktif: technician?.aktif === false ? '0' : '1',
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

  // Pencarian yang gagal karena ID tidak terdaftar juga disorot pada kolom ID.
  useEffect(() => {
    if (!(lookup.error instanceof APIError)) return
    const violation = lookup.error.violations()['id_operator']
    if (violation) setError('id_operator', { type: 'server', message: violation })
  }, [lookup.error, setError])

  /**
   * Mencari pegawai lalu mengisikan hasilnya ke form.
   *
   * Atasan hanya diisi bila pengguna mengosongkannya — usulan direktori adalah atasan
   * menurut struktur organisasi, dan itu belum tentu atasan penanganan klaim. Server
   * memakai aturan yang sama persis saat menyimpan.
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
        // Surel diusulkan hanya bila kosong. Ia DIKETIK di sistem lama — pencarian di sana
        // menyalin nama dan id saja — sehingga surel milik master ini, bukan milik
        // direktori, dan tidak boleh menimpa yang sudah diisi petugas.
        if (getValues('email').trim() === '' && answer.pegawai.email !== '') {
          setValue('email', answer.pegawai.email)
        }
      },
    })
  }

  // onBlur ikut dipisahkan, bukan hanya ref: isian ID memicu pencarian saat ditinggalkan,
  // dan menimpa onBlur milik React Hook Form akan mematikan penandaan "sudah disentuh"
  // yang dipakainya. Keduanya dirantai di bawah, tidak saling menggantikan.
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
    save.mutate(
      {
        ...values,
        aktif: values.aktif === '1',
        ubah: editing,
      },
      { onSuccess: onClose },
    )
  }

  const busy = save.isPending || lookup.isPending

  return (
    /*
      Panel ini muncul di atas tabel, bukan sebagai dialog melayang.

      Alasannya praktis: pengguna sering perlu melihat petugas lain yang sudah ada untuk
      memastikan grup dan atasan yang diisinya masuk akal — dan dialog yang menutup layar
      justru menyembunyikan jawabannya.
    */
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={editing ? 'Ubah PIC teknik' : 'Tambah PIC teknik'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {editing ? 'Ubah PIC Teknik' : 'Tambah PIC Teknik'}
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          {editing
            ? 'ID operator tetap, karena data klaim menyimpannya. Nama disegarkan dari direktori setiap kali disimpan.'
            : 'Isi ID operator lalu tekan Cari. Nama diambil dari direktori pegawai, tidak diketik.'}
        </p>
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}
        {lookup.isError && <LookupErrorMessage error={lookup.error} />}

        <div className="grid gap-5 sm:grid-cols-2">
          {editing ? (
            <div>
              <span className="block text-sm font-medium text-slate-700">ID Operator</span>
              {/*
                Digambar sebagai kotak mati, bukan input ber-`disabled`. Input yang
                dinonaktifkan tetap terlihat seperti isian dan mengundang pengguna
                mengkliknya; kotak ini jelas bukan tempat mengetik.
              */}
              <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 font-mono text-sm text-slate-500">
                {technician.id_operator}
              </p>
              <p className="mt-1.5 text-xs text-slate-500">
                Tidak dapat diubah — setiap klaim menyimpan ID ini.
              </p>
              <input type="hidden" {...remainingOperatorID} ref={refOperatorID} />
            </div>
          ) : (
            <div>
              <div className="flex items-end gap-2">
                <div className="flex-1">
                  <Field
                    id="id_operator"
                    label="ID Operator"
                    placeholder="Contoh: PICTEKNIK05"
                    maxLength={MAX_OPERATOR_ID_LENGTH}
                    autoComplete="off"
                    icon={<SearchIcon className="h-4 w-4" />}
                    error={errors.id_operator?.message}
                    disabled={busy}
                    {...remainingOperatorID}
                    ref={(element) => {
                      refOperatorID(element)
                      firstField.current = element
                    }}
                    // Pencarian juga berjalan saat pengguna meninggalkan kolom, meniru
                    // sistem lama yang memicunya tanpa tombol. Tombolnya tetap ada supaya
                    // tindakannya terlihat — dan supaya dapat diulang tanpa mengetik ulang.
                    onBlur={handleOperatorIDBlur}
                  />
                </div>
                <Button
                  tone="kedua"
                  onClick={searchDirectory}
                  disabled={busy}
                  className="mb-[1.5rem] shrink-0"
                  aria-label="Cari pegawai di direktori"
                >
                  {lookup.isPending ? 'Mencari…' : 'Cari'}
                </Button>
              </div>
            </div>
          )}

          <div>
            <span className="block text-sm font-medium text-slate-700">Nama</span>
            <p className="mt-1.5 flex min-h-[2.75rem] items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 text-sm text-slate-700">
              {name === '' ? (
                <span className="text-slate-400">Belum dicari</span>
              ) : (
                <span className="font-medium text-slate-900">{name}</span>
              )}
            </p>
            <p className="mt-1.5 text-xs text-slate-500">
              Berasal dari direktori pegawai; tidak dapat diketik.
            </p>
          </div>

          <Field
            id="email"
            label="Email"
            type="email"
            placeholder="nama@sinarmas.co.id"
            maxLength={MAX_EMAIL_LENGTH}
            autoComplete="off"
            hint="Tujuan pemberitahuan penugasan untuk petugas ini."
            error={errors.email?.message}
            disabled={busy}
            {...register('email')}
          />

          <Field
            id="lini_bisnis"
            label="Lini Bisnis"
            placeholder="Contoh: NONMBU"
            maxLength={MAX_BUSINESS_LINE_LENGTH}
            autoComplete="off"
            hint="Teks bebas — boleh dikosongkan."
            error={errors.lini_bisnis?.message}
            disabled={busy}
            {...register('lini_bisnis')}
          />

          <Field
            id="grup"
            label="Grup"
            placeholder="Contoh: TEKNIK JAKARTA"
            maxLength={MAX_GROUP_LENGTH}
            autoComplete="off"
            error={errors.grup?.message}
            disabled={busy}
            {...register('grup')}
          />

          <div>
            <Field
              id="atasan"
              label="Atasan"
              placeholder="ID operator atasan"
              maxLength={MAX_SUPERVISOR_LENGTH}
              autoComplete="off"
              hint={
                supervisorName === ''
                  ? 'Diisi ID operator, bukan nama. Terisi sendiri setelah pencarian.'
                  : undefined
              }
              error={errors.atasan?.message}
              disabled={busy}
              {...register('atasan')}
            />
            {supervisorName !== '' && (
              <p className="mt-1.5 text-xs text-slate-500">
                Menurut direktori: <span className="font-medium text-slate-700">{supervisorName}</span>
              </p>
            )}
          </div>

          <Field
            id="kuota"
            label="Kuota"
            type="number"
            min={0}
            max={MAX_QUOTA}
            hint="Banyaknya pekerjaan yang boleh dipikul petugas ini."
            error={errors.kuota?.message}
            disabled={busy}
            {...register('kuota', { valueAsNumber: true })}
          />

          <Field
            id="kuota_luar"
            label="Kuota Sistem Lain"
            type="number"
            min={0}
            max={MAX_QUOTA}
            hint="Beban petugas yang sama di aplikasi lain. Diisi manual sampai API penggantinya ada."
            error={errors.kuota_luar?.message}
            disabled={busy}
            {...register('kuota_luar', { valueAsNumber: true })}
          />

          <SelectField
            id="aktif"
            label="Status"
            options={[
              { value: '1', label: 'Aktif' },
              { value: '0', label: 'Nonaktif' },
            ]}
            emptyText="— pilih —"
            error={errors.aktif?.message}
            disabled={busy}
            {...register('aktif')}
          />

          {editing && (
            <div>
              <span className="block text-sm font-medium text-slate-700">Beban Kerja</span>
              <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 text-sm text-slate-500">
                {technician.beban_kerja} pekerjaan berjalan
              </p>
              <p className="mt-1.5 text-xs text-slate-500">
                Dihitung sistem; tidak dapat diubah dari layar ini.
              </p>
            </div>
          )}
        </div>

        {/* Peringatan menonaktifkan diberikan SEBELUM disimpan, bukan sesudah. Petugas
            yang dinonaktifkan hilang dari daftar, dan tanpa keterangan ini pengguna akan
            mengira datanya terhapus. */}
        {editing && technician.aktif && (
          <p className="rounded-kontrol bg-amber-50 px-3 py-2 text-xs text-amber-800 ring-1 ring-amber-100">
            Menonaktifkan petugas membuatnya hilang dari daftar — sama seperti di sistem
            lama. Datanya tidak dihapus dan masih dapat dibuka lewat pencarian ID.
          </p>
        )}

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

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Galat pencarian dipisahkan dari galat simpan.
 *
 * Keduanya terjadi pada saat yang berbeda dan menuntut tindakan yang berbeda: yang satu
 * saat mengisi ID, yang lain saat menekan Simpan. Menyatukannya akan membuat pesan
 * pencarian bertahan di layar setelah pengguna memperbaiki ID-nya.
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

/** Galat yang berasal dari direktori pegawai, dipakai kedua jalur. */
function parseDirectory(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Pencarian pegawai belum dapat dijalankan. Periksa koneksi lalu coba lagi.',
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
      // ID yang tidak terdaftar sudah disorot di kolomnya; kotak pesan hanya akan
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
      return { title: 'Pencarian gagal', description: error.message, tone: 'gangguan' }
  }
}

function parseSave(error: APIError): MessageContent | null {
  switch (error.kode) {
    case ErrorCode.technicianExists:
      return {
        title: 'ID operator sudah terdaftar',
        description: 'Petugas dengan ID itu sudah ada di entitas ini. Buka datanya lalu ubah di sana.',
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
      return parseDirectory(error) ?? { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
  }
}
