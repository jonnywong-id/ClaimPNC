import { useMemo, useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useInputAcceptation, useSubmitInputAcceptation } from './api'
import type { DetailResponse, Field, Grid, GridRow, Group } from './types'

/**
 * Acceptation Claim — Flow Action `InputAcceptation` pada kelas
 * `ASM-FW-GCNMFW-Work-ClaimTreatyNonProp`.
 *
 * Layar ini dibuka dari nomor klaim di Inbox Claim Treaty Non Prop (`MENU_ID 55`), bukan dari
 * menu. Di Pega, sel Claim.ID pada grid inbox ber-`pyAction openAssignment`, dan Flow Action
 * inilah satu-satunya yang terdaftar pada kelas objek kerjanya.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * SELURUHNYA dari server. Sebelas kelompok, ~50 isian, dan 13 grid dibaca dari
 * `Section/InputAcceptation-Section.xml` dan tercatat di
 * `internal/inputacceptation/section.go`. Menyalinnya ke sini berarti daftar yang sama hidup
 * di dua tempat, dan yang satu akan tertinggal saat yang lain diperbaiki.
 *
 * Judulnya "Acceptation Claim" — `<pyValue>` kontainer di section, bukan nama menu.
 *
 * # Yang dapat diubah, dan yang hanya ditampilkan
 *
 * Section menandai 120 dari 273 selnya read-only. Layar menggambar isian yang dapat diubah
 * sebagai kotak isian dan sisanya sebagai teks — dan pembedaan itu datang dari server
 * (`dapat_diubah`), bukan ditebak di sini. Yang ditebak akan berselisih dengan yang divalidasi
 * server, dan pengguna baru mengetahuinya saat Submit ditolak.
 */
export function InputAcceptationPage() {
  const { no_klaim: claimID = '' } = useParams()
  const portal = useSelectedPortal((state) => state.alias)
  const detail = useInputAcceptation(claimID)

  if (portal === null) {
    return (
      <PageFrame claimID={claimID}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Akseptasi klaim treaty milik satu badan hukum, dan aplikasi ini melayani ' +
            'empat. Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (detail.isPending) {
    return (
      <PageFrame claimID={claimID}>
        <p className="mt-6 text-sm text-slate-600" role="status">
          Memuat akseptasi {claimID}…
        </p>
      </PageFrame>
    )
  }

  if (detail.isError || !detail.data) {
    return (
      <PageFrame claimID={claimID}>
        <ErrorMessage
          title="Akseptasi tidak dapat dibuka"
          description={messageOf(detail.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  return <Loaded data={detail.data} claimID={claimID} />
}

/**
 * Loaded menggambar layar yang isinya sudah tiba.
 *
 * Ia dipisah supaya keadaan form — isian yang sedang diubah — lahir bersama datanya. Menaruh
 * `useState` di komponen luar berarti keadaan itu bertahan saat pengguna berpindah ke klaim
 * lain, dan perubahan pada klaim A akan terbawa ke klaim B.
 */
function Loaded({ data, claimID }: { data: DetailResponse; claimID: string }) {
  const submit = useSubmitInputAcceptation(claimID)

  // Hanya SELISIHNYA yang disimpan, bukan salinan seluruh isian. Dengan begitu nilai yang
  // tidak disentuh pengguna tetap datang dari server pada setiap muat ulang, dan tidak ada
  // kemungkinan layar mengirim balik nilai lama yang sudah berubah di server.
  const [changed, setChanged] = useState<Record<string, string>>({})

  const editableGrids = useMemo(() => collectEditableGrids(data), [data])

  /**
   * Pelanggaran per isian, dipetakan ke kuncinya supaya dapat ditandai di tempatnya.
   *
   * `APIError.detail` memakai `field` ATAU `kolom` — keduanya hidup berdampingan di kontrak
   * galat hari ini (`TKT-F1-004` belum selesai), dan modul ini menampung keduanya alih-alih
   * menebak yang mana. Satu pesan di atas form akan memaksa pengguna menebak isian mana yang
   * ditolak dari ~11 isian yang dapat diubah.
   */
  const violations = useMemo(() => {
    if (!(submit.error instanceof APIError)) return {}
    const byField: Record<string, string> = {}
    for (const item of submit.error.detail) {
      const key = item.field ?? item.kolom
      if (key) byField[key] = item.pesan
    }
    return byField
  }, [submit.error])

  function change(key: string, value: string) {
    setChanged((previous) => ({ ...previous, [key]: value }))
  }

  function send() {
    submit.mutate({ isian: changed, tabel: editableGrids })
  }

  const dirty = Object.keys(changed).length > 0

  return (
    <PageFrame claimID={data.no_klaim} statusWork={data.status_kerja}>
      <div className="mt-6 space-y-8">
        {data.kelompok.map((group) => (
          <GroupSection
            key={group.kode}
            group={group}
            changed={changed}
            violations={violations}
            onChange={change}
          />
        ))}
      </div>

      <SubmitBar
        dirty={dirty}
        pending={submit.isPending}
        error={submit.isError ? messageOf(submit.error) : ''}
        success={submit.isSuccess ? submit.data.pesan : ''}
        onSubmit={send}
      />

      <PlannedDifferences lines={data.selisih_terencana} />

      <p className="mt-6 text-xs text-slate-500">
        Terakhir diubah oleh {data.operator_pengubah || '—'}.
      </p>
    </PageFrame>
  )
}

function PageFrame({
  claimID,
  statusWork,
  children,
}: {
  claimID: string
  statusWork?: string
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <Link
          to="/inbox-claim-treaty-non-prop"
          className={[
            'text-sm text-blue-700 underline-offset-2 hover:underline',
            'focus:outline-none focus-visible:rounded-kontrol',
            'focus-visible:ring-2 focus-visible:ring-blue-500/50',
          ].join(' ')}
        >
          ← Kembali ke Claims In Progress
        </Link>

        {/*
          Judulnya "Acceptation Claim" — `<pyValue>` kontainer di section, bukan nama menu.
          `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar ulang.
        */}
        <h1 className="mt-2 text-xl font-semibold text-slate-900">Acceptation Claim</h1>

        <p className="mt-1 text-sm text-slate-600">
          Klaim {claimID}
          {statusWork ? ` — status alur kerja: ${statusWork}` : ''}
        </p>
      </header>
      {children}
    </div>
  )
}

/** GroupSection menggambar satu kelompok isian beserta grid sesudahnya. */
function GroupSection({
  group,
  changed,
  violations,
  onChange,
}: {
  group: Group
  changed: Record<string, string>
  violations: Record<string, string>
  onChange: (key: string, value: string) => void
}) {
  return (
    <section className="rounded-kartu border border-slate-200 bg-white px-4 py-4">
      <h2 className="text-base font-semibold text-slate-900">{group.judul}</h2>

      {group.isian.length > 0 && (
        <dl className="mt-3 grid grid-cols-1 gap-x-6 gap-y-3 md:grid-cols-2 xl:grid-cols-3">
          {group.isian.map((field) => (
            <FieldRow
              key={field.kunci}
              field={field}
              value={changed[field.kunci] ?? field.nilai}
              violation={violations[field.kunci]}
              onChange={onChange}
            />
          ))}
        </dl>
      )}

      {group.tabel.map((grid) => (
        <GridTable key={grid.kode} grid={grid} />
      ))}
    </section>
  )
}

/**
 * FieldRow menggambar satu isian.
 *
 * Isian terhalang digambar dengan alasannya, bukan disembunyikan dan bukan dibiarkan kosong
 * tanpa keterangan. Menyembunyikannya membuat pengguna yang membandingkan layar ini dengan
 * Pega mengira isiannya hilang.
 */
function FieldRow({
  field,
  value,
  violation,
  onChange,
}: {
  field: Field
  value: string
  violation: string | undefined
  onChange: (key: string, value: string) => void
}) {
  const id = `isian-${field.kunci}`

  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium text-slate-500">
        <label htmlFor={field.dapat_diubah && !field.terhalang ? id : undefined}>
          {field.judul}
        </label>
      </dt>

      {field.terhalang ? (
        <dd className="mt-0.5">
          <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-900">
            belum tersedia
          </span>
          {field.alasan_terhalang && (
            <p className="mt-1 text-xs text-slate-500">{field.alasan_terhalang}</p>
          )}
        </dd>
      ) : field.dapat_diubah ? (
        <dd className="mt-0.5">
          <input
            id={id}
            value={value}
            onChange={(event) => onChange(field.kunci, event.target.value)}
            aria-invalid={violation ? true : undefined}
            className={[
              'w-full rounded-kontrol border px-2 py-1 text-sm text-slate-900',
              'transition-[border-color,box-shadow] duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20',
              violation
                ? 'border-red-500 focus:border-red-600'
                : 'border-slate-300 focus:border-blue-500',
            ].join(' ')}
          />
          {violation && (
            <p className="mt-1 text-xs text-red-700" role="alert">
              {violation}
            </p>
          )}
        </dd>
      ) : (
        <dd className="mt-0.5 text-sm break-words text-slate-900">{value || '—'}</dd>
      )}
    </div>
  )
}

/**
 * GridTable menggambar satu tabel.
 *
 * Tabelnya digulir menyamping pada layar sempit, bukan dilipat: grid terlebar di sini punya
 * sepuluh kolom, dan melipatnya menyembunyikan kolom nilai uang yang justru menjadi alasan
 * layar ini dibuka.
 */
function GridTable({ grid }: { grid: Grid }) {
  if (grid.terhalang) {
    return (
      <div className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3">
        <h3 className="text-sm font-semibold text-amber-900">
          {grid.judul} belum tersedia
        </h3>
        {grid.alasan_terhalang && (
          <p className="mt-1 text-sm text-slate-700">{grid.alasan_terhalang}</p>
        )}
      </div>
    )
  }

  return (
    <div className="mt-4">
      <h3 className="text-sm font-medium text-slate-800">{grid.judul}</h3>

      <div className="mt-2 overflow-x-auto">
        <table className="min-w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50 text-left">
              {grid.kolom.map((column) => (
                <th
                  key={column.kunci}
                  scope="col"
                  className="px-3 py-2 font-medium whitespace-nowrap text-slate-700"
                >
                  {column.judul}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {grid.baris.length === 0 ? (
              <tr>
                <td
                  colSpan={grid.kolom.length}
                  className="px-3 py-3 text-slate-500 italic"
                >
                  Tidak ada baris.
                </td>
              </tr>
            ) : (
              grid.baris.map((row, index) => (
                <tr key={index} className="border-b border-slate-100">
                  {grid.kolom.map((column) => (
                    <td
                      key={column.kunci}
                      className="px-3 py-2 whitespace-nowrap text-slate-900"
                    >
                      {row[column.kunci] || '—'}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

/**
 * SubmitBar menggambar tombol Submit beserta jawabannya.
 *
 * Tombolnya mati selama tidak ada yang berubah. Submit yang tidak mengubah apa pun tetap
 * menembak server dan tetap ditolak, dan penolakan atas permintaan kosong hanya membingungkan.
 */
function SubmitBar({
  dirty,
  pending,
  error,
  success,
  onSubmit,
}: {
  dirty: boolean
  pending: boolean
  error: string
  success: string
  onSubmit: () => void
}) {
  return (
    <div className="mt-6 flex flex-col items-end gap-2 border-t border-slate-200 pt-4">
      <Button tone="utama" disabled={!dirty || pending} onClick={onSubmit}>
        {pending ? 'Menyimpan…' : 'Submit'}
      </Button>

      {!dirty && !error && !success && (
        <p className="text-xs text-slate-500">Belum ada perubahan untuk disimpan.</p>
      )}

      {error && (
        <p className="max-w-3xl text-right text-sm text-red-700" role="alert">
          {error}
        </p>
      )}

      {success && (
        <p className="text-sm text-green-700" role="status">
          {success}
        </p>
      )}
    </div>
  )
}

/**
 * Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah layar.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini. Tanpa catatan ini, sembilan isian
 * yang selalu kosong dan Submit yang menolak akan dilaporkan berulang kali sebagai kerusakan
 * oleh orang yang membandingkan kedua layar berdampingan.
 */
function PlannedDifferences({ lines }: { lines: string[] }) {
  if (lines.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">
        Yang berbeda dari layar lama, dan itu disengaja
      </h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
        {lines.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </section>
  )
}

/**
 * collectEditableGrids menyusun muatan grid untuk Submit.
 *
 * Hanya kolom ber-`dapat_diubah` yang ikut, dan barisnya dikirim UTUH per grid — bukan sebagai
 * selisih — karena urutan baris bermakna di layar ini.
 *
 * Grid yang tidak punya satu pun kolom yang dapat diubah TIDAK ikut sama sekali. Mengirimnya
 * kosong akan membuat server memeriksa muatan yang tidak pernah berisi apa pun.
 */
function collectEditableGrids(data: DetailResponse): Record<string, GridRow[]> {
  const result: Record<string, GridRow[]> = {}

  for (const group of data.kelompok) {
    for (const grid of group.tabel) {
      if (grid.terhalang) continue

      const editable = grid.kolom.filter((column) => column.dapat_diubah)
      if (editable.length === 0) continue

      result[grid.kode] = grid.baris.map((row) => {
        const kept: GridRow = {}
        for (const column of editable) {
          kept[column.kunci] = row[column.kunci] ?? ''
        }
        return kept
      })
    }
  }

  return result
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
