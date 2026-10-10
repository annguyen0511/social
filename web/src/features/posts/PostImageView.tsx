import { assetUrl } from '../../api/client'
import type { PostImage } from '../../api/types'

/**
 * The picture attached to a post.
 *
 * width and height are set as attributes, not just styles: a browser derives
 * the aspect ratio from them and reserves the space before a single byte of
 * the image arrives. Without them the text below jumps down the moment each
 * picture lands, which in a feed happens over and over as you scroll.
 *
 * Ảnh đính kèm một bài viết.
 *
 * width và height được đặt làm thuộc tính chứ không chỉ là style: trình duyệt
 * suy ra tỉ lệ khung hình từ chúng và chừa sẵn chỗ trước khi một byte nào của
 * ảnh về tới. Thiếu chúng thì phần chữ bên dưới bị đẩy xuống ngay lúc mỗi tấm
 * ảnh hiện ra, mà trong một feed thì chuyện đó lặp đi lặp lại suốt lúc cuộn.
 */
export function PostImageView({ image, alt }: { image: PostImage; alt: string }) {
  return (
    <img
      src={assetUrl(image.url)}
      width={image.width}
      height={image.height}
      alt={alt}
      // Chỉ tải khi sắp cuộn tới. Một feed hai mươi bài có ảnh mà tải hết
      // ngay từ đầu là hai mươi lần tải cho thứ người dùng chưa nhìn thấy.
      //
      // Only fetched as it comes into view. A feed of twenty posts with
      // pictures would otherwise be twenty downloads for things nobody has
      // looked at yet.
      loading="lazy"
      className="mt-3 w-full rounded-lg border border-border bg-muted object-contain"
    />
  )
}
