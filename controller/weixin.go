package controller

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/goyoubbs/model"
	"github.com/ego008/youdb"
	"github.com/ohko/hst"
	"github.com/rs/xid"
)

const (
	// 光明城市业主
	wxToken     = "weixin20180922"
	wxAppID     = "wx68786fa52688144f"
	wxAppSecret = "8bfc961270bce17418bd3fc36561c71c"
)

// accessToken
type stAccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
}

// 用户信息
type stUserInfo struct {
	OpenID     string `json:"openid"`
	NickName   string `json:"nickname"`
	Sex        int    `json:"sex"`
	HeadImgURL string `json:"headimgurl"` // "http://thirdwx.qlogo.cn/mmopen/g3MonUZtNHkdmzicIlibx6iaFqAc56vxLSUfpb6n5WKSYVY0ChQKkiaJSgQ1dZuTOgvLLrhJbERQQ4eMsv84eavHiaiceqxibJxCfHe/46",
}

// /weixin?signature=460816b7f5630cee69b94417f3468c8cfd9632ec&echostr=1370323151484609439&timestamp=1544249756&nonce=1987360397
// 接口接入验证
func (h *BaseHandler) Weixin(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("Weixin:", r.RequestURI)
	signature, timestamp, nonce, echostr := r.FormValue("signature"), r.FormValue("timestamp"), r.FormValue("nonce"), r.FormValue("echostr")
	kvs := []string{wxToken, timestamp, nonce}
	sort.Strings(kvs)
	ss := strings.Join(kvs, "")
	s1 := sha1.Sum([]byte(ss))
	if signature == hex.EncodeToString(s1[:]) {
		fmt.Fprint(w, echostr)
	} else {
		fmt.Fprint(w, "failed")
	}
}

// 网页授权域名
func (h *BaseHandler) WeixinDomain(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "u5f2OffVxYWsDSY7")
}

// 第一步：用户同意授权，获取code
// https://open.weixin.qq.com/connect/oauth2/authorize?appid=wx68786fa52688144f&redirect_uri=https%3A%2F%2Fgmcs.lyl.hk%2Fweixinlogin&response_type=code&scope=snsapi_base&state=STATE#wechat_redirect
// => /weixinlogin?code=001XJoar03nWao1zzH8r08bpar0XJoa1&state=STATE
// 第二步：通过code换取网页授权access_token
func (h *BaseHandler) WeixinLogin(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("WeixinLogin:", r.RequestURI)
	var accessSt stAccessToken
	var uri string
	code := r.FormValue("code")
	act := r.FormValue("act")
	uri = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=" + wxAppID + "&secret=" + wxAppSecret + "&code=" + code + "&grant_type=authorization_code"
	log.Println("au:", uri)
	bs, _, err := hst.HSRequest("GET", uri, "", "")
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	if bytes.Contains(bs, []byte("errcode")) {
		fmt.Fprint(w, string(bs))
		return
	}
	if err := json.Unmarshal(bs, &accessSt); err != nil {
		fmt.Fprint(w, err)
		return
	}

	if accessSt.OpenID == "" {
		fmt.Fprint(w, "OpenID empty")
		return
	}

	var userInfoSt stUserInfo
	{ // 第四步：拉取用户信息(需scope为 snsapi_userinfo)
		bs, _, err := hst.HSRequest("GET", "https://api.weixin.qq.com/sns/userinfo?access_token="+accessSt.AccessToken+"&openid="+accessSt.OpenID+"&lang=zh_CN", "", "")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		if bytes.Contains(bs, []byte("errcode")) {
			fmt.Fprint(w, string(bs))
			return
		}
		if err := json.Unmarshal(bs, &userInfoSt); err != nil {
			fmt.Fprint(w, err)
			return
		}
		if userInfoSt.NickName == "" {
			fmt.Fprint(w, "NickName empty")
			return
		}
	}

	// 用户自动登录或注册
	db := h.App.Db
	openid := userInfoSt.OpenID
	nickName := userInfoSt.NickName
	timeStamp := uint64(time.Now().UTC().Unix())
	uobj, err := model.UserGetByName(db, openid)
	if err != nil { // 增加用户
		userId, _ := db.HnextSequence("user")
		siteCf := h.App.Cf.Site
		flag := 5
		if siteCf.RegReview {
			flag = 1
		}

		uobj := model.User{
			Id:       userId,
			Name:     openid,
			NickName: nickName,
			// Password:      rec.Password,
			Flag:          flag,
			RegTime:       timeStamp,
			LastLoginTime: timeStamp,
			Session:       xid.New().String(),
		}

		// uidStr := strconv.FormatUint(userId, 10)
		// err := util.GenerateAvatar("male", nameLow, 73, 73, "static/avatar/"+uidStr+".jpg")
		// if err != nil {
		// 	uobj.Avatar = "0"
		// } else {
		// uobj.Avatar = uidStr
		// }
		uobj.Avatar = userInfoSt.HeadImgURL

		jb, _ := json.Marshal(uobj)
		db.Hset("user", youdb.I2b(uobj.Id), jb)
		db.Hset("user_name2uid", []byte(openid), youdb.I2b(userId))
		db.Hset("user_flag:"+strconv.Itoa(flag), youdb.I2b(uobj.Id), []byte(""))

		h.SetCookie(w, "SessionID", strconv.FormatUint(uobj.Id, 10)+":"+uobj.Session, 365)
	} else { // 用户登录
		sessionid := xid.New().String()
		uobj.NickName = userInfoSt.NickName
		uobj.Avatar = userInfoSt.HeadImgURL
		uobj.LastLoginTime = timeStamp
		uobj.Session = sessionid
		jb, _ := json.Marshal(uobj)
		db.Hset("user", youdb.I2b(uobj.Id), jb)
		h.SetCookie(w, "SessionID", strconv.FormatUint(uobj.Id, 10)+":"+sessionid, 365)
	}
	switch act {
	case "newpost":
		http.Redirect(w, r, "https://ohko.cn/newpost/2", 302)
	default:
		http.Redirect(w, r, "/", 302)
	}
}
