import { useEffect, useId, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type Clause,
  type ClauseBusiness,
  type ClauseCategory,
  type ClauseInput,
} from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { BusinessPicker } from './BusinessPicker'

type Props = {
  /** Baris yang sedang disunting; null berarti mode tambah. */
  edited: Clause | null
  /** Sedang memuat isi baris yang disunting. */
  isLoading: boolean
  category: ClauseCategory[]
  isSaving: boolean
  error: unknown
  onSave: (input: ClauseInput) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot pada isiannya (lihat violationsOf).
 * Yang ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian tertentu,
 * karena itulah yang tidak dapat diperbaiki pengguna dengan mengetik.
 */
function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.validationFailed:
        return Object.keys(error.violations()).length > 0
          ? null
          : { title: 'Belum dapat disimpan', description: error.message, tone: 'penolakan' }
      case ErrorCode.notFound:
        return {
          title: 'Pasal ini sudah tidak ada',
          description:
            'Mungkin sudah dihapus petugas lain. Muat ulang daftarnya — penghapusan di layar ini permanen.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih portal entitas di bagian atas halaman, lalu simpan lagi.',
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
        return {
          title: 'Terjadi kesalahan pada sistem',
          description: 'Isian Anda belum tersimpan. Coba beberapa saat lagi.',
          tone: 'gangguan',
        }
    }
  }
  return null
}

/** Mengambil pelanggaran per isian dari galat validasi server. */
function violationsOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {}
}

/**
 * ClauseForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: blok "TAMBAH DATA" dan "UBAH DATA" pada
 * `Section/BrowsePasalDeatailMaster-Section.xml` memuat isian yang sama persis. Isian yang
 * dibandingkan pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * # Urutan isiannya mengikuti layar lama
 *
 *	No Pasal · Deskripsi · Kategori · Bisnis · Isi Pasal
 *
 * Itu urutan sel pada section aslinya, dan ia dipertahankan apa adanya (`D-13`) meski
 * "Isi Pasal" di akhir terasa tidak lazim — ia isian terpanjang, dan menaruhnya di bawah
 * membuat isian pendek tetap terlihat bersamaan.
 *
 * # Satu-satunya isian wajib adalah No Pasal
 *
 * `Activity/CNMInsertPasalDataMaster-Act.xml` memeriksa tepat satu hal, dan menolak dengan
 * "Silahkan ISI No Pasal Terlebih Dahulu". Tidak ada pemeriksaan lain — Isi Pasal boleh
 * kosong, Kategori boleh tidak dipilih, dan No Pasal boleh kembar. Keputusan Work Owner
 * 2026-09-19: "jalankan as is".
 *
 * Karena aturannya tinggal satu, form ini TIDAK memakai zod maupun React Hook Form seperti
 * form master lain. Memasang keduanya untuk satu aturan berarti tiga lapis perantara demi
 * satu perbandingan dengan teks kosong — dan isian Bisnis yang berupa daftar justru lebih
 * jernih dikelola sebagai state biasa. Server tetap memeriksa ulang seluruhnya.
 */
export function ClauseForm({
  edited,
  isLoading,
  category,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const editMode = edited !== null
  const categoryId = useId()

  const [number, setNumber] = useState('')
  const [text, setText] = useState('')
  const [description, setDescription] = useState('')
  const [categoryCode, setCategoryCode] = useState('')
  const [business, setBusiness] = useState<ClauseBusiness[]>([])
  const [localError, setLocalError] = useState('')

  // Isian disesuaikan ketika baris yang disunting berganti — termasuk saat isinya baru
  // selesai dimuat dari server, karena daftar TIDAK memuat lini bisnis dan isian itu baru
  // ada setelah satu baris dibaca tersendiri.
  useEffect(() => {
    setNumber(edited?.no_pasal ?? '')
    setText(edited?.isi_pasal ?? '')
    setDescription(edited?.deskripsi ?? '')
    setCategoryCode(edited?.kategori ?? '')
    setBusiness(edited?.bisnis ?? [])
    setLocalError('')
  }, [edited])

  const serverViolation = violationsOf(error)
  const numberError = localError || serverViolation['no_pasal'] || ''
  const message = messageFor(error)
  const title = editMode ? 'Ubah Pasal Kerugian' : 'Tambah Pasal Kerugian'

  /**
   * Pilihan Kategori, ditambah satu pilihan bayangan bila kode tersimpan tidak dikenal.
   *
   * Baris lama dapat memuat kode selain "1", "2", dan kosong — daftar pilihan aslinya
   * tidak ada di export (`R-16`), sehingga kode lain memang mungkin ada. Tanpa pilihan
   * bayangan ini, membuka baris semacam itu akan menampilkan dropdown yang tampak
   * "Notifikasi", dan menyimpannya akan DIAM-DIAM mengganti kodenya.
   *
   * Dengan pilihan bayangan, kode aslinya tetap terpilih dan tetap tersimpan utuh selama
   * pengguna tidak sengaja menggantinya.
   */
  const knownCode = new Set(category.map((option) => option.kode))
  const options = [
    ...category.map((option) => ({ value: option.kode, label: option.nama })),
    ...(categoryCode !== '' && !knownCode.has(categoryCode)
      ? [{ value: categoryCode, label: `Notifikasi (kode lama ${categoryCode})` }]
      : []),
  ]

  function submit(event: React.FormEvent) {
    event.preventDefault()

    // Pemeriksaan yang sama persis dengan server, supaya pengguna tidak perlu menunggu
    // satu perjalanan jaringan untuk mengetahui isian yang jelas belum terisi. Yang
    // BERWENANG tetap server — ia memeriksanya lagi.
    if (number.trim() === '') {
      setLocalError('Silahkan ISI No Pasal Terlebih Dahulu.')
      return
    }
    setLocalError('')

    onSave({
      no_pasal: number.trim(),
      isi_pasal: text.trim(),
      deskripsi: description.trim(),
      kategori: categoryCode,
      bisnis: business,
    })
  }

  return (
    <form
      onSubmit={submit}
      noValidate
      aria-label={title}
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">{title}</h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — nomornya diterbitkan server dari isi tabel. */}
      {editMode && (
        <p className="text-sm text-slate-600">
          ID: <span className="font-medium text-slate-900">{edited.id}</span>
        </p>
      )}

      {/* Pemuatan disebutkan terang-terangan: daftar tidak memuat lini bisnis, sehingga
          isian Bisnis benar-benar kosong sampai baris itu selesai dibaca. Tanpa keterangan
          ini, jeda singkat itu terbaca sebagai "pasal ini memang tidak punya lini bisnis". */}
      {isLoading && <p className="text-sm text-slate-500">Memuat isi pasal…</p>}

      <Field
        id="no_pasal"
        label="No Pasal"
        type="text"
        value={number}
        disabled={isLoading}
        error={numberError}
        onChange={(e) => setNumber(e.target.value)}
      />

      <Field
        id="deskripsi"
        label="Deskripsi"
        type="text"
        value={description}
        disabled={isLoading}
        error={serverViolation['deskripsi']}
        onChange={(e) => setDescription(e.target.value)}
      />

      <SelectField
        id={categoryId}
        label="Kategori"
        options={options}
        value={categoryCode}
        disabled={isLoading}
        error={serverViolation['kategori']}
        onChange={(e) => setCategoryCode(e.target.value)}
      />

      <BusinessPicker value={business} onChange={setBusiness} disabled={isLoading} />

      {/* Isi Pasal berupa textarea, mengikuti `pxTextArea` pada section aslinya. Komponen
          bersama `Field` hanya melayani <input>, dan menaikkan textarea ke sana berarti
          menebak bentuk yang dibutuhkan modul lain sebelum modul itu ada. */}
      <div>
        <label htmlFor="isi_pasal" className="block text-sm font-medium text-slate-700">
          Isi Pasal
        </label>
        <textarea
          id="isi_pasal"
          rows={6}
          value={text}
          disabled={isLoading}
          aria-invalid={serverViolation['isi_pasal'] ? 'true' : 'false'}
          onChange={(e) => setText(e.target.value)}
          className={[
            'mt-1.5 w-full rounded-kontrol border bg-white px-3 py-2.5 text-sm text-slate-900',
            'placeholder:text-slate-400',
            'transition-[border-color,box-shadow,background-color] duration-150 ease-halus',
            'focus:outline-none focus:ring-4',
            'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
            serverViolation['isi_pasal']
              ? 'border-red-400 focus:border-red-500 focus:ring-red-500/15'
              : 'border-slate-300 hover:border-slate-400 focus:border-blue-500 focus:ring-blue-500/15',
          ].join(' ')}
        />
        {serverViolation['isi_pasal'] && (
          <p className="mt-1.5 text-sm text-red-700" role="alert">
            {serverViolation['isi_pasal']}
          </p>
        )}
      </div>

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        <Button type="submit" tone="utama" disabled={isSaving || isLoading}>
          {isSaving ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}
