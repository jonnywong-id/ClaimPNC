import { useEffect, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useBandingHargaSalvageDecide } from './api'
import { BandingHargaSalvageError, type AppealRow } from './types'

/**
 * Kotak penegasan tombol **Approve** dan **Reject** pada kolom Action.
 *
 * # Kenapa ia kotak tersendiri, bukan isian di dalam sel
 *
 * Di layar lama, catatan komite adalah isian sempit di dalam kolom Action
 * (`Section/ButtonApproveRejectedRequest`), dan kedua tombolnya menyimpan SEKETIKA — satu
 * klik, putusan tercatat. Di sini keduanya dipisah menjadi dua langkah, dan alasannya bukan
 * selera:
 *
 * PERTAMA, putusan ini menyentuh nilai uang dan **tidak dapat dibatalkan** — penyaring
 * `TGLAPPROVE IS NULL` membuat penekanan kedua ditolak, bukan menimpa yang pertama. Tindakan
 * yang tidak dapat ditarik kembali layak dibaca dulu sebelum dijalankan.
 *
 * KEDUA, yang sedang diputuskan adalah SELISIH dua angka, dan pada sel selebar 10rem kedua
 * angka itu tidak pernah terlihat berdampingan. Di sini keduanya digambar bersama beserta
 * selisihnya, sehingga komite memutuskan sambil melihat nilai yang ia putuskan.
 *
 * Langkah tambahan ini **bukan** perubahan aturan bisnis: hasil akhirnya sama persis dengan
 * satu klik di layar lama. Yang bertambah hanya kesempatan membaca sebelum menekan.
 */
type Props = {
  /** Baris yang sedang diputuskan. `null` berarti kotak ini tertutup. */
  row: AppealRow | null

  /** `true` bila yang ditekan Approve, `false` bila Reject. */
  approve: boolean

  onCancel: () => void

  /** Dipanggil setelah putusannya benar-benar tersimpan, membawa pesan dari server. */
  onDecided: (message: string) => void
}

export function DecisionConfirm({ row, approve, onCancel, onDecided }: Props) {
  const [note, setNote] = useState('')
  const decide = useBandingHargaSalvageDecide()

  /**
   * Catatan dikosongkan setiap kali barisnya berganti.
   *
   * Tanpa ini, catatan yang ditulis untuk satu barang akan terbawa ke barang berikutnya —
   * dan ia tersimpan sebagai alasan keputusan atas barang yang salah. Galat sebelumnya ikut
   * dibuang, supaya pesan penolakan barang sebelumnya tidak tampak seperti penolakan barang
   * ini.
   */
  useEffect(() => {
    setNote('')
    decide.reset()
    // Sengaja hanya bergantung pada identitas baris: menyertakan `decide` akan menjalankan
    // efek ini setiap kali keadaan mutasi berubah, termasuk tepat setelah berhasil — yang
    // justru membuang pesan yang baru saja diterima.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [row?.detail_object, approve])

  if (row === null) return null

  const putusan = approve ? 'Approve' : 'Reject'

  function submit() {
    if (row === null) return

    decide.mutate(
      {
        detail_object: row.detail_object,
        id_salvage: row.id_salvage,
        harga_request: row.harga_request,
        catatan: note.trim(),
        setujui: approve,
      },
      { onSuccess: (result) => onDecided(result.pesan) },
    )
  }

  return (
    <section
      className="mt-4 rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut"
      aria-label={`Penegasan ${putusan} banding harga`}
    >
      <header className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold text-slate-900">
            {putusan} banding harga — {row.no_klaim}
          </h2>
          <p className="mt-1 text-xs text-slate-600">
            {row.nama_barang || 'Barang tanpa nama'} · Detail Object {row.detail_object}
          </p>
        </div>
        <Button tone="halus" onClick={onCancel} disabled={decide.isPending}>
          Batal
        </Button>
      </header>

      <PriceComparison row={row} />

      <div className="mt-4">
        <label
          htmlFor="inbox-banding-harga-salvage-catatan"
          className="block text-sm font-medium text-slate-700"
        >
          Note Checker
        </label>
        {/*
          Catatan TIDAK diwajibkan, meniru layar lama — di sana ia isian biasa tanpa penanda
          wajib. Mewajibkannya di sini berarti menambah penghalang yang tidak pernah ada,
          tanpa perubahan aturan yang pernah diputuskan siapa pun.
        */}
        <textarea
          id="inbox-banding-harga-salvage-catatan"
          rows={3}
          value={note}
          disabled={decide.isPending}
          onChange={(event) => setNote(event.target.value)}
          placeholder="Alasan keputusan Anda — boleh dikosongkan."
          className={[
            'mt-1 block w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm',
            'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
            'disabled:bg-slate-50 disabled:text-slate-500',
          ].join(' ')}
        />
        <p className="mt-1 text-xs text-slate-500">
          Tersimpan bersama keputusan Anda dan terbaca kembali di tab History Cheker.
          Keputusan ini <strong>tidak dapat diubah</strong> setelah tersimpan.
        </p>
      </div>

      {decide.isError && (
        <div className="mt-4">
          <ErrorMessage
            title={titleOf(decide.error)}
            description={messageOf(decide.error)}
            tone={toneOf(decide.error)}
          />
        </div>
      )}

      <div className="mt-4 flex flex-wrap gap-2">
        <Button tone="utama" onClick={submit} disabled={decide.isPending}>
          {decide.isPending ? 'Menyimpan…' : `Simpan ${putusan}`}
        </Button>
        <Button tone="halus" onClick={onCancel} disabled={decide.isPending}>
          Batal
        </Button>
      </div>
    </section>
  )
}

/**
 * Kedua angka yang sedang dipertentangkan, beserta selisihnya.
 *
 * Selisihnya dihitung di peramban dan SEMATA untuk dibaca — tidak satu pun yang dikirim
 * kembali ke server. Yang dikirim adalah `harga_request` apa adanya sebagaimana diterima,
 * sehingga pembulatan tampilan tidak pernah menjadi pembulatan nilai (`I-12`).
 */
function PriceComparison({ row }: { row: AppealRow }) {
  const barang = parseMoney(row.harga_barang)
  const request = parseMoney(row.harga_request)
  const terbaca = barang !== null && request !== null

  return (
    <dl className="mt-3 grid gap-3 sm:grid-cols-3">
      <Figure label="Harga Barang (PIC)" value={money(row.harga_barang)} />
      <Figure label="Harga Request (balai lelang)" value={money(row.harga_request)} />
      <Figure
        label="Selisih"
        value={terbaca ? money(String(request - barang)) : '—'}
        tone={terbaca && request < barang ? 'turun' : 'biasa'}
      />
    </dl>
  )
}

/**
 * parseMoney membaca nilai uang, dan mengembalikan `null` bila ia TIDAK terbaca.
 *
 * # Kenapa ia ada, alih-alih memakai `Number()` langsung
 *
 * Karena `Number('')` menghasilkan **0**, dan `Number.isFinite(0)` bernilai benar. Kolom
 * `HARGABARANG` pada tabel checker **boleh kosong** — ada barisnya di data produksi, dengan
 * harga request terisi sementara harga barangnya tidak. Dengan `Number()` polos, selisihnya
 * terhitung `request - 0` dan panel ini akan menampilkan seluruh harga request sebagai
 * "Selisih", seolah harga dasarnya memang nol.
 *
 * Itu kelas cacat yang sama dengan `GETSELISIHJAM` yang `D-49` butir 10 perintahkan
 * diperbaiki: **nol yang tidak dapat dibedakan dari kegagalan membaca**. Di sini akibatnya
 * langsung — komite membaca angka selisih yang salah tepat sebelum memutuskan.
 *
 * Nilai nol yang SUNGGUHAN tetap terbaca sebagai nol; yang ditolak hanya teks kosong dan
 * teks yang bukan angka.
 */
function parseMoney(raw: string): number | null {
  const text = raw.trim()
  if (text === '') return null

  const parsed = Number(text.replace(',', '.'))
  return Number.isFinite(parsed) ? parsed : null
}

function Figure({
  label,
  value,
  tone = 'biasa',
}: {
  label: string
  value: string
  tone?: 'biasa' | 'turun'
}) {
  return (
    <div className="rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd
        className={[
          'mt-0.5 text-sm font-semibold tabular-nums',
          tone === 'turun' ? 'text-red-700' : 'text-slate-900',
        ].join(' ')}
      >
        {value}
      </dd>
    </div>
  )
}

/** money menggambar nilai uang; nilai yang tidak terbaca digambar apa adanya, bukan nol. */
function money(raw: string): string {
  const parsed = parseMoney(raw)
  if (parsed === null) return raw.trim() === '' ? '—' : raw
  return parsed.toLocaleString('id-ID', { maximumFractionDigits: 2 })
}

/**
 * Judul galat dibedakan, karena kedua galat yang paling mungkin menuntut tindakan berbeda.
 *
 * "Sudah diputus" bukan kerusakan — ia jawaban yang sah atas penekanan kedua, atau atas
 * baris yang komite lain sudah tangani. Menampilkannya dengan nada yang sama seperti galat
 * sistem akan membuat pengguna melaporkannya sebagai kerusakan.
 */
function titleOf(error: unknown): string {
  if (error instanceof APIError) {
    if (error.kode === BandingHargaSalvageError.alreadyDecided) {
      return 'Banding ini sudah diputus'
    }
    if (error.kode === BandingHargaSalvageError.validationFail) {
      return 'Keputusan belum dapat disimpan'
    }
  }
  return 'Keputusan tidak tersimpan'
}

function toneOf(error: unknown): 'penolakan' | 'gangguan' {
  if (error instanceof APIError && error.kode === BandingHargaSalvageError.alreadyDecided) {
    return 'gangguan'
  }
  return 'penolakan'
}

/**
 * messageOf mengambil pesan yang layak dibaca pengguna.
 *
 * Pelanggaran per isian ikut digabung: pada galat validasi, pesan utamanya hanya menyebut
 * "perbaiki yang ditandai", dan isian yang ditandai di sini bukan isian yang diketik
 * pengguna — melainkan identitas baris yang dikirim layar. Tanpa menggabungkannya, pengguna
 * membaca perintah memperbaiki sesuatu yang tidak terlihat olehnya.
 */
function messageOf(error: unknown): string {
  if (error instanceof APIError) {
    const violations = Object.values(error.violations())
    if (violations.length > 0) return `${error.message} ${violations.join(' ')}`
    return error.message
  }
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
