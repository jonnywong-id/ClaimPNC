import { useEffect, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { centsToRupiah, rupiahToCents } from '@/components/format'

import { useClaimReport, useSaveClaimReport } from './api'
import { EMPTY_DETAIL, FIELD_LIMIT, type ClaimReportDetail } from './types'

/**
 * Form **Input Receive Document** — pengganti flow action dengan nama yang sama pada
 * `Flow/InputReceiveDocument.xml`.
 *
 * # Ia yang membuat tombol "Buat Baru" berarti
 *
 * Di sistem lama, menekan Buat Baru menjalankan `CreateNewCaseRCV`, yang membuat berkas
 * KOSONG lalu menyerahkannya ke alur Receive Document. Alur itu punya satu assignment —
 * "Receive Document" — dan assignment itu merender form ini. Berkasnya lahir kosong justru
 * supaya form inilah yang mengisinya.
 *
 * Tanpa layar ini, tombol Buat Baru menerbitkan berkas yang tidak dapat diapa-apakan —
 * dan itu tampak seperti tombol yang tidak bekerja.
 *
 * # Berkas milik Pega dibuka dalam modus baca saja
 *
 * Selama masa paralel, tepat satu sistem yang menulis sebuah baris (`ADR-0004`, `P-1`).
 * Kewenangannya dihitung SERVER dan dikirim sebagai `dapat_disunting`; layar tidak
 * menyimpulkannya sendiri dari kolom `asal`.
 */
export function ClaimReportFormPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()

  const berkas = useClaimReport(id)
  const simpan = useSaveClaimReport(id)

  const [values, setValues] = useState<ClaimReportDetail>(EMPTY_DETAIL)
  const [estimateText, setEstimateText] = useState('')

  // Isian form diisi SEKALI dari jawaban server, lalu menjadi milik pengguna. Menyalinnya
  // pada setiap render akan menimpa ketikan yang sedang berjalan setiap kali TanStack
  // Query menyegarkan datanya di latar belakang.
  useEffect(() => {
    if (!berkas.data?.isian) return
    setValues(berkas.data.isian)
    setEstimateText(centsToRupiah(berkas.data.isian.estimasi_kerugian))
  }, [berkas.data])

  const editable = berkas.data?.dapat_disunting ?? false
  const violation = simpan.error instanceof APIError ? simpan.error.violations() : {}

  function set<K extends keyof ClaimReportDetail>(field: K, value: ClaimReportDetail[K]) {
    setValues((previous) => ({ ...previous, [field]: value }))
  }

  if (berkas.isPending) {
    return (
      <FormFrame id={id}>
        <p className="mt-6 text-sm text-slate-500" role="status">
          Memuat berkas…
        </p>
      </FormFrame>
    )
  }

  if (berkas.isError) {
    return (
      <FormFrame id={id}>
        <div className="mt-6">
          <ErrorMessage
            title="Berkas tidak dapat dibuka"
            description={messageOf(berkas.error)}
            tone="gangguan"
          />
        </div>
      </FormFrame>
    )
  }

  return (
    <FormFrame id={id} report={berkas.data?.laporan.posisi}>
      {!editable && (
        <div className="mt-4">
          <ErrorMessage
            title="Berkas ini hanya dapat dibaca"
            description={
              'Ia masih dikelola sistem lama, dan selama masa paralel hanya satu sistem ' +
              'yang boleh menulis sebuah berkas. Isinya dapat dilihat di sini; ' +
              'mengubahnya dilakukan di Pega.'
            }
            tone="gangguan"
          />
        </div>
      )}

      {simpan.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Berkas tidak dapat disimpan"
            description={messageOf(simpan.error)}
            tone="penolakan"
          />
        </div>
      )}

      {simpan.isSuccess && !simpan.isPending && (
        <p
          className="mt-4 rounded-kartu border border-blue-200 bg-blue-50/80 px-4 py-3 text-sm text-blue-900"
          role="status"
        >
          Berkas tersimpan.
        </p>
      )}

      <form
        className="mt-6 space-y-6"
        onSubmit={(e) => {
          e.preventDefault()
          if (!editable) return
          simpan.mutate({ ...values, estimasi_kerugian: rupiahToCents(estimateText) || 0 })
        }}
      >
        <Group title="Dokumen masuk">
          <Field
            id="tanggal_terima_dokumen"
            label="Tanggal Terima Dokumen"
            type="date"
            value={values.tanggal_terima_dokumen}
            onChange={(e) => set('tanggal_terima_dokumen', e.target.value)}
            error={violation['tanggal_terima_dokumen']}
            disabled={!editable}
          />
          <Field
            id="nama_pelapor"
            label="Nama Pengirim / Pelapor Dokumen"
            value={values.nama_pelapor}
            onChange={(e) => set('nama_pelapor', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['nama_pelapor']}
            disabled={!editable}
          />
          <Field
            id="email_pelapor"
            label="Email Pengirim"
            type="email"
            value={values.email_pelapor}
            onChange={(e) => set('email_pelapor', e.target.value)}
            maxLength={FIELD_LIMIT.email}
            error={violation['email_pelapor']}
            disabled={!editable}
          />
          <Field
            id="telepon_pelapor"
            label="No. HP Pengirim"
            value={values.telepon_pelapor}
            onChange={(e) => set('telepon_pelapor', e.target.value)}
            maxLength={FIELD_LIMIT.telepon}
            error={violation['telepon_pelapor']}
            disabled={!editable}
          />
          <Field
            id="nama_kurir"
            label="Nama Kurir ASM"
            value={values.nama_kurir}
            onChange={(e) => set('nama_kurir', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['nama_kurir']}
            disabled={!editable}
          />
          <Field
            id="jumlah_dokumen"
            label="Total Jumlah Dokumen"
            type="number"
            min={0}
            max={FIELD_LIMIT.jumlahDokumen}
            value={values.jumlah_dokumen === 0 ? '' : String(values.jumlah_dokumen)}
            onChange={(e) => set('jumlah_dokumen', Number(e.target.value) || 0)}
            error={violation['jumlah_dokumen']}
            disabled={!editable}
            hint="Rincian per dokumen menunggu modul penyimpanan dokumen."
          />
        </Group>

        <Group title="Polis dan kejadian">
          <Field
            id="nomor_polis"
            label="Nomor Polis"
            value={values.nomor_polis}
            onChange={(e) => set('nomor_polis', e.target.value)}
            maxLength={FIELD_LIMIT.polis}
            error={violation['nomor_polis']}
            disabled={!editable}
          />
          <Field
            id="tanggal_kejadian"
            label="Tanggal Kejadian"
            type="date"
            value={values.tanggal_kejadian}
            onChange={(e) => set('tanggal_kejadian', e.target.value)}
            error={violation['tanggal_kejadian']}
            disabled={!editable}
          />
          <Field
            id="tertanggung"
            label="Nama Tertanggung"
            value={values.tertanggung}
            onChange={(e) => set('tertanggung', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['tertanggung']}
            disabled={!editable}
          />
          <Field
            id="nama_bisnis"
            label="Nama Bisnis"
            value={values.nama_bisnis}
            onChange={(e) => set('nama_bisnis', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['nama_bisnis']}
            disabled={!editable}
          />
          <Field
            id="nomor_rujukan"
            label="No. Referensi/Placing Slip"
            value={values.nomor_rujukan}
            onChange={(e) => set('nomor_rujukan', e.target.value)}
            maxLength={FIELD_LIMIT.rujukan}
            error={violation['nomor_rujukan']}
            disabled={!editable}
          />
          {/*
            Uang diketik sebagai teks lalu diubah menjadi SEN saat dikirim. Memakai
            <input type="number"> untuk rupiah membuat peramban menyimpannya sebagai
            pecahan biner, dan `ADR-0016` menuntut presisi penuh.
          */}
          <Field
            id="estimasi_kerugian"
            label="Estimasi Kerugian"
            value={estimateText}
            onChange={(e) => setEstimateText(e.target.value)}
            placeholder="2.500.000,00"
            inputMode="decimal"
            error={violation['estimasi_kerugian']}
            disabled={!editable}
            hint="Angka yang disebut pelapor; bukan nilai klaim."
          />
          <Field
            id="lokasi_kejadian"
            label="Lokasi Kejadian"
            value={values.lokasi_kejadian}
            onChange={(e) => set('lokasi_kejadian', e.target.value)}
            maxLength={FIELD_LIMIT.lokasi}
            error={violation['lokasi_kejadian']}
            disabled={!editable}
          />
          <Field
            id="subjek_email"
            label="Subject Email"
            value={values.subjek_email}
            onChange={(e) => set('subjek_email', e.target.value)}
            maxLength={FIELD_LIMIT.subjek}
            error={violation['subjek_email']}
            disabled={!editable}
          />
        </Group>

        <Group title="Keterangan" full>
          <TextArea
            id="kronologis"
            label="Kronologis Kejadian"
            value={values.kronologis}
            onChange={(value) => set('kronologis', value)}
            maxLength={FIELD_LIMIT.narasi}
            error={violation['kronologis']}
            disabled={!editable}
          />
          <TextArea
            id="rincian_kerusakan"
            label="Rincian Kerusakan"
            value={values.rincian_kerusakan}
            onChange={(value) => set('rincian_kerusakan', value)}
            maxLength={FIELD_LIMIT.narasi}
            error={violation['rincian_kerusakan']}
            disabled={!editable}
          />
          <TextArea
            id="alasan"
            label="Keterangan Belum Transfer"
            value={values.alasan}
            onChange={(value) => set('alasan', value)}
            maxLength={FIELD_LIMIT.catatan}
            error={violation['alasan']}
            disabled={!editable}
            rows={3}
          />
          <TextArea
            id="keterangan_belum_registrasi"
            label="Keterangan Belum Registrasi"
            value={values.keterangan_belum_registrasi}
            onChange={(value) => set('keterangan_belum_registrasi', value)}
            maxLength={FIELD_LIMIT.catatan}
            error={violation['keterangan_belum_registrasi']}
            disabled={!editable}
            rows={3}
          />
        </Group>

        <div className="flex flex-wrap items-center gap-3 border-t border-slate-200 pt-5">
          {editable && (
            <Button type="submit" tone="utama" disabled={simpan.isPending}>
              {simpan.isPending ? 'Menyimpan…' : 'Simpan'}
            </Button>
          )}
          <Button type="button" tone="kedua" onClick={() => navigate('/inbox/laporan-klaim')}>
            Kembali ke daftar
          </Button>
        </div>
      </form>

      <p className="mt-6 text-xs text-slate-500">
        Tiga bagian form lama belum ada di sini: data pelapor beserta alamatnya — yang di
        sistem lama terikat area Heavy Equipment dan berada di luar lingkup migrasi — daftar
        rincian dokumen yang menunggu modul penyimpanan dokumen, serta riwayat komunikasi
        dan progres yang dimiliki modul lain.
      </p>
    </FormFrame>
  )
}

function FormFrame({
  id,
  report,
  children,
}: {
  id: string
  report?: string | undefined
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <nav className="text-xs text-slate-500">
        <Link to="/inbox/laporan-klaim" className="underline hover:text-slate-800">
          Inbox Laporan Klaim
        </Link>
        <span aria-hidden="true"> › </span>
        <span>{id}</span>
      </nav>

      <header className="mt-2 border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Input Receive Document</h1>
        <p className="mt-1 text-sm text-slate-600">
          Berkas <span className="font-medium text-slate-900">{id}</span>
          {report && <> · {report}</>}
        </p>
      </header>

      {children}
    </div>
  )
}

/** Satu kelompok isian, mengikuti pengelompokan form lama. */
function Group({
  title,
  children,
  full = false,
}: {
  title: string
  children: ReactNode
  full?: boolean
}) {
  return (
    <section className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">{title}</h2>
      <div className={['mt-4 grid gap-4', full ? '' : 'sm:grid-cols-2'].join(' ')}>{children}</div>
    </section>
  )
}

/**
 * Isian bernarasi panjang.
 *
 * Ia tidak memakai `Field` karena `Field` membungkus `<input>`, dan kronologis kejadian
 * berlebar 4.000 karakter — satu baris tidak dapat menampungnya. Bentuk, kelas, dan
 * penandaan aria-nya dijaga sama dengan `Field` supaya form tidak terlihat seperti
 * dirakit dari dua aplikasi berbeda.
 */
function TextArea({
  id,
  label,
  value,
  onChange,
  maxLength,
  error,
  disabled,
  rows = 5,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  maxLength: number
  error?: string | undefined
  disabled?: boolean
  rows?: number
}) {
  const errorID = `${id}-galat`
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <textarea
        id={id}
        rows={rows}
        value={value}
        maxLength={maxLength}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorID : undefined}
        className={[
          'mt-1.5 w-full rounded-kontrol border bg-white px-3 py-2.5 text-sm text-slate-900',
          'transition-[border-color,box-shadow] duration-150 ease-halus',
          'focus:outline-none focus:ring-4',
          error
            ? 'border-red-400 focus:border-red-500 focus:ring-red-500/15'
            : 'border-slate-300 hover:border-slate-400 focus:border-blue-500 focus:ring-blue-500/15',
          'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
        ].join(' ')}
      />
      {error && (
        <p id={errorID} className="mt-1.5 text-sm text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}

function messageOf(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
