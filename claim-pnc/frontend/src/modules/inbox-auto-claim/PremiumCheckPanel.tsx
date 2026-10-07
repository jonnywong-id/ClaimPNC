import { useState, type FormEvent } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { SelectField } from '@/components/SelectField'
import { formatRupiah } from '@/lib/money'

import { useAutoClaimPremiumCheck, useAutoClaimPremiumChoices } from './api'
import { loadMessage } from './messages'

/**
 * Tab Cek Premi — tab keempat layar lama (`InboxAutoClaim-Harness.xml` :50640).
 *
 * Petugas memilih Nama Bisnis dan Sumber Bisnis, lalu tombol Cek Premi menampilkan:
 *
 *   - Total Premi — premi terbayar menurut layanan `GetPremiumPaid_SPK`;
 *   - Total Klaim — nilai klaim Kredit yang sudah Sukses Klaim untuk pasangan yang sama.
 *
 * # Dua perbedaan yang disengaja dari layar lama
 *
 *   - Kolom "Max Premi (%)" tidak ada: activity mana pun di export tidak pernah mengisinya,
 *     sehingga di Pega selalu kosong.
 *   - Hasil tetap tampil walau belum ada klaim sukses. Layar lama menyembunyikan seluruh
 *     hasilnya bila Total Klaim kosong (`pyContainerVisibleWhen TempPremi.District!=''`),
 *     sehingga petugas tidak dapat membedakan "belum ada klaim" dari "tombol tidak bekerja".
 *
 * Isian memakai dropdown, bukan autocomplete bebas: yang dikirim harus KODE yang ada di
 * master, dan teks ketikan bebas tidak punya kode.
 */
export function PremiumCheckPanel() {
  const choices = useAutoClaimPremiumChoices()

  const [business, setBusiness] = useState('')
  const [source, setSource] = useState('')
  const [pair, setPair] = useState<{ bisnis: string; sumber: string } | null>(null)

  const check = useAutoClaimPremiumCheck(pair)

  const businessList = choices.data?.bisnis ?? []
  const sourceList = choices.data?.sumber_bisnis ?? []

  function submit(event: FormEvent) {
    event.preventDefault()
    if (business === '' || source === '') return
    if (pair !== null && pair.bisnis === business && pair.sumber === source) {
      // Pasangan yang sama ditekan lagi: premi dapat berubah sejak pemeriksaan terakhir.
      check.refetch()
      return
    }
    setPair({ bisnis: business, sumber: source })
  }

  if (choices.isError) {
    const message = loadMessage(choices.error)
    return (
      <div className="mt-5">
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      </div>
    )
  }

  const result = check.data
  const businessName = businessList.find((b) => b.kode === result?.kode_bisnis)?.nama ?? ''
  const sourceName = sourceList.find((s) => s.kode === result?.kode_sumber_bisnis)?.nama ?? ''

  return (
    <div className="mt-5 flex flex-col gap-5">
      <form
        onSubmit={submit}
        aria-label="Cek Premi"
        className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] md:items-end"
      >
        <SelectField
          id="cek-premi-bisnis"
          label="Nama Bisnis"
          value={business}
          onChange={(event) => setBusiness(event.target.value)}
          disabled={choices.isPending}
          emptyText={choices.isPending ? 'Memuat…' : '— pilih bisnis —'}
          options={businessList.map((b) => ({ value: b.kode, label: b.nama || b.kode }))}
        />
        <SelectField
          id="cek-premi-sumber"
          label="Sumber Bisnis"
          value={source}
          onChange={(event) => setSource(event.target.value)}
          disabled={choices.isPending}
          emptyText={choices.isPending ? 'Memuat…' : '— pilih sumber bisnis —'}
          options={sourceList.map((s) => ({ value: s.kode, label: s.nama || s.kode }))}
        />
        <Button
          type="submit"
          tone="utama"
          disabled={business === '' || source === '' || check.isFetching}
        >
          {check.isFetching ? 'Memeriksa…' : 'Cek Premi'}
        </Button>
      </form>

      {check.isError && (
        <ErrorMessage
          title="Cek premi gagal"
          description={
            check.error instanceof APIError
              ? check.error.message
              : 'Periksa koneksi jaringan Anda, lalu coba lagi.'
          }
          tone={
            check.error instanceof NetworkError ||
            (check.error instanceof APIError && check.error.status >= 500)
              ? 'gangguan'
              : 'penolakan'
          }
        />
      )}

      {result && !check.isError && (
        <section
          aria-label="Hasil cek premi"
          className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut sm:p-5"
        >
          <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
            <Item label="Nama Bisnis" value={businessName || '—'} />
            <Item label="ID Bisnis" value={result.kode_bisnis} />
            <Item label="Sumber Bisnis" value={sourceName || '—'} />
            <Item label="ID Sumber Bisnis" value={result.kode_sumber_bisnis} />
            <Item
              label="Total Premi"
              value={formatRupiah(result.total_premi, { withoutSymbol: true })}
              strong
            />
            <Item
              label="Total Klaim"
              value={
                result.total_klaim === ''
                  ? 'Belum ada klaim Sukses Klaim'
                  : formatRupiah(result.total_klaim, { withoutSymbol: true })
              }
              strong={result.total_klaim !== ''}
            />
          </dl>
        </section>
      )}
    </div>
  )
}

function Item({ label, value, strong }: Readonly<{ label: string; value: string; strong?: boolean }>) {
  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd
        className={
          strong
            ? 'mt-0.5 text-lg font-semibold tabular-nums text-slate-900'
            : 'mt-0.5 break-words text-slate-800'
        }
      >
        {value}
      </dd>
    </div>
  )
}
