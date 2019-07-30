package controller

import (
	"fmt"
	"strings"
	"net/http"
)

func (h *BaseHandler) ViewAtTpl(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "MicroMessenger") {
		fmt.Fprint(w, "此服务仅为微信授权后使用")
		return
	}

	tpl := r.FormValue("tpl")
	if tpl == "desktop" || tpl == "mobile" {
		h.SetCookie(w, "tpl", tpl, 1)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
