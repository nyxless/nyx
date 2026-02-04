package x

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"time"
)

var machineId int

func init() {
	machineId = getMachineId()
}

func getMachineId() int { //{{{
	h, err := os.Hostname()
	if nil != err {
		h = GetLocalIp()
	}

	return Crc32(h)&0x7FFFFFF | 0x8000000
} // }}}

// 获取唯一ID, 9位machineId + 9位纳秒 + 6位随机int
func GetUUID() string { //{{{
	return fmt.Sprintf("%d%d%d", machineId, time.Now().Nanosecond()&0x7FFFFFF|0x8000000, RandIntn(1000000))
} //}}}

func MD5(str string) string { // {{{
	h := md5.New()
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil))
} // }}}

func MD5File(file string) (string, error) { // {{{
	str, err := FileGetContents(file)
	if err != nil {
		return "", err
	}

	return MD5(str), nil
} // }}}

func Crc32(str string) int { // {{{
	ieee := crc32.NewIEEE()
	io.WriteString(ieee, str)
	return int(ieee.Sum32())
} // }}}

// SHA256哈希函数, 第二参数为密钥，有更高安全需求时使用
func Sha256(input string, keys ...string) string { // {{{
	var key string
	if len(keys) > 0 {
		key = keys[0]
	}

	var h hash.Hash
	if key == "" {
		h = sha256.New()
	} else {
		h = hmac.New(sha256.New, []byte(key))
	}

	h.Write([]byte(input))

	return hex.EncodeToString(h.Sum(nil))
} // }}}

// 验证sha256
func VerifySha256(shaStr, input string, keys ...string) bool { // {{{
	newSha := Sha256(input, keys...)

	if len(keys) == 0 {
		return newSha == shaStr
	}
	newShaBytes, _ := hex.DecodeString(newSha)
	shaStrBytes, _ := hex.DecodeString(shaStr)
	// 比较是否相等（使用恒定时间比较防止时序攻击）
	return hmac.Equal(newShaBytes, shaStrBytes)
} // }}}

// 为字符串生成哈希值
func Hash(s string) uint64 { // {{{
	return xxhash.Sum64String(s)
} // }}}

// 为slice生成哈希值
func HashSlice(s []any) uint64 { // {{{
	var sb strings.Builder
	for i, k := range s {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(AsString(k))
	}

	return xxhash.Sum64String(sb.String())
} // }}}

// 为map生成哈希值
func HashMap(m map[string]any) uint64 { // {{{

	return xxhash.Sum64String(MapToString(m))
} // }}}
