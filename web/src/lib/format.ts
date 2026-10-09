const dateTime = new Intl.DateTimeFormat('vi-VN', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

/**
 * Formats a timestamp the API sent.
 *
 * The API serialises times as RFC 3339 strings in UTC, which Date parses and
 * Intl renders in the reader's own time zone — so a post written at 14:00 in
 * Hanoi reads as 14:00 there and as the right local hour anywhere else.
 *
 * Định dạng một mốc thời gian do API gửi về.
 *
 * API trả thời gian dưới dạng chuỗi RFC 3339 theo giờ UTC, Date đọc được và
 * Intl hiển thị theo múi giờ của chính người xem — nên một bài viết lúc 14:00
 * ở Hà Nội hiện đúng 14:00 ở đó, và đúng giờ địa phương ở nơi khác.
 */
export function formatDateTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : dateTime.format(date)
}

const number = new Intl.NumberFormat('vi-VN')

/**
 * Formats a count. Four figures and up get a separator, so 1234 reads as
 * 1.234 rather than as a number the eye has to parse digit by digit.
 *
 * Định dạng một con số đếm. Từ bốn chữ số trở lên sẽ có dấu phân cách, nên
 * 1234 đọc thành 1.234 thay vì một dãy số mà mắt phải đếm từng chữ.
 */
export const formatCount = (value: number): string => number.format(value)
