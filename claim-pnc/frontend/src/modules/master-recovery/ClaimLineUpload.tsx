import { useRef } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type RecoveryClaimLine } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { formatMoney } from '@/lib/money'

import { useDownloadTemplate, useReadClaimLine } from './api'

type Props = {
  claimLine: RecoveryClaimLine[]
  onChange: (line: RecoveryClaimLine[]) => void
  disabled: boolean
}

/**
 * Panel **Upload Data Klaim** beserta grid hasilnya.
 *
 * Menggantikan tombol unggah, tautan **Format File**, dan grid dua kolom pada
 * `Section/OutstandingMasterRecovery-Section.xml` (`:3783` "No Polis" dan `:3872`
 * "Nilai Klaims").
 *
 * # Bagaimana barisnya sampai tersimpan
 *
 * Berkas dibaca SERVER, bukan di peramban — aturan bentuknya satu, dan pemanggil lain
 * kelak memakai pembaca yang sama. Barisnya dikembalikan untuk ditampilkan, lalu dikirim
 * kembali bersama permintaan simpan sebagai isi kolom JSON_POLIS. Itu meniru sistem lama,
 * yang menyusunnya di klipboard sebelum mengubahnya menjadi teks JSON.
 *
 * # Yang sengaja dibuat berbeda dari layar lama
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Baris cacat | tidak diketahui perlakuannya | ditolak dan disebut nomor barisnya |
 * | Jumlah nilai | tidak ada | dihitung dan ditampilkan sebelum menyimpan |
 * | Membatalkan unggahan | tidak ada | daftar dapat dikosongkan kembali |
 */
export function ClaimLineUpload({ claimLine, onChange, disabled }: Props) {
  const read = useReadClaimLine()
  const template = useDownloadTemplate()
  const picker = useRef<HTMLInputElement | null>(null)

  function choose(file: File | undefined) {
    if (!file) return
    read.mutate(file, { onSuccess: (answer) => onChange(answer.baris_klaim) })
  }

  const total = claimLine.reduce((sum, row) => sum + row.nilai_klaim, 0)
  const rejected = read.data?.baris_ditolak ?? []

  const columns: Column<RecoveryClaimLine>[] = [
    {
      key: 'nomor_polis',
      title: 'No Polis',
      value: (row) => row.nomor_polis,
      render: (row) => <span className="font-mono text-xs text-slate-800">{row.nomor_polis}</span>,
    },
    {
      key: 'nilai_klaim',
      title: 'Nilai Klaim (Rp)',
      width: '12rem',
      alignRight: true,
      // Yang diurutkan adalah nilai POLOS, bukan yang sudah diformat: "1.000.000" dan
      // "900.000" diurutkan sebagai teks akan menaruh yang kecil di atas.
      value: (row) => String(row.nilai_klaim).padStart(20, '0'),
      render: (row) => (
        <span className="font-mono text-xs tabular-nums text-slate-800">
          {formatMoney(row.nilai_klaim)}
        </span>
      ),
    },
  ]

  return (
    <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <div>
          <h3 className="text-base font-semibold text-slate-900">Data Klaim (opsional)</h3>
          <p className="mt-1 max-w-2xl text-sm text-slate-600">
            Daftar polis yang tercakup batch ini, diunggah sebagai berkas CSV dua kolom.
            Isinya tersimpan menyatu dengan batch — bukan sebagai data tersendiri.
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button
            tone="halus"
            onClick={() => void template.mutate()}
            disabled={template.isPending}
          >
            {template.isPending ? 'Menyiapkan…' : 'Unduh format'}
          </Button>
          <Button
            tone="kedua"
            onClick={() => picker.current?.click()}
            disabled={disabled || read.isPending}
          >
            {read.isPending ? 'Membaca…' : 'Unggah data klaim'}
          </Button>
          {claimLine.length > 0 && (
            <Button tone="halus" onClick={() => onChange([])} disabled={disabled}>
              Kosongkan
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-4 p-5">
        {/*
          Input berkas disembunyikan dan dipicu tombol di atas. Input berkas bawaan
          peramban tidak dapat diberi gaya, dan tampilannya berbeda di setiap sistem —
          tombol yang seragam dengan tombol lain di layar ini jauh lebih terbaca.
        */}
        <input
          ref={picker}
          type="file"
          accept=".csv,text/csv"
          className="sr-only"
          aria-label="Pilih berkas CSV data klaim"
          onChange={(event) => {
            choose(event.target.files?.[0])
            // Dikosongkan supaya memilih berkas yang SAMA dua kali tetap memicu
            // pembacaan. Tanpa ini, mengunggah ulang berkas yang baru diperbaiki tidak
            // akan terjadi apa-apa, dan petugas mengira aplikasinya menggantung.
            event.target.value = ''
          }}
        />

        {read.isError && <ReadErrorMessage error={read.error} />}

        {rejected.length > 0 && (
          /*
            Baris yang ditolak ditampilkan BERSAMA yang diterima, bukan menggantikannya.
            Berkas dengan satu baris cacat tetap berguna, dan petugas berhak tahu persis
            baris mana yang tidak ikut — nomor barisnya disebut supaya ia dapat
            menemukannya di berkas aslinya.
          */
          <div className="rounded-kartu border border-amber-200 bg-amber-50 p-4">
            <p className="text-sm font-medium text-amber-900">
              {rejected.length} baris tidak dapat dibaca dan tidak ikut disimpan
            </p>
            <ul className="mt-2 space-y-1 text-xs text-amber-800">
              {rejected.map((item, index) => (
                <li key={index}>• {item.pesan}</li>
              ))}
            </ul>
          </div>
        )}

        {claimLine.length === 0 ? (
          <p className="rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-center text-sm text-slate-500">
            Belum ada data klaim. Batch tetap dapat disimpan tanpa daftar ini.
          </p>
        ) : (
          <>
            <DataTable
              columns={columns}
              rows={claimLine}
              rowKey={(row) => `${row.nomor_polis}-${row.nilai_klaim}`}
              searchLabel="Cari nomor polis"
              emptyMessage="Tidak ada baris yang cocok."
            />
            {/*
              Jumlah ditampilkan supaya petugas dapat membandingkannya dengan angka yang
              diketiknya sendiri di isian Pembayaran SEBELUM menyimpan. Sistem lama tidak
              punya penjumlahan ini sama sekali.
            */}
            <div className="flex items-center justify-between rounded-kontrol bg-slate-50 px-4 py-3 text-sm">
              <span className="text-slate-600">{claimLine.length} polis</span>
              <span className="font-medium text-slate-900">
                Jumlah nilai klaim: Rp {formatMoney(total)}
              </span>
            </div>
          </>
        )}
      </div>
    </section>
  )
}

function ReadErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Berkas belum terbaca. Periksa koneksi lalu unggah lagi."
        tone="gangguan"
      />
    )
  }
  if (!(error instanceof APIError)) {
    return (
      <ErrorMessage
        title="Berkas gagal dibaca"
        description="Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi."
        tone="gangguan"
      />
    )
  }

  const message = parse(error)
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

function parse(error: APIError): { title: string; description: string; tone: ErrorTone } {
  switch (error.kode) {
    case ErrorCode.claimFileEmpty:
      return {
        title: 'Berkas tidak memuat data',
        description:
          'Tidak ada satu baris pun yang dapat dibaca. Pakai berkas contoh sebagai acuan bentuknya.',
        tone: 'penolakan',
      }
    case ErrorCode.claimFileTooBig:
      return {
        title: 'Berkas terlalu besar',
        description: 'Pecah menjadi beberapa berkas lalu unggah bergantian.',
        tone: 'penolakan',
      }
    case ErrorCode.claimFileUnreadable:
      return {
        title: 'Bentuk berkas tidak dikenali',
        description:
          'Pakai berkas CSV dengan dua kolom: nomor polis dan nilai klaim. Unduh berkas contoh untuk acuannya.',
        tone: 'penolakan',
      }
    default:
      return { title: 'Berkas gagal dibaca', description: error.message, tone: 'gangguan' }
  }
}
