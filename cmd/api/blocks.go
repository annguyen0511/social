package main

import (
	"errors"
	"net/http"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
)

// errBlockedFromViewing is logged, never sent. See hideWhenBlocked.
// errBlockedFromViewing chỉ dùng để ghi log, không bao giờ gửi đi. Xem hideWhenBlocked.
var errBlockedFromViewing = errors.New("a block exists between the two users")

// hideWhenBlocked answers 404 for anything belonging to someone on either side
// of a block with the reader.
//
// 404 and not 403, which is the one decision worth explaining. A 403 says
// "this exists and you may not have it", which confirms the account or the
// post is real and tells the blocked person exactly who shut them out. A 404
// says nothing at all, and that is what blocking is for: not a locked door
// with a sign on it, but an absence.
//
// The writes that already answer 403 — follow, close friend, repost — stay as
// they are. Those are actions aimed at a person the reader named, so refusing
// out loud reveals nothing they did not already know.
//
// hideWhenBlocked trả về 404 cho mọi thứ thuộc về người đang có lệnh chặn với
// người đọc, ở bất kỳ chiều nào.
//
// 404 chứ không phải 403, và đây là quyết định đáng giải thích. 403 nói "thứ
// này có tồn tại và bạn không được phép", tức là xác nhận tài khoản hay bài
// viết đó có thật, đồng thời cho người bị chặn biết chính xác ai đã khoá cửa
// với họ. 404 thì không nói gì cả, và đó mới là mục đích của việc chặn: không
// phải một cánh cửa khoá có treo biển, mà là sự vắng mặt.
//
// Các thao tác ghi vốn đã trả 403 — theo dõi, bạn thân, repost — giữ nguyên.
// Đó là những hành động nhắm vào một người mà chính người đọc đã nêu tên, nên
// từ chối thành tiếng cũng không tiết lộ điều gì họ chưa biết.
func (app *application) hideWhenBlocked(w http.ResponseWriter, r *http.Request, otherID int64) bool {
	// Nobody blocks themselves, and asking would be a wasted query on every
	// request a user makes about their own things.
	//
	// Không ai tự chặn mình, và hỏi làm gì cũng chỉ tốn một truy vấn thừa ở
	// mỗi request người dùng hỏi về thứ của chính họ.
	viewerID := authUser(r).ID
	if viewerID == otherID {
		return false
	}

	blocked, err := app.store.Block.Exists(r.Context(), viewerID, otherID)
	if err != nil {
		app.internalServerError(w, r, err)
		return true
	}
	if blocked {
		app.notFoundResponse(w, r, errBlockedFromViewing)
		return true
	}
	return false
}

// requireNotBlockedUser guards everything hanging off /users/{userID}.
//
// It is deliberately not on /friend-ship/{userID}: unblocking lives there, and
// a gate that hides the blocked person would make the block impossible to
// lift. Asking for the friendship status has to keep working too, or the
// interface cannot tell you why someone disappeared.
//
// requireNotBlockedUser canh mọi thứ nằm dưới /users/{userID}.
//
// Nó cố tình không đặt ở /friend-ship/{userID}: lệnh gỡ chặn nằm ở đó, và một
// cánh cổng giấu người bị chặn đi sẽ khiến chính lệnh chặn không gỡ được nữa.
// Việc hỏi trạng thái quan hệ cũng phải chạy được, nếu không giao diện sẽ
// không giải thích nổi vì sao một người biến mất.
func (app *application) requireNotBlockedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, ok := getUserFromContext(r)
		if !ok {
			app.internalServerError(w, r, errors.New("user missing from request context"))
			return
		}

		if app.hideWhenBlocked(w, r, target.ID) {
			return
		}

		next.ServeHTTP(w, r.WithContext(r.Context()))
	})
}

// hideWhenNotVisible answers 404 for a private post the reader may not open.
//
// 404 for the same reason a block does: a private post is not a locked door,
// it is a post most people were never shown. Saying "forbidden" would tell a
// stranger that the author wrote something they are being kept out of.
//
// The list queries filter on the same rule in SQL. This is the gate for the
// single-post routes, where there is one post and the rule is cheaper to
// check in Go than to fold into the lookup.
//
// hideWhenNotVisible trả về 404 cho một bài riêng tư mà người đọc không được
// mở.
//
// 404 vì cùng lý do với việc chặn: một bài riêng tư không phải cánh cửa khoá,
// nó là bài mà phần lớn mọi người chưa từng được cho xem. Nói "không được
// phép" là cho người lạ biết tác giả đã viết một thứ mà họ bị gạt ra ngoài.
//
// Các truy vấn danh sách lọc theo đúng luật này bằng SQL. Đây là cổng cho các
// route thao tác trên một bài đơn lẻ, nơi chỉ có một bài và việc kiểm trong
// Go rẻ hơn là nhét vào câu truy vấn tra cứu.
func (app *application) hideWhenNotVisible(w http.ResponseWriter, r *http.Request, post *model.Post) bool {
	if post.Visibility == store.VisibilityPublic {
		return false
	}

	viewerID := authUser(r).ID
	if post.UserID == viewerID {
		return false
	}

	// The author's list decides, not the reader's: being someone's close
	// friend is not mutual unless both added each other.
	//
	// Danh sách của tác giả mới quyết định, không phải của người đọc: việc là
	// bạn thân của ai đó không có tính hai chiều, trừ khi cả hai cùng thêm
	// nhau.
	isCloseFriend, err := app.store.CloseFriend.IsCloseFriend(r.Context(), post.UserID, viewerID)
	if err != nil {
		app.internalServerError(w, r, err)
		return true
	}
	if !isCloseFriend {
		app.notFoundResponse(w, r, errors.New("the post is private"))
		return true
	}
	return false
}
