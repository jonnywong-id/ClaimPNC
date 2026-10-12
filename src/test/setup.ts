import '@testing-library/jest-dom/vitest'
import { configure } from '@testing-library/react'

// `findBy*` dan `waitFor` bawaannya menyerah setelah 1 detik. Di runner CI yang sibuk,
// layar yang memuat data tiruan dapat butuh lebih lama dari itu — dan kegagalannya terbaca
// sebagai "elemen tidak ditemukan", bukan sebagai mesin yang lambat.
configure({ asyncUtilTimeout: 5_000 })
