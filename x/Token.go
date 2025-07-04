package x

import (
	"strings"
)

func GenApiToken(appid, secret string) string { // {{{
	return GenToken(appid, secret, false)
} // }}}

func GenRpcToken(appid, secret string) string { // {{{
	return GenToken(appid, secret, true)
} // }}}

func GenToken(appid, secret string, is_rpc bool, ts ...int) string { // {{{
	var t int
	if len(ts) > 0 {
		t = ts[0]
	}

	if t <= 0 {
		t = Now()
	}

	rpc := "0"
	if is_rpc {
		rpc = "1"
	}

	s := AsString(t)
	return MD5(appid+secret+s) + "." + appid + "." + s + "." + rpc
} // }}}

func CheckToken(token string) (string, bool) { // {{{
	tks := strings.SplitN(token, ".", 4)
	if len(tks) < 3 { //允许省略最后的rpc标识部分
		return "", false
	}

	appid := tks[1]
	t := tks[2]
	rpc := "0"
	if len(tks) > 3 {
		rpc = tks[3]
	} else {
		token += ".0"
	}

	secret, ok := ConfAuthApp[appid]
	if !ok {
		return appid, false
	}

	is_rpc := rpc == "1"

	ti := AsInt(t)
	if GenToken(appid, secret, is_rpc, ti) != token {
		return appid, false
	}

	ttl := 0
	if is_rpc {
		ttl = Conf_auth_rpc_check_ttl
	} else {
		ttl = Conf_auth_api_check_ttl
	}

	if ttl > 0 {
		return appid, Now()-ti < ttl
	}

	return appid, true
} // }}}
