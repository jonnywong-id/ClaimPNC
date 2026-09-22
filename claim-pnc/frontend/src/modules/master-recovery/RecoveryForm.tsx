import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type Recovery,
  type RecoveryClaimLine,
  type RecoveryPrincipal,
} from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { formatMoney, parseMoney, remainder } from '@/lib/money'

import { ClaimLineUpload } from './ClaimLineUpload'
import { useLookupPolicy, useSaveRecovery, useUploadPaymentProof } from './api'

/** Batas panjang; sama dengan masterrecovery.Max* di backend. */
const MAX_PRINCIPAL_NAME = 200
const MAX_CLIENT_ID = 100
const MAX_VIRTUAL_ACCOUNT = 1000
const MAX_REMARK = 2000
const MAX_CASE_POSITION = 100
const MAX_POLICY_NO = 200

/**
 * money adalah pembaca isian nilai uang.
 *
 * Ia menerima TEKS, bukan angka, karena isiannya memang kotak teks: petugas mengetik
 * "1.000.000" dengan pemisah ribuan, dan `<input type="number">` menolaknya. Yang
 * ditegakkan sama dengan server — rupiah utuh, tidak boleh negatif.
 */
function money(label: string) {
  return z
    .string()
    .trim()
    .transform((value, ctx) => {
      if (value === '') return 0
      const parsed = parseMoney(value)
      if (parsed === null) {
        ctx.addIssue({ code: 'custom', message: `${label} harus berupa angka rupiah utuh.` })
        return z.NEVER
      }
      if (parsed < 0) {
        ctx.addIssue({ code: 'custom', message: `${label} tidak boleh negatif.` })
        return z.NEVER
      }
      return parsed
    })
}

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan langsung
 * ke API tidak melewati layar ini sama sekali.
 *
 * Keenam isian wajibnya diambil dari prasyarat penolakan di
 * `Activity/Insert_mst_recoveryKlaimASM-Act.xml:802`. Sisa TIDAK ada di sini karena ia
 * DIHITUNG — layar memperlihatkannya, server menetapkannya.
 */
const schema = z.object({
  nama_principal: z
    .string()
    .trim()
    .min(1, 'Nama principal wajib diisi.')
    .max(MAX_PRINCIPAL_NAME, `Nama principal paling panjang ${MAX_PRINCIPAL_NAME} karakter.`),
  client_id: z.string().trim().max(MAX_CLIENT_ID, `Client ID paling panjang ${MAX_CLIENT_ID} karakter.`),
  nomor_virtual_account: z
    .string()
    .trim()
    .max(MAX_VIRTUAL_ACCOUNT, `Nomor virtual account paling panjang ${MAX_VIRTUAL_ACCOUNT} karakter.`),
  tahun: z.string().trim().min(1, 'Tahun wajib dipilih.'),
  nilai_klaim: money('Nilai klaim'),
  pembayaran_sebelumnya: money('Nilai pembayaran sebelumnya'),
  pembayaran: money('Pembayaran'),
  keterangan: z
    .string()
    .trim()
    .min(1, 'Keterangan wajib diisi.')
    .max(MAX_REMARK, `Keterangan paling panjang ${MAX_REMARK} karakter.`),
  posisi_kasus: z
    .string()
    .trim()
    .min(1, 'Posisi kasus wajib diisi.')
    .max(MAX_CASE_POSITION, `Posisi kasus paling panjang ${MAX_CASE_POSITION} karakter.`),
  nomor_polis: z
    .string()
    .trim()
    .max(MAX_POLICY_NO, `Nomor polis paling panjang ${MAX_POLICY_NO} karakter.`),
})

type FieldValues = z.input<typeof schema>
type ParsedValues = z.output<typeof schema>

type Props = {
  /** Nomor batch PERKIRAAN, untuk ditampilkan saja. */
  nextBatch: number | undefined
  year: string[]
  principal: RecoveryPrincipal[]
  onSaved: (saved: Recovery, policyResolved: boolean) => void
}

/**
 * Form entri batch recovery.
 *
 * Menggantikan `Section/OutstandingMasterRecovery-Section.xml` — sembilan isian, tombol
 * unggah bukti bayar, panel data klaim, dan tombol **Transfer Recovery**.
 *
 * # Yang sengaja dibuat berbeda dari layar lama
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Isian kurang | satu pesan "Wajib ISI semua field" | tiap kolom ditandai di tempatnya |
 * | Sisa | isian biasa yang dapat disunting | dihitung, ditampilkan, tidak dapat diketik |
 * | Nomor polis salah | baru ketahuan setelah tersimpan | dicari saat ditekan Cari |
 * | Nilai negatif | diterima | ditolak |
 * | Judul kolom | nama properti Pega | nama yang dibaca manusia (`D-19`) |
 * | Layar sempit | digulir menyamping | tersusun satu kolom (`D-12`) |
 */
export function RecoveryForm({ nextBatch, year, principal, onSaved }: Props) {
  const save = useSaveRecovery()
  const lookup = useLookupPolicy()
  const upload = useUploadPaymentProof()

  const [claimLine, setClaimLine] = useState<RecoveryClaimLine[]>([])
  const [paymentProof, setPaymentProof] = useState<{ id: string; nama: string } | null>(null)

  const {
    register,
    handleSubmit,
    setValue,
    control,
    reset,
    setError,
    formState: { errors },
  } = useForm<FieldValues, unknown, ParsedValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama_principal: '',
      client_id: '',
      nomor_virtual_account: '',
      tahun: year[0] ?? '',
      nilai_klaim: '',
      pembayaran_sebelumnya: '',
      pembayaran: '',
      keterangan: '',
      posisi_kasus: '',
      nomor_polis: '',
    },
  })

  // Sisa dihitung ulang pada setiap ketikan, meniru aksi `HitungSisaKlaimRecovery` yang di
  // layar lama dijalankan pada perubahan ketiga isian nilai.
  const watched = useWatch({
    control,
    name: ['nilai_klaim', 'pembayaran_sebelumnya', 'pembayaran'],
  })
  const computedRemainder = useMemo(() => {
    const claimAmount = parseMoney(watched[0] ?? '') ?? 0
    const previousPayment = parseMoney(watched[1] ?? '') ?? 0
    const payment = parseMoney(watched[2] ?? '') ?? 0
    return remainder(claimAmount, previousPayment, payment)
  }, [watched])

  // Nomor polis diawasi terpisah: ia dipakai tombol Cari, dan tombolnya harus mati selama
  // isiannya masih kosong.
  const policyNo = useWatch({ control, name: 'nomor_polis' }) ?? ''
  const year0 = year[0]
  const chosenYear = useWatch({ control, name: 'tahun' })

  // Tahun terbaru dipilih lebih dulu — SETELAH daftarnya tiba.
  //
  // Ini perbaikan atas cacat yang nyata, bukan kerapian: `defaultValues` dibaca SEKALI saat
  // form pertama dirakit, dan pada saat itu daftar tahun masih kosong karena permintaannya
  // belum dijawab. Tanpa efek ini, isian Tahun tetap kosong meski daftarnya sudah terisi,
  // dan petugas harus memilihnya sendiri setiap kali membuka layar — atau menekan Simpan
  // lalu ditolak "Tahun wajib dipilih".
  //
  // Hanya mengisi yang MASIH KOSONG, sehingga pilihan yang sudah diubah petugas tidak
  // ditimpa ketika daftarnya dimuat ulang.
  useEffect(() => {
    if (year0 && !chosenYear) setValue('tahun', year0)
  }, [year0, chosenYear, setValue])

  // Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di
  // kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya
  // berguna bila layar menyorotnya di tempat isiannya.
  useEffect(() => {
    if (!(save.error instanceof APIError)) return
    const violation = save.error.violations()
    for (const name of [
      'nama_principal',
      'client_id',
      'nomor_virtual_account',
      'tahun',
      'nilai_klaim',
      'pembayaran_sebelumnya',
      'pembayaran',
      'keterangan',
      'posisi_kasus',
      'nomor_polis',
    ] as const) {
      const message = violation[name]
      if (message) setError(name, { type: 'server', message })
    }
  }, [save.error, setError])

  /** Memilih principal mengisi ketiga isian sekaligus, seperti autocomplete layar lama. */
  function choosePrincipal(clientID: string) {
    const chosen = principal.find((p) => p.client_id === clientID)
    if (!chosen) return
    setValue('nama_principal', chosen.nama_principal, { shouldValidate: true })
    setValue('client_id', chosen.client_id, { shouldValidate: true })
    setValue('nomor_virtual_account', chosen.nomor_virtual_account, { shouldValidate: true })
  }

  function send(values: ParsedValues) {
    save.mutate(
      {
        ...values,
        id_dokumen: paymentProof?.id ?? '',
        // Kolom NOHPLL. Namanya menyiratkan nomor telepon; isinya di sistem lama adalah
        // nomor catatan log layanan, dan layar ini tidak punya nomor semacam itu —
        // dikirim kosong, bukan diisi nilai karangan.
        id_log_layanan: '',
        baris_klaim: claimLine,
      },
      {
        onSuccess: (answer) => {
          onSaved(answer.recovery, answer.identitas_polis_terisi)
          // Form dikosongkan HANYA setelah server menjawab berhasil. Mengosongkannya lebih
          // dulu akan membuang isian petugas saat penyimpanan gagal.
          reset()
          setClaimLine([])
          setPaymentProof(null)
          lookup.reset()
        },
      },
    )
  }

  const busy = save.isPending
  const policy = lookup.data

  return (
    <form onSubmit={handleSubmit(send)} noValidate className="space-y-6" aria-label="Catat batch recovery">
      {save.isError && <SaveErrorMessage error={save.error} />}

      {/* ── Principal ─────────────────────────────────────────────────────────── */}
      <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
        <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
          <h3 className="text-base font-semibold text-slate-900">Principal</h3>
          <p className="mt-1 text-sm text-slate-600">
            Pihak penjamin yang mengembalikan dana, beserta rekening virtual tempat dana
            diterima.
          </p>
        </div>

        <div className="space-y-5 p-5">
          <SelectField
            id="pilih-principal"
            label="Pilih dari master"
            emptyText="— isi manual di bawah —"
            options={principal.map((p) => ({
              value: p.client_id,
              label: `${p.nama_principal} · ${p.nomor_virtual_account}`,
            }))}
            disabled={busy}
            onChange={(event) => choosePrincipal(event.target.value)}
          />

          <div className="grid gap-5 sm:grid-cols-2">
            <Field
              id="nama-principal"
              label="Nama Principal"
              placeholder="Contoh: PT CONTOH PENJAMINAN"
              maxLength={MAX_PRINCIPAL_NAME}
              autoComplete="off"
              error={errors.nama_principal?.message}
              disabled={busy}
              {...register('nama_principal')}
            />
            <Field
              id="client-id"
              label="Client ID"
              placeholder="Opsional"
              maxLength={MAX_CLIENT_ID}
              autoComplete="off"
              error={errors.client_id?.message}
              disabled={busy}
              {...register('client_id')}
            />
          </div>

          <Field
            id="nomor-virtual-account"
            label="Virtual Account Number"
            placeholder="Opsional — terisi sendiri bila principal dipilih dari master"
            maxLength={MAX_VIRTUAL_ACCOUNT}
            autoComplete="off"
            error={errors.nomor_virtual_account?.message}
            disabled={busy}
            {...register('nomor_virtual_account')}
          />
        </div>
      </section>

      {/* ── Nilai ─────────────────────────────────────────────────────────────── */}
      <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
        <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
          <h3 className="text-base font-semibold text-slate-900">Nilai</h3>
          <p className="mt-1 text-sm text-slate-600">
            Seluruhnya dalam rupiah utuh, tanpa sen. Sisa dihitung sendiri dan tidak dapat
            diketik.
          </p>
        </div>

        <div className="space-y-5 p-5">
          <div className="grid gap-5 sm:grid-cols-2">
            <SelectField
              id="tahun"
              label="Tahun"
              options={year.map((value) => ({ value, label: value }))}
              error={errors.tahun?.message}
              disabled={busy}
              {...register('tahun')}
            />
            <div>
              <span className="block text-sm font-medium text-slate-700">Nomor Batch</span>
              {/*
                Digambar sebagai kotak mati, bukan input ber-`disabled`. Input yang
                dinonaktifkan tetap terlihat seperti isian dan mengundang pengguna
                mengkliknya; kotak ini jelas bukan tempat mengetik.
              */}
              <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 font-mono text-sm text-slate-500">
                {nextBatch ?? '—'}
              </p>
              <p className="mt-1.5 text-xs text-slate-500">
                Perkiraan. Nomor yang mengikat diterbitkan saat disimpan.
              </p>
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-3">
            <Field
              id="nilai-klaim"
              label="Nilai Klaim (Rp)"
              inputMode="numeric"
              placeholder="0"
              autoComplete="off"
              error={errors.nilai_klaim?.message}
              disabled={busy}
              {...register('nilai_klaim')}
            />
            <Field
              id="pembayaran-sebelumnya"
              label="Nilai Pembayaran Sebelumnya (Rp)"
              inputMode="numeric"
              placeholder="0"
              autoComplete="off"
              hint="Bila diisi, Sisa dihitung dari angka ini."
              error={errors.pembayaran_sebelumnya?.message}
              disabled={busy}
              {...register('pembayaran_sebelumnya')}
            />
            <Field
              id="pembayaran"
              label="Pembayaran (Rp)"
              inputMode="numeric"
              placeholder="0"
              autoComplete="off"
              error={errors.pembayaran?.message}
              disabled={busy}
              {...register('pembayaran')}
            />
          </div>

          {/*
            Sisa ditampilkan beserta RUMUS yang dipakai, bukan angkanya saja.

            Cabang kedua aturan ini mengabaikan Pembayaran, dan itu tampak keliru bagi siapa
            pun yang membacanya — termasuk petugas. Menyebutkan rumusnya membuat angka yang
            muncul dapat dipertanggungjawabkan alih-alih tampak salah hitung.
          */}
          <div className="rounded-kartu border border-slate-200 bg-slate-50 p-4">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <span className="text-sm font-medium text-slate-700">Sisa</span>
              <span
                className={
                  'font-mono text-xl font-semibold tabular-nums ' +
                  (computedRemainder < 0 ? 'text-red-700' : 'text-slate-900')
                }
              >
                Rp {formatMoney(computedRemainder)}
              </span>
            </div>
            <p className="mt-1 text-xs text-slate-500">
              {(parseMoney(watched[1] ?? '') ?? 0) === 0
                ? 'Nilai Klaim − Pembayaran, karena pembayaran sebelumnya nol.'
                : 'Nilai Klaim − Nilai Pembayaran Sebelumnya, mengikuti aturan sistem lama.'}
            </p>
            {computedRemainder < 0 && (
              <p className="mt-2 text-xs font-medium text-red-700">
                Sisa negatif — pembayaran melampaui nilai klaim. Periksa kembali angkanya
                sebelum menyimpan.
              </p>
            )}
          </div>
        </div>
      </section>

      {/* ── Keterangan ────────────────────────────────────────────────────────── */}
      <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
        <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
          <h3 className="text-base font-semibold text-slate-900">Keterangan</h3>
        </div>
        <div className="grid gap-5 p-5 sm:grid-cols-2">
          <Field
            id="keterangan"
            label="Keterangan"
            placeholder="Contoh: pengembalian sebagian tahap 1"
            maxLength={MAX_REMARK}
            autoComplete="off"
            error={errors.keterangan?.message}
            disabled={busy}
            {...register('keterangan')}
          />
          <Field
            id="posisi-kasus"
            label="Posisi Kasus"
            placeholder="Contoh: dalam proses litigasi"
            maxLength={MAX_CASE_POSITION}
            autoComplete="off"
            error={errors.posisi_kasus?.message}
            disabled={busy}
            {...register('posisi_kasus')}
          />
        </div>
      </section>

      {/* ── Polis ─────────────────────────────────────────────────────────────── */}
      <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
        <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
          <h3 className="text-base font-semibold text-slate-900">Polis Acuan (opsional)</h3>
          <p className="mt-1 max-w-2xl text-sm text-slate-600">
            Bila diisi, identitas lini bisnis, cabang, agen, dan marketing dicari dari nomor
            ini dan ikut tersimpan. Batch tetap dapat disimpan tanpanya.
          </p>
        </div>

        <div className="space-y-4 p-5">
          <div className="flex flex-wrap items-end gap-3">
            <div className="min-w-[16rem] flex-1">
              <Field
                id="nomor-polis"
                label="Nomor Polis"
                placeholder="Opsional"
                maxLength={MAX_POLICY_NO}
                autoComplete="off"
                error={errors.nomor_polis?.message}
                disabled={busy}
                {...register('nomor_polis')}
              />
            </div>
            {/*
              Nilainya dibaca dari form, bukan dari DOM. Membaca elemen langsung akan
              melewati state React Hook Form — dan nilai yang dikirim pencarian lalu dapat
              berbeda dari nilai yang tersimpan saat batch disimpan.
            */}
            <Button
              tone="kedua"
              disabled={busy || lookup.isPending || policyNo.trim() === ''}
              onClick={() => lookup.mutate(policyNo.trim())}
            >
              {lookup.isPending ? 'Mencari…' : 'Cari identitas'}
            </Button>
          </div>

          {lookup.isError && <PolicyErrorMessage error={lookup.error} />}

          {policy && (
            <dl className="grid gap-3 rounded-kartu border border-emerald-200 bg-emerald-50 p-4 text-sm sm:grid-cols-4">
              {[
                ['Lini bisnis', policy.id_lini_bisnis],
                ['Cabang', policy.id_cabang],
                ['Agen', policy.id_agen],
                ['Marketing', policy.id_marketing],
              ].map(([label, value]) => (
                <div key={label}>
                  <dt className="text-xs text-emerald-800">{label}</dt>
                  <dd className="mt-0.5 font-mono text-sm font-medium text-emerald-950">
                    {value || '—'}
                  </dd>
                </div>
              ))}
            </dl>
          )}
        </div>
      </section>

      {/* ── Bukti bayar ───────────────────────────────────────────────────────── */}
      <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
        <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
          <h3 className="text-base font-semibold text-slate-900">Bukti Bayar (opsional)</h3>
          <p className="mt-1 text-sm text-slate-600">
            Pindaian bukti transfer, paling besar 5 MB.
          </p>
        </div>

        <div className="space-y-4 p-5">
          <label
            htmlFor="bukti-bayar"
            className="block text-sm font-medium text-slate-700"
          >
            Berkas bukti bayar
          </label>
          <input
            id="bukti-bayar"
            type="file"
            disabled={busy || upload.isPending}
            className="block w-full text-sm text-slate-600 file:mr-4 file:rounded-kontrol file:border-0 file:bg-slate-100 file:px-4 file:py-2 file:text-sm file:font-medium file:text-slate-700 hover:file:bg-slate-200"
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (!file) return
              upload.mutate(
                { berkas: file },
                {
                  onSuccess: (answer) =>
                    setPaymentProof({ id: answer.id_dokumen, nama: answer.nama_berkas }),
                },
              )
              event.target.value = ''
            }}
          />

          {upload.isPending && <p className="text-sm text-slate-500">Mengunggah…</p>}
          {upload.isError && <UploadErrorMessage error={upload.error} />}

          {paymentProof && (
            <div className="flex flex-wrap items-center justify-between gap-2 rounded-kartu border border-emerald-200 bg-emerald-50 px-4 py-3">
              <div>
                <p className="text-sm font-medium text-emerald-900">{paymentProof.nama}</p>
                <p className="mt-0.5 font-mono text-xs text-emerald-800">
                  ID dokumen: {paymentProof.id}
                </p>
              </div>
              <Button tone="halus" onClick={() => setPaymentProof(null)} disabled={busy}>
                Lepas
              </Button>
            </div>
          )}
        </div>
      </section>

      <ClaimLineUpload claimLine={claimLine} onChange={setClaimLine} disabled={busy} />

      <div className="flex flex-wrap gap-2 border-t border-slate-200 pt-5">
        <Button type="submit" tone="utama" disabled={busy}>
          {busy ? 'Menyimpan…' : 'Transfer Recovery'}
        </Button>
        <Button
          tone="halus"
          disabled={busy}
          onClick={() => {
            reset()
            setClaimLine([])
            setPaymentProof(null)
            lookup.reset()
            save.reset()
          }}
        >
          Kosongkan isian
        </Button>
      </div>
    </form>
  )
}

/** Tabel ringkas batch yang baru tersimpan, ditampilkan pemanggil. */
export function SavedRecoveryTable({ rows }: { rows: Recovery[] }) {
  return (
    <DataTable
      columns={[
        {
          key: 'nomor_batch',
          title: 'Batch',
          width: '6rem',
          value: (row) => String(row.nomor_batch).padStart(12, '0'),
          render: (row) => (
            <span className="font-mono text-xs font-medium text-slate-800">
              {row.nomor_batch}
            </span>
          ),
        },
        {
          key: 'nama_principal',
          title: 'Principal',
          value: (row) => row.nama_principal,
        },
        { key: 'tahun', title: 'Tahun', width: '6rem', value: (row) => row.tahun },
        {
          key: 'nilai_klaim',
          title: 'Nilai Klaim (Rp)',
          width: '11rem',
          alignRight: true,
          value: (row) => String(row.nilai_klaim).padStart(20, '0'),
          render: (row) => (
            <span className="font-mono text-xs tabular-nums">{formatMoney(row.nilai_klaim)}</span>
          ),
        },
        {
          key: 'sisa',
          title: 'Sisa (Rp)',
          width: '11rem',
          alignRight: true,
          value: (row) => String(row.sisa).padStart(20, '0'),
          render: (row) => (
            <span
              className={
                'font-mono text-xs font-medium tabular-nums ' +
                (row.sisa < 0 ? 'text-red-700' : 'text-slate-900')
              }
            >
              {formatMoney(row.sisa)}
            </span>
          ),
        },
      ]}
      rows={rows}
      rowKey={(row) => String(row.nomor_batch)}
      title="Tersimpan pada sesi ini"
      description="Daftar ini hanya berisi batch yang Anda simpan sejak layar dibuka — sistem lama tidak menyediakan cara membaca kembali batch yang sudah tercatat."
      emptyMessage="Belum ada batch yang disimpan pada sesi ini."
      searchLabel="Cari principal"
    />
  )
}

// ── Pesan galat ───────────────────────────────────────────────────────────────────

function SaveErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Batch belum tersimpan. Isian Anda masih ada — periksa koneksi lalu simpan lagi."
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

type MessageContent = { title: string; description: string; tone: ErrorTone }

function parseSave(error: APIError): MessageContent | null {
  switch (error.kode) {
    case ErrorCode.validationFailed:
      return Object.keys(error.violations()).length > 0
        ? null
        : { title: 'Isian belum benar', description: error.message, tone: 'penolakan' }

    case ErrorCode.recoveryBatchTaken:
      return {
        title: 'Nomor batch baru saja dipakai',
        description:
          'Petugas lain menyimpan lebih dulu. Tekan Transfer Recovery sekali lagi — isian Anda tidak hilang.',
        tone: 'penolakan',
      }

    case ErrorCode.portalNotStated:
    case ErrorCode.portalUnknown:
      return {
        title: 'Portal entitas belum dipilih',
        description: 'Pilih portal entitas di bilah atas halaman, lalu simpan lagi.',
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
      return { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
  }
}

function PolicyErrorMessage({ error }: { error: unknown }) {
  if (error instanceof APIError && error.kode === ErrorCode.recoveryPolicyNotFound) {
    return (
      <ErrorMessage
        title="Nomor polis tidak ditemukan"
        description="Periksa kembali nomornya. Batch tetap dapat disimpan tanpa identitas polis — keempat kolomnya akan kosong."
        tone="penolakan"
      />
    )
  }
  return (
    <ErrorMessage
      title="Identitas polis belum dapat dicari"
      description="Pencarian menyeberang ke basis data lain dan sedang tidak dapat ditembak. Batch tetap dapat disimpan."
      tone="gangguan"
    />
  )
}

function UploadErrorMessage({ error }: { error: unknown }) {
  if (error instanceof APIError && error.kode === ErrorCode.validationFailed) {
    const violation = error.violations()['bukti_bayar']
    return (
      <ErrorMessage
        title="Berkas ditolak"
        description={violation ?? error.message}
        tone="penolakan"
      />
    )
  }
  return (
    <ErrorMessage
      title="Bukti bayar gagal diunggah"
      description="Batch belum tersimpan. Coba unggah ulang; bila tetap gagal, simpan batch tanpa bukti bayar."
      tone="gangguan"
    />
  )
}
