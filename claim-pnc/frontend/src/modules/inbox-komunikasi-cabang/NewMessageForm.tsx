import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useKomunikasiCabangBranches, useKomunikasiCabangSendMessage } from './api'

type Props = {
  /** Menutup form tanpa mengirim apa pun. */
  onClose: () => void

  /** Dipanggil setelah pesannya benar-benar tersimpan. */
  onSent: (pesan: string) => void
}

/**
 * Form **"Kirim Pesan"** — pembuatan percakapan BARU.
 *
 * # Apa yang digantikan, dan kenapa ia sempat dikira tidak ada
 *
 * Bukan section tersendiri, melainkan **blok tersembunyi** di dalam
 * `Section/InboxKomunikasi-Section.xml` — kontainer `S11` yang bersyarat
 * `pyContainerVisibleWhen = IsUpdateKomunikasi`.
 *
 * Rantai syaratnya dua tingkat:
 *
 *     IsUpdateKomunikasi  ->  evaluateWhen("IsInsertKomunikasi")
 *     IsInsertKomunikasi  ->  TempInputKomunikasi.pyLabel = "Insert"
 *
 * Penandanya diset `"Insert"` oleh `CNMShowInsertKomunikasi_dt` saat "Tambah" ditekan, dan
 * dikosongkan lagi pada langkah terakhir `PNCSendMessageKomunikasiCabang` setelah pesannya
 * terkirim — itulah yang menutup formnya sendiri.
 *
 * Mekanisme itu tidak dibawa: ia cara Pega menyimpan keadaan layar di server. Di sini
 * keadaan layar tinggal di layar, dan yang dibawa adalah PERILAKUNYA — tertutup saat dibuka,
 * terbuka saat "Tambah" ditekan, tertutup lagi setelah terkirim.
 *
 * # Ketiga isiannya, dibaca dari section
 *
 *     pxDropdown      -> TempInputKomunikasi.CaseID     tujuan: PUSAT | CABANG
 *     pxAutoComplete  -> TempInputKomunikasi.CityID     cabang, WAJIB bila tujuannya CABANG
 *     pxTextArea      -> TempInputKomunikasi.City       isi pesan
 *
 * Perhatikan isian ketiga: sebuah properti bernama `City` menyimpan ISI PESAN. Tidak satu pun
 * nama itu dibawa (`D-19`).
 */
export function NewMessageForm({ onClose, onSent }: Props) {
  const [destination, setDestination] = useState('PUSAT')
  const [branch, setBranch] = useState('')
  const [message, setMessage] = useState('')

  // Daftar cabang diambil hanya selama form ini hidup. Mayoritas pembukaan layar tidak
  // berakhir dengan pengiriman pesan, sehingga mengambilnya lebih awal berarti satu
  // perjalanan jaringan yang hampir selalu tidak terpakai.
  const branches = useKomunikasiCabangBranches(true)
  const send = useKomunikasiCabangSendMessage()

  const toBranch = destination === 'CABANG'
  const trimmed = message.trim()

  // Tombolnya menunggu SELURUH isian yang wajib, bukan hanya pesannya. Membiarkannya aktif
  // lalu menolak di peladen memang menghasilkan pesan galat yang benar, tetapi menyuruh
  // pengguna menekan tombol untuk diberi tahu apa yang sudah terlihat kosong di layarnya.
  const ready = trimmed !== '' && (!toBranch || branch !== '') && !send.isPending

  return (
    <form
      className="mt-4 rounded-kartu border border-slate-200 bg-white px-4 py-4 shadow-sm"
      aria-label="Kirim pesan baru"
      onSubmit={(event) => {
        event.preventDefault()
        if (!ready) return

        send.mutate(
          {
            tujuan: destination,
            // Cabang dikosongkan saat tujuannya PUSAT. Peladen mengabaikannya juga, tetapi
            // mengirim isian yang tidak berlaku membuat badan permintaan menyatakan sesuatu
            // yang tidak dimaksudkan penggunanya.
            cabang: toBranch ? branch : '',
            pesan: trimmed,
          },
          {
            onSuccess: (result) => {
              setMessage('')
              setBranch('')
              onSent(result.pesan)
            },
          },
        )
      }}
    >
      <h2 className="text-sm font-semibold text-slate-900">Kirim Pesan</h2>

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <div>
          <label
            htmlFor="komunikasi-tujuan"
            className="block text-xs font-medium text-slate-800"
          >
            Tujuan
          </label>
          <select
            id="komunikasi-tujuan"
            value={destination}
            onChange={(event) => {
              setDestination(event.target.value)
              // Pemilih cabang dikosongkan saat tujuannya berpindah ke PUSAT, supaya
              // pilihannya tidak tertinggal tak terlihat lalu ikut terkirim.
              if (event.target.value !== 'CABANG') setBranch('')
            }}
            className="mt-1.5 w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm text-slate-900 focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
          >
            {(branches.data?.tujuan ?? ['PUSAT', 'CABANG']).map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>

        {/*
          Pemilih cabang digambar HANYA bila tujuannya cabang. Di layar lama ia selalu
          tampak; di sini ia muncul saat relevan, dan itu selisih tampilan yang tidak
          mengubah satu pun baris yang tersimpan.

          Alasannya: isian wajib yang tampak tetapi tidak berlaku adalah sumber kebingungan
          yang lebih besar daripada isian yang muncul saat dibutuhkan.
        */}
        {toBranch && (
          <div>
            <label
              htmlFor="komunikasi-cabang"
              className="block text-xs font-medium text-slate-800"
            >
              Cabang <span className="text-red-600">*</span>
            </label>
            <select
              id="komunikasi-cabang"
              value={branch}
              onChange={(event) => setBranch(event.target.value)}
              disabled={branches.isPending}
              className="mt-1.5 w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm text-slate-900 focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400 disabled:bg-slate-50"
            >
              <option value="">
                {branches.isPending ? 'Memuat cabang…' : 'Pilih cabang…'}
              </option>
              {(branches.data?.cabang ?? []).map((option) => (
                <option key={option.kode} value={option.kode}>
                  {option.nama}
                </option>
              ))}
            </select>

            {branches.isSuccess && branches.data.cabang.length === 0 && (
              <p className="mt-1.5 text-xs text-amber-800">
                Tidak ada cabang yang dapat dipilih. Laporkan ke tim teknis — daftar ini
                dibaca dari data surveyor, dan kosongnya berarti sumber itu tidak terbaca.
              </p>
            )}
          </div>
        )}
      </div>

      <div className="mt-3">
        <label htmlFor="komunikasi-pesan" className="block text-xs font-medium text-slate-800">
          Pesan
        </label>
        <textarea
          id="komunikasi-pesan"
          value={message}
          onChange={(event) => setMessage(event.target.value)}
          rows={3}
          maxLength={4000}
          placeholder="Tulis pesan untuk penerima…"
          className="mt-1.5 w-full rounded-kartu border border-slate-300 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
        />
      </div>

      <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs text-slate-500">
          {message.length >= 3500 ? `${message.length} / 4.000 karakter` : ''}
        </span>

        <div className="flex flex-wrap gap-2">
          <Button type="submit" disabled={!ready}>
            {send.isPending ? 'Mengirim…' : 'Kirim Pesan'}
          </Button>
          <Button tone="kedua" onClick={onClose} disabled={send.isPending}>
            Batal
          </Button>
        </div>
      </div>

      {branches.isError && (
        <div className="mt-3">
          <ErrorMessage
            title="Daftar cabang tidak dapat dimuat"
            description={messageOf(branches.error)}
            tone="gangguan"
          />
        </div>
      )}

      {send.isError && (
        <div className="mt-3">
          <ErrorMessage
            title="Pesan tidak terkirim"
            description={messageOf(send.error)}
            tone="gangguan"
          />
        </div>
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
