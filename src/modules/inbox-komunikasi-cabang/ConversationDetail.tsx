import { useEffect, useRef, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { CloseIcon } from '@/components/Icon'
import { formatPegaDateTime } from '@/components/format'

import { useKomunikasiCabangDetail, useKomunikasiCabangReply } from './api'
import type { Attachment, ThreadMessage } from './types'

type Props = {
  /** Nomor percakapan yang sedang dibuka. */
  id: string
  onClose: () => void
}

/** Satu ucapan beserta nomor urutnya pada utas. */
type ThreadRow = ThreadMessage & { nomor: number }

const TITLE_ID = 'judul-detail-komunikasi-cabang'

/**
 * Layar **Detail Komunikasi Cabang** — popup utas satu percakapan.
 *
 * # Apa yang digantikan
 *
 * Tombol "Detail Komunikasi" pada grid menjalankan
 * `Data Transform/DetailKomunikasi_dt-DT.xml`, yang menyiapkan dua parameter — nomor
 * percakapan dan kode cabangnya — lalu flow action `DETAILKOMUNIKASICABANG_11` ("DETAIL
 * KOMUNIKASI CABANG") menyisipkan `Section/BalasKomunikasiCabang-Section.xml`.
 *
 * Bentuknya, dibaca dari section itu dan dari layar Pega yang berjalan:
 *
 *   1. grid TIGA kolom — Tanggal · Pengirim · Pesan — bernomor baris
 *   2. isian berlabel "Masukkan Balasan" (`pxTextInput`, SATU baris)
 *   3. tombol "Balas"
 *
 * # Ia POPUP, dan itu berubah pada 2026-10-10
 *
 * Versi pertama menggambarnya sebagai panel yang menempel di bawah tabel, dengan alasan
 * petugas kembali ke daftarnya begitu selesai membaca. Alasan itu benar, tetapi bentuknya
 * tidak: di Pega ia dialog melayang berjudul "DETAIL KOMUNIKASI CABANG" dengan tombol
 * silang di pojok, dan daftarnya tetap terlihat di belakangnya. Panel yang menempel justru
 * MENDORONG daftarnya ke bawah layar — tepat kebalikan dari menjaga tempat.
 *
 * # SATU ARTEFAKNYA MASIH HILANG DARI EXPORT
 *
 * Kueri pemasok grid-nya (`GetInboxKomunikasiCabang_detail`) TIDAK ADA di export mana pun —
 * kelas `R-16`. Yang dibangun karena itu adalah bentuk yang terbaca dari section-nya sendiri.
 * Ia TIDAK mengarang penyaring yang tidak dapat dibaca dari mana pun; yang dipakai hanyalah
 * nomor percakapan, satu-satunya parameter yang benar-benar dikirim tombolnya. Dinyatakan
 * lewat `selisih_terencana`.
 *
 * Activity tombol Balas (`PNCReplyMessageCabang`) sempat hilang pula, dan DITERIMA pada
 * 2026-09-24 bersama `RDB List/ReplyKomunikasi-SQL.xml`. Sejak itu kotak balasan di bawah
 * bukan lagi keterangan melainkan isian yang benar-benar menyimpan.
 */
export function ConversationDetail({ id, onClose }: Props) {
  const detail = useKomunikasiCabangDetail(id)

  // Escape menutup popup, seperti dialog mana pun yang dikenal pengguna. Pola yang sama
  // dipakai popup Detail pada Inbox OS Klaim per Cabang.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  // Fokus dipindahkan KE DALAM dialog saat ia terbuka.
  //
  // Tanpa ini fokus papan ketik tertinggal di tombol "Detail Komunikasi" yang kini berada di
  // belakang lapisan gelap: penekanan Tab berikutnya menjelajahi tabel yang tidak dapat
  // disentuh, dan pembaca layar tidak pernah tahu ada dialog yang terbuka.
  const panel = useRef<HTMLDivElement>(null)
  useEffect(() => {
    panel.current?.focus()
  }, [])

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby={TITLE_ID}
    >
      <div
        ref={panel}
        tabIndex={-1}
        className="w-full max-w-2xl rounded-kartu bg-white shadow-angkat focus:outline-none"
      >
        <header className="flex items-center justify-between gap-4 border-b border-slate-200 px-5 py-3">
          {/*
            Judulnya ditulis dengan huruf biasa dan DIBESARKAN oleh gaya, bukan diketik
            sebagai huruf besar semua. Yang terlihat sama dengan Pega; yang dibacakan
            pembaca layar tetap kalimat, bukan deretan huruf yang dieja satu per satu.
          */}
          <h2
            id={TITLE_ID}
            className="text-sm font-semibold tracking-wide text-slate-900 uppercase"
          >
            Detail Komunikasi Cabang
          </h2>

          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup Detail Komunikasi Cabang"
            className={[
              'shrink-0 rounded-kontrol p-1 text-slate-500',
              'transition-colors duration-150 ease-halus hover:bg-slate-100 hover:text-slate-800',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
            ].join(' ')}
          >
            <CloseIcon className="h-4 w-4" />
          </button>
        </header>

        <div className="px-5 py-4">
          {detail.isPending && (
            <p className="text-sm text-slate-500">Memuat percakapan…</p>
          )}

          {detail.isError && (
            <ErrorMessage
              title="Percakapan tidak dapat dibuka"
              description={messageOf(detail.error)}
              tone="gangguan"
            />
          )}

          {detail.isSuccess && (
            <>
              <Thread messages={detail.data.pesan} />
              <Attachments items={detail.data.lampiran} />
              {detail.data.balas_tersedia && <ReplyBox id={id} />}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

/**
 * Utas pesan, digambar sebagai GRID tiga kolom bernomor baris.
 *
 * # Kenapa tabel, padahal dulu daftar
 *
 * Versi pertama menggambarnya sebagai utas percakapan, dengan alasan kolom "Pesan" berisi
 * kalimat utuh yang akan terpotong di dalam sel. Alasan itu masuk akal dan tetap tidak
 * terbukti: `DataTable` memutus kalimat panjang ke baris berikutnya (`break-words`), bukan
 * memotongnya, sehingga kekhawatiran yang melahirkan bentuk utas tidak pernah terjadi.
 *
 * Yang pasti terjadi adalah selisihnya dengan layar lama, dan Work Owner memintanya
 * diseragamkan (2026-10-10). Ketiga judul kolomnya diambil dari section apa adanya —
 * "Tanggal", "Pengirim", "Pesan" — beserta kolom nomor baris di paling kiri.
 *
 * # Kolom "Pengirim" di sini TIDAK sama dengan "Pengirim(Dari)" di daftar
 *
 * Yang ini berisi Operator ID apa adanya; yang di daftar dirakit menjadi `operator (asal)`.
 * Itu bukan kelalaian: tabel riwayat tidak memuat kolom asal sama sekali, sehingga tidak ada
 * yang dapat dirakit.
 */
function Thread({ messages }: { messages: ThreadMessage[] }) {
  const rows: ThreadRow[] = messages.map((message, index) => ({
    ...message,
    nomor: index + 1,
  }))

  return (
    <DataTable<ThreadRow>
      columns={threadColumns()}
      rows={rows}
      // Nomor urut, bukan isi barisnya: dua ucapan yang kebetulan sama persis — tanggal,
      // pengirim, dan pesannya — tetap dua baris yang berbeda.
      rowKey={(row) => String(row.nomor)}
      label="Riwayat percakapan"
      gridLines
      hideSearch
      emptyMessage={
        'Percakapan ini belum memuat satu pun ucapan. Ia mungkin dibuat lewat layar lain, ' +
        'atau sebelum riwayat percakapan mulai dicatat.'
      }
    />
  )
}

/**
 * threadColumns menyusun keempat kolom grid utas.
 *
 * SELURUHNYA `noSort`, sama alasannya dengan grid di layar daftar: grid Pega tidak dapat
 * diurutkan, dan di sini pengurutan justru merusak — sebuah utas percakapan yang diurutkan
 * menurut pengirim berhenti menjadi utas.
 */
function threadColumns(): Column<ThreadRow>[] {
  return [
    {
      key: 'nomor',
      // Tanpa judul, seperti di Pega.
      title: '',
      width: '3rem',
      noSort: true,
      value: (row) => String(row.nomor),
    },
    {
      key: 'tanggal',
      title: 'Tanggal',
      width: '9rem',
      noSort: true,
      value: (row) => formatPegaDateTime(row.tanggal),
    },
    {
      key: 'pengirim',
      title: 'Pengirim',
      width: '9rem',
      noSort: true,
      value: (row) => row.pengirim,
    },
    {
      key: 'pesan',
      title: 'Pesan',
      noSort: true,
      value: (row) => row.pesan,
    },
  ]
}

/**
 * Daftar lampiran percakapan.
 *
 * Sumbernya `RDB List/GetDocumentKomunikasi1-SQL.xml`. Keputusan Work Owner 2026-09-24:
 * lampiran ikut dibangun, BACA-SAJA.
 *
 * # Ia TIDAK digambar ketika kosong, dan itu berubah pada 2026-10-10
 *
 * Popup Pega tidak memuat bagian lampiran sama sekali. Menggambar judul "Lampiran" beserta
 * kalimat "tidak ada lampiran" pada percakapan yang memang tidak punya lampiran membuat
 * popup ini terlihat berbeda dari acuannya justru pada keadaan yang paling sering terjadi.
 *
 * Bagiannya tetap ada ketika benar-benar ADA lampiran: ia kemampuan yang sudah diputuskan,
 * dan menyembunyikan dokumen yang ada bukan paritas melainkan kehilangan.
 */
function Attachments({ items }: { items: Attachment[] }) {
  if (items.length === 0) return null

  return (
    <section className="mt-5 border-t border-slate-200 pt-4">
      <h3 className="text-xs font-semibold tracking-wide text-slate-700 uppercase">
        Lampiran
      </h3>

      <ul className="mt-2 space-y-2">
        {items.map((item, index) => (
          <li
            key={`${item.id_dokumen}|${index}`}
            className="flex flex-wrap items-start justify-between gap-3 rounded-kartu border border-slate-200 px-3 py-2"
          >
            <div className="min-w-0">
              <p className="text-sm text-slate-800">
                {item.rincian_dokumen || item.jenis_dokumen || '—'}
              </p>
              <p className="mt-0.5 text-xs text-slate-500">
                {item.jenis_dokumen || '—'}
                {item.catatan ? ` · ${item.catatan}` : ''}
              </p>
            </div>

            <span
              className={[
                'rounded-full px-2 py-0.5 text-xs whitespace-nowrap',
                item.sudah_diunggah
                  ? 'bg-emerald-100 text-emerald-900'
                  : 'bg-amber-100 text-amber-900',
              ].join(' ')}
            >
              {item.sudah_diunggah
                ? `Sudah Upload · ${formatPegaDateTime(item.tanggal_unggah)}`
                : 'Belum Upload'}
            </span>
          </li>
        ))}
      </ul>

      {/*
        Ketiadaan tombol unduh dinyatakan, bukan dibiarkan terbaca sebagai kelalaian.

        Berkasnya hidup di penyimpanan dokumen internal (`D-16`), dan seam pengambilnya belum
        dibangun di modul ini. Yang ada di tabel hanyalah metadata-nya.
      */}
      <p className="mt-2 text-xs text-slate-500">
        Berkasnya belum dapat diunduh dari sini — yang tersimpan di tabel ini hanya
        keterangannya. Unduh lewat Pega bila isinya dibutuhkan.
      </p>
    </section>
  )
}

/**
 * Kotak **"Masukkan Balasan"** beserta tombol **"Balas"**.
 *
 * # Apa yang digantikan
 *
 * `Section/BalasKomunikasiCabang-Section.xml` menggambar satu isian berlabel "Masukkan
 * Balasan" dan satu tombol "Balas", yang menjalankan `PNCReplyMessageCabang` dengan tiga
 * parameter: nomor percakapan, isi balasan, dan `kodecabang`.
 *
 * Yang ketiga TIDAK dikirim dari sini, dan itu disengaja. Namanya menyebut kode cabang tetapi
 * isinya `TempView2.City`, yang berasal dari `CASEID` — penanda kanal, bukan kode cabang.
 * Peladen mengisinya sendiri dari konstanta domainnya; mengirimkannya dari layar berarti
 * penanda yang menentukan baris mana yang boleh diubah datang dari permintaan.
 *
 * Penjawabnya juga tidak dikirim: ia diambil dari sesi. Jejak yang isinya ditentukan pengirim
 * permintaan bukan jejak, dan `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang.
 *
 * # Isian SATU BARIS, dan itu berubah pada 2026-10-10
 *
 * Section menuliskan kontrolnya `pxTextInput` — isian satu baris — dan layar Pega yang
 * berjalan menggambarnya demikian. Versi pertama memakai `textarea` tiga baris karena
 * balasan dapat panjang. Itu dugaan yang masuk akal dan tetap salah terhadap acuannya.
 *
 * Batas 4.000 huruf dari peladen DIPERTAHANKAN sebagai `maxLength`. Pencacah hurufnya
 * dicabut bersama `textarea`: pada isian satu baris, angka yang menghitung menuju 4.000
 * menjanjikan ruang yang tidak dapat dilihat penulisnya.
 *
 * # Kenapa isiannya TIDAK dikosongkan saat gagal
 *
 * Karena kalimat yang sudah diketik adalah pekerjaan penggunanya. Mengosongkannya saat
 * permintaan ditolak — jaringan putus, cabang tidak terbaca, percakapan baru saja ditutup
 * orang lain — membuang pekerjaan itu tanpa ia sempat menyalinnya.
 *
 * Yang dikosongkan hanyalah yang BERHASIL tersimpan, dan hanya setelah peladen menjawab.
 */
function ReplyBox({ id }: { id: string }) {
  const [message, setMessage] = useState('')
  const reply = useKomunikasiCabangReply()

  const trimmed = message.trim()

  return (
    <form
      className="mt-5 border-t border-slate-200 pt-4"
      onSubmit={(event) => {
        event.preventDefault()
        if (trimmed === '' || reply.isPending) return

        reply.mutate(
          { komunikasi: id, pesan: trimmed },
          { onSuccess: () => setMessage('') },
        )
      }}
    >
      <label
        htmlFor={`balasan-${id}`}
        className="block text-xs font-medium text-slate-800"
      >
        Masukkan Balasan
      </label>

      <input
        id={`balasan-${id}`}
        type="text"
        value={message}
        onChange={(event) => setMessage(event.target.value)}
        maxLength={4000}
        className="mt-1.5 w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm text-slate-900 focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
      />

      <div className="mt-2">
        <Button type="submit" disabled={trimmed === '' || reply.isPending}>
          {reply.isPending ? 'Menyimpan…' : 'Balas'}
        </Button>
      </div>

      {reply.isError && (
        <div className="mt-2">
          <ErrorMessage
            title="Balasan tidak tersimpan"
            description={messageOf(reply.error)}
            tone="gangguan"
          />
        </div>
      )}

      {reply.isSuccess && (
        <p className="mt-2 text-xs text-emerald-700" role="status">
          {reply.data.pesan}
        </p>
      )}
    </form>
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
