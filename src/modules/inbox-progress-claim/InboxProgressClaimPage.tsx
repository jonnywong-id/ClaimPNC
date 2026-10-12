import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { ErrorMessage } from '@/components/ErrorMessage'

import { ProgressSection } from './ProgressSection'
import { useProgressClaimMetadata } from './api'

/**
 * Inbox Progress Claim — menu `MENU_ID 65`, pengganti harness `ProgressClaim_Harness`.
 *
 * Isinya pemantauan progres klaim yang masih berjalan: sudah sampai posisi mana sebuah
 * klaim, apa status progresnya, dan kapan ia harus ditindaklanjuti berikutnya. Per `D-79`
 * ia benar-benar Inbox — barisnya pekerjaan yang menunggu, hilang begitu klaimnya tutup,
 * dan punya tenggat berupa Next Follow Up.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/ProgressClaim_Section-Section.xml` apa adanya: beberapa bagian
 * bertumpuk dalam satu halaman, masing-masing memuat datanya sendiri saat dibuka. Ia
 * sengaja TIDAK dijadikan bilah tab seperti Inbox Admin — bentuk kedua layar itu memang
 * berbeda di Pega, dan `D-13` menetapkan tampilan meniru Pega.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena tiap bagian punya kolom yang berbeda, dan daftar itu adalah hasil pembacaan
 * export Pega yang tercatat di backend. Di layar ini alasannya lebih kuat lagi: judul
 * kolomnya memakai alias Pega yang sebagian menyatakan hal yang bukan isinya — kolom
 * berjudul `District` berisi nama tertanggung — dan arti sebenarnya ikut dikirim server
 * sebagai keterangan.
 *
 * # Satu bagian yang tidak dibawa
 *
 * "Approval Progress Klaim" berada di luar lingkup (keputusan Work Owner 2026-09-21): ia
 * satu-satunya bagian yang MENULIS, dan tabel yang ditulisnya masih dimiliki sistem lama
 * selama masa berjalan paralel. Ketiadaannya dijelaskan di bawah, bukan dibiarkan sebagai
 * bagian yang hilang tanpa keterangan.
 */
export function InboxProgressClaimPage() {
  const navigate = useNavigate()
  const portal = useSelectedPortal((state) => state.alias)
  const meta = useProgressClaimMetadata()

  /**
   * Tujuan tombol "Lihat Detail Klaim".
   *
   * Layar tujuannya adalah `MENU_ID 75` "View Claim" (`PNCViewClaim`) — modul tersendiri
   * yang belum dibangun. Di sistem lama, sel pertama grid memanggil
   * `StatusProgress_act11`, yang menyusun kunci teknis Pega dari NOMOR KLAIM
   * (`"ASM-FW-GCNMFW-WORK " + pyID`) lalu membuka klaimnya.
   *
   * Yang dikirim di sini adalah nomor klaimnya saja. Awalan kelas Pega tidak dibentuk
   * ulang: `D-22` menetapkan sistem baru tidak pernah menuliskannya lagi, dan penyusunan
   * kunci itu — bila memang masih dibutuhkan — adalah urusan modul View Claim, bukan
   * urusan layar ini.
   */
  function openClaim(claimNumber: string) {
    navigate(`/view-claim/${encodeURIComponent(claimNumber)}`)
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Progres klaim milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (meta.isError) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  const sections = meta.data?.bagian ?? []
  const defaultSection = meta.data?.bagian_bawaan ?? ''

  return (
    <PageFrame>
      {sections.map((section) => (
        <ProgressSection
          key={section.kode}
          section={section}
          lines={meta.data?.lini_bisnis ?? []}
          openByDefault={section.kode === defaultSection}
          ready={meta.isSuccess}
          onOpenClaim={openClaim}
        />
      ))}

      <Notes limitations={meta.data?.keterbatasan ?? []} />
    </PageFrame>
  )
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Progress Claim</h1>
        <p className="mt-1 text-sm text-slate-600">
          Pemantauan progres klaim yang masih berjalan: posisi, status progres, dan tenggat
          tindak lanjut berikutnya.
        </p>
      </header>
      {children}
    </div>
  )
}

/**
 * Catatan di bawah layar: keterbatasan yang berlaku.
 *
 * Datang dari SERVER, bukan ditulis tetap di sini, supaya hilang dengan sendirinya begitu
 * penghalangnya hilang. Tanpa catatan ini, penyaring cabang yang belum aktif dan bagian
 * Approval yang tidak ada akan dilaporkan berulang kali sebagai kerusakan.
 */
function Notes({ limitations }: { limitations: string[] }) {
  if (limitations.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">Yang perlu diketahui</h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
        {limitations.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </section>
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
