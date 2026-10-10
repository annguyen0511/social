import { Globe, Lock } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from 'cn'
import type { Visibility } from '../../api/types'

// Mirrors the validate tags on createPostRequest in cmd/api/posts.go. The
// server stays the authority; this only saves a doomed round trip.
//
// Khớp với các tag validate của createPostRequest trong cmd/api/posts.go.
// Server vẫn là nơi quyết định; chỗ này chỉ để đỡ một lượt gọi mạng chắc chắn
// thất bại.
export const MAX_TITLE = 100
export const MAX_CONTENT = 1000

// images và imageError nằm ở đây dù form này không vẽ ô chọn ảnh: chúng là
// một phần của bài viết đang soạn, và bước chọn ảnh trong CreatePostDialog
// ghi vào cùng một chỗ. Giữ chung một kiểu nghĩa là chỉ có một bản nháp.
//
// images and imageError live here even though this form draws no picker:
// they are part of the post being written, and the picture step in
// CreatePostDialog writes into the same place. One type means one draft.
export type CropArea = { x: number; y: number; width: number; height: number }

// Khớp với maxImages trong cmd/api/posts.go.
// Mirrors maxImages in cmd/api/posts.go.
export const MAX_IMAGES = 10

/**
 * One chosen picture, with everything needed to crop it.
 *
 * The crop frame's own state — offset, zoom, ratio — is kept here rather than
 * inside ImageCropper, so that switching to another picture and back leaves
 * the frame exactly where it was. A cropper holding its own state would reset
 * on every remount and quietly overwrite the stored rectangle with a fresh
 * default.
 *
 * `size` is read off the file when it is picked, which is also what lets
 * `crop` start as the whole picture. Every draft therefore always has a
 * rectangle, and the server's all-or-none rule is met without the dialog
 * having to reason about which pictures the user happened to look at.
 *
 * Một tấm ảnh đã chọn, kèm mọi thứ cần để cắt nó.
 *
 * Trạng thái của khung cắt — độ lệch, zoom, tỉ lệ — được giữ ở đây chứ không
 * nằm trong ImageCropper, để chuyển sang ảnh khác rồi quay lại thì khung vẫn
 * đứng nguyên chỗ cũ. Một cropper tự giữ trạng thái sẽ reset mỗi lần được
 * dựng lại và âm thầm ghi đè hình chữ nhật đã lưu bằng một giá trị mặc định.
 *
 * `size` được đọc từ file ngay khi chọn, và đó cũng là thứ cho phép `crop`
 * khởi đầu bằng cả tấm ảnh. Nhờ vậy mọi bản nháp luôn có một hình chữ nhật,
 * và luật "đủ cả hoặc không cái nào" của server được thoả mà dialog không
 * phải suy xét người dùng đã kịp xem những ảnh nào.
 */
export type ImageDraft = {
  // Khoá bền để React và các blob: URL không bị xáo khi bỏ một ảnh ở giữa.
  // A stable key, so React and the blob: URLs are not shuffled when a
  // picture in the middle is removed.
  id: string
  file: File
  size: { width: number; height: number }
  crop: CropArea
  view: { x: number; y: number; zoom: number; ratio: number | null }
}

export const newImageDraft = (file: File, size: { width: number; height: number }): ImageDraft => ({
  id: `${file.name}:${file.size}:${file.lastModified}:${Math.random().toString(36).slice(2)}`,
  file,
  size,
  crop: { x: 0, y: 0, width: size.width, height: size.height },
  view: { x: 0, y: 0, zoom: 1, ratio: null },
})

export type PostFields = {
  title: string
  content: string
  tags: string
  visibility: Visibility
  images: ImageDraft[]
  imageError: string | null
}

export const emptyPost: PostFields = {
  title: '',
  content: '',
  tags: '',
  // Mặc định công khai: chọn sai theo hướng kín thì không ai thấy bài, chọn
  // sai theo hướng mở thì bài lọt ra ngoài. Nhưng người dùng mong bài mình
  // viết có người đọc, nên mặc định phải là thứ họ chờ đợi, và nút kia nằm
  // ngay cạnh.
  //
  // Public by default: erring closed means nobody sees the post, erring open
  // means it leaks. People expect what they write to be read, though, so the
  // default has to be what they expect, with the other option right beside it.
  visibility: 'public',
  images: [],
  imageError: null,
}

/**
 * Lays a draft out as the multipart form cmd/api/posts.go reads.
 *
 * The crop fields repeat in parallel with the files: the nth crop_x belongs
 * to the nth picture. The server insists on all four fields, one entry per
 * picture, or none at all — a shorter list could only be matched up by
 * guessing, and a wrong guess crops the wrong photo. Every draft carries a
 * rectangle from the moment it was chosen, so this always sends the full set.
 *
 * Dựng một bản nháp thành đúng form multipart mà cmd/api/posts.go đọc.
 *
 * Các field vùng cắt lặp lại song song với các file: crop_x thứ n thuộc về
 * tấm ảnh thứ n. Server đòi đủ bốn field, mỗi ảnh một giá trị, hoặc không
 * field nào — một danh sách ngắn hơn chỉ ghép lại được bằng cách đoán, mà
 * đoán sai là cắt nhầm ảnh. Mọi bản nháp đều mang một hình chữ nhật từ lúc
 * được chọn, nên chỗ này luôn gửi đủ bộ.
 */
export function buildPostBody(fields: PostFields): FormData {
  const body = new FormData()
  body.append('title', fields.title.trim())
  body.append('content', fields.content.trim())
  body.append('tags', fields.tags)
  body.append('visibility', fields.visibility)

  for (const image of fields.images) {
    body.append('images', image.file)
    body.append('crop_x', String(Math.round(image.crop.x)))
    body.append('crop_y', String(Math.round(image.crop.y)))
    body.append('crop_width', String(Math.round(image.crop.width)))
    body.append('crop_height', String(Math.round(image.crop.height)))
  }

  return body
}

const options: { value: Visibility; label: string; hint: string; icon: typeof Globe }[] = [
  { value: 'public', label: 'Công khai', hint: 'Ai cũng xem được', icon: Globe },
  { value: 'private', label: 'Bạn thân', hint: 'Chỉ bạn thân của bạn xem được', icon: Lock },
]

/**
 * Tags travel as an array but are typed as one comma-separated line, so the
 * form stays a plain text field. Empty pieces are dropped, which is what makes
 * a trailing comma harmless.
 *
 * Tag truyền đi dưới dạng mảng nhưng được gõ trên một dòng ngăn bằng dấu
 * phẩy, nên ô nhập vẫn chỉ là một ô chữ bình thường. Các phần rỗng bị loại,
 * và đó là thứ khiến một dấu phẩy thừa ở cuối không gây lỗi.
 */
export const parseTags = (value: string): string[] =>
  value
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)

export const isValidPost = (fields: PostFields): boolean =>
  fields.title.trim().length > 0 &&
  fields.title.trim().length <= MAX_TITLE &&
  fields.content.trim().length > 0 &&
  fields.content.trim().length <= MAX_CONTENT

export function PostForm({
  id,
  fields,
  onChange,
  onSubmit,
}: {
  id: string
  fields: PostFields
  onChange: (next: PostFields) => void
  onSubmit: () => void
}) {
  return (
    <form
      id={id}
      className="space-y-4"
      onSubmit={(event) => {
        event.preventDefault()
        onSubmit()
      }}
    >
      <div className="space-y-1.5">
        <Label htmlFor={`${id}-title`}>Tiêu đề</Label>
        <Input
          id={`${id}-title`}
          value={fields.title}
          maxLength={MAX_TITLE}
          onChange={(event) => onChange({ ...fields, title: event.target.value })}
        />
      </div>

      <div className="space-y-1.5">
        <div className="flex items-baseline justify-between">
          <Label htmlFor={`${id}-content`}>Nội dung</Label>
          <span className="text-xs text-muted-foreground">
            {fields.content.length}/{MAX_CONTENT}
          </span>
        </div>
        <Textarea
          id={`${id}-content`}
          value={fields.content}
          maxLength={MAX_CONTENT}
          rows={5}
          onChange={(event) => onChange({ ...fields, content: event.target.value })}
        />
      </div>

      <div className="space-y-1.5">
        <Label>Ai xem được</Label>
        <div className="grid grid-cols-2 gap-2">
          {options.map(({ value, label, hint, icon: Icon }) => (
            <button
              key={value}
              type="button"
              aria-pressed={fields.visibility === value}
              onClick={() => onChange({ ...fields, visibility: value })}
              className={cn(
                'flex items-start gap-2.5 rounded-lg border p-3 text-left transition-colors',
                fields.visibility === value
                  ? 'border-foreground bg-muted'
                  : 'border-border hover:bg-muted',
              )}
            >
              <Icon className="mt-0.5 size-4 shrink-0" />
              <span className="min-w-0">
                <span className="block text-sm font-medium">{label}</span>
                <span className="block text-xs text-muted-foreground">{hint}</span>
              </span>
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-1.5">
        <Label htmlFor={`${id}-tags`}>Thẻ</Label>
        <Input
          id={`${id}-tags`}
          value={fields.tags}
          placeholder="go, postgres"
          onChange={(event) => onChange({ ...fields, tags: event.target.value })}
        />
        <p className="text-xs text-muted-foreground">Ngăn cách bằng dấu phẩy.</p>
      </div>
    </form>
  )
}
