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
	return MD5(appid+secret+s) + "." + rpc + "." + appid + "." + s
} // }}}

func CheckToken(token string) bool { // {{{
	tks := strings.SplitN(token, ".", 4)
	if len(tks) < 4 {
		return false
	}

	rpc := tks[1]
	appid := tks[2]
	t := tks[3]

	secret, ok := Conf_access_auth[appid]
	if !ok {
		return false
	}

	is_rpc := rpc == "1"

	ti := AsInt(t)
	if GenToken(appid, secret, is_rpc, ti) != token {
		return false
	}

	ttl := 0
	if is_rpc {
		ttl = Conf_rpc_auth_ttl
	} else {
		ttl = Conf_api_auth_ttl
	}

	if ttl > 0 {
		return Now()-ti < ttl
	}

	return true
} // }}}
