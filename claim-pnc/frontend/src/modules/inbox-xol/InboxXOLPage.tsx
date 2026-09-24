import { useState, type ReactNode } from 'react'

import { useSelectedPortal } from '@/app/portal'
import { ErrorMessage } from '@/components/ErrorMessage'

import { AdvicePanel } from './AdvicePanel'
import { ApprovalPanel } from './ApprovalPanel'
import { ClaimPanel } from './ClaimPanel'

/**
 * Inbox XOL — menu `MENU_ID 53`, pengganti harness `Inbox_XOL_Harness`.
 *
 * XOL — Excess of Loss — adalah treaty reasuransi non-proporsional yang menanggung
 * kerugian di atas batas tertentu (`CONTEXT.md`). Layar ini bukan layar klaim perorangan:
 * ia mengakumulasi klaim satu tahun perjanjian, lalu memperlihatkan pemberitahuan PLA/DLA
 * yang sudah diterbitkan kepada para reasuradur beserta status persetujuannya.
 *
 * # Dua tab, bukan empat
 *
 * `Section/InboxClaimXOL-Section.xml` memuat dua tab teratas, masing-masing dengan
 * kondisi tampil yang membaca access group:
 *
 *	Inbox XOL        GCNMFW:PncPICTeknik
 *	Inbox XOL Komite GCNMFW:CaseManager
 *
 * Tiga judul yang tampak seperti tab — "Generated DLA PLA XOL", "Cari Data DLA PLA XOL",
 * dan "INSERT DOL DAN COL" — sebenarnya TOMBOL di dalam tab pertama; ketiganya terdaftar
 * sebagai `pyButtonLabel`. Dua yang pertama menjadi panel PLA/DLA di bawah; yang ketiga
 * menulis data dan karena itu belum dipindahkan.
 *
 * # Kedua tab terlihat oleh SETIAP pengguna hari ini
 *
 * Pemisahan peran belum dapat ditegakkan: tabel peran adalah `TKT-F3-004`, yang dapat
 * dibangun tetapi belum dapat diisi karena penugasan operator ke peran tidak ada di basis
 * data maupun di export (`11-SECURITY.md` §3.1). Selama modul ini MEMBACA SAJA,
 * taruhannya terbatas — antrean komite yang terlihat bukan antrean yang dapat disetujui.
 *
 * # Layar ini tidak mengubah apa pun
 *
 * Keputusan Work Owner 2026-09-20. Keempat tabel yang ditulis sistem lama tetap dimiliki
 * Pega selama masa paralel (`P-1`), sehingga menulis dari sini berarti dua sistem menulis
 * satu tabel dengan aturan berbeda. Tombol yang belum tersedia dinyatakan demikian di
 * tempatnya, bukan disembunyikan.
 */
export function InboxXOLPage() {
  const [tab, setTab] = useState<TabKey>('inbox')
  const portal = useSelectedPortal((state) => state.alias)

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Perjanjian XOL dan nilai klaimnya milik satu badan hukum, dan aplikasi ini ' +
            'melayani empat. Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  return (
    <PageFrame>
      <Tabs active={tab} onSelect={setTab} />

      {tab === 'inbox' ? (
        <>
          <ClaimPanel />
          <AdvicePanel />
          <InsertNotice />
        </>
      ) : (
        <ApprovalPanel active={tab === 'komite'} />
      )}
    </PageFrame>
  )
}

type TabKey = 'inbox' | 'komite'

const TABS: Array<{ key: TabKey; label: string }> = [
  { key: 'inbox', label: 'Inbox XOL' },
  { key: 'komite', label: 'Inbox XOL Komite' },
]

/**
 * Tabs menggambar kedua tab beserta keadaannya.
 *
 * Tab yang tidak aktif TIDAK dibongkar isinya dari DOM oleh komponen ini — yang
 * mengendalikan pemuatan adalah `enabled` pada hook masing-masing panel. Dengan begitu,
 * membuka layar tidak menembak permintaan tab yang belum dilihat siapa pun.
 */
function Tabs({ active, onSelect }: { active: TabKey; onSelect: (key: TabKey) => void }) {
  return (
    <div className="mt-4 overflow-x-auto" role="tablist" aria-label="Bagian Inbox XOL">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {TABS.map((entry) => {
          const selected = entry.key === active
          return (
            <button
              key={entry.key}
              type="button"
              role="tab"
              aria-selected={selected}
              onClick={() => onSelect(entry.key)}
              className={
                'rounded-t-kontrol px-4 py-2 text-sm font-medium transition ' +
                (selected
                  ? 'border-b-2 border-blue-600 bg-white text-blue-700'
                  : 'border-b-2 border-transparent text-slate-600 hover:text-slate-900')
              }
            >
              {entry.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}

/**
 * InsertNotice menjelaskan ketiadaan tombol "INSERT DOL DAN COL".
 *
 * Tombol itu MENULIS ke `POOLDATA.XOL_TABLE_ALL_KLAIM` — menghapus lalu menyisipkan ulang
 * baris untuk satu tanggal dan penyebab kerugian. Selama masa paralel tabel itu masih
 * dimiliki Pega (`P-1`).
 *
 * Keterangannya ditulis di tempat tombolnya dulu berada, bukan di kaki halaman: pengguna
 * yang mencari tombol mencarinya di sini.
 */
function InsertNotice() {
  return (
    <p className="mt-4 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
      Menambah data DOL dan Cause Of Loss belum tersedia di aplikasi baru. Selama masa
      paralel, penambahannya masih dilakukan lewat aplikasi Pega; layar ini menampilkan
      hasilnya.
    </p>
  )
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox XOL</h1>
        <p className="mt-1 text-sm text-slate-600">
          Akumulasi klaim per perjanjian Excess of Loss, beserta pemberitahuan PLA/DLA
          kepada reasuradur dan antrean persetujuannya.
        </p>
      </header>
      {children}
    </div>
  )
}
