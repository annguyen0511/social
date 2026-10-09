import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

// Mirrors the validate tags on createPostRequest in cmd/api/posts.go. The
// server stays the authority; this only saves a doomed round trip.
//
// Khớp với các tag validate của createPostRequest trong cmd/api/posts.go.
// Server vẫn là nơi quyết định; chỗ này chỉ để đỡ một lượt gọi mạng chắc chắn
// thất bại.
export const MAX_TITLE = 100
export const MAX_CONTENT = 1000

export type PostFields = { title: string; content: string; tags: string }

export const emptyPost: PostFields = { title: '', content: '', tags: '' }

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
