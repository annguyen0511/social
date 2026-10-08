/// <reference types="vite/client" />

// Declaring the variables the app reads gives import.meta.env a real type, so
// a typo in the name is a compile error instead of undefined at runtime.
//
// Khai báo các biến mà ứng dụng đọc giúp import.meta.env có kiểu thật, nên gõ
// sai tên là lỗi biên dịch chứ không phải undefined lúc chạy.
interface ImportMetaEnv {
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
