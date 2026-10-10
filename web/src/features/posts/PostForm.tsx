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

// image và imageError nằm ở đây dù form này không vẽ ô chọn ảnh: chúng là
// một phần của bài viết đang soạn, và bước chọn ảnh trong CreatePostDialog
// ghi vào cùng một chỗ. Giữ chung một kiểu nghĩa là chỉ có một bản nháp.
//
// image and imageError live here even though this form draws no picker: they
// are part of the post being written, and the picture step in
// CreatePostDialog writes into the same place. One type means one draft.
export type CropArea = { x: number; y: number; width: number; height: number }

export type PostFields = {
  title: string
  content: string
  tags: string
  visibility: Visibility
  image: File | null
  imageError: string | null
  // Toạ độ vùng cắt theo điểm ảnh của chính file gốc. null khi chưa chọn ảnh
  // hoặc người dùng chưa đụng vào khung cắt.
  //
  // The crop rectangle in the original file's own pixels. null when no
  // picture is chosen, or the frame has not been touched yet.
  crop: CropArea | null
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
  crop: null,
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
