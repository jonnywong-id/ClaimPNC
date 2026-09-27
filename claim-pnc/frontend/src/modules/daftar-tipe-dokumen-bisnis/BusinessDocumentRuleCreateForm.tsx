import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type BusinessChoice,
  type BusinessDocumentRuleInput,
  type MasterChoice,
} from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

type Props = {
  businesses: BusinessChoice[]
  /** Tombol "Pilih semua" ditampilkan atau tidak — meniru `pyVisible` layar lama. */
  mayBulkSelect: boolean
  /**
   * Baris awal, terisi saat form dibuka lewat Copy.
   *
   * `| undefined` ditulis eksplisit karena proyek ini menyalakan
   * `exactOptionalPropertyTypes` — tanpa itu, meneruskan `undefined` secara sengaja
   * ditolak kompilator.
   */
  initialRules?: BusinessDocumentRuleInput[] | undefined
  documentTypes: MasterChoice[]
  detailDocuments: MasterChoice[]
  objectDocuments: MasterChoice[]
  isSaving: boolean
  error: unknown
  onSave: (businessIDs: string[], rules: BusinessDocumentRuleInput[]) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa sambungan jaringan, lalu simpan sekali lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.businessDocumentRuleBusinessRequired:
        return {
          title: 'Nama Bisnis belum di isi',
          description: 'Pilih sekurang-kurangnya satu lini bisnis sebelum menyimpan.',
          tone: 'penolakan',
        }
      // Tidak ada cabang untuk "baris dokumen kosong", dan itu disengaja: di Pega keadaan
      // itu berlalu diam-diam — perulangannya berputar nol kali lalu layar tertutup.
      // Modul ini menirunya.
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih entitas di bagian atas layar, lalu simpan sekali lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description: 'Hubungi tim infrastruktur bila keadaan ini berlanjut.',
          tone: 'gangguan',
        }
      default:
        return { title: 'Penyimpanan gagal', description: error.message, tone: 'gangguan' }
    }
  }
  return null
}

/** labelFor menyusun teks isian dari kode yang sudah tersimpan. */
function labelFor(list: MasterChoice[], id: string): string {
  if (id === '') return ''
  const matched = list.find((item) => item.id === id)
  // Kode yang tidak ada di master ditampilkan APA ADANYA, bukan dikosongkan —
  // mengosongkannya akan membuat salinan diam-diam kehilangan rujukannya.
  return matched ? `${matched.nama} (${matched.id})` : id
}

/**
 * idFromLabel mencari kode dari teks yang diketik.
 *
 * Kosong bila tidak cocok, dan itu MENIRU Pega: autocomplete-nya
 * ber-`pyAllowFreeFormInput=true`, sehingga teks di luar daftar boleh diketik dan yang
 * terjadi hanyalah kodenya tidak terisi.
 */
function idFromLabel(list: MasterChoice[], typed: string): string {
  const clean = typed.trim()
  if (clean === '') return ''
  const matched = list.find(
    (item) => `${item.nama} (${item.id})` === clean || item.nama === clean || item.id === clean,
  )
  return matched?.id ?? ''
}

/**
 * RowDraft menyimpan TEKS yang diketik petugas berdampingan dengan kode hasil
 * pencariannya.
 *
 * Keduanya harus disimpan, dan itu bukan kelebihan: bila teks isian diturunkan dari
 * kodenya, setiap ketikan yang belum cocok dengan master akan menghasilkan kode kosong —
 * dan isian yang nilainya berasal dari kode itu ikut terhapus setiap kali satu huruf
 * diketik. Isiannya menjadi mustahil diisi.
 *
 * Cacat itu benar-benar terjadi pada versi pertama form ini dan ditangkap pengujian.
 */
type RowDraft = BusinessDocumentRuleInput & {
  tipe_dokumen_teks: string
  object_dokumen_teks: string
  detail_dokumen_teks: string
}

/** emptyRule adalah baris dokumen yang baru ditambahkan dan belum diisi. */
function emptyRule(): RowDraft {
  return {
    id_tipe_dokumen: '',
    id_object_dokumen: '',
    id_detail_dokumen: '',
    detail_dokumen: '',
    status_wajib: false,
    minimum_dokumen: 0,
    tipe_dokumen_teks: '',
    object_dokumen_teks: '',
    detail_dokumen_teks: '',
  }
}

/** toInput membuang teks bantu sebelum baris dikirim ke server. */
function toInput(row: RowDraft): BusinessDocumentRuleInput {
  return {
    id_tipe_dokumen: row.id_tipe_dokumen,
    id_object_dokumen: row.id_object_dokumen,
    id_detail_dokumen: row.id_detail_dokumen,
    detail_dokumen: row.detail_dokumen,
    status_wajib: row.status_wajib,
    minimum_dokumen: row.minimum_dokumen,
  }
}

/**
 * Form penambahan: beberapa lini bisnis dikali beberapa baris dokumen.
 *
 * # Kenapa bentuknya jamak kali jamak
 *
 * Karena begitulah layar lamanya bekerja, dan itu tertulis apa adanya sebagai komentar
 * langkah di `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml`:
 *
 *	"kalo tambah > bisa banyak bisnis & banyak dokumen"
 *
 * Aturan kelengkapan dokumen berulang nyaris sama di banyak lini bisnis, sehingga memaksa
 * petugas memasukkannya satu per satu akan mengubah pekerjaan sepuluh menit menjadi
 * sepuluh jam. Menyederhanakannya menjadi satu bisnis per penyimpanan bukan
 * penyederhanaan melainkan penghapusan fitur.
 *
 * # Yang TIDAK ditiru dari layar lama
 *
 * Tombol "Tamban semua bisnis NONMBU" mengecualikan LIMA kode bisnis yang ditulis
 * langsung di dalam rule (`Activity/SetAllBusiness-Act.xml:984`), dan hanya terlihat oleh
 * operator ber-`pyPosition='NONMBU'`. Keduanya tidak dibawa: kode bisnis di dalam kode
 * adalah persis yang `D-15` perintahkan dihapus, dan kewenangan berdasarkan properti
 * operator adalah model izin yang `D-59` gantikan dengan izin per menu.
 *
 * Penggantinya "Pilih semua" biasa — tanpa pengecualian dan tanpa syarat jabatan.
 * Akibatnya petugas dapat memilih lini MBU yang dulu dikecualikan, dan itu SELISIH
 * TERENCANA yang perlu diketahui Work Owner.
 */
export function BusinessDocumentRuleCreateForm({
  businesses,
  mayBulkSelect,
  initialRules,
  documentTypes,
  detailDocuments,
  objectDocuments,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const [selected, setSelected] = useState<string[]>([])

  // Baris awal dibentuk SEKALI. Bila ia dihitung ulang setiap render, setiap ketikan
  // petugas akan tertimpa oleh baris asal salinannya.
  const [rules, setRules] = useState<RowDraft[]>(() => {
    if (initialRules === undefined || initialRules.length === 0) return [emptyRule()]
    return initialRules.map((rule) => ({
      ...rule,
      tipe_dokumen_teks: labelFor(documentTypes, rule.id_tipe_dokumen),
      object_dokumen_teks: labelFor(objectDocuments, rule.id_object_dokumen),
      detail_dokumen_teks: labelFor(detailDocuments, rule.id_detail_dokumen),
    }))
  })

  // Bisnis yang dilewati pemilihan massal — kelimanya lini MBU. Bisnis yang sama TETAP
  // dapat dicentang satu per satu, persis seperti di Pega.
  const bulkSelectable = businesses.filter((item) => !item.dikecualikan_pilih_semua)

  const message = messageFor(error)

  function toggleBusiness(id: string) {
    setSelected((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    )
  }

  function updateRule(index: number, patch: Partial<RowDraft>) {
    setRules((current) =>
      current.map((rule, position) => (position === index ? { ...rule, ...patch } : rule)),
    )
  }

  function removeRule(index: number) {
    setRules((current) => current.filter((_rule, position) => position !== index))
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        onSave(selected, rules.map(toInput))
      }}
      noValidate
      aria-label="Tambah Data"
      className="space-y-5 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">Tambah Data</h2>

      {message && (
        <ErrorMessage
          title={message.title}
          description={message.description}
          tone={message.tone}
        />
      )}

      <section>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold text-slate-800">
            Nama Bisnis
            <span className="ml-2 font-normal text-slate-500">
              {selected.length} dari {businesses.length} dipilih
            </span>
          </h3>
          <div className="flex gap-2">
            {/*
              Tombolnya hanya tampil bagi pyPosition NONMBU, meniru `pyVisible` layar lama.
              Ia penyembunyian tampilan, bukan kewenangan: bisnis yang sama tetap dapat
              dicentang satu per satu oleh siapa pun yang membuka layar ini.
            */}
            {mayBulkSelect && (
              <Button
                tone="halus"
                onClick={() => setSelected(bulkSelectable.map((item) => item.id))}
              >
                Pilih semua
              </Button>
            )}
            <Button tone="halus" onClick={() => setSelected([])}>
              Kosongkan
            </Button>
          </div>
        </div>

        <div className="mt-2 grid max-h-56 gap-1 overflow-y-auto rounded border border-slate-200 p-3 sm:grid-cols-2 lg:grid-cols-3">
          {businesses.length === 0 ? (
            <p className="text-sm text-slate-500">
              Daftar bisnis tidak dapat dimuat. Aturan tetap dapat disimpan bila daftarnya muncul
              kembali.
            </p>
          ) : (
            businesses.map((business) => (
              <label key={business.id} className="flex items-center gap-2 text-sm text-slate-700">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300"
                  checked={selected.includes(business.id)}
                  onChange={() => toggleBusiness(business.id)}
                />
                <span>
                  {business.nama_bisnis}
                  <span className="ml-1 text-xs text-slate-400">({business.id})</span>
                </span>
              </label>
            ))
          )}
        </div>
      </section>

      <section>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold text-slate-800">Dokumen</h3>
          <Button tone="kedua" onClick={() => setRules((current) => [...current, emptyRule()])}>
            Tambah baris
          </Button>
        </div>

        <div className="mt-2 space-y-3">
          {rules.map((rule, index) => {
            // Pilihan Detail Dokumen menyempit mengikuti Tipe Dokumen pada BARIS YANG SAMA
            // — penyempitan yang di Pega dikerjakan server lewat parameter `idDocument`.
            const narrowedDetails = rule.id_tipe_dokumen
              ? detailDocuments.filter((item) => item.id_induk === rule.id_tipe_dokumen)
              : detailDocuments

            return (
              <div key={index} className="rounded border border-slate-200 p-3">
                <div className="grid gap-3 sm:grid-cols-2">
                  {/*
                    Ketiga isian rujukan di bawah adalah autocomplete ketik-cari, bukan
                    dropdown — meniru `pyAutoComplete` ber-`pyAllowFreeFormInput=true` di
                    Pega. Daftar masternya dapat memuat ratusan baris, dan di layar lama
                    satu-satunya cara memakainya memang dengan mengetik.

                    Teks di luar daftar TETAP boleh diketik; yang terjadi hanyalah kodenya
                    tidak terisi, persis seperti Pega.
                  */}
                  <label className="text-sm">
                    <span className="block font-medium text-slate-700">Tipe Dokumen</span>
                    <input
                      type="text"
                      list={`tipe-dokumen-${index}`}
                      autoComplete="off"
                      className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2"
                      value={rule.tipe_dokumen_teks}
                      onChange={(event) =>
                        // Detail Dokumen ikut dikosongkan saat tahapnya berganti. Tanpa itu,
                        // rincian milik tahap sebelumnya tetap terpilih meski sudah hilang
                        // dari daftar — dan tersimpan sebagai pasangan yang tidak pernah
                        // muncul di layar unggah mana pun.
                        updateRule(index, {
                          tipe_dokumen_teks: event.target.value,
                          id_tipe_dokumen: idFromLabel(documentTypes, event.target.value),
                          id_detail_dokumen: '',
                          detail_dokumen_teks: '',
                        })
                      }
                    />
                    <datalist id={`tipe-dokumen-${index}`}>
                      {documentTypes.map((item) => (
                        <option key={item.id} value={`${item.nama} (${item.id})`} />
                      ))}
                    </datalist>
                  </label>

                  <label className="text-sm">
                    <span className="block font-medium text-slate-700">Object Dokumen</span>
                    <input
                      type="text"
                      list={`objek-dokumen-${index}`}
                      autoComplete="off"
                      className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2"
                      value={rule.object_dokumen_teks}
                      onChange={(event) =>
                        updateRule(index, {
                          object_dokumen_teks: event.target.value,
                          id_object_dokumen: idFromLabel(objectDocuments, event.target.value),
                        })
                      }
                    />
                    <datalist id={`objek-dokumen-${index}`}>
                      {objectDocuments.map((item) => (
                        <option key={item.id} value={`${item.nama} (${item.id})`} />
                      ))}
                    </datalist>
                  </label>
                </div>

                <label className="mt-3 block text-sm">
                  <span className="block font-medium text-slate-700">Detail Dokumen</span>
                  <input
                    type="text"
                    list={`detail-dokumen-${index}`}
                    autoComplete="off"
                    className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2"
                    value={rule.detail_dokumen_teks}
                    onChange={(event) =>
                      updateRule(index, {
                        detail_dokumen_teks: event.target.value,
                        id_detail_dokumen: idFromLabel(detailDocuments, event.target.value),
                      })
                    }
                  />
                  <datalist id={`detail-dokumen-${index}`}>
                    {narrowedDetails.map((item) => (
                      <option key={item.id} value={`${item.nama} (${item.id})`} />
                    ))}
                  </datalist>
                </label>

                <label className="mt-3 block text-sm">
                  <span className="block font-medium text-slate-700">Nama Dokumen</span>
                  <input
                    type="text"
                    autoComplete="off"
                    className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2"
                    value={rule.detail_dokumen}
                    onChange={(event) => updateRule(index, { detail_dokumen: event.target.value })}
                  />
                </label>

                <div className="mt-3 flex flex-wrap items-end justify-between gap-3">
                  <label className="flex items-center gap-2 text-sm text-slate-700">
                    <input
                      type="checkbox"
                      className="h-4 w-4 rounded border-slate-300"
                      checked={rule.status_wajib}
                      onChange={(event) =>
                        updateRule(index, { status_wajib: event.target.checked })
                      }
                    />
                    Status Wajib
                  </label>

                  <label className="text-sm">
                    <span className="block font-medium text-slate-700">Minimum Dokumen</span>
                    <input
                      type="text"
                      inputMode="numeric"
                      autoComplete="off"
                      className="mt-1 w-28 rounded border border-slate-300 bg-white px-3 py-2"
                      value={String(rule.minimum_dokumen)}
                      onChange={(event) =>
                        updateRule(index, {
                          minimum_dokumen: Number(event.target.value.replace(/[^0-9]/g, '')) || 0,
                        })
                      }
                    />
                  </label>

                  {rules.length > 1 && (
                    <Button tone="halus" onClick={() => removeRule(index)}>
                      Buang baris
                    </Button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </section>

      {/*
        Jumlah baris yang akan lahir disebutkan terang-terangan. Penyimpanan ini dapat
        menghasilkan puluhan baris sekaligus, dan tidak ada jalur hapus untuk
        membatalkannya — angka ini satu-satunya kesempatan petugas menyadari bahwa ia
        memilih lebih banyak daripada yang dimaksud.
      */}
      <p className="text-sm text-slate-600">
        Akan tersimpan{' '}
        <span className="font-semibold text-slate-900">{selected.length * rules.length}</span> baris
        aturan ({selected.length} bisnis × {rules.length} dokumen). Baris yang tersimpan{' '}
        <span className="font-medium">tidak dapat dihapus</span> dari layar ini.
      </p>

      <div className="flex flex-wrap justify-end gap-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}
