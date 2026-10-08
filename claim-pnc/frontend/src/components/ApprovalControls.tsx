import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import type { Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

/**
 * Kontrol bersama layar master yang punya alur persetujuan tiga tab — Approve, Reject,
 * Waiting Approval: Master Kategori Sparepart, Master Tipe Sparepart, Master Grouping
 * Sparepart, Master Sparepart, Master Panel, dan Master Bengkel.
 *
 * Keenam layar itu menggambar bagian-bagian ini dengan markup yang sama persis; yang
 * berbeda hanya kata bendanya ("tipe", "kategori", "panel", …). Satu tempat berarti satu
 * perbaikan berlaku untuk keenamnya sekaligus.
 */

export type ApprovalTab<Id extends string = string> = Readonly<{
  id: Id
  label: string
  description: string
}>

/** Deretan tab di bawah kepala layar. */
export function ApprovalTabNav<Id extends string>({
  label,
  tabs,
  active,
  onSelect,
}: Readonly<{
  label: string
  tabs: readonly ApprovalTab<Id>[]
  active: Id
  onSelect: (id: Id) => void
}>) {
  return (
    <nav aria-label={label} className="mt-4 flex flex-wrap gap-1 border-b border-slate-200">
      {tabs.map((t) => (
        <button
          key={t.id}
          type="button"
          aria-current={active === t.id ? 'page' : undefined}
          onClick={() => onSelect(t.id)}
          className={[
            'rounded-t px-3 py-2 text-sm font-medium',
            'transition-colors duration-150 ease-halus',
            'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
            active === t.id
              ? 'border-b-2 border-blue-600 text-blue-700'
              : 'text-slate-500 hover:text-slate-800',
          ].join(' ')}
        >
          {t.label}
        </button>
      ))}
    </nav>
  )
}

/**
 * Keterangan tab beserta portal entitas yang sedang dilihat.
 *
 * Entitas disebut terang-terangan. Satu aplikasi melayani empat badan hukum dengan basis
 * data terpisah, dan "data siapa ini" tidak boleh hanya diandaikan pengguna (ADR-0030,
 * R-20).
 */
export function PortalNote({
  description,
  portal,
}: Readonly<{ description: string; portal: string | null | undefined }>) {
  return (
    <p className="mt-3 text-xs text-slate-500">
      {description}{' '}
      <span className="ml-1">
        Portal entitas: <span className="font-medium text-slate-700">{portal ?? '—'}</span>
      </span>
    </p>
  )
}

type DecisionResult = { jumlah_berubah: number; status_label: string }

/** Pesan hasil keputusan Approve/Reject — galat atau jumlah baris yang berpindah. */
export function DecisionFeedback({
  decide,
  noun,
}: Readonly<{
  decide: { isError: boolean; error: unknown; isSuccess: boolean; data?: DecisionResult | undefined }
  noun: string
}>) {
  return (
    <>
      {decide.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Keputusan belum tersimpan"
            description={
              decide.error instanceof APIError
                ? decide.error.message
                : 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.'
            }
            tone="gangguan"
          />
        </div>
      )}

      {decide.isSuccess && decide.data && (
        <p className="mt-4 rounded-kontrol border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">
          {decide.data.jumlah_berubah} {noun} dipindahkan ke{' '}
          <span className="font-medium">{decide.data.status_label}</span>.
        </p>
      )}
    </>
  )
}

/**
 * Kolom centang pada tab Waiting Approval. Dikembalikan sebagai larik supaya dapat
 * disebar langsung di depan kolom lain — kosong di tab selain Waiting Approval.
 */
export function selectColumn<T>({
  enabled,
  chosen,
  idOf,
  nameOf,
  disabled,
  onToggle,
}: Readonly<{
  enabled: boolean
  chosen: ReadonlySet<string>
  idOf: (row: T) => string
  nameOf: (row: T) => string
  disabled: boolean
  onToggle: (id: string) => void
}>): Column<T>[] {
  if (!enabled) return []
  return [
    {
      key: 'pilih',
      title: 'Pilih',
      width: '4.5rem',
      noSort: true,
      value: (row) => (chosen.has(idOf(row)) ? 'dipilih' : ''),
      render: (row) => (
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/50"
            checked={chosen.has(idOf(row))}
            disabled={disabled}
            onChange={() => onToggle(idOf(row))}
          />
          <span className="sr-only">Pilih {nameOf(row)}</span>
        </label>
      ),
    },
  ]
}

/** Membalik keanggotaan `id` pada himpunan centang, tanpa mengubah himpunan asalnya. */
export function toggled(current: ReadonlySet<string>, id: string): Set<string> {
  const next = new Set(current)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  return next
}

/**
 * DecisionBar adalah tombol Approve dan Reject untuk seluruh baris yang dicentang, dan —
 * pada layar yang tabelnya punya kolom ALASAN_TOLAK — satu isian Catatan.
 *
 * Tombolnya mati selama belum ada yang dicentang — bukan disembunyikan. Tombol yang hilang
 * membuat pengguna mencari fiturnya; tombol yang mati menunjukkan apa yang harus dilakukan
 * lebih dulu.
 */
export function DecisionBar({
  noun,
  count,
  isBusy,
  onApprove,
  onReject,
  onClear,
  note,
}: Readonly<{
  noun: string
  count: number
  isBusy: boolean
  onApprove: () => void
  onReject: () => void
  onClear: () => void
  note?: { value: string; onChange: (value: string) => void }
}>) {
  const controls = (
    <>
      <span className="min-w-0 flex-1 text-sm text-slate-700">
        {count === 0 ? (
          `Centang ${noun} yang akan diputuskan.`
        ) : (
          <>
            <span className="font-medium">
              {count} {noun}
            </span>{' '}
            dipilih.
          </>
        )}
      </span>
      {count > 0 && (
        <Button tone="halus" onClick={onClear} disabled={isBusy}>
          Bersihkan
        </Button>
      )}
      {/*
        Namanya "Approve terpilih", bukan "Approve" saja.

        Bukan sekadar demi kejelasan kalimat: tab di atasnya juga bernama "Approve" dan
        "Reject" — caption Pega yang memang harus ditiru (D-13) — sehingga tombol bernama
        sama membuat dua kontrol yang sama sekali berbeda tidak dapat dibedakan dari
        namanya. Pembaca layar mengumumkan keduanya dengan kata yang sama persis.
      */}
      <Button tone="utama" onClick={onApprove} disabled={isBusy || count === 0}>
        {isBusy ? 'Menyimpan…' : 'Approve terpilih'}
      </Button>
      <Button tone="kedua" onClick={onReject} disabled={isBusy || count === 0}>
        Reject terpilih
      </Button>
    </>
  )

  if (!note) {
    return (
      <div className="mt-4 flex flex-wrap items-center gap-2 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
        {controls}
      </div>
    )
  }

  return (
    <div className="mt-4 space-y-3 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <div className="flex flex-wrap items-center gap-2">{controls}</div>

      {/*
        Catatan hanya tersimpan pada keputusan TOLAK; kolomnya memang bernama ALASAN_TOLAK.
        Itu dinyatakan di layar, bukan dibiarkan menjadi kejutan saat petugas mengetiknya
        lalu menekan Approve.
      */}
      <div>
        <label htmlFor="catatan-keputusan" className="block text-sm font-medium text-slate-700">
          Catatan
        </label>
        <input
          id="catatan-keputusan"
          type="text"
          value={note.value}
          maxLength={250}
          disabled={isBusy || count === 0}
          onChange={(event) => note.onChange(event.target.value)}
          className={
            'mt-1 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-slate-900 ' +
            'shadow-lembut transition-[border-color,box-shadow] duration-150 ease-halus ' +
            'focus:border-blue-500 focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20 ' +
            'disabled:bg-slate-100 disabled:text-slate-500'
          }
        />
        <p className="mt-1 text-xs text-slate-500">
          Tersimpan sebagai alasan penolakan. Pada keputusan Approve, catatan ini tidak ikut
          tersimpan.
        </p>
      </div>
    </div>
  )
}
