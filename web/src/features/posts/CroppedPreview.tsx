import type { CropArea } from './PostForm'

/**
 * Shows a chosen file as the crop rectangle will leave it, without producing
 * a cropped file.
 *
 * The crop only ever exists as four numbers travelling to the server, so
 * there is no cropped bitmap on this side to show. Drawing one to a canvas
 * just for the preview would re-encode the picture for nothing — the very
 * cost that sending coordinates was meant to avoid.
 *
 * So the window does the cropping. A box with the crop's own aspect ratio
 * clips an oversized copy of the whole picture, shifted so that the kept
 * region lands inside it. Nothing is decoded twice and the result is exact.
 *
 * Hiện file đã chọn đúng như vùng cắt sẽ để lại, mà không tạo ra file đã cắt.
 *
 * Vùng cắt chỉ tồn tại dưới dạng bốn con số đi xuống server, nên phía này
 * không có bitmap đã cắt nào để hiển thị. Vẽ ra canvas chỉ để xem trước là
 * mã hoá lại tấm ảnh một cách vô ích — đúng cái giá mà việc gửi toạ độ sinh
 * ra để tránh.
 *
 * Nên chính khung nhìn làm việc cắt. Một ô mang đúng tỉ lệ của vùng cắt sẽ
 * che bớt một bản phóng to của cả tấm ảnh, dịch đi sao cho phần được giữ rơi
 * vào bên trong nó. Không có gì bị giải mã hai lần và kết quả thì chính xác.
 */
export function CroppedPreview({
  src,
  crop,
  naturalSize,
  className,
}: {
  src: string
  crop: CropArea | null
  naturalSize: { width: number; height: number } | null
  className?: string
}) {
  // Without either piece there is nothing to crop against, so the whole
  // picture is the honest thing to show.
  //
  // Thiếu một trong hai thì không có gì để cắt theo, nên hiện cả tấm ảnh mới
  // là trung thực.
  if (!crop || !naturalSize || crop.width <= 0 || crop.height <= 0) {
    return <img src={src} alt="" className={className} />
  }

  // Percentages, not pixels: the box can be any size the layout gives it and
  // the picture scales with it. Width percentages resolve against the box's
  // width and top/left percentages against its height and width in turn,
  // which is exactly the pair of ratios needed.
  //
  // Dùng phần trăm chứ không phải pixel: cái ô to bao nhiêu tuỳ bố cục, và
  // tấm ảnh co giãn theo. Phần trăm của width tính theo bề rộng của ô, còn
  // top/left tính lần lượt theo chiều cao và bề rộng của nó — đúng bằng cặp
  // tỉ lệ đang cần.
  return (
    <div
      className={className}
      style={{ aspectRatio: `${crop.width} / ${crop.height}`, position: 'relative', overflow: 'hidden' }}
    >
      <img
        src={src}
        alt=""
        style={{
          position: 'absolute',
          width: `${(naturalSize.width / crop.width) * 100}%`,
          height: `${(naturalSize.height / crop.height) * 100}%`,
          left: `${(-crop.x / crop.width) * 100}%`,
          top: `${(-crop.y / crop.height) * 100}%`,
          maxWidth: 'none',
        }}
      />
    </div>
  )
}
