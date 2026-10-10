import { useEffect, useMemo, useRef, useState } from 'react'
import { ImagePlus, Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ImageCropper } from './ImageCropper'
import type { CropArea } from './PostForm'
import { cn } from 'cn'

// Mirrors upload.MaxBytes and the formats image.Decode is set up to read.
// Checking here only saves a doomed round trip; the server re-encodes
// whatever arrives and stays the authority.
//
// Khớp với upload.MaxBytes và các định dạng mà image.Decode được cài để đọc.
// Kiểm ở đây chỉ để đỡ một lượt gọi mạng chắc chắn thất bại; server mã hoá
// lại mọi thứ nhận được và vẫn là nơi quyết định.
const MAX_BYTES = 5 * 1024 * 1024
const ACCEPT = 'image/jpeg,image/png,image/gif'

/**
 * The first step of writing a post: pick the picture, or decide not to.
 *
 * A post here does not need a picture, which is the one place this parts ways
 * with Instagram. So the empty state offers both doors, and the one that
 * skips the picture is a quiet link rather than a second big button — most
 * people arriving at this screen came to post a photo.
 *
 * Bước đầu khi viết bài: chọn ảnh, hoặc quyết định không chọn.
 *
 * Bài viết ở đây không bắt buộc có ảnh, và đó là điểm duy nhất khác
 * Instagram. Nên màn hình rỗng mở sẵn cả hai cửa, mà cửa bỏ qua ảnh là một
 * đường dẫn nhỏ chứ không phải cái nút to thứ hai — phần lớn người tới màn
 * hình này là để đăng ảnh.
 */
export function ImagePicker({
  image,
  error,
  onPick,
  onClear,
  onSkip,
  onCrop,
}: {
  image: File | null
  error: string | null
  onPick: (file: File) => void
  onClear: () => void
  onSkip: () => void
  onCrop: (area: CropArea) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  // A blob: URL is a reference the browser holds until it is revoked, so the
  // old one has to go whenever the file changes or this unmounts.
  //
  // blob: URL là một tham chiếu trình duyệt giữ lại cho tới khi bị thu hồi,
  // nên phải bỏ cái cũ mỗi lần đổi file hoặc khi component rời màn hình.
  const preview = useMemo(() => (image ? URL.createObjectURL(image) : null), [image])
  useEffect(() => {
    if (!preview) return
    return () => URL.revokeObjectURL(preview)
  }, [preview])

  const choose = (file: File | undefined) => {
    if (file) onPick(file)
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPT}
        className="sr-only"
        onChange={(event) => {
          choose(event.target.files?.[0])
          // Chọn lại đúng file vừa bỏ sẽ không bắn onChange nếu giá trị của
          // input còn nguyên.
          //
          // Picking the very same file again fires no onChange unless the
          // input's value is cleared.
          event.target.value = ''
        }}
      />

      {preview ? (
        <ImageCropper src={preview} onCropped={onCrop} />
      ) : (
        // Thả file vào là cách người ta mong đợi ở một màn hình như thế này,
        // và nó không thay thế nút bấm mà chỉ thêm một lối vào.
        //
        // Dropping a file is what people expect of a screen like this, and it
        // adds a way in rather than replacing the button.
        <div
          onDragOver={(event) => {
            event.preventDefault()
            setDragging(true)
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(event) => {
            event.preventDefault()
            setDragging(false)
            choose(event.dataTransfer.files?.[0])
          }}
          className={cn(
            'grid min-h-0 flex-1 place-items-center p-6 transition-colors',
            dragging && 'bg-muted',
          )}
        >
          <div className="space-y-4 text-center">
            <ImagePlus className="mx-auto size-16 text-muted-foreground" strokeWidth={1} />
            <div>
              <p className="font-medium">Kéo ảnh vào đây</p>
              <p className="text-sm text-muted-foreground">JPEG, PNG hoặc GIF, tối đa 5 MB</p>
            </div>
            <Button type="button" onClick={() => inputRef.current?.click()}>
              Chọn ảnh từ máy
            </Button>
            <p>
              <button
                type="button"
                onClick={onSkip}
                className="text-sm text-muted-foreground underline underline-offset-2 hover:text-foreground"
              >
                Đăng bài không có ảnh
              </button>
            </p>
          </div>
        </div>
      )}

      {error && <p className="px-4 pb-2 text-center text-sm text-destructive">{error}</p>}

      {image && (
        <div className="flex shrink-0 items-center gap-2 border-t border-border p-3">
          <div className="relative">
            <img
              src={preview ?? ''}
              alt=""
              className="size-16 rounded-md border border-border object-cover"
            />
            <button
              type="button"
              onClick={onClear}
              aria-label="Bỏ ảnh"
              className="absolute -top-1.5 -right-1.5 grid size-5 place-items-center rounded-full bg-foreground text-background"
            >
              <X className="size-3" />
            </button>
          </div>

          {/* Ô thêm ảnh để sẵn chỗ cho nhiều ảnh, hiện đang vô hiệu vì mỗi
              bài mới chỉ mang được một tấm. Vẽ ra ở trạng thái khoá thay vì
              giấu đi, để chỗ của nó rõ ràng khi mở ra sau này.

              The add slot holds the place for multiple pictures, disabled for
              now because a post carries only one. Drawn locked rather than
              hidden, so where it goes is obvious when it opens up later. */}
          <button
            type="button"
            disabled
            title="Nhiều ảnh — sắp có"
            className="grid size-16 cursor-not-allowed place-items-center rounded-md border border-dashed border-border text-muted-foreground opacity-50"
          >
            <Plus />
          </button>
        </div>
      )}
    </div>
  )
}

export const validateImage = (file: File): string | null => {
  if (!file.type.startsWith('image/')) return 'Hãy chọn một file ảnh.'
  if (file.size > MAX_BYTES) return 'Ảnh phải nhỏ hơn 5 MB.'
  return null
}
