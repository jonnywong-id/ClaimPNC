import type {
  KomiteAdjustmentLine,
  KomiteAttachment,
  KomiteCase,
  KomiteCoMemberRow,
  KomiteFacOffer,
  KomiteSpreadingRow,
  KomiteTransferDetail,
} from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { formatRupiah } from '@/lib/money'

import { formatPersen, formatTanggal, formatWaktu } from './format'

/**
 * Kedua kolom tab "Claim Detail" — tata letak `Section/ShowTransferDetail` bagian atas.
 *
 * Label ditulis PERSIS seperti di Pega (huruf besar, "S/D", "CO MEMBER") supaya anggota
 * komite yang berpindah dari layar lama langsung mengenali tempat setiap angka (`D-13`).
 *
 * Kolom kiri: kepala klaim dan polis. Kolom kanan: LEADER, CO MEMBER, List Spreading,
 * List Reas Fac-Out, dan Dominan Factor — di Pega dihitung
 * `CalculatedSpredingForClaimKomite`; di sini dihitung server dengan rumus yang sama
 * (lihat `internal/komite/committeesheet.go`), layar hanya menggambarnya.
 *
 * Nilai kasus (`kasus`) menjadi cadangan untuk kepala: case yang klaimnya tidak terbaca
 * tetap menunjukkan nomor polis dan tertanggung dari daftar inbox.
 */
export function ClaimSheet({
  kasus,
  transfer,
}: Readonly<{
  kasus: KomiteCase
  transfer?: KomiteTransferDetail | undefined
}>) {
  return (
    <section
      aria-label="Claim Detail"
      className="grid gap-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-sm lg:grid-cols-2"
    >
      <LeftColumn kasus={kasus} transfer={transfer} />
      <RightColumn transfer={transfer} />
    </section>
  )
}

type Row = { label: string; value: string; hint?: string }

function SheetRows({ rows }: Readonly<{ rows: Row[] }>) {
  return (
    <dl className="divide-y divide-slate-100">
      {rows.map((row) => (
        <div key={row.label} className="grid grid-cols-[11rem_1fr] gap-3 py-1.5">
          <dt className="text-xs font-semibold tracking-wide text-slate-500">{row.label}</dt>
          <dd className="min-w-0 break-words text-sm text-slate-900">
            {row.value}
            {row.hint && <span className="mt-0.5 block text-xs text-slate-500">{row.hint}</span>}
          </dd>
        </div>
      ))}
    </dl>
  )
}

function LeftColumn({
  kasus,
  transfer,
}: Readonly<{
  kasus: KomiteCase
  transfer?: KomiteTransferDetail | undefined
}>) {
  const klaim = transfer?.klaim ?? undefined
  const coverage = transfer?.coverage ?? []
  const travel = klaim?.group_panel === '005'
  const kelas = klaim?.kelas_asuransi || kasus.nama_bisnis || '—'

  const periode = transfer?.polis
    ? `${formatTanggal(transfer.polis.mulai)} S/D ${formatTanggal(transfer.polis.berakhir)}`
    : '—'

  // `.Survey.CouseOfLos` bernilai tunggal di Pega; satu klaim di sini dapat punya beberapa
  // coverage, sehingga sebab kerugiannya dirangkai — tanpa pengulangan.
  const sebab = Array.from(
    new Set(coverage.map((c) => c.sebab_kerugian?.trim()).filter((v): v is string => !!v)),
  ).join(', ')

  const rows: Row[] = [
    {
      label: 'CREATE COMITEE DATE',
      value: formatWaktu(transfer?.komite?.tanggal_dibuat || kasus.tanggal_input),
    },
    {
      label: 'STATUS PREMI',
      value: '—',
      hint:
        'Tidak tersimpan di basis data: Pega membacanya langsung dari layanan premi ' +
        '(GetPremiumPaymentStatus) saat case dibuka.',
    },
    { label: 'INSURED', value: klaim?.tertanggung || kasus.nama_tertanggung || '—' },
    { label: 'CLASS OF INSURANCE', value: kelas },
    { label: 'POLICY No.', value: klaim?.nomor_polis || kasus.nomor_polis || '—' },
    { label: 'REGISTER DATE', value: formatTanggal(klaim?.tanggal_register) },
    { label: 'PERIOD OF INSURANCE', value: periode },
    { label: 'LOCATION', value: klaim?.lokasi || '—' },
    { label: 'BRANCH', value: klaim?.cabang || kasus.cabang || '—' },
    { label: 'SOURCE OF BUSINESS', value: klaim?.sumber_bisnis || kasus.sumber_bisnis || '—' },
    { label: 'INTEREST INSURED', value: kelas },
    { label: 'CLAIM No.', value: klaim?.nomor_klaim || kasus.nomor_klaim || '—' },
    {
      label: travel ? 'DATE OF EVENT' : 'DATE OF ACCIDENT',
      value: formatTanggal(klaim?.tanggal_kejadian),
    },
  ]
  if (!travel) rows.push({ label: 'NATURE OF LOSS', value: sebab || '—' })

  const reserves: Row[] = [
    { label: travel ? 'CURRENCY' : 'RESERVES', value: klaim?.kode_mata_uang || '—' },
  ]
  if (klaim?.ex_gratia) reserves.push({ label: 'EX GRATIA', value: klaim.ex_gratia })

  return (
    <div className="min-w-0 space-y-4">
      <SheetRows rows={rows} />
      {!travel && coverage.length > 0 && (
        <div>
          <p className="mb-1 text-xs font-semibold tracking-wide text-slate-500">Coverage</p>
          <ul className="space-y-1 rounded-kartu border border-slate-200 bg-slate-50/60 p-3">
            {coverage.map((c, i) => (
              <li
                key={`${c.id_objek}-${c.id_coverage}-${i}`}
                className="flex flex-wrap justify-between gap-2 text-sm text-slate-900"
              >
                <span>{c.nama_coverage || 'Coverage tanpa nama'}</span>
                <span className="tabular-nums text-slate-600">
                  TSI {formatRupiah(c.nilai_tsi, { withoutSymbol: true })}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
      <SheetRows rows={reserves} />
    </div>
  )
}

function RightColumn({ transfer }: Readonly<{ transfer?: KomiteTransferDetail | undefined }>) {
  const leader = transfer?.leader
  const baris = transfer?.baris ?? []
  const penuh = transfer?.spreading_lengkap ?? false
  const dominan = transfer?.faktor_dominan ?? []

  return (
    <div className="min-w-0 space-y-4">
      <SheetRows
        rows={[
          {
            label: 'LEADER',
            value: leader ? `${leader.nama}   ${formatPersen(leader.persen)}` : '—',
          },
        ]}
      />

      {baris.length === 0 ? (
        <p className="rounded-kartu border border-dashed border-slate-300 p-3 text-sm text-slate-600">
          CO MEMBER dan List Spreading dihitung dari baris adjustment, dan case ini belum
          punya barisnya.
        </p>
      ) : (
        baris.map((line, i) => (
          <LineShares
            key={`${line.id_objek}-${line.id_coverage}-${i}`}
            line={line}
            penuh={penuh}
            facOut={transfer?.fac_out ?? []}
            berlabel={baris.length > 1}
          />
        ))
      )}

      <div>
        <p className="mb-1 text-xs font-semibold tracking-wide text-slate-500">Dominan Factor</p>
        {dominan.length === 0 ? (
          <p className="text-sm text-slate-600">—</p>
        ) : (
          <ul className="list-inside list-disc text-sm text-slate-900">
            {dominan.map((d) => (
              <li key={d}>{d}</li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function nilai(v: string | undefined): string {
  return v ? formatRupiah(v, { withoutSymbol: true }) : '—'
}

const coMemberColumns: Column<KomiteCoMemberRow>[] = [
  { key: 'nama', title: 'Asuransi', value: (r) => r.nama },
  { key: 'mata_uang', title: 'Currency', value: (r) => r.mata_uang ?? '' },
  { key: 'persen', title: 'Share (%)', value: (r) => r.persen ?? '', alignRight: true },
  { key: 'nilai', title: 'Result Value', value: (r) => nilai(r.nilai), alignRight: true },
]

const spreadingFull: Column<KomiteSpreadingRow>[] = [
  { key: 'treaty', title: 'Spreading', value: (r) => r.nama_treaty || r.jenis_treaty || '' },
  { key: 'mata_uang', title: 'Currency', value: (r) => r.mata_uang ?? '' },
  { key: 'persen', title: 'Share (%)', value: (r) => r.persen ?? '', alignRight: true },
  { key: 'nilai', title: 'Result Value', value: (r) => nilai(r.nilai), alignRight: true },
]

const spreadingShort: Column<KomiteSpreadingRow>[] = [spreadingFull[0]!, spreadingFull[2]!]

/**
 * Bagian kanan satu baris adjustment: CO MEMBER, List Spreading, dan List Reas Fac-Out.
 *
 * Bentuk spreading mengikuti `spreading_lengkap` dari server — `ShowTransferDetail`
 * memilihnya dengan `tempCvg.ObjectSurveyor == 'ASURANSI SINAR MAS'`, dan Fac-Out hanya
 * muncul pada bentuk empat kolom.
 */
function LineShares({
  line,
  penuh,
  facOut,
  berlabel,
}: Readonly<{
  line: KomiteAdjustmentLine
  penuh: boolean
  facOut: KomiteFacOffer[]
  berlabel: boolean
}>) {
  const coMember = line.co_member ?? []
  const spreading = line.spreading ?? []

  const facColumns: Column<KomiteFacOffer>[] = [
    { key: 'reas', title: 'Reins FAC OUT', value: (r) => r.reasuradur },
    { key: 'mata_uang', title: 'Currency', value: () => line.kode_mata_uang ?? '' },
    { key: 'persen', title: 'Spreding (%)', value: (r) => r.persen_polis ?? '', alignRight: true },
    { key: 'nilai', title: 'Result Value', value: () => '—', alignRight: true },
  ]

  return (
    <div className="space-y-3">
      {berlabel && (
        <p className="font-mono text-xs text-slate-500">
          Objek {line.id_objek || '—'} · Coverage {line.id_coverage || '—'}
        </p>
      )}

      {coMember.length > 0 && (
        <DataTable
          title="CO MEMBER"
          columns={coMemberColumns}
          rows={coMember}
          rowKey={(r) => `${r.nama}-${r.persen}`}
          hideSearch
          searchable={false}
        />
      )}

      <DataTable
        title="List Spreading"
        description={`Result Value = nilai komite ${nilai(line.nilai_komite)} × Share / 100`}
        columns={penuh ? spreadingFull : spreadingShort}
        rows={spreading}
        rowKey={(r) => `${r.jenis_treaty}-${r.persen}`}
        emptyMessage="Coverage ini belum punya baris spreading."
        showHeaderWhenEmpty
        hideSearch
        searchable={false}
      />

      {penuh && (
        <DataTable
          title="List Reas Fac-Out"
          description={
            'Share yang tampil adalah share pada dokumen polis. Pega menghitung ulang ' +
            'Spreding (%) dan Result Value dari ShareOffered, TSI Sublimit, dan Limit of ' +
            'Liability per coverage — hitungan itu belum dibawa.'
          }
          columns={facColumns}
          rows={facOut}
          rowKey={(r) => `${r.reasuradur}-${r.persen_polis}`}
          emptyMessage="Polis ini tidak punya reasuradur fac-out."
          showHeaderWhenEmpty
          hideSearch
          searchable={false}
        />
      )}
    </div>
  )
}

/**
 * Tab "Policy Detail" — kepala polis yang dibaca modul ini.
 *
 * `ViewPolicyDetail` di Pega menampilkan seluruh snapshot polis (objek, coverage, premi);
 * rincian penuh itu milik modul `B-1` Polis & Snapshot. Yang ada di sini hanya kepala
 * polisnya, dan layar menyatakannya.
 */
export function PolicyTab({
  kasus,
  transfer,
}: Readonly<{
  kasus: KomiteCase
  transfer?: KomiteTransferDetail | undefined
}>) {
  const klaim = transfer?.klaim ?? undefined
  const leader = transfer?.leader
  const rows: Row[] = [
    { label: 'POLICY No.', value: klaim?.nomor_polis || kasus.nomor_polis || '—' },
    { label: 'INSURED', value: klaim?.tertanggung || kasus.nama_tertanggung || '—' },
    { label: 'CLASS OF INSURANCE', value: klaim?.kelas_asuransi || kasus.nama_bisnis || '—' },
    {
      label: 'PERIOD OF INSURANCE',
      value: transfer?.polis
        ? `${formatTanggal(transfer.polis.mulai)} S/D ${formatTanggal(transfer.polis.berakhir)}`
        : '—',
    },
    { label: 'BRANCH', value: klaim?.cabang || kasus.cabang || '—' },
    { label: 'SOURCE OF BUSINESS', value: klaim?.sumber_bisnis || kasus.sumber_bisnis || '—' },
    { label: 'GROUP PANEL', value: klaim?.group_panel || '—' },
    { label: 'POSISI KOASURANSI', value: klaim?.peran_koasuransi || '—' },
    { label: 'LEADER', value: leader ? `${leader.nama}   ${formatPersen(leader.persen)}` : '—' },
    { label: 'SHARE ASM', value: formatPersen(klaim?.persen_asm_share) },
  ]

  return (
    <section
      aria-label="Policy Detail"
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-sm"
    >
      <SheetRows rows={rows} />
      <p className="mt-4 text-xs leading-relaxed text-slate-500">
        Rincian penuh polis — daftar objek, coverage, dan premi pada <code>ViewPolicyDetail</code>{' '}
        — milik modul B-1 Polis &amp; Snapshot dan belum ditampilkan di sini.
      </p>
    </section>
  )
}

const attachmentColumns: Column<KomiteAttachment>[] = [
  { key: 'nama', title: 'Nama berkas', value: (r) => r.nama ?? '' },
  { key: 'catatan', title: 'Catatan', value: (r) => r.catatan ?? '' },
  { key: 'kategori', title: 'Kategori', value: (r) => r.kategori ?? '' },
  { key: 'oleh', title: 'Diunggah oleh', value: (r) => r.diunggah_oleh ?? '' },
  { key: 'pada', title: 'Tanggal', value: (r) => formatWaktu(r.diunggah_pada) },
]

/**
 * Tab "Lampiran Dokumen" — lampiran klaim yang dikomitekan (`POOLDATA.DATA_ATTACHFILE`).
 *
 * `UploadDocumentKomite` di Pega juga menerima unggahan KE case komite; jalur itu belum
 * ada, dan layar menyatakannya alih-alih menampilkan tombol yang tidak bekerja.
 */
export function AttachmentTab({ transfer }: Readonly<{ transfer?: KomiteTransferDetail | undefined }>) {
  return (
    <div className="space-y-3">
      <DataTable
        title="Lampiran Dokumen"
        description="Berkas yang sudah diunggah untuk klaim ini."
        columns={attachmentColumns}
        rows={transfer?.lampiran ?? []}
        rowKey={(r) => r.id}
        emptyMessage="Belum ada lampiran untuk klaim ini."
        showHeaderWhenEmpty
        hideSearch
        searchable={false}
      />
      <p className="text-xs leading-relaxed text-slate-500">
        Mengunggah dokumen ke case komite (<code>UploadDocumentKomite</code>) belum tersedia.
        Dokumen klaim diunggah dari tab Unggah Dokumen pada layar klaimnya.
      </p>
    </div>
  )
}
