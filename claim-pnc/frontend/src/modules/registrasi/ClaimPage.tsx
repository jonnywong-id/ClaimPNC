import { forwardRef, useEffect, useState, type InputHTMLAttributes, type ReactNode } from 'react'
import { useFieldArray, useForm, type Control, type UseFormRegister } from 'react-hook-form'
import { Link, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { FormField } from '@/components/FormField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatPercent, formatRupiah, formatDate, rupiahToCents, centsToRupiah } from '@/components/format'

import {
  useFlow,
  useClaim,
  useCompleteStage,
  useSaveRegister,
  violationsFrom,
  messagesByField,
} from './api'
import { StagePath } from './StagePath'
import type { Claim, Violation, RegisterRequest, Task } from './types'

/** Pengenal tahap Input Register, satu-satunya tahap yang isiannya dimiliki modul ini. */
const TAHAP_INPUT_REGISTER = 'input-register'

/**
 * Layar kerja satu klaim.
 *
 * Ia menampilkan di mana klaim berada, apa yang menantinya, dan tindakan yang tersedia
 * pada tahap sekarang. Hanya satu tahap yang punya formulir di sini — Input Register.
 * Tahap lain ditutup dengan satu tindakan, karena isiannya milik modul lain (`B-5`,
 * `B-8`, `B-11`) yang belum dibangun.
 */
export function ClaimPage() {
  const { claimID } = useParams<{ claimID: string }>()
  const klaim = useClaim(claimID)
  const flow = useFlow()

  if (klaim.isPending) {
    return <Frame><p className="text-sm text-slate-500">Memuat klaim…</p></Frame>
  }

  if (klaim.isError) {
    return (
      <Frame>
        <ErrorMessage title="Klaim tidak dapat dimuat" note={errorMessage(klaim.error)} tone="gangguan" />
      </Frame>
    )
  }

  const content = klaim.data
  const atInputRegister = content.klaim.tahap_kini === TAHAP_INPUT_REGISTER

  return (
    <Frame>
      <ClaimHeader klaim={content.klaim} />

      {flow.data && content.jalur && content.jalur.length > 0 && (
        <div className="mt-6">
          <StagePath tahap={flow.data.tahap} jalur={content.jalur} currentStage={content.klaim.tahap_kini} />
        </div>
      )}

      {content.klaim.tahap_kini === '' && (
        <p className="mt-6 rounded border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700">
          Klaim ini sudah selesai pada alur Register. Tidak ada tugas yang menunggu.
        </p>
      )}

      {content.tugas && !atInputRegister && <StageActions tugas={content.tugas} />}
      {content.tugas && atInputRegister && <FormRegister klaim={content.klaim} tugas={content.tugas} />}

      {!content.tugas && content.klaim.tahap_kini !== '' && (
        <p className="mt-6 rounded border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          Klaim berada di tahap {content.klaim.tahap_kini}, tetapi tidak ada tugas terbuka yang
          dapat Anda kerjakan. Tugasnya mungkin milik rekan kerja, atau berada di antrean
          bersama yang belum Anda ambil.
        </p>
      )}
    </Frame>
  )
}

function Frame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <Link to="/registrasi" className="text-sm text-slate-600 underline">
        ← Kembali ke inbox
      </Link>
      <div className="mt-4">{children}</div>
    </div>
  )
}

function ClaimHeader({ klaim }: { klaim: Claim }) {
  return (
    <header className="border-b border-slate-200 pb-4">
      <h1 className="text-xl font-semibold text-slate-900">
        {klaim.nomor || 'Klaim belum bernomor'}
      </h1>
      <dl className="mt-3 grid gap-x-8 gap-y-3 sm:grid-cols-3">
        <Row label="Nomor polis" value={klaim.polis.nomor} />
        <Row label="Lini bisnis" value={`${klaim.polis.nama_lini} (${klaim.polis.lini})`} />
        <Row label="Tertanggung" value={klaim.polis.nama_tertanggung || '—'} />
        <Row
          label="Periode polis"
          value={`${formatDate(klaim.polis.mulai_pertanggungan)} – ${formatDate(klaim.polis.akhir_pertanggungan)}`}
        />
        <Row label="Status Proses" value={klaim.status_proses} />
        <Row label="Status Klaim" value={klaim.status_klaim || '—'} />
      </dl>
    </header>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="text-sm text-slate-900">{value}</dd>
    </div>
  )
}

/**
 * Tindakan penutup untuk tahap yang isiannya milik modul lain.
 *
 * Nama tindakannya datang dari SERVER — ia adalah `tindakan_keluar` tahap yang
 * bersangkutan, bukan teks yang ditebak layar. Bila layar mengirim tindakan yang bukan
 * penutup tahap itu, server menolaknya (`ADR-0023`).
 */
function StageActions({ tugas }: { tugas: Task }) {
  const done = useCompleteStage()

  return (
    <section className="mt-6 rounded border border-slate-200 p-4">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">
        Tahap {tugas.nama_tahap}
      </h2>
      <p className="mt-2 text-sm text-slate-600">
        Isian tahap ini dimiliki modul lain yang belum dibangun. Yang tersedia di sini
        adalah memindahkan klaim ke tahap berikutnya sesuai alur Register.
      </p>

      <div className="mt-4 flex flex-wrap gap-3">
        <button
          type="button"
          disabled={done.isPending}
          onClick={() => done.mutate({ taskID: tugas.id, action: tugas.tindakan_keluar })}
          className="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {done.isPending ? 'Memproses…' : `Selesaikan (${tugas.tindakan_keluar})`}
        </button>
        <button
          type="button"
          disabled={done.isPending}
          onClick={() =>
            done.mutate({ taskID: tugas.id, action: tugas.tindakan_keluar, kembali: true })
          }
          className="rounded border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
        >
          Kembali (Back)
        </button>
      </div>

      {done.isError && (
        <div className="mt-3">
          <ErrorMessage title="Tahap tidak dapat ditutup" note={errorMessage(done.error)} tone="penolakan" />
        </div>
      )}

      {done.isSuccess && done.data.jejak_keputusan && done.data.jejak_keputusan.length > 0 && (
        <DecisionTrace trace={done.data.jejak_keputusan} />
      )}
    </section>
  )
}

/**
 * Jejak keputusan alur yang baru saja dilewati.
 *
 * Di sistem lama, klaim berpindah tahap tanpa penjelasan apa pun yang terlihat petugas —
 * dan pada satu keputusan, tujuannya bahkan bergantung pada peran orang yang menekan
 * tombol. Menampilkan cabang yang dipilih membuat perpindahan itu dapat dijelaskan tanpa
 * membaca kode.
 */
function DecisionTrace({ trace }: { trace: string[] }) {
  return (
    <div className="mt-4 rounded border border-slate-200 bg-slate-50 p-3">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
        Percabangan yang dilewati
      </p>
      <ul className="mt-2 space-y-1 text-sm text-slate-700">
        {trace.map((j) => (
          <li key={j}>{j}</li>
        ))}
      </ul>
    </div>
  )
}

// ── Formulir Input Register ────────────────────────────────────────────────────────

type SpreadingInput = {
  jenis_treaty: string
  nama: string
  share: string
  objek_fac_offer: string
  dihapus: boolean
}

type CoverageInput = {
  id: string
  penyebab_kerugian: string
  tsi: string
  spreading: SpreadingInput[]
}

type InsuredItemInput = {
  id: string
  nama: string
  lokasi: string
  coverage: CoverageInput[]
}

type RegisterFormValues = {
  tanggal_kejadian: string
  tanggal_lapor: string
  tanggal_terima_dokumen: string
  lokasi: string
  kronologi: string
  pelapor_nama: string
  pelapor_telepon: string
  pelapor_email: string
  pelapor_alamat: string
  pelapor_hubungan: string
  pelapor_hubungan_lainnya: string
  nilai_estimasi: string
  mata_uang: string
  nomor_slik: string
  user_teknis: string
  rcv_id: string
  ex_gratia: boolean
  status_pucl: string
  transfer_compliance: boolean
  objek: InsuredItemInput[]
}

/** Kode hubungan pelapor yang menuntut keterangan tambahan (langkah 28 sistem lama). */
const HUBUNGAN_LAIN_LAIN = '7'

/**
 * Formulir tahap Input Register.
 *
 * # Kenapa tidak ada aturan bisnis di sini
 *
 * Seluruh aturan — urutan tanggal, periode polis, duplikasi, kelengkapan, total
 * spreading — ditegakkan SERVER, dan layar ini hanya menampilkan jawabannya. Itu bukan
 * kemalasan: `ADR-0023` menetapkan aturan yang berada di layar dapat dipintas lewat
 * pemanggilan langsung ke API, dan aturan yang ditulis di dua tempat pasti berbeda cepat
 * atau lambat.
 *
 * Yang dilakukan layar adalah memastikan setiap pesan dari server menempel pada KOLOM
 * yang benar, supaya petugas tahu apa yang harus ia perbaiki.
 */
function FormRegister({ klaim, tugas }: { klaim: Claim; tugas: Task }) {
  const save = useSaveRegister()
  const [violations, setViolations] = useState<Violation[]>([])

  const { register, control, handleSubmit, watch, reset } = useForm<RegisterFormValues>({
    defaultValues: fromClaim(klaim),
  })

  // Klaim yang dimuat ulang — misalnya setelah tombol Back — mengisi ulang formulir.
  useEffect(() => {
    reset(fromClaim(klaim))
  }, [klaim, reset])

  const objek = useFieldArray({ control, name: 'objek' })
  const fieldErrors = messagesByField(violations)
  const hubungan = watch('pelapor_hubungan')

  const submit = (content: RegisterFormValues, kembali: boolean) => {
    setViolations([])
    save.mutate(toRequest(content, tugas.id, kembali), {
      onError: (failure) => setViolations(violationsFrom(failure)),
    })
  }

  return (
    <form className="mt-6 space-y-6" onSubmit={handleSubmit((content) => submit(content, false))}>
      {save.isError && (
        <ErrorMessage
          title={violations.length > 0 ? 'Klaim belum dapat disimpan' : 'Penyimpanan gagal'}
          note={errorMessage(save.error)}
          tone={violations.length > 0 ? 'penolakan' : 'gangguan'}
        />
      )}

      {violations.length > 1 && (
        <div className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          <p className="font-medium">{violations.length} ketentuan belum terpenuhi:</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            {violations.map((p, i) => (
              <li key={`${p.kode}-${i}`}>{p.pesan}</li>
            ))}
          </ul>
        </div>
      )}

      {save.isSuccess && save.data.large_loss && (
        <div className="rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          <p className="font-medium">Notice of Large Losses diterbitkan.</p>
          <p className="mt-1">
            Nilai estimasi melampaui ambang, sehingga pemberitahuan ke Underwriting dan
            pimpinan ikut dicatat.
          </p>
        </div>
      )}

      <Section title="Data kejadian">
        <div className="grid gap-4 sm:grid-cols-3">
          <FormField id="tanggal_kejadian" label="Tanggal kejadian" type="date"
            failure={fieldErrors['tanggal_kejadian']} {...register('tanggal_kejadian')} />
          <FormField id="tanggal_lapor" label="Tanggal lapor" type="date"
            failure={fieldErrors['tanggal_lapor']} {...register('tanggal_lapor')} />
          <FormField id="tanggal_terima_dokumen" label="Tanggal terima dokumen" type="date"
            failure={fieldErrors['tanggal_terima_dokumen']} {...register('tanggal_terima_dokumen')} />
        </div>

        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <FormField id="lokasi" label="Lokasi kejadian" failure={fieldErrors['lokasi']} {...register('lokasi')} />
          <FormField id="user_teknis" label="PIC Teknik" {...register('user_teknis')} />
        </div>

        <div className="mt-4">
          <label htmlFor="kronologi" className="block text-sm font-medium text-slate-700">
            Kronologi
          </label>
          <textarea
            id="kronologi"
            rows={3}
            className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none"
            {...register('kronologi')}
          />
        </div>
      </Section>

      <Section title="Pelapor">
        <div className="grid gap-4 sm:grid-cols-2">
          <FormField id="pelapor_nama" label="Nama pelapor" {...register('pelapor_nama')} />
          <FormField id="pelapor_telepon" label="Telepon" {...register('pelapor_telepon')} />
          <FormField id="pelapor_email" label="Email" type="email" {...register('pelapor_email')} />
          <FormField id="pelapor_alamat" label="Alamat" {...register('pelapor_alamat')} />
          <FormField id="pelapor_hubungan" label="Kode hubungan dengan tertanggung"
            inputMode="numeric" {...register('pelapor_hubungan')} />
          {hubungan === HUBUNGAN_LAIN_LAIN && (
            <FormField id="hubungan_lainnya" label="Sebutkan hubungannya"
              failure={fieldErrors['hubungan_lainnya']} {...register('pelapor_hubungan_lainnya')} />
          )}
        </div>
      </Section>

      <Section title="Nilai dan kelengkapan">
        <div className="grid gap-4 sm:grid-cols-3">
          <FormField id="nilai_estimasi" label="Nilai estimasi klaim" inputMode="decimal"
            failure={fieldErrors['nilai_estimasi']} {...register('nilai_estimasi')} />
          <FormField id="mata_uang" label="Mata uang" {...register('mata_uang')} />
          <FormField id="nomor_slik" label="Nomor SLIK"
            failure={fieldErrors['nomor_slik']} {...register('nomor_slik')} />
        </div>

        <div className="mt-4 grid gap-4 sm:grid-cols-3">
          <FormField id="rcv_id" label="ID Receive Document" {...register('rcv_id')} />
          <FormField id="status_pucl" label="Status RCL/PUCL" inputMode="numeric" {...register('status_pucl')} />
          <div className="flex items-end gap-6 pb-2">
            <Toggle label="Ex gratia" {...register('ex_gratia')} />
            <Toggle label="Transfer Compliance" {...register('transfer_compliance')} />
          </div>
        </div>

        {fieldErrors['nomor_polis'] && (
          <p className="mt-3 text-sm text-red-700">{fieldErrors['nomor_polis']}</p>
        )}
      </Section>

      <Section title="Objek pertanggungan">
        {fieldErrors['objek'] && <p className="mb-3 text-sm text-red-700">{fieldErrors['objek']}</p>}
        {fieldErrors['penyebab_kerugian'] && (
          <p className="mb-3 text-sm text-red-700">{fieldErrors['penyebab_kerugian']}</p>
        )}
        {fieldErrors['spreading'] && <p className="mb-3 text-sm text-red-700">{fieldErrors['spreading']}</p>}

        <div className="space-y-4">
          {objek.fields.map((f, i) => (
            <InsuredItemEditor
              key={f.id}
              index={i}
              control={control}
              register={register}
              onRemove={() => objek.remove(i)}
            />
          ))}
        </div>

        <button
          type="button"
          onClick={() => objek.append({ id: '', nama: '', lokasi: '', coverage: [] })}
          className="mt-4 rounded border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
        >
          Tambah objek
        </button>
      </Section>

      <SpreadingSummary values={watch('objek')} />

      <div className="flex flex-wrap gap-3">
        <button
          type="submit"
          disabled={save.isPending}
          className="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {save.isPending ? 'Menyimpan…' : 'Simpan dan terbitkan nomor klaim'}
        </button>
        <button
          type="button"
          disabled={save.isPending}
          onClick={handleSubmit((content) => submit(content, true))}
          className="rounded border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
        >
          Kembali (Back)
        </button>
      </div>

      <p className="text-xs text-slate-500">
        Nomor klaim terbit hanya bila seluruh ketentuan terpenuhi, dan setelah terbit
        tidak dapat ditarik kembali. Tombol Kembali menyimpan isian lalu mengembalikan
        klaim ke tahap View Polis tanpa menerbitkan nomor.
      </p>
    </form>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded border border-slate-200 p-4">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">{title}</h2>
      <div className="mt-3">{children}</div>
    </section>
  )
}

/**
 * Sakelar adalah satu kotak centang berlabel.
 *
 * Ref diteruskan supaya `register` dari React Hook Form dapat memegang elemennya —
 * tanpa itu, nilainya tidak pernah ikut terkirim.
 */
const Toggle = forwardRef<
  HTMLInputElement,
  { label: string } & InputHTMLAttributes<HTMLInputElement>
>(function Toggle({ label, ...remaining }, ref) {
  return (
    <label className="flex items-center gap-2 text-sm text-slate-700">
      <input ref={ref} type="checkbox" className="h-4 w-4 rounded border-slate-300" {...remaining} />
      {label}
    </label>
  )
})

function InsuredItemEditor({
  index,
  control,
  register,
  onRemove,
}: {
  index: number
  control: Control<RegisterFormValues>
  register: UseFormRegister<RegisterFormValues>
  onRemove: () => void
}) {
  const coverage = useFieldArray({ control, name: `objek.${index}.coverage` })

  return (
    <div className="rounded border border-slate-200 bg-slate-50 p-3">
      <div className="grid gap-3 sm:grid-cols-3">
        <FormField id={`objek-${index}-id`} label="Kode objek" {...register(`objek.${index}.id`)} />
        <FormField id={`objek-${index}-nama`} label="Nama objek" {...register(`objek.${index}.nama`)} />
        <FormField id={`objek-${index}-lokasi`} label="Lokasi objek" {...register(`objek.${index}.lokasi`)} />
      </div>

      <div className="mt-3 space-y-3">
        {coverage.fields.map((f, j) => (
          <CoverageEditor
            key={f.id}
            itemIndex={index}
            index={j}
            control={control}
            register={register}
            onRemove={() => coverage.remove(j)}
          />
        ))}
      </div>

      <div className="mt-3 flex gap-3">
        <button
          type="button"
          onClick={() => coverage.append({ id: '', penyebab_kerugian: '', tsi: '', spreading: [] })}
          className="rounded border border-slate-300 bg-white px-3 py-1 text-sm text-slate-700 hover:bg-slate-100"
        >
          Tambah coverage
        </button>
        <button
          type="button"
          onClick={onRemove}
          className="rounded border border-red-200 bg-white px-3 py-1 text-sm text-red-700 hover:bg-red-50"
        >
          Hapus objek
        </button>
      </div>
    </div>
  )
}

function CoverageEditor({
  itemIndex,
  index,
  control,
  register,
  onRemove,
}: {
  itemIndex: number
  index: number
  control: Control<RegisterFormValues>
  register: UseFormRegister<RegisterFormValues>
  onRemove: () => void
}) {
  const nama = `objek.${itemIndex}.coverage.${index}` as const
  const spreading = useFieldArray({ control, name: `${nama}.spreading` })

  return (
    <div className="rounded border border-slate-200 bg-white p-3">
      <div className="grid gap-3 sm:grid-cols-3">
        <FormField id={`${nama}-id`} label="Kode coverage" {...register(`${nama}.id`)} />
        <FormField id={`${nama}-sebab`} label="Penyebab kerugian" {...register(`${nama}.penyebab_kerugian`)} />
        <FormField id={`${nama}-tsi`} label="TSI" inputMode="decimal" {...register(`${nama}.tsi`)} />
      </div>

      <table className="mt-3 w-full border-collapse text-sm">
        <caption className="sr-only">Spreading reasuransi</caption>
        <thead>
          <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-500">
            <th scope="col" className="py-1 pr-2 font-medium">Treaty</th>
            <th scope="col" className="py-1 pr-2 font-medium">Nama</th>
            <th scope="col" className="py-1 pr-2 font-medium">Share %</th>
            <th scope="col" className="py-1 pr-2 font-medium">Objek Fac Offer</th>
            <th scope="col" className="py-1 font-medium">Hapus</th>
          </tr>
        </thead>
        <tbody>
          {spreading.fields.map((f, n) => (
            <tr key={f.id} className="border-b border-slate-100">
              <td className="py-1 pr-2">
                <input className="w-24 rounded border border-slate-300 px-2 py-1"
                  aria-label="Jenis treaty" {...register(`${nama}.spreading.${n}.jenis_treaty`)} />
              </td>
              <td className="py-1 pr-2">
                <input className="w-full rounded border border-slate-300 px-2 py-1"
                  aria-label="Nama treaty" {...register(`${nama}.spreading.${n}.nama`)} />
              </td>
              <td className="py-1 pr-2">
                <input className="w-28 rounded border border-slate-300 px-2 py-1" inputMode="decimal"
                  aria-label="Share persen" {...register(`${nama}.spreading.${n}.share`)} />
              </td>
              <td className="py-1 pr-2">
                <input className="w-full rounded border border-slate-300 px-2 py-1"
                  aria-label="Objek Fac Offer" {...register(`${nama}.spreading.${n}.objek_fac_offer`)} />
              </td>
              <td className="py-1">
                <button type="button" onClick={() => spreading.remove(n)}
                  className="rounded border border-red-200 px-2 py-1 text-xs text-red-700 hover:bg-red-50">
                  Hapus
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="mt-2 flex gap-3">
        <button
          type="button"
          onClick={() => spreading.append({ jenis_treaty: '', nama: '', share: '', objek_fac_offer: '', dihapus: false })}
          className="rounded border border-slate-300 px-3 py-1 text-sm text-slate-700 hover:bg-slate-100"
        >
          Tambah spreading
        </button>
        <button
          type="button"
          onClick={onRemove}
          className="rounded border border-red-200 px-3 py-1 text-sm text-red-700 hover:bg-red-50"
        >
          Hapus coverage
        </button>
      </div>
    </div>
  )
}

/**
 * Ringkasan total share, ditampilkan sebelum petugas menekan Simpan.
 *
 * Ia BUKAN validasi — penolakan tetap milik server. Ia perhitungan yang sama yang sudah
 * ada di layar, ditunjukkan lebih awal, supaya petugas tidak perlu menekan Simpan untuk
 * tahu totalnya belum 100%.
 */
function SpreadingSummary({ values }: { values: InsuredItemInput[] | undefined }) {
  if (!values || values.length === 0) return null

  let totalE4 = 0
  let count = 0
  let totalTSI = 0

  for (const o of values) {
    for (const c of o.coverage ?? []) {
      totalTSI += rupiahToCents(c.tsi || '0') || 0
      for (const s of c.spreading ?? []) {
        if (s.dihapus) continue
        const num = Number((s.share || '0').replace(',', '.'))
        if (!Number.isNaN(num)) totalE4 += Math.round(num * 10_000)
        count += 1
      }
    }
  }

  const hundred = totalE4 >= 999_999 && totalE4 <= 1_000_001

  return (
    <section className="rounded border border-slate-200 p-4 text-sm">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">Ringkasan</h2>
      <dl className="mt-3 grid gap-x-8 gap-y-2 sm:grid-cols-3">
        <Row label="Baris spreading" value={String(count)} />
        <Row label="Total TSI" value={formatRupiah(totalTSI)} />
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Total share
          </dt>
          <dd className={hundred ? 'text-sm text-slate-900' : 'text-sm font-medium text-red-700'}>
            {formatPercent(totalE4)}
            {!hundred && ' — belum 100%'}
          </dd>
        </div>
      </dl>
    </section>
  )
}

// ── Terjemahan antara bentuk layar dan bentuk API ──────────────────────────────────

function fromClaim(klaim: Claim): RegisterFormValues {
  return {
    tanggal_kejadian: klaim.tanggal_kejadian,
    tanggal_lapor: klaim.tanggal_lapor,
    tanggal_terima_dokumen: klaim.tanggal_terima_dokumen,
    lokasi: klaim.lokasi,
    kronologi: klaim.kronologi,
    pelapor_nama: klaim.pelapor.nama,
    pelapor_telepon: klaim.pelapor.telepon,
    pelapor_email: klaim.pelapor.email,
    pelapor_alamat: klaim.pelapor.alamat,
    pelapor_hubungan: klaim.pelapor.hubungan ? String(klaim.pelapor.hubungan) : '',
    pelapor_hubungan_lainnya: klaim.pelapor.hubungan_lainnya,
    nilai_estimasi: centsToRupiah(klaim.nilai_estimasi_sen),
    mata_uang: klaim.mata_uang || klaim.polis.mata_uang || 'IDR',
    nomor_slik: klaim.nomor_slik,
    user_teknis: klaim.user_teknis,
    rcv_id: klaim.rcv_id,
    ex_gratia: klaim.ex_gratia,
    status_pucl: klaim.status_pucl ? String(klaim.status_pucl) : '0',
    transfer_compliance: klaim.transfer_compliance,
    objek: klaim.objek.map((o) => ({
      id: o.id,
      nama: o.nama,
      lokasi: o.lokasi,
      coverage: o.coverage.map((c) => ({
        id: c.id,
        penyebab_kerugian: c.penyebab_kerugian,
        tsi: centsToRupiah(c.tsi_sen),
        spreading: c.spreading.map((s) => ({
          jenis_treaty: s.jenis_treaty,
          nama: s.nama,
          share: (s.share / 10_000).toString(),
          objek_fac_offer: s.objek_fac_offer,
          dihapus: s.dihapus,
        })),
      })),
    })),
  }
}

function toRequest(content: RegisterFormValues, taskID: string, kembali: boolean): RegisterRequest {
  return {
    tugas_id: taskID,
    tanggal_kejadian: content.tanggal_kejadian,
    tanggal_lapor: content.tanggal_lapor,
    tanggal_terima_dokumen: content.tanggal_terima_dokumen,
    lokasi: content.lokasi,
    kronologi: content.kronologi,
    pelapor: {
      nama: content.pelapor_nama,
      telepon: content.pelapor_telepon,
      email: content.pelapor_email,
      alamat: content.pelapor_alamat,
      hubungan: Number(content.pelapor_hubungan) || 0,
      hubungan_lainnya: content.pelapor_hubungan_lainnya,
    },
    nilai_estimasi_sen: rupiahToCents(content.nilai_estimasi) || 0,
    mata_uang: content.mata_uang,
    nomor_slik: content.nomor_slik,
    ex_gratia: content.ex_gratia,
    user_teknis: content.user_teknis,
    rcv_id: content.rcv_id,
    status_pucl: Number(content.status_pucl) || 0,
    transfer_compliance: content.transfer_compliance,
    kembali,
    objek: (content.objek ?? []).map((o) => ({
      id: o.id,
      nama: o.nama,
      lokasi: o.lokasi,
      coverage: (o.coverage ?? []).map((c) => ({
        id: c.id,
        penyebab_kerugian: c.penyebab_kerugian,
        tsi_sen: rupiahToCents(c.tsi) || 0,
        spreading: (c.spreading ?? []).map((s) => ({
          jenis_treaty: s.jenis_treaty,
          nama: s.nama,
          // Share dikirim sebagai persen dikali 10.000 — presisi yang dipakai server
          // untuk memvalidasi totalnya.
          share: Math.round((Number((s.share || '0').replace(',', '.')) || 0) * 10_000),
          dihapus: s.dihapus,
          objek_fac_offer: s.objek_fac_offer,
        })),
      })),
    })),
  }
}

function errorMessage(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
