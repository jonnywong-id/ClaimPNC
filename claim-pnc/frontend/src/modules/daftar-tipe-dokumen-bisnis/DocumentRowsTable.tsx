import type { MasterChoice } from '@/api/types'
import { Button } from '@/components/Button'

/**
 * RowDraft menyimpan TEKS yang diketik petugas berdampingan dengan kode hasil
 * pencariannya, ditambah `id` baris bila ia sudah tersimpan.
 *
 * Teks dan kode keduanya harus disimpan, dan itu bukan kelebihan: bila teks isian
 * diturunkan dari kodenya, setiap ketikan yang belum cocok dengan master menghasilkan kode
 * kosong — dan isian yang nilainya berasal dari kode itu ikut terhapus setiap kali satu
 * huruf diketik. Isiannya menjadi mustahil diisi.
 *
 * Cacat itu benar-benar terjadi pada versi pertama form ini dan ditangkap pengujian.
 *
 * `id` kosong berarti baris BARU. Layar Tambah dan Copy selalu mengosongkannya; layar Ubah
 * mengisinya untuk baris yang sudah ada dan mengosongkannya untuk baris yang baru
 * ditambahkan — campuran itu memang bentuk penyimpanan sistem lama.
 */
export type RowDraft = {
  id: string
  id_tipe_dokumen: string
  id_object_dokumen: string
  id_detail_dokumen: string
  detail_dokumen: string
  status_wajib: boolean
  minimum_dokumen: number
  tipe_dokumen_teks: string
  object_dokumen_teks: string
  detail_dokumen_teks: string
}

/** emptyRow adalah baris dokumen yang baru ditambahkan dan belum diisi. */
export function emptyRow(): RowDraft {
  return {
    id: '',
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

/** labelFor menyusun teks isian dari kode yang sudah tersimpan. */
export function labelFor(list: MasterChoice[], id: string): string {
  if (id === '') return ''
  const matched = list.find((item) => item.id === id)
  // Kode yang tidak ada di master ditampilkan APA ADANYA, bukan dikosongkan —
  // mengosongkannya akan membuat baris diam-diam kehilangan rujukannya saat disimpan ulang.
  return matched ? `${matched.nama} (${matched.id})` : id
}

/**
 * idFromLabel mencari kode dari teks yang diketik.
 *
 * Kosong bila tidak cocok, dan itu MENIRU Pega: autocomplete-nya
 * ber-`pyAllowFreeFormInput=true`, sehingga teks di luar daftar boleh diketik dan yang
 * terjadi hanyalah kodenya tidak terisi.
 */
export function idFromLabel(list: MasterChoice[], typed: string): string {
  const clean = typed.trim()
  if (clean === '') return ''
  const matched = list.find(
    (item) => `${item.nama} (${item.id})` === clean || item.nama === clean || item.id === clean,
  )
  return matched?.id ?? ''
}

const CELL_INPUT =
  'w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500'

/** Isian yang teksnya terisi tetapi kodenya tidak ditemukan — ditandai, bukan ditolak. */
const CELL_INPUT_UNRESOLVED =
  'w-full rounded border border-amber-400 bg-amber-50 px-2 py-1.5 text-sm focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500'

/**
 * cellClass memilih gaya isian menurut apakah teksnya sudah mengenali kodenya.
 *
 * # Kenapa ditandai, bukan dikosongkan atau ditolak
 *
 * Autocomplete di Pega ber-`pyAllowFreeFormInput=true`: teks di luar daftar boleh diketik,
 * dan yang terjadi hanyalah kodenya tidak terisi. Menolaknya akan menambah validasi yang
 * tidak ada di layar lama; mengosongkan teksnya membuat isian mustahil diketik.
 *
 * Tetapi MEMBIARKANNYA DIAM terbukti berbahaya. Baris berkode kosong tetap tersimpan, lalu
 * hilang dari seluruh layar — pada 2026-10-04 inilah yang terbaca sebagai "tersimpan tetapi
 * tidak muncul, tanpa pesan galat". Penandaan ini satu-satunya kesempatan petugas
 * menyadarinya sebelum menekan Simpan.
 */
function cellClass(text: string, code: string): string {
  return text.trim() !== '' && code === '' ? CELL_INPUT_UNRESOLVED : CELL_INPUT
}

type Props = {
  rows: RowDraft[]
  documentTypes: MasterChoice[]
  detailDocuments: MasterChoice[]
  objectDocuments: MasterChoice[]
  onChange: (index: number, patch: Partial<RowDraft>) => void
  onAdd: () => void
  onRemove: (index: number) => void
  /**
   * Tindakan tambahan di ujung kanan setiap baris — dipakai layar Ubah untuk tombol Jenis
   * Klaim. Mengembalikan null berarti baris itu tidak punya tindakan apa pun.
   */
  rowAction?: ((row: RowDraft, index: number) => React.ReactNode) | undefined
}

/**
 * Tabel baris dokumen, dipakai BERSAMA oleh layar Tambah, Copy, dan Ubah.
 *
 * # Kenapa tabel, bukan kartu bertumpuk
 *
 * Karena begitulah bentuknya di layar lama, dan bentuk itu bukan selera: petugas membaca
 * belasan baris aturan satu lini bisnis sekaligus dan membandingkannya kolom demi kolom.
 * Versi pertama modul ini menumpuknya sebagai kartu, yang membuat perbandingan itu
 * mustahil — satu baris memakan seluruh tinggi layar.
 *
 * Urutan kolomnya ditiru apa adanya dari layar yang berjalan:
 *
 *	Tipe Dokumen | Detail Dokumen | Object Dokumen | Nama File | Status Wajib | Minimum Dokumen
 *
 * # "Nama File" tidak ada di export
 *
 * Label itu TIDAK ditemukan di `Section/InputListDetailTypeDocumentBusiness_sect` maupun
 * `Section/BrowseListDetailTypeDocumentBusiness_sect` — keduanya nol kecocokan untuk
 * `nama file`, `NAMA_FILE`, maupun pola sejenis. Layar yang berjalan hari ini karena itu
 * LEBIH BARU daripada snapshot export yang menjadi dasar migrasi ini (`R-16`).
 *
 * Yang dipakai di sini adalah kolom `DETAIL_DOKUMEN` pada baris itu sendiri — satu-satunya
 * teks bebas yang tersimpan per baris, dan satu-satunya kandidat yang tersisa setelah
 * `DOC_TYPE_DT_ID` terpakai oleh kolom "Detail Dokumen" di sebelahnya. Pemetaan itu
 * DUGAAN, bukan temuan, dan perlu dikonfirmasi Work Owner sebelum gerbang 1.
 */
export function DocumentRowsTable({
  rows,
  documentTypes,
  detailDocuments,
  objectDocuments,
  onChange,
  onAdd,
  onRemove,
  rowAction,
}: Props) {
  return (
    <section>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-semibold text-slate-800">Dokumen</h3>
        <Button tone="kedua" onClick={onAdd}>
          Tambah baris
        </Button>
      </div>

      {/*
        Tabelnya digulung mendatar pada layar sempit, bukan dipecah menjadi kartu. Enam
        kolom tidak muat di lebar telepon, dan mengubah bentuknya di sana akan membuat
        petugas yang sama melihat dua susunan berbeda (`D-12`).
      */}
      <div className="mt-2 overflow-x-auto rounded border border-slate-200">
        <table className="w-full min-w-[56rem] border-collapse text-left">
          <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-600">
            <tr>
              <th scope="col" className="px-2 py-2 font-medium">
                Tipe Dokumen
              </th>
              <th scope="col" className="px-2 py-2 font-medium">
                Detail Dokumen
              </th>
              <th scope="col" className="px-2 py-2 font-medium">
                Object Dokumen
              </th>
              <th scope="col" className="px-2 py-2 font-medium">
                Nama File
              </th>
              <th scope="col" className="w-28 px-2 py-2 font-medium">
                Status Wajib
              </th>
              <th scope="col" className="w-32 px-2 py-2 font-medium">
                Minimum Dokumen
              </th>
              <th scope="col" className="w-40 px-2 py-2 text-right font-medium">
                <span className="sr-only">Aksi</span>
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {rows.map((row, index) => {
              // Pilihan Detail Dokumen menyempit mengikuti Tipe Dokumen pada BARIS YANG
              // SAMA — penyempitan yang di Pega dikerjakan server lewat parameter
              // `idDocument` pada autocomplete `BrowseVLstDetTypeDoc_RD`.
              const narrowedDetails = row.id_tipe_dokumen
                ? detailDocuments.filter((item) => item.id_induk === row.id_tipe_dokumen)
                : detailDocuments

              return (
                <tr key={index} className="align-top">
                  <td className="px-2 py-2">
                    {/*
                      Ketiga isian rujukan adalah autocomplete ketik-cari, bukan dropdown —
                      meniru `pyAutoComplete` ber-`pyAllowFreeFormInput=true` di Pega.
                      Masternya dapat memuat ratusan baris, dan teks di luar daftar TETAP
                      boleh diketik; yang terjadi hanyalah kodenya tidak terisi.
                    */}
                    <input
                      type="text"
                      list={`tipe-dokumen-${index}`}
                      autoComplete="off"
                      aria-label={`Tipe Dokumen baris ${index + 1}`}
                      className={cellClass(row.tipe_dokumen_teks, row.id_tipe_dokumen)}
                      value={row.tipe_dokumen_teks}
                      onChange={(event) =>
                        // Detail Dokumen ikut dikosongkan saat tahapnya berganti. Tanpa itu,
                        // rincian milik tahap sebelumnya tetap terpilih meski sudah hilang
                        // dari daftar — dan tersimpan sebagai pasangan yang tidak pernah
                        // muncul di layar unggah mana pun.
                        onChange(index, {
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
                  </td>

                  <td className="px-2 py-2">
                    <input
                      type="text"
                      list={`detail-dokumen-${index}`}
                      autoComplete="off"
                      aria-label={`Detail Dokumen baris ${index + 1}`}
                      className={cellClass(row.detail_dokumen_teks, row.id_detail_dokumen)}
                      value={row.detail_dokumen_teks}
                      onChange={(event) =>
                        onChange(index, {
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
                  </td>

                  <td className="px-2 py-2">
                    <input
                      type="text"
                      list={`objek-dokumen-${index}`}
                      autoComplete="off"
                      aria-label={`Object Dokumen baris ${index + 1}`}
                      className={cellClass(row.object_dokumen_teks, row.id_object_dokumen)}
                      value={row.object_dokumen_teks}
                      onChange={(event) =>
                        onChange(index, {
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
                  </td>

                  <td className="px-2 py-2">
                    <input
                      type="text"
                      autoComplete="off"
                      aria-label={`Nama File baris ${index + 1}`}
                      className={CELL_INPUT}
                      value={row.detail_dokumen}
                      onChange={(event) => onChange(index, { detail_dokumen: event.target.value })}
                    />
                  </td>

                  <td className="px-2 py-2">
                    {/*
                      Dropdown YA/TIDAK, bukan kotak centang. Keduanya menyimpan nilai yang
                      sama, tetapi bentuknya ditiru dari layar lama: centang yang tidak
                      tercentang terbaca sebagai "belum diisi", sedangkan TIDAK terbaca
                      sebagai jawaban. Pada aturan yang menentukan dokumen wajib atau tidak,
                      bedanya nyata.
                    */}
                    <select
                      aria-label={`Status Wajib baris ${index + 1}`}
                      className={CELL_INPUT}
                      value={row.status_wajib ? 'YA' : 'TIDAK'}
                      onChange={(event) =>
                        onChange(index, { status_wajib: event.target.value === 'YA' })
                      }
                    >
                      <option value="YA">YA</option>
                      <option value="TIDAK">TIDAK</option>
                    </select>
                  </td>

                  <td className="px-2 py-2">
                    <input
                      type="text"
                      inputMode="numeric"
                      autoComplete="off"
                      aria-label={`Minimum Dokumen baris ${index + 1}`}
                      className={CELL_INPUT}
                      value={String(row.minimum_dokumen)}
                      onChange={(event) =>
                        onChange(index, {
                          minimum_dokumen:
                            Number(event.target.value.replaceAll(/[^0-9]/g, '')) || 0,
                        })
                      }
                    />
                  </td>

                  <td className="px-2 py-2 text-right">
                    <div className="flex flex-wrap justify-end gap-1">
                      {rowAction?.(row, index)}
                      {rows.length > 1 && (
                        <Button
                          tone="halus"
                          onClick={() => onRemove(index)}
                          aria-label={`Buang baris ${index + 1}`}
                        >
                          Buang
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/*
        "Buang" hanya mengeluarkan baris dari layar. Baris yang SUDAH tersimpan tetap ada
        di basis data — tidak ada satu pun jalur hapus terhadap tabel ini di seluruh sistem
        lama, dan `D-66` melarang penghapusan fisik. Dikatakan terang-terangan supaya
        petugas tidak mengira ia baru saja menghapus aturan.
      */}
      {/*
        Peringatan yang terbaca, bukan sekadar warna. Isian bertanda kuning adalah isian
        yang teksnya terisi tetapi KODENYA tidak ditemukan — dan baris seperti itu tersimpan
        tanpa rujukan, lalu hilang dari seluruh layar.
      */}
      {rows.some(
        (row) =>
          (row.tipe_dokumen_teks.trim() !== '' && row.id_tipe_dokumen === '') ||
          (row.detail_dokumen_teks.trim() !== '' && row.id_detail_dokumen === '') ||
          (row.object_dokumen_teks.trim() !== '' && row.id_object_dokumen === ''),
      ) && (
        <p className="mt-2 rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-900">
          Isian bertanda kuning belum cocok dengan daftarnya, sehingga kodenya{' '}
          <span className="font-medium">tidak akan tersimpan</span>. Pilih dari daftar yang
          muncul saat mengetik.
        </p>
      )}

      <p className="mt-2 text-xs text-slate-500">
        Buang hanya mengeluarkan baris dari layar ini. Baris yang sudah tersimpan tidak dapat
        dihapus.
      </p>
    </section>
  )
}
