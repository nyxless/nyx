package x

import (
	"context"
	"strings"
)

type Token interface {
	CheckToken(ctx context.Context, header MAPS, params MAP) bool
}

type DefaultToken struct{}

func (dt *DefaultToken) CheckToken(ctx context.Context, hd MAPS, params MAP) bool {
	appid, _ := hd[Conf_auth_appid_key]
	nonce, _ := hd["nonce"]
	timestamp, _ := hd["timestamp"]
	authorization, _ := hd["authorization"]
	token := strings.TrimPrefix(authorization, "Bearer ")

	secret, _ := ConfAuthApp[appid]

	ttl := 0
	if AsInt(ctx.Value("mode")) == 1 {
		ttl = Conf_auth_rpc_check_ttl
	} else {
		ttl = Conf_auth_api_check_ttl
	}

	if ttl > 0 && Now()-AsInt(timestamp) > ttl {
		return false
	}

	return VerifySha256(token, appid+nonce+timestamp, secret)
}

/*
func genToken(appid, secret string, is_rpc bool, ts ...int) string { // {{{
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
	tks := strings.SplitN(token, ".", 3)

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
	if genToken(appid, secret, is_rpc, ti) != token {
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
*/
