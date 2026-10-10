import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
import { cn } from 'cn'
import { postForm } from '../../api/client'
import type { Post } from '../../api/types'
import { CroppedPreview } from './CroppedPreview'
import { ImagePicker } from './ImagePicker'
import { buildPostBody, emptyPost, isValidPost, PostForm, type PostFields } from './PostForm'

type Step = 'image' | 'details'

/**
 * Writing a post, in two steps: pick the picture, then write about it.
 *
 * Splitting them keeps either screen from being crowded, and it matches the
 * order people actually work in — you choose the photo you want to post
 * before you know what to say about it.
 *
 * The step is not a route. A half-written post is not something to put in the
 * address bar: reloading or sharing that URL would land on an empty form
 * claiming to be step two.
 *
 * Viết một bài, theo hai bước: chọn ảnh, rồi viết về nó.
 *
 * Tách ra giúp không màn hình nào bị chật, và nó khớp với thứ tự người ta
 * thật sự làm — chọn tấm ảnh muốn đăng trước khi biết sẽ nói gì về nó.
 *
 * Bước không phải là một route. Một bài viết dở dang không phải thứ nên đưa
 * lên thanh địa chỉ: tải lại hay chia sẻ URL đó sẽ rơi vào một biểu mẫu rỗng
 * tự xưng là bước hai.
 */
export function CreatePostDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [step, setStep] = useState<Step>('image')
  const [fields, setFields] = useState<PostFields>(emptyPost)
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  useEffect(() => {
    if (!open) return
    setStep('image')
    setFields(emptyPost)
    setShown(0)
  }, [open])

  // Một blob: URL cho mỗi ảnh, thu hồi cả bảng khi danh sách đổi. Bước hai
  // chỉ xem chứ không sửa, nên nó không cần biết gì về khung cắt ngoài hình
  // chữ nhật đã lưu trên bản nháp.
  //
  // One blob: URL per picture, the whole table revoked when the list
  // changes. Step two only looks, never edits, so it needs to know nothing
  // about the crop frame beyond the rectangle stored on the draft.
  const previews = useMemo(
    () => fields.images.map((image) => ({ id: image.id, url: URL.createObjectURL(image.file) })),
    [fields.images],
  )
  useEffect(
    () => () => previews.forEach(({ url }) => URL.revokeObjectURL(url)),
    [previews],
  )

  const [shown, setShown] = useState(0)

  // Bỏ một ảnh ở bước một rồi quay lại có thể để lại chỉ số trỏ ra ngoài
  // danh sách; kẹp lại lúc render chứ không trong effect, để không có một
  // nhịp nào hiện ra khoảng trắng.
  //
  // Removing a picture in step one and coming back can leave the index
  // pointing past the list; clamped during render rather than in an effect,
  // so there is no frame in which a blank shows.
  const current = previews[Math.min(shown, previews.length - 1)] ?? null
  const currentCrop = fields.images[Math.min(shown, fields.images.length - 1)]?.crop ?? null

  const mutation = useMutation({
    mutationFn: () => {
      // A form rather than JSON, because the pictures travel with the post.
      // The server sends back 400 and creates nothing if any of the files is
      // not an image, so there is no half-made post to clean up.
      //
      // Dùng form chứ không phải JSON, vì những tấm ảnh đi cùng bài viết.
      // Server trả 400 và không tạo gì nếu có file nào không phải ảnh, nên
      // không có bài nào dở dang phải dọn.
      return postForm<Post>('/v1/posts', buildPostBody(fields))
    },
    onSuccess: (created) => {
      // A new post belongs in the author's own feed and on their profile, and
      // both are now one row out of date.
      //
      // Bài mới thuộc về feed của chính tác giả và trang cá nhân của họ, mà cả
      // hai đều vừa thiếu mất một dòng.
      queryClient.invalidateQueries({ queryKey: ['feed'] })
      queryClient.invalidateQueries({ queryKey: ['userPosts'] })
      onOpenChange(false)
      navigate(`/posts/${created.id}`)
    },
  })

  const onImage = step === 'image'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* Chiều cao PHẢI xác định, không chỉ là max-h.
          Cropper đặt nội dung bằng position: absolute nên nó không có chiều
          cao tự thân; cha mà để auto thì cả khung cắt sập về gần 0 và người
          dùng chẳng thấy gì để kéo. h-[85dvh] là thứ cho hàng 1fr một kích
          thước thật để truyền xuống.

          Nó cũng giữ khung không nhảy giữa hai bước: bước chọn ảnh gần như
          trống còn bước chi tiết là cả một biểu mẫu.

          The height MUST be definite, not just a max.
          Cropper lays its content out with position: absolute, so it has no
          height of its own; leave the parent on auto and the whole crop frame
          collapses to nearly nothing, with no visible area to drag. h-[85dvh]
          is what gives the 1fr row a real size to pass down.

          It also keeps the frame from jumping between steps: the picture step
          is nearly empty while the details step is a whole form. */}
      <DialogContent
        showCloseButton={false}
        className="grid h-[85dvh] max-h-[48rem] w-[calc(100%-2rem)] max-w-5xl grid-rows-[auto_1fr] gap-0 overflow-hidden p-0 sm:max-w-5xl"
      >
        <header className="flex h-12 shrink-0 items-center justify-between border-b border-border px-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={mutation.isPending}
            onClick={() => (onImage ? onOpenChange(false) : setStep('image'))}
          >
            <ArrowLeft />
            {onImage ? 'Huỷ' : 'Quay lại'}
          </Button>

          <DialogTitle className="text-sm font-semibold">
            {onImage ? 'Chọn ảnh' : 'Bài viết mới'}
          </DialogTitle>
          <DialogDescription className="sr-only">
            Chọn ảnh ở bước một, rồi viết nội dung ở bước hai.
          </DialogDescription>

          {onImage ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              // Bỏ qua ảnh được, nhưng phải đi qua đường dẫn trong màn hình
              // chứ không phải nút này — nút này chỉ sáng khi đã có ảnh, để
              // "Tiếp" luôn có nghĩa là "xong bước chọn ảnh".
              //
              // Skipping the picture is allowed, but through the link inside
              // the screen rather than this button — it only lights up once
              // there is an image, so "Tiếp" always means "done picking".
              disabled={fields.images.length === 0}
              onClick={() => setStep('details')}
            >
              Tiếp
            </Button>
          ) : (
            <Button
              type="button"
              size="sm"
              disabled={!isValidPost(fields) || mutation.isPending}
              onClick={() => mutation.mutate()}
            >
              {mutation.isPending && <Loader2 className="animate-spin" />}
              {mutation.isPending ? 'Đang đăng' : 'Đăng'}
            </Button>
          )}
        </header>

        <div className="min-h-0 overflow-hidden">
          {onImage ? (
            <ImagePicker
              images={fields.images}
              error={fields.imageError}
              onChange={(update) =>
                setFields((current) => ({ ...current, images: update(current.images) }))
              }
              onError={(imageError) => setFields((current) => ({ ...current, imageError }))}
              onSkip={() => setStep('details')}
            />
          ) : (
            <div className="flex h-full min-h-0">
              {/* Ảnh ở lại bên trái suốt bước hai, để người viết nhìn thấy
                  thứ mình đang viết về. Bài không có ảnh thì cột này biến
                  mất và biểu mẫu chiếm trọn bề ngang.

                  The picture stays on the left through step two, so whoever
                  is writing can see what they are writing about. With no
                  image the column disappears and the form takes the width. */}
              {current && (
                <div className="hidden min-h-0 w-1/2 shrink-0 grid-rows-[1fr_auto] bg-muted/40 p-4 sm:grid">
                  {/* Phần đã cắt, không phải file gốc: đây là thứ sẽ được
                      đăng, nên xem trước phải khớp với nó.

                      The cropped region, not the original file: this is what
                      gets posted, so the preview has to match it. */}
                  <div className="grid min-h-0 place-items-center">
                    <CroppedPreview
                      key={current.id}
                      src={current.url}
                      crop={currentCrop}
                      className="max-h-full max-w-full object-contain"
                    />
                  </div>

                  {/* Dải ảnh chỉ hiện khi có nhiều hơn một tấm: với một ảnh
                      thì nó chẳng nói thêm điều gì.

                      The strip only appears past one picture: with a single
                      image it says nothing the preview has not already
                      said. */}
                  {previews.length > 1 && (
                    <div className="mt-3 flex items-center justify-center gap-2">
                      {previews.map(({ id, url }, index) => (
                        <button
                          key={id}
                          type="button"
                          onClick={() => setShown(index)}
                          aria-current={id === current.id}
                          aria-label={`Xem ảnh ${index + 1}`}
                          className={cn(
                            'size-10 overflow-hidden rounded border-2 transition-colors',
                            id === current.id ? 'border-foreground' : 'border-transparent',
                          )}
                        >
                          <img src={url} alt="" className="size-full object-cover" />
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div className="min-w-0 flex-1 overflow-y-auto p-5">
                <PostForm
                  id="create-post"
                  fields={fields}
                  onChange={setFields}
                  onSubmit={() => mutation.mutate()}
                />

                {mutation.isError && (
                  <p className="mt-4 text-sm text-destructive">{mutation.error.message}</p>
                )}
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
