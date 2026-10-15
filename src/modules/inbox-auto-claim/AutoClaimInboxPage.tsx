import { useState } from 'react'

import type { AutoClaimTab, AutoClaimUploadResponse } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import { useAutoClaimTabList, useRefreshAutoClaim } from './api'
import { CompanyBrowser } from './CompanyBrowser'
import { PremiumCheckPanel } from './PremiumCheckPanel'
import { PANEL_ID, SourceTab, tabId } from './SourceTab'
import { UploadForm } from './UploadForm'

/**
 * Layar Inbox Auto Claim.
 *
 * Pengganti `InboxAutoClaim/InboxAutoClaim-Harness.xml`. Susunannya dari atas:
 *
 *   - tiga tab jenis klaim (Asuransi Kredit · ANEKA · Travel), masing-masing satu tabel;
 *   - satu panel tab berisi tombol Upload Data Klaim untuk tab itu, lalu daftar
 *     perusahaan di kiri dan grid batch perusahaan terpilih di kanan (`CompanyBrowser`).
 *
 * # Letak tombol (permintaan Work Owner 2026-09-29)
 *
 *   - Upload Data Klaim di DALAM panel tab, bukan di kepala halaman.
 *   - Proses Klaim dan Generate DLA per BARIS batch di grid kanan. Proses Klaim di layar
 *     lama satu per tab dan memproses semua batch; di sini per batch.
 *   - Catatan "belum tersedia" untuk kedua tombol itu dibuang. Mesinnya memang belum
 *     dibangun, sehingga keduanya nonaktif dengan alasan di `title`.
 *
 * # Tab keempat: Cek Premi
 *
 * Tab Cek Premi layar lama (`InboxAutoClaim-Harness.xml` :50640) tidak membaca tabel batch
 * apa pun, sehingga ia tidak datang dari daftar tab server (yang memetakan tab ke tabel).
 * Ia ditambahkan di sini dan menampilkan `PremiumCheckPanel` — tanpa tombol Upload.
 */

/** Kode tab Cek Premi. Tidak bertabrakan dengan kode tab server (aneka, kredit, travel). */
const PREMIUM_TAB = 'cek-premi'

const premiumTab: AutoClaimTab = { kode: PREMIUM_TAB, label: 'Cek Premi', tabel: '' }
export function AutoClaimInboxPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Tab yang sedang terbuka. Bawaannya datang dari server supaya layar tidak memuat
  // daftar tab-nya sendiri.
  const [source, setSource] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploadNote, setUploadNote] = useState<AutoClaimUploadResponse | null>(null)

  const tabList = useAutoClaimTabList()
  const activeSource = source === '' ? (tabList.data?.bawaan ?? '') : source
  const allTab = tabList.data ? [...tabList.data.tab, premiumTab] : []
  const activeTab = allTab.find((t) => t.kode === activeSource)
  const premiumActive = activeSource === PREMIUM_TAB
  const activeTable = activeTab?.tabel ?? ''
  const activeLabel = activeTab?.label ?? ''

  const reload = useRefreshAutoClaim()

  return (
    <main className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Inbox Auto Claim</h1>
          <p className="text-sm text-slate-600">
            Batch klaim borongan dari perusahaan rekanan beserta hasil pemrosesannya.
          </p>
        </div>
        <Button tone="kedua" onClick={reload.refresh} disabled={reload.isFetching}>
          {reload.isFetching ? 'Memuat…' : 'Refresh'}
        </Button>
      </header>

      {/* Tab duduk di atas segalanya: ia memilih TABEL yang dibaca seluruh layar, bukan
          menyaring isi tabel yang sama. */}
      <div className="mt-4">
        <SourceTab tab={allTab} selected={activeSource} onSelect={setSource} />
      </div>

      {activeSource !== '' && (
        // Satu panel untuk tab aktif. Tombol Upload Data Klaim ada DI DALAM panel ini
        // (permintaan Work Owner 2026-09-29), seperti layar lama yang menaruh satu tombol
        // Upload pada tiap tab: berkas selalu masuk ke tab yang sedang terbuka.
        <div
          role="tabpanel"
          id={PANEL_ID}
          aria-labelledby={tabId(activeSource)}
          className="mt-4 rounded-kartu border border-slate-200 bg-white/60 p-4 sm:p-5"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-xs text-slate-500">
              Portal entitas: <span className="font-medium text-slate-700">{portal ?? '—'}</span>
              {' · '}
              Tab: <span className="font-medium text-slate-700">{activeLabel || '…'}</span>
            </p>
            {!premiumActive && (
              <Button
                tone="utama"
                onClick={() => setUploadOpen(true)}
                disabled={uploadOpen || portal === null}
              >
                Upload Data Klaim
              </Button>
            )}
          </div>

          {!premiumActive && <UploadResult note={uploadNote} onClose={() => setUploadNote(null)} />}

          {portal === null ? (
            <div className="mt-5">
              <ErrorMessage
                title="Portal entitas belum dipilih"
                description="Data klaim dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
                tone="penolakan"
              />
            </div>
          ) : premiumActive ? (
            <PremiumCheckPanel />
          ) : (
            // `key` pada tab membuat berpindah tab MEMASANG ULANG isinya: kata kunci,
            // perusahaan terpilih, dan halaman ikut hilang. Bukan kerapian — ketiga tab
            // membaca tabel berbeda, dan kode perusahaan di satu tab belum tentu ada di tab
            // lain.
            <CompanyBrowser key={activeSource} source={activeSource} table={activeTable} />
          )}
        </div>
      )}

      {uploadOpen && (
        <UploadForm
          source={activeSource}
          sourceLabel={activeLabel}
          onClose={() => setUploadOpen(false)}
          onUploaded={(result) => {
            setUploadNote(result)
            setUploadOpen(false)
          }}
        />
      )}
    </main>
  )
}

/**
 * Panel hasil unggahan, di dalam panel tab.
 *
 * Ia membedakan TIGA keadaan, dan pembedaannya bukan kerapian tampilan: yang "bertanda"
 * tersimpan dan terlihat di grid, yang "ditolak" tidak ada di mana pun. Pengguna yang
 * mengira keduanya sama akan mencari baris yang tidak pernah tersimpan.
 */
function UploadResult({
  note,
  onClose,
}: {
  note: AutoClaimUploadResponse | null
  onClose: () => void
}) {
  if (note === null) return null

  return (
    <div
      role="status"
      className={
        note.ditolak.length > 0
          ? 'mt-4 rounded-kartu border border-amber-200 bg-amber-50/80 p-4 text-sm shadow-lembut'
          : 'mt-4 rounded-kartu border border-emerald-200 bg-emerald-50/80 p-4 text-sm shadow-lembut'
      }
    >
      <p className="font-medium text-slate-900">
        {note.jumlah_baris === 0
          ? 'Tidak ada baris yang tersimpan'
          : `${note.jumlah_baris} baris tersimpan`}
      </p>

      {note.batch.length > 0 && (
        <ul className="mt-1 space-y-0.5 text-slate-800">
          {note.batch.map((b) => (
            <li key={`${b.kode_perusahaan}-${b.batch}`}>
              {b.nama_perusahaan === '' ? b.kode_perusahaan : b.nama_perusahaan} — batch{' '}
              <span className="font-medium">{b.batch}</span>, {b.jumlah_baris} baris
              {b.jumlah_bertanda > 0 && (
                <span className="text-amber-800">
                  {' '}
                  ({b.jumlah_lolos} menunggu proses, {b.jumlah_bertanda} bertanda gagal)
                </span>
              )}
            </li>
          ))}
        </ul>
      )}

      {note.ditolak.length > 0 && (
        <div className="mt-3 border-t border-amber-200 pt-3">
          <p className="font-medium text-amber-900">{note.ditolak.length} baris TIDAK tersimpan</p>
          <p className="mt-0.5 text-xs text-amber-800">
            Perusahaan rekanannya tidak dapat ditemukan dari nomor polis, sehingga barisnya tidak
            punya tempat di grid mana pun. Perbaiki nomor polisnya lalu unggah ulang baris ini saja.
          </p>
          <ul className="mt-2 space-y-0.5 text-xs text-amber-900">
            {note.ditolak.map((row) => (
              <li key={`${row.baris}-${row.nomor_polis}`}>
                Baris <span className="font-medium tabular-nums">{row.baris}</span> —{' '}
                {row.nomor_polis === '' ? '(nomor polis kosong)' : row.nomor_polis} — {row.pesan}
              </li>
            ))}
          </ul>
        </div>
      )}

      {note.batch.length > 0 && (
        <p className="mt-2 text-xs text-slate-700">
          Baris yang menunggu proses belum menjadi klaim sampai batch-nya diproses.
        </p>
      )}

      <div className="mt-3">
        <Button tone="halus" onClick={onClose}>
          Tutup
        </Button>
      </div>
    </div>
  )
}
