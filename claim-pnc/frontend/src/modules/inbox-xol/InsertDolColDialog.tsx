import { useEffect, useRef, useState } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useCauseOfLoss, useInsertDolCol, useMasters } from './api'
import { messageOf } from './errors'
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
 * "Simpan" MENULIS ke `POOLDATA.XOL_TABLE_ALL_KLAIM`, dan tabel itu masih dimiliki Pega
 * selama masa paralel (`P-1`). Ia tetap digambar dan tetap dapat ditekan; yang menolak
 * adalah server, dengan menyebutkan sebabnya.
 */
export function InsertDolColDialog({ onClose }: { onClose: () => void }) {
  const [form, setForm] = useState<InsertDolColForm>(EMPTY_INSERT_FORM)

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
            simpan.mutate(form)
          }}
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              id="insert-dol"
              label="Date Of Loss"
              placeholder="dd/mm/yyyy"
              value={form.tanggal_kejadian}
              onChange={(event) =>
                setForm({ ...form, tanggal_kejadian: event.target.value })
              }
            />

            <SelectField
              id="insert-col"
              label="Cause Of Loss"
              options={causeOptions}
              emptyText={causes.isPending ? '— memuat —' : 'Pilih Cause Of Loss'}
              value={form.sebab_kerugian}
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
            Penolakan server digambar APA ADANYA. Pesannya sudah menyebutkan sebabnya —
            kewenangan menulis tabel XOL masih di Pega — dan menggantinya dengan kalimat
            layar akan membuat dua sumber kebenaran untuk satu keadaan yang sama.
          */}
          {simpan.isError && (
            <ErrorMessage
              title="Belum dapat disimpan"
              description={messageOf(simpan.error)}
              tone="gangguan"
            />
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
}: {
  rows: MasterXOL[]
  loading: boolean
  error: string | null
  selected: string
  onPick: (masterID: string) => void
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
  )
}
