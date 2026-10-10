import { useEffect, useMemo, useRef, useState } from 'react'
import { ImagePlus, Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ImageCropper } from './ImageCropper'
import { MAX_IMAGES, newImageDraft, type ImageDraft } from './PostForm'
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
 * The first step of writing a post: pick the pictures, or decide not to.
 *
 * A post here does not need a picture, which is the one place this parts ways
 * with Instagram. So the empty state offers both doors, and the one that
 * skips the picture is a quiet link rather than a second big button — most
 * people arriving at this screen came to post a photo.
 *
 * With several pictures chosen, one is being cropped and the rest wait in the
 * strip below. The strip is both the thing that says how many there are and
 * the way to move between them, so there is no separate switcher to learn.
 *
 * Bước đầu khi viết bài: chọn ảnh, hoặc quyết định không chọn.
 *
 * Bài viết ở đây không bắt buộc có ảnh, và đó là điểm duy nhất khác
 * Instagram. Nên màn hình rỗng mở sẵn cả hai cửa, mà cửa bỏ qua ảnh là một
 * đường dẫn nhỏ chứ không phải cái nút to thứ hai — phần lớn người tới màn
 * hình này là để đăng ảnh.
 *
 * Khi đã chọn nhiều ảnh, một tấm đang được cắt còn những tấm còn lại chờ ở
 * dải bên dưới. Dải đó vừa là thứ cho biết có bao nhiêu ảnh vừa là cách
 * chuyển giữa chúng, nên không có bộ chuyển nào khác phải học.
 */
export function ImagePicker({
  images,
  error,
  onChange,
  onError,
  onSkip,
}: {
  images: ImageDraft[]
  error: string | null
  // Nhận một hàm cập nhật chứ không phải mảng đã dựng xong: ảnh được thêm
  // sau khi đọc xong kích thước, nên một mảng dựng từ `images` của lần
  // render cũ sẽ bỏ mất lượt chọn chen vào giữa.
  //
  // Takes an updater rather than a finished array: pictures are added after
  // their size has been read, so an array built from the `images` of an
  // older render would drop a pick that landed in between.
  onChange: (update: (previous: ImageDraft[]) => ImageDraft[]) => void
  onError: (message: string | null) => void
  onSkip: () => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [activeId, setActiveId] = useState<string | null>(null)

  // Tấm đang cắt. Giữ theo id chứ không theo chỉ số, để bỏ một ảnh phía
  // trước không lặng lẽ nhảy sang ảnh khác.
  //
  // The one being cropped. Tracked by id rather than index, so removing a
  // picture before it does not quietly jump to a different one.
  const active = images.find((image) => image.id === activeId) ?? images[0] ?? null

  // Mỗi blob: URL là một tham chiếu trình duyệt giữ tới khi bị thu hồi. Dựng
  // cả bảng một lần rồi thu hồi theo từng cái đã rời danh sách, chứ không
  // dựng lại tất cả mỗi lần thêm một ảnh.
  //
  // Each blob: URL is a reference the browser holds until it is revoked.
  // Built as one table and revoked per entry that has left the list, rather
  // than rebuilt wholesale every time a picture is added.
  const urls = useMemo(() => {
    const table = new Map<string, string>()
    for (const image of images) table.set(image.id, URL.createObjectURL(image.file))
    return table
  }, [images])

  useEffect(() => () => urls.forEach((url) => URL.revokeObjectURL(url)), [urls])

  const add = (files: FileList | null | undefined) => {
    if (!files || files.length === 0) return

    const room = MAX_IMAGES - images.length
    if (room <= 0) {
      onError(`Một bài chỉ mang được tối đa ${MAX_IMAGES} ảnh.`)
      return
    }

    // Báo lỗi cho cả nhóm thay vì cho từng file: người dùng vừa chọn một
    // lượt, nên một dòng nói rõ chuyện gì xảy ra là đủ.
    //
    // One message for the batch rather than one per file: the user picked
    // them in a single go, so a single line saying what happened is enough.
    const chosen = Array.from(files)
    const tooMany = chosen.length > room
    const accepted: File[] = []
    let rejected: string | null = null

    for (const file of chosen.slice(0, room)) {
      const problem = validateImage(file)
      if (problem) rejected = problem
      else accepted.push(file)
    }

    onError(
      rejected ??
        (tooMany ? `Một bài chỉ mang được tối đa ${MAX_IMAGES} ảnh, nên chỉ ${room} tấm được thêm.` : null),
    )
    if (accepted.length === 0) return

    // Kích thước thật phải biết trước khi dựng bản nháp: nó là vùng cắt khởi
    // đầu, và là tỉ lệ mà lựa chọn "Gốc" dùng.
    //
    // The natural size has to be known before the draft exists: it is the
    // starting crop, and the ratio the "Gốc" option uses.
    Promise.all(accepted.map(readSize)).then((sizes) => {
      const drafts = accepted.map((file, i) => newImageDraft(file, sizes[i]))
      // Cắt lại theo danh sách mới nhất, không theo phép đo lúc bấm: giữa
      // hai thời điểm đó có thể đã có lượt khác thêm ảnh vào.
      //
      // Trimmed against the newest list rather than the count measured at
      // click time: another pick may have added pictures in between.
      onChange((previous) => [...previous, ...drafts].slice(0, MAX_IMAGES))

      // Nhảy tới tấm đầu trong nhóm vừa thêm: đó là thứ người dùng vừa làm,
      // nên nó phải là thứ đang hiện ra.
      //
      // Jump to the first of the batch just added: it is what the user just
      // did, so it is what should be on screen.
      setActiveId(drafts[0].id)
    })
  }

  const remove = (id: string) => {
    onChange((previous) => previous.filter((image) => image.id !== id))
    onError(null)
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPT}
        multiple
        className="sr-only"
        onChange={(event) => {
          add(event.target.files)
          // Chọn lại đúng file vừa bỏ sẽ không bắn onChange nếu giá trị của
          // input còn nguyên.
          //
          // Picking the very same file again fires no onChange unless the
          // input's value is cleared.
          event.target.value = ''
        }}
      />

      {active ? (
        <ImageCropper
          // Khoá theo id để react-easy-crop dựng lại khi đổi ảnh. Trạng thái
          // khung nằm trên bản nháp nên không mất gì, còn thiếu khoá thì
          // cropper giữ lại phép đo của ảnh cũ.
          //
          // Keyed by id so react-easy-crop remounts when the picture
          // changes. The frame's state lives on the draft so nothing is
          // lost, while without the key the cropper keeps the old picture's
          // measurements.
          key={active.id}
          src={urls.get(active.id) ?? ''}
          draft={active}
          onChange={(next) =>
            onChange((previous) =>
              previous.map((image) => (image.id === next.id ? next : image)),
            )
          }
        />
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
            add(event.dataTransfer.files)
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
              <p className="text-sm text-muted-foreground">
                JPEG, PNG hoặc GIF, tối đa 5 MB mỗi ảnh và {MAX_IMAGES} ảnh một bài
              </p>
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

      {images.length > 0 && (
        <div className="flex shrink-0 items-center gap-2 overflow-x-auto border-t border-border p-3">
          {images.map((image, index) => (
            <div key={image.id} className="relative shrink-0">
              <button
                type="button"
                onClick={() => setActiveId(image.id)}
                aria-current={image.id === active?.id}
                aria-label={`Ảnh ${index + 1}`}
                className={cn(
                  'block size-16 overflow-hidden rounded-md border-2 transition-colors',
                  image.id === active?.id ? 'border-foreground' : 'border-transparent',
                )}
              >
                <img src={urls.get(image.id) ?? ''} alt="" className="size-full object-cover" />
              </button>
              <button
                type="button"
                onClick={() => remove(image.id)}
                aria-label={`Bỏ ảnh ${index + 1}`}
                className="absolute -top-1.5 -right-1.5 grid size-5 place-items-center rounded-full bg-foreground text-background"
              >
                <X className="size-3" />
              </button>
            </div>
          ))}

          {/* Ô thêm ảnh biến mất khi đã đủ: một nút bị khoá ở đây chỉ mời
              người ta bấm rồi không có gì xảy ra, mà con số giới hạn đã nằm
              ngay trong thông báo lỗi khi họ cố vượt.

              The add slot disappears once the post is full: a disabled
              button here only invites a click that does nothing, and the
              limit is already stated in the message if they try to exceed
              it. */}
          {images.length < MAX_IMAGES && (
            <button
              type="button"
              onClick={() => inputRef.current?.click()}
              aria-label="Thêm ảnh"
              className="grid size-16 shrink-0 place-items-center rounded-md border border-dashed border-border text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <Plus />
            </button>
          )}
        </div>
      )}
    </div>
  )
}

export const validateImage = (file: File): string | null => {
  if (!file.type.startsWith('image/')) return 'Hãy chọn file ảnh.'
  if (file.size > MAX_BYTES) return 'Mỗi ảnh phải nhỏ hơn 5 MB.'
  return null
}

/**
 * Reads a file's pixel dimensions without decoding it onto the page.
 *
 * A broken file resolves to 1×1 rather than rejecting: the server is the one
 * that decides what is a picture, and a draft that cannot be measured still
 * needs a crop rectangle for the form to stay well-formed. The server will
 * turn it down with a 400 and nothing is created.
 *
 * Đọc kích thước điểm ảnh của một file mà không giải mã nó lên trang.
 *
 * File hỏng sẽ trả về 1×1 chứ không báo lỗi: server mới là nơi quyết định
 * thứ gì là ảnh, còn một bản nháp không đo được vẫn cần một hình chữ nhật để
 * form giữ đúng dạng. Server sẽ từ chối bằng 400 và không tạo ra gì.
 */
const readSize = (file: File): Promise<{ width: number; height: number }> =>
  new Promise((resolve) => {
    const url = URL.createObjectURL(file)
    const image = new Image()
    const done = (width: number, height: number) => {
      URL.revokeObjectURL(url)
      resolve({ width, height })
    }
    image.onload = () => done(image.naturalWidth, image.naturalHeight)
    image.onerror = () => done(1, 1)
    image.src = url
  })
