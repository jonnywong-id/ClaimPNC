import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useKomunikasiCabangDetail } from './api'
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
 * # DUA ARTEFAKNYA HILANG DARI EXPORT
 *
 * Kueri pemasok grid-nya (`GetInboxKomunikasiCabang_detail`) dan activity tombol Balas
 * (`PNCReplyMessageCabang`) TIDAK ADA di export mana pun — keduanya kelas `R-16`.
 *
 * Yang dibangun karena itu adalah bentuk yang terbaca dari section-nya sendiri. Ia TIDAK
 * mengarang penyaring yang tidak dapat dibaca dari mana pun; yang dipakai hanyalah nomor
 * percakapan, satu-satunya parameter yang benar-benar dikirim tombolnya. Dinyatakan lewat
 * `selisih_terencana`.
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
            {detail.data.tindakan_masih_di_pega && <ReplyBoxNotice />}
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
 */
function Thread({ messages }: { messages: ThreadMessage[] }) {
  if (messages.length === 0) {
    return (
      <p className="text-sm text-slate-500">
        Percakapan ini tidak memuat satu pun pesan.
      </p>
    )
  }

  return (
    <ol className="space-y-3">
      {messages.map((message, index) => (
        <li
          key={`${message.tanggal}|${message.operator_pengirim}|${index}`}
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

          {/*
            Balasan digambar sebagai baris TERSENDIRI di bawah pesannya, bukan sebagai kolom
            keempat.

            Section lama tidak menampilkannya sama sekali, padahal satu baris tabel menyimpan
            pesan DAN balasannya — sehingga utasnya terbaca separuh. Ketiga kolom aslinya
            tetap utuh; yang ditambahkan adalah barisnya. Dinyatakan lewat
            `selisih_terencana`.
          */}
          {message.jawaban !== '' && (
            <div className="mt-2 border-l-2 border-blue-300 pl-3">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="text-xs font-medium text-blue-800">
                  Jawaban{message.penjawab ? ` · ${message.penjawab}` : ''}
                </span>
                <span className="text-xs tabular-nums text-slate-500">
                  {message.tanggal_jawaban || '—'}
                </span>
              </div>
              <p className="mt-1 text-sm whitespace-pre-wrap text-slate-800">
                {message.jawaban}
              </p>
            </div>
          )}
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
 * Keterangan kotak balasan yang belum dapat dipakai.
 *
 * # Kenapa kotaknya TIDAK digambar sama sekali
 *
 * Karena kotak isian yang tampak dapat diketik tetapi menolak saat dikirim lebih buruk
 * daripada tidak ada: pengguna sudah mengetik kalimatnya, dan kalimat itu hilang. Yang
 * digambar karena itu keterangannya saja.
 *
 * Ini BERBEDA dari tombol pada bilah atas, yang tetap digambar — tombol yang hilang membuat
 * layar terbaca rusak, sementara tombol yang menjawab alasan justru mengarahkan. Kotak
 * isian tidak punya sifat itu.
 *
 * Tindakannya sendiri belum dapat dibangun sekalipun diputuskan: activity di balik tombol
 * "Balas" (`PNCReplyMessageCabang`) TIDAK ADA di export mana pun, sehingga tidak ada yang
 * dapat dibaca untuk ditulis ulang.
 */
function ReplyBoxNotice() {
  return (
    <div className="mt-5 rounded-kartu border border-amber-200 bg-amber-50 px-3 py-2.5">
      <p className="text-xs text-slate-700">
        <span className="font-medium">Membalas belum tersedia di sini.</span> Balasan
        menulis ke tabel komunikasi, dan selama Pega dan sistem baru berjalan berdampingan
        tabel itu hanya boleh ditulis satu sistem — hari ini Pega. Balas lewat Pega.
      </p>
    </div>
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
