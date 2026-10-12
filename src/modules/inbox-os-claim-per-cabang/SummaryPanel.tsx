import { Button } from '@/components/Button'
import { formatDate } from '@/components/format'
import { formatRupiah } from '@/lib/money'

import type { AgeBucket, SummaryGroup, SummaryResponse } from './types'

/**
 * Panel ringkasan di atas grid — kartu angka, sebaran umur, dan rincian per COB maupun per
 * sumber bisnis.
 *
 * # Ia TIDAK ada di Pega
 *
 * Layar lama hanya punya judul, tombol ekspor, dan grid. Panel ini diminta Work Owner
 * (2026-10-08), sehingga ia tidak punya pembanding untuk uji kesetaraan. Yang dapat diuji
 * hanyalah bahwa angkanya konsisten dengan grid di bawahnya — dan grid itu sengaja
 * DIPERTAHANKAN, bukan diganti, supaya penyelia tetap dapat menelusuri per klaim.
 *
 * # Dua nilai uang, dan keduanya ditampilkan
 *
 * Work Owner meminta total porsi treaty OR per cabang. Ia ditampilkan apa adanya — tetapi
 * tidak sendirian: tabel treaty hampir tidak memuat klaim PNC, sehingga angkanya akan Rp 0
 * di hampir semua cabang. Tanpa kartu estimasi di sebelahnya, panel ini akan terbaca sebagai
 * layar yang rusak padahal angkanya memang nol.
 */
export function SummaryPanel({
  data,
  isLoading,
  error,
  onRefresh,
  isRefreshing,
}: {
  data: SummaryResponse | undefined
  isLoading: boolean
  error: unknown
  onRefresh: () => void
  isRefreshing: boolean
}) {
  if (isLoading) {
    return (
      <section className="mt-4 rounded-kartu border border-slate-200 bg-white p-4">
        <p className="text-sm text-slate-500">Memuat ringkasan…</p>
      </section>
    )
  }

  // Panel yang gagal dimuat TIDAK menghentikan layar — grid di bawahnya tetap berguna.
  // Yang ditampilkan satu baris keterangan, bukan pita galat yang mendominasi layar.
  //
  // Bentuk jawaban ikut diperiksa, bukan hanya galatnya. Jawaban 200 yang bentuknya lain —
  // rute yang salah diarahkan, atau peladen versi lama — akan membuat `.map` melempar, dan
  // lemparan di sini MENJATUHKAN SELURUH HALAMAN termasuk gridnya. Janji "panel yang gagal
  // tidak mengosongkan grid" hanya berlaku bila kemungkinan itu ditutup di sini.
  if (error || !data || !Array.isArray(data.sebaran_umur)) {
    return (
      <section className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-kartu border border-slate-200 bg-white p-4">
        <p className="text-sm text-slate-600">
          Ringkasan tidak dapat dimuat. Daftar klaim di bawah tetap dapat dipakai.
        </p>
        <Button tone="kedua" disabled={isRefreshing} onClick={onRefresh}>
          {isRefreshing ? 'Memuat…' : 'Coba lagi'}
        </Button>
      </section>
    )
  }

  return (
    <section className="mt-4 space-y-4">
      <header className="flex flex-wrap items-baseline justify-between gap-2">
        {/*
          Tanggalnya ditulis seperti di grid — "9 Oktober 2026", bukan "2026-10-09". Satu
          layar yang menuliskan tanggal dengan dua cara memaksa pembacanya menerjemahkan
          salah satunya setiap kali, dan kolom Registration Date di bawah sudah memakai
          bentuk panjang.
        */}
        <p className="text-sm text-slate-600">
          Posisi{' '}
          <span className="font-medium text-slate-900">{formatDate(data.posisi)}</span>
          {' · '}
          {data.total_berkas} berkas outstanding
        </p>
        {/*
          Nada `kedua`, dan itu bukan pilihan rasa: dokumentasi Button menyebut "Refresh"
          sebagai contoh nadanya sendiri. Tombol buatan tangan yang sempat dipakai di sini
          memakai teks `slate-700` di atas latar putih, sehingga ia terbaca seperti tombol
          yang MATI padahal aktif — tepat kelas cacat yang pustaka komponen baku ada untuk
          mencegahnya.
        */}
        <Button tone="kedua" disabled={isRefreshing} onClick={onRefresh}>
          {isRefreshing ? 'Memuat…' : 'Muat data baru'}
        </Button>
      </header>

      {/*
        TIGA kolom, bukan empat. Kartunya memang tiga sejak kartu "Reserve treaty OR"
        dibuang, dan kisi berkolom empat menyisakan satu petak kosong di kanan — terbaca
        seperti kartu yang gagal dimuat, bukan seperti ruang yang disengaja.
      */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {/*
          "OR" di sini **Own Retention**, bukan porsi treaty OR — dua hal yang namanya
          nyaris sama dan angkanya jauh berbeda.

          Yang dijumlahkan adalah kolom grid "Reserve Claim ASM Share", yakni reserve yang
          ditahan ASM sendiri. Porsi treaty OR (`reserve × share ASM × bagian OR`) adalah
          angka LAIN yang hampir selalu Rp 0, karena tabel treaty nyaris tidak memuat klaim
          PNC — hanya 1 dari 50 cabang yang akan menampilkannya bukan nol.

          Yang memastikan bacaan ini: pada rancangan yang diberikan Work Owner, rincian
          "Reserve OR ASM per COB" memuat puluhan juta rupiah. Porsi treaty OR tidak mungkin
          sebesar itu, sehingga yang dimaksud memang reserve retensi sendiri.
        */}
        <Tile
          value={formatRupiah(data.total_estimasi)}
          label="Reserve OR ASM tertahan"
          note="Dijumlahkan dari kolom Reserve Claim ASM Share di bawah"
        />
        <Tile value={String(data.total_berkas)} label="Berkas outstanding" />
        <Tile
          value={String(data.umur_di_atas_2_tahun)}
          label="Umur di atas 2 tahun"
          alert={data.umur_di_atas_2_tahun > 0}
        />
      </div>

      <AgeBar buckets={data.sebaran_umur} total={data.total_berkas} />

      <div className="grid gap-4 lg:grid-cols-2">
        <GroupTable title="Reserve OR ASM per COB" heading="COB" rows={data.per_cob} />
        <SourceList rows={data.per_sumber_bisnis} />
      </div>
    </section>
  )
}

/** Tile menggambar satu kartu angka. */
function Tile({
  value,
  label,
  note,
  alert = false,
  muted = false,
}: {
  value: string
  label: string
  note?: string
  alert?: boolean
  muted?: boolean
}) {
  return (
    <div className="rounded-kartu border border-slate-200 bg-white p-4">
      <p
        className={[
          'text-2xl font-semibold tabular-nums',
          alert ? 'text-red-700' : muted ? 'text-slate-400' : 'text-slate-900',
        ].join(' ')}
      >
        {value}
      </p>
      <p className="mt-1 text-sm text-slate-600">{label}</p>
      {note ? <p className="mt-1 text-xs text-slate-500">{note}</p> : null}
    </div>
  )
}

/**
 * AgeBar menggambar sebaran umur sebagai satu batang bersegmen, lalu keterangannya.
 *
 * Keempat pita SELALU muncul di keterangan, termasuk yang kosong. Batang yang pitanya hilang
 * akan berubah bentuk dari hari ke hari, dan pembacanya tidak dapat tahu apakah pita itu nol
 * atau tidak ada.
 *
 * Lebar segmen memakai persentase JUMLAH BERKAS, bukan nilai uang — satu klaim bernilai besar
 * tidak boleh membuat pita yang berisi satu berkas tampak menguasai seluruh batang.
 */
function AgeBar({ buckets, total }: { buckets: AgeBucket[]; total: number }) {
  const warna = ['bg-emerald-700', 'bg-lime-600', 'bg-amber-500', 'bg-red-700']
  const lewatDuaTahun = buckets[3]?.berkas ?? 0
  const persenTua = total > 0 ? Math.round((lewatDuaTahun / total) * 100) : 0

  return (
    <div className="rounded-kartu border border-slate-200 bg-white p-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="text-sm font-semibold text-slate-900">
          Umur berkas sejak registrasi
        </h3>
        <p className="text-xs text-slate-500">
          {persenTua}% berkas sudah lewat 2 tahun
        </p>
      </div>

      {total > 0 ? (
        <div className="mt-3 flex h-9 overflow-hidden rounded-md" role="presentation">
          {buckets.map((b, i) =>
            b.berkas === 0 ? null : (
              <div
                key={b.label}
                className={`flex items-center justify-center ${warna[i]}`}
                style={{ width: `${(b.berkas / total) * 100}%` }}
                title={`${b.label}: ${b.berkas} berkas`}
              >
                <span className="px-1 text-xs font-medium text-white tabular-nums">
                  {b.berkas}
                </span>
              </div>
            ),
          )}
        </div>
      ) : (
        <p className="mt-3 text-sm text-slate-500">
          Tidak ada klaim berjalan di cabang ini.
        </p>
      )}

      {/*
        Keterangannya `text-sm`, bukan `text-xs`. Keempat pita muat satu baris dengan sisa
        ruang, dan ukuran terkecil hanya masuk akal ketika ruangnya memang sempit — di sini
        ia justru membuat angka yang paling sering dibaca penyelia menjadi yang terkecil di
        layar.
      */}
      <ul className="mt-3 flex flex-wrap gap-x-6 gap-y-1.5 text-sm text-slate-600">
        {buckets.map((b, i) => (
          <li key={b.label} className="flex items-center gap-1.5">
            <span className={`h-3 w-3 rounded-sm ${warna[i]}`} aria-hidden="true" />
            <span>{b.label}</span>
            <span className="font-medium text-slate-900">{b.berkas} berkas</span>
            <span className="text-slate-400">·</span>
            <span>{formatRupiah(b.nilai)}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

/**
 * SourceList menggambar rincian per sumber bisnis sebagai DAFTAR, bukan tabel.
 *
 * Bentuknya berbeda dari rincian per COB dengan sengaja, dan bukan semata demi variasi:
 * judul kolom "Sumbis" pada tabel akan bertabrakan dengan judul kolom grid di bawahnya,
 * sehingga pembaca layar membacakan dua "Sumbis" yang menunjuk hal berbeda. Daftar tidak
 * punya judul kolom, sehingga persoalan itu hilang.
 */
function SourceList({ rows }: { rows: SummaryGroup[] }) {
  const terbesar = rows.reduce((max, r) => Math.max(max, Number(r.nilai) || 0), 0)

  return (
    <div className="rounded-kartu border border-slate-200 bg-white p-4">
      <h3 className="text-sm font-semibold text-slate-900">Sumber bisnis</h3>

      {rows.length === 0 ? (
        <p className="mt-3 text-sm text-slate-500">Belum ada data.</p>
      ) : (
        <ul className="mt-3 space-y-3">
          {rows.map((r) => (
            <li key={r.nama}>
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="text-sm text-slate-900">{r.nama}</span>
                <span className="text-xs text-slate-600 tabular-nums">
                  {r.berkas} berkas · {formatRupiah(r.nilai)}
                </span>
              </div>
              <div className="mt-1 h-2 rounded-sm bg-slate-100">
                <div
                  className="h-2 rounded-sm bg-emerald-700"
                  style={{
                    width:
                      terbesar > 0
                        ? `${((Number(r.nilai) || 0) / terbesar) * 100}%`
                        : '0%',
                  }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

/**
 * GroupTable menggambar satu daftar rincian beserta batang pembandingnya.
 *
 * Batangnya relatif terhadap baris TERBESAR, bukan terhadap total. Relatif terhadap total,
 * daftar dengan sepuluh baris seimbang akan tampak seperti sepuluh batang yang sama-sama
 * pendek, dan perbandingan antarbarisnya hilang.
 */
function GroupTable({
  title,
  heading,
  rows,
}: {
  title: string
  heading: string
  rows: SummaryGroup[]
}) {
  const terbesar = rows.reduce((max, r) => Math.max(max, Number(r.nilai) || 0), 0)

  return (
    <div className="rounded-kartu border border-slate-200 bg-white p-4">
      <h3 className="text-sm font-semibold text-slate-900">{title}</h3>

      {rows.length === 0 ? (
        <p className="mt-3 text-sm text-slate-500">Belum ada data.</p>
      ) : (
        <table className="mt-3 w-full text-sm">
          <thead>
            <tr className="text-left text-xs uppercase tracking-wide text-slate-500">
              <th className="pb-1 font-medium">{heading}</th>
              <th className="pb-1 text-right font-medium">Berkas</th>
              <th className="pb-1 text-right font-medium">Reserve OR ASM</th>
              <th className="pb-1 text-right font-medium" title="Umur di atas 2 tahun">
                &gt;2 thn
              </th>
              <th className="pb-1" />
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.nama} className="border-t border-slate-100">
                <td className="py-1.5 text-slate-900">{r.nama}</td>
                <td className="py-1.5 text-right tabular-nums text-slate-700">
                  {r.berkas}
                </td>
                <td className="py-1.5 text-right tabular-nums text-slate-900">
                  {formatRupiah(r.nilai)}
                </td>
                <td className="py-1.5 text-right tabular-nums">
                  {r.umur_di_atas_2_tahun > 0 ? (
                    <span className="rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700">
                      {r.umur_di_atas_2_tahun}
                    </span>
                  ) : (
                    <span className="text-slate-400">—</span>
                  )}
                </td>
                <td className="w-2/5 py-1.5 pl-3">
                  <div className="h-2 rounded-sm bg-slate-100">
                    <div
                      className="h-2 rounded-sm bg-emerald-700"
                      style={{
                        width:
                          terbesar > 0
                            ? `${((Number(r.nilai) || 0) / terbesar) * 100}%`
                            : '0%',
                      }}
                    />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
