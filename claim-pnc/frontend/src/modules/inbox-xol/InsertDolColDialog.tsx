import { useEffect, useRef, useState } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { DateField, isoToText } from '@/components/DateField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { SelectField } from '@/components/SelectField'

import { useCauseOfLoss, useInsertDolCol, useMasters } from './api'
import { isValidationError, messageOf, violationsOf } from './errors'
import { EMPTY_INSERT_FORM, type InsertDolColForm, type MasterXOL } from './types'

/**
 * Modal di balik tombol "INSERT DOL DAN COL".
 *
 * # Isinya persis modal lama
 *
 * Urutannya diambil dari `Section/InboxClaimXOL-Section.xml`, bukan dikarang:
 *
 *	:4621	tombol INSERT DOL DAN COL	pembukanya
 *	:5064	Date Of Loss			isian
 *	:5385	Cause Of Loss			dropdown (`pyNoSelectionText` = "Pilih Cause Of Loss")
 *	:7577	grid PILIH MASTER XOL		Tahun · Kurs · Group Business
 *	:9847	tombol Pilih			satu per baris grid
 *	:10840	tombol Simpan			menulis
 *
 * # "Pilih" menentukan yang DITULISI, bukan yang ditampilkan
 *
 * Ia menetapkan perjanjian mana yang akan menerima baris DOL/COL baru — tahun dan kursnya
 * yang dipakai. Grid di belakang modal TIDAK berubah karenanya: grid itu menggabungkan
 * seluruh perjanjian, dan di layar lama pun tidak ada penyaring perjanjian sama sekali.
 *
 * Versi sebelumnya memperlakukan "Pilih" sebagai penyaring grid. Itu keliru, dan keliru
 * yang mahal: grid jadi menampilkan satu perjanjian saja — pada data nyata, perjanjian
 * yang kebetulan belum punya klaim.
 *
 * # "Simpan" menulis SATU BARIS PER GROUP BUSINESS
 *
 * Bukan satu baris. Activity lama memecah daftar group business perjanjian yang dipilih,
 * lalu mengulang sisipannya untuk setiap anggotanya. Jumlah yang benar-benar tertulis
 * karena itu ditampilkan setelah berhasil — perjanjian berisi lima group business
 * menghasilkan lima baris, dan pengguna berhak tahu itu.
 *
 * # Kenapa isian tanggalnya DateField, bukan isian teks biasa
 *
 * Versi sebelumnya memakai Field berplaceholder "dd/mm/yyyy": tanggalnya HARUS diketik,
 * karena tidak ada kalender yang dapat dibuka. DateField membawa tombol kalender di sisi
 * kanan isian, dan tetap menerima ketikan bagi yang lebih cepat mengetik.
 *
 * Nilainya ISO (`YYYY-MM-DD`) selama berada di layar, lalu diubah menjadi `DD/MM/YYYY`
 * tepat saat dikirim — bentuk kolom `DOL`, dan bentuk yang sama dengan `tanggal_kejadian`
 * di seluruh kontrak modul ini. Konversinya dilakukan DI SINI, terlihat, bukan disembunyikan
 * di dalam hook: yang membacanya nanti perlu melihat bahwa kedua bentuk itu memang berbeda.
 */
export function InsertDolColDialog({ onClose }: { onClose: () => void }) {
  const [form, setForm] = useState<InsertDolColForm>(EMPTY_INSERT_FORM)

  // Tanggal disimpan ISO selama di layar — bentuk yang dipakai DateField. Ia TERPISAH
  // dari form.tanggal_kejadian, yang berbentuk DD/MM/YYYY karena itulah bentuk kontrak.
  // Menyatukan keduanya berarti mengubah bentuk di setiap ketikan, dan tanggal separuh
  // jadi akan terbaca sebagai tanggal yang salah.
  const [lossDateISO, setLossDateISO] = useState('')

  const masters = useMasters()
  const causes = useCauseOfLoss()
  const simpan = useInsertDolCol()

  const closeRef = useRef<HTMLButtonElement>(null)

  // Fokus dipindahkan ke dalam modal saat ia terbuka. Tanpa ini, pengguna papan ketik
  // tetap berada di tombol pembukanya — di BELAKANG lapisan gelap — dan menekan Tab
  // menelusuri layar yang sedang tertutup.
  useEffect(() => {
    closeRef.current?.focus()
  }, [])

  // Pelanggaran per isian, supaya pesannya menempel di kolom yang salah — bukan satu
  // kalimat di atas formulir yang memaksa pengguna menebak kolom mana yang dimaksud.
  const violations = violationsOf(simpan.error)

  const causeOptions = (causes.data?.sebab_kerugian ?? []).map((cause) => ({
    // Nilainya DESKRIPSI, bukan ID: itulah yang tersimpan di kolom CAUSEOFLOSS.
    value: cause.deskripsi,
    label: cause.deskripsi,
  }))

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Insert DOL dan COL"
      onKeyDown={(event) => {
        if (event.key === 'Escape') onClose()
      }}
    >
      <div className="my-8 w-full max-w-3xl rounded-kartu bg-white p-6 shadow-terbang">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">INSERT DOL DAN COL</h2>
            <p className="mt-1 text-sm text-slate-600">
              Pilih perjanjian XOL untuk mengisi grid di belakang, atau tambahkan Date Of
              Loss dan Cause Of Loss baru.
            </p>
          </div>

          {/*
            Tombol polos, bukan komponen Button: ia perlu `ref` untuk menerima fokus saat
            modal dibuka, dan Button tidak meneruskan ref. Gayanya disamakan dengan
            `tone="halus"`.
          */}
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            aria-label="Tutup"
            className="rounded-kontrol px-2 py-1 text-sm font-medium text-slate-600 transition hover:bg-slate-100 hover:text-slate-900 focus:outline-none focus-visible:ring-4 focus-visible:ring-slate-400/35"
          >
            Tutup
          </button>
        </div>

        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            simpan.mutate(form, {
              /*
                Isian DIKOSONGKAN setelah berhasil.

                Sisipannya tidak memeriksa duplikat — itu perilaku sistem lama, dibawa apa
                adanya (`P-5`). Membiarkan isian tetap terisi berarti satu klik tak sengaja
                menulis baris yang sama untuk kedua kalinya, dan gridnya akan menjumlahkan
                keduanya tanpa tanda apa pun bahwa itu salah.

                Modalnya TIDAK ditutup: pengguna yang hendak mendaftarkan beberapa
                kombinasi berturut-turut tidak perlu membukanya lagi, dan pesan jumlah
                baris tetap terbaca.
              */
              onSuccess: () => {
                setForm(EMPTY_INSERT_FORM)
                setLossDateISO('')
              },
            })
          }}
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <DateField
              id="insert-dol"
              label="Date Of Loss"
              value={lossDateISO}
              error={violations['tanggal_kejadian']}
              onChange={(iso) => {
                setLossDateISO(iso)
                // isoToText mengembalikan '' untuk ISO yang belum lengkap, sehingga
                // isian setengah jadi tidak pernah terkirim sebagai tanggal.
                setForm({ ...form, tanggal_kejadian: isoToText(iso) })
              }}
            />

            <SelectField
              id="insert-col"
              label="Cause Of Loss"
              options={causeOptions}
              emptyText={causes.isPending ? '— memuat —' : 'Pilih Cause Of Loss'}
              value={form.sebab_kerugian}
              error={violations['sebab_kerugian']}
              onChange={(event) =>
                setForm({ ...form, sebab_kerugian: event.target.value })
              }
            />
          </div>

          {causes.isError && (
            <ErrorMessage
              title="Daftar Cause Of Loss tidak dapat dimuat"
              description={messageOf(causes.error)}
              tone="gangguan"
            />
          )}

          <MasterPicker
            rows={masters.data?.perjanjian ?? []}
            loading={masters.isPending}
            error={masters.isError ? messageOf(masters.error) : null}
            selected={form.id_master}
            // Modal TIDAK ditutup di sini. Memilih perjanjian baru separuh pekerjaan —
            // Date Of Loss dan Cause Of Loss masih harus diisi sebelum Simpan.
            onPick={(masterID) => setForm({ ...form, id_master: masterID })}
            violation={violations['id_master']}
          />

          <div className="flex flex-wrap items-center gap-2 border-t border-slate-200 pt-4">
            <Button type="submit" tone="utama" disabled={simpan.isPending}>
              {simpan.isPending ? 'Menyimpan…' : 'Simpan'}
            </Button>

            <Button tone="halus" onClick={onClose}>
              Batal
            </Button>
          </div>

          {/*
            Galat VALIDASI tidak digambar dua kali. Ia sudah menempel di isiannya
            masing-masing lewat prop error, dan mengulanginya sebagai kotak merah di
            bawah formulir membuat satu kesalahan terbaca seperti dua.

            Galat lain — penolakan kewenangan, perjanjian hilang, basis data mati —
            digambar APA ADANYA. Pesannya datang dari server dan sudah menyebutkan
            sebabnya; menggantinya dengan kalimat layar akan membuat dua sumber
            kebenaran untuk satu keadaan yang sama.
          */}
          {simpan.isError && !isValidationError(simpan.error) && (
            <ErrorMessage
              title="Belum dapat disimpan"
              description={messageOf(simpan.error)}
              tone="gangguan"
            />
          )}

          {/*
            Hasilnya menyebut JUMLAH BARIS, bukan sekadar "tersimpan". Satu simpan
            menuliskan satu baris per Group Business perjanjian, dan angka itulah
            satu-satunya cara pengguna mengetahui berapa banyak yang ditulis atas namanya.
          */}
          {simpan.isSuccess && (
            <p
              role="status"
              className="rounded-kontrol border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800"
            >
              Tersimpan — {simpan.data?.jumlah_baris ?? 0} baris ditulis, satu per Group
              Business perjanjian. Grid di belakang sudah diperbarui.
            </p>
          )}
        </form>
      </div>
    </div>
  )
}

/**
 * MasterPicker menggambar grid "PILIH MASTER XOL" beserta tombol "Pilih" per baris.
 *
 * Ia grid, bukan dropdown — dan perbedaannya bukan selera: ketiga kolomnya dibaca
 * berdampingan saat memilih. Kurs khususnya menentukan SELURUH angka di grid belakang,
 * dan nilai "USD" yang tidak dapat ditelusuri kursnya tidak dapat diperiksa siapa pun.
 */
function MasterPicker({
  rows,
  loading,
  error,
  selected,
  onPick,
  violation,
}: {
  rows: MasterXOL[]
  loading: boolean
  error: string | null
  selected: string
  onPick: (masterID: string) => void
  /** Pesan validasi "perjanjian wajib dipilih", bila server mengirimkannya. */
  violation?: string | undefined
}) {
  const columns: Column<MasterXOL>[] = [
    {
      key: 'tahun',
      title: 'Tahun',
      value: (row) => row.tahun,
      width: '7rem',
    },
    {
      key: 'kurs',
      title: 'Kurs',
      value: (row) => String(row.kurs),
      render: (row) => new Intl.NumberFormat('id-ID').format(row.kurs),
      alignRight: true,
      width: '8rem',
    },
    {
      key: 'group_business',
      title: 'Group Business',
      value: (row) => row.group_business,
      render: (row) => row.group_business || <span className="text-slate-400">—</span>,
    },
    {
      key: 'aksi',
      title: '',
      noSort: true,
      alignRight: true,
      width: '6rem',
      value: () => '',
      render: (row) => (
        <Button
          tone="kedua"
          onClick={() => onPick(row.id)}
          aria-label={`Pilih perjanjian ${row.tahun}`}
          className={row.id === selected ? 'ring-2 ring-blue-500/40' : undefined}
        >
          Pilih
        </Button>
      ),
    },
  ]

  return (
    <div>
      <DataTable<MasterXOL>
        columns={columns}
        rows={rows}
        rowKey={(row) => row.id}
        title="PILIH MASTER XOL"
        isLoading={loading}
        error={
          error ? (
            <ErrorMessage
              title="Daftar perjanjian XOL tidak dapat dimuat"
              description={error}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage="Belum ada perjanjian XOL pada entitas ini."
      />

      {/*
        Pesannya di BAWAH grid, bukan di atas: yang harus diperbaiki pengguna adalah
        menekan "Pilih" pada salah satu baris, dan pesan di atas grid panjang akan
        tergulung keluar layar justru saat barisnya dicari.
      */}
      {violation && <p className="mt-1.5 text-sm text-red-700">{violation}</p>}
    </div>
  )
}
