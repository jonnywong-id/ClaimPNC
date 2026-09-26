import { APIError, NetworkError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import type { DetailBarang, DetailKey, DetailResponse } from './types'

/**
 * Baris grid barang beserta urutannya.
 *
 * `DataTable.rowKey` hanya menerima barisnya, sedangkan dua barang boleh bernama sama
 * dengan isi yang sama pula — dua baut, dicatat dua kali. Urutan ditempelkan di sini
 * supaya kuncinya tetap tunggal tanpa mengubah komponen bersama.
 */
type BarangBernomor = DetailBarang & { urutan: number }

type Props = {
  /** Dengan apa panel ini dibuka — ID pengajuan, atau nomor klaim. */
  detailKey: DetailKey

  /** Nilai kuncinya: ID pengajuan pada jalur `'pengajuan'`, nomor klaim pada `'klaim'`. */
  reference: string

  /**
   * Hasil permintaan rincian, DIPEGANG PEMANGGIL.
   *
   * Komponen ini tidak menembak server sendiri, dan itu bukan kerapian: halaman perlu
   * membaca jawabannya untuk memutuskan menggambar panel ini atau form "Menambahkan Data
   * Salvage" — klaim yang belum punya pengajuan masuk ke form, bukan ke panel. Bila
   * komponen ini menembaknya sendiri, permintaan yang sama berjalan dua kali dan
   * keputusan itu diambil atas jawaban yang belum tentu sama.
   */
  data: DetailResponse | undefined
  isPending: boolean
  isError: boolean
  error: unknown

  onClose: () => void
}

/**
 * Satu isian pada kepala panel.
 *
 * Label dan isinya ditumpuk, bukan disejajarkan dalam satu baris, supaya isian yang panjang
 * — lokasi salvage, remark — tidak memaksa seluruh kolom melebar.
 */
function Isian({ label, value }: { label: string; value: string }) {
  const clean = value.trim()

  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-0.5 break-words text-sm text-slate-900">
        {clean === '' ? <span className="text-slate-400">—</span> : clean}
      </dd>
    </div>
  )
}

/**
 * Satu kelompok isian, dengan judulnya.
 *
 * Kelompoknya mengikuti pengelompokan layar lama, bukan urutan kolom di basis data: layar
 * lama menempatkan data pengajuan, data akseptasi, data lelang, dan data PIC survey sebagai
 * bagian yang terpisah — dan petugas mencarinya per bagian, bukan per kolom.
 */
function Kelompok({ judul, children }: { judul: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4">
      <h4 className="mb-3 text-sm font-semibold text-slate-800">{judul}</h4>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
        {children}
      </dl>
    </section>
  )
}

const ITEM_COLUMNS: Column<BarangBernomor>[] = [
  {
    key: 'nama_barang',
    title: 'Nama Barang',
    width: '16rem',
    value: (row) => row.nama_barang,
  },
  {
    key: 'jumlah',
    title: 'Jumlah',
    width: '8rem',
    // Jumlah dan satuan dirangkai DI SINI, bukan di dalam SQL. Di layar lama keduanya
    // dirangkai di dalam kueri, dan sejak itu jumlahnya berhenti menjadi angka.
    value: (row) => `${row.jumlah} ${row.satuan}`.trim(),
  },
  {
    key: 'total_nilai',
    title: 'Total Nilai',
    width: '10rem',
    alignRight: true,
    value: (row) => row.total_nilai,
  },
  {
    key: 'status_terjual',
    title: 'Status Terjual',
    width: '10rem',
    value: (row) => row.status_terjual,
  },
  {
    key: 'nama_pemenang',
    title: 'Pemenang',
    width: '12rem',
    value: (row) => row.nama_pemenang,
  },
  {
    key: 'no_akseptasi',
    title: 'No Akseptasi',
    width: '10rem',
    value: (row) => row.no_akseptasi,
  },
  {
    key: 'nilai_akseptasi',
    title: 'Nilai Akseptasi',
    width: '10rem',
    alignRight: true,
    value: (row) => row.nilai_akseptasi,
  },
  {
    key: 'remark',
    title: 'Remark',
    width: '14rem',
    value: (row) => row.remark,
  },
]

function messageOf(error: unknown): { title: string; description: string } {
  if (error instanceof APIError) {
    return { title: 'Detail tidak dapat dibuka', description: error.message }
  }
  if (error instanceof NetworkError) {
    return {
      title: 'Detail tidak dapat dibuka',
      description: 'Sambungan ke server terputus. Periksa jaringan lalu coba lagi.',
    }
  }
  return {
    title: 'Detail tidak dapat dibuka',
    description: 'Terjadi kesalahan pada sistem.',
  }
}

/**
 * Panel "Detail Salvage".
 *
 * Menggantikan layar yang di Pega dibuka `Activity/SetDataDetailSalvage_act-Act.xml` lalu
 * digambar `Section/DataDetail_Salvage-Section.xml` beserta grid
 * `Section/DetailPengajuanSalvage-Section.xml`.
 *
 * # Panel di bawah grid, bukan halaman terpisah
 *
 * Sama seperti panel rincian di modul lain, dan alasannya sama: petugas membuka rincian
 * untuk MEMBANDINGKANNYA dengan baris di daftar — posisi, nilai, PIC — dan halaman terpisah
 * memaksanya mengingat baris yang baru saja ia lihat.
 *
 * # Dua hal yang sengaja digambar apa adanya
 *
 *   - "Posisi Salvage" adalah LABEL, bukan kode. Kodenya tetap dikirim server dan
 *     ditampilkan kecil di sebelahnya, karena itulah yang dicari orang yang menelusuri ke
 *     Pega.
 *   - Penanda "pengajuan sebelum 17 Juli 2023" ditampilkan TANPA tafsiran. Tidak ada satu
 *     pun rule di export yang memakainya selain menggambarnya, sehingga menuliskan artinya
 *     berarti mengarang.
 */
export function DetailSalvagePanel({
  detailKey,
  reference,
  data,
  isPending,
  isError,
  error,
  onClose,
}: Props) {
  const detail = { data, isPending, isError, error }

  return (
    <section
      className="mt-6 rounded-xl border border-slate-300 bg-slate-50 p-4"
      aria-label="Detail Salvage"
    >
      <header className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold text-slate-900">Detail Salvage</h3>
          <p className="mt-0.5 text-sm text-slate-600">
            {detailKey === 'klaim' ? 'Klaim ' : 'Pengajuan '}
            <span className="font-medium text-slate-800">{reference}</span>
            {detail.data && detail.data.ada_pengajuan && detailKey === 'klaim' ? (
              <>
                {' · pengajuan '}
                <span className="font-medium text-slate-800">{detail.data.id_salvage}</span>
              </>
            ) : null}
            {detail.data && detailKey === 'pengajuan' ? (
              <>
                {' · klaim '}
                <span className="font-medium text-slate-800">{detail.data.no_klaim}</span>
              </>
            ) : null}
          </p>
        </div>

        <Button type="button" tone="kedua" onClick={onClose}>
          Tutup
        </Button>
      </header>

      {detail.isError ? (
        <ErrorMessage
          title={messageOf(detail.error).title}
          description={messageOf(detail.error).description}
          tone="gangguan"
        />
      ) : null}

      {detail.isPending && !detail.isError ? (
        <p className="py-6 text-center text-sm text-slate-500">Memuat rincian…</p>
      ) : null}

      {/*
        Klaim TANPA pengajuan tidak pernah sampai ke sini — halaman mengarahkannya ke form
        "Menambahkan Data Salvage". Keputusan itu dipegang SATU tempat; menambahkan
        cabangnya di sini pula berarti dua tempat memutuskan hal yang sama, dan yang satu
        akan tertinggal.
      */}
      {detail.data ? (
        <div className="space-y-4">
          <Kelompok judul="Data Klaim">
            <Isian label="No Klaim" value={detail.data.no_klaim} />
            <Isian label="Nama Bisnis" value={detail.data.nama_bisnis} />
            <Isian label="PIC Teknik" value={detail.data.pic} />
            <Isian label="Tgl Kejadian" value={detail.data.tanggal_kejadian} />
          </Kelompok>

          <Kelompok judul="Data Pengajuan">
            <Isian label="ID Salvage" value={detail.data.id_salvage} />
            <Isian label="Tanggal Input Salvage" value={detail.data.tanggal_input_salvage} />
            <Isian label="Jenis Salvage" value={detail.data.jenis_salvage} />
            <Isian label="Quantity Salvage" value={detail.data.quantity_salvage} />
            <Isian label="Estimasi" value={detail.data.estimasi} />
            <Isian label="Mata Uang" value={detail.data.mata_uang} />
            <Isian label="Nama Object" value={detail.data.nama_object} />
            <Isian label="Nama Coverage" value={detail.data.nama_coverage} />
            <Isian label="Lokasi Salvage" value={detail.data.lokasi_salvage} />
            <Isian
              label="Lokasi di Jabodetabek"
              value={detail.data.lokasi_salvage_di_jabodetabek ? 'Ya' : 'Tidak'}
            />
            <Isian label="Remark" value={detail.data.remark} />
          </Kelompok>

          <Kelompok judul="Posisi dan Akseptasi">
            <div className="min-w-0">
              <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
                Posisi Salvage
              </dt>
              <dd className="mt-0.5 break-words text-sm text-slate-900">
                {detail.data.posisi_salvage.trim() === '' ? (
                  <span className="text-slate-400">—</span>
                ) : (
                  <>
                    {detail.data.posisi_salvage}
                    {detail.data.kode_posisi_salvage.trim() === '' ? null : (
                      <span className="ml-2 text-xs text-slate-500">
                        kode {detail.data.kode_posisi_salvage}
                      </span>
                    )}
                  </>
                )}
              </dd>
            </div>
            <Isian label="Tanggal Transfer GA" value={detail.data.tanggal_transfer_ga} />
            <Isian label="Tanggal Akseptasi" value={detail.data.tanggal_akseptasi} />
            <Isian label="No Akseptasi" value={detail.data.no_akseptasi} />
            <Isian label="Nilai Salvage" value={detail.data.nilai_salvage} />
            <Isian label="Email" value={detail.data.email} />
          </Kelompok>

          <Kelompok judul="Lelang">
            <Isian label="Nilai Penawaran" value={detail.data.nilai_penawaran} />
            <Isian label="Nama Pemenang" value={detail.data.nama_pemenang} />
            <Isian label="Tanggal Lelang" value={detail.data.tanggal_lelang} />
          </Kelompok>

          <Kelompok judul="PIC Survey">
            <Isian label="Nama PIC Survey" value={detail.data.nama_pic_survey} />
            <Isian label="No Telp PIC Survey" value={detail.data.no_telp_pic_survey} />
            <Isian label="Email PIC Survey" value={detail.data.email_pic_survey} />
          </Kelompok>

          <section className="rounded-lg border border-slate-200 bg-white p-4">
            <h4 className="mb-3 text-sm font-semibold text-slate-800">
              Detail Pengajuan Salvage
            </h4>

            <DataTable
              columns={ITEM_COLUMNS}
              rows={detail.data.barang.map((row, index) => ({ ...row, urutan: index }))}
              rowKey={(row) => String(row.urutan)}
              emptyMessage="Pengajuan ini tidak memiliki rincian barang."
              hideSearch
            />
          </section>

          {detail.data.pengajuan_sebelum_juli_2023 ? (
            <p className="text-xs text-slate-500">
              Pengajuan ini dibuat sebelum 17 Juli 2023. Sistem lama menandainya, tetapi tidak
              ada satu pun aturan di dalamnya yang menjelaskan akibat penandaan itu — sehingga
              penanda ini ditampilkan apa adanya, tanpa tafsiran.
            </p>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
