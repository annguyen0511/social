package main

import (
	"net/http"
)

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("internal error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	msg := "internal server error"
	writeJSONError(w, http.StatusInternalServerError, msg)
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("bad request error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusBadRequest, err.Error())
}

func (app *application) unauthorizedResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("unauthorized error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	// The reason never reaches the client: "no such email" and "wrong
	// password" must look identical, or the endpoint becomes a way to find
	// out which addresses are registered.
	//
	// Lý do không bao giờ tới client: "email không tồn tại" và "sai mật khẩu"
	// phải giống hệt nhau, nếu không endpoint này thành công cụ dò xem địa chỉ
	// nào đã đăng ký.
	msg := "invalid credentials"
	writeJSONError(w, http.StatusUnauthorized, msg)
}

func (app *application) forbiddenResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("forbidden error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusForbidden, err.Error())
}

func (app *application) conflictResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("conflict error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	msg := "resource conflict"
	writeJSONError(w, http.StatusConflict, msg)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("not found error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	msg := "not found"
	writeJSONError(w, http.StatusNotFound, msg)
}
