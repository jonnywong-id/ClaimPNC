import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useKomunikasiCabangDetail, useKomunikasiCabangReply } from './api'
import type { Attachment, ThreadMessage } from './types'

type Props = {
  /** Nomor percakapan yang sedang dibuka. */
  id: string
  onClose: () => void
}

/**
 * Layar **Detail Komunikasi** — panel utas satu percakapan.
 *
 * # Apa yang digantikan
 *
 * Tombol "Detail Komunikasi" pada grid menjalankan
 * `Data Transform/DetailKomunikasi_dt-DT.xml`, yang menyiapkan dua parameter — nomor
 * percakapan dan kode cabangnya — lalu flow action `DETAILKOMUNIKASICABANG_11` ("DETAIL
 * KOMUNIKASI CABANG") menyisipkan `Section/BalasKomunikasiCabang-Section.xml`.
 *
 * Section itu menggambar TIGA kolom — Tanggal, Pengirim, Pesan — beserta kotak "Masukkan
 * Balasan" dan tombol "Balas".
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
 *
 * # Kenapa PANEL, bukan halaman tujuan
 *
 * Karena yang dibuka bukan halaman baru melainkan isi sebuah baris yang sedang dilihat, dan
 * petugas kembali ke daftarnya begitu selesai membaca. Panel di halaman yang sama menjaga
 * tab dan nomor halaman tetap di tempatnya.
 */
export function ConversationDetail({ id, onClose }: Props) {
  const detail = useKomunikasiCabangDetail(id)

  return (
    <section
      className="mt-6 rounded-kartu border border-slate-200 bg-white shadow-sm"
      aria-label={`Detail komunikasi ${id}`}
    >
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold text-slate-900">Detail Komunikasi</h2>
          <p className="mt-1 text-xs text-slate-600">
            Percakapan <span className="font-medium tabular-nums">{id}</span>
            {detail.data?.asal ? ` · dari ${detail.data.asal}` : ''}
          </p>
        </div>
        <Button tone="kedua" onClick={onClose}>
          Tutup
        </Button>
      </header>

      <div className="px-4 py-4">
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
    </section>
  )
}

/**
 * Utas pesan.
 *
 * # Kenapa daftar, bukan tabel
 *
 * Section lama menggambarnya sebagai grid tiga kolom. Di sini ia digambar sebagai utas,
 * dan alasannya satu: kolom "Pesan" berisi kalimat utuh, bukan nilai pendek. Di dalam sel
 * tabel, kalimat itu terpotong atau memaksa barisnya setinggi beberapa baris — dan
 * percakapan yang terpotong tidak dapat dibaca sebagai percakapan.
 *
 * Ketiga isian section tetap digambar seluruhnya dan dengan judul yang sama; yang berubah
 * hanya tata letaknya. Itu selisih tampilan, bukan selisih isi.
 *
 * # Balasan adalah UCAPAN, bukan isian
 *
 * Sampai 2026-09-24 komponen ini menggambar balasan sebagai blok tersendiri DI DALAM setiap
 * ucapan, karena utasnya dibaca dari tabel percakapan — yang menyimpan pesan dan balasan pada
 * satu baris.
 *
 * Keterangan Work Owner mengoreksinya: utas dibaca dari tabel RIWAYAT, tempat setiap pesan
 * dan setiap balasan menempati barisnya sendiri. Blok balasan karena itu dicabut — ia kini
 * satu `<li>` seperti ucapan lainnya.
 */
function Thread({ messages }: { messages: ThreadMessage[] }) {
  if (messages.length === 0) {
    return (
      <p className="text-sm text-slate-500">
        Percakapan ini belum memuat satu pun ucapan. Ia mungkin dibuat lewat layar lain, atau
        sebelum riwayat percakapan mulai dicatat.
      </p>
    )
  }

  return (
    <ol className="space-y-3">
      {messages.map((message, index) => (
        <li
          key={`${message.tanggal}|${message.pengirim}|${index}`}
          className="rounded-kartu border border-slate-200 bg-slate-50 px-3 py-2.5"
        >
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <span className="text-xs font-medium text-slate-800">
              {message.pengirim || '—'}
            </span>
            <span className="text-xs tabular-nums text-slate-500">
              {message.tanggal || '—'}
            </span>
          </div>

          <p className="mt-1.5 text-sm whitespace-pre-wrap text-slate-800">
            {message.pesan || '—'}
          </p>
        </li>
      ))}
    </ol>
  )
}

/**
 * Daftar lampiran percakapan.
 *
 * Sumbernya `RDB List/GetDocumentKomunikasi1-SQL.xml`. Keputusan Work Owner 2026-09-24:
 * lampiran ikut dibangun, BACA-SAJA.
 *
 * # Kenapa status unggahnya datang dari server
 *
 * Karena kueri lama menyimpulkannya dari sebuah pencacah yang membandingkan hasil hitung
 * dengan nol secara TERBALIK, sehingga jawabannya selalu "Belum Upload" berapa pun isinya.
 * Cacat itu tidak dibawa, dan kesimpulan penggantinya hidup di satu tempat — di server —
 * supaya tidak ada dua kesimpulan yang dapat menyimpang.
 */
function Attachments({ items }: { items: Attachment[] }) {
  return (
    <section className="mt-5 border-t border-slate-200 pt-4">
      <h3 className="text-xs font-semibold tracking-wide text-slate-700 uppercase">
        Lampiran
      </h3>

      {items.length === 0 ? (
        <p className="mt-2 text-sm text-slate-500">
          Tidak ada lampiran pada percakapan ini.
        </p>
      ) : (
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
                  ? `Sudah Upload · ${item.tanggal_unggah}`
                  : 'Belum Upload'}
              </span>
            </li>
          ))}
        </ul>
      )}

      {/*
        Ketiadaan tombol unduh dinyatakan, bukan dibiarkan terbaca sebagai kelalaian.

        Berkasnya hidup di penyimpanan dokumen internal (`D-16`), dan seam pengambilnya belum
        dibangun di modul ini. Yang ada di tabel hanyalah metadata-nya.
      */}
      {items.length > 0 && (
        <p className="mt-2 text-xs text-slate-500">
          Berkasnya belum dapat diunduh dari sini — yang tersimpan di tabel ini hanya
          keterangannya. Unduh lewat Pega bila isinya dibutuhkan.
        </p>
      )}
    </section>
  )
}

/**
 * Kotak **"Masukkan Balasan"** beserta tombol **"Balas"**.
 *
 * # Apa yang digantikan
 *
 * `Section/BalasKomunikasiCabang-Section.xml` menggambar satu kotak isian berlabel "Masukkan
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

      <textarea
        id={`balasan-${id}`}
        value={message}
        onChange={(event) => setMessage(event.target.value)}
        rows={3}
        maxLength={4000}
        placeholder="Tulis balasan untuk percakapan ini…"
        className="mt-1.5 w-full rounded-kartu border border-slate-300 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
      />

      <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
        {/*
          Pencacah huruf digambar SETELAH 3.500, bukan sepanjang waktu. Angka yang selalu
          terlihat mengalihkan perhatian dari menulis; yang muncul saat batasnya mendekat
          justru memberi tahu tepat ketika ia berguna.
        */}
        <span className="text-xs text-slate-500">
          {message.length >= 3500 ? `${message.length} / 4.000 karakter` : ''}
        </span>

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
