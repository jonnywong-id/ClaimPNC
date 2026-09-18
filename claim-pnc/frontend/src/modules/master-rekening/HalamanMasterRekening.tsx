import { useState } from 'react'

import { GalatAPI } from '@/api/klien'
import { StatusRekening, type Rekening } from '@/api/tipe'

import { FormRekening } from './FormRekening'
import { TabelRekening } from './TabelRekening'
import { gunakanDaftarRekening, gunakanPutuskanRekening, type SaringanRekening } from './api'

/**
 * Lima tab, sama persis dengan layar lama.
 *
 * Section Pega-nya lima berkas terpisah — BrowseMasterCariDataRekening,
 * ApprovalMasterRekening, BrowseMasterRekeningApproval, BrowseMasterRekeningApprove,
 * BrowseMasterRekeningReject — yang isinya nyaris sama dan karena itu berbeda-beda di
 * tempat yang tidak disengaja. Di sini kelimanya satu layar dengan saringan berbeda.
 */
const TAB = [
  { id: 'cari', label: 'Cari Data Rekening' },
  { id: 'komite', label: 'Komite Approval' },
  { id: 'menunggu', label: 'Waiting Approval' },
  { id: 'disetujui', label: 'Approve' },
  { id: 'ditolak', label: 'Reject' },
] as const

type IdTab = (typeof TAB)[number]['id']

function saringanUntuk(tab: IdTab, pencarian: Pencarian): SaringanRekening {
  const dasar: SaringanRekening = {
    nomorRekening: pencarian.nomorRekening,
    namaPemilik: pencarian.namaPemilik,
    namaBank: pencarian.namaBank,
  }
  switch (tab) {
    case 'komite':
      return { ...dasar, status: StatusRekening.menunggu, komiteSaya: true }
    case 'menunggu':
      return { ...dasar, status: StatusRekening.menunggu }
    case 'disetujui':
      return { ...dasar, status: StatusRekening.disetujui }
    case 'ditolak':
      return { ...dasar, status: StatusRekening.ditolak }
    default:
      return dasar
  }
}

type Pencarian = {
  nomorRekening: string
  namaPemilik: string
  namaBank: string
}

const PENCARIAN_KOSONG: Pencarian = { nomorRekening: '', namaPemilik: '', namaBank: '' }

/** HalamanMasterRekening adalah layar pengelolaan master rekening. */
export function HalamanMasterRekening() {
  const [tab, setTab] = useState<IdTab>('cari')
  const [pencarian, setPencarian] = useState<Pencarian>(PENCARIAN_KOSONG)
  const [formTerbuka, setFormTerbuka] = useState(false)

  const saringan = saringanUntuk(tab, pencarian)
  const daftar = gunakanDaftarRekening(saringan)

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Master Rekening</h1>
        <p className="mt-1 text-sm text-slate-600">
          Rekening tujuan pembayaran klaim. Rekening baru menunggu keputusan komite
          sebelum dapat dipakai.
        </p>
      </header>

      <nav aria-label="Tab master rekening" className="mt-4 flex flex-wrap gap-1 border-b border-slate-200">
        {TAB.map((t) => (
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
                <FormRekening padaBerhasil={() => setFormTerbuka(false)} />
              </div>
            </div>
          )}
        </section>
      )}

      <section className="mt-4" aria-label="Pencarian">
        <div className="grid gap-3 sm:grid-cols-3">
          <IsianCari
            id="cariNomor"
            label="No rekening"
            nilai={pencarian.nomorRekening}
            ubah={(v) => setPencarian((p) => ({ ...p, nomorRekening: v }))}
          />
          <IsianCari
            id="cariPemilik"
            label="Nama pemilik"
            nilai={pencarian.namaPemilik}
            ubah={(v) => setPencarian((p) => ({ ...p, namaPemilik: v }))}
          />
          <IsianCari
            id="cariBank"
            label="Nama bank"
            nilai={pencarian.namaBank}
            ubah={(v) => setPencarian((p) => ({ ...p, namaBank: v }))}
          />
        </div>
      </section>

      <section className="mt-6">
        {daftar.isError && (
          <p role="alert" className="rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
            Daftar rekening tidak dapat dimuat. Coba beberapa saat lagi.
          </p>
        )}

        <TabelRekening
          baris={daftar.data?.rekening ?? []}
          memuat={daftar.isPending}
          aksi={
            tab === 'komite' || tab === 'menunggu'
              ? (rekening) => <TindakanKomite rekening={rekening} />
              : undefined
          }
        />

        {daftar.data && daftar.data.jumlah > daftar.data.rekening.length && (
          <p className="mt-3 text-sm text-slate-500">
            Menampilkan {daftar.data.rekening.length} dari {daftar.data.jumlah} rekening.
            Persempit pencarian untuk melihat sisanya.
          </p>
        )}
      </section>
    </div>
  )
}

function IsianCari({
  id,
  label,
  nilai,
  ubah,
}: {
  id: string
  label: string
  nilai: string
  ubah: (nilai: string) => void
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        id={id}
        value={nilai}
        onChange={(e) => ubah(e.target.value)}
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
function TindakanKomite({ rekening }: { rekening: Rekening }) {
  const putuskan = gunakanPutuskanRekening()
  const [catatan, setCatatan] = useState(rekening.catatan)

  const kirim = (status: typeof StatusRekening.disetujui | typeof StatusRekening.ditolak) => {
    putuskan.mutate({
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
        onChange={(e) => setCatatan(e.target.value)}
        placeholder="Keterangan approval atasan"
        className="w-full rounded border border-slate-300 px-2 py-1 text-sm focus:border-slate-500 focus:outline-none"
      />
      <div className="flex gap-2">
        <button
          type="button"
          disabled={putuskan.isPending}
          onClick={() => kirim(StatusRekening.disetujui)}
          className="rounded bg-green-700 px-3 py-1 text-xs font-medium text-white hover:bg-green-800 disabled:opacity-60"
        >
          Approve
        </button>
        <button
          type="button"
          disabled={putuskan.isPending}
          onClick={() => kirim(StatusRekening.ditolak)}
          className="rounded bg-red-700 px-3 py-1 text-xs font-medium text-white hover:bg-red-800 disabled:opacity-60"
        >
          Reject
        </button>
      </div>
      {putuskan.isError && <PesanKeputusan galat={putuskan.error} />}
    </div>
  )
}

function PesanKeputusan({ galat }: { galat: unknown }) {
  if (galat instanceof GalatAPI && galat.kode === 'isian_tidak_sah') {
    return (
      <p role="alert" className="text-xs text-red-700">
        {galat.detail.map((p) => p.pesan).join(' ')}
      </p>
    )
  }
  if (galat instanceof GalatAPI && galat.kode === 'keputusan_sudah_diambil') {
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
