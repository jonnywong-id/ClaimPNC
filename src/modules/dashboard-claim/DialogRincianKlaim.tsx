import { useEffect, useRef, useState, type ReactElement } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useRincianKlaim } from './api'
import { ambil } from './dokumen'
import { isPA, isHE, penjagaTerpenuhi } from './kondisiLiniBisnis'
import { GridRincian, Isian_, type Isian } from './BagianRincian'
import { SubPopupRincian } from './SubPopupRincian'
import {
  ISIAN_CATATAN,
  ISIAN_KEPALA,
  ISIAN_PENERIMA_DETAIL,
  ISIAN_REGISTER,
  ISIAN_COVERAGE_DETAIL,
  ISIAN_COVERAGE_GRID,
  ISIAN_COVERAGE_HE,
  ISIAN_DETAIL_PROGRESS,
  ISIAN_DETAIL_SURVEYOR,
  ISIAN_HASIL_SURVEY,
  ISIAN_SURVEY,
  KOLOM_COVERAGE_HE,
  KOLOM_OBJEK_HE,
  KOLOM_COVERAGE,
  KOLOM_HASIL_SURVEY,
  KOLOM_DOKUMEN,
  KOLOM_POSISI,
  KOLOM_RIWAYAT_LAMPIRAN,
  KOLOM_PENERIMA,
  KOLOM_INVESTIGASI,
  KOLOM_KOMITE_HE,
  VARIAN_OBJEK_ADJ,
  VARIAN_OBJEK_ESTIMASI,
  VARIAN_OBJEK_REGISTER,
  type VarianObjek,
} from './spesifikasiRincian'

/**
 * Popup rincian klaim — terbuka saat nomor klaim diklik.
 *
 * # Apa yang digantikan
 *
 * `Section/DashboardClaimShow_Sec-Section.xml` memasang kolom nomor klaim sebagai **Link**:
 *
 *	pyAction       showHarness
 *	pyHarnessName  ViewTempDetailClaim
 *	pyActivity     setDataViewKlaim_Act
 *	pyTarget       popup
 *
 * Jadi popup, bukan halaman penuh — dan itu ditiru apa adanya.
 *
 * # Sepuluh bagian, dan baru kepalanya yang dibangun
 *
 * Harness `ViewTempDetailClaim` merakit sepuluh section, seluruhnya ada di export:
 *
 *	ViewInputRegisterDetail · ViewInputEstimasiDetail · ViewShowObjectAdj ·
 *	ViewShowReceiver · ViewHasilSurvey · ViewUploadDocument ·
 *	GCNMViewAttachmentHistory · CatatanToAnalyst_Section · ProgressCloseClaim ·
 *	ShowObjectHEPICTeknis_komite
 *
 * Yang MENGHALANGI isinya bukan data melainkan **label**: Pega mengambil teks label dari
 * Field Value rule, dan export ini **tidak punya folder Field Value sama sekali**. Nama
 * propertinya ada, tetapi nama itu menyesatkan — `.Currency` memuat nomor polis, `.City`
 * memuat nomor klaim (utang teknis §4.2). Menggambar label dari nama properti akan memberi
 * judul "Currency" pada kolom berisi nomor polis.
 *
 * Karena itu bagian-bagiannya dibangun setelah labelnya diketahui, bukan ditebak.
 */
export function DialogRincianKlaim({
  klaimID,
  nomorKlaim,
  onTutup,
}: {
  klaimID: string
  nomorKlaim: string
  onTutup: () => void
}) {
  /*
    Baris yang sedang dibuka sub-popup-nya — padanan `pyEditAction` Pega.

    Satu state untuk ketujuh grid, bukan tujuh: hanya satu sub-popup yang dapat terbuka pada
    satu waktu, dan tujuh state yang harus saling mematikan adalah tujuh kesempatan untuk
    dua popup terbuka bersamaan.
  */
  const [sub, setSub] = useState<{
    judul: string
    baris: Record<string, unknown>
    isian: Isian[]
  } | null>(null)

  const rincian = useRincianKlaim(klaimID)
  const tutupRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    tutupRef.current?.focus()

    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onTutup()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onTutup])

  const judul = `Detail Klaim ${nomorKlaim}`

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-label={judul}
    >
      <div className="max-h-[90vh] w-full max-w-4xl overflow-y-auto rounded-kartu bg-white p-6 shadow-terbang">
        <div className="mb-4 flex items-start justify-between gap-4">
          <h2 className="text-lg font-semibold text-slate-900">{judul}</h2>
          <button
            ref={tutupRef}
            type="button"
            onClick={onTutup}
            aria-label="Tutup"
            className={[
              'rounded-kontrol px-3 py-1.5 text-sm text-slate-500',
              'transition ease-halus hover:bg-slate-100 hover:text-slate-700',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
              'focus-visible:outline-blue-600',
            ].join(' ')}
          >
            ✕
          </button>

        </div>

        {rincian.isPending ? (
          <p className="py-8 text-center text-sm text-slate-500">Memuat rincian klaim…</p>
        ) : null}

        {rincian.isError ? (
          <ErrorMessage
            tone="gangguan"
            title="Rincian klaim tidak dapat dibaca"
            description={
              rincian.error instanceof APIError
                ? rincian.error.message
                : 'Terjadi kesalahan pada sistem.'
            }
          />
        ) : null}

        {rincian.data !== undefined ? (
          <>
            <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">
              <Nilai label="No Klaim" isi={rincian.data.nomor_klaim} />
              <Nilai label="PIC Teknik" isi={rincian.data.pic_teknik} />
              <Nilai label="Admin PNC" isi={rincian.data.admin_pnc} />
              <Nilai label="Status Proses" isi={rincian.data.status_proses} />
            </dl>

            {/*
              Tiga isian yang labelnya dipakai BERSAMA oleh ViewInputRegisterDetail,
              ViewInputEstimasiDetail, dan ViewShowReceiver sekaligus — tanda ia milik
              kerangka popup, bukan satu tab tertentu. Karena itu digambar di kepala.
            */}
            <div className="mt-4">
              <Isian_ dokumen={rincian.data.dokumen} isian={ISIAN_KEPALA} />
            </div>

            {Object.keys(rincian.data.dokumen).length === 0 ? (
              <p className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
                Klaim ini belum punya dokumen rincian di <code>POOLDATA.JSON_KLAIM</code>.
              </p>
            ) : (
              <IsiBertab dokumen={rincian.data.dokumen} onSub={setSub} />
            )}
          </>
        ) : null}

        {sub !== null ? (
          <SubPopupRincian
            judul={sub.judul}
            baris={sub.baris}
            isian={sub.isian}
            onTutup={() => setSub(null)}
          />
        ) : null}
        <div className="mt-6 flex justify-end">
          <Button tone="utama" onClick={onTutup}>
            Tutup
          </Button>
        </div>
      </div>
    </div>
  )
}

/** Satu nilai berlabel. Nilai kosong digambar sebagai tanda pisah, bukan sel hampa. */
function Nilai({ label, isi }: { label: string; isi: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-0.5 text-sm text-slate-900">{isi === '' ? '—' : isi}</dd>
    </div>
  )
}

/**
 * Isi popup yang dibagi menjadi **tab**, mengikuti `Harness/ViewTempDetailClaim-Harness.xml`.
 *
 * KOREKSI (2026-10-08): sebelumnya ketiga belas bagian ditumpuk ke bawah dalam satu gulungan
 * panjang. Itu **model kita sendiri** — layar lama membaginya ke tab, dan Work Owner
 * menunjukkannya berdampingan.
 *
 * # Urutan tab disalin, bukan disusun ulang
 *
 * Harness itu mendefinisikan SEMBILAN tab lewat elemen `pyTitle`, dalam urutan di bawah.
 * Urutannya terlihat janggal — "Penerima Pembayaran Klaim" mendahului "Registrasi" — dan
 * justru itu alasan menyalinnya: menyusun ulang agar "masuk akal" adalah membuat model
 * sendiri lagi.
 *
 * # Dua tab yang tidak selalu tampil
 *
 * Tangkapan layar Pega memperlihatkan TUJUH tab, bukan sembilan. Yang tidak muncul adalah
 * **Catatan Untuk Analis** dan tab objek Heavy Equipment — dan pada klaim itu keduanya memang
 * tidak punya isi.
 *
 * Tidak ada `pyVisible` pada satu pun tab di harness, jadi syaratnya **tidak terbaca dari
 * export**. Yang dipakai di sini: keduanya digambar hanya bila ada isinya. Itu cocok dengan
 * yang terlihat, tetapi ia **pengganti, bukan replikasi** — dicatat supaya tidak terbaca
 * sebagai fakta.
 *
 * # Riwayat Lampiran BUKAN tab
 *
 * `GCNMViewAttachmentHistory` dirakit harness yang sama tetapi di luar susunan tab, dan di
 * layar Pega ia memang tampil sebagai grid "Attachment" di BAWAH isi tab. Ditiru begitu.
 */
function IsiBertab({
  dokumen,
  onSub,
}: {
  dokumen: Record<string, unknown>
  onSub: (nilai: { judul: string; baris: Record<string, unknown>; isian: Isian[] }) => void
}) {
  /*
    Kedua tab bersyarat memakai penjaga Pega apa adanya, bukan "ada isinya atau tidak".

    Keduanya dipasang sebagai `pyContainerVisibleWhen` pada wadah tab di
    `Harness/ViewTempDetailClaim-Harness.xml`:

      Catatan Untuk Analis   IsPA   -> Policy.Quotation.GroupPanel = "002"
      (judul dinamis)        IsHE   -> BusinessType = "HE" atau "ContractorsPM"

    Tujuh tab lain tanpa penjaga, jadi selalu tampil. Itu cocok dengan layar Pega yang
    memperlihatkan tujuh tab pada klaim yang bukan PA dan bukan HE.
  */
  const tampilCatatan = isPA(dokumen)
  const tampilHE = isHE(dokumen)

  /*
    Judul tab HE diambil dari `pyWorkPage.ClaimData.pyLabel` — Pega memakai nilai itu sebagai
    judul, bukan teks tetap. Dibaca dari dokumen; bila tidak ada, dipakai "Objek" supaya tabnya
    tidak muncul tanpa judul sama sekali.
  */
  const labelHE = ambil(dokumen, 'ClaimData.pyLabel')
  const judulHE = labelHE.ada && labelHE.nilai.trim() !== '' ? labelHE.nilai.trim() : 'Objek'

  const tab: { kunci: string; judul: string; isi: () => ReactElement }[] = [
    {
      kunci: 'penerima',
      judul: 'Penerima Pembayaran Klaim',
      isi: () => (
        <GridRincian
          dokumen={dokumen}
          jalur="ClaimData.ReceiverClaim"
          kolom={KOLOM_PENERIMA}
          onBuka={(baris) =>
            onSub({ judul: 'Detail Penerima Klaim', baris, isian: ISIAN_PENERIMA_DETAIL })
          }
          kosong="Klaim ini belum punya penerima ganti rugi."
        />
      ),
    },
    {
      kunci: 'registrasi',
      judul: 'Registrasi',
      isi: () => (
        <div className="space-y-6">
          <Isian_ dokumen={dokumen} isian={ISIAN_REGISTER} />
          <GridObjek
            dokumen={dokumen}
            varian={VARIAN_OBJEK_REGISTER}
            kosong="Klaim ini tidak punya objek pertanggungan."
          />
        </div>
      ),
    },
    ...(tampilCatatan
      ? [
          {
            kunci: 'catatan',
            judul: 'Catatan Untuk Analis',
            isi: () => <Isian_ dokumen={dokumen} isian={ISIAN_CATATAN} />,
          },
        ]
      : []),
    {
      kunci: 'survey',
      judul: 'Hasil Survey',
      isi: () => (
        <div className="space-y-6">
          <Isian_ dokumen={dokumen} isian={ISIAN_SURVEY} />
          {isPA(dokumen) ? (
            <GridRincian
              dokumen={dokumen}
              jalur="ClaimData.SurveyorList"
              kolom={KOLOM_INVESTIGASI}
              kosong="Belum ada investigasi."
              onBuka={(baris) =>
                onSub({ judul: 'Detail Hasil Surveyor', baris, isian: ISIAN_DETAIL_SURVEYOR })
              }
            />
          ) : (
            <GridRincian
              dokumen={dokumen}
              jalur="ClaimData.SurveyResults"
              kolom={KOLOM_HASIL_SURVEY}
              kosong="Belum ada hasil survey."
              onBuka={(baris) => onSub({ judul: 'Hasil Survey', baris, isian: ISIAN_HASIL_SURVEY })}
            />
          )}
          {/*
            Grid kedua section yang sama. Keduanya SALING MENGGANTIKAN, bukan ditumpuk:
            `ViewHasilSurvey` memasang `pyContainerVisibleWhen` = `!isPA_PNC` pada grid
            pertama dan `isPA_PNC` pada yang ini. Klaim PA melihat grid investigasi, yang
            lain melihat grid survey.
          */}
        </div>
      ),
    },
    {
      kunci: 'estimasi',
      judul: 'Estimasi',
      isi: () => (
        <GridObjek
          dokumen={dokumen}
          varian={VARIAN_OBJEK_ESTIMASI}
          kosong="Klaim ini tidak punya objek pertanggungan."
        />
      ),
    },
    ...(tampilHE
      ? [
          {
            kunci: 'he',
            judul: judulHE,
            isi: () => (
              <div className="space-y-6">
                <GridRincian
                  dokumen={dokumen}
                  jalur="ClaimData.ObjectListHE"
                  kolom={KOLOM_OBJEK_HE}
                  onBuka={(baris) => onSub({ judul: 'Coverage HE', baris, isian: ISIAN_COVERAGE_HE })}
                  kosong="Klaim ini belum punya objek Heavy Equipment."
                />
                <GridRincian
                  dokumen={dokumen}
                  jalur="ClaimData.KomiteList"
                  kolom={KOLOM_KOMITE_HE}
                  kosong="Belum ada keputusan komite."
                />
              </div>
            ),
          },
        ]
      : []),
    {
      kunci: 'adj',
      judul: 'Adjustment & Akseptasi',
      isi: () => (
        <div className="space-y-6">
          <GridObjek
            dokumen={dokumen}
            varian={VARIAN_OBJEK_ADJ}
            kosong="Klaim ini tidak punya objek pertanggungan."
          />
          <GridRincian
            dokumen={dokumen}
            jalur="ClaimData.ObjectCoverageList"
            kolom={KOLOM_COVERAGE}
            onBuka={(baris) => onSub({ judul: 'Detail Coverage', baris, isian: ISIAN_COVERAGE_DETAIL })}
            kosong="Klaim ini belum punya coverage."
          />
          <GridRincian
            dokumen={dokumen}
            jalur="ClaimData.CoverageList"
            kolom={KOLOM_COVERAGE_HE}
            kosong="Belum ada coverage per objek."
            onBuka={(baris) => onSub({ judul: 'Coverage', baris, isian: ISIAN_COVERAGE_GRID })}
          />
        </div>
      ),
    },
    {
      kunci: 'dokumen',
      judul: 'Dokumen',
      isi: () => (
        <GridRincian
          dokumen={dokumen}
          jalur="ClaimData.AttachmentList"
          kolom={KOLOM_DOKUMEN}
          kosong="Klaim ini belum punya dokumen terunggah."
        />
      ),
    },
    {
      kunci: 'progress',
      judul: 'Progress Klaim',
      isi: () => (
        <GridRincian
          dokumen={dokumen}
          jalur="ClaimData.ProgressList"
          kolom={KOLOM_POSISI}
          onBuka={(baris) => onSub({ judul: 'Detail Progress', baris, isian: ISIAN_DETAIL_PROGRESS })}
          kosong="Belum ada posisi tercatat."
        />
      ),
    },
  ]

  const [aktif, setAktif] = useState(tab[0]?.kunci ?? '')
  const terpilih = tab.find((t) => t.kunci === aktif) ?? tab[0]

  return (
    <div className="mt-6">
      <div role="tablist" aria-label="Bagian rincian klaim" className="flex flex-wrap gap-1 border-b border-slate-200">
        {tab.map((t) => (
          <button
            key={t.kunci}
            type="button"
            role="tab"
            aria-selected={t.kunci === terpilih?.kunci}
            onClick={() => setAktif(t.kunci)}
            className={[
              'rounded-t-kontrol px-3 py-2 text-sm font-medium transition ease-halus',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
              'focus-visible:outline-blue-600',
              t.kunci === terpilih?.kunci
                ? 'border-b-2 border-blue-600 text-blue-700'
                : 'text-slate-500 hover:bg-slate-100 hover:text-slate-700',
            ].join(' ')}
          >
            {t.judul}
          </button>
        ))}
      </div>

      <div className="pt-6">{terpilih?.isi()}</div>

      {/*
        DI LUAR tab — `GCNMViewAttachmentHistory` dirakit harness yang sama tetapi di bawah
        susunan tab, dan layar Pega menggambarnya persis begitu.
      */}
      <section className="mt-8 border-t border-slate-200 pt-6">
        <h3 className="mb-3 text-sm font-semibold text-slate-900">Attachment</h3>
        <GridRincian
          dokumen={dokumen}
          jalur="ClaimData.AttachmentHistory"
          kolom={KOLOM_RIWAYAT_LAMPIRAN}
          kosong="Belum ada riwayat lampiran."
        />
      </section>
    </div>
  )
}

/**
 * Grid objek yang memilih variannya memakai **penjaga Pega**.
 *
 * KOREKSI (2026-10-08): sebelumnya variannya dipilih dari data, dengan alasan dua When rule
 * kosong di export. Alasan itu salah — lihat keterangan VarianObjek di spesifikasiRincian.ts.
 *
 * Varian pertama yang penjaganya terpenuhi yang digambar, persis seperti Pega menilai
 * `pyContainerVisibleWhen` dari atas ke bawah. Bila TIDAK ADA yang terpenuhi, tidak ada grid
 * yang digambar — sama seperti Pega, dan itu disengaja: menggambar varian cadangan akan
 * menampilkan kolom untuk lini yang bukan miliknya.
 */
function GridObjek({
  dokumen,
  varian,
  kosong,
}: {
  dokumen: Record<string, unknown>
  varian: VarianObjek[]
  kosong: string
}) {
  const cocok = varian.find((v) => penjagaTerpenuhi(v.penjaga, dokumen))

  if (cocok === undefined) {
    return (
      <p className="rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
        Lini bisnis klaim ini tidak termasuk yang digambar layar lama pada bagian ini.
      </p>
    )
  }

  return (
    <GridRincian dokumen={dokumen} jalur="ClaimData.ObjectList" kolom={cocok.kolom} kosong={kosong} />
  )
}
