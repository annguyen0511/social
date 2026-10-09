import { useEffect, useState } from 'react'

/**
 * Returns value only after it has stopped changing for delay milliseconds.
 *
 * Used to keep a search box from turning every keystroke into a request. The
 * debounced value is what goes into the queryKey, so Query sees one key per
 * pause in typing instead of one per character.
 *
 * Trả về value chỉ sau khi nó ngừng thay đổi trong delay milisecond.
 *
 * Dùng để một ô tìm kiếm không biến mỗi lần gõ thành một request. Giá trị đã
 * debounce mới là thứ đưa vào queryKey, nên Query chỉ thấy một key cho mỗi
 * lần người dùng ngừng gõ, thay vì một key cho mỗi ký tự.
 */
export function useDebounced<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    // Clearing on every change is what makes this a debounce rather than a
    // delay: a keystroke arriving before the timer fires cancels it.
    //
    // Việc dọn timer ở mỗi lần thay đổi mới làm nên debounce chứ không phải
    // một khoảng chờ: một lần gõ đến trước khi timer chạy sẽ huỷ timer đó.
    return () => clearTimeout(id)
  }, [value, delay])

  return debounced
}
