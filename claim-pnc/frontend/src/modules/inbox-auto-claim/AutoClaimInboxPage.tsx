import { useState } from 'react'

import type { AutoClaimUploadResponse } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import { useAutoClaimTabList, useRefreshAutoClaim } from './api'
import { CompanyTable } from './CompanyTable'
import { SourceTab } from './SourceTab'
import { UploadForm } from './UploadForm'

/**
 * Layar Inbox Auto Claim.
 *
 * Pengganti `InboxAutoClaim/InboxAutoClaim-Harness.xml`. Susunannya dari atas:
 *
 *   - tiga tab jenis klaim (Asuransi Kredit · ANEKA · Travel), masing-masing satu tabel;
 *   - tabel perusahaan selebar layar dengan pencarian dan paginasi;
 *   - baris perusahaan yang diklik MELAR menampilkan grid batch 9 kolom layar lama
 *     (`Section/Inbox_AS_KREDIT_Sect-Section.xml`), dan tombol Detail pada grid itu
 *     membuka rincian baris di tempat yang sama.
 *
 * Bentuk tabel-dengan-baris-melar itu keputusan Work Owner 2026-09-27, menggantikan donut
 * dan grid panjang di bawahnya. Kolom grid batch dan tombolnya tidak berubah (`D-13`).
 *
 * # TIGA DARI TUJUH TOMBOL BELUM DAPAT DIKERJAKAN, DAN ITU DINYATAKAN DI LAYAR
 *
 * Yang sudah jalan: Upload Data Klaim, DETAIL, EXPORT BERHASIL, EXPORT GAGAL.
 * Yang belum: Proses Klaim, Generate DLA, Cek Premi — ketiganya memanggil mesin yang
 * modulnya belum dibangun.
 *
 * Tombol yang mesinnya belum dibangun TETAP DITAMPILKAN dalam keadaan nonaktif beserta
 * alasannya, mengikuti perlakuan yang sama pada butir menu yang belum punya layar
 * (keputusan Work Owner 2026-09-18). Dengan begitu kemajuan migrasi terbaca langsung dari
 * layar, dan petugas tidak melaporkan tombol yang "hilang".
 *
 * Menyembunyikannya akan membuat layar ini tampak SELESAI padahal separuh alurnya belum
 * ada — dan itu kesalahpahaman yang paling mahal di antara semua pilihan.
 */
export function AutoClaimInboxPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Tab yang sedang terbuka. Bawaannya datang dari server supaya layar tidak memuat
  // daftar tab-nya sendiri.
  const [source, setSource] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploadNote, setUploadNote] = useState<AutoClaimUploadResponse | null>(null)

  const tabList = useAutoClaimTabList()
  const activeSource = source === '' ? (tabList.data?.bawaan ?? '') : source
  const activeTab = tabList.data?.tab.find((t) => t.kode === activeSource)
  const activeTable = activeTab?.tabel ?? ''
  const activeLabel = activeTab?.label ?? ''

  const reload = useRefreshAutoClaim()

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Inbox Auto Claim</h1>
          <p className="text-sm text-slate-600">
            Batch klaim borongan dari perusahaan rekanan beserta hasil pemrosesannya.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={reload.refresh} disabled={reload.isFetching}>
            {reload.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={() => setUploadOpen(true)} disabled={uploadOpen}>
            Upload Data Klaim
          </Button>
        </div>
      </header>

      <p className="mt-3 text-xs text-slate-500">
        Portal entitas: <span className="font-medium text-slate-700">{portal ?? '—'}</span>
      </p>

      {/* Tab duduk di atas segalanya: ia memilih TABEL yang dibaca seluruh layar, bukan
          menyaring isi tabel yang sama. */}
      <div className="mt-4">
        <SourceTab tab={tabList.data?.tab ?? []} selected={activeSource} onSelect={setSource} />
      </div>

      <PendingActions />

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

      {uploadNote && (
        // Panel hasil unggahan membedakan TIGA keadaan, dan pembedaannya bukan kerapian
        // tampilan: yang "bertanda" tersimpan dan terlihat di grid, yang "ditolak" tidak
        // ada di mana pun. Pengguna yang mengira keduanya sama akan mencari baris yang
        // tidak pernah tersimpan.
        <div
          role="status"
          className={
            uploadNote.ditolak.length > 0
              ? 'mt-5 rounded-kartu border border-amber-200 bg-amber-50/80 p-4 text-sm shadow-lembut'
              : 'mt-5 rounded-kartu border border-emerald-200 bg-emerald-50/80 p-4 text-sm shadow-lembut'
          }
        >
          <p className="font-medium text-slate-900">
            {uploadNote.jumlah_baris === 0
              ? 'Tidak ada baris yang tersimpan'
              : `${uploadNote.jumlah_baris} baris tersimpan`}
          </p>

          {uploadNote.batch.length > 0 && (
            <ul className="mt-1 space-y-0.5 text-slate-800">
              {uploadNote.batch.map((b) => (
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

          {uploadNote.ditolak.length > 0 && (
            <div className="mt-3 border-t border-amber-200 pt-3">
              <p className="font-medium text-amber-900">
                {uploadNote.ditolak.length} baris TIDAK tersimpan
              </p>
              <p className="mt-0.5 text-xs text-amber-800">
                Perusahaan rekanannya tidak dapat ditemukan dari nomor polis, sehingga barisnya
                tidak punya tempat di grid mana pun. Perbaiki nomor polisnya lalu unggah ulang baris
                ini saja.
              </p>
              <ul className="mt-2 space-y-0.5 text-xs text-amber-900">
                {uploadNote.ditolak.map((row) => (
                  <li key={`${row.baris}-${row.nomor_polis}`}>
                    Baris <span className="font-medium tabular-nums">{row.baris}</span> —{' '}
                    {row.nomor_polis === '' ? '(nomor polis kosong)' : row.nomor_polis} —{' '}
                    {row.pesan}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {uploadNote.batch.length > 0 && (
            <p className="mt-2 text-xs text-slate-700">
              Baris yang menunggu proses belum menjadi klaim. Pemrosesan menunggu tombol Proses
              Klaim, yang belum tersedia di aplikasi ini.
            </p>
          )}

          <div className="mt-3">
            <Button tone="halus" onClick={() => setUploadNote(null)}>
              Tutup
            </Button>
          </div>
        </div>
      )}

      {portal === null ? (
        <div className="mt-5">
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data klaim dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        </div>
      ) : activeSource === '' ? null : (
        // `key` pada tab membuat berpindah tab MEMASANG ULANG tabelnya: kata kunci,
        // halaman, dan baris yang terbuka ikut hilang. Bukan kerapian — ketiga tab
        // membaca tabel berbeda, dan kode perusahaan yang terbuka di satu tab belum tentu
        // ada di tab lain.
        <CompanyTable key={activeSource} source={activeSource} table={activeTable} />
      )}
    </main>
  )
}

/**
 * Tiga tombol layar lama yang mesinnya belum dibangun.
 *
 * Mereka ditampilkan NONAKTIF beserta alasannya, bukan disembunyikan — perlakuan yang
 * sama dengan butir menu yang belum punya layar. Menyembunyikannya membuat layar ini
 * tampak selesai padahal separuh alurnya belum ada.
 *
 * Setiap tombol menyebut modul yang menahannya, supaya "kapan tersedia" dapat dijawab
 * tanpa membuka dokumen apa pun.
 */
function PendingActions() {
  const pending = [
    {
      label: 'Proses Klaim',
      reason:
        'Membuat case klaim dari setiap baris batch. Menunggu modul Input Register, Objek & Coverage, Estimasi, dan Akseptasi.',
    },
    {
      label: 'Generate DLA',
      reason: 'Menerbitkan DLA untuk klaim batch ini. Menunggu modul PLA/Pre-DLA/DLA.',
    },
    {
      label: 'Cek Premi',
      reason:
        'Membaca total premi dan total klaim lewat layanan REST di luar aplikasi. Menunggu modul Integrasi Sistem Luar.',
    },
  ]

  return (
    <section
      aria-labelledby="aksi-belum-tersedia"
      className="mt-4 rounded-kartu border border-slate-200 bg-slate-50/70 p-4"
    >
      <h2 id="aksi-belum-tersedia" className="text-sm font-medium text-slate-800">
        Belum tersedia di aplikasi ini
      </h2>
      <p className="mt-1 text-xs text-slate-600">
        Ketiga tindakan berikut ada di layar lama dan masih dikerjakan di Pega. Gunakan aplikasi
        lama untuk menjalankannya.
      </p>
      <ul className="mt-3 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        {pending.map((item) => (
          <li key={item.label} className="flex items-start gap-2">
            <Button tone="kedua" disabled aria-describedby={`alasan-${item.label}`}>
              {item.label}
            </Button>
            <span id={`alasan-${item.label}`} className="max-w-xs text-xs text-slate-600">
              {item.reason}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
