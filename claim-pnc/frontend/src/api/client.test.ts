import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  APIError,
  HEADER_PORTAL,
  NetworkError,
  blobAPI,
  callAPI,
  downloadAPI,
  simpanBerkas,
  unduhBerkas,
  uploadAPI,
} from './client'

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

/** Memasang fetch tiruan yang mencatat setiap panggilan lalu menjawab sesuai `reply`. */
function installFetch(reply: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    return Promise.resolve(reply(url, init))
  })
}

function failFetch() {
  vi.stubGlobal('fetch', () => Promise.reject(new TypeError('Failed to fetch')))
}

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  })
}

function headersOf(call: Call | undefined): Record<string, string> {
  return (call?.init?.headers ?? {}) as Record<string, string>
}

/** Menangkap galat dari janji yang diharapkan gagal. */
async function catchError(promise: Promise<unknown>): Promise<unknown> {
  try {
    await promise
  } catch (error) {
    return error
  }
  throw new Error('janji tidak gagal')
}

let createObjectURL: ReturnType<typeof vi.fn>
let revokeObjectURL: ReturnType<typeof vi.fn>
let clicked: { href: string; download: string }[]

beforeEach(() => {
  calls = []
  clicked = []
  // jsdom tidak menyediakan URL objek; keduanya ditiru supaya unduhan dapat diamati.
  createObjectURL = vi.fn(() => 'blob:contoh')
  revokeObjectURL = vi.fn()
  Object.assign(URL, { createObjectURL, revokeObjectURL })
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push({ href: this.href, download: this.download })
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('APIError.violations', () => {
  it('menyatukan peta field dan senarai detail berkunci field maupun kolom', () => {
    const error = new APIError(
      'validasi_gagal',
      'ditolak',
      422,
      [
        { field: 'nama', pesan: 'nama wajib' },
        { kolom: 'kode', pesan: 'kode wajib' },
        // Butir tanpa nama kolom tidak dapat ditempatkan di mana pun, sehingga dibuang.
        { pesan: 'tanpa kolom' },
      ],
      { email: 'email salah' },
    )

    expect(error.name).toBe('APIError')
    expect(error.status).toBe(422)
    expect(error.violations()).toEqual({
      email: 'email salah',
      nama: 'nama wajib',
      kode: 'kode wajib',
    })
  })

  it('mengosongkan detail dan field bila tidak diberikan', () => {
    const error = new APIError('galat', 'pesan', 500)
    expect(error.detail).toEqual([])
    expect(error.field).toEqual({})
    expect(error.violations()).toEqual({})
  })
})

describe('NetworkError', () => {
  it('membawa pesan baku dan nama kelasnya', () => {
    const error = new NetworkError()
    expect(error.name).toBe('NetworkError')
    expect(error.message).toBe('Tidak dapat menghubungi server Claim PNC.')
  })
})

describe('callAPI', () => {
  it('mengirim GET tanpa badan, dengan token dan portal di header', async () => {
    installFetch(() => jsonResponse(200, { isi: 1 }))

    const result = await callAPI<{ isi: number }>('/api/contoh', { token: 'tkn', portal: 'ASM' })

    expect(result).toEqual({ isi: 1 })
    expect(calls[0]?.init?.method).toBe('GET')
    expect(calls[0]?.init?.body).toBeNull()
    expect(headersOf(calls[0])).toEqual({
      Accept: 'application/json',
      Authorization: 'Bearer tkn',
      [HEADER_PORTAL]: 'ASM',
    })
  })

  it('tidak menulis Authorization maupun X-Portal bila keduanya kosong', async () => {
    installFetch(() => jsonResponse(200, {}))

    await callAPI('/api/contoh', { token: null, portal: null })

    expect(headersOf(calls[0])).toEqual({ Accept: 'application/json' })
  })

  it('mengubah badan menjadi JSON dan menyetel Content-Type', async () => {
    installFetch(() => jsonResponse(201, { ok: true }))

    await callAPI('/api/contoh', { metode: 'POST', body: { a: 1 } })

    expect(calls[0]?.init?.method).toBe('POST')
    expect(calls[0]?.init?.body).toBe('{"a":1}')
    expect(headersOf(calls[0])['Content-Type']).toBe('application/json')
  })

  it('mengirim FormData apa adanya tanpa Content-Type', async () => {
    installFetch(() => jsonResponse(200, {}))
    const form = new FormData()
    form.append('x', 'y')

    await callAPI('/api/contoh', { metode: 'PUT', body: form })

    expect(calls[0]?.init?.body).toBe(form)
    expect(headersOf(calls[0])['Content-Type']).toBeUndefined()
  })

  it('mengembalikan undefined untuk 204', async () => {
    installFetch(() => new Response(null, { status: 204 }))

    await expect(callAPI('/api/contoh', { metode: 'DELETE' })).resolves.toBeUndefined()
  })

  it('mengembalikan null untuk badan sukses yang kosong atau bukan JSON', async () => {
    installFetch((url) =>
      url === '/kosong' ? new Response('   ', { status: 200 }) : new Response('<html>', { status: 200 }),
    )

    await expect(callAPI('/kosong')).resolves.toBeNull()
    await expect(callAPI('/html')).resolves.toBeNull()
  })

  it('melempar APIError lengkap dengan detail dan field yang disaring', async () => {
    installFetch(() =>
      jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Isian belum benar.',
        detail: [{ field: 'nama', pesan: 'wajib' }],
        // Nilai bukan teks dibuang satu per satu, bukan seluruh peta.
        field: { email: 'salah', angka: 5 },
      }),
    )

    const error = (await catchError(callAPI('/api/contoh'))) as APIError

    expect(error).toBeInstanceOf(APIError)
    expect(error.kode).toBe('validasi_gagal')
    expect(error.message).toBe('Isian belum benar.')
    expect(error.status).toBe(422)
    expect(error.detail).toEqual([{ field: 'nama', pesan: 'wajib' }])
    expect(error.field).toEqual({ email: 'salah' })
  })

  it('memakai kode dan pesan baku bila badan galat kosong', async () => {
    installFetch(() => new Response('', { status: 500 }))

    const error = (await catchError(callAPI('/api/contoh'))) as APIError

    expect(error.kode).toBe('galat_internal')
    expect(error.message).toBe('Terjadi kesalahan pada sistem.')
    expect(error.detail).toEqual([])
    expect(error.field).toEqual({})
  })

  it('menolak detail bukan senarai dan field berbentuk senarai', async () => {
    installFetch(() => jsonResponse(400, { kode: 'x', detail: 'bukan senarai', field: ['a'] }))

    const error = (await catchError(callAPI('/api/contoh'))) as APIError

    expect(error.detail).toEqual([])
    expect(error.field).toEqual({})
  })

  it('melempar NetworkError ketika fetch gagal', async () => {
    failFetch()

    await expect(callAPI('/api/contoh')).rejects.toBeInstanceOf(NetworkError)
  })
})

describe('downloadAPI', () => {
  it('mengunduh blob dengan nama berkas lalu melepas URL objeknya', async () => {
    installFetch(() => new Response('isi,csv', { status: 200 }))

    await downloadAPI('/api/berkas', 'data.csv', { token: 'tkn', portal: 'ASM' })

    expect(headersOf(calls[0])).toEqual({ Authorization: 'Bearer tkn', [HEADER_PORTAL]: 'ASM' })
    expect(clicked).toEqual([{ href: 'blob:contoh', download: 'data.csv' }])
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:contoh')
  })

  it('tidak mengirim header apa pun tanpa token dan portal', async () => {
    installFetch(() => new Response('x', { status: 200 }))

    await downloadAPI('/api/berkas', 'a.csv')

    expect(headersOf(calls[0])).toEqual({})
  })

  it('melempar APIError dengan kode server atau pesan baku', async () => {
    installFetch((url) =>
      url === '/kode'
        ? jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
        : new Response('', { status: 500 }),
    )

    const withCode = (await catchError(downloadAPI('/kode', 'a'))) as APIError
    expect(withCode.kode).toBe('tidak_ditemukan')
    expect(withCode.message).toBe('Tidak ada.')
    expect(withCode.status).toBe(404)

    const plain = (await catchError(downloadAPI('/kosong', 'a'))) as APIError
    expect(plain.kode).toBe('galat_internal')
    expect(plain.message).toBe('Berkas tidak dapat diunduh.')
    expect(clicked).toEqual([])
  })

  it('melempar NetworkError ketika fetch gagal', async () => {
    failFetch()
    await expect(downloadAPI('/api/berkas', 'a')).rejects.toBeInstanceOf(NetworkError)
  })
})

describe('blobAPI', () => {
  it('membaca nama berkas dan sifat inline dari Content-Disposition', async () => {
    installFetch(
      () =>
        new Response('pdf', {
          status: 200,
          headers: { 'Content-Disposition': `inline; filename*=UTF-8''bukti%20bayar.pdf` },
        }),
    )

    const result = await blobAPI('/api/blob', { token: 'tkn', portal: 'ASM' })

    expect(headersOf(calls[0])).toEqual({ Authorization: 'Bearer tkn', [HEADER_PORTAL]: 'ASM' })
    expect(result.nama).toBe('bukti bayar.pdf')
    expect(result.bolehDitampilkan).toBe(true)
    expect(await result.isi.text()).toBe('pdf')
  })

  it('menganggap berkas tanpa Content-Disposition sebagai unduhan tanpa nama', async () => {
    installFetch(() => new Response('xlsx', { status: 200 }))

    const result = await blobAPI('/api/blob')

    expect(headersOf(calls[0])).toEqual({})
    expect(result.nama).toBe('')
    expect(result.bolehDitampilkan).toBe(false)
  })

  it('menganggap attachment sebagai unduhan dan tetap membaca namanya', async () => {
    installFetch(
      () =>
        new Response('x', {
          status: 200,
          headers: { 'Content-Disposition': 'attachment; filename="data.xlsx"' },
        }),
    )

    const result = await blobAPI('/api/blob')

    expect(result.nama).toBe('data.xlsx')
    expect(result.bolehDitampilkan).toBe(false)
  })

  it('melempar APIError dengan kode server atau pesan baku', async () => {
    installFetch((url) =>
      url === '/kode'
        ? jsonResponse(404, { kode: 'dokumen_tidak_ditemukan', pesan: 'Hilang.' })
        : new Response('', { status: 502 }),
    )

    const withCode = (await catchError(blobAPI('/kode'))) as APIError
    expect(withCode.kode).toBe('dokumen_tidak_ditemukan')
    expect(withCode.message).toBe('Hilang.')

    const plain = (await catchError(blobAPI('/kosong'))) as APIError
    expect(plain.kode).toBe('galat_internal')
    expect(plain.message).toBe('Berkas tidak dapat dibuka.')
    expect(plain.status).toBe(502)
  })

  it('melempar NetworkError ketika fetch gagal', async () => {
    failFetch()
    await expect(blobAPI('/api/blob')).rejects.toBeInstanceOf(NetworkError)
  })
})

describe('uploadAPI', () => {
  const berkas = new File(['isi'], 'bukti.pdf', { type: 'application/pdf' })

  it('mengirim berkas dan keterangan sebagai multipart dengan header sesi', async () => {
    installFetch(() => jsonResponse(201, { id_dokumen: 'D-1' }))

    const result = await uploadAPI<{ id_dokumen: string }>('/api/unggah', {
      berkas,
      keterangan: 'bukti transfer',
      token: 'tkn',
      portal: 'ASM',
    })

    expect(result).toEqual({ id_dokumen: 'D-1' })
    expect(calls[0]?.init?.method).toBe('POST')
    const form = calls[0]?.init?.body as FormData
    expect((form.get('berkas') as File).name).toBe('bukti.pdf')
    expect(form.get('keterangan')).toBe('bukti transfer')
    expect(headersOf(calls[0])).toEqual({
      Accept: 'application/json',
      Authorization: 'Bearer tkn',
      [HEADER_PORTAL]: 'ASM',
    })
  })

  it('tidak menyertakan keterangan maupun header sesi bila kosong', async () => {
    installFetch(() => jsonResponse(200, {}))

    await uploadAPI('/api/unggah', { berkas })

    const form = calls[0]?.init?.body as FormData
    expect(form.has('keterangan')).toBe(false)
    expect(headersOf(calls[0])).toEqual({ Accept: 'application/json' })
  })

  it('melempar APIError dengan detail dan field dari server', async () => {
    installFetch(() =>
      jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Berkas ditolak.',
        detail: [{ kolom: 'berkas', pesan: 'terlalu besar' }],
        field: { bukti_bayar: 'jenis tidak dikenal' },
      }),
    )

    const error = (await catchError(uploadAPI('/api/unggah', { berkas }))) as APIError

    expect(error.kode).toBe('validasi_gagal')
    expect(error.violations()).toEqual({
      bukti_bayar: 'jenis tidak dikenal',
      berkas: 'terlalu besar',
    })
  })

  it('memakai kode dan pesan baku bila badan galat kosong', async () => {
    installFetch(() => new Response('', { status: 500 }))

    const error = (await catchError(uploadAPI('/api/unggah', { berkas }))) as APIError

    expect(error.kode).toBe('galat_internal')
    expect(error.message).toBe('Terjadi kesalahan pada sistem.')
    expect(error.detail).toEqual([])
  })

  it('melempar NetworkError ketika fetch gagal', async () => {
    failFetch()
    await expect(uploadAPI('/api/unggah', { berkas })).rejects.toBeInstanceOf(NetworkError)
  })
})

describe('unduhBerkas', () => {
  it('mengirim GET tanpa badan dan membaca nama berkas yang dibersihkan', async () => {
    installFetch(
      () =>
        new Response('a,b', {
          status: 200,
          headers: { 'Content-Disposition': 'attachment; filename="../laporan/ekspor.csv"' },
        }),
    )

    const file = await unduhBerkas('/api/ekspor', { token: 'tkn', portal: 'ASM' })

    expect(calls[0]?.init?.method).toBe('GET')
    expect(calls[0]?.init?.body).toBeNull()
    expect(headersOf(calls[0])).toEqual({ Authorization: 'Bearer tkn', [HEADER_PORTAL]: 'ASM' })
    // Pemisah jalur dibuang supaya simpanan tidak dapat diarahkan keluar folder Unduhan.
    expect(file.namaBerkas).toBe('..laporanekspor.csv')
    expect(await file.blob.text()).toBe('a,b')
  })

  it('mengirim metode dan badan JSON untuk unduhan yang berupa tindakan', async () => {
    installFetch(() => new Response('x', { status: 200 }))

    await unduhBerkas('/api/ekspor', { metode: 'POST', body: { revisi: 2 } })

    expect(calls[0]?.init?.method).toBe('POST')
    expect(calls[0]?.init?.body).toBe('{"revisi":2}')
    expect(headersOf(calls[0])).toEqual({ 'Content-Type': 'application/json' })
  })

  it('memakai nama cadangan bila header kosong, tidak cocok, atau namanya hanya pemisah', async () => {
    const dispositions = [null, 'attachment', 'attachment; filename="/"']
    let index = 0
    installFetch(() => {
      const value = dispositions[index++]
      return new Response('x', {
        status: 200,
        headers: value ? { 'Content-Disposition': value } : {},
      })
    })

    for (let i = 0; i < dispositions.length; i++) {
      const file = await unduhBerkas('/api/ekspor')
      expect(file.namaBerkas).toBe('unduhan.csv')
    }
  })

  it('melempar APIError dengan kode server atau pesan baku', async () => {
    installFetch((url) =>
      url === '/kode'
        ? jsonResponse(403, { kode: 'dilarang', pesan: 'Tidak berwenang.' })
        : new Response('', { status: 500 }),
    )

    const withCode = (await catchError(unduhBerkas('/kode'))) as APIError
    expect(withCode.kode).toBe('dilarang')
    expect(withCode.status).toBe(403)

    const plain = (await catchError(unduhBerkas('/kosong'))) as APIError
    expect(plain.kode).toBe('galat_internal')
    expect(plain.message).toBe('Berkas tidak dapat diunduh.')
  })

  it('melempar NetworkError ketika fetch gagal', async () => {
    failFetch()
    await expect(unduhBerkas('/api/ekspor')).rejects.toBeInstanceOf(NetworkError)
  })
})

describe('simpanBerkas', () => {
  it('memicu unduhan bernama lalu membuang tautan dan URL objeknya', () => {
    simpanBerkas({ namaBerkas: 'ekspor.csv', blob: new Blob(['a']) })

    expect(createObjectURL).toHaveBeenCalledTimes(1)
    expect(clicked).toEqual([{ href: 'blob:contoh', download: 'ekspor.csv' }])
    expect(document.body.querySelector('a')).toBeNull()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:contoh')
  })
})
