import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { AccountErrorCode, AccountStatus, type Account } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField, type SelectOption } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'

import {
  useSubmitAccount,
  useUpdateAccount,
  useBankList,
  type AccountFields,
} from './api'

/**
 * Aturan wajib di sini adalah CERMINAN tanda bintang merah pada layar Pega, bukan
 * karangan sendiri.
 *
 * Sebelas kolom bertanda `*` di layar `Memperbaharui Data`: NOMOR REKENING, NAMA BANK,
 * Input Nama, NAMA CABANG BANK, NOMOR TELEPON, Email, ALAMAT, TIPE REKENING,
 * STATUS AKTIF, KTP/NIK/NPWP, dan Email Inputor. Seluruhnya wajib di sini pula.
 *
 * Server tetap memeriksa ulang semuanya — validasi peramban hanya mempercepat umpan
 * balik dan dapat dilewati siapa pun dengan memanggil API langsung.
 */
const schema = z.object({
  nomorRekening: z
    .string()
    .trim()
    .min(1, 'Nomor rekening wajib diisi.')
    .regex(/^[0-9-]+$/, 'Nomor rekening hanya boleh berisi angka.'),
  kodeBank: z.string().trim().min(1, 'Nama bank wajib dipilih.'),
  namaPemilik: z.string().trim().min(1, 'Nama wajib diisi.'),
  cabangBank: z.string().trim().min(1, 'Nama cabang bank wajib diisi.'),
  telepon: z.string().trim().min(1, 'Nomor telepon wajib diisi.'),
  email: z.string().trim().min(1, 'Email wajib diisi.').email('Format email tidak benar.'),
  alamatBank: z.string().trim().min(1, 'Alamat wajib diisi.'),
  tipeRekening: z.string().trim().min(1, 'Tipe rekening wajib dipilih.'),
  statusAktif: z.string().trim().min(1, 'Status aktif wajib dipilih.'),
  nik: z.string().trim().min(1, 'KTP/NIK/NPWP wajib diisi.'),
  emailInputor: z
    .string()
    .trim()
    .min(1, 'Email inputor wajib diisi.')
    .email('Format email tidak benar.'),
  idDokumen: z.string().trim(),
})

type FieldValues = z.infer<typeof schema>

/**
 * Tipe rekening — isi kolom ACCOUNT_TYPE.
 *
 * Nilainya diambil apa adanya dari activity SetTipeRekening, yang mengisi daftar
 * pilihan layar lama:
 *
 *	TempTipeBank.pxResults(<APPEND>).Telephone = "BIASA"
 *	TempTipeBank.pxResults(<APPEND>).Telephone = "VA"
 */
const ACCOUNT_TYPE_OPTIONS: SelectOption[] = [
  { value: 'BIASA', label: 'BIASA' },
  { value: 'VA', label: 'VA' },
]

/**
 * Status aktif — isi kolom STS_AKTIF.
 *
 * Sandinya "Ya"/"Tidak", BUKAN "1"/"0". Nilainya dari activity yang sama:
 *
 *	TempTipeBank.pxResults(<APPEND>).NomorKontrak = "Ya"
 *	TempTipeBank.pxResults(<APPEND>).NomorKontrak = "Tidak"
 *
 * Di layar Pega ia dropdown, bukan kotak centang — dan itu ditiru di sini: kotak centang
 * tidak punya keadaan "belum dipilih", sehingga tidak dapat mewajibkan pengguna
 * menentukan pilihannya.
 */
const ACTIVE_OPTIONS: SelectOption[] = [
  { value: 'Ya', label: 'Ya' },
  { value: 'Tidak', label: 'Tidak' },
]

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  account: Account | null
  onClose: () => void
}

/**
 * Form tambah dan ubah Master Rekening.
 *
 * Susunan kolom, label, tanda wajib, dan jenis kontrolnya mengikuti layar Pega
 * `Memperbaharui Data` apa adanya. Yang sebelumnya berbeda dan kini diluruskan:
 *
 * | Hal | Sebelumnya di sini | Pega — dan sekarang |
 * |---|---|---|
 * | Bank | dua kolom: dropdown kode + teks nama | **satu** dropdown NAMA BANK |
 * | Nama pemilik | input satu baris | **textarea** "Input Nama" |
 * | Alamat | input, label "Alamat bank" | **textarea**, label "ALAMAT" |
 * | Status aktif | kotak centang | **dropdown** `--Pilih--` |
 * | Nomor telepon | opsional | **wajib** |
 * | Email Inputor | diambil diam-diam dari sesi | **kolom wajib** yang diisi pengguna |
 * | Catatan | ada | **tidak ada** — NOTE diisi komite saat menyetujui, bukan pengaju |
 * | ID dokumen | kolom teks | tombol **Upload Document** |
 *
 * # Bentuknya mengikuti layar master lain, bukan hanya layar Pega
 *
 * Isian memakai `Field`, `SelectField`, dan `TextAreaField` dari pustaka komponen; panel
 * dan tombolnya mengikuti Master Tipe Surveyors. Yang ditiru dari Pega adalah **kolom
 * apa saja dan apa namanya** (`D-13`), bukan bagaimana sebuah kotak isian digambar —
 * dan `U-2` menetapkan yang kedua itu satu untuk seluruh aplikasi.
 */
export function AccountForm({ account, onClose }: Props) {
  const submit = useSubmitAccount()
  const update = useUpdateAccount()
  const bank = useBankList()

  const editing = account !== null
  const action = editing ? update : submit
  const firstField = useRef<HTMLInputElement | null>(null)

  // Tombol Upload Document di Pega membuka pengunggah berkas. Modul dokumen (`S-1`)
  // belum ada, sehingga yang dibuka di sini adalah isian ID dokumen — satu-satunya
  // bagian yang memang dibutuhkan komite untuk menyetujui. Letak dan namanya tetap
  // sama supaya layar tidak berubah bentuk saat pengunggah sungguhan dipasang.
  const [showDocument, setShowDocument] = useState(false)

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: initialValues(account),
  })

  // Saat baris lain dipilih untuk diubah, form harus memuat ulang isinya. Tanpa ini,
  // React Hook Form mempertahankan nilai pertama dan pengguna menyunting data yang
  // salah tanpa menyadarinya.
  useEffect(() => {
    reset(initialValues(account))
    setShowDocument(false)
  }, [account, reset])

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan
  // ketik harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    firstField.current?.focus()
  }, [])

  // Galat validasi dari server dipindahkan ke kolomnya masing-masing. Tanpa langkah
  // ini, pengguna melihat satu kotak merah berisi daftar kolom dan harus mencocokkan
  // sendiri kalimat mana milik kolom mana.
  useEffect(() => {
    const error = action.error
    if (!(error instanceof APIError) || error.kode !== AccountErrorCode.invalidInput) return

    for (const [field, message] of Object.entries(error.violations())) {
      const column = COLUMN_MAP[field]
      if (column) setError(column, { type: 'server', message })
    }
  }, [action.error, setError])

  const { ref: refNumber, ...remainingNumber } = register('nomorRekening')

  function send(values: FieldValues) {
    const payload = fieldsFrom(values, bank.data?.bank ?? [])

    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal.
    if (editing && account) {
      update.mutate(
        { kodeBank: account.kode_bank, nomorRekening: account.nomor_rekening, values: payload },
        { onSuccess: onClose },
      )
      return
    }
    submit.mutate(payload, { onSuccess: onClose })
  }

  const bankOptions: SelectOption[] = (bank.data?.bank ?? []).map((b) => ({
    value: b.kode,
    label: b.nama,
  }))

  // Saat mengubah rekening yang banknya tidak ada di daftar — misalnya daftar gagal
  // dimuat, atau banknya sudah dihapus dari master — pilihannya tetap ditampilkan.
  // Tanpa ini, menyimpan ulang rekening lama diam-diam memindahkannya ke bank lain.
  if (account && !bankOptions.some((o) => o.value === account.kode_bank)) {
    bankOptions.unshift({ value: account.kode_bank, label: account.nama_bank || account.kode_bank })
  }

  return (
    /*
      Panel muncul di atas tabel, bukan sebagai dialog melayang — sama dengan Master Tipe
      Surveyors dan master lainnya. Garis aksen di tepi kiri menandai bahwa panel ini
      keadaan sementara, bukan bagian tetap halaman.
    */
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={editing ? 'Ubah rekening' : 'Tambah rekening'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {editing ? `Memperbaharui Data — ${account.nomor_rekening}` : 'Memperbaharui Data'}
        </h3>

        {/*
          Kedua kalimat ini ada di layar Pega, di atas panel isian. Keduanya menjelaskan
          hal yang tidak terlihat dari kolomnya sendiri — kenapa nomor telepon harus
          nomor WhatsApp, dan apa yang terjadi setelah atasan menyetujui.
        */}
        <div className="mt-2 space-y-1 text-xs leading-relaxed text-slate-600">
          <p>
            No. Telp agar diisi nomor yang terhubung dengan WhatsApp untuk mengirim
            notifikasi dari Kasir jika sudah diproses bayar.
          </p>
          <p>Setelah atasan aksep rekening, rekening akan menunggu persetujuan dari Kasir.</p>
        </div>
      </div>

      <div className="space-y-5 p-5">
        {action.isError && <SaveErrorMessage error={action.error} />}

        {/*
          Peringatan ini muncul HANYA saat mengubah rekening yang sudah diputuskan
          komite, dan ia bukan hiasan: menyimpan akan mencabut persetujuannya. Pengguna
          yang mengira sedang membetulkan satu huruf perlu tahu bahwa rekening itu
          berhenti dapat dipakai sampai komite memutuskan ulang.
        */}
        {editing && account.status !== AccountStatus.menunggu && (
          <ErrorMessage
            title="Menyimpan akan mencabut persetujuan komite"
            description="Rekening ini sudah diputuskan komite. Setelah disimpan, ia kembali menunggu keputusan atas data yang baru, dan belum dapat dipakai membayar klaim sampai disetujui lagi."
            tone="penolakan"
          />
        )}

        <div className="grid gap-5 sm:grid-cols-2">
          <Field
            id="nomorRekening"
            label="NOMOR REKENING *"
            inputMode="numeric"
            autoComplete="off"
            // Nomor rekening adalah bagian kunci alaminya. Mengubahnya pada baris yang
            // sudah ada berarti memindahkan rekening ke kunci lain — itu pengajuan
            // baru, bukan perubahan.
            readOnly={editing}
            hint={
              editing
                ? 'Belum dapat diubah di sini. Layar lama mengizinkannya lewat OLDACCOUNT_NO; itu pekerjaan yang masih tersisa.'
                : undefined
            }
            error={errors.nomorRekening?.message}
            disabled={action.isPending}
            {...remainingNumber}
            ref={(element) => {
              refNumber(element)
              firstField.current = element
            }}
          />

          <div>
            <SelectField
              id="kodeBank"
              label="NAMA BANK *"
              options={bankOptions}
              emptyText="--Pilih--"
              error={errors.kodeBank?.message}
              disabled={action.isPending}
              {...register('kodeBank')}
            />
            {bank.isError && (
              <p className="mt-1 text-xs text-amber-800">
                Daftar bank tidak dapat dimuat. Muat ulang halaman; bila tetap kosong,
                hak baca master bank perlu diminta ke DBA.
              </p>
            )}
          </div>

          <TextAreaField
            id="namaPemilik"
            label="Input Nama *"
            rows={3}
            error={errors.namaPemilik?.message}
            disabled={action.isPending}
            {...register('namaPemilik')}
          />

          <TextAreaField
            id="alamatBank"
            label="ALAMAT *"
            rows={3}
            error={errors.alamatBank?.message}
            disabled={action.isPending}
            {...register('alamatBank')}
          />

          <Field
            id="cabangBank"
            label="NAMA CABANG BANK *"
            autoComplete="off"
            error={errors.cabangBank?.message}
            disabled={action.isPending}
            {...register('cabangBank')}
          />

          <Field
            id="telepon"
            label="NOMOR TELEPON *"
            inputMode="tel"
            autoComplete="off"
            hint="Nomor yang terhubung dengan WhatsApp."
            error={errors.telepon?.message}
            disabled={action.isPending}
            {...register('telepon')}
          />

          <Field
            id="email"
            label="Email *"
            type="email"
            autoComplete="off"
            error={errors.email?.message}
            disabled={action.isPending}
            {...register('email')}
          />

          <Field
            id="nik"
            label="KTP/NIK/NPWP *"
            inputMode="numeric"
            autoComplete="off"
            error={errors.nik?.message}
            disabled={action.isPending}
            {...register('nik')}
          />

          <SelectField
            id="tipeRekening"
            label="TIPE REKENING *"
            options={ACCOUNT_TYPE_OPTIONS}
            emptyText="--Pilih--"
            error={errors.tipeRekening?.message}
            disabled={action.isPending}
            {...register('tipeRekening')}
          />

          <SelectField
            id="statusAktif"
            label="STATUS AKTIF *"
            options={ACTIVE_OPTIONS}
            emptyText="--Pilih--"
            error={errors.statusAktif?.message}
            disabled={action.isPending}
            {...register('statusAktif')}
          />

          <div className="sm:col-span-2">
            <Field
              id="emailInputor"
              label="Email Inputor *"
              type="email"
              autoComplete="off"
              placeholder="Wajib masukan email Anda untuk Notif Approval dari Kasir"
              hint="Ke alamat inilah Kasir mengirim pemberitahuan saat rekening disetujui."
              error={errors.emailInputor?.message}
              disabled={action.isPending}
              {...register('emailInputor')}
            />
          </div>
        </div>

        <div className="border-t border-slate-100 pt-5">
          <Button
            tone="kedua"
            onClick={() => setShowDocument((open) => !open)}
            disabled={action.isPending}
          >
            Upload Document
          </Button>

          {showDocument && (
            <div className="mt-3 max-w-md">
              <Field
                id="idDokumen"
                label="ID dokumen buku rekening"
                autoComplete="off"
                hint="Pengunggah berkas adalah modul S-1 yang belum dibangun. Isi ID dokumen yang sudah ada di penyimpanan — komite tidak dapat menyetujui rekening tanpa buku rekening."
                error={errors.idDokumen?.message}
                disabled={action.isPending}
                {...register('idDokumen')}
              />
            </div>
          )}
        </div>

        <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
          <Button type="submit" tone="utama" disabled={action.isPending}>
            {action.isPending && <Spinner />}
            {action.isPending ? 'Menyimpan…' : 'Simpan'}
          </Button>
          <Button tone="halus" onClick={onClose} disabled={action.isPending}>
            Batal
          </Button>
        </div>
      </div>
    </form>
  )
}

/** Pemutar kecil pada tombol yang sedang bekerja. */
function Spinner() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" className="h-4 w-4 animate-spin">
      <circle cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" strokeOpacity="0.3" strokeWidth="2" />
      <path
        d="M8 1.5a6.5 6.5 0 0 1 6.5 6.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

/** initialValues menyusun isian awal — kosong saat menambah, terisi saat mengubah. */
function initialValues(account: Account | null): FieldValues {
  return {
    nomorRekening: account?.nomor_rekening ?? '',
    kodeBank: account?.kode_bank ?? '',
    namaPemilik: account?.nama_pemilik ?? '',
    cabangBank: account?.cabang_bank ?? '',
    telepon: account?.telepon ?? '',
    email: account?.email ?? '',
    alamatBank: account?.alamat_bank ?? '',
    tipeRekening: account?.tipe_rekening ?? '',
    // Nol-nilai bool tidak dapat membedakan "nonaktif" dari "belum dipilih", sehingga
    // status dibawa sebagai teks di formulir dan baru diubah menjadi bool saat dikirim.
    statusAktif: account === null ? '' : account.aktif ? 'Ya' : 'Tidak',
    nik: account?.nik ?? '',
    emailInputor: account?.email_inputor ?? '',
    idDokumen: account?.id_dokumen ?? '',
  }
}

/**
 * fieldsFrom mengubah isian formulir menjadi badan permintaan.
 *
 * Nama bank diturunkan dari kode yang dipilih, bukan diketik terpisah. Di layar Pega
 * keduanya satu kontrol; memisahkannya menjadi dua kolom seperti sebelumnya membuka
 * kemungkinan `BANKID` dan `BANK_NAME` saling bertentangan di dalam satu baris.
 */
function fieldsFrom(
  values: FieldValues,
  banks: { kode: string; nama: string }[],
): AccountFields {
  const selected = banks.find((b) => b.kode === values.kodeBank)

  return {
    nomorRekening: values.nomorRekening,
    namaPemilik: values.namaPemilik,
    namaBank: selected?.nama ?? values.kodeBank,
    cabangBank: values.cabangBank,
    alamatBank: values.alamatBank,
    kodeBank: values.kodeBank,
    tipeRekening: values.tipeRekening,
    email: values.email,
    telepon: values.telepon,
    nik: values.nik,
    emailInputor: values.emailInputor,
    idDokumen: values.idDokumen,
    catatan: '',
    aktif: values.statusAktif === 'Ya',
  }
}

/** COLUMN_MAP memetakan nama field server ke nama field formulir. */
const COLUMN_MAP: Record<string, keyof FieldValues> = {
  nomor_rekening: 'nomorRekening',
  nama_pemilik: 'namaPemilik',
  nama_bank: 'kodeBank',
  kode_bank: 'kodeBank',
  cabang_bank: 'cabangBank',
  alamat_bank: 'alamatBank',
  tipe_rekening: 'tipeRekening',
  email: 'email',
  telepon: 'telepon',
  nik: 'nik',
  email_penginput: 'emailInputor',
}

/**
 * Galat simpan dibedakan menurut KODE-nya, bukan teks pesannya.
 *
 * Jenisnya menuntut tindak lanjut berbeda: nomor yang bentrok dapat diperbaiki pengguna,
 * rekening yang sudah diputuskan tidak dapat ditolong dengan mencoba ulang.
 */
function SaveErrorMessage({ error }: { error: unknown }) {
  const content = messageFor(error)
  if (content === null) return null
  return <ErrorMessage title={content.title} description={content.description} tone={content.tone} />
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Perubahan belum tersimpan. Periksa koneksi lalu coba lagi.',
      tone: 'gangguan',
    }
  }
  if (!(error instanceof APIError)) {
    return {
      title: 'Gagal menyimpan',
      description: 'Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi.',
      tone: 'gangguan',
    }
  }

  switch (error.kode) {
    case AccountErrorCode.alreadyExists:
      return {
        title: 'Nomor rekening sudah terdaftar',
        description:
          'Nomor ini sudah ada dan belum ditolak komite. Gunakan data yang sudah ada, atau tunggu keputusan komite atas pengajuan sebelumnya.',
        tone: 'penolakan',
      }

    case AccountErrorCode.invalidInput:
      // Bila detailnya ada, isiannya sudah disorot di tempatnya; kotak pesan hanya akan
      // mengulang hal yang sama.
      return Object.keys(error.violations()).length > 0
        ? null
        : {
            title: 'Ada isian yang belum benar',
            description: error.message,
            tone: 'penolakan',
          }

    case AccountErrorCode.alreadyDecided:
      return {
        title: 'Rekening sudah diputuskan komite',
        description:
          'Rekening yang sudah disetujui atau ditolak tidak dapat diubah. Ajukan rekening baru bila datanya perlu berubah.',
        tone: 'penolakan',
      }

    default:
      return {
        title: 'Gagal menyimpan',
        description: error.message,
        tone: 'gangguan',
      }
  }
}
