import { useMemo, useState, type ReactNode } from "react";

import { Button } from "@/components/Button";
import { DataTable, type Column } from "@/components/DataTable";
import { ErrorMessage } from "@/components/ErrorMessage";
import { Field } from "@/components/Field";

import {
  useEksporReas,
  useReasList,
  useReasListCounts,
  useReasMetadata,
  type ParameterDaftar,
} from "./api";
import { DetailKlaimPanel } from "./DetailKlaimPanel";
import { StatusSummary } from "./StatusSummary";
import { bukanReasuradur, formatTanggal, pesanGalat, pesanMuat } from "./pesan";
import type { Baris, Daftar } from "./types";

/**
 * Layar Inbox PLA DLA — milik REASURADUR.
 *
 * Pengganti `Harness/InboxPLADLA-Harness.xml` (`MENU_ID 45`).
 *
 * # Apa yang ditampilkan layar ini, dan untuk siapa
 *
 * **Klaim yang pemberitahuannya sudah DIKIRIMKAN kepada mitra reasuransi yang sedang
 * masuk.** Login pemanggil dicocokkan ke `POOLDATA.T_REINSURER.LOGIN`, dan hasil
 * pencocokan itulah yang menentukan klaim mana yang terlihat.
 *
 * # ENAM daftar, dan di Pega tabel "Status / Jumlah" adalah navigasinya
 *
 * Seluruhnya dikendalikan satu nilai di Pega — `TempView.CityID`, yang `SetDataPLADLA`
 * terima sebagai `param.tipe`:
 *
 *	PLA                       PLA sudah dikirimkan kepada Anda, DLA belum
 *	PLA & DLA                 DLA sudah dikirimkan kepada Anda
 *	CLOSE CLAIM               klaimnya sudah selesai dan tidak menunggu penutupan
 *	NOT ANSWERED              ada pesan untuk Anda yang belum Anda jawab
 *	NOT REPLIED FROM ASM      pesan yang Anda kirim dan belum dijawab
 *	REPLIED FROM ASM          pesan yang Anda kirim dan sudah dijawab
 *
 * Layar Pega TIDAK punya bilah tab — `pyLayoutFormat>TABBED` nol kemunculan di
 * `Section/InboxDLAReas_sect-Section.xml`. Yang berpindah daftar adalah ANGKA pada tabel
 * "Status / Jumlah": selnya ber-`pyFormat=pxLink` dengan aksi `refresh` yang membawa
 * kode daftarnya. Bilah tab yang sempat dipakai di sini dicabut 2026-10-09.
 *
 * # Kenapa "DATA PLA DLA XOL KLAIM" TIDAK ada di sini
 *
 * `SetDataPLADLA` memang punya cabang `param.tipe=="7"` yang mengisi `DataKomiteXOL`, dan
 * sectionnya memang punya wadah berjudul "DATA PLA DLA XOL KLAIM". Tetapi wadah itu
 * bersyarat `pyContainerVisibleWhen = TempView.CityID==7`, dan `CityID` HANYA pernah
 * diisi dari `.CityID` sebuah baris tabel "Status / Jumlah" — literal `7` nol kemunculan
 * sebagai nilai `tipe` di seluruh export. Tabelnya sendiri berisi enam baris.
 *
 * Artinya di Pega tampilan itu **tidak pernah dapat tergambar**: ia kode mati. Ia sempat
 * dibangun di sini sebagai tab ketujuh (2026-09-28) lalu sebagai panel permanen
 * (2026-10-09 pagi), dan keduanya memberi akses kepada sesuatu yang layar lama tidak
 * pernah tampilkan. Dicabut 2026-10-09 atas keputusan Work Owner.
 *
 * # JANGAN tertukar dengan "Inbox PLA, DLA, Pre DLA" (`MENU_ID 44`)
 *
 * Penyaringnya BERLAWANAN ARAH. Layar itu menampilkan dokumen yang belum dikirim, untuk
 * petugas internal; layar ini menampilkan dokumen yang sudah dikirim, untuk penerimanya.
 *
 * # Petugas internal yang membuka layar ini DITOLAK, bukan dibiarkan melihat layar kosong
 *
 * Login yang tidak terdaftar sebagai mitra menerima pesan yang menjelaskan sebabnya dan
 * menunjuk menu yang benar. Di Pega ia hanya melihat layar kosong tanpa satu pun
 * keterangan — dan menyimpulkan sistemnya rusak.
 *
 * # Susunan layar mengikuti Pega dari atas ke bawah (`D-13`)
 *
 *	donat + tabel "Status / Jumlah"      satu panel, selebar layar
 *	"Search" + kotak Claim No + tombol   panel tersendiri
 *	"Export To Excel"                    rata kiri, di bawah pencarian
 *	grid                                 selebar layar, berkolom nomor urut
 *
 * Sampai 2026-10-11 susunannya berbeda: kotak cari dan ketiga tombol berada **di dalam**
 * kepala grid, dan tabel "Status / Jumlah" melayang di sebelah KANAN grid. Keduanya tidak
 * ada di layar lama, dan Work Owner meminta tata letaknya dikembalikan ke Pega.
 *
 * # Layar induk BACA-SAJA; yang menulis ada di RINCIAN
 *
 * Tombol **"Detail Claim"** membuka panel rincian klaim, dan di dalamnya ada satu operasi
 * tulis: balasan komunikasi oleh mitra reasuransi.
 *
 * Dua tombol lain pada layar Pega TIDAK dibawa, dan keduanya bukan penundaan:
 *
 *	"DLA"     berada di dalam wadah bersyarat `1==2` — tidak pernah tergambar di Pega
 *	"Detail"  hanya tampil bagi satu Operator ID yang ditulis tetap di dalam rule (`D-15`)
 */
export function InboxPLADLAReasPage() {
  const meta = useReasMetadata();

  const [tab, setTab] = useState("");
  const [page, setPage] = useState(1);

  // Kotak "Claim No" dipegang DUA KALI: yang sedang diketik, dan yang sudah dikirim.
  //
  // Layar lamanya memakai tombol "Search Data", bukan pencarian saat mengetik — dan
  // penyaringnya menyapu tabel klaim yang sama dengan modul lain.
  const [ketikan, setKetikan] = useState("");
  const [dicari, setDicari] = useState("");

  // Klaim yang rinciannya sedang terbuka. Null berarti panelnya tertutup.
  const [rincian, setRincian] = useState<Baris | null>(null);

  const daftar = meta.data?.daftar ?? [];
  const aktif = tab || meta.data?.daftar_bawaan || "";
  const daftarAktif = daftar.find((item) => item.kode === aktif);

  const parameter: ParameterDaftar = useMemo(
    () => ({ tab: aktif, page, cari: dicari }),
    [aktif, page, dicari],
  );

  const siapDaftar = aktif !== "";
  const list = useReasList(parameter, siapDaftar);
  const ekspor = useEksporReas();

  // Tabel "Status / Jumlah" TIDAK menerima daftar yang sedang terbuka.
  //
  // Ia menyebut keenamnya sekaligus — itulah yang membuatnya dapat dipakai berpindah —
  // sehingga berpindah daftar tidak membuatnya dihitung ulang. Yang mengubahnya hanyalah
  // pencarian, karena angkanya memang menyusut bersama tabel di sebelahnya.
  const ringkas = useReasListCounts(dicari, siapDaftar);

  // Penolakan "bukan mitra terdaftar" dikenali dari galat daftarnya.
  const ditolak = bukanReasuradur(list.error);

  function pilihDaftar(kode: string) {
    setTab(kode);
    setPage(1);

    // Panel rincian DIBUANG saat tampilan berpindah.
    //
    // Tanpa ini, rincian klaim dari tab sebelumnya tetap terbuka di bawah daftar yang
    // sudah berganti — dan pengguna membaca rincian yang tidak ada hubungannya dengan
    // baris mana pun yang sedang ia lihat.
    setRincian(null);
  }

  function cari() {
    setDicari(ketikan.trim());
    setPage(1);
  }

  if (meta.isPending) {
    return (
      <Bingkai>
        <p className="text-sm text-slate-600">Memuat keterangan layar…</p>
      </Bingkai>
    );
  }

  if (meta.isError) {
    const pesan = pesanMuat(meta.error);
    return (
      <Bingkai>
        <ErrorMessage
          title={pesan.title}
          description={pesan.description}
          tone={pesan.tone}
        />
      </Bingkai>
    );
  }

  // Penolakan "bukan mitra terdaftar" menggantikan SELURUH isi layar, bukan digambar
  // sebagai galat di atas tabel kosong.
  //
  // Ia bukan kegagalan sementara yang layak dicoba ulang: selama pendaftarannya belum
  // diubah, jawabannya akan sama. Menggambar tabel, tombol ekspor, dan bilah tab di
  // bawahnya hanya menawarkan hal-hal yang seluruhnya akan ditolak.
  if (ditolak) {
    const pesan = pesanMuat(list.error);
    return (
      <Bingkai>
        <ErrorMessage
          title={pesan.title}
          description={pesan.description}
          tone={pesan.tone}
        />
      </Bingkai>
    );
  }

  return (
    <Bingkai>
      {/*
        Panel ringkasan di ATAS, selebar layar — bukan melayang di sebelah kanan grid.

        Di dalamnya donat dan tabel "Status / Jumlah" berdampingan, dan keduanya
        menampilkan angka yang sama. Ia menumpuk pada layar sempit, dan itu bukan
        penyimpangan: `D-12` menuntut layar ini tetap terpakai di tablet.
      */}
      <StatusSummary
        rows={ringkas.data?.baris ?? []}
        active={aktif}
        isLoading={ringkas.isLoading}
        onSelect={pilihDaftar}
      />

      {ringkas.isError && (
        <ErrorMessage
          title="Jumlah tiap daftar tidak dapat dihitung"
          description={pesanGalat(ringkas.error)}
          tone="gangguan"
        />
      )}

      {/*
        Kotak cari beserta tombolnya berdiri SENDIRI di atas grid, seperti di Pega —
        bukan di dalam kepala grid.

        Ia `<form>` supaya Enter di dalam kotaknya mencari, sama seperti menekan tombolnya.
        Di layar lama perilaku itu datang gratis dari form Pega; di sini ia harus disebut.
      */}
      <form
        className="rounded-kartu border border-slate-200 bg-white px-5 py-4 shadow-lembut"
        onSubmit={(event) => {
          event.preventDefault();
          cari();
        }}
      >
        <div className="flex flex-wrap items-end gap-2">
          <div className="min-w-[16rem] flex-1 sm:max-w-sm">
            <Field
              id="pladla-reas-cari"
              label="Search"
              /*
                Nama bagi pembaca layar menyebut KEDUANYA.

                Yang terlihat hanyalah "Search" — itu yang tertulis di layar lama — tetapi
                "Search" sendirian tidak memberi tahu apa yang dicari. Nama lengkapnya tetap
                memuat judul yang terlihat, sebagaimana dituntut WCAG 2.5.3: pengguna
                perintah suara yang mengucapkan "Search" tetap mengenai kotak ini.
              */
              aria-label="Search — Claim No"
              placeholder="Claim No"
              value={ketikan}
              onChange={(event) => {
                setKetikan(event.target.value);
              }}
            />
          </div>

          <Button type="submit" disabled={list.isFetching}>
            {list.isFetching ? "Mencari…" : "Search Data"}
          </Button>
          <Button
            type="button"
            tone="kedua"
            onClick={() => {
              list.refetch();
            }}
            disabled={list.isFetching}
          >
            Refresh
          </Button>
        </div>

        {dicari !== "" && (
          <p className="mt-2 text-xs text-slate-600" role="status">
            {list.data?.paginasi.total ?? 0} baris cocok dengan “{dicari}”.
          </p>
        )}
      </form>

      {/*
        "Export To Excel" berdiri sendiri DI BAWAH pencarian dan rata kiri, persis seperti
        di layar lama — bukan berjajar dengan "Search Data" di kepala grid.
      */}
      <div>
        <Button
          type="button"
          tone="kedua"
          disabled={ekspor.isPending}
          onClick={() => ekspor.mutate(parameter)}
        >
          {ekspor.isPending ? "Menyiapkan…" : "Export To Excel"}
        </Button>

        {ekspor.error != null && (
          <div className="mt-3">
            <ErrorMessage
              title="Berkas ekspor tidak dapat diambil"
              description={pesanGalat(ekspor.error)}
              tone="gangguan"
            />
          </div>
        )}
      </div>

      {daftarAktif && (
        <p className="text-sm text-slate-600">{daftarAktif.keterangan}</p>
      )}

      <DataTable<Baris>
        columns={kolomDaftar(daftarAktif, rincian, setRincian)}
        rows={list.data?.baris ?? []}
        rowKey={(row) => row.kunci_klaim}
        label={`Daftar ${daftarAktif?.nama ?? ""}`}
        isLoading={list.isLoading}
        error={
          list.isError ? (
            <ErrorMessage
              title="Daftar tidak dapat dimuat"
              description={pesanGalat(list.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={pesanKosong(dicari, daftarAktif)}
        /*
          Kotak cari bawaan tabel DIMATIKAN — yang berlaku adalah formulir di atas.

          `serverSearch` tetap dikirim meski kotaknya tidak tergambar, dan itu bukan sisa:
          ia yang mematikan penyaringan dan pengurutan di peramban. Tanpanya, tabel akan
          menyaring SATU HALAMAN dari sekian — hasil yang bohong.
        */
        hideSearch
        serverSearch={{
          value: ketikan,
          onChange: setKetikan,
          matchCount: list.data?.paginasi.total,
        }}
        // Grid Pega berkotak penuh, bukan hanya bergaris antarbaris.
        gridLines
        pagination={{
          page: list.data?.paginasi.halaman ?? 1,
          size: list.data?.paginasi.ukuran ?? 10,
          total: list.data?.paginasi.total ?? 0,
          totalPage: list.data?.paginasi.total_halaman ?? 1,
          onPageChange: setPage,
          isLoading: list.isFetching,
        }}
      />

      {rincian && (
        <DetailKlaimPanel
          kunciKlaim={rincian.kunci_klaim}
          nomorKlaim={rincian.no_klaim}
          onClose={() => setRincian(null)}
        />
      )}
    </Bingkai>
  );
}

/** Bingkai adalah judul layar beserta ruang isinya. */
function Bingkai({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-5 p-6">
      <header>
        <h1 className="text-lg font-semibold text-slate-900">
          Inbox PLA / DLA
        </h1>
        <p className="mt-1 text-sm text-slate-600">
          Klaim yang pemberitahuan PLA atau DLA-nya sudah dikirimkan kepada Anda
          sebagai mitra reasuransi.
        </p>
      </header>

      {children}
    </div>
  );
}

/**
 * kolomDaftar menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * Alias Pega-nya menyesatkan hampir seluruhnya — `"TSI"` berarti kunci klaim,
 * `"MARKETING"` berarti nama lini bisnis, `"CURRENCY"` berarti tanggal kejadian — dan
 * menulis ulang pemetaannya di layar berarti dua tempat yang dapat bergeser.
 */
function kolomDaftar(
  daftar: Daftar | undefined,
  rincian: Baris | null,
  bukaRincian: (row: Baris | null) => void,
): Column<Baris>[] {
  if (!daftar) return [];

  const kolom: Column<Baris>[] = [
    /*
      Kolom nomor urut, sama seperti kolom paling kiri grid Pega.

      Nomornya datang dari DataTable, bukan dari indeks baris di layar ini: ia berlanjut
      antarhalaman, sehingga baris pertama halaman kedua bernomor 11 — bukan 1 lagi.

      Judulnya ditulis "No" meskipun di Pega kepala kolomnya kosong. Kepala kolom yang
      benar-benar kosong membuat pembaca layar menyebut selnya tanpa menyebutkan ia
      nomor urut, dan tidak ada yang hilang dari tampilannya karena satu kata itu.
    */
    {
      key: "nomor_urut",
      title: "No",
      width: "3.5rem",
      noSort: true,
      alignCenter: true,
      value: () => "",
      render: (_row, nomor) => nomor,
    },
    ...daftar.kolom.map((item) => ({
      key: item.kunci,
      title: item.judul,
      value: (row: Baris) => nilaiSel(row, item.kunci),
      render: (row: Baris) => gambarSel(row, item.kunci, item.tanggal),
    })),
  ];

  // Apakah barisnya punya tombol rincian datang dari SERVER, bukan dicocokkan di layar.
  if (!daftar.punya_rincian) return kolom;

  return [
    ...kolom,
    {
      key: "rincian",
      title: "Aksi",
      noSort: true,
      alignRight: true,
      value: () => "",
      render: (row) => {
        const terbuka = rincian?.kunci_klaim === row.kunci_klaim;

        return (
          <Button
            type="button"
            tone="kedua"
            aria-pressed={terbuka}
            onClick={() => bukaRincian(terbuka ? null : row)}
          >
            {terbuka ? "Tutup" : "Detail Claim"}
          </Button>
        );
      },
    },
  ];
}

/** nilaiSel mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function nilaiSel(row: Baris, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci];
  return typeof sel === "string" ? sel : "";
}

/**
 * gambarSel menggambar satu sel.
 *
 * Kolom "Status" jatuh ke KODENYA bila artinya tidak ada di master. Kode status yang
 * tidak terdaftar memang terjadi — domainnya 33 kode dan data lama memuat kode di luar
 * itu — dan sel kosong akan membuat barisnya tampak rusak.
 */
function gambarSel(row: Baris, kunci: string, tanggal: boolean) {
  if (kunci === "status") {
    const isi = row.status || row.kode_status;
    if (isi === "") return <span className="text-slate-400">—</span>;
    return isi;
  }

  const isi = nilaiSel(row, kunci);
  if (tanggal) return formatTanggal(isi);
  if (isi === "") return <span className="text-slate-400">—</span>;
  return isi;
}

/**
 * pesanKosong menjelaskan MENGAPA daftarnya kosong.
 *
 * Kalimatnya berbeda antara daftar pemberitahuan dan daftar komunikasi, dan itu bukan
 * kehalusan: keduanya kosong karena sebab yang berbeda. Satu kalimat untuk keduanya akan
 * membuat mitra yang membuka "Komunikasi Masuk" membaca keterangan tentang pemberitahuan
 * yang tidak ada hubungannya dengan tab itu.
 */
function pesanKosong(dicari: string, daftar: Daftar | undefined): string {
  if (dicari !== "") {
    return `Tidak ada klaim yang nomornya mengandung "${dicari}" pada daftar ini.`;
  }

  // Sumbernya datang dari SERVER. Sebelumnya layar menyimpulkannya dari awalan kode tab,
  // dan kesimpulan itu pecah diam-diam begitu kodenya berubah mengikuti judul sebenarnya
  // — tanpa satu pun galat, hanya kalimat yang salah.
  if (daftar?.sumber === "komunikasi") {
    return "Belum ada klaim dengan komunikasi pada tahap ini yang menyangkut Anda.";
  }

  return "Belum ada klaim pada tahap ini yang pemberitahuannya dikirimkan kepada Anda.";
}
