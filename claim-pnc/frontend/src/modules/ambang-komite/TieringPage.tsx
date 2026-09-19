import { useState, type FormEvent } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { KomiteMode, type KomiteApprover, type KomiteTieringResponse } from '@/api/types'
import { WarningIcon, ScaleIcon } from '@/components/Icon'
import { Field } from '@/components/Field'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'
import { formatRupiah, parseRupiah } from '@/lib/money'

import { useThresholdList, useTiering } from './api'

/**
 * Layar Penjenjangan Komite.
 *
 * Menjawab satu pertanyaan: untuk nilai klaim sekian pada lini bisnis tertentu, **siapa
 * saja yang harus menyetujui, dan dalam urutan apa**.
 *
 * # Kenapa layar ini ada, padahal alur komitenya sendiri belum dibangun
 *
 * Aturan penjenjangan adalah bagian modul Komite yang menentukan **siapa berwenang
 * menyetujui uang**, dan ia sudah dapat dihitung sekarang karena masternya lengkap.
 * Yang belum ada adalah alur keputusannya — inbox, tombol setuju dan tolak, pencatatan
 * jejaknya — dan itu bergantung pada modul Estimasi (`B-5`) serta Penugasan (`B-6`) yang
 * belum dibangun.
 *
 * Menampilkan perhitungannya lebih dulu membuat aturannya **dapat diperiksa pengguna
 * bisnis sebelum ada satu klaim pun yang bergantung padanya**. Ketujuh kasus pada spesifikasi
 * B-7 dapat dicoba sendiri di layar ini, dan bila ada yang meleset, ia ketahuan sekarang
 * — bukan setelah alur keputusan dibangun di atasnya.
 *
 * # Kenapa hasilnya dapat ditautkan
 *
 * Perhitungan ini tidak mengubah apa pun, sehingga permintaannya berupa GET dan seluruh
 * isiannya ada di alamat halaman. Seseorang yang menemukan angka yang meragukan dapat
 * mengirimkan tautannya apa adanya kepada Work Owner.
 */
export function TieringPage() {
  const master = useThresholdList()

  const [typedValue, setTypedValue] = useState('')
  const [line, setLine] = useState('')
  const [applicant, setApplicant] = useState('')
  const [requested, setRequested] = useState<Requested | null>(null)
  const [fieldError, setFieldError] = useState<string | undefined>(undefined)

  // Mode ditentukan per PORTAL, bukan per lini, sehingga ia dibaca dari master — bukan
  // disimpulkan dari lini yang sedang dipilih.
  const singleApproverMode = master.data?.mode === KomiteMode.singleApprover

  const result = useTiering(
    requested?.value ?? '',
    requested?.line ?? '',
    requested?.applicant ?? '',
    requested !== null,
  )

  function compute(event: FormEvent) {
    event.preventDefault()

    // Diurai di layar, bukan di server: pemisah ribuan berarti berbeda antar bahasa —
    // titik adalah ribuan di Indonesia dan desimal di Inggris — sehingga penafsirannya
    // harus terjadi di tempat yang tahu bahasanya.
    const canonical = parseRupiah(typedValue)
    if (canonical === null) {
      setFieldError('Masukkan nilai klaim, misalnya 75.000.000.')
      setRequested(null)
      return
    }
    if (line === '') {
      setFieldError(undefined)
      setRequested(null)
      return
    }

    setFieldError(undefined)
    setRequested({ value: canonical, line, applicant: applicant.trim() })
  }

  const lineOptions = master.data?.lini ?? []

  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Komite</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Penjenjangan</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          Penjenjangan Komite
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Masukkan nilai klaim dan lini bisnisnya untuk melihat siapa saja yang harus
          menyetujui. Aturannya <strong>kumulatif</strong>: setiap jenjang yang ambang
          bawahnya sudah terlampaui ikut menyetujui.
        </p>
      </header>

      <form
        onSubmit={compute}
        className="mb-6 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
      >
        <div className="grid gap-4 sm:grid-cols-[1fr_auto] sm:items-end">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              id="nilai-klaim"
              label="Nilai klaim"
              inputMode="numeric"
              autoComplete="off"
              placeholder="75.000.000"
              value={typedValue}
              onChange={(e) => setTypedValue(e.target.value)}
              error={fieldError}
              hint="Boleh diketik dengan titik ribuan."
            />

            <div>
              <label
                htmlFor="lini-bisnis"
                className="mb-1.5 block text-sm font-medium text-slate-800"
              >
                Lini bisnis
              </label>
              <select
                id="lini-bisnis"
                value={line}
                onChange={(e) => setLine(e.target.value)}
                required
                disabled={master.isPending || lineOptions.length === 0}
                className={[
                  'w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2.5',
                  'text-sm text-slate-900',
                  'transition-[border-color,box-shadow] duration-150 ease-halus',
                  'hover:border-slate-400 disabled:bg-slate-50 disabled:text-slate-400',
                  'focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15',
                ].join(' ')}
              >
                <option value="">Pilih lini bisnis…</option>
                {lineOptions.map((l) => (
                  <option key={l} value={l}>
                    {l}
                  </option>
                ))}
              </select>
              {/*
                Daftarnya datang DARI MASTER, bukan dari daftar tetap di dalam kode
                (`D-15`). Lini yang ditambahkan ke master langsung muncul di sini tanpa
                rilis ulang.
              */}
              <p className="mt-1.5 text-xs text-slate-500">
                {master.isPending
                  ? 'Memuat daftar lini…'
                  : `${lineOptions.length} lini punya jenjang persetujuan.`}
              </p>
            </div>

            {/*
              Isian ini muncul pada SEMUA mode sejak Work Owner menetapkan pengecualian
              penginput berlaku di seluruh entitas (2026-09-18) — bukan hanya di Simasnet
              seperti sistem lama.
            */}
            <Field
              id="operator-pengaju"
              label="Operator ID pengaju"
              autoComplete="off"
              placeholder="ELLENSUPRIYATI"
              value={applicant}
              onChange={(e) => setApplicant(e.target.value)}
              hint={
                singleApproverMode
                  ? 'Dikecualikan dari calon penyetuju. Opsional.'
                  : 'Dikecualikan supaya tidak menyetujui pengajuannya sendiri. Opsional.'
              }
            />
          </div>

          <Button tone="utama" type="submit" disabled={result.isFetching}>
            <ScaleIcon className="h-4 w-4" />
            {result.isFetching ? 'Menghitung…' : 'Hitung'}
          </Button>
        </div>
      </form>

      {result.isError && <ComputeErrorMessage error={result.error} />}

      {result.data && <TieringResult result={result.data} />}

      {requested === null && !result.isError && <ExampleCases />}
    </div>
  )
}

/** Isian yang sudah dikirim ke server; berbeda dari isian yang masih diketik. */
type Requested = { value: string; line: string; applicant: string }

function TieringResult({ result }: { result: KomiteTieringResponse }) {
  const singleApprover = result.mode === KomiteMode.singleApprover

  return (
    <section className="rounded-kartu border border-slate-200 bg-white shadow-lembut">
      <header className="border-b border-slate-200 p-5">
        <h2 className="text-base font-semibold text-slate-900">
          {result.jumlah_jenjang === 0
            ? 'Tidak ada yang menyetujui'
            : singleApprover
              ? 'Satu penyetuju dipilih'
              : `${result.jumlah_jenjang} jenjang harus menyetujui`}
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          Klaim <strong className="tabular-nums">{formatRupiah(result.nilai)}</strong> pada
          lini <strong>{result.lini}</strong>
          {result.berpita_nilai && (
            <>
              {' '}
              — masuk <strong>pita {result.pita}</strong>, dan akumulasi berjalan di dalam
              pita itu saja
            </>
          )}
          .
        </p>
      </header>

      {result.tanpa_penyetuju ? (
        <div className="p-5">
          {/*
            Dua sebab yang sangat berbeda, dan membedakannya menentukan siapa yang harus
            bertindak: master yang tidak menjangkau nilai ini adalah urusan Work Owner,
            sedangkan seluruh penyetuju yang tersingkir karena penginputnya adalah
            keadaan yang berulang setiap kali orang itu mengajukan.
          */}
          {(result.tersingkir?.length ?? 0) > 0 ? (
            <ErrorMessage
              title="Seluruh penyetuju tersingkir"
              description={
                `Satu-satunya yang berwenang pada tangga ini adalah ` +
                `${result.tersingkir?.map((a) => a.nama || a.operator_id).join(', ')}, ` +
                `dan mereka dikecualikan karena mengajukan klaim ini sendiri. ` +
                `Klaim seperti ini akan berhenti tanpa penyetuju — laporkan ke Work Owner.`
              }
              tone="penolakan"
            />
          ) : (
            <ErrorMessage
              title="Tidak ada jenjang yang cocok"
              description={
                'Tidak satu pun baris master yang ambang bawahnya terlampaui nilai ini. ' +
                'Klaim sebesar ini tidak akan mendapat persetujuan siapa pun — laporkan ' +
                'ke Work Owner sebelum alur komite dipakai.'
              }
              tone="penolakan"
            />
          )}
        </div>
      ) : (
        <ol className="divide-y divide-slate-100">
          {result.penyetuju.map((a) => (
            <ApproverRow key={a.id_ambang} approver={a} />
          ))}
        </ol>
      )}

      {singleApprover && <SingleApproverNote result={result} />}

      {!singleApprover && (result.tersingkir?.length ?? 0) > 0 && !result.tanpa_penyetuju && (
        <ExcludedNote excluded={result.tersingkir ?? []} />
      )}

      {result.urutan_tidak_pasti && <OrderNote />}
    </section>
  )
}

/**
 * Pada mode satu-penyetuju, yang dapat diperiksa BUKAN siapa yang terpilih — itu acak —
 * melainkan apakah kumpulan yang layak sudah benar.
 *
 * Karena itu kandidatnya ditampilkan utuh beserta siapa yang dikecualikan. Tanpa itu,
 * layar hanya menyodorkan satu nama yang berubah-ubah setiap kali tombol ditekan, dan
 * tidak ada cara memastikan aturannya berjalan benar.
 */
function SingleApproverNote({ result }: { result: KomiteTieringResponse }) {
  const candidates = result.kandidat ?? []

  return (
    <div className="border-t border-slate-200 bg-slate-50/70 p-5">
      <h3 className="text-sm font-semibold text-slate-900">
        Dipilih acak dari {candidates.length} calon
      </h3>
      <p className="mt-1 text-sm leading-relaxed text-slate-600">
        Portal ini memilih <strong>satu</strong> penyetuju, bukan mengakumulasi jenjang.
        Calonnya adalah semua yang berwenang pada jenjang terendah, dan satu di antaranya
        dipilih acak untuk menyebar beban.
        {result.dikecualikan_penginput ? (
          <>
            {' '}
            <strong>{result.dikecualikan_penginput}</strong> dikeluarkan dari daftar calon
            karena dialah yang mengajukan.
          </>
        ) : null}
      </p>

      {candidates.length > 0 && (
        <ul className="mt-3 flex flex-wrap gap-2">
          {candidates.map((c) => (
            <li
              key={c.id_ambang}
              className={[
                'rounded-full px-3 py-1 text-xs font-medium',
                c.operator_id === result.penyetuju[0]?.operator_id
                  ? 'bg-blue-100 text-blue-900'
                  : 'bg-white text-slate-600 ring-1 ring-slate-200',
              ].join(' ')}
            >
              {c.nama || c.operator_id}
              {c.operator_id === result.penyetuju[0]?.operator_id && ' · terpilih'}
            </li>
          ))}
        </ul>
      )}

      <p className="mt-3 text-xs leading-relaxed text-slate-500">
        Karena pemilihannya acak, hasil yang berbeda pada nilai yang sama <strong>bukan</strong>{' '}
        cacat. Yang harus tetap sama adalah daftar calonnya.
      </p>
    </div>
  )
}

/**
 * Setiap penyetuju ditampilkan bersama ALASANNYA — ambang yang membuatnya ikut.
 *
 * Tanpa alasan itu, layar hanya menyodorkan daftar nama dan pengguna tidak punya cara
 * memeriksa apakah hasilnya masuk akal.
 */
function ApproverRow({ approver }: { approver: KomiteApprover }) {
  return (
    <li className="flex items-start gap-4 px-5 py-4">
      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-100 text-sm font-semibold text-blue-800 tabular-nums">
        {approver.urutan}
      </span>
      <div className="min-w-0 flex-1">
        <p className="font-medium text-slate-900">
          {approver.nama || approver.operator_id}
          {approver.sedang_absen && (
            <span className="ml-2 inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900">
              sedang absen
            </span>
          )}
        </p>
        <p className="mt-0.5 text-sm text-slate-600">
          {approver.operator_id} · ikut karena nilai klaim melampaui{' '}
          <span className="tabular-nums">{formatRupiah(approver.batas_bawah)}</span>
        </p>
      </div>
      <span className="shrink-0 text-xs text-slate-400">
        jenjang {approver.jenjang} · baris {approver.id_ambang}
      </span>
    </li>
  )
}

/**
 * Pengurangan jumlah penyetuju karena penginput DILAPORKAN, tidak terjadi diam-diam.
 *
 * Tanpa keterangan ini, dua orang yang menghitung klaim yang sama akan mendapat jumlah
 * penyetuju yang berbeda dan tidak punya cara mengetahui sebabnya.
 */
function ExcludedNote({ excluded }: { excluded: KomiteApprover[] }) {
  return (
    <div className="flex items-start gap-3 border-t border-amber-200 bg-amber-50/70 p-4">
      <WarningIcon className="mt-0.5 h-4 w-4 shrink-0 text-amber-700" />
      <p className="text-sm leading-relaxed text-amber-900">
        <strong>{excluded.map((a) => a.nama || a.operator_id).join(', ')}</strong>{' '}
        dikeluarkan dari daftar penyetuju karena mengajukan klaim ini sendiri. Jumlah
        penyetujunya karena itu{' '}
        <strong>berkurang {excluded.length}</strong> dari tangga masternya.
      </p>
    </div>
  )
}

/**
 * Keraguan urutan dilaporkan di layar, bukan disembunyikan.
 *
 * Kueri sistem lama mengurutkan dengan `ORDER BY DEGREE` saja, sehingga saat dua baris
 * ber-DEGREE sama urutannya ditentukan basis data dan dapat berubah antar eksekusi.
 * Aplikasi ini mengurutkannya secara pasti, dan memberitahukannya supaya perbedaan
 * urutan terhadap Pega pada kasus seri tidak terbaca sebagai cacat.
 */
function OrderNote() {
  return (
    <div className="flex items-start gap-3 border-t border-amber-200 bg-amber-50/70 p-4">
      <WarningIcon className="mt-0.5 h-4 w-4 shrink-0 text-amber-700" />
      <p className="text-sm leading-relaxed text-amber-900">
        Ada dua penyetuju atau lebih dengan nomor jenjang yang sama. Master tidak
        menentukan siapa lebih dulu di antara mereka, sehingga sistem lama dapat
        mengurutkannya berbeda-beda. Di sini urutannya ditetapkan dari ambang terkecil ke
        terbesar — <strong>siapa</strong> yang menyetujui tetap sama.
      </p>
    </div>
  )
}

/**
 * Ketujuh kasus pada spesifikasi B-7 disajikan sebagai contoh yang dapat dicoba.
 *
 * Nilainya bukan hiasan: inilah kasus yang dipakai membuktikan aturan kumulatif benar,
 * dan menampilkannya membuat pengguna bisnis dapat memeriksanya sendiri tanpa perlu
 * membuka dokumen.
 */
function ExampleCases() {
  const cases = [
    { line: 'PA', value: 'Rp 5.000.000', tiers: 1 },
    { line: 'PA', value: 'Rp 75.000.000', tiers: 3 },
    { line: 'PA', value: 'Rp 150.000.000', tiers: 4 },
    { line: 'TRAVEL', value: 'Rp 150.000.000', tiers: 3 },
    { line: 'NONMBU', value: 'Rp 80.000.000', tiers: 2 },
    { line: 'NONMBU', value: 'Rp 750.000.000', tiers: 2 },
    { line: 'NONMBU', value: 'Rp 2.000.000.000', tiers: 3 },
  ]

  return (
    <section className="rounded-kartu border border-slate-200 bg-slate-50/60 p-5">
      <h2 className="text-sm font-semibold text-slate-900">Kasus yang dapat dicoba</h2>
      <p className="mt-1 text-sm leading-relaxed text-slate-600">
        Tujuh kasus berikut dipakai membuktikan aturan penjenjangan. Cobalah salah satunya
        untuk memastikan hasilnya sesuai.
      </p>
      <ul className="mt-3 grid gap-x-6 gap-y-1.5 text-sm text-slate-700 sm:grid-cols-2">
        {cases.map((c) => (
          <li key={`${c.line}-${c.value}`} className="flex items-baseline justify-between gap-3">
            <span>
              <strong className="font-medium text-slate-900">{c.line}</strong>{' '}
              <span className="tabular-nums">{c.value}</span>
            </span>
            <span className="shrink-0 text-slate-500">{c.tiers} jenjang</span>
          </li>
        ))}
      </ul>
    </section>
  )
}

function ComputeErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Perhitungan belum dapat dijalankan. Periksa koneksi lalu tekan Hitung lagi."
        tone="gangguan"
      />
    )
  }
  if (error instanceof APIError) {
    return <ErrorMessage title="Tidak dapat menghitung" description={error.message} tone="penolakan" />
  }
  return (
    <ErrorMessage
      title="Tidak dapat menghitung"
      description="Perhitungan belum dapat dijalankan. Coba tekan Hitung lagi."
      tone="gangguan"
    />
  )
}
