import { useEffect, useMemo, useRef } from 'react'
import type { CropArea } from './PostForm'

// Khung xem trước không cần to hơn thứ server sẽ tạo ra, mà một vùng cắt
// 3000×3000 vẽ ra canvas là 36 MB bộ nhớ cho một tấm ảnh chỉ để nhìn.
//
// The preview never needs to be larger than what the server will produce,
// and a 3000×3000 crop on a canvas is 36 MB of memory for a picture that is
// only being looked at.
const MAX_PREVIEW = 1080

/**
 * Shows a chosen file as the crop rectangle will leave it.
 *
 * A canvas, not a div with the picture shifted inside it. Both can draw the
 * right pixels, but only a canvas is a replaced element: it has an intrinsic
 * size, so max-width and max-height scale it down keeping its shape, exactly
 * the way an img behaves. A div carrying only an aspect-ratio has no width of
 * its own — its only child is absolutely positioned and contributes nothing —
 * so it collapses to zero and nothing appears at all.
 *
 * drawImage copies pixels; it does not encode. The file still travels to the
 * server untouched with four coordinates beside it, and the picture is still
 * encoded exactly once, on the server.
 *
 * Hiện file đã chọn đúng như vùng cắt sẽ để lại.
 *
 * Dùng canvas chứ không phải một div với tấm ảnh dịch bên trong. Cả hai đều
 * vẽ ra đúng điểm ảnh, nhưng chỉ canvas là replaced element: nó có kích thước
 * tự thân, nên max-width và max-height thu nhỏ nó mà vẫn giữ hình dạng, đúng
 * cách một thẻ img hành xử. Một div chỉ mang aspect-ratio thì không có bề
 * rộng của riêng nó — con duy nhất bên trong định vị tuyệt đối nên không đóng
 * góp gì — thành ra nó co về 0 và chẳng hiện ra thứ gì.
 *
 * drawImage sao chép điểm ảnh chứ không mã hoá. File vẫn lên server nguyên
 * vẹn kèm bốn toạ độ, và tấm ảnh vẫn chỉ bị mã hoá đúng một lần, ở server.
 */
export function CroppedPreview({
  src,
  crop,
  className,
}: {
  src: string
  crop: CropArea | null
  className?: string
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const valid = crop !== null && crop.width > 0 && crop.height > 0

  // Tính lúc render chứ không phải trong effect, và đặt thẳng lên thuộc tính
  // width/height. Một canvas chưa được gán kích thước mặc định là 300×150,
  // nên để effect gán sau sẽ hiện thoáng một ô sai tỉ lệ rồi mới nhảy về
  // đúng. Gán trong effect còn xoá sạch bitmap, nên nó phải nằm ngoài.
  //
  // Computed during render rather than in the effect, and set straight on the
  // width/height attributes. A canvas with no size set defaults to 300×150,
  // so leaving it to the effect flashes a wrongly shaped box before snapping
  // into place. Assigning in the effect also wipes the bitmap, so it belongs
  // outside it.
  const size = useMemo(() => {
    if (!crop || !valid) return { width: 0, height: 0 }
    const scale = Math.min(1, MAX_PREVIEW / Math.max(crop.width, crop.height))
    return {
      width: Math.max(1, Math.round(crop.width * scale)),
      height: Math.max(1, Math.round(crop.height * scale)),
    }
  }, [crop, valid])

  useEffect(() => {
    if (!valid || !crop) return

    const canvas = canvasRef.current
    if (!canvas) return

    const image = new Image()

    image.onload = () => {
      const context = canvas.getContext('2d')
      if (!context) return

      // Nguồn tính theo điểm ảnh của file gốc — đúng hệ toạ độ mà
      // croppedAreaPixels trả về và CropRect phía server dùng. Ba nơi cùng
      // đọc một bộ số là lý do xem trước khớp với kết quả đăng lên.
      //
      // The source is in the original file's pixels — the same coordinates
      // croppedAreaPixels reports and the server's CropRect cuts from. Three
      // places reading one set of numbers is why the preview matches what
      // gets posted.
      context.drawImage(
        image,
        crop.x,
        crop.y,
        crop.width,
        crop.height,
        0,
        0,
        size.width,
        size.height,
      )
    }

    image.src = src

    // Ảnh có thể về sau khi component đã rời màn hình, hoặc sau khi vùng cắt
    // đã đổi; bỏ handler để lần vẽ cũ không ghi đè lên lần mới.
    //
    // The image can arrive after this has unmounted, or after the crop has
    // changed; dropping the handler keeps a stale draw from landing on top of
    // a newer one.
    return () => {
      image.onload = null
    }
  }, [src, crop, valid, size])

  // Thiếu vùng cắt thì hiện cả tấm ảnh — trung thực hơn là vẽ một khung sai.
  // Without a crop, show the whole picture — honester than drawing a wrong one.
  if (!valid) {
    return <img src={src} alt="" className={className} />
  }

  return <canvas ref={canvasRef} width={size.width} height={size.height} className={className} />
}
