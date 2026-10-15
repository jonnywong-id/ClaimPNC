import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoute } from "@/app/App";
import { useSelectedPortal } from "@/app/portal";
import { useSession } from "@/app/session";

import type { MetadataResponse, Tab, WorkItem } from "./types";

const PATH = "/api/inbox-rcl-pucl";
const TAB_PATH = `${PATH}/tab`;
const EXPORT_PATH = `${PATH}/ekspor`;
const CLAIM_PATH = `${PATH}/klaim/`;

const SAMPLE_PROFILE = {
  identitas: "90000004",
  nama: "Contoh Petugas RCL",
  jenis: "KARYAWAN",
  login: "petugasrclcontoh",
  email: "contoh.rcl@example.invalid",
  perusahaan: "ASM",
};

const PORTAL_LIST = {
  portal: [
    { id: "202600101", nama: "ASURANSI SINAR MAS", alias: "ASM", siap: true },
  ],
  utama: "ASM",
};

/**
 * Kolom grid — identik di ketiga tab kecuali judul kolom keenam.
 *
 * Kesamaan itu bukan kebetulan: ketiga section menggambar sel yang sama persis, dan yang
 * membedakan tab hanyalah penyaringnya. Justru karena itu layar ini menuntut keterangan
 * per tab — tidak ada petunjuk visual apa pun yang membedakan ketiganya.
 *
 * SATU perkecualian: tab "Cetak Surat" tidak menerima kolom "Tanggal Cetak Surat", karena
 * kolom itu justru penyaring tabnya dan tidak pernah dapat terisi di sana. Lihat
 * `TAB_CETAK`.
 */
function kolom(judulJalur: string): Tab["kolom"] {
  return [
    { kunci: "no_case", judul: "Nomor Case" },
    { kunci: "no_polis", judul: "No Polis" },
    { kunci: "nama_tertanggung", judul: "Nama Tertanggung" },
    { kunci: "tanggal_masuk_inbox", judul: "Tanggal Masuk Inbox" },
    { kunci: "deskripsi_analyst", judul: "Deskripsi Analyst" },
    { kunci: "status_rcl_pucl", judul: judulJalur },
    { kunci: "tanggal_cetak_surat", judul: "Tanggal Cetak Surat" },
    { kunci: "lama_klaim", judul: "Lama Klaim" },
    { kunci: "status_kadaluarsa", judul: "Status Kadaluarsa" },
    { kunci: "tanggal_dibuat", judul: "Tanggal Dibuat" },
  ] as Tab["kolom"];
}

/**
 * Tab "Cetak Surat" — TANPA kolom "Tanggal Cetak Surat", dan TANPA catatan.
 *
 * Keduanya hilang bersamaan pada 2026-10-11, dan memang harus bersamaan: catatan itu
 * tidak menjelaskan apa pun selain kolom tersebut.
 *
 * Tiruan ini sengaja menyusun kolomnya dengan MEMBUANG dari daftar bersama, bukan dengan
 * menulis daftar kesembilan kolomnya sendiri — daftar kedua yang ditulis tangan dapat
 * bergeser urutan dari yang pertama, dan pergeseran itulah yang justru harus tertangkap
 * uji.
 */
const TAB_CETAK: Tab = {
  kode: "1",
  nama: "Cetak Surat",
  keterangan: "Klaim RCL/PUCL yang suratnya BELUM dicetak.",
  kolom: kolom("Status RCL/PUCL").filter(
    (column) => column.kunci !== "tanggal_cetak_surat",
  ),
  punya_laporan_rentang_tanggal: true,
  punya_reminder: false,
  terhalang: false,
};

const TAB_LENGKAP: Tab = {
  kode: "2",
  nama: "Kelengkapan Dokumen",
  keterangan: "Klaim yang suratnya SUDAH dicetak tetapi belum disetujui.",
  kolom: kolom("Status RCL/PUCL"),
  punya_laporan_rentang_tanggal: false,
  punya_reminder: true,
  terhalang: false,
};

const TAB_MSIG: Tab = {
  kode: "3",
  nama: "Klaim MSIG",
  keterangan: "Klaim RCL/PUCL jalur MSIG.",
  kolom: kolom("Status"),
  punya_laporan_rentang_tanggal: false,
  punya_reminder: false,
  catatan:
    "Tab ini nyaris selalu kosong. Penandanya hanya terisi pada segelintir klaim.",
  terhalang: false,
};

const METADATA: MetadataResponse = {
  tab: [TAB_CETAK, TAB_LENGKAP, TAB_MSIG],
  tab_bawaan: "1",
  kolom_laporan: [
    { kunci: "no_case", judul: "Nomor Case" },
    { kunci: "status_klaim", judul: "Status Klaim" },
  ],
  portal: "ASM",
};

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * `tanggal_cetak_surat` sengaja KOSONG: begitulah yang selalu dikirim server pada tab
 * "Cetak Surat", karena penyaring tab itu justru "surat belum dicetak". Isiannya tetap
 * ada di baris — server tetap mengirimnya — meski tab itu tidak lagi menggambar kolomnya.
 * Uji di bawah memastikan layar menggambar kolom yang DIKIRIM server, bukan daftar kolom
 * yang ditulis tetap di layar: isian yang ada tanpa kolomnya tidak boleh muncul.
 */
const BARIS: WorkItem = {
  referensi: "ASM-FW-GCNMFW-WORK PNC-700001",
  no_case: "PNC-700001",
  no_polis: "CONTOH-RCL-0001",
  nama_tertanggung: "Tertanggung Contoh Satu",
  tanggal_masuk_inbox: "2026-09-10 09:30:00",
  deskripsi_analyst: "Dokumen pendukung tidak lengkap.",
  status_rcl_pucl: "RCL",
  tanggal_cetak_surat: "",
  // Kolomnya berjudul "Lama Klaim" tetapi berisi TANGGAL — kolomnya terverifikasi
  // bertipe timestamp di Oracle, dan Work Owner menjelaskan isinya tanggal kirim untuk
  // proses PUCL. Baris contoh yang mengisinya dengan bilangan menceritakan layar yang
  // tidak ada.
  lama_klaim: "2026-09-10 09:29:59",
  status_kadaluarsa: "Belum Kadaluarsa",
  tanggal_dibuat: "2026-08-02 08:15:00",
};

type Call = { url: string; init?: RequestInit | undefined };

let calls: Call[] = [];

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal("fetch", (url: string, init?: RequestInit) => {
    calls.push({ url, init });
    // Bilah atas memuat pemilih portal dan menu, sehingga SETIAP layar di balik sesi ikut
    // memanggil keduanya. Dijawab otomatis supaya uji layar ini menguji layarnya.
    if (url === "/api/portal")
      return Promise.resolve(jsonResponse(200, PORTAL_LIST));
    if (url === "/api/menu")
      return Promise.resolve(jsonResponse(200, { menu: [] }));
    return Promise.resolve(answer(url, init));
  });
}

/**
 * Jawaban layar kerja satu klaim.
 *
 * `up` sengaja SAMA dengan `nama_peserta`: begitulah Pega mengisinya — kedua penetapan di
 * `SetDataLampiranSuratRCLPUCL_Act` menunjuk ekspresi yang sama persis. Uji di bawah
 * memastikan layar menjelaskannya alih-alih membiarkannya terbaca sebagai kerusakan.
 */
const DETAIL = {
  referensi: BARIS.referensi,
  no_case: BARIS.no_case,
  lampiran_surat: {
    rcl_pucl: "RCL",
    kode_rcl_pucl: "1",
    deskripsi_analyst: "Dokumen pendukung tidak lengkap.",
    no_polis: "CONTOH-RCL-0001",
    tanggal_kejadian: "2026-09-01",
    nama_peserta: "Objek Contoh Satu",
    up: "Objek Contoh Satu",
    jumlah_tagihan: "15000000",
  },
  penerimaan_dokumen: { komentar_pucl: "Menunggu kelengkapan dari cabang." },
  isian_belum_terpetakan: ["NIK", "Perihal", "Email LOD"],
  tab_penerimaan_dokumen_tampil: true,
  tindakan_masih_di_pega: true,
  portal: "ASM",
};

/** Peladen tiruan yang menjawab bentuk layar, satu baris antrean, dan layar kerjanya. */
function stubDefaultFetch(rows: WorkItem[] = [BARIS]) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA);
    if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL);
    if (url.startsWith(EXPORT_PATH)) {
      return new Response("Nomor Case\nPNC-700001\n", {
        status: 200,
        headers: {
          "Content-Type": "text/csv",
          "Content-Disposition": 'attachment; filename="rcl-pucl.csv"',
        },
      });
    }

    // Tab yang diminta menentukan bentuk jawabannya. Menjawab tab yang sama untuk setiap
    // permintaan akan membuat uji perpindahan tab lulus tanpa membuktikan apa pun — dan di
    // layar ini bahayanya lebih besar, karena ketiga tab menggambar kolom yang identik.
    const wanted = new URL(url, "https://uji.invalid").searchParams.get("tab");
    const answered =
      wanted === TAB_MSIG.kode
        ? TAB_MSIG
        : wanted === TAB_LENGKAP.kode
          ? TAB_LENGKAP
          : TAB_CETAK;

    const baris = answered.kode === TAB_MSIG.kode ? [] : rows;

    return jsonResponse(200, {
      tab: answered,
      baris,
      paginasi: {
        halaman: 1,
        ukuran: 50,
        total: baris.length,
        total_halaman: 1,
      },
      portal: "ASM",
    });
  });
}

/**
 * stubDefaultFetch yang juga menjawab `POST /reminder`.
 *
 * Badan permintaannya diserahkan ke pemanggil supaya uji dapat memeriksa APA yang dikirim,
 * bukan hanya bahwa sesuatu terkirim — di layar ini yang dikirim adalah hal yang paling
 * penting: nomor klaim, dan BUKAN alamat surel.
 */
function stubDefaultFetchWithReminder(
  answer: (body: string) => unknown,
  rows: WorkItem[] = [BARIS],
) {
  stubFetch((url, init) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA);
    if (url.startsWith(`${PATH}/reminder`)) {
      const body = String((init as RequestInit | undefined)?.body ?? "");
      return jsonResponse(200, answer(body));
    }
    if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL);

    const wanted = new URL(url, "https://uji.invalid").searchParams.get("tab");
    const answered =
      wanted === TAB_MSIG.kode
        ? TAB_MSIG
        : wanted === TAB_LENGKAP.kode
          ? TAB_LENGKAP
          : TAB_CETAK;

    const baris = answered.kode === TAB_MSIG.kode ? [] : rows;

    return jsonResponse(200, {
      tab: answered,
      baris,
      paginasi: {
        halaman: 1,
        ukuran: 50,
        total: baris.length,
        total_halaman: 1,
      },
      portal: "ASM",
    });
  });
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/inbox-rcl-pucl"]}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada salah satu tab, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara bilah tab baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage();
  await screen.findByRole("tab", { name: /Cetak Surat/ });
}

function lastListCall(): Call | undefined {
  return [...calls]
    .reverse()
    .find((c) => c.url === PATH || c.url.startsWith(`${PATH}?`));
}

function lastExportCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH));
}

beforeEach(() => {
  calls = [];
  useSession.setState({
    token: "token-uji",
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  });
  useSelectedPortal.getState().select("ASM");
});

afterEach(() => {
  vi.unstubAllGlobals();
  useSession.getState().clear();
  useSelectedPortal.getState().clear();
});

describe("bentuk layar", () => {
  it("menggambar ketiga tab", async () => {
    stubDefaultFetch();
    await renderLoaded();

    expect(
      screen.getByRole("tab", { name: /Cetak Surat/ }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /Klaim MSIG/ })).toBeInTheDocument();
  });

  it("membuka tab bawaan yang ditetapkan server", async () => {
    stubDefaultFetch();
    await renderLoaded();

    expect(screen.getByRole("tab", { name: /Cetak Surat/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("menggambar kolom dari server, bukan dari daftar yang ditulis di layar", async () => {
    stubDefaultFetch();
    await renderLoaded();

    for (const judul of [
      "Nomor Case",
      "No Polis",
      "Nama Tertanggung",
      "Tanggal Masuk Inbox",
      "Deskripsi Analyst",
      "Lama Klaim",
      "Status Kadaluarsa",
    ]) {
      expect(
        await screen.findByRole("columnheader", { name: judul }),
      ).toBeInTheDocument();
    }
  });

  it('TIDAK menggambar "Tanggal Cetak Surat" di tab Cetak Surat', async () => {
    // Kolom itu adalah PENYARING tab ini (`belum dicetak`), sehingga setiap baris yang
    // tampil di sini pasti berkolom kosong — dan baris yang terisi sudah berpindah
    // sendiri ke tab "Kelengkapan Dokumen".
    //
    // Sampai 2026-10-11 kolomnya tetap digambar, ditemani satu kotak keterangan yang
    // menjelaskan mengapa ia selalu kosong. Keputusan Work Owner: kolom yang tidak dapat
    // berisi apa pun tidak ditampilkan, dan keterangannya ikut hilang.
    //
    // Yang diuji KEDUANYA sekaligus. Menguji kolomnya saja akan lolos meski layar masih
    // membela kolom yang sudah tidak ada.
    stubDefaultFetch();
    await renderLoaded();

    await screen.findByRole("columnheader", { name: "Nomor Case" });

    expect(
      screen.queryByRole("columnheader", { name: "Tanggal Cetak Surat" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/selalu kosong di tab ini/),
    ).not.toBeInTheDocument();
  });

  it('TETAP menggambar "Tanggal Cetak Surat" di tab Kelengkapan Dokumen', async () => {
    // Di sana kolomnya memang terisi — itulah tanggal yang menjadi alasan klaimnya ada di
    // tab tersebut. Membuangnya dari SELURUH layar akan menyembunyikan data yang nyata.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );

    expect(
      await screen.findByRole("columnheader", { name: "Tanggal Cetak Surat" }),
    ).toBeInTheDocument();
  });

  it("menyatakan layar ini antrean bersama, bukan pekerjaan pemanggil", async () => {
    // Tanpa keterangan ini, petugas yang terbiasa dengan inbox lain akan mengira daftarnya
    // keliru karena memuat pekerjaan orang lain.
    stubDefaultFetch();
    await renderLoaded();

    expect(screen.getByText(/antrean bersama/i)).toBeInTheDocument();
  });

  it("tidak lagi menggambar panel selisih terencana", async () => {
    // Panel ini sempat dihapus pada 2026-09-30 lalu DIKEMBALIKAN pada hari yang sama atas
    // ralat Work Owner. Penghapusan kali ini menempuh keputusan tersendiri — Work Owner,
    // 2026-10-06, berlaku untuk SELURUH layar — bukan diulang atas nama merapikan layar
    // yang panjang.
    //
    // Daftarnya tetap hidup di kode Go: `D-54` masih menuntutnya sebagai pemetaan selisih
    // ke butir `P-5` pada uji kesetaraan gerbang 1. Yang berubah adalah ia berhenti
    // menjadi isi layar.
    stubDefaultFetch();
    await renderLoaded();

    expect(
      screen.queryByText(/Yang berbeda dari layar lama/),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Tab "Klaim MSIG" nyaris selalu kosong/),
    ).not.toBeInTheDocument();
  });

  it("menunjukkan DI MANA surat dicetak, tanpa menyebut Pega", async () => {
    // Pertanyaan penggunanya tidak berubah — "mencetak suratnya di mana?" — tetapi
    // jawabannya berubah 2026-10-11.
    //
    // Sebelumnya di sini berdiri kotak "Yang dikerjakan lewat Pega" yang menyatakan Cetak
    // Surat dan Reminder PUCL hanya dapat dikerjakan di Pega. Dua-duanya tidak lagi benar:
    // Cetak Surat dikerjakan sistem ini lewat "Download Dokumen" di layar kerja klaim, dan
    // Reminder PUCL tidak pernah digambar bahkan di Pega (syarat tombolnya `1==2`).
    //
    // Yang diuji karena itu DUA hal sekaligus: keterangan barunya ada, dan kalimat lamanya
    // benar-benar hilang. Menguji yang pertama saja akan lolos meski keduanya tergambar
    // berdampingan.
    stubDefaultFetch();
    await renderLoaded();

    expect(screen.getByText(/Mencetak surat RCL\/PUCL/)).toBeInTheDocument();
    expect(screen.getByText(/Download Dokumen/)).toBeInTheDocument();

    expect(
      screen.queryByText(/Yang dikerjakan lewat Pega/),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/Reminder PUCL/)).not.toBeInTheDocument();
  });
});

describe("Reminder PUCL", () => {
  it('menggambar tombolnya HANYA di tab "Kelengkapan Dokumen"', async () => {
    // Tombolnya ada di Pega — `InboxKelengkapanDocPUCLRCL_Section:1156` — dan modul ini
    // sempat menyatakan sebaliknya. Kesimpulan lama itu benar untuk LAYAR KERJA, tempat
    // syarat tombol bernama sama ditulis `1==2`, dan salah untuk INBOX.
    //
    // Yang menentukan tampil-tidaknya adalah `punya_reminder` dari server, bukan
    // perbandingan kode tab di layar — sehingga uji ini juga menjaga aturan itu tetap di
    // satu tempat.
    stubDefaultFetch();
    await renderLoaded();

    // Tab "Cetak Surat" terbuka lebih dulu: tombolnya TIDAK boleh ada di sana.
    expect(
      screen.queryByRole("button", { name: "Reminder PUCL" }),
    ).not.toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );

    expect(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    ).toBeInTheDocument();
  });

  it("membuka modal berisi daftar klaim yang dapat dipilih", async () => {
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );

    const modal = await screen.findByRole("dialog", { name: "Reminder PUCL" });
    expect(modal).toBeInTheDocument();

    // Barisnya baris tab itu juga — penyaring kueri reminder identik dengannya.
    const pilih = await screen.findByRole("checkbox", {
      name: /Pilih PNC-700001/,
    });
    expect(pilih).not.toBeChecked();

    await userEvent.click(pilih);
    expect(pilih).toBeChecked();
    expect(screen.getByText("1 klaim dipilih.")).toBeInTheDocument();
  });

  it('"Pilih semua" mengirim NOMOR KLAIM, bukan kunci baris', async () => {
    // Jalur kedua menuju cacat yang sama. `toggle` dan `toggleAll` mengisi himpunan yang
    // sama, tetapi lewat kode yang berbeda — memperbaiki yang satu tidak otomatis
    // memperbaiki yang lain, dan tidak ada apa pun yang menandainya.
    let badan = "";
    stubDefaultFetchWithReminder((body) => {
      badan = body;
      return {
        hasil: [{ nomor_klaim: "PNC-700001", status: "terkirim" }],
        ringkasan: {
          terkirim: 1,
          tanpa_email: 0,
          tidak_memenuhi_syarat: 0,
          gagal: 0,
        },
      };
    });
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );
    await screen.findByRole("dialog", { name: "Reminder PUCL" });

    await userEvent.click(
      screen.getByRole("checkbox", { name: "Pilih semua" }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Kirim Reminder" }),
    );

    expect(
      await screen.findByText("1 pengingat terkirim."),
    ).toBeInTheDocument();
    expect(JSON.parse(badan)).toEqual({ nomor_klaim: ["PNC-700001"] });
  });

  it("TIDAK dapat mencentang baris yang tidak punya nomor klaim", async () => {
    // Pengingat ditujukan ke sebuah klaim. Baris tanpa nomor tidak menunjuk klaim mana
    // pun, dan membiarkannya dicentang hanya menghasilkan satu baris
    // "tidak-memenuhi-syarat" yang membingungkan.
    stubDefaultFetchWithReminder(
      () => ({
        hasil: [],
        ringkasan: {
          terkirim: 0,
          tanpa_email: 0,
          tidak_memenuhi_syarat: 0,
          gagal: 0,
        },
      }),
      [{ ...BARIS, no_case: "" }],
    );
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );
    await screen.findByRole("dialog", { name: "Reminder PUCL" });

    expect(
      await screen.findByRole("checkbox", { name: "Pilih" }),
    ).toBeDisabled();
  });

  it('"Pilih semua" mencentang seluruh baris halaman ini', async () => {
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );

    await screen.findByRole("dialog", { name: "Reminder PUCL" });
    await userEvent.click(
      screen.getByRole("checkbox", { name: "Pilih semua" }),
    );

    expect(
      screen.getByRole("checkbox", { name: /Pilih PNC-700001/ }),
    ).toBeChecked();
  });

  it("TIDAK membekukan keadaan basis data menjadi kalimat di layar", async () => {
    // Sampai 2026-10-12 tombol kirimnya MATI, ditemani kalimat "Email Tertanggung kosong
    // pada seluruh baris, dihitung 2026-10-01".
    //
    // Kalimat itu benar saat ditulis dan menjadi salah begitu Work Owner mengisi kolomnya
    // — persis kegagalan yang sama dengan "Kesembilannya" di kaki layar kerja. Keadaan
    // basis data tidak boleh dibekukan menjadi kalimat; yang melaporkannya harus peladen,
    // per klaim, saat pengiriman dicoba.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );
    await screen.findByRole("dialog", { name: "Reminder PUCL" });

    expect(screen.queryByText(/Belum dapat dikirim/)).not.toBeInTheDocument();
    expect(
      screen.queryByText(/kosong pada seluruh baris/),
    ).not.toBeInTheDocument();
  });

  it("mematikan tombol kirim hanya selama belum ada yang dicentang", async () => {
    // Mati karena "tidak ada yang dipilih" adalah keadaan LAYAR yang benar; mati karena
    // keadaan basis data yang ditebak di muka tidak.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );
    await screen.findByRole("dialog", { name: "Reminder PUCL" });

    expect(
      screen.getByRole("button", { name: "Kirim Reminder" }),
    ).toBeDisabled();

    await userEvent.click(
      await screen.findByRole("checkbox", { name: /Pilih PNC-700001/ }),
    );

    expect(
      screen.getByRole("button", { name: "Kirim Reminder" }),
    ).toBeEnabled();
  });

  it("MENGIRIM nomor klaim yang dicentang, bukan alamat surelnya", async () => {
    // Alamat surel sengaja TIDAK ikut: peladen membacanya ulang dari basis data. Ini
    // tindakan yang mengirim sesuatu keluar, dan alamat yang datang dari layar dapat
    // ditulis siapa pun yang dapat memanggil API.
    let badan = "";
    stubDefaultFetchWithReminder((body) => {
      badan = body;
      return {
        hasil: [{ nomor_klaim: "PNC-700001", status: "terkirim" }],
        ringkasan: {
          terkirim: 1,
          tanpa_email: 0,
          tidak_memenuhi_syarat: 0,
          gagal: 0,
        },
      };
    });
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );
    await screen.findByRole("dialog", { name: "Reminder PUCL" });

    await userEvent.click(
      await screen.findByRole("checkbox", { name: /Pilih PNC-700001/ }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Kirim Reminder" }),
    );

    expect(
      await screen.findByText("1 pengingat terkirim."),
    ).toBeInTheDocument();

    // Diperiksa PERSIS, bukan dengan `toContain`.
    //
    // Versi pertama uji ini memakai `expect(badan).toContain('PNC-700001')` dan LULUS
    // meski yang terkirim adalah kunci baris `"PNC-700001|PNC-700001"` — kunci itu memuat
    // nomor klaimnya, sehingga pemeriksaan "memuat" tidak dapat membedakan keduanya.
    //
    // Cacatnya baru terlihat di layar sebagai "klaim tidak lagi di antrean ini" — kalimat
    // yang menuduh DATA padahal yang salah kuncinya. Pembandingan persis yang
    // menangkapnya.
    expect(JSON.parse(badan)).toEqual({ nomor_klaim: ["PNC-700001"] });
  });

  it("melaporkan hasil PER KLAIM, bukan satu kalimat untuk semuanya", async () => {
    // Pengiriman sebagian adalah hasil yang lazim di sini: sebagian klaim punya alamat
    // surel, sebagian belum. Satu kalimat "berhasil" menyembunyikan klaim yang justru
    // perlu ditindak, dan satu kalimat "gagal" membuat petugas mengirim ulang ke orang
    // yang sudah menerimanya.
    stubDefaultFetchWithReminder(() => ({
      hasil: [
        { nomor_klaim: "PNC-700001", status: "terkirim" },
        { nomor_klaim: "PNC-700002", status: "tanpa-email" },
      ],
      ringkasan: {
        terkirim: 1,
        tanpa_email: 1,
        tidak_memenuhi_syarat: 0,
        gagal: 0,
      },
    }));
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Reminder PUCL" }),
    );
    await screen.findByRole("dialog", { name: "Reminder PUCL" });

    await userEvent.click(
      await screen.findByRole("checkbox", { name: /Pilih PNC-700001/ }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Kirim Reminder" }),
    );

    expect(
      await screen.findByText("1 pengingat terkirim."),
    ).toBeInTheDocument();

    // Yang TIDAK terkirim disebut satu per satu, beserta SEBABNYA — bukan hanya dihitung.
    expect(
      screen.getByText(/belum punya Email Tertanggung/),
    ).toBeInTheDocument();
    expect(screen.getByText("PNC-700002")).toBeInTheDocument();
    expect(
      screen.getByText(/Email Tertanggung belum diisi pada klaim ini/),
    ).toBeInTheDocument();
  });
});

describe("catatan per tab", () => {
  it("tidak menggambar catatan apa pun di tab Cetak Surat", async () => {
    // Tab ini sempat membawa satu catatan — penjelasan kolom "Tanggal Cetak Surat" yang
    // selalu kosong — dan catatan itu hilang bersama kolomnya pada 2026-10-11.
    //
    // Yang dijaga uji ini bukan isinya melainkan ketiadaannya: server mengirim `catatan`
    // sebagai data, sehingga catatan yang dihidupkan kembali tanpa alasan akan muncul di
    // layar tanpa satu baris pun disunting di sini.
    stubDefaultFetch();
    await renderLoaded();

    await screen.findByText(/Klaim RCL\/PUCL yang suratnya BELUM dicetak/);
    expect(
      screen.queryByText(/selalu kosong di tab ini/),
    ).not.toBeInTheDocument();
  });

  it("menampilkan catatan tab Klaim MSIG tentang kemungkinan kosong", async () => {
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(screen.getByRole("tab", { name: /Klaim MSIG/ }));

    expect(
      await screen.findByText(/nyaris selalu kosong. Penandanya hanya terisi/),
    ).toBeInTheDocument();
  });
});

describe("rentang tanggal", () => {
  it("hanya muncul di tab yang punya laporan rentang tanggal", async () => {
    stubDefaultFetch();
    await renderLoaded();

    expect(screen.getByLabelText("FROM RCL/PUCL")).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/);

    expect(screen.queryByLabelText("FROM RCL/PUCL")).not.toBeInTheDocument();
  });

  it("menyatakan bahwa tanggalnya TIDAK menyaring tabel", async () => {
    // Ini peringatan yang mencegah laporan kerusakan palsu: pengguna mengisi tanggal,
    // tabel tidak berubah, dan tanpa kalimat ini ia akan mengira layarnya rusak.
    stubDefaultFetch();
    await renderLoaded();

    expect(
      screen.getByText(/Kedua tanggal ini hanya dipakai tombol unduh/),
    ).toBeInTheDocument();
  });

  it("tidak ikut dikirim pada permintaan daftar", async () => {
    // Grid memang tidak tersaring tanggal. Mengirimkannya akan membuat orang mengira
    // sebaliknya saat menelusuri lalu lintas jaringan.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.type(screen.getByLabelText("FROM RCL/PUCL"), "2026-09-01");

    const call = lastListCall();
    expect(call?.url ?? "").not.toContain("dari=");
  });
});

describe("perpindahan tab", () => {
  it("meminta tab yang dipilih ke server", async () => {
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/);

    expect(lastListCall()?.url).toContain("tab=2");
  });

  it("mengganti judul kolom jalur pada tab Klaim MSIG", async () => {
    // Satu-satunya perbedaan visual antartab. Ia dibawa apa adanya dari section
    // (`D-13`), dan datang dari server — bukan ditebak layar.
    //
    // Stub-nya DITULIS SENDIRI di sini, bukan memakai stubDefaultFetch: di sana tab MSIG
    // sengaja dijawab kosong untuk menguji pesan antrean kosong, dan tabel kosong tidak
    // menggambar satu pun kepala kolom. Uji judul kolom karena itu menuntut tabnya berisi.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA);

      const wanted = new URL(url, "https://uji.invalid").searchParams.get(
        "tab",
      );
      const answered = wanted === TAB_MSIG.kode ? TAB_MSIG : TAB_CETAK;

      return jsonResponse(200, {
        tab: answered,
        baris: [BARIS],
        paginasi: { halaman: 1, ukuran: 50, total: 1, total_halaman: 1 },
        portal: "ASM",
      });
    });
    await renderLoaded();

    expect(
      await screen.findByRole("columnheader", { name: "Status RCL/PUCL" }),
    ).toBeInTheDocument();

    await userEvent.click(screen.getByRole("tab", { name: /Klaim MSIG/ }));

    expect(
      await screen.findByRole("columnheader", { name: "Status" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("columnheader", { name: "Status RCL/PUCL" }),
    ).not.toBeInTheDocument();
  });
});

describe("isi tabel", () => {
  it("menggambar baris dari server", async () => {
    stubDefaultFetch();
    await renderLoaded();

    expect(await screen.findByText("PNC-700001")).toBeInTheDocument();
    expect(screen.getByText("Tertanggung Contoh Satu")).toBeInTheDocument();
  });

  it("menomori barisnya di kolom paling kiri, seperti grid Pega", async () => {
    // `pyGridNumbering = true` pada ketiga section. Judul kolomnya KOSONG di sana, dan
    // nomor urut tidak menuntut penjelasan bagi pembaca layar.
    stubDefaultFetch();
    await renderLoaded();

    await screen.findByText("PNC-700001");

    const baris = screen.getAllByRole("row")[1];
    expect(baris).not.toBeUndefined();
    expect(baris?.firstElementChild).toHaveTextContent("1");
  });

  it("menggambar kolom tanggal sebagai dd/MM/yy H:mm, bukan teks mentah", async () => {
    // Bentuk sel tanggal pada grid Pega, mis. `09/10/26 17:05`. Sebelum 2026-10-10 hanya
    // nilai ber-bentuk `YYYY-MM-DD` yang diformat, sehingga nilai yang SELALU membawa jam
    // tidak pernah cocok dan tampil mentah.
    stubDefaultFetch();
    await renderLoaded();

    expect(await screen.findByText("10/09/26 9:30")).toBeInTheDocument();
    expect(screen.queryByText("2026-09-10 09:30:00")).toBeNull();
  });

  it('menggambar "Lama Klaim" sebagai waktu RELATIF, bukan tanggal penuh', async () => {
    // Sel itu satu-satunya di section ini yang ber-`pyDateTimeFormat = DateTime-Frame`,
    // format waktu relatif bawaan Pega — di layar ia berbunyi "21 hours ago".
    //
    // Yang diuji bentuknya, bukan angkanya: tanggal contohnya tetap, sedangkan "sekarang"
    // bergerak setiap kali uji ini dijalankan.
    stubDefaultFetch();
    await renderLoaded();

    await screen.findByText("PNC-700001");

    expect(screen.getByText(/ ago$/)).toBeInTheDocument();
    expect(screen.queryByText("2026-09-10 09:29:59")).toBeNull();
  });

  it("menggambar sel kosong sebagai tanda pisah, bukan dibiarkan hampa", async () => {
    // Tanda pisah menyatakan "tidak ada isinya" alih-alih "gagal dimuat" — dua keadaan
    // yang tampak sama persis bila selnya dibiarkan hampa.
    //
    // Yang dipakai di sini "Deskripsi Analyst": ia sel yang KEBETULAN kosong, dan itulah
    // yang perlakuan ini layani. Sampai 2026-10-11 uji ini bersandar pada "Tanggal Cetak
    // Surat", yang justru kolom yang TIDAK MUNGKIN terisi di tab ini — dan kolom itu kini
    // tidak digambar sama sekali.
    stubDefaultFetch([{ ...BARIS, deskripsi_analyst: "" }]);
    await renderLoaded();

    await screen.findByText("PNC-700001");
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
  });

  it("menggambar Nomor Case sebagai TAUTAN, bukan teks biasa", async () => {
    // Begitulah bentuknya di ketiga section RCL/PUCL: sel "Nomor Case" adalah
    // `pyUIElement = link` ber-`pyLabel = .pyID` (`D-13`).
    stubDefaultFetch();
    await renderLoaded();

    expect(
      await screen.findByRole("button", { name: /PNC-700001/ }),
    ).toBeInTheDocument();
  });

  it('TIDAK menggambar kolom tombol "Lihat Detail"', async () => {
    // Versi pertama modul ini menambahkan kolom tombol yang TIDAK ADA di Pega sama sekali.
    // Uji ini menjaga agar ia tidak kembali: kolom yang tidak pernah ada membuat pengguna
    // mengira ada dua cara berbeda membuka baris, dan menggeser lebar kolom lain.
    stubDefaultFetch();
    await renderLoaded();

    await screen.findByText("Tertanggung Contoh Satu");
    expect(
      screen.queryByRole("button", { name: /Lihat Detail/ }),
    ).not.toBeInTheDocument();
  });

  it("membuka LAYAR KERJA klaim, bukan panel di halaman antrean", async () => {
    // Di Pega, tautannya menjalankan `SetAssignmentInboxPUCL_act` dengan satu parameter:
    // `inskey = .pzInsKey`, dan Open Assignment membuka klaimnya pada tahap alur kerjanya.
    // Yang dituju karena itu layar TERSENDIRI — section `SendtoRCLPUCL` — bukan pratinjau
    // baris di bawah tabel.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      await screen.findByRole("button", { name: /PNC-700001/ }),
    );

    expect(await screen.findByText("Lampiran Surat")).toBeInTheDocument();
    expect(screen.getByText("Penerimaan Dokumen")).toBeInTheDocument();

    // Antreannya BENAR-BENAR ditinggalkan: bilah tab tidak lagi tergambar.
    expect(
      screen.queryByRole("tab", { name: /Cetak Surat/ }),
    ).not.toBeInTheDocument();
  });

  it("membawa kunci klaim yang TERKODEKAN ke alamat layar kerja", async () => {
    // `pzInsKey` memuat SPASI (`ASM-FW-GCNMFW-WORK PNC-700001`). Spasi mentah di dalam
    // alamat bukan alamat yang sah, dan permintaannya akan berangkat dengan kunci yang
    // rusak — tanpa satu pun galat di sisi layar.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      await screen.findByRole("button", { name: /PNC-700001/ }),
    );
    await screen.findByText("Lampiran Surat");

    const claimCall = [...calls]
      .reverse()
      .find((c) => c.url.startsWith(CLAIM_PATH));
    expect(claimCall?.url).toContain("ASM-FW-GCNMFW-WORK%20PNC-700001");
  });

  it("mengembalikan tab dan halaman yang sama saat kembali dari layar kerja", async () => {
    // Tanpa ini, petugas yang membuka satu klaim dari tab kedua mendarat kembali di tab
    // pertama — dan antrean ini dikerjakan berpuluh baris sehari.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/);

    await userEvent.click(
      await screen.findByRole("button", { name: /PNC-700001/ }),
    );
    await screen.findByText("Lampiran Surat");

    await userEvent.click(
      screen.getByRole("button", { name: /Kembali ke antrean/ }),
    );

    expect(
      await screen.findByRole("tab", {
        name: /Kelengkapan Dokumen/,
        selected: true,
      }),
    ).toBeInTheDocument();
  });

  it("menjelaskan antrean kosong menurut sebabnya", async () => {
    // "Tidak ada data" tidak cukup: ketiga tab punya kolom yang identik, sehingga pengguna
    // tidak punya petunjuk apakah ia berada di tab yang salah atau antreannya memang sepi.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(screen.getByRole("tab", { name: /Klaim MSIG/ }));

    expect(
      await screen.findByText(/tab ini memang diperkirakan kosong/i),
    ).toBeInTheDocument();
  });
});

describe("ekspor", () => {
  it("mengirim rentang tanggal pada tab Cetak Surat", async () => {
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.type(screen.getByLabelText("FROM RCL/PUCL"), "2026-09-01");
    await userEvent.type(screen.getByLabelText("TO RCL/PUCL"), "2026-09-30");
    await userEvent.click(
      screen.getByRole("button", { name: /Export to Excel/ }),
    );

    const call = lastExportCall();
    expect(call?.url).toContain("dari=2026-09-01");
    expect(call?.url).toContain("sampai=2026-09-30");
  });

  it("TIDAK mengirim rentang tanggal pada tab yang mengekspor tabel", async () => {
    // Parameter yang tidak berarti apa-apa di alamat unduhan membuat orang menduga ia
    // berpengaruh.
    stubDefaultFetch();
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/);

    await userEvent.click(
      screen.getByRole("button", { name: /Export to Excel/ }),
    );

    const call = lastExportCall();
    expect(call?.url ?? "").not.toContain("dari=");
    expect(call?.url).toContain("tab=2");
  });

  it('menamai tombolnya "Export to Excel" di SETIAP tab, persis seperti di Pega', async () => {
    // Terverifikasi pada ketiga section sebagai `pyLabel = "Export to Excel"`. Sampai
    // 2026-10-10 tab "Cetak Surat" menamainya "Unduh Laporan Harian" — nama yang tidak
    // pernah ada di layar lama, sehingga petugas mencari tombol yang tidak ada (`D-13`).
    stubDefaultFetch();
    await renderLoaded();

    expect(
      screen.getByRole("button", { name: "Export to Excel" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Unduh Laporan Harian/ }),
    ).toBeNull();

    await userEvent.click(
      screen.getByRole("tab", { name: /Kelengkapan Dokumen/ }),
    );
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/);

    expect(
      screen.getByRole("button", { name: "Export to Excel" }),
    ).toBeInTheDocument();
  });

  it("menaruh tombolnya SEBARIS dengan kedua isian tanggal, bukan di pojok layar", async () => {
    // Ketiga section menggambar FROM, lalu TO, lalu tombolnya — berurutan di dalam satu
    // tata letak yang sama di atas grid. Tombol di pojok kanan atas adalah letak yang
    // dikarang, dan ia memisahkan tombol dari isian yang mengisinya.
    stubDefaultFetch();
    await renderLoaded();

    const bilah = screen.getByLabelText("FROM RCL/PUCL").closest("section");
    expect(bilah).not.toBeNull();
    expect(bilah).toContainElement(
      screen.getByRole("button", { name: "Export to Excel" }),
    );
    expect(bilah).toContainElement(screen.getByLabelText("TO RCL/PUCL"));
  });

  it("menyebutkan kolom berkas laporan sebelum diunduh", async () => {
    // Isinya berbeda dari tabel yang sedang dilihat, dan perbedaan itu tidak boleh baru
    // ketahuan setelah berkasnya dibuka.
    stubDefaultFetch();
    await renderLoaded();

    expect(screen.getByText(/Berisi:.*Status Klaim/)).toBeInTheDocument();
  });

  it("menampilkan pesan per isian saat rentangnya ditolak server", async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA);
      if (url.startsWith(EXPORT_PATH)) {
        return jsonResponse(422, {
          kode: "validasi_gagal",
          pesan: "Permintaan belum benar.",
          detail: [
            { field: "dari", pesan: 'Tanggal "FROM RCL/PUCL" wajib diisi.' },
            { field: "sampai", pesan: 'Tanggal "TO RCL/PUCL" wajib diisi.' },
          ],
        });
      }
      return jsonResponse(200, {
        tab: TAB_CETAK,
        baris: [BARIS],
        paginasi: { halaman: 1, ukuran: 50, total: 1, total_halaman: 1 },
        portal: "ASM",
      });
    });

    await renderLoaded();
    await userEvent.click(
      screen.getByRole("button", { name: /Export to Excel/ }),
    );

    // Pesan per isian lebih berguna daripada pesan umum: ia menyebut isian MANA yang harus
    // diperbaiki, dan di layar ini isiannya ada dua.
    expect(await screen.findByRole("alert")).toHaveTextContent(
      /FROM RCL\/PUCL/,
    );
  });
});

describe("portal", () => {
  it("meminta pengguna memilih entitas lebih dulu", async () => {
    useSelectedPortal.getState().clear();
    stubDefaultFetch();

    renderPage();

    expect(
      await screen.findByText(/Pilih entitas lebih dulu/),
    ).toBeInTheDocument();
  });
});
