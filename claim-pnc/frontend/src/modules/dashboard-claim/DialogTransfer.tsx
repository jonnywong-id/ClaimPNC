import { useEffect, useRef, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useAjukanTransfer, useDaftarPIC, usePenyaringDashboard } from './api'
import type {
  BarisKlaim,
  BarisPIC,
  LingkupTransfer,
  PenyaringDashboard,
  PermintaanTransfer,
} from './types'

/**
 * Dialog Transfer — **memilih PIC Teknik dari daftar**, bukan mengetik User ID.
 *
 * # Koreksi terhadap bentuk sebelumnya
 *
 * Dialog ini semula berupa formulir dengan isian bebas User ID Lama / User ID Baru / Type
 * User. Itu **salah**. `Section/PNCTransferManagement_sec-Section.xml` menggambar sebuah
 * **grid**: daftar petugas teknis dengan tombol **"Assign"** pada setiap baris. Yang
 * dipilih pengguna adalah barisnya; tombolnya mengirim `UserID` baris itu ke
 * `PNC_ReassignPNCTeknik`.
 *
 * Perbedaannya bukan tampilan. Isian bebas menerima operator yang tidak ada atau yang tidak
 * aktif — dua hal yang justru disaring oleh daftarnya.
 *
 * # Tanpa penyaring lini bisnis — mengikuti layar lama
 *
 * Di Pega daftar ini disaring `OperatorID.pyPosition`
 * (`PNCTransferManagement_sec:3494` → `GCNMTransferAssignmentManager_act:967`), dan properti
 * itu di sana rupanya diisi "NONMBU", "TRAVEL", "BONDING", atau "PA". Nilai itu tidak ada di
 * sistem baru: HCC/HCQ mengembalikan jabatan sebenarnya (`Placement.PositionName`).
 *
 * TIGA bentuk sudah dicoba dan ketiganya keliru:
 *
 *   1. memblokir daftar dengan peringatan "tidak dapat disaring"
 *   2. memakai jabatan sesi — daftar kosong tanpa penjelasan
 *   3. meminta pengguna memilih lini bisnis — langkah yang di layar lama TIDAK ADA
 *
 * Work Owner menunjukkan layar Pega-nya: hanya daftar, paginasi, dan tombol Assign. Yang
 * berlaku sekarang karena itu sama — tanpa dropdown.
 *
 * **Selisih yang diterima:** daftarnya lebih luas daripada daftar Pega, sehingga memindahkan
 * klaim ke petugas lini lain mungkin terjadi. Penyempitannya menunggu `F-4`.
 *
 * # Assign MEMINDAHKAN, bukan mengantre
 *
 * Bentuk pertama hanya mencatat permintaan dan menyerahkan pelaksanaannya ke Pega. Work
 * Owner memutuskan (2026-10-06) seluruh jalur ini mengikuti Pega apa adanya, karena antrean
 * itu **tidak punya pelaksana** — tidak ada satu pun job Pega yang membacanya, sehingga
 * permintaan menumpuk dan pengguna menunggu sesuatu yang tidak akan datang.
 *
 * Yang dipindahkan adalah **PIC Teknik klaim** (`USERTEKNIS_1`), persis seperti
 * `PNC_ReassignPNCTeknik`.
 *
 * # Satu selisih pada jalur massal, dan ia harus diketahui
 *
 * "Transfer All Case By UserID" tanpa centang memindahkan seluruh pekerjaan satu operator.
 * Di Pega jalur itu — `GCNMTransferDataKlaim_act` — juga memanggil `pxTransferAssignment`,
 * yang memindahkan **penugasan Pega** di `PC_ASSIGN_WORKLIST`. Di sini `USERTEKNIS_1` saja
 * yang berpindah.
 *
 * Akibatnya selama masa paralel: daftar di aplikasi ini menunjukkan petugas yang baru,
 * sedangkan **inbox Pega masih menunjukkan yang lama**. Menulis `PC_ASSIGN_WORKLIST` melanggar
 * `P-1` — tabel itu ditulis Pega — sehingga selisihnya diterima, bukan ditambal diam-diam.
 */
export function DialogTransfer({
  lingkup,
  nomorKlaim,
  klaimID,
  baris = [],
  semuaCocok = false,
  penyaringAktif,
  onTutup,
}: {
  lingkup: LingkupTransfer
  nomorKlaim?: string
  klaimID?: string
  /**
   * Baris yang tercentang saat "Transfer All Case By UserID" ditekan.
   *
   * Kosong berarti tidak ada yang dicentang, dan pemindahannya mengikuti **User ID Lama** —
   * seluruh pekerjaan satu operator, seperti `GCNMTransferDataKlaim_act`. Terisi berarti
   * yang dipindahkan adalah **klaim-klaim itu saja**.
   *
   * Keduanya dibedakan di layar, bukan dibiarkan ditebak: memindahkan seluruh pekerjaan
   * seseorang dan memindahkan tiga klaim adalah dua hal yang tidak dapat dibatalkan dengan
   * mudah bila tertukar.
   */
  baris?: BarisKlaim[]

  /**
   * Benar bila yang dipindahkan SELURUH hasil penyaring — "Select All" lintas halaman.
   *
   * Dibedakan dari `baris` yang terisi: yang satu memindahkan klaim yang dicentang satu per
   * satu, yang lain memindahkan himpunan yang hanya diketahui server. Keduanya tidak dapat
   * diwakili satu bentuk tanpa salah satunya berbohong tentang jumlahnya.
   */
  semuaCocok?: boolean

  /**
   * Penyaring yang sedang berlaku, dikirim bersama permintaan pada lingkup `saring`.
   *
   * Dinamai `penyaringAktif` supaya tidak bertabrakan dengan `penyaring` di dalam komponen
   * ini, yang menampung PILIHAN penyaring dari server — dua hal berbeda yang namanya mirip.
   */
  penyaringAktif?: PenyaringDashboard

  onTutup: () => void
}) {
  const [cari, setCari] = useState('')
  const [userLama, setUserLama] = useState('')
  const [tipePengguna, setTipePengguna] = useState('')
  const [halaman, setHalaman] = useState(1)

  const penyaring = usePenyaringDashboard()
  const pilihanTipe = (penyaring.data?.tipe_pengguna ?? [])
    .filter((p) => p.nilai !== '')
    .map((p) => ({ value: p.nilai, label: p.label }))

  const daftar = useDaftarPIC(true, cari, halaman)
  const ajukan = useAjukanTransfer()
  const tutupRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    tutupRef.current?.focus()

    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onTutup()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onTutup])

  const pelanggaran = ajukan.error instanceof APIError ? ajukan.error.violations() : {}
  const berhasil = ajukan.isSuccess

  /**
   * Berapa klaim yang berpindah.
   *
   * Dua sumber, karena kedua jalur tahu jumlahnya di tempat yang berbeda:
   *
   *   - massal tanpa centang  hanya SERVER yang tahu, lewat `jumlah_pindah`
   *   - sisanya               layar sudah tahu: satu per permintaan yang dikirimnya
   *
   * Yang per-centang mengirim satu permintaan per klaim, sehingga `jumlah_pindah` jawaban
   * terakhir selalu 1 — memakainya di sana akan melaporkan "1 klaim" untuk tiga centang.
   */
  const massalSeluruhnya = lingkup === 'massal' && baris.length === 0
  const jumlahPindah = massalSeluruhnya
    ? (ajukan.data?.permintaan.jumlah_pindah ?? 0)
    : Math.max(baris.length, 1)

  /**
   * Memindahkan klaim ke petugas yang dipilih.
   *
   * Tiga bentuk, dan ketiganya memindahkan hal yang berbeda:
   *
   *   1. lingkup 'baris'                  satu klaim, dari tombol Transfer pada barisnya
   *   2. lingkup 'massal' + baris kosong  seluruh pekerjaan satu operator
   *   3. lingkup 'massal' + baris terisi  satu permintaan per klaim yang tercentang
   *
   * Bentuk ketiga sengaja memakai lingkup 'baris' per klaim, bukan satu permintaan yang
   * membawa daftar. Alasannya tetap berlaku meski antreannya sudah dicabut: satu permintaan
   * per klaim membuat kegagalan satu klaim tidak menjatuhkan klaim lain, dan jejak auditnya
   * menyebut klaim mana — `D-59` menjadikan jejak itu satu-satunya kontrol pengimbang.
   */
  function assign(pic: BarisPIC) {
    // "Select All" — SATU permintaan membawa penyaringnya, bukan 1.639 permintaan per klaim.
    if (semuaCocok) {
      ajukan.mutate({
        body: { lingkup: 'saring', user_id_baru: pic.operator_id },
        ...(penyaringAktif ? { penyaring: penyaringAktif } : {}),
      })
      return
    }

    if (lingkup === 'massal' && baris.length > 0) {
      for (const klaim of baris) {
        ajukan.mutate({
          body: {
            lingkup: 'baris',
            user_id_baru: pic.operator_id,
            ...(klaim.klaim_id ? { klaim_id: klaim.klaim_id } : {}),
            ...(klaim.nomor_klaim ? { nomor_klaim: klaim.nomor_klaim } : {}),
          },
        })
      }
      return
    }

    const body: PermintaanTransfer = {
      lingkup,
      user_id_baru: pic.operator_id,
      ...(lingkup === 'baris'
        ? {
            ...(klaimID ? { klaim_id: klaimID } : {}),
            ...(nomorKlaim ? { nomor_klaim: nomorKlaim } : {}),
          }
        : { user_id_lama: userLama, tipe_pengguna: tipePengguna }),
    }

    ajukan.mutate({ body })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      // Nama aksesibilitasnya memakai `judul()` yang sama dengan judul yang terlihat.
      // Keduanya sempat ditulis terpisah lalu menyimpang: pembaca layar menyebut "Transfer
      // seluruh pekerjaan" sementara mata membaca "Transfer All Case By UserID".
      aria-label={judul(lingkup)}
    >
      <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-kartu bg-white p-6 shadow-terbang">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            {/* Judul layar lama: pyLabel "Transfer Assignment". */}
            <h2 className="text-lg font-semibold text-slate-900">
              {judul(lingkup)}
            </h2>
            {lingkup === 'baris' && nomorKlaim ? (
              <p className="mt-1 text-sm text-slate-600">No Klaim {nomorKlaim}</p>
            ) : null}
          </div>
          {/* Tombol polos, bukan Button: ia perlu `ref` untuk menerima fokus saat dibuka. */}
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

        {berhasil ? (
          <>
            {/*
              Nol BUKAN keberhasilan yang sama dengan sekian klaim berpindah, dan warnanya pun
              berbeda. "Seluruh pekerjaan si A" yang ternyata tidak ada klaimnya adalah
              jawaban yang benar, tetapi pengguna yang membacanya sebagai hijau polos akan
              mengira pemindahannya terjadi.
            */}
            {jumlahPindah === 0 ? (
              <div className="rounded-kontrol border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                <p className="font-medium">Tidak ada klaim yang berpindah.</p>
                <p className="mt-1 leading-relaxed">
                  Petugas itu tidak memegang klaim yang masih berjalan. Periksa kembali{' '}
                  <strong>User ID Lama</strong>-nya.
                </p>
              </div>
            ) : (
              <div className="rounded-kontrol border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900">
                <p className="font-medium">
                  {jumlahPindah === 1
                    ? 'PIC Teknik berpindah.'
                    : `${jumlahPindah} klaim berpindah.`}
                </p>
                <p className="mt-1 leading-relaxed">
                  {jumlahPindah === 1 ? 'Klaim ini' : 'Klaim tersebut'} sekarang dipegang
                  petugas yang dipilih. Muat ulang daftarnya untuk melihat perubahannya.
                </p>
              </div>
            )}
            <div className="mt-4 flex justify-end">
              <Button tone="utama" onClick={onTutup}>
                Tutup
              </Button>
            </div>
          </>
        ) : (
          <div className="space-y-4">
            {lingkup === 'massal' && baris.length > 0 ? (
              <p className="rounded-kontrol border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-900">
                <strong>{baris.length}</strong> klaim tercentang akan dipindahkan ke petugas
                yang dipilih di bawah.
              </p>
            ) : null}

            {lingkup === 'massal' && baris.length === 0 ? (
              <>
                <Field
                  id="transfer-user-lama"
                  label="User ID Lama"
                  value={userLama}
                  onChange={(event) => setUserLama(event.target.value)}
                  error={pelanggaran['user_id_lama']}
                  hint="Seluruh pekerjaan operator ini akan dipindahkan ke petugas yang dipilih di bawah."
                />

                {/*
                  "Pilih Type User" pada layar lama. Ia bukan hiasan:
                  `GCNMTransferDataKlaim_act` BERCABANG pada ketiga nilainya, sehingga
                  permintaan tanpa Type User yang dikenal tidak akan cocok dengan satu
                  cabang pun saat dijalankan.
                */}
                <SelectField
                  id="transfer-tipe-pengguna"
                  label="Type User"
                  value={tipePengguna}
                  options={pilihanTipe}
                  emptyText="Pilih Type User"
                  error={pelanggaran['tipe_pengguna']}
                  onChange={(event) => setTipePengguna(event.target.value)}
                />
              </>
            ) : null}

            <Field
              id="transfer-cari-pic"
              label="Cari PIC Teknik"
              value={cari}
              onChange={(event) => setCari(event.target.value)}
              hint="Nama atau User ID."
            />

            <DaftarPIC
              state={daftar}
              halaman={halaman}
              onHalaman={setHalaman}
              onAssign={assign}
              mengirim={ajukan.isPending}
            />

            {ajukan.isError ? (
              <ErrorMessage
                tone={Object.keys(pelanggaran).length > 0 ? 'penolakan' : 'gangguan'}
                title="PIC Teknik tidak dapat dipindahkan"
                description={pesanGalat(ajukan.error)}
              />
            ) : null}

            <div className="flex justify-end">
              <Button tone="kedua" type="button" onClick={onTutup}>
                Batal
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

/**
 * Daftar petugas beserta tombol **Assign** per baris.
 *
 * # Satu kolom, bukan tiga
 *
 * `Section/PNCTransferManagement_sec-Section.xml` menggambar **satu** kolom — berjudul
 * "Nama", berisi `.City` — ditambah tombol Assign, dan nilai itu pula yang dikirim sebagai
 * `UserID` ke `PNC_ReassignPNCTeknik`. Jadi yang tampil adalah **Operator ID**, meski
 * judulnya "Nama". Judul yang menyesatkan itu milik layar lama dan diikuti apa adanya.
 *
 * Bentuk sebelumnya menggambar tiga kolom — Nama, Tim, Beban. Keduanya yang terakhir
 * tambahan sendiri, bukan replikasi, dan dihapus demi kesetaraan. Beban (`COUNTER_QUOTA`)
 * punya alasan yang masih berdiri — ia pencacah yang dipakai `BrowsePICRandomTeam-SQL`
 * untuk memilih petugas paling senggang (`R-04`) — tetapi menambah kolom yang tidak ada di
 * Pega adalah keputusan Work Owner, bukan keputusan yang diambil sambil menulis kode.
 *
 * # Paginasi
 *
 * Layar lama memaginasi daftar ini ("Page 1 of 2"), jadi paginasinya replikasi, bukan
 * tambahan.
 */
function DaftarPIC({
  state,
  halaman,
  onHalaman,
  onAssign,
  mengirim,
}: {
  state: ReturnType<typeof useDaftarPIC>
  halaman: number
  onHalaman: (nilai: number) => void
  onAssign: (pic: BarisPIC) => void
  mengirim: boolean
}) {
  if (state.isPending) {
    return <p className="px-1 py-6 text-center text-sm text-slate-500">Memuat daftar PIC…</p>
  }

  if (state.isError) {
    return (
      <ErrorMessage
        tone="gangguan"
        title="Daftar PIC tidak dapat dibaca"
        description={pesanGalat(state.error)}
      />
    )
  }

  const baris = state.data?.pic ?? []
  const total = state.data?.halaman?.total ?? 0
  const ukuran = state.data?.halaman?.ukuran ?? 0
  const jumlahHalaman = ukuran > 0 ? Math.max(1, Math.ceil(total / ukuran)) : 1

  if (baris.length === 0) {
    return (
      <p className="rounded-kontrol border border-dashed border-slate-300 px-4 py-6 text-center text-sm text-slate-600">
        Tidak ada PIC Teknik aktif yang cocok.
      </p>
    )
  }

  return (
    <div className="space-y-3">
      <div className="max-h-72 overflow-y-auto rounded-kontrol border border-slate-200">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-600">
            <tr>
              {/* Nomor urut baris — layar lama pun menomorinya. */}
              <th scope="col" className="w-10 px-3 py-2 text-right">
                <span className="sr-only">Nomor</span>
              </th>
              <th scope="col" className="px-3 py-2">Nama</th>
              <th scope="col" className="px-3 py-2" />
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {baris.map((pic, urutan) => (
              <tr key={pic.operator_id}>
                <td className="px-3 py-2 text-right tabular-nums text-slate-500">
                  {(halaman - 1) * ukuran + urutan + 1}
                </td>
                <td className="px-3 py-2 font-medium text-slate-900">{pic.operator_id}</td>
                <td className="px-3 py-2 text-right">
                  <Button tone="kedua" disabled={mengirim} onClick={() => onAssign(pic)}>
                    Assign
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {jumlahHalaman > 1 ? (
        <div className="flex items-center justify-end gap-2 text-sm text-slate-600">
          <Button
            tone="halus"
            disabled={halaman <= 1 || mengirim}
            onClick={() => onHalaman(1)}
            aria-label="Halaman pertama"
          >
            ««
          </Button>
          <Button
            tone="halus"
            disabled={halaman <= 1 || mengirim}
            onClick={() => onHalaman(halaman - 1)}
            aria-label="Halaman sebelumnya"
          >
            «
          </Button>
          <span>
            Halaman {halaman} dari {jumlahHalaman}
          </span>
          <Button
            tone="halus"
            disabled={halaman >= jumlahHalaman || mengirim}
            onClick={() => onHalaman(halaman + 1)}
            aria-label="Halaman berikutnya"
          >
            »
          </Button>
        </div>
      ) : null}
    </div>
  )
}

/**
 * Judul dialog — SATU sumber untuk judul yang terlihat dan nama aksesibilitasnya.
 *
 * Keduanya sempat ditulis terpisah lalu menyimpang. Disatukan di sini supaya penyimpangan
 * itu tidak mungkin terulang tanpa sengaja.
 *
 * Teksnya mengikuti layar lama: pyLabel "Transfer Assignment" untuk per baris, dan caption
 * tombolnya untuk yang massal.
 */
function judul(lingkup: LingkupTransfer): string {
  return lingkup === 'baris' ? 'Transfer Assignment' : 'Transfer All Case By UserID'
}

/** Membaca pesan galat yang layak dibaca pengguna. */
function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
