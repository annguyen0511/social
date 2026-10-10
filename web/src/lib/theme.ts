export type Theme = 'light' | 'dark' | 'system'

// Shared with the inline script in index.html. If one changes, the other has
// to change with it, or a reload shows the wrong theme for a moment and then
// corrects itself.
//
// Dùng chung với đoạn script nội tuyến trong index.html. Đổi một bên thì phải
// đổi cả bên kia, nếu không mỗi lần tải lại trang sẽ hiện sai giao diện trong
// chốc lát rồi mới tự sửa.
export const THEME_KEY = 'theme'

/**
 * Reads the stored preference, defaulting to following the system.
 *
 * Every access is guarded: localStorage throws rather than returning null in
 * a private window or when site data is blocked, and a theme toggle is not
 * worth taking the whole page down for.
 *
 * Đọc lựa chọn đã lưu, mặc định là đi theo hệ thống.
 *
 * Mọi lần truy cập đều được bọc lại: localStorage ném lỗi chứ không trả null
 * khi ở cửa sổ ẩn danh hoặc khi dữ liệu trang bị chặn, mà một cái nút đổi
 * giao diện thì không đáng để kéo sập cả trang.
 */
export function readTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_KEY)
    if (stored === 'light' || stored === 'dark' || stored === 'system') return stored
  } catch {
    // Ignored on purpose, see above. / Cố tình bỏ qua, xem giải thích ở trên.
  }
  return 'system'
}

export function storeTheme(theme: Theme): void {
  try {
    localStorage.setItem(THEME_KEY, theme)
  } catch {
    // The choice still applies to this page, it just will not survive a
    // reload. / Lựa chọn vẫn có hiệu lực với trang này, chỉ là không sống sót
    // qua lần tải lại.
  }
}

export const prefersDark = (): boolean =>
  window.matchMedia('(prefers-color-scheme: dark)').matches

/**
 * Puts the theme on the document.
 *
 * The class goes on <html> rather than <body> because the CSS variables are
 * declared on .dark and everything has to inherit them, including the page
 * background that shows through before the app has mounted.
 *
 * Đặt giao diện lên document.
 *
 * Class nằm trên <html> chứ không phải <body>, vì các biến CSS được khai báo
 * trong .dark và mọi thứ phải kế thừa chúng, kể cả nền trang hiện ra trước
 * khi ứng dụng kịp mount.
 */
export function applyTheme(theme: Theme): void {
  const dark = theme === 'dark' || (theme === 'system' && prefersDark())
  document.documentElement.classList.toggle('dark', dark)
}
