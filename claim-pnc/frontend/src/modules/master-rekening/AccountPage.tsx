import { useState } from 'react'

import { APIError } from '@/api/client'
import { AccountStatus, type Account } from '@/api/types'

import { AccountForm } from './AccountForm'
import { AccountTable } from './AccountTable'
import { useAccountList, useDecideAccount, type AccountFilter } from './api'

/**
 * Lima tab, sama persis dengan layar lama.
 *
 * Section Pega-nya lima berkas terpisah — BrowseMasterCariDataRekening,
 * ApprovalMasterRekening, BrowseMasterRekeningApproval, BrowseMasterRekeningApprove,
 * BrowseMasterRekeningReject — yang isinya nyaris sama dan karena itu berbeda-beda di
 * tempat yang tidak disengaja. Di sini kelimanya satu layar dengan saringan berbeda.
 */
const TABS = [
  { id: 'cari', label: 'Cari Data Rekening' },
  { id: 'komite', label: 'Komite Approval' },
  { id: 'menunggu', label: 'Waiting Approval' },
  { id: 'disetujui', label: 'Approve' },
  { id: 'ditolak', label: 'Reject' },
] as const

type TabId = (typeof TABS)[number]['id']

function filterFor(tab: TabId, search: SearchBox): AccountFilter {
  const base: AccountFilter = {
    nomorRekening: search.nomorRekening,
    namaPemilik: search.namaPemilik,
    namaBank: search.namaBank,
  }
  switch (tab) {
    case 'komite':
      return { ...base, status: AccountStatus.menunggu, komiteSaya: true }
    case 'menunggu':
      return { ...base, status: AccountStatus.menunggu }
    case 'disetujui':
      return { ...base, status: AccountStatus.disetujui }
    case 'ditolak':
      return { ...base, status: AccountStatus.ditolak }
    default:
      return base
  }
}

type SearchBox = {
  nomorRekening: string
  namaPemilik: string
  namaBank: string
}

const EMPTY_SEARCH: SearchBox = { nomorRekening: '', namaPemilik: '', namaBank: '' }

/** HalamanMasterRekening adalah layar pengelolaan master rekening. */
export function AccountPage() {
  const [tab, setTab] = useState<TabId>('cari')
  const [search, setSearch] = useState<SearchBox>(EMPTY_SEARCH)
  const [formTerbuka, setFormTerbuka] = useState(false)

  const filter = filterFor(tab, search)
  const list = useAccountList(filter)

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Master Rekening</h1>
        <p className="mt-1 text-sm text-slate-600">
          Rekening tujuan pembayaran klaim. Rekening baru menunggu decision komite
          before dapat dipakai.
        </p>
      </header>

      <nav aria-label="Tab master rekening" className="mt-4 flex flex-wrap gap-1 border-b border-slate-200">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            aria-current={tab === t.id ? 'page' : undefined}
            onClick={() => setTab(t.id)}
            className={
              'rounded-t px-3 py-2 text-sm font-medium ' +
              (tab === t.id
                ? 'border-b-2 border-slate-900 text-slate-900'
                : 'text-slate-500 hover:text-slate-800')
            }
          >
            {t.label}
          </button>
        ))}
      </nav>

      {tab === 'cari' && (
        <section className="mt-4">
          <button
            type="button"
            onClick={() => setFormTerbuka((terbuka) => !terbuka)}
            className="rounded border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
          >
            {formTerbuka ? 'Tutup formulir' : 'Tambah rekening'}
          </button>

          {formTerbuka && (
            <div className="mt-4 rounded border border-slate-200 p-4">
              <h2 className="text-sm font-semibold text-slate-900">Rekening baru</h2>
              <div className="mt-3">
                <AccountForm onSuccess={() => setFormTerbuka(false)} />
              </div>
            </div>
          )}
        </section>
      )}

      <section className="mt-4" aria-label="Pencarian">
        <div className="grid gap-3 sm:grid-cols-3">
          <SearchFields
            id="cariNomor"
            label="No rekening"
            nilai={search.nomorRekening}
            edit={(v) => setSearch((p) => ({ ...p, nomorRekening: v }))}
          />
          <SearchFields
            id="cariPemilik"
            label="Nama pemilik"
            nilai={search.namaPemilik}
            edit={(v) => setSearch((p) => ({ ...p, namaPemilik: v }))}
          />
          <SearchFields
            id="cariBank"
            label="Nama bank"
            nilai={search.namaBank}
            edit={(v) => setSearch((p) => ({ ...p, namaBank: v }))}
          />
        </div>
      </section>

      <section className="mt-6">
        {list.isError && (
          <p role="alert" className="rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
            Daftar rekening tidak dapat dimuat. Coba beberapa saat lagi.
          </p>
        )}

        <AccountTable
          rows={list.data?.rekening ?? []}
          loading={list.isPending}
          aksi={
            tab === 'komite' || tab === 'menunggu'
              ? (rekening) => <CommitteeAction rekening={rekening} />
              : undefined
          }
        />

        {list.data && list.data.jumlah > list.data.rekening.length && (
          <p className="mt-3 text-sm text-slate-500">
            Menampilkan {list.data.rekening.length} dari {list.data.jumlah} rekening.
            Persempit search untuk melihat rest.
          </p>
        )}
      </section>
    </div>
  )
}

function SearchFields({
  id,
  label,
  nilai,
  edit,
}: {
  id: string
  label: string
  nilai: string
  edit: (nilai: string) => void
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        id={id}
        value={nilai}
        onChange={(e) => edit(e.target.value)}
        className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none"
      />
    </div>
  )
}

/**
 * TindakanKomite menampilkan tombol setujui dan tolak.
 *
 * Keterangan approval wajib diisi untuk KEDUANYA di layar ini, walaupun server hanya
 * mewajibkannya saat menyetujui. Alasannya: komite yang menolak tanpa alasan membuat
 * pengaju mengulang pengajuan yang sama persis, karena tidak ada yang memberitahunya
 * apa yang salah.
 */
function CommitteeAction({ rekening }: { rekening: Account }) {
  const decide = useDecideAccount()
  const [catatan, setNote] = useState(rekening.catatan)

  const send = (status: typeof AccountStatus.disetujui | typeof AccountStatus.ditolak) => {
    decide.mutate({
      kodeBank: rekening.kode_bank,
      nomorRekening: rekening.nomor_rekening,
      status,
      catatan,
    })
  }

  return (
    <div className="flex min-w-[18rem] flex-col gap-2">
      <label className="sr-only" htmlFor={`catatan-${rekening.kode_bank}-${rekening.nomor_rekening}`}>
        Keterangan approval atasan
      </label>
      <input
        id={`catatan-${rekening.kode_bank}-${rekening.nomor_rekening}`}
        value={catatan}
        onChange={(e) => setNote(e.target.value)}
        placeholder="Keterangan approval atasan"
        className="w-full rounded border border-slate-300 px-2 py-1 text-sm focus:border-slate-500 focus:outline-none"
      />
      <div className="flex gap-2">
        <button
          type="button"
          disabled={decide.isPending}
          onClick={() => send(AccountStatus.disetujui)}
          className="rounded bg-green-700 px-3 py-1 text-xs font-medium text-white hover:bg-green-800 disabled:opacity-60"
        >
          Approve
        </button>
        <button
          type="button"
          disabled={decide.isPending}
          onClick={() => send(AccountStatus.ditolak)}
          className="rounded bg-red-700 px-3 py-1 text-xs font-medium text-white hover:bg-red-800 disabled:opacity-60"
        >
          Reject
        </button>
      </div>
      {decide.isError && <DecisionMessage error={decide.error} />}
    </div>
  )
}

function DecisionMessage({ error }: { error: unknown }) {
  if (error instanceof APIError && error.kode === 'isian_tidak_sah') {
    return (
      <p role="alert" className="text-xs text-red-700">
        {error.detail.map((v) => v.pesan).join(' ')}
      </p>
    )
  }
  if (error instanceof APIError && error.kode === 'keputusan_sudah_diambil') {
    return (
      <p role="alert" className="text-xs text-red-700">
        Rekening ini sudah diputuskan komite lain. Muat ulang daftar.
      </p>
    )
  }
  return (
    <p role="alert" className="text-xs text-red-700">
      Keputusan tidak tersimpan. Coba lagi.
    </p>
  )
}
