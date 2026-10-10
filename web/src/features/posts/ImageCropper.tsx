import { useState } from 'react'
import Cropper, { type Area, type MediaSize } from 'react-easy-crop'
import { Slider } from '@/components/ui/slider'
import { cn } from 'cn'

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
 */

// null nghĩa là giữ nguyên tỉ lệ của ảnh gốc, và chỉ biết được sau khi ảnh
// tải xong — nên nó phải là một lựa chọn riêng chứ không phải một con số.
//
// null means keep the picture's own ratio, which is only knowable once it has
// loaded — so it has to be its own option rather than a number.
const ratios: { label: string; value: number | null }[] = [
  { label: 'Gốc', value: null },
  { label: '1:1', value: 1 },
  { label: '4:5', value: 4 / 5 },
  { label: '16:9', value: 16 / 9 },
]

export function ImageCropper({
  src,
  onCropped,
}: {
  src: string
  onCropped: (area: Area) => void
}) {
  const [crop, setCrop] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [ratio, setRatio] = useState<number | null>(null)
  const [naturalRatio, setNaturalRatio] = useState(1)

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="relative min-h-0 flex-1 bg-black">
        <Cropper
          image={src}
          crop={crop}
          zoom={zoom}
          // Cropper cần một con số, nên "Gốc" được dịch thành tỉ lệ thật của
          // ảnh — đo được khi nó tải xong.
          //
          // Cropper wants a number, so "Gốc" resolves to the picture's real
          // ratio, measured once it has loaded.
          aspect={ratio ?? naturalRatio}
          minZoom={1}
          maxZoom={4}
          onCropChange={setCrop}
          onZoomChange={setZoom}
          onMediaLoaded={(size: MediaSize) =>
            setNaturalRatio(size.naturalWidth / size.naturalHeight)
          }
          // croppedAreaPixels is already in the original file's coordinates,
          // which is exactly what the server's CropRect expects.
          //
          // croppedAreaPixels vốn đã tính theo toạ độ của file gốc, đúng thứ
          // CropRect phía server đang chờ.
          onCropComplete={(_area, pixels) => onCropped(pixels)}
        />
      </div>

      <div className="flex shrink-0 flex-wrap items-center gap-4 border-t border-border p-3">
        <div className="flex gap-1">
          {ratios.map(({ label, value }) => (
            <button
              key={label}
              type="button"
              aria-pressed={ratio === value}
              onClick={() => setRatio(value)}
              className={cn(
                'rounded-md px-2.5 py-1 text-xs transition-colors',
                ratio === value ? 'bg-foreground text-background' : 'hover:bg-muted',
              )}
            >
              {label}
            </button>
          ))}
        </div>

        <label className="flex min-w-40 flex-1 items-center gap-2 text-xs text-muted-foreground">
          Thu phóng
          <Slider
            value={[zoom]}
            min={1}
            max={4}
            step={0.01}
            onValueChange={([next]) => setZoom(next)}
            aria-label="Thu phóng"
            className="flex-1"
          />
        </label>
      </div>
    </div>
  )
}
