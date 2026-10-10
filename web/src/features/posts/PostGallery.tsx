import { useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { assetUrl } from '../../api/client'
import type { PostImage } from '../../api/types'
import { PostImageView } from './PostImageView'
import { cn } from 'cn'

/**
 * The pictures on a post: one picture, or a swipeable row of them.
 *
 * Built on native scroll-snap rather than a carousel library. The browser
 * already does the hard parts — momentum, touch, snapping, keyboard scroll —
 * and doing it this way means the pictures are reachable with JavaScript off
 * and the arrows are an addition rather than the only way through.
 *
 * Position is read back from the scroll offset rather than driven by state,
 * because a swipe moves the strip without asking anyone. State that tried to
 * own the position would be wrong the moment a finger touched it.
 *
 * Ảnh của một bài: một tấm, hoặc một dải kéo ngang được.
 *
 * Dựng trên scroll-snap sẵn có của trình duyệt chứ không thêm thư viện
 * carousel. Trình duyệt đã làm sẵn phần khó — quán tính, cảm ứng, bám điểm,
 * cuộn bằng bàn phím — và làm cách này nghĩa là ảnh vẫn xem được khi tắt
 * JavaScript, còn hai mũi tên chỉ là thứ thêm vào chứ không phải đường duy
 * nhất để đi qua.
 *
 * Vị trí được đọc lại từ độ lệch cuộn chứ không do state điều khiển, vì một
 * cú kéo làm dải dịch đi mà chẳng hỏi ai. State mà cố nắm vị trí thì sẽ sai
 * ngay khoảnh khắc có ngón tay chạm vào.
 */
export function PostGallery({
  images,
  alt,
  to,
}: {
  images: PostImage[]
  alt: string
  // Nơi để đi khi bấm vào ảnh, và chỉ áp dụng cho bài một ảnh. Dải nhiều
  // ảnh không dẫn đi đâu: ở đó cú bấm đã có nghĩa khác — kéo, bấm mũi tên,
  // bấm điểm tròn — và một cú kéo kết thúc bằng click sẽ thành điều hướng
  // ngoài ý muốn.
  //
  // Where a click on the picture goes, and only for a single-image post. A
  // strip of several leads nowhere: there a click already means something
  // else — swipe, arrow, dot — and a drag that ends in a click would become
  // navigation nobody asked for.
  to?: string
}) {
  const stripRef = useRef<HTMLDivElement>(null)
  const [at, setAt] = useState(0)

  if (images.length === 0) return null
  if (images.length === 1) {
    const only = <PostImageView image={images[0]} alt={alt} />
    return to ? (
      <Link to={to} className="block">
        {only}
      </Link>
    ) : (
      only
    )
  }

  const go = (index: number) => {
    const strip = stripRef.current
    if (!strip) return
    // Cuộn theo bề rộng khung nhìn, không theo bề rộng ảnh: mỗi ảnh chiếm
    // trọn một khung, nên đó là bước nhảy đúng dù tấm ảnh cao hay thấp.
    //
    // Scrolled by the viewport's width, not the picture's: each image fills
    // exactly one frame, so that is the right step whatever shape it is.
    strip.scrollTo({ left: strip.clientWidth * index, behavior: 'smooth' })
  }

  const clamped = Math.min(Math.max(at, 0), images.length - 1)

  return (
    <div className="group/gallery relative mt-3">
      <div
        ref={stripRef}
        onScroll={(event) => {
          const strip = event.currentTarget
          setAt(Math.round(strip.scrollLeft / strip.clientWidth))
        }}
        className="flex snap-x snap-mandatory overflow-x-auto scroll-smooth rounded-lg border border-border bg-muted [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {images.map((image, index) => (
          <img
            key={image.url}
            src={assetUrl(image.url)}
            width={image.width}
            height={image.height}
            alt={`${alt} — ảnh ${index + 1}/${images.length}`}
            // Tấm đầu tải ngay, các tấm sau chờ tới lúc được kéo tới. Trong
            // một feed, ảnh thứ hai của bài thứ mười là thứ gần như chắc
            // chắn không ai xem.
            //
            // The first loads now, the rest wait until swiped to. In a feed,
            // the second picture of the tenth post is something almost
            // nobody will look at.
            loading={index === 0 ? 'eager' : 'lazy'}
            className="w-full shrink-0 basis-full snap-center object-contain"
          />
        ))}
      </div>

      {/* Mũi tên chỉ hiện khi trỏ chuột vào và khi còn chỗ để đi: một mũi
          tên bấm vào không làm gì thì tệ hơn là không có mũi tên nào. Trên
          cảm ứng chúng không bao giờ hiện, mà ở đó thì kéo là thao tác tự
          nhiên rồi.

          The arrows appear on hover and only where there is somewhere to
          go: an arrow that does nothing when clicked is worse than no arrow.
          On touch they never appear, and there swiping is the natural move
          anyway. */}
      {clamped > 0 && <Arrow side="left" onClick={() => go(clamped - 1)} />}
      {clamped < images.length - 1 && <Arrow side="right" onClick={() => go(clamped + 1)} />}

      <div className="pointer-events-none absolute top-2 right-2 rounded-full bg-black/55 px-2 py-0.5 text-xs font-medium text-white">
        {clamped + 1}/{images.length}
      </div>

      <div className="mt-2 flex justify-center gap-1.5">
        {images.map((image, index) => (
          <button
            key={image.url}
            type="button"
            onClick={() => go(index)}
            aria-label={`Xem ảnh ${index + 1}`}
            aria-current={index === clamped}
            className={cn(
              'size-1.5 rounded-full transition-colors',
              index === clamped ? 'bg-foreground' : 'bg-foreground/25',
            )}
          />
        ))}
      </div>
    </div>
  )
}

function Arrow({ side, onClick }: { side: 'left' | 'right'; onClick: () => void }) {
  const Icon = side === 'left' ? ChevronLeft : ChevronRight
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={side === 'left' ? 'Ảnh trước' : 'Ảnh sau'}
      className={cn(
        'absolute top-1/2 grid size-8 -translate-y-1/2 place-items-center rounded-full bg-black/55 text-white opacity-0 transition-opacity group-hover/gallery:opacity-100 focus-visible:opacity-100',
        side === 'left' ? 'left-2' : 'right-2',
      )}
    >
      <Icon className="size-4" />
    </button>
  )
}
