import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type ReasMember } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useReasMemberList } from './api'

/**
 * Ukuran halaman.
 *
 * **15**, disamakan dengan Master Login, Master Rekening, Master Status Klaim, Master Status
 * Progres, Master Pasal Kerugian, dan Master Penolakan Klaim.
 *
 * Berbeda dari layar-layar itu, angkanya di sini TIDAK dibaca dari `pyPageSize` gridnya:
 * section `BrowseListMemberReas` tidak ada di export (`R-16`), sehingga ukuran halaman layar
 * lamanya tidak diketahui. Yang dipakai adalah angka yang sudah berlaku di layar master lain
 * — memperkenalkan angka ketiga tanpa dasar hanya akan membuat satu layar terasa berbeda
 * tanpa alasan yang dapat dijelaskan.
 */
const PAGE_SIZE = 15

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian ' +
            'atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar member reas tidak dapat dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
    tone: 'gangguan',
  }
}

/**
 * Layar Master Reas.
 *
 * Pengganti `Harness/DataMemberReas-harness.xml` atas tabel POOLDATA.T_REINSURER
 * (MENU_ID 35).
 *
 * # Apa yang ditampilkan layar ini
 *
 * **Daftar member reasuransi** — pihak yang menerima pemberitahuan PLA, Pre-DLA, dan DLA
 * atas klaim entitas ini, beserta login portal dan alamat surel tujuannya.
 *
 * # Layar BACA-SAJA, dan itu keputusan berdasar bukti
 *
 * Harness lamanya memuat satu grid dan satu tombol Refresh — tidak ada tombol Tambah maupun
 * Simpan. Dan satu-satunya penulis `T_REINSURER` di sistem lama adalah **alur PLA/DLA**,
 * bukan layar master:
 *
 *	Database/UPDATEREAS.prc              prosedur upsert-nya
 *	RDB List/UpdateEmailReas-SQL.xml     satu-satunya pemanggil prosedur itu
 *	Activity/UpdateDetailPLA2-Act.xml    memanggilnya  ← layar detail PLA
 *	Activity/UpdateDetailDLA2-Act.xml    memanggilnya  ← layar detail DLA
 *
 * Jadi baris reasuransi lahir dan berubah sebagai efek samping pengiriman PLA/DLA, bukan
 * lewat pemeliharaan master. Layar ini meniru itu apa adanya (`P-5`) alih-alih menambah
 * kewenangan yang tidak pernah ada — pada tabel yang menentukan ke mana pemberitahuan klaim
 * dikirim, kewenangan yang tidak pernah diminta siapa pun adalah risiko tanpa imbalan.
 *
 * # Satu keterbatasan bukti yang dinyatakan, bukan ditutupi
 *
 * Section grid `BrowseListMemberReas` **tidak ada di antara 2.634 berkas export** (`R-16`),
 * sehingga daftar kolom dan ada-tidaknya tombol simpan di dalamnya tidak terbukti. Keenam
 * kolom di bawah adalah REKONSTRUKSI dari kueri yang benar-benar ada atas tabel itu.
 *
 * # Satu perusahaan dapat muncul beberapa kali, dan kolom Tipe yang menjelaskannya
 *
 * Tanpa kolom Tipe, daftar akan terlihat memuat nama yang sama berkali-kali tanpa sebab.
 * Lihat catatan pada kolomnya.
 */
export function ReasMemberPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useReasMemberList()
  const rows = list.data?.member_reas ?? []

  /*
    Enam kolom, satu-lawan-satu dengan kolom yang dibaca kueri lama atas tabel ini.

    Kelimanya yang pertama diambil dari `RDB List/BrowseEmailReas-SQL.xml`, SELECT terlengkap
    atas tabel ini di seluruh export:

      select email as "City", reinsurername as "District", login as "DistrictID",
             reinsurerid as "CityID", country as "Country"
        from pooldata.t_reinsurer

    Alias klipboardnya TIDAK dibawa — alamat surel dialiaskan menjadi "City", dan nama
    perusahaan menjadi "District". Judul kolom di sini mengikuti ISINYA.

    Kolom keenam — Tipe — adalah PENAMBAHAN terhadap SELECT itu, dan alasannya dinyatakan
    pada kolomnya sendiri.
  */
  const columns: Column<ReasMember>[] = [
    {
      key: 'kode_reas',
      title: 'Kode Reas',
      width: '9rem',
      value: (row) => row.kode_reas,
      render: (row) => <span className="font-medium text-slate-800">{row.kode_reas}</span>,
    },
    {
      key: 'nama_reas',
      title: 'Nama Reas',
      width: '18rem',
      value: (row) => row.nama_reas,
    },
    {
      key: 'login',
      title: 'Login',
      width: '14rem',
      value: (row) => row.login,
      render: (row) =>
        row.login ? (
          row.login
        ) : (
          // Login kosong berarti mitra pada baris itu tidak akan pernah melihat klaimnya
          // sendiri — lima kueri inbox menyaringnya. Menampilkannya sebagai sel kosong
          // membuat keadaan itu tidak terlihat sama sekali.
          <span className="text-amber-700">belum ada</span>
        ),
    },
    {
      key: 'email',
      title: 'Email',
      value: (row) => row.email,
      render: (row) =>
        row.email ? (
          row.email
        ) : (
          // Baris tanpa surel gagal dalam diam: dokumen PLA/DLA-nya terbit, tercatat
          // terkirim, dan tidak pernah sampai ke siapa pun.
          <span className="text-amber-700">belum ada</span>
        ),
    },
    {
      key: 'negara',
      title: 'Negara',
      width: '10rem',
      value: (row) => row.negara,
      render: (row) => (row.negara ? row.negara : <span className="text-slate-400">—</span>),
    },
    {
      /*
        Tipe TIDAK ada di SELECT mana pun pada rule lama — ia hanya dipakai sebagai
        PENYARING (`BrowseEmailReas`) dan sebagai pembanding (`GetDataPreDLA`,
        `substr(NODLA,0,1) = TYPE`).

        Ia ditampilkan di sini sebagai PENAMBAHAN yang disadari, dan alasannya bukan
        kelengkapan: tanpa kolom ini, satu perusahaan dengan tiga jenis dokumen muncul
        sebagai tiga baris yang terlihat kembar, dengan surel berbeda-beda dan tanpa satu
        pun keterangan mengapa. Daftar seperti itu tidak dapat dipercaya pembacanya.

        Nilainya ditampilkan APA ADANYA. Artinya dalam bahasa bisnis tidak diketahui
        (`R-16`), dan menerjemahkannya menjadi label berarti mengarang.
      */
      key: 'tipe',
      title: 'Tipe',
      width: '9rem',
      value: (row) => row.tipe,
      render: (row) => (
        <span className="flex items-center gap-2">
          <span>{row.tipe || <span className="text-slate-400">—</span>}</span>
          {row.cadangan && (
            <span
              className="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-medium text-slate-600"
              title="Dipakai bila tidak ada baris yang cocok dengan jenis dokumen yang dikirim"
            >
              cadangan
            </span>
          )}
        </span>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya "Master Reas" mengikuti MENU_DESC pada
              Database/m_menu_aplikasi_pnc.csv (MENU_ID 35). Caption section lamanya sendiri
              berbunyi "Data Member" — dipakai sebagai keterangan di bawahnya (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Reas</h1>
          <p className="text-sm text-slate-600">
            Data Member — daftar mitra reasuransi penerima pemberitahuan PLA, Pre-DLA, dan
            DLA pada entitas ini.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/* SATU tombol, dan itu memang satu-satunya yang ada di harness lamanya:
              `pyButtonLabel Refresh` pada Section/ListMemberReas. Captionnya dipakai apa
              adanya (D-13). */}
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20).

          Pada layar ini ia berarti lebih dari sekadar kerapian: yang tertera adalah alamat
          surel tujuan pemberitahuan klaim, dan mitra satu badan hukum bukan mitra badan
          hukum lain. */}
      <p className="mt-3 text-xs text-slate-500">
        Daftar ini memuat seluruh member reas pada entitas yang sedang dibuka.
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar member reas…</p>
        ) : list.isError ? (
          (() => {
            const message = loadMessage(list.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <DataTable
            columns={columns}
            rows={rows}
            /*
              Kunci barisnya TIGA kolom, bukan kode reas saja.

              Kunci alaminya memang begitu — `Database/UPDATEREAS.prc` memeriksa keberadaan
              baris dengan REINSURERID + REINSURERNAME + TYPE sekaligus — dan memakai kode
              reas sendirian akan membuat tiga baris milik satu perusahaan berbagi kunci yang
              sama. React akan menganggap ketiganya satu baris.

              Pemisahnya \u001f (unit separator), bukan tanda baca biasa yang dapat muncul di
              dalam nama perusahaan.
            */
            rowKey={(row) => [row.kode_reas, row.nama_reas, row.tipe].join('\u001f')}
            description="Sumber: POOLDATA.T_REINSURER"
            searchLabel="Cari kode, nama, login, atau email"
            pageSize={PAGE_SIZE}
            emptyMessage="Belum ada member reas pada entitas ini."
          />
        )}
      </section>

      {/* Tiga keterbatasan yang nyata, dinyatakan di kaki halaman alih-alih ditemukan
          pengguna sendiri.

          Ketiganya berasal dari bentuk tabel dan dari bukti yang tersedia, bukan dari
          pilihan modul ini — dan ketiganya tidak terlihat dari layar bila tidak disebutkan. */}
      <footer className="mt-6 space-y-2 border-t border-slate-200 pt-4 text-xs text-slate-500">
        <p>
          <strong className="text-slate-700">Layar ini hanya menampilkan.</strong>{' '}
          Data member reas dibentuk dan diperbarui oleh proses pengiriman PLA dan DLA, bukan
          dari layar ini — sama seperti di aplikasi lama. Untuk mengubah alamat email
          tujuan, buka detail PLA atau DLA yang bersangkutan.
        </p>
        <p>
          <strong className="text-slate-700">Satu perusahaan dapat muncul beberapa kali.</strong>{' '}
          Setiap baris melayani satu jenis dokumen, yang ditandai kolom Tipe. Baris bertanda{' '}
          <em>cadangan</em> dipakai ketika tidak ada baris yang cocok dengan jenis dokumen
          yang sedang dikirim.
        </p>
        <p>
          <strong className="text-slate-700">Login menentukan klaim yang dilihat mitra.</strong>{' '}
          Kolom Login dipakai membatasi klaim mana yang tampil di layar seorang mitra
          reasuransi. Bila ada login yang kosong, atau satu login dipakai dua kode reas,
          laporkan ke administrator Claim PNC — keduanya tidak menimbulkan pesan kesalahan
          apa pun.
        </p>
      </footer>
    </main>
  )
}
