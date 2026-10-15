import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { blobAPI, callAPI, downloadAPI, HEADER_PORTAL } from "@/api/client";
import { useSelectedPortal } from "@/app/portal";
import { useSession } from "@/app/session";

import type {
  CountersResponse,
  DashboardFilterInput,
  DecisionRequest,
  DecisionResponse,
  DetailExportInput,
  ListResponse,
  MetadataResponse,
  PaymentDetail,
  PaymentDocument,
  PaymentSaveRequest,
  PaymentSaveResponse,
  PeriodInput,
} from "./types";

const PATH = "/api/inbox-manager";

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean satu badan hukum bukan
 * antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 *
 * Token ikut pula, karena jawaban keterangan layar BERBEDA menurut pengguna — yang dikirim
 * hanyalah tab yang boleh ia lihat.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ["inbox-manager", "tab", portal, token] as const,

  counters: (portal: string | null, token: string | null) =>
    ["inbox-manager", "ringkasan", portal, token] as const,

  list: (
    portal: string | null,
    token: string | null,
    tab: string,
    page: number,
    period: string,
  ) => ["inbox-manager", "daftar", portal, token, tab, page, period] as const,

  payment: (portal: string | null, token: string | null, acceptedNo: string) =>
    ["inbox-manager", "payment-akseptasi", portal, token, acceptedNo] as const,
};

/**
 * Hook keterangan layar — tab yang boleh dilihat, kolomnya, aturan keputusannya, dan selisih
 * terencana.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab dan aturan keputusan tiap antrean adalah HASIL PEMBACAAN export Pega,
 * dan tempat pembacaan itu tercatat adalah backend (`internal/inboxmanager/tab.go`).
 * Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan
 * tertinggal saat yang lain diperbaiki.
 *
 * Di layar ini ada alasan kedua yang lebih menentukan: aturan keputusan menentukan tombol mana
 * yang digambar. Menyusunnya di layar berarti tombol Setujui dapat tergambar aktif pada
 * antrean yang server justru menolaknya — dan pada tab Payment Klaim Akseptasi, penolakan itu
 * menyangkut uang.
 */
export function useInboxManagerMetadata() {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Lima menit: cukup lama untuk tidak diambil ulang setiap perpindahan tab, cukup pendek
    // untuk tidak menahan perubahan lini bisnis sepanjang hari.
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Hook pencacah di kepala layar.
 *
 * # TIDAK DIPAKAI layar saat ini
 *
 * Angka pencacah sempat digambar sebagai lencana pada tab dan sebagai kartu ringkasan;
 * keduanya dicabut atas permintaan Work Owner (2026-10-08) karena Pega tidak menggambarnya di
 * sana — ia menampilkannya sebagai tabel tersendiri berkolom `Status` dan `Jumlah`
 * (`Section/InboxManager_Sec`, grid `TempCountDashboard.pxResults`).
 *
 * Hook dan rutenya DIPERTAHANKAN, tidak dihapus: tabel itu ada di Pega dan dapat diminta
 * kemudian. Yang dihapus hanya pemanggilannya, supaya layar tidak menembak server untuk
 * sesuatu yang tidak digambarnya.
 *
 * # Kenapa ia permintaan TERSENDIRI
 *
 * Karena isinya tidak berubah saat tab berpindah. Menempelkannya ke setiap pemuatan tab
 * berarti sepuluh kueri pencacah dijalankan ulang pada setiap klik tab.
 */
export function useInboxManagerCounters() {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useQuery({
    queryKey: keys.counters(portal, token),
    queryFn: () =>
      callAPI<CountersResponse>(`${PATH}/ringkasan`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Pencacah berubah saat penyelia lain memutuskan, jadi cache-nya pendek.
    staleTime: 15 * 1000,
  });
}

/** Hook isi satu tab. */
export function useInboxManagerList(
  tab: string,
  page: number,
  period: PeriodInput | null,
  filter: DashboardFilterInput | null,
  enabled: boolean,
) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  const params = buildParams(tab, page, period, filter);

  return useQuery({
    queryKey: keys.list(portal, token, tab, page, params.toString()),
    queryFn: () => callAPI<ListResponse>(pathWith(params), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    staleTime: 15 * 1000,
  });
}

/**
 * Hook tombol Setujui dan Tolak.
 *
 * # Kenapa daftar DAN pencacah sama-sama dibatalkan cache-nya
 *
 * Karena keduanya berubah bersamaan: baris yang diputuskan hilang dari antreannya, dan
 * angka di kepala layar ikut berkurang. Membatalkan salah satunya saja akan menyisakan angka
 * yang tidak cocok dengan daftar di bawahnya — dan pada layar persetujuan, angka yang tidak
 * cocok adalah hal pertama yang membuat penyelia berhenti mempercayainya.
 */
export function useInboxManagerDecide() {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: DecisionRequest) =>
      callAPI<DecisionResponse>(`${PATH}/keputusan`, {
        metode: "POST",
        token,
        portal,
        body,
      }),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["inbox-manager", "daftar"] });
      queryClient.invalidateQueries({
        queryKey: ["inbox-manager", "ringkasan"],
      });
    },
  });
}

/**
 * Hook tombol Export.
 *
 * # Berkasnya CSV
 *
 * Sama seperti seluruh ekspor di aplikasi ini, dan sama pula dengan sistem lama yang memakai
 * `pxConvertResultsToCSV`.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh sebagai
 * Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi peladen. Pada
 * antrean persetujuan yang berisi puluhan baris, itu tidak berarti apa-apa.
 */
export function useExportInboxManager() {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useMutation({
    mutationFn: async (tab: string) => {
      const header: Record<string, string> = {};
      if (token) header["Authorization"] = `Bearer ${token}`;
      if (portal) header[HEADER_PORTAL] = portal;

      const params = new URLSearchParams();
      if (tab) params.set("tab", tab);

      const query = params.toString();
      const address = query ? `${PATH}/ekspor?${query}` : `${PATH}/ekspor`;

      const response = await fetch(address, { headers: header });
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak bisa
        // lagi. Yang dibaca di sini adalah kasus pertama.
        const body = (await response.json().catch(() => null)) as {
          pesan?: string;
        } | null;
        throw new Error(body?.pesan ?? "Berkas ekspor tidak dapat diambil.");
      }

      const blob = await response.blob();
      downloadBlob(blob, filenameOf(response) ?? "inbox-manager.csv");
    },
  });
}

function paymentPath(acceptedNo: string): string {
  return `${PATH}/payment-akseptasi/${encodeURIComponent(acceptedNo)}`;
}

/**
 * Hook panel detail Payment Klaim Akseptasi — tombol DETAILS (`GetDataAkseptasiKlaimKasir`).
 *
 * Nomor kosong berarti panelnya tertutup, dan tidak ada yang diambil.
 */
export function usePaymentDetail(acceptedNo: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useQuery({
    queryKey: keys.payment(portal, token, acceptedNo),
    queryFn: () =>
      callAPI<PaymentDetail>(paymentPath(acceptedNo), { token, portal }),
    enabled: acceptedNo !== "" && token !== null && portal !== null,
    // Detail selalu diambil segar saat dibuka: barisnya dapat diputuskan orang lain kapan saja.
    staleTime: 0,
  });
}

/**
 * Hook tombol SIMPAN (`SaveApprovalAkseptasiPaymentLeader_Act`).
 *
 * Daftar, pencacah, dan detail dibatalkan cache-nya bersama — alasan yang sama dengan
 * useInboxManagerDecide.
 */
export function useSavePayment(acceptedNo: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: PaymentSaveRequest) =>
      callAPI<PaymentSaveResponse>(`${paymentPath(acceptedNo)}/simpan`, {
        metode: "POST",
        token,
        portal,
        body,
      }),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["inbox-manager", "daftar"] });
      queryClient.invalidateQueries({
        queryKey: ["inbox-manager", "ringkasan"],
      });
      queryClient.invalidateQueries({
        queryKey: ["inbox-manager", "payment-akseptasi"],
      });
    },
  });
}

/**
 * Hook tombol Dokumen — membuka satu lampiran.
 *
 * Lampiran di layanan penyimpanan dibuka lewat alamat yang DIMINTA ULANG setiap kali: server
 * memperpanjangnya bila sudah lewat EXPDATE (modul Dokumen Penunjang), sehingga tautan yang
 * tersimpan di layar tidak pernah basi.
 *
 * Lampiran lama diambil sebagai blob; ditampilkan atau diunduh ditentukan SERVER lewat
 * `Content-Disposition` — pola tombol View Document Master Recovery.
 */
export function useOpenPaymentDocument(acceptedNo: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useMutation({
    mutationFn: async (document: PaymentDocument) => {
      const base = `${paymentPath(acceptedNo)}/dokumen/${encodeURIComponent(document.kunci)}`;

      if (document.tertaut) {
        const link = await callAPI<{ url: string }>(`${base}/tautan`, {
          token,
          portal,
        });
        window.open(link.url, "_blank", "noopener,noreferrer");
        return;
      }

      const file = await blobAPI(base, { token, portal });
      const address = URL.createObjectURL(file.isi);
      if (file.bolehDitampilkan) {
        window.open(address, "_blank", "noopener,noreferrer");
        // Dicabut setelah peramban sempat membacanya; mencabutnya seketika membatalkan tab
        // yang baru dibuka.
        window.setTimeout(() => URL.revokeObjectURL(address), 60_000);
        return;
      }

      const anchor = window.document.createElement("a");
      anchor.href = address;
      anchor.download = file.nama || document.nama || "dokumen";
      anchor.click();
      URL.revokeObjectURL(address);
    },
  });
}

/** Nama berkas "Export Data Detail Klaim" — sama dengan `FileName` di Pega. */
export const DETAIL_EXPORT_FILENAME = "Data Detail Klaim.csv";

/**
 * Hook tombol "Export to Excel" pada blok "Export Data Detail Klaim" tab Outstanding.
 *
 * Ekspor TERSENDIRI, bukan ekspor antrean di kepala layar: berkasnya 90 kolom dan disaring
 * hanya oleh rentang tanggal — tidak oleh PIC yang diklik, Reinsurer, maupun Kategori OS,
 * sama seperti `Activity/ExportDataDetailKlaim` di Pega.
 *
 * Tanggal kosong tidak dikirim; keduanya kosong berarti seluruh periode.
 */
export function useExportDetailKlaim() {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useMutation({
    mutationFn: (range: DetailExportInput) => {
      const params = new URLSearchParams();
      if (range.dari) params.set("dari", range.dari);
      if (range.sampai) params.set("sampai", range.sampai);

      const query = params.toString();
      const address = query
        ? `${PATH}/ekspor-detail?${query}`
        : `${PATH}/ekspor-detail`;

      return downloadAPI(address, DETAIL_EXPORT_FILENAME, { token, portal });
    },
  });
}

/**
 * buildParams menyusun parameter permintaan.
 *
 * Ia dipakai daftar DAN kunci cache, supaya dua permintaan yang penyaringnya berbeda tidak
 * pernah berbagi satu entri cache.
 */
function buildParams(
  tab: string,
  page: number,
  period: PeriodInput | null,
  filter: DashboardFilterInput | null,
): URLSearchParams {
  const params = new URLSearchParams();
  if (tab) params.set("tab", tab);
  if (page > 1) params.set("halaman", String(page));

  // Penyaring yang kosong TIDAK dikirim, supaya "All" menghasilkan alamat yang sama dengan
  // membuka tabnya tanpa menyentuh penyaring sama sekali — dan karena itu berbagi satu entri
  // cache.
  if (filter?.reinsurer) params.set("reinsurer", filter.reinsurer);
  if (filter?.kategori_os) params.set("kategori_os", filter.kategori_os);
  if (filter?.pic) params.set("pic", filter.pic);

  if (period) {
    params.set("bentuk_periode", period.bentuk);
    if (period.bentuk === "bulan") {
      if (period.bulan) params.set("bulan", period.bulan);
    } else {
      if (period.dari) params.set("dari", period.dari);
      if (period.sampai) params.set("sampai", period.sampai);
    }
  }

  return params;
}

function pathWith(params: URLSearchParams): string {
  const query = params.toString();
  return query ? `${PATH}?${query}` : PATH;
}

/** filenameOf membaca nama berkas dari header Content-Disposition. */
function filenameOf(response: Response): string | null {
  const disposition = response.headers.get("Content-Disposition");
  if (!disposition) return null;
  const found = /filename="([^"]+)"/.exec(disposition);
  return found?.[1] ?? null;
}

/**
 * downloadBlob menyimpan berkas lewat tautan sementara.
 *
 * URL objeknya DICABUT setelah dipakai. Tanpa itu, blob-nya tetap dipegang peramban sampai tab
 * ditutup — dan pada layar yang dipakai sepanjang hari, setiap ekspor menumpuk memori yang
 * tidak pernah dilepas.
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
