import { useEffect, useState } from "react";

import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";
import { Field } from "@/components/Field";

import type { RejectLetterRequest } from "./types";

/**
 * Form "Download Dokumen Reject" — delapan isian beserta grid Alasan.
 *
 * # Dari mana bentuknya berasal
 *
 * `Section/FormRejectClaim_section-Section.xml`. Kedelapan labelnya diambil dari
 * `pyLabelPreview`, BUKAN `pyLabelFieldValue` — dua di antaranya tersimpan terpotong di
 * sana (`Up` menjadi `"U"`).
 *
 * Nama tombolnya **"Generate PDF"**, bukan "Download", dan itu bukan pilihan kata kami:
 * begitulah tertulis di sel kontrolnya. Namanya memang lebih tepat — tombol itu tidak
 * mengirim berkas ke peramban melainkan MELAMPIRKAN surat ke klaim.
 *
 * # Isiannya tidak tersimpan sebagai data klaim
 *
 * Tidak ada kolom basis data untuk kedelapannya; ia bahan surat, berumur satu permintaan.
 * Karena itu dialog ini memulai dari keadaan kosong setiap kali dibuka, kecuali tiga
 * isian yang memang dipra-isi dari klaim.
 *
 * # Satu hal yang MENUNGGU jawaban Work Owner
 *
 * Pada templat Pega yang benar-benar dirender (`HTML/RejectRefundLetter-HTML.xml`), sel
 * nilainya KOSONG — kedelapan isian ini dan seluruh baris Alasan tidak muncul di surat.
 * Sistem baru mengisinya, mengikuti templat kembarnya yang merge field-nya terpasang
 * (`HTML/RejectLetter-HTML.xml`). Lihat catatan paket `rejectpdf` di backend.
 */
export function RejectLetterDialog({
  prefill,
  onBatal,
  onTerbitkan,
  sedangMenerbitkan,
  galat,
}: {
  prefill: RejectLetterPrefill;
  onBatal: () => void;
  onTerbitkan: (isian: RejectLetterRequest) => void;
  sedangMenerbitkan: boolean;
  galat: string | null;
}) {
  const [isian, setIsian] = useState<RejectLetterRequest>(() => ({
    up: "",
    jabatan: "",
    nama_pasien: prefill.nama_pasien ?? "",
    tempat_kejadian: prefill.tempat_kejadian ?? "",
    tanggal_kejadian: prefill.tanggal_kejadian ?? "",
    tanggal_keluar_rawat_inap: "",
    nilai_klaim_dibayarkan: "",
    tanggal_pembayaran: "",
    alasan: [""],
  }));

  // Escape menutup dialog, seperti dialog mana pun yang dikenal pengguna.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape" && !sedangMenerbitkan) onBatal();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onBatal, sedangMenerbitkan]);

  function ubah(kunci: keyof RejectLetterRequest, nilai: string) {
    setIsian((sebelum) => ({ ...sebelum, [kunci]: nilai }));
  }

  function ubahAlasan(indeks: number, nilai: string) {
    setIsian((sebelum) => ({
      ...sebelum,
      alasan: sebelum.alasan.map((a, i) => (i === indeks ? nilai : a)),
    }));
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-surat-penolakan"
    >
      <div className="max-h-full w-full max-w-2xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2
          id="judul-surat-penolakan"
          className="text-lg font-semibold text-slate-900"
        >
          Download Dokumen Reject
        </h2>

        <p className="mt-2 text-sm text-slate-600">
          Surat yang terbit <strong>melampirkan diri ke klaim ini</strong> dan muncul di
          grid Dokumen. Menekan Generate PDF lagi akan mengganti surat sebelumnya, bukan
          menambah yang baru.
        </p>

        {/*
          Keterangan ini TIDAK ada di Pega, dan ditambahkan dengan sadar.

          Templat suratnya menggambar baris data dan daftar alasan KOSONG — suratnya
          dicetak lalu dilengkapi tangan. Tanpa keterangan ini, petugas mengisi delapan
          isian di bawah, menekan Generate PDF, lalu mencari isiannya di surat dan tidak
          menemukannya. Yang terjadi berikutnya adalah ia menekan tombolnya berulang kali.

          Jadi ia bukan perubahan form — formnya tetap sama persis dengan Pega. Ia
          menjelaskan apa yang akan terjadi, yang di Pega memang tidak pernah dijelaskan.
        */}
        <p className="mt-2 text-sm text-slate-600">
          Surat dicetak dalam <strong>format isian</strong>: baris data dan daftar alasan
          sengaja dibiarkan kosong untuk dilengkapi tangan setelah dicetak, mengikuti
          format surat yang berlaku.
        </p>

        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <Field
            id="surat-up"
            label="Up"
            value={isian.up}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("up", e.target.value)}
          />
          <Field
            id="surat-jabatan"
            label="Jabatan"
            value={isian.jabatan}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("jabatan", e.target.value)}
          />
          <Field
            id="surat-nama-pasien"
            label="Nama Pasien / No.Reg"
            value={isian.nama_pasien}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("nama_pasien", e.target.value)}
          />
          <Field
            id="surat-tempat-kejadian"
            label="Tempat Kejadian / Perawatan"
            value={isian.tempat_kejadian}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("tempat_kejadian", e.target.value)}
          />
          <Field
            id="surat-tanggal-kejadian"
            label="Tanggal Kejadian / Tanggal Masuk Rawat Inap"
            value={isian.tanggal_kejadian}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("tanggal_kejadian", e.target.value)}
          />
          <Field
            id="surat-tanggal-keluar"
            label="Tanggal Keluar Rawat Inap"
            // Terbuka KOSONG dan diketik petugas: isian ini tidak punya kolom di
            // `T_CLAIM_PNC`, sehingga tidak ada yang dapat dipra-isi. Di Pega pun ia
            // kosong ketika klaimnya bukan berasal dari Receive Document.
            hint="Tidak terbawa dari klaim — ketik bila diperlukan."
            value={isian.tanggal_keluar_rawat_inap}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("tanggal_keluar_rawat_inap", e.target.value)}
          />
          <Field
            id="surat-nilai-dibayarkan"
            label="Nilai Klaim Yang Sudah Dibayarkan"
            // TEKS, bukan angka — selnya `pxTextInput` di Pega, dan isinya ditempelkan apa
            // adanya ke surat. Menjadikannya angka akan menolak masukan yang Pega terima.
            value={isian.nilai_klaim_dibayarkan}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("nilai_klaim_dibayarkan", e.target.value)}
          />
          <Field
            id="surat-tanggal-pembayaran"
            label="Tanggal Pembayaran Claim"
            value={isian.tanggal_pembayaran}
            disabled={sedangMenerbitkan}
            onChange={(e) => ubah("tanggal_pembayaran", e.target.value)}
          />
        </div>

        <fieldset className="mt-5">
          <legend className="text-sm font-semibold text-slate-900">Alasan</legend>

          <div className="mt-2 space-y-2">
            {isian.alasan.map((alasan, indeks) => (
              <Field
                // Indeks sebagai kunci memang dihindari pada daftar yang dapat diurutkan
                // ulang. Di sini barisnya hanya bertambah di ujung dan tidak pernah
                // berpindah tempat, sehingga indeksnya memang identitas yang stabil.
                key={indeks}
                id={`surat-alasan-${indeks}`}
                label={`Alasan ${indeks + 1}`}
                value={alasan}
                disabled={sedangMenerbitkan}
                onChange={(e) => ubahAlasan(indeks, e.target.value)}
              />
            ))}
          </div>

          {/*
            Tombol tambah baris — `pyAction=addRow` pada grid Alasan di Pega.

            Baris KOSONG tidak perlu dihapus pengguna: penomoran surat membuangnya tanpa
            melompatkan nomor. Karena itu tidak ada tombol hapus baris di sini, dan di
            Pega pun tidak ada.
          */}
          <Button
            type="button"
            tone="kedua"
            className="mt-2"
            disabled={sedangMenerbitkan}
            onClick={() =>
              setIsian((sebelum) => ({ ...sebelum, alasan: [...sebelum.alasan, ""] }))
            }
          >
            Tambah Alasan
          </Button>
        </fieldset>

        {galat && (
          <div className="mt-4">
            <ErrorMessage
              tone="gangguan"
              title="Surat penolakan gagal dibuat"
              description={galat}
            />
          </div>
        )}

        <div className="mt-6 flex justify-end gap-2">
          <Button
            type="button"
            tone="kedua"
            disabled={sedangMenerbitkan}
            onClick={onBatal}
          >
            Batal
          </Button>
          <Button
            type="button"
            tone="utama"
            disabled={sedangMenerbitkan}
            onClick={() => onTerbitkan(isian)}
          >
            {sedangMenerbitkan ? "Membuat surat…" : "Generate PDF"}
          </Button>
        </div>
      </div>
    </div>
  );
}

/**
 * RejectLetterPrefill adalah tiga isian yang terbawa dari klaim.
 *
 * `AutoFillFormReject_Pre` mengisi EMPAT; yang keempat — Tanggal Keluar Rawat Inap —
 * tidak punya kolom basis data. Lihat kueri `find_reject_prefill` di backend.
 */
export type RejectLetterPrefill = {
  nama_pasien?: string | undefined;
  tempat_kejadian?: string | undefined;
  tanggal_kejadian?: string | undefined;
};
