import { useState, type ReactNode } from "react";

import { APIError } from "@/api/client";
import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";
import { ReloadIcon } from "@/components/Icon";

import { KlaimView, ProduktivitasView } from "./DashboardViews";
import { NavList } from "./NavList";
import { OutstandingView, PIC_ALL } from "./OutstandingView";
import { QueuePanel } from "./QueuePanel";
import {
  useExportInboxManager,
  useInboxManagerCounters,
  useInboxManagerDecide,
  useInboxManagerList,
  useInboxManagerMetadata,
} from "./api";
import { messageOf } from "./controls";
import type {
  DashboardFilterInput,
  Panel,
  PeriodInput,
  Tab,
  Verdict,
} from "./types";

/** Kode tab Produktivitas Klaim — kontainer `FlagManager.AlasanKlaim==2` di Pega. */
const TAB_PRODUKTIVITAS = "2";

/** Penyaring kosong — padanan pilihan "All" di layar lama. */
const EMPTY_FILTER: DashboardFilterInput = {
  reinsurer: "",
  kategori_os: "",
  pic: "",
};

/**
 * Inbox Manager — menu `MENU_ID 58`, pengganti harness `UserInbox_Harness`.
 *
 * Isinya MEJA KERJA PENYELIA: tiga dashboard dan sembilan antrean persetujuan dalam satu
 * layar.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * ISINYA dari Pega. Harness-nya cangkang — satu `pyInclude` ke `Section/InboxManager_Sec`,
 * yang menyertakan TIGA BELAS kontainer bersyarat `FlagManager.AlasanKlaim==1` sampai `==13`.
 * Angka 1..13 itu kode tab yang sesungguhnya, dan dipakai apa adanya di sini. Kolom, label
 * tombol, dan angkanya datang dari server.
 *
 * BENTUKNYA dari rancangan layar Work Owner 2026-10-14: daftar kiri berisi dashboard dan
 * antrean sejajar, dan setiap bagian adalah kartu. Rancangan itu diikuti untuk tata letak;
 * kolom antrean, teks tombol, dan Rupiah utuh tetap seperti Pega (keputusan Work Owner pada
 * hari yang sama).
 *
 * # Modul ini MENULIS, dan itu membedakannya dari seluruh layar inbox lain
 *
 * Layar inbox lain baca-saja karena tabel objek kerja dan tabel penugasan masih dimiliki Pega
 * (`P-1`). Yang ditulis di sini adalah kolom PERSETUJUAN pada tabel POOLDATA.
 *
 * Satu jalur sengaja DITAHAN: menyetujui pembayaran pada tab Payment Klaim Akseptasi. Di Pega
 * persetujuan itu ikut menjalankan transfer ke kasir, dan rantainya belum lengkap. Alasannya
 * ditampilkan di tempat tombolnya, bukan disembunyikan.
 */
export function InboxManagerPage() {
  const [tabCode, setTabCode] = useState("");
  const [page, setPage] = useState(1);
  const [period, setPeriod] = useState<PeriodInput | null>(null);
  const [resultMessage, setResultMessage] = useState("");

  /*
    Penyaring dashboard disimpan DUA KALI: yang sedang diisi pengguna, dan yang sudah
    diterapkan — layar lama punya tombol "Filter" tersendiri, sehingga mengubah dropdown TIDAK
    langsung memuat ulang angkanya.
  */
  const [draftFilter, setDraftFilter] =
    useState<DashboardFilterInput>(EMPTY_FILTER);
  const [filter, setFilter] = useState<DashboardFilterInput>(EMPTY_FILTER);

  const metadata = useInboxManagerMetadata();
  const counters = useInboxManagerCounters();

  const tabs = metadata.data?.tab ?? [];

  /*
    Tab "Approval Master" tidak punya tempat di daftar kiri — rancangan layar menaruh
    kesembilan antreannya langsung sejajar. Bila server menjadikannya tab bawaan, yang dibuka
    antrean pertamanya, sama seperti Pega yang langsung membuka sub-tab pertama.
  */
  const requested = tabCode || metadata.data?.tab_bawaan || "";
  const requestedTab = tabs.find((tab) => tab.kode === requested);
  const activeCode =
    requestedTab?.jenis === "ringkasan"
      ? (tabs.find((tab) => tab.induk === requested)?.kode ?? requested)
      : requested;
  const activeTab = tabs.find((tab) => tab.kode === activeCode);

  const list = useInboxManagerList(
    activeCode,
    page,
    activeTab?.punya_penyaring_periode ? period : null,
    filter,
    Boolean(activeCode),
  );

  const decide = useInboxManagerDecide();
  const exportQueue = useExportInboxManager();

  function selectTab(code: string) {
    setTabCode(code);
    setPage(1);
    setResultMessage("");
    decide.reset();

    // Penyaring dashboard DIBUANG saat bagian berganti — membiarkannya menempel membuat
    // bagian lain memuat dengan penyaring yang tidak terlihat di layarnya.
    setDraftFilter(EMPTY_FILTER);
    setFilter(EMPTY_FILTER);
  }

  function submitDecision(verdict: Verdict, keys: string[], reason: string) {
    if (!activeTab) return;

    setResultMessage("");
    decide.mutate(
      { tab: activeTab.kode, keputusan: verdict, kunci: keys, alasan: reason },
      { onSuccess: (response) => setResultMessage(response.pesan) },
    );
  }

  function refresh() {
    void metadata.refetch();
    void counters.refetch();
    void list.refetch();
  }

  const line = metadata.data?.lini_bisnis_anda ?? "";

  if (metadata.isError) {
    return (
      <PageFrame line="" loading={false} onRefresh={refresh}>
        <ErrorMessage
          tone="gangguan"
          title="Layar tidak dapat dimuat"
          description={messageOf(
            metadata.error,
            "Keterangan layar tidak dapat diambil.",
          )}
        />
      </PageFrame>
    );
  }

  return (
    <PageFrame line={line} loading={metadata.isLoading} onRefresh={refresh}>
      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[18rem_minmax(0,1fr)]">
        <NavList
          tabs={tabs}
          counters={counters.data?.pencacah ?? []}
          active={activeCode}
          onSelect={selectTab}
        />

        <div className="min-w-0 space-y-4">
          {activeTab ? (
            <TabBody
              tab={activeTab}
              list={list}
              page={page}
              onPageChange={setPage}
              period={period}
              onPeriodChange={(next) => {
                setPeriod(next);
                setPage(1);
              }}
              draftFilter={draftFilter}
              onDraftChange={setDraftFilter}
              /*
                PIC yang diklik TIDAK ikut draf: ia berlaku seketika, dan Filter / Clear
                Filter hanya menyentuh Reinsurer dan Kategori OS. Di Pega keduanya pun nilai
                klipboard yang berbeda (`tempFilter.UserTeknis` versus isian dropdown).
              */
              onApplyFilter={() => setFilter({ ...draftFilter, pic: filter.pic })}
              onClearFilter={() => {
                setDraftFilter(EMPTY_FILTER);
                setFilter({ ...EMPTY_FILTER, pic: filter.pic });
              }}
              selectedPIC={filter.pic}
              onSelectPIC={(pic) =>
                setFilter({ ...filter, pic: pic === PIC_ALL ? "" : pic })
              }
              onDecide={submitDecision}
              isDeciding={decide.isPending}
              onExport={() => exportQueue.mutate(activeTab.kode)}
              isExporting={exportQueue.isPending}
              resultMessage={resultMessage}
              decideError={
                decide.isError
                  ? messageOf(decide.error, "Keputusan gagal.")
                  : undefined
              }
            />
          ) : null}

          {exportQueue.isError ? (
            <ErrorMessage
              tone="gangguan"
              title="Berkas tidak dapat diunduh"
              description={messageOf(exportQueue.error, "Berkas tidak dapat diunduh.")}
            />
          ) : null}
        </div>
      </div>

      {/*
        Blok "Yang berbeda dari layar lama" DICABUT dari layar (2026-10-07). Daftarnya TETAP
        dikirim server (`selisih_terencana`) untuk uji kesetaraan gerbang 1 (`D-54`).
      */}
    </PageFrame>
  );
}

/**
 * Pembungkus halaman — judul, cip lini bisnis, dan tombol Segarkan.
 *
 * # Lini bisnis tetap ditampilkan, termasuk saat kosong
 *
 * Ia yang menyaring ketiga dashboard. Saat kosong, keterangan panjangnya tetap digambar:
 * kosong berarti master pengguna perlu DILENGKAPI, dan dashboard menampilkan SELURUH lini.
 */
function PageFrame({
  line,
  loading,
  onRefresh,
  children,
}: {
  line: string;
  loading: boolean;
  onRefresh: () => void;
  children: ReactNode;
}) {
  return (
    <div className="halaman px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold text-slate-900">Inbox Manager</h1>
          {/*
            Sifat layar ini dinyatakan di layar: ia satu-satunya layar inbox yang MENULIS, dan
            tidak satu pun bagiannya menyaring menurut pengguna yang login. Teks dari rancangan
            layar Work Owner 2026-10-14.
          */}
          <p className="mt-1 text-sm text-slate-600">
            Pengajuan{" "}
            <span className="font-medium text-slate-800">seluruh petugas</span>,
            bukan hanya milik Anda — setiap keputusan tercatat atas nama Anda.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {!loading && line !== "" ? (
            <span className="inline-flex h-9 items-center rounded-kontrol bg-blue-50 px-3 text-xs font-semibold text-blue-700">
              Lini bisnis: {line}
            </span>
          ) : null}
          <Button type="button" onClick={onRefresh}>
            <ReloadIcon className="h-4 w-4" />
            Segarkan
          </Button>
        </div>
      </header>

      {!loading && line === "" ? (
        <p className="mt-5 rounded-kontrol border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700">
          Lini bisnis Anda <span className="font-medium">belum diisi</span> pada
          master pengguna, sehingga ketiga dashboard menampilkan SELURUH lini
          bisnis pada entitas ini. Bila Anda seharusnya melihat satu lini saja,
          sampaikan ke tim teknis — nilainya dapat dilengkapi tanpa perubahan
          aplikasi.
        </p>
      ) : null}

      <div className="mt-5">{children}</div>
    </div>
  );
}

type TabBodyProps = {
  tab: Tab;
  list: ReturnType<typeof useInboxManagerList>;
  page: number;
  onPageChange: (page: number) => void;
  period: PeriodInput | null;
  onPeriodChange: (period: PeriodInput | null) => void;
  draftFilter: DashboardFilterInput;
  onDraftChange: (filter: DashboardFilterInput) => void;
  onApplyFilter: () => void;
  onClearFilter: () => void;
  selectedPIC: string;
  onSelectPIC: (pic: string) => void;
  onDecide: (verdict: Verdict, keys: string[], reason: string) => void;
  isDeciding: boolean;
  onExport: () => void;
  isExporting: boolean;
  resultMessage: string;
  decideError?: string | undefined;
};

/** Isi bagian yang sedang terbuka, digambar menurut JENISNYA. */
function TabBody(props: TabBodyProps) {
  const { tab, list } = props;

  /*
    Galat pemuatan digambar DI DALAM kartunya, bukan sebagai pengganti kartu: isian
    penyaringnya harus tetap ada, karena galat yang paling sering justru berasal dari sana
    (rentang lintas tahun yang ditolak).

    `sumber_antrean_tidak_terbaca` sengaja TIDAK digambar (Work Owner, 2026-10-08): pesannya
    untuk tim teknis, dan antreannya tampil kosong seperti antrean yang memang belum berisi.
  */
  const sourceDown =
    list.error instanceof APIError &&
    list.error.kode === "sumber_antrean_tidak_terbaca";

  const failure =
    list.isError && !sourceDown ? (
      <ErrorMessage
        tone="gangguan"
        title={`"${tab.nama}" tidak dapat dimuat`}
        description={messageOf(list.error, "Isi bagian ini tidak dapat diambil.")}
      />
    ) : null;

  if (tab.jenis === "dashboard") {
    /*
      Gridnya TETAP digambar meski isinya belum ada — termasuk saat permintaannya ditolak,
      seperti Pega yang menampilkan alert tanpa membuang tabelnya. Bentuknya lalu diambil dari
      keterangan layar (`tab.panel`).
    */
    const panels: Panel[] =
      list.data?.panel ??
      (tab.panel ?? []).map((shape) => ({ ...shape, baris: [] }));

    if (panels.some((panel) => panel.penyaring_baris === "pic") || tab.punya_ekspor_detail) {
      return (
        <OutstandingView
          tab={tab}
          panels={panels}
          filters={list.data?.penyaring ?? []}
          isLoading={list.isLoading}
          failure={failure}
          selectedPIC={props.selectedPIC}
          onSelectPIC={props.onSelectPIC}
          draftFilter={props.draftFilter}
          onDraftChange={props.onDraftChange}
          onApplyFilter={props.onApplyFilter}
          onClearFilter={props.onClearFilter}
        />
      );
    }

    const shared = {
      tab,
      panels,
      isLoading: list.isLoading,
      failure,
      period: props.period,
      onPeriodChange: props.onPeriodChange,
      applied: list.data?.periode,
      refreshedAt: list.data?.disegarkan_pada,
    };

    // Produktivitas membandingkan dua periode; Klaim tidak. Kodenya kode kontainer Pega
    // (`FlagManager.AlasanKlaim==2`, `inboxmanager.TabProduktivitas`) — bentuk kolomnya saja
    // tidak cukup, karena panelnya boleh kosong.
    const comparison =
      tab.kode === TAB_PRODUKTIVITAS ||
      panels.some((panel) =>
        panel.kolom.some((column) => column.kunci.endsWith("_periode_tahun_lalu")),
      );
    return comparison ? <ProduktivitasView {...shared} /> : <KlaimView {...shared} />;
  }

  return (
    <div className="space-y-3">
      {failure}

      <QueuePanel
        tab={tab}
        rows={list.data?.baris ?? []}
        pagination={list.data?.paginasi}
        isLoading={list.isLoading}
        page={props.page}
        onPageChange={props.onPageChange}
        onDecide={props.onDecide}
        isDeciding={props.isDeciding}
        onExport={props.onExport}
        isExporting={props.isExporting}
        onRefresh={() => void list.refetch()}
        isRefreshing={list.isFetching}
        resultMessage={props.resultMessage}
        decideError={props.decideError}
      />
    </div>
  );
}
