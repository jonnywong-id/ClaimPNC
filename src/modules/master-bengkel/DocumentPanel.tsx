import { useId, useRef, useState } from "react";

import { APIError, NetworkError } from "@/api/client";
import type { Workshop } from "@/api/types";
import { Button } from "@/components/Button";
import { ErrorMessage, type ErrorTone } from "@/components/ErrorMessage";

import {
  useOpenWorkshopDocument,
  useUploadWorkshopDocument,
  useWorkshopDocument,
} from "./api";

/**
 * Panel "Upload Document" Master Bengkel.
 *
 * # Apa yang digantikannya di Pega
 *
 *	Section/BrowseMasterHE   tombol "Upload Document"
 *	  -> Flow Action UploadDocument         @baseclass 01-01-89
 *	       -> Section UploadDocument        satu pemilih berkas, daftar "Nama File"
 *	       -> Activity SaveFilePenunjang    Call GCNMUploadResult64
 *
 * # Ke mana berkasnya pergi
 *
 * Ke jalur penyimpanan dokumen yang SUDAH ADA di aplikasi ini — modul `dokumenpenunjang`,
 * tiruan rantai `InsertDokumenPNC` Pega sampai ujungnya. Modul ini tidak membangunnya ulang;
 * ia menyambung lewat seam `DocumentUploader`, sama seperti Master Sparepart dan Master
 * Panel.
 *
 * Di sistem lama berkasnya TIDAK pernah sampai ke mana pun: `SaveAttachmentToDB_Sql`
 * memanggil prosedurnya tanpa menyertakan isi berkas, dan `IMAGEID` tidak pernah terisi.
 * Barisnya tercatat, berkasnya hilang.
 *
 * # Satu bengkel, satu dokumen
 *
 * `BENGKEL_HE` hanya punya SATU kolom `DOKUMENID`. Mengunggah berkas kedua MENGGANTI yang
 * pertama. Dinyatakan di layar sebelum pengguna memilih berkas, bukan dilaporkan sesudahnya.
 *
 * # Widget berkas bawaan disembunyikan
 *
 * Pada `<input type="file">` bawaan, SELURUH kontrolnya satu sasaran klik — termasuk teks
 * "No file chosen", yang terbaca sebagai keterangan dan bukan tombol. `sr-only` menyisakan
 * tombol sendiri sebagai satu-satunya pemicu, tanpa melepas tautan label untuk pembaca layar.
 */
export function DocumentPanel({
  workshop,
  available,
  onClose,
}: {
  workshop: Workshop;
  /** Jalur unggah siap dipakai. Dibaca dari `unggah_tersedia` pada daftar bengkel. */
  available: boolean;
  onClose: () => void;
}) {
  const fileId = useId();
  const input = useRef<HTMLInputElement>(null);

  const document = useWorkshopDocument(workshop.id_bengkel, true);
  const upload = useUploadWorkshopDocument();
  const openDocument = useOpenWorkshopDocument();

  const [chosen, setChosen] = useState<File | null>(null);
  const [tooBig, setTooBig] = useState(false);
  const [blocked, setBlocked] = useState(false);

  const attached = document.data?.dokumen ?? null;

  // Batas yang sama dengan backend. Diperiksa di peramban JUGA, bukan hanya di server:
  // berkas yang ditolak server tetap harus diunggah seluruhnya lebih dulu, dan pada koneksi
  // kantor itu menit-menit yang terbuang.
  const maxBytes = 10 * 1024 * 1024;

  // 404 di sini berarti "belum ada dokumen", bukan kerusakan — dan itu keadaan yang wajar
  // bagi sebagian besar bengkel. Ia karena itu TIDAK digambar sebagai galat.
  const loadFailed =
    document.isError &&
    !(document.error instanceof APIError && document.error.status === 404);

  function pick(file: File | null) {
    upload.reset();
    if (file !== null && file.size > maxBytes) {
      setTooBig(true);
      setChosen(null);
      if (input.current) input.current.value = "";
      return;
    }
    setTooBig(false);
    setChosen(file);
  }

  function clear() {
    setTooBig(false);
    setChosen(null);
    if (input.current) input.current.value = "";
  }

  /**
   * Membuka berkas dokumen di tab baru.
   *
   * # Tabnya dibuka SEBELUM alamatnya diminta, dan itu bukan kelalaian urutan
   *
   * Peramban hanya mengizinkan tab baru yang terbuka sebagai akibat LANGSUNG dari klik.
   * `window.open` yang dipanggil setelah permintaan jaringan selesai sudah terlepas dari
   * kliknya, dan diblokir sebagai popup — pengguna menekan tombol, menunggu, lalu tidak
   * terjadi apa-apa.
   *
   * Karena itu tab kosong dibuka lebih dulu, lalu diarahkan setelah alamatnya datang, dan
   * ditutup kembali bila permintaannya gagal.
   */
  function open() {
    setBlocked(false);
    const tab = window.open("", "_blank");
    if (tab === null) {
      // Popup diblokir pengaturan peramban. Halaman ini TIDAK dialihkan — pengguna sedang
      // berada di panel dokumennya, dan memindahkannya diam-diam lebih mengganggu daripada
      // menyatakan sebabnya.
      setBlocked(true);
      return;
    }

    openDocument.mutate(
      { id: workshop.id_bengkel },
      {
        onSuccess: (data) => {
          // Tautan ke halaman ini diputus: tab yang dibuka memegang `opener` dan dapat
          // mengarahkan halaman asalnya.
          tab.opener = null;
          tab.location.replace(data.url);
        },
        onError: () => {
          // Tab kosong yang ditinggalkan terbuka hanya membingungkan — pengguna melihat
          // halaman putih tanpa keterangan, sementara sebabnya ada di panel ini.
          tab.close();
        },
      },
    );
  }

  function submit() {
    if (chosen === null) return;
    upload.mutate(
      { id: workshop.id_bengkel, berkas: chosen },
      {
        onSuccess: () => {
          clear();
          onClose();
        },
      },
    );
  }

  return (
    <section
      className="mt-4 rounded-lg border border-slate-200 bg-white p-5"
      aria-label={`Unggah dokumen bengkel ${workshop.nama_bengkel}`}
    >
      <header className="mb-4">
        <h2 className="text-base font-semibold text-slate-900">
          Upload Document
        </h2>
        <p className="mt-0.5 text-sm text-slate-600">
          {workshop.id_bengkel} · {workshop.nama_bengkel}
        </p>
      </header>

      <div className="space-y-3">
        {!available && (
          <ErrorMessage
            title="Unggah dokumen belum tersedia"
            description="Layanan penyimpanan dokumen belum terpasang pada lingkungan ini. Data bengkel tetap dapat diubah; dokumennya menyusul setelah layanan itu tersedia."
            tone="gangguan"
          />
        )}

        {loadFailed && (
          <ErrorMessage
            title="Keterangan dokumen tidak dapat dimuat"
            description="Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC."
            tone="gangguan"
          />
        )}

        {attached !== null && (
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-sm text-slate-600">
              Dokumen saat ini:{" "}
              <span className="font-medium text-slate-800">
                {attached.nama_berkas}
              </span>
              {attached.diunggah_oleh !== "" && (
                <span className="text-slate-500">
                  {" "}
                  — diunggah {attached.diunggah_oleh}
                </span>
              )}
            </p>

            {/*
              Tombolnya hanya muncul untuk dokumen yang berkasnya benar-benar tersimpan.
              Dokumen warisan Pega tidak punya berkas, dan menawarkan tombol yang pasti
              gagal hanya memindahkan kekecewaan ke satu klik berikutnya.
            */}
            {attached.berisi && (
              <Button
                tone="kedua"
                onClick={open}
                disabled={openDocument.isPending}
              >
                {openDocument.isPending ? "Membuka…" : "Buka Dokumen"}
              </Button>
            )}
          </div>
        )}

        {attached !== null && !attached.berisi && (
          /*
            Dokumen warisan: barisnya ada, berkasnya tidak pernah tersimpan karena jalur
            unggah sistem lama tidak pernah mengisi IMAGEID. Dinyatakan supaya pengguna tahu
            mengapa dokumen yang "ada" tidak dapat dibuka.
          */
          <p className="text-sm text-amber-700">
            Berkas dokumen ini tidak tersimpan. Dokumen yang diunggah lewat
            sistem lama hanya mencatat keterangannya. Unggah ulang berkasnya
            bila masih dibutuhkan.
          </p>
        )}

        {blocked && (
          <ErrorMessage
            title="Tab baru diblokir peramban"
            description="Dokumen dibuka di tab baru, dan peramban Anda menahannya. Izinkan popup untuk alamat ini, lalu tekan Buka Dokumen sekali lagi."
            tone="gangguan"
          />
        )}

        {openDocument.isError && <OpenError error={openDocument.error} />}

        <div>
          <label
            htmlFor={fileId}
            className="block text-sm font-medium text-slate-700"
          >
            Berkas dokumen
          </label>
          <input
            id={fileId}
            ref={input}
            type="file"
            disabled={!available || upload.isPending}
            onChange={(event) => pick(event.target.files?.[0] ?? null)}
            className="sr-only"
          />
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <Button
              tone="kedua"
              onClick={() => {
                input.current?.click();
              }}
              disabled={!available || upload.isPending}
            >
              Pilih Berkas
            </Button>
          </div>
          <p className="mt-1.5 text-xs text-slate-500">Maksimal 10 MB.</p>
        </div>

        {/* Daftar "Nama File" — meniru grid modal Pega beserta baris kosongnya. */}
        <div>
          <p className="text-sm font-medium text-slate-700">Nama File</p>
          <div className="mt-1 rounded-kontrol border border-slate-200">
            <p className="px-3 py-2 text-sm text-slate-600">
              {chosen === null ? "Data tidak ada" : chosen.name}
            </p>
          </div>
        </div>

        {attached !== null && chosen !== null && (
          <p className="text-sm text-amber-700">
            Bengkel ini sudah punya dokumen. Unggahan baru akan menggantikannya.
          </p>
        )}

        {tooBig && (
          <ErrorMessage
            title="Berkas terlalu besar"
            description="Ukuran berkas melebihi batas 10 MB. Perkecil berkasnya lalu pilih ulang."
            tone="penolakan"
          />
        )}

        {upload.isError && <UploadError error={upload.error} />}

        <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 pt-3">
          <Button
            tone="halus"
            onClick={() => {
              clear();
              onClose();
            }}
            disabled={upload.isPending}
          >
            Batal
          </Button>
          <Button
            tone="utama"
            onClick={submit}
            disabled={!available || chosen === null || upload.isPending}
          >
            {upload.isPending ? "Mengunggah…" : "Submit"}
          </Button>
        </div>
      </div>
    </section>
  );
}

/**
 * Pesan galat saat membuka dokumen.
 *
 * Dipisahkan dari UploadError karena yang gagal berbeda dan yang boleh dilakukan pengguna
 * pun berbeda: unggahan yang gagal mungkin berarti berkasnya belum tersimpan, sedangkan
 * pembukaan yang gagal TIDAK pernah berarti itu — berkasnya tetap ada di penyimpanan.
 * Pesannya menyatakan hal itu supaya tidak ada yang mengunggah ulang tanpa perlu.
 */
function OpenError({ error }: { error: unknown }) {
  let title = "Dokumen tidak dapat dibuka";
  let description = "Coba beberapa saat lagi. Dokumennya tetap tersimpan.";
  let tone: ErrorTone = "gangguan";

  if (error instanceof NetworkError) {
    description =
      "Sambungan terputus saat meminta alamat berkasnya. Coba lagi.";
  } else if (error instanceof APIError) {
    switch (error.kode) {
      case "dokumen_tanpa_isi":
        // Keadaan data, bukan gangguan — mengulang tidak akan mengubah apa pun.
        title = "Berkas dokumen ini tidak tersimpan";
        description = error.message;
        tone = "penolakan";
        break;
      case "dokumen_tidak_ditemukan":
        title = "Dokumen tidak ditemukan";
        description = error.message;
        tone = "penolakan";
        break;
      case "tautan_dokumen_gagal":
        description = error.message;
        break;
      default:
        description = error.message;
    }
  }

  return <ErrorMessage title={title} description={description} tone={tone} />;
}

/**
 * Pesan galat unggahan.
 *
 * Digolongkan menurut APA YANG BOLEH DILAKUKAN PENGGUNA, bukan menurut apa yang rusak — dan
 * itu satu-satunya hal yang ingin diketahui pengguna saat unggahan gagal. Kode dan kalimatnya
 * SAMA PERSIS dengan Master Sparepart dan Master Panel.
 */
function UploadError({ error }: { error: unknown }) {
  let title = "Dokumen gagal diunggah";
  let description =
    "Berkasnya belum terkirim. Pilih berkasnya lalu unggah ulang.";
  let tone: ErrorTone = "gangguan";

  if (error instanceof NetworkError) {
    description =
      "Berkasnya tidak terkirim karena sambungan terputus. Unggah ulang.";
  } else if (error instanceof APIError) {
    switch (error.kode) {
      case "unggah_separuh_jalan":
        // Satu-satunya golongan yang pengulangannya BERBAHAYA — mengulang menumpuk berkas
        // ganda di layanan penyimpanan. Nadanya dinaikkan supaya terbaca berbeda.
        title = "Dokumen terkirim tetapi belum tercatat";
        description =
          "Berkasnya sudah sampai di penyimpanan, tetapi catatannya gagal disimpan sehingga belum tertaut ke bengkel. JANGAN unggah ulang — laporkan ke administrator.";
        tone = "penolakan";
        break;
      case "berkas_terlalu_besar":
        title = "Berkas terlalu besar";
        description = error.message;
        tone = "penolakan";
        break;
      case "unggah_tidak_sah":
        title = "Berkas tidak dapat diterima";
        description = error.message;
        tone = "penolakan";
        break;
      case "layanan_unggah_tidak_tersedia":
        title = "Layanan penyimpanan dokumen sedang tidak tersedia";
        description = `${error.message} Unggah ulang setelah layanannya pulih.`;
        break;
      default:
        description = error.message;
    }
  }

  return <ErrorMessage title={title} description={description} tone={tone} />;
}
