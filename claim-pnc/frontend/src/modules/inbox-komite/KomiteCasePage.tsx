import { useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'

import { APIError, NetworkError } from '@/api/client'
import {
  KomiteOutcome,
  type KomiteAdjustmentLine,
  type KomiteCase,
  type KomiteTransferDetail,
} from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { ReloadIcon, ScaleIcon } from '@/components/Icon'
import { TabBar } from '@/components/TabBar'
import { formatRupiah } from '@/lib/money'

import { useKomiteCase } from './api'
import { AttachmentTab, ClaimSheet, PolicyTab } from './ClaimSheet'
import { CommitteeBottom } from './CommitteeBottom'
import { KmtnDecisionForm } from './KmtnDecisionForm'
import { formatTanggal } from './format'

/** Ketiga tab `Section/ShowTransfer` — Claim Detail, Policy Detail, Lampiran Dokumen. */
const TABS = [
  { kode: 'klaim', nama: 'Claim Detail' },
  { kode: 'polis', nama: 'Policy Detail' },
  { kode: 'lampiran', nama: 'Lampiran Dokumen' },
]

/**
 * Layar rincian kasus komite — "Lihat Detail Transfer".
 *
 * # Apa yang ia gantikan
 *
 * Tiga hal yang di sistem lama berurutan, dan di sini menjadi satu layar:
 *
 * | Pega | Di sini |
 * |---|---|
 * | `Activity/SetAssignmentKomite-Act.xml` | permintaan rincian ke server |
 * | `OBJ-OPEN-BY-HANDLE` atas `"ASSIGN-WORKLIST " + inskey + "!Komite_Flow"` | pemeriksaan kepemilikan di server |
 * | Flow action `ViewTransferDtl` → section `ShowTransfer` | halaman ini |
 *
 * Judulnya — "CLAIM COMMITTEE" dan "CLAIM No." — diambil dari `Section/ShowTransfer`
 * apa adanya, supaya anggota komite yang berpindah dari layar lama mengenali tempatnya.
 *
 * "CLAIM No." di layar lama terikat ke `.CoverID`, dan properti itu ternyata memang NOMOR
 * KLAIM: `InsertUpdateKomiteList` menyalinnya ke `TempInput.NoClaim`, yang menjadi kolom
 * `T_CLAIM_KOMITE_LIST.NO_KLAIM`. Diukur, keduanya sama pada 159.467 dari 159.468 baris.
 *
 * # Isi ShowTransfer, dan status masing-masing bagian
 *
 * Ditelusuri 2026-09-29. Section itu bukan sekadar bungkus: ia yang menentukan JENIS komite
 * lewat tujuh sel label bersyarat atas `.TransferType` dan `.Adjustment.PaymentType`.
 *
 * | Bagian | Status |
 * |---|---|
 * | judul bersyarat (7 label) | **dibangun** — dirakit server, lihat `TransferDetail.Judul` |
 * | `ShowTransferDetail` kolom kiri & kanan | **dibangun** — ClaimSheet (tab Claim Detail) |
 * | `ShowTransferDetail` rincian transfer | **dibangun** — TransferSection di bawah |
 * | `ShowTransferDetailHE` | belum — lini HE belum ditangani modul ini |
 * | blok surveyor | belum — kolomnya ada tetapi kosong pada seluruh case komite |
 * | `UploadDocumentKomite` | **sebagian** — tab Lampiran Dokumen menampilkan lampiran klaim; unggah ke case komite belum |
 * | `ViewPolicyDetail` | **sebagian** — tab Policy Detail menampilkan kepala polis; rincian penuh milik `B-1` |
 *
 * Tiga bagian ShowTransfer SENGAJA tidak dibangun karena memang tidak berlaku lagi:
 * `.AcceptStatus`, `.RadApprove`, `.RejectedCode`, dan `.Comment` adalah form keputusan,
 * dan Work Owner mencabut tombol keputusan. Satu di antaranya bahkan mati di Pega sendiri —
 * sel `.AcceptStatus` bersyarat `1==2`, yang tidak pernah benar.
 *
 * Yang belum dibangun tetap dinyatakan di layar, bukan disembunyikan dengan tata letak yang
 * tampak lengkap. Halaman yang terlihat utuh padahal isinya belum ada adalah cara paling
 * cepat membuat orang mengira modulnya selesai.
 *
 * # Keputusan hanya untuk case KMTN
 *
 * Work Owner mencabut tombol keputusan 2026-09-29, lalu pada hari yang sama membukanya
 * kembali KHUSUS case KMTN (form ShowTransfer, lihat KmtnDecisionForm dan
 * `docs/keputusan-implementasi.md` §113). Case Pega tetap tanpa tombol keputusan, dan jalur
 * tulis modul komite sendiri (`/api/komite/inbox/{nomor}/keputusan`, §68 dan §69) tetap
 * tertutup.
 */
export function KomiteCasePage() {
  const params = useParams<{ nomor: string }>()
  const caseID = params.nomor ?? ''
  const detail = useKomiteCase(caseID)
  const [tab, setTab] = useState('klaim')

  const found = detail.data?.kasus
  const transfer = detail.data?.transfer

  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Proses Klaim</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li>
              <Link to="/komite/inbox" className="hover:text-slate-700 hover:underline">
                Inbox Komite
              </Link>
            </li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Detail Transfer</li>
          </ol>
        </nav>

        {/*
          Judulnya datang dari server sebagai satu teks jadi.

          `Section/ShowTransfer` merakitnya dari tujuh sel label bersyarat, dan salah satu
          syaratnya — `IsTravel` — MENYEMBUNYIKAN akhiran "- ADJUSTMENT" pada Group Panel
          005. Merakitnya di sini berarti menyalin enam syarat Pega ke React.

          Sebelum rinciannya tiba, "CLAIM COMMITTEE" telanjang yang ditampilkan: itu pula
          yang Pega tampilkan ketika tidak satu pun syarat terpenuhi.
        */}
        <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight text-slate-900">
          <ScaleIcon className="h-6 w-6 text-slate-400" />
          {detail.data?.transfer?.judul ?? 'CLAIM COMMITTEE'}
        </h1>
        <p className="mt-2 font-mono text-sm text-slate-600">
          Claim No. {found?.nomor_klaim || caseID}
        </p>
      </header>

      <div className="mb-5 flex flex-wrap gap-2">
        <Button tone="kedua" onClick={() => { detail.refetch() }} disabled={detail.isFetching}>
          <ReloadIcon className={`h-4 w-4 ${detail.isFetching ? 'animate-spin' : ''}`} />
          {detail.isFetching ? 'Memuat…' : 'Muat ulang'}
        </Button>
        <Link
          to="/komite/inbox"
          className="inline-flex items-center rounded-kartu border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
        >
          Kembali ke inbox
        </Link>
      </div>

      {detail.isPending && (
        <output className="block text-sm text-slate-600">
          Memuat rincian kasus komite…
        </output>
      )}

      {detail.isError && <LoadError error={detail.error} caseID={caseID} />}

      {found && (
        <>
          <TabBar tabs={TABS} active={tab} onSelect={setTab} label="Bagian kasus komite" />
          <div className="mt-4">
            {tab === 'klaim' && (
              <>
                <ClaimSheet kasus={found} transfer={transfer} />
                <CaseFacts item={found} />
                {transfer?.klaim && <ClaimSection klaim={transfer.klaim} />}
                {(transfer?.coverage.length ?? 0) > 0 && (
                  <CoverageSection coverages={transfer!.coverage} />
                )}
                {transfer && <TransferSection transfer={transfer} />}
                {transfer && <CommitteeBottom transfer={transfer} />}
                <KmtnDecisionForm caseID={caseID} />
                <MissingParts heDapatDinilai={transfer?.he_dapat_dinilai ?? false} />
              </>
            )}
            {tab === 'polis' && <PolicyTab kasus={found} transfer={transfer} />}
            {tab === 'lampiran' && <AttachmentTab transfer={transfer} />}
          </div>
        </>
      )}
    </div>
  )
}

/**
 * FactSection adalah kartu berjudul berisi daftar label–nilai dua kolom, diikuti isi
 * tambahan bagian itu.
 */
function FactSection({
  headingId,
  title,
  rows,
  children,
}: Readonly<{
  headingId: string
  title: string
  rows: { label: string; value: string }[]
  children?: ReactNode
}>) {
  return (
    <section
      aria-labelledby={headingId}
      className="mt-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 id={headingId} className="text-base font-semibold text-slate-900">
        {title}
      </h2>

      <dl className="mt-4 grid gap-x-6 gap-y-3 sm:grid-cols-2">
        {rows.map((row) => (
          <div key={row.label} className="min-w-0 border-b border-slate-100 pb-2">
            <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
              {row.label}
            </dt>
            <dd className="mt-0.5 truncate text-sm text-slate-900">{row.value}</dd>
          </div>
        ))}
      </dl>
      {children}
    </section>
  )
}

/**
 * Medan case dari daftar inbox (`InboxRegisterKomite_RD`) yang TIDAK ada di ClaimSheet.
 *
 * Nomor polis, tertanggung, bisnis, sumber bisnis, dan cabang sudah tergambar di kolom kiri
 * ClaimSheet — dengan nilai daftar ini sebagai cadangannya — sehingga tidak diulang di sini.
 * Menampilkan satu nilai dua kali di layar yang sama membuat orang bertanya mana yang benar.
 */
function CaseFacts({ item }: Readonly<{ item: KomiteCase }>) {
  const rows: { label: string; value: string }[] = [
    { label: 'Nomor case', value: item.nomor_case },
    { label: 'Tgl komite', value: formatTanggal(item.tanggal_komite) },
    { label: 'Tgl input', value: formatTanggal(item.tanggal_input) },
    { label: 'Aging komite', value: `${item.aging_komite} hari` },
    { label: 'Status kerja', value: item.status_kerja || '—' },
  ]

  return (
    <FactSection headingId="judul-fakta-komite" title="Data kasus" rows={rows}>
      {/*
        Keputusan yang tercatat DI PEGA ditampilkan terpisah dari data kasusnya, dan diberi
        nama yang menyebut asalnya. Selama masa paralel, keputusan Pega dan keputusan sistem
        ini dapat berbeda — dan perbedaan itu harus terbaca, bukan diselesaikan diam-diam
        dengan memilih salah satu.
      */}
      <div className="mt-5 rounded-kartu border border-slate-200 bg-slate-50 p-3">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
          Keputusan tercatat di Pega
        </p>
        <p className="mt-1 text-sm text-slate-900">
          {item.keputusan_pega
            ? outcomeLabel(item.keputusan_pega)
            : 'Belum ada keputusan yang tercatat.'}
        </p>
      </div>
    </FactSection>
  )
}

/**
 * Bagian yang belum dapat dibangun, disebut satu per satu beserta sebabnya.
 *
 * Disebut per sub-section, bukan sebagai satu kalimat "sebagian belum ada". Nama rule-nya
 * yang membuat kalimat ini dapat ditindaklanjuti: yang meminta ke Tim Pega tahu persis apa
 * yang diminta, dan yang membacanya kelak tahu kapan bagian ini boleh dihapus.
 */
function MissingParts({ heDapatDinilai }: Readonly<{ heDapatDinilai: boolean }>) {
  const parts: { name: string; reason: string }[] = [
    {
      name: 'ShowTransferDetailHE',
      reason: heDapatDinilai
        ? 'varian Heavy Equipment — rule-nya ada, tetapi lini HE belum ditangani modul ini'
        : 'varian Heavy Equipment — rule-nya ada, dan syaratnya (IsHE) belum dapat dinilai ' +
          'sama sekali: kolom BUSINESSTYPE kosong pada seluruh case komite',
    },
    {
      name: 'Blok surveyor',
      reason:
        'ShowTransfer menampilkannya pada komite jenis Survey — kolom SURVEYORNAME_1, ' +
        'SURVEYORTYPE_1, dan SURVEYDATE_1 ada, tetapi terisi 0 dari 610 case komite: ' +
        'datanya hidup di case klaim, bukan di case komite',
    },
    {
      name: 'UploadDocumentKomite',
      reason:
        'unggah ke case komite — tab Lampiran Dokumen menampilkan lampiran klaimnya, ' +
        'tetapi kunci PC_LINK_ATTACHMENT ke case komite belum ditemukan',
    },
    {
      name: 'ViewPolicyDetail',
      reason:
        'rincian penuh polis — tab Policy Detail menampilkan kepala polis; objek, coverage, ' +
        'dan premi milik modul B-1 Polis & Snapshot',
    },
    {
      name: 'STATUS PREMI · Detail Premi',
      reason:
        'Pega membacanya dari layanan premi (GetPremiumPaymentStatus, Connect-REST) dan ' +
        'tidak menyimpannya di basis data',
    },
    {
      name: 'Hitungan Fac-Out',
      reason:
        'Spreding (%) dan Result Value Fac-Out diturunkan dari ShareOffered, TSI Sublimit, dan ' +
        'Limit of Liability per coverage polis dengan syarat yang tidak terbaca di export',
    },
  ]

  return (
    <section
      aria-labelledby="judul-bagian-belum-ada"
      className="mt-5 rounded-kartu border border-slate-200 bg-white p-5"
    >
      <h2 id="judul-bagian-belum-ada" className="text-base font-semibold text-slate-900">
        Bagian yang belum dibangun
      </h2>
      <p className="mt-1 text-sm leading-relaxed text-slate-600">
        Layar lama merakit beberapa bagian di bawah satu flow action{' '}
        <code>ViewTransferDtl</code>. Rincian transfernya sudah ada di atas; sisanya belum, dan
        sebabnya berbeda-beda:
      </p>
      <ul className="mt-3 space-y-2">
        {parts.map((part) => (
          <li key={part.name} className="text-sm text-slate-700">
            <span className="font-mono text-xs text-slate-900">{part.name}</span>
            <span className="text-slate-500"> — {part.reason}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}

/**
 * Galat dibedakan, karena tindak lanjutnya berbeda.
 *
 * `404` di sini berarti dua hal yang SENGAJA disamakan server: kasusnya tidak ada, atau ia
 * bukan milik pemanggil. Membedakannya di peramban akan mengubah alamat halaman ini menjadi
 * alat untuk menebak nomor case — lihat catatan pada `komite.ErrNotAssigned`.
 */
function LoadError({ error, caseID }: Readonly<{ error: unknown; caseID: string }>) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Rincian kasus komite belum dapat dimuat. Periksa koneksi lalu tekan Muat ulang."
        tone="gangguan"
      />
    )
  }
  if (error instanceof APIError && error.status === 404) {
    return (
      <ErrorMessage
        title="Kasus itu tidak ada di inbox Anda"
        description={`Nomor ${caseID} tidak ditemukan, atau ia bukan pekerjaan Anda. Kembali ke inbox lalu pilih dari daftar.`}
        tone="penolakan"
      />
    )
  }
  return (
    <ErrorMessage
      title="Rincian kasus gagal dimuat"
      description={error instanceof APIError ? error.message : 'Terjadi kesalahan pada sistem.'}
      tone="gangguan"
    />
  )
}

function outcomeLabel(outcome: KomiteCase['keputusan_pega']): string {
  switch (outcome) {
    case KomiteOutcome.approved:
      return 'Diterima'
    case KomiteOutcome.rejected:
      return 'Ditolak'
    case KomiteOutcome.returned:
      return 'Dikembalikan'
    default:
      return 'Menunggu'
  }
}

/**
 * Rincian "Lihat Detail Transfer" — pengganti `Section/ShowTransferDetail`.
 *
 * # Kenapa nilai uang muncul DI SINI padahal daftarnya tanpa nilai uang
 *
 * Bukan pertentangan, melainkan dua layar yang berbeda. `InboxRegisterKomite_RD` — sumber
 * daftar — memang tidak memuat satu pun nilai uang, dan `§69` mengikutinya. Yang memuatnya
 * adalah `ShowTransferDetail`, yang di sistem lama pun hanya tergambar setelah sebuah case
 * ditekan.
 *
 * # Ketiadaan ditampilkan sebagai KETERANGAN, bukan sebagai nol rupiah
 *
 * Dari 189 case yang dapat muncul di inbox, 41 punya baris adjustment. Sisanya belum sampai
 * ke tahap itu, atau berjenis komite yang memang tidak memilikinya.
 *
 * Menggambar "Rp 0" untuk keadaan itu adalah kesalahan yang paling mahal di layar ini: nol
 * yang tidak dapat dibedakan dari "belum ada" terbaca sebagai angka yang sudah diputuskan.
 */
function TransferSection({ transfer }: Readonly<{ transfer: KomiteTransferDetail }>) {
  return (
    <section
      aria-labelledby="judul-detail-transfer"
      className="mt-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 id="judul-detail-transfer" className="text-base font-semibold text-slate-900">
        Detail transfer
      </h2>

      {/*
        Ketiadaan NILAI UANG dinyatakan tersendiri, bukan dengan menyembunyikan seluruh
        blok: 148 dari 189 case punya klaim dan coverage lengkap tanpa satu pun baris
        adjustment. Menyamakan keduanya membuat mayoritas layar tampak kosong padahal
        datanya ada.
      */}
      {transfer.nilai_uang_kosong ? (
        <p className="mt-2 text-sm leading-relaxed text-slate-600">
          Belum ada baris transfer untuk kasus ini. Baris adjustment lahir pada tahap
          tertentu, dan komite jenis Survey maupun Liable Klaim memang tidak memilikinya —
          jadi ini bukan angka nol, melainkan belum ada angkanya.
        </p>
      ) : (
        <>
          {transfer.komite && <CommitteeRecord record={transfer.komite} />}
          {transfer.baris.length > 0 && (
            <div className="mt-4 space-y-4">
              {transfer.baris.map((line, index) => (
                <AdjustmentCard
                  key={`${line.nomor_klaim}-${line.id_objek}-${line.id_coverage}-${index}`}
                  line={line}
                />
              ))}
            </div>
          )}
        </>
      )}
    </section>
  )
}

/**
 * Data klaim yang dikomitekan — `.KomiteClaimData.*` pada `ShowTransferDetail`.
 *
 * Dibaca dari `POOLDATA.T_CLAIM_PNC` lewat kunci BER-PREFIX. Terbaca pada 189 dari 189
 * case, jadi blok ini nyaris selalu tergambar.
 */
function ClaimSection({ klaim }: Readonly<{ klaim: NonNullable<KomiteTransferDetail['klaim']> }>) {
  // Tanggal kejadian, tanggal register, dan lokasi sudah di kolom kiri ClaimSheet.
  const rows: { label: string; value: string }[] = [
    { label: 'Status klaim', value: klaim.status_klaim || '—' },
    { label: 'Share ASM', value: klaim.persen_asm_share ? `${klaim.persen_asm_share}%` : '—' },
    { label: 'Koasuransi', value: klaim.koasuransi || '—' },
  ]

  return (
    <FactSection headingId="judul-klaim-komite" title="Klaim yang dikomitekan" rows={rows}>
      {/*
        Kronologi dan rekomendasi TIDAK dipotong: keduanya kalimat yang dibaca anggota
        komite saat memutuskan, dan potongan kalimat lebih buruk daripada kalimat panjang.
      */}
      {klaim.kronologi && (
        <div className="mt-4">
          <p className="text-xs font-medium uppercase tracking-wide text-slate-500">Kronologi</p>
          <p className="mt-1 whitespace-pre-line text-sm leading-relaxed text-slate-800">
            {klaim.kronologi}
          </p>
        </div>
      )}

      {klaim.rekomendasi && (
        <div className="mt-4">
          <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Rekomendasi
          </p>
          <p className="mt-1 whitespace-pre-line text-sm leading-relaxed text-slate-800">
            {klaim.rekomendasi}
          </p>
        </div>
      )}
    </FactSection>
  )
}

/**
 * Blok analisis komite — `.Komite.*` pada `ShowTransferDetail`.
 *
 * Isinya kolom `POOLDATA.T_CLAIM_OBJECTCOVERAGE`, bukan tabel kerja dan bukan
 * `T_CLAIM_KOMITE_LIST`; keduanya sudah diperiksa dan tidak memuatnya.
 *
 * Seluruh baris digambar tanpa paginasi: diukur 1 sampai 3 baris per case, tidak satu pun
 * melampaui tiga.
 */
function CoverageSection({
  coverages,
}: Readonly<{
  coverages: KomiteTransferDetail['coverage']
}>) {
  return (
    <section
      aria-labelledby="judul-analisis-komite"
      className="mt-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 id="judul-analisis-komite" className="text-base font-semibold text-slate-900">
        Objek dan analisis komite
      </h2>

      <div className="mt-4 space-y-4">
        {coverages.map((c, index) => (
          <CoverageCard key={`${c.id_objek}-${c.id_coverage}-${index}`} coverage={c} />
        ))}
      </div>
    </section>
  )
}

function CoverageCard({
  coverage,
}: Readonly<{
  coverage: KomiteTransferDetail['coverage'][number]
}>) {
  const analisis: { label: string; value: string }[] = [
    { label: 'Keadaan kerugian', value: coverage.keadaan_kerugian ?? '' },
    { label: 'Luas kerugian', value: coverage.luas_kerugian ?? '' },
    { label: 'Tanggung jawab hukum', value: coverage.tanggung_jawab_hukum ?? '' },
    { label: 'Catatan', value: coverage.catatan ?? '' },
    { label: 'Diagnosa', value: coverage.diagnosa ?? '' },
  ].filter((row) => row.value.trim() !== '')

  return (
    <article className="rounded-kartu border border-slate-200 bg-slate-50/60 p-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="min-w-0 text-sm font-medium text-slate-900">
          {coverage.nama_coverage || 'Coverage tanpa nama'}
        </p>
        <p className="font-mono text-xs text-slate-500">
          Objek {coverage.id_objek || '—'} · Coverage {coverage.id_coverage || '—'}
        </p>
      </div>

      <dl className="mt-3 grid gap-x-6 gap-y-2 sm:grid-cols-2">
        <div className="min-w-0">
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Sebab kerugian
          </dt>
          <dd className="mt-0.5 text-sm text-slate-900">{coverage.sebab_kerugian || '—'}</dd>
        </div>
        <div className="min-w-0">
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
            TSI objek
          </dt>
          <dd className="mt-0.5 text-sm font-medium text-slate-900">
            {formatRupiah(coverage.nilai_tsi)}
          </dd>
        </div>
      </dl>

      {/*
        Blok analisis hanya digambar bila ADA isinya — `analisis_terisi` adalah kesimpulan
        server, bukan tebakan layar. Judul blok yang di bawahnya kosong lebih buruk
        daripada tidak ada blok sama sekali; 53 dari 79 baris berada dalam keadaan itu.
      */}
      {coverage.analisis_terisi && analisis.length > 0 && (
        <div className="mt-3 space-y-3 border-t border-slate-200 pt-3">
          {analisis.map((row) => (
            <div key={row.label}>
              <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
                {row.label}
              </p>
              <p className="mt-1 whitespace-pre-line text-sm leading-relaxed text-slate-800">
                {row.value}
              </p>
            </div>
          ))}
        </div>
      )}
    </article>
  )
}

/** Keputusan komite menurut PEGA — diberi nama yang menyebut asalnya. */
function CommitteeRecord({ record }: Readonly<{ record: NonNullable<KomiteTransferDetail['komite']> }>) {
  const rows: { label: string; value: string }[] = [
    { label: 'Tipe komite', value: record.tipe_komite || '—' },
    { label: 'Anggota komite', value: record.nama_komite || '—' },
    { label: 'Jenjang', value: record.jenjang ? String(record.jenjang) : '—' },
    { label: 'Nilai klaim', value: formatRupiah(record.nilai_klaim) },
    { label: 'Share ASM', value: record.persen_asm_share ? `${record.persen_asm_share}%` : '—' },
    { label: 'Tgl keputusan', value: formatTanggal(record.tanggal_komite) },
  ]

  return (
    <div className="mt-3 rounded-kartu border border-slate-200 bg-slate-50 p-3">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
        Keputusan komite di Pega
      </p>
      <dl className="mt-2 grid gap-x-6 gap-y-2 sm:grid-cols-3">
        {rows.map((row) => (
          <div key={row.label} className="min-w-0">
            <dt className="text-xs text-slate-500">{row.label}</dt>
            <dd className="truncate text-sm text-slate-900">{row.value}</dd>
          </div>
        ))}
      </dl>
      {record.catatan && (
        <p className="mt-2 border-t border-slate-200 pt-2 text-sm text-slate-700">
          {record.catatan}
        </p>
      )}
    </div>
  )
}

/**
 * Satu baris adjustment.
 *
 * Digambar sebagai KARTU, bukan baris tabel. Enam nilai uang berdampingan pada satu baris
 * tabel memaksa pembacanya menghitung kolom untuk tahu angka mana yang sedang ia lihat —
 * dan yang dibaca di sini adalah angka yang akan disetujui.
 */
function AdjustmentCard({ line }: Readonly<{ line: KomiteAdjustmentLine }>) {
  const uang: { label: string; value: string; tegas?: boolean }[] = [
    { label: 'Gross', value: formatRupiah(line.nilai_gross) },
    { label: 'Usulan', value: formatRupiah(line.nilai_usulan) },
    { label: 'Akseptasi', value: formatRupiah(line.nilai_akseptasi), tegas: true },
    { label: 'Salvage', value: formatRupiah(line.nilai_salvage) },
    {
      label: line.persen_asm_share ? `Share ASM (${line.persen_asm_share}%)` : 'Share ASM',
      value: formatRupiah(line.nilai_asm_share),
    },
    { label: 'Risiko sendiri', value: formatRupiah(line.nilai_risiko_sendiri) },
  ]

  return (
    <div className="rounded-kartu border border-slate-200 p-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="font-mono text-sm font-medium text-slate-900">{line.nomor_klaim}</p>
        <p className="text-xs text-slate-500">
          {[
            line.id_objek && `Objek ${line.id_objek}`,
            line.id_coverage && `Coverage ${line.id_coverage}`,
            line.jenis_pembayaran && `Jenis ${line.jenis_pembayaran}`,
          ]
            .filter(Boolean)
            .join(' · ') || '—'}
        </p>
      </div>

      {line.ex_gratia && (
        <p className="mt-1 inline-flex rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-800 ring-1 ring-amber-100">
          Ex-Gratia
        </p>
      )}

      <dl className="mt-3 grid gap-x-6 gap-y-2 sm:grid-cols-3">
        {uang.map((item) => (
          <div key={item.label} className="min-w-0">
            <dt className="text-xs text-slate-500">{item.label}</dt>
            <dd
              className={`truncate tabular-nums ${
                item.tegas ? 'text-sm font-semibold text-slate-900' : 'text-sm text-slate-700'
              }`}
            >
              {item.value}
            </dd>
          </div>
        ))}
      </dl>

      {line.nomor_akseptasi && (
        <p className="mt-2 text-xs text-slate-500">
          Akseptasi {line.nomor_akseptasi} · {formatTanggal(line.tanggal_akseptasi)}
        </p>
      )}
      {line.sebab_kerugian && (
        <p className="mt-2 border-t border-slate-100 pt-2 text-sm leading-relaxed text-slate-700">
          {line.sebab_kerugian}
        </p>
      )}
      {line.catatan && <p className="mt-1 text-sm text-slate-600">{line.catatan}</p>}
    </div>
  )
}
