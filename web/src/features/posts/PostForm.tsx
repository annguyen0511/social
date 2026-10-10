import { useEffect, useMemo, useRef } from 'react'
import { Globe, Lock, Trash, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
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

export type PostFields = {
  title: string
  content: string
  tags: string
  visibility: Visibility
  image: File | null
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
  image: null,
  imageError: null,
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

// Mirrors upload.MaxBytes and the formats image.Decode is set up to read.
// Checking here only saves a doomed round trip; the server re-encodes
// whatever arrives and stays the authority.
//
// Khớp với upload.MaxBytes và các định dạng mà image.Decode được cài để đọc.
// Kiểm ở đây chỉ để đỡ một lượt gọi mạng chắc chắn thất bại; server mã hoá
// lại mọi thứ nhận được và vẫn là nơi quyết định.
const MAX_IMAGE_BYTES = 5 * 1024 * 1024
const ACCEPT_IMAGE = 'image/jpeg,image/png,image/gif'

export function PostForm({
  id,
  fields,
  onChange,
  onSubmit,
  // Sửa bài chưa đổi được ảnh ở phiên bản này, nên ô chọn ảnh bị ẩn hẳn thay
  // vì hiện ra rồi im lặng không có tác dụng.
  //
  // This version cannot change a post's picture after the fact, so the
  // picker is hidden outright rather than shown and silently ignored.
  allowImage = true,
}: {
  id: string
  fields: PostFields
  onChange: (next: PostFields) => void
  onSubmit: () => void
  allowImage?: boolean
}) {
  const fileInput = useRef<HTMLInputElement>(null)

  // A blob: URL is a reference the browser holds until it is revoked, so the
  // old one has to go whenever the file changes or the form unmounts.
  //
  // blob: URL là một tham chiếu trình duyệt giữ lại cho tới khi bị thu hồi,
  // nên phải bỏ cái cũ mỗi lần đổi file hoặc khi form rời màn hình.
  const preview = useMemo(
    () => (fields.image ? URL.createObjectURL(fields.image) : null),
    [fields.image],
  )
  useEffect(() => {
    if (!preview) return
    return () => URL.revokeObjectURL(preview)
  }, [preview])

  const pick = (chosen: File | undefined) => {
    if (!chosen) return
    if (!chosen.type.startsWith('image/')) {
      onChange({ ...fields, image: null, imageError: 'Hãy chọn một file ảnh.' })
      return
    }
    if (chosen.size > MAX_IMAGE_BYTES) {
      onChange({ ...fields, image: null, imageError: 'Ảnh phải nhỏ hơn 5 MB.' })
      return
    }
    onChange({ ...fields, image: chosen, imageError: null })
  }

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

      {allowImage && (
        <div className="space-y-1.5">
          <Label>Ảnh</Label>

          {preview && (
            <img
              src={preview}
              alt="Ảnh sẽ đăng kèm bài"
              className="max-h-64 w-full rounded-lg border border-border object-contain"
            />
          )}

          <div className="flex gap-2">
            <input
              ref={fileInput}
              type="file"
              accept={ACCEPT_IMAGE}
              className="sr-only"
              onChange={(event) => pick(event.target.files?.[0])}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => fileInput.current?.click()}
            >
              <Upload />
              {fields.image ? 'Đổi ảnh' : 'Chọn ảnh'}
            </Button>

            {fields.image && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  onChange({ ...fields, image: null, imageError: null })
                  // Chọn lại đúng file vừa bỏ sẽ không bắn onChange nếu giá
                  // trị của input còn nguyên.
                  //
                  // Picking the very same file again fires no onChange unless
                  // the input's value is cleared.
                  if (fileInput.current) fileInput.current.value = ''
                }}
              >
                <Trash />
                Bỏ ảnh
              </Button>
            )}
          </div>

          <p className="text-xs text-muted-foreground">
            Tuỳ chọn. JPEG, PNG hoặc GIF, tối đa 5 MB. Ảnh theo chế độ hiển thị của bài.
          </p>
          {fields.imageError && (
            <p className="text-sm text-destructive">{fields.imageError}</p>
          )}
        </div>
      )}

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
