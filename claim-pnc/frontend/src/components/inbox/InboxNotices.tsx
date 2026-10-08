import type { ReactNode } from 'react'

import { ErrorMessage } from '@/components/ErrorMessage'

/**
 * Keterangan tab yang digambar tetapi belum dapat diisi.
 *
 * Alasan dan pemiliknya datang dari SERVER, bukan ditulis tetap di layar, supaya keduanya
 * hilang dengan sendirinya begitu penghalangnya hilang. Menyebut pemiliknya penting:
 * penghalang tanpa alamat tidak pernah hilang.
 */
export function BlockedNotice({
  tab,
}: Readonly<{
  tab: { nama: string; alasan_terhalang?: string | undefined; pemilik_penghalang?: string | undefined }
}>) {
  return (
    <div className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-4">
      <h2 className="text-sm font-semibold text-amber-900">{tab.nama} belum tersedia</h2>
      <p className="mt-2 text-sm text-slate-700">{tab.alasan_terhalang}</p>
      {tab.pemilik_penghalang && (
        <p className="mt-2 text-xs text-slate-600">
          <span className="font-medium">Menunggu:</span> {tab.pemilik_penghalang}
        </p>
      )}
    </div>
  )
}

/**
 * EntityNotice menyebut terang-terangan entitas yang sedang dilihat. Satu aplikasi melayani
 * empat badan hukum dengan basis data terpisah, dan "antrean siapa ini" tidak boleh hanya
 * diandaikan pengguna (ADR-0030, R-20).
 */
export function EntityNotice({ lead, portal }: Readonly<{ lead: ReactNode; portal: string }>) {
  return (
    <p className="mt-3 text-xs text-slate-500">
      <span>{lead}</span>
      <span className="ml-1">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">{portal}</span>
      </span>
    </p>
  )
}

/**
 * TruncatedNotice menyatakan bahwa daftar dipotong, alih-alih membiarkannya senyap seperti
 * `pyMaxRecords=500` pada sistem lama. Batas yang diketahui adalah batas; batas yang senyap
 * adalah data yang hilang.
 */
export function TruncatedNotice({ noun, limit }: Readonly<{ noun: string; limit: number }>) {
  return (
    <p className="mt-4 rounded-kartu border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900">
      {noun} <span className="font-medium">lebih panjang</span> daripada yang dapat
      ditampilkan sekaligus. Yang tampil {limit} pekerjaan pertama;
      sisanya belum terlihat. Pakai kotak pencarian untuk mempersempit daftar.
    </p>
  )
}

/**
 * Catatan khusus satu tab, digambar di antara keterangan tab dan tabelnya.
 *
 * Isinya datang dari server — mis. penjelasan kenapa sebuah tab diperkirakan kosong —
 * supaya pengguna tidak mengira tabel kosong itu kerusakan.
 */
export function TabNotice({ text }: Readonly<{ text: string }>) {
  return (
    <div className="mt-3 rounded-kartu border border-sky-200 bg-sky-50 px-4 py-3">
      <p className="text-xs text-slate-700">{text}</p>
    </div>
  )
}

/**
 * Daftar catatan berbutir di bawah tabel — bawaannya "selisih terhadap Pega yang sudah
 * diputuskan".
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di layar. Tanpa catatan ini, perbedaan
 * yang disengaja akan dilaporkan berulang kali sebagai kerusakan oleh orang yang
 * membandingkan layar lama dan baru berdampingan. Tidak digambar bila daftarnya kosong.
 */
export function NoteList({
  lines,
  title = 'Yang berbeda dari layar lama, dan itu disengaja',
}: Readonly<{ lines: string[]; title?: string | undefined }>) {
  if (lines.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">{title}</h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
        {lines.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </section>
  )
}

/** Satu selisih terencana yang punya ringkasan dan rincian. */
export type DetailedDifference = {
  ringkas: string
  rincian: string
}

/**
 * Selisih terencana yang tiap butirnya dapat dibuka untuk membaca rinciannya.
 *
 * Memakai `details` bawaan peramban, bukan buka-tutup yang ditulis sendiri. Isinya tetap
 * ada di halaman saat tertutup, sehingga pencarian peramban (Ctrl+F) dan pembaca layar
 * tetap menemukannya — dan tidak ada state yang dapat menyimpang antara apa yang tergambar
 * dan apa yang dikirim server.
 */
export function DetailedDifferences({ lines }: Readonly<{ lines: DetailedDifference[] }>) {
  if (lines.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">
        Yang berbeda dari layar lama, dan itu disengaja
      </h2>

      <ul className="mt-2 space-y-2 text-xs text-slate-600">
        {lines.map((line) => (
          <li key={line.ringkas}>
            <details className="group">
              <summary
                className={[
                  'flex cursor-pointer list-none items-start gap-2',
                  'rounded-kontrol text-slate-700 marker:content-none',
                  'hover:text-slate-900',
                  'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                ].join(' ')}
              >
                <span
                  aria-hidden="true"
                  className="mt-px shrink-0 text-slate-400 transition-transform duration-150 ease-halus group-open:rotate-90"
                >
                  ›
                </span>
                <span>{line.ringkas}</span>
              </summary>

              <p className="mt-1 pl-5 text-slate-500">{line.rincian}</p>
            </details>
          </li>
        ))}
      </ul>
    </section>
  )
}

/**
 * loadGate menyusun isi layar selama bentuk layarnya belum siap — sedang dimuat, atau
 * gagal dimuat — atau `null` bila layar boleh digambar.
 */
export function loadGate(
  state: { isPending: boolean; isError: boolean; error: unknown },
  describe: (error: unknown) => {
    title: string
    description: string
    tone: 'penolakan' | 'gangguan'
  },
): ReactNode {
  if (state.isPending) {
    return <p className="text-sm text-slate-600">Memuat keterangan layar…</p>
  }
  if (state.isError) {
    const message = describe(state.error)
    return (
      <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
    )
  }
  return null
}

/**
 * portalGate adalah screenGate untuk layar yang hanya terhalang oleh portal yang belum
 * dipilih; kegagalan memuatnya ditangani layar itu sendiri.
 */
export function portalGate(portal: string | null, subject: string): ReactNode {
  return screenGate({ portal, subject, failed: false, error: null, describe: () => '' })
}

type GateInput = {
  /** Portal (entitas) yang sedang dipilih; `null` bila belum ada. */
  portal: string | null
  /** Pokok kalimat penjelas portal, mis. "Antrean kepatuhan". */
  subject: string
  /** Akhir kalimat penjelas portal. */
  action?: string | undefined
  /** Apakah pemuatan metadata layar gagal. */
  failed: boolean
  error: unknown
  describe: (error: unknown) => string
}

/**
 * screenGate menyusun pesan penghalang sebelum isi layar inbox digambar, atau `null`
 * bila layar boleh dibuka.
 *
 * Dua penghalangnya selalu sama: portal belum dipilih — data inbox milik satu badan hukum,
 * sementara aplikasi melayani empat — dan metadata layar gagal dimuat.
 */
export function screenGate({
  portal,
  subject,
  action = 'membukanya',
  failed,
  error,
  describe,
}: GateInput): ReactNode {
  if (portal === null) {
    return (
      <ErrorMessage
        title="Pilih entitas lebih dulu"
        description={
          `${subject} milik satu badan hukum, dan aplikasi ini melayani empat. ` +
          `Pilih portal di bilah atas untuk ${action}.`
        }
        tone="gangguan"
      />
    )
  }

  if (failed) {
    return (
      <ErrorMessage title="Layar tidak dapat dibuka" description={describe(error)} tone="gangguan" />
    )
  }

  return null
}
