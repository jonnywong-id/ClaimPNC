import { useState, type ReactNode } from 'react'

import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
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
 * # Layar dibuka KOSONG, dan itu memang perilaku aslinya
 *
 * Ketiga judul yang tampak seperti tab — "Generated DLA PLA XOL", "Cari Data DLA PLA XOL",
 * dan "Approval XOL" — adalah TOMBOL; ketiganya terdaftar sebagai `pyButtonLabel` di
 * `Section/InboxClaimXOL-Section.xml`. Menekan salah satunya memanggil
 * `Activity/ToFlaggingDataXOLByRequest` yang mengisi satu properti penanda —
 * `FlagDataForSerachingXOL.source := Param.ParFlags` — lalu menyegarkan section.
 *
 * Isi layar bergantung penuh pada penanda itu. Ketiga wadah isi memakai
 * `pyContainerVisibleWhen` atasnya:
 *
 *	:3858	source=='1'	PncPICTeknik  → PILIH MASTER XOL · DATA XOL BASED ON DOL AND COL · PLA/DLA
 *	:21730	source=='2'	siapa saja    → Sec_Detail_claim_XOL (cari + hasilnya)
 *	:23129	source=='1'	CaseManager   → Approval XOL · DATA MASTER XOL
 *
 * Sebelum satu tombol ditekan, `source` masih kosong dan **tidak satu pun wadah tampil** —
 * yang terlihat hanya deretan tombolnya. Itulah tampilan awal yang ditiru di sini, dan
 * itu pula sebabnya tidak ada permintaan yang berangkat saat layar baru dibuka.
 *
 * # "Generated DLA PLA XOL" dan "Approval XOL" membuka hal yang SAMA
 *
 * Keduanya mengirim `ParFlags = 1`. Yang memisahkan isinya di Pega bukan tombolnya
 * melainkan access group pembacanya — dan kedua wadah `source=='1'` sama-sama
 * memperbolehkan `GCNMFW:Administrators`. Akun yang memegangnya melihat KEEMPAT blok
 * sekaligus saat salah satu tombol ditekan: INSERT DOL DAN COL, DATA XOL BASED ON DOL AND
 * COL, DATA XOL KLAIM, dan DATA MASTER XOL.
 *
 * Itulah keadaan yang berlaku di sini, dan bukan karena dipilih: peran belum dapat
 * ditegakkan — tabel peran adalah `TKT-F3-004`, yang dapat dibangun tetapi belum dapat
 * diisi karena penugasan operator ke peran tidak ada di basis data maupun di export
 * (`11-SECURITY.md` §3.1). Setiap pengguna karena itu berkelakuan seperti akun
 * ber-access-group lengkap.
 *
 * Dua tombol yang berbuat sama memang terasa ganjil, dan ganjil itu MILIK layar aslinya.
 * Menjadikannya dua isi yang berbeda akan mengarang pembedaan yang tidak ada di sumbernya.
 *
 * # Layar ini tidak mengubah apa pun
 *
 * Keputusan Work Owner 2026-09-20. Keempat tabel yang ditulis sistem lama tetap dimiliki
 * Pega selama masa paralel (`P-1`), sehingga menulis dari sini berarti dua sistem menulis
 * satu tabel dengan aturan berbeda. Tombol yang belum tersedia dinyatakan demikian di
 * tempatnya, bukan disembunyikan.
 */
export function InboxXOLPage() {
  const [view, setView] = useState<ViewKey | null>(null)
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
      <ActionBand
        label="Inbox XOL"
        leadLabel="Generated DLA PLA XOL"
        active={view}
        onSelect={setView}
      />

      <ActionBand
        label="Inbox XOL Komite"
        leadLabel="Approval XOL"
        active={view}
        onSelect={setView}
      />

      <ContentBand view={view} />
    </PageFrame>
  )
}

/**
 * ViewKey menggantikan `FlagDataForSerachingXOL.source`, nilai demi nilai.
 *
 *	'data'	↔ ParFlags 1	"Generated DLA PLA XOL" · "Approval XOL"
 *	'cari'	↔ ParFlags 2	"Cari Data DLA PLA XOL"
 *
 * `null` adalah keadaan awal — padanan `source` yang masih kosong, yakni layar yang baru
 * dibuka dan belum menampilkan apa pun.
 */
type ViewKey = 'data' | 'cari'

type BandProps = {
  label: string
  leadLabel: string
  active: ViewKey | null
  onSelect: (key: ViewKey) => void
}

/**
 * ActionBand menggambar satu deret tombol, meniru satu baris di layar lama.
 *
 * Tiap deret berisi tombol khasnya sendiri ditambah "Cari Data DLA PLA XOL" yang BERULANG
 * di kedua deret. Pengulangan itu bukan kekeliruan salinan: section lama benar-benar
 * memuat tombol itu dua kali (`:1940` dan `:3052`), keduanya mengirim `ParFlags = 2`, dan
 * keduanya membuka wadah yang sama. Ia ada dua kali karena tiap peran hanya melihat satu
 * deret, dan keduanya butuh jalan ke panel pencarian.
 *
 * Deret dibungkus `<section>` berlabel supaya kedua tombol "Cari Data DLA PLA XOL" tetap
 * dapat dibedakan — oleh pembaca layar maupun oleh pengujian — tanpa mengubah teks yang
 * dilihat pengguna.
 */
function ActionBand({ label, leadLabel, active, onSelect }: BandProps) {
  return (
    <section
      aria-label={label}
      className="mt-4 rounded-kartu border border-slate-200 bg-white p-3 shadow-lembut"
    >
      <div className="flex flex-wrap items-center gap-2">
        <BandButton tone="utama" pressed={active === 'data'} onClick={() => onSelect('data')}>
          {leadLabel}
        </BandButton>

        <BandButton
          tone="kedua"
          pressed={active === 'cari'}
          onClick={() => onSelect('cari')}
        >
          Cari Data DLA PLA XOL
        </BandButton>
      </div>
    </section>
  )
}

/**
 * BandButton menambahkan satu hal yang TIDAK ada di layar lama: tanda tombol mana yang
 * sedang terbuka.
 *
 * Di Pega warna tombol tetap — ia gaya yang dipatok (`Strong` dan `Dashboard`), bukan
 * keadaan. Akibatnya tidak ada petunjuk sama sekali tentang isi yang sedang tampil milik
 * tombol yang mana, dan dengan dua tombol "Cari Data DLA PLA XOL" yang serupa persis itu
 * benar-benar menyesatkan.
 *
 * `aria-pressed` ditambahkan karena ketiga tombol ini memang berkelakuan seperti sakelar
 * yang saling meniadakan, bukan seperti perintah sekali jalan.
 */
function BandButton({
  tone,
  pressed,
  onClick,
  children,
}: {
  tone: 'utama' | 'kedua'
  pressed: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <Button
      tone={tone}
      aria-pressed={pressed}
      onClick={onClick}
      className={pressed ? 'ring-2 ring-blue-500/40 ring-offset-1' : undefined}
    >
      {children}
    </Button>
  )
}

/**
 * ContentBand menggambar wadah isi — satu saja, mengikuti penanda yang sedang berlaku.
 *
 * Panel yang tidak tampil TIDAK dirakit sama sekali, bukan sekadar disembunyikan. Itu yang
 * menjaga janji tampilan awal: hook di tiap panel baru menembak permintaannya saat panel
 * itu benar-benar dipasang, sehingga membuka layar ini tidak memanggil satu pun rute.
 */
function ContentBand({ view }: { view: ViewKey | null }) {
  if (view === null) {
    return (
      <section
        aria-label="Isi Inbox XOL"
        className="mt-4 rounded-kartu border border-slate-200 bg-white px-4 py-6 text-sm text-slate-600 shadow-lembut"
      >
        Pilih salah satu tombol di atas untuk menampilkan datanya.
      </section>
    )
  }

  return (
    <section aria-label="Isi Inbox XOL">
      {view === 'data' ? (
        <>
          {/*
            Urutannya mengikuti urutan wadahnya di section lama, bukan selera:
            INSERT DOL DAN COL + DATA XOL BASED ON DOL AND COL (`:3858`), lalu
            DATA XOL KLAIM + DATA MASTER XOL (`:23129`).
          */}
          <ClaimPanel />
          <ApprovalPanel active />
        </>
      ) : (
        <AdvicePanel />
      )}
    </section>
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
