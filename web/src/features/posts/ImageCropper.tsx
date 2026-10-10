import Cropper, { type Area } from 'react-easy-crop'
import { Slider } from '@/components/ui/slider'
import { cn } from 'cn'
import type { ImageDraft } from './PostForm'

/**
 * The crop frame over a chosen picture.
 *
 * The frame stays still and the picture moves behind it, which is both what
 * people expect and the easier model: the whole state is an offset, a zoom
 * and an aspect ratio, and the result is one rectangle in the original
 * image's own pixels.
 *
 * That rectangle is all that gets sent. The file goes up untouched and the
 * server cuts from it, so the picture is encoded exactly once instead of
 * once by the browser and again by the server.
 *
 * Nothing is held here. Every piece of state lives on the draft and comes
 * back down as props, because a post can carry several pictures now and
 * flipping between two of them must not disturb either frame — state kept
 * inside would reset on each remount.
 *
 * Khung cắt đặt lên tấm ảnh đã chọn.
 *
 * Khung đứng yên còn ảnh di chuyển phía sau, vừa đúng thứ người ta mong đợi
 * vừa là mô hình dễ hơn: toàn bộ trạng thái chỉ là một độ lệch, một mức zoom
 * và một tỉ lệ khung, còn kết quả là một hình chữ nhật tính theo đúng điểm
 * ảnh của bản gốc.
 *
 * Chỉ hình chữ nhật đó được gửi đi. File gốc lên nguyên vẹn và server cắt từ
 * nó, nên tấm ảnh chỉ bị mã hoá đúng một lần thay vì một lần ở trình duyệt
 * rồi một lần nữa ở server.
 *
 * Component này không giữ gì cả. Mọi trạng thái nằm trên bản nháp rồi truyền
 * xuống làm props, vì giờ một bài có thể mang nhiều ảnh và việc lật qua lại
 * giữa hai tấm không được làm xê dịch khung nào — trạng thái giữ bên trong
 * sẽ reset mỗi lần component được dựng lại.
 */

// null nghĩa là giữ nguyên tỉ lệ của ảnh gốc. Tỉ lệ thật đo được từ kích
// thước đã đọc lúc chọn file, nên nó vẫn là một lựa chọn riêng chứ không
// phải một con số.
//
// null means keep the picture's own ratio. The real ratio comes from the
// size read when the file was picked, so this stays its own option rather
// than a number.
const ratios: { label: string; value: number | null }[] = [
  { label: 'Gốc', value: null },
  { label: '1:1', value: 1 },
  { label: '4:5', value: 4 / 5 },
  { label: '16:9', value: 16 / 9 },
]

export function ImageCropper({
  src,
  draft,
  onChange,
}: {
  src: string
  draft: ImageDraft
  onChange: (next: ImageDraft) => void
}) {
  const { view, size } = draft
  const naturalRatio = size.width / size.height

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="relative min-h-0 flex-1 bg-black">
        <Cropper
          image={src}
          crop={{ x: view.x, y: view.y }}
          zoom={view.zoom}
          // Cropper cần một con số, nên "Gốc" được dịch thành tỉ lệ thật của
          // ảnh.
          //
          // Cropper wants a number, so "Gốc" resolves to the picture's real
          // ratio.
          aspect={view.ratio ?? naturalRatio}
          minZoom={1}
          maxZoom={4}
          onCropChange={({ x, y }) => onChange({ ...draft, view: { ...view, x, y } })}
          onZoomChange={(zoom) => onChange({ ...draft, view: { ...view, zoom } })}
          // croppedAreaPixels is already in the original file's coordinates,
          // which is exactly what the server's CropRect expects.
          //
          // croppedAreaPixels vốn đã tính theo toạ độ của file gốc, đúng thứ
          // CropRect phía server đang chờ.
          onCropComplete={(_area: Area, pixels: Area) => onChange({ ...draft, crop: pixels })}
        />
      </div>

      <div className="flex shrink-0 flex-wrap items-center gap-4 border-t border-border p-3">
        <div className="flex gap-1">
          {ratios.map(({ label, value }) => (
            <button
              key={label}
              type="button"
              aria-pressed={view.ratio === value}
              onClick={() => onChange({ ...draft, view: { ...view, ratio: value } })}
              className={cn(
                'rounded-md px-2.5 py-1 text-xs transition-colors',
                view.ratio === value ? 'bg-foreground text-background' : 'hover:bg-muted',
              )}
            >
              {label}
            </button>
          ))}
        </div>

        <label className="flex min-w-40 flex-1 items-center gap-2 text-xs text-muted-foreground">
          Thu phóng
          <Slider
            value={[view.zoom]}
            min={1}
            max={4}
            step={0.01}
            onValueChange={([zoom]) => onChange({ ...draft, view: { ...view, zoom } })}
            aria-label="Thu phóng"
            className="flex-1"
          />
        </label>
      </div>
    </div>
  )
}
