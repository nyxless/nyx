package x

import (
	"context"
	"encoding/binary"
	"github.com/nyxless/nyx/x/pb"
	"google.golang.org/grpc/peer"
	"math"
	"net"
	"time"
	"unsafe"
)

// 获取 grpc 的客户端IP
func GetRpcIp(ctx context.Context) string { // {{{
	pr, ok := peer.FromContext(ctx)
	if !ok || pr.Addr == nil {
		return ""
	}

	switch addr := pr.Addr.(type) {
	case *net.TCPAddr:
		return addr.IP.String()
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.IPAddr:
		return addr.IP.String()
	case *net.UnixAddr:
		return ""
	}

	host, _, err := net.SplitHostPort(pr.Addr.String())
	if err != nil {
		return pr.Addr.String()
	}

	return host
} // }}}

// context缓存中获取ip
func GetRpcCtxIp(ctx context.Context) (context.Context, string) { // {{{
	if ip, ok := ctx.Value("ip").(string); ok {
		return ctx, ip
	}

	ip := GetRpcIp(ctx)
	return context.WithValue(ctx, "ip", ip), ip
} // }}}

// 基础类型和byte[] 互转，用于 rpc 通信数据的编解码
// 支持的类型：int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
// float32, float64, bool, string, complex64, complex128, time.Time
func DataToBytes(value any) (pb.Type, []byte) { // {{{
	switch v := value.(type) {
	case nil:
		return pb.Type_NIL, nil

	// 有符号整数
	case int:
		return pb.Type_INT, intToBytes(v)
	case int8:
		return pb.Type_INT8, []byte{byte(v)}
	case int16:
		return pb.Type_INT16, int16ToBytes(v)
	case int32:
		return pb.Type_INT32, int32ToBytes(v)
	case int64:
		return pb.Type_INT64, int64ToBytes(v)

	// 无符号整数
	case uint:
		return pb.Type_UINT, uintToBytes(v)
	case uint8:
		return pb.Type_UINT8, []byte{v}
	case uint16:
		return pb.Type_UINT16, uint16ToBytes(v)
	case uint32:
		return pb.Type_UINT32, uint32ToBytes(v)
	case uint64:
		return pb.Type_UINT64, uint64ToBytes(v)

	// 浮点数
	case float32:
		return pb.Type_FLOAT32, float32ToBytes(v)
	case float64:
		return pb.Type_FLOAT64, float64ToBytes(v)

	// 布尔值
	case bool:
		if v {
			return pb.Type_BOOL, []byte{1}
		}
		return pb.Type_BOOL, []byte{0}

	// 字符串和字符
	case string:
		return pb.Type_STRING, []byte(v)
	case []byte:
		return pb.Type_BYTES, v

	// 复数
	case complex64:
		return pb.Type_COMPLEX64, complex64ToBytes(v)
	case complex128:
		return pb.Type_COMPLEX128, complex128ToBytes(v)

	// 时间
	case time.Time:
		return pb.Type_TIME, timeToBytes(v)
	default:
		return pb.Type_OBJECT, JsonEncodeToBytes(v)
	}
} // }}}

// ================= DataToBytes 辅助转换函数 =================
// {{{
// int 转换（平台相关）
func intToBytes(v int) []byte { // {{{
	if unsafe.Sizeof(int(0)) == 4 {
		// 32位系统
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(v))
		return buf
	}

	// 64位系统
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(v))
	return buf
} // }}}

func int16ToBytes(v int16) []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(v))
	return buf
}

func int32ToBytes(v int32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(v))
	return buf
}

func int64ToBytes(v int64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(v))
	return buf
}

// uint 转换
func uintToBytes(v uint) []byte { // {{{
	if unsafe.Sizeof(uint(0)) == 4 {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(v))
		return buf
	}
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(v))
	return buf
} // }}}

func uint16ToBytes(v uint16) []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, v)
	return buf
}

func uint32ToBytes(v uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v)
	return buf
}

func uint64ToBytes(v uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, v)
	return buf
}

// 浮点数转换
func float32ToBytes(v float32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
	return buf
}

func float64ToBytes(v float64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(v))
	return buf
}

// 复数转换
func complex64ToBytes(v complex64) []byte {
	realPart := math.Float32bits(real(v))
	imagPart := math.Float32bits(imag(v))
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], realPart)
	binary.LittleEndian.PutUint32(buf[4:8], imagPart)
	return buf
}

func complex128ToBytes(v complex128) []byte {
	realPart := math.Float64bits(real(v))
	imagPart := math.Float64bits(imag(v))
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:8], realPart)
	binary.LittleEndian.PutUint64(buf[8:16], imagPart)
	return buf
}

func timeToBytes(v time.Time) []byte {
	nano := v.UnixNano()
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(nano))
	return buf
}

// }}}

func BytesToData(typ pb.Type, data []byte) any { // {{{
	switch typ {
	case pb.Type_NIL:
		return nil

	// 有符号整数
	case pb.Type_INT:
		return bytesToInt(data)
	case pb.Type_INT8:
		if len(data) < 1 {
			return int8(0)
		}
		return int8(data[0])
	case pb.Type_INT16:
		return bytesToInt16(data)
	case pb.Type_INT32:
		return bytesToInt32(data)
	case pb.Type_INT64:
		return bytesToInt64(data)

	// 无符号整数
	case pb.Type_UINT:
		return bytesToUint(data)
	case pb.Type_UINT8:
		if len(data) < 1 {
			return uint8(0)
		}
		return uint8(data[0])
	case pb.Type_UINT16:
		return bytesToUint16(data)
	case pb.Type_UINT32:
		return bytesToUint32(data)
	case pb.Type_UINT64:
		return bytesToUint64(data)

	// 浮点数
	case pb.Type_FLOAT32:
		return bytesToFloat32(data)
	case pb.Type_FLOAT64:
		return bytesToFloat64(data)

	// 布尔值
	case pb.Type_BOOL:
		if len(data) < 1 {
			return false
		}
		return data[0] != 0

	// 字符串和字节
	case pb.Type_STRING:
		return string(data)
	case pb.Type_BYTES:
		// 返回新的切片，避免修改原数据
		result := make([]byte, len(data))
		copy(result, data)
		return result

	// 复数
	case pb.Type_COMPLEX64:
		return bytesToComplex64(data)
	case pb.Type_COMPLEX128:
		return bytesToComplex128(data)

	// 时间
	case pb.Type_TIME:
		return bytesToTime(data)

	// 对象
	case pb.Type_OBJECT:
		return JsonDecodeBytes(data)

	default:
		return string(data)
	}
} // }}}

// ================= BytesToData 辅助转换函数 =================
// {{{
func bytesToInt(data []byte) int { // {{{
	if len(data) == 0 {
		return 0
	}
	if unsafe.Sizeof(int(0)) == 4 {
		if len(data) < 4 {
			// 填充不足的字节
			padded := make([]byte, 4)
			copy(padded, data)
			return int(binary.LittleEndian.Uint32(padded))
		}
		return int(binary.LittleEndian.Uint32(data))
	}
	if len(data) < 8 {
		padded := make([]byte, 8)
		copy(padded, data)
		return int(binary.LittleEndian.Uint64(padded))
	}
	return int(binary.LittleEndian.Uint64(data))
} // }}}

func bytesToInt16(data []byte) int16 { // {{{
	if len(data) < 2 {
		padded := make([]byte, 2)
		copy(padded, data)
		return int16(binary.LittleEndian.Uint16(padded))
	}
	return int16(binary.LittleEndian.Uint16(data))
} // }}}

func bytesToInt32(data []byte) int32 { // {{{
	if len(data) < 4 {
		padded := make([]byte, 4)
		copy(padded, data)
		return int32(binary.LittleEndian.Uint32(padded))
	}
	return int32(binary.LittleEndian.Uint32(data))
} // }}}

func bytesToInt64(data []byte) int64 { // {{{
	if len(data) < 8 {
		padded := make([]byte, 8)
		copy(padded, data)
		return int64(binary.LittleEndian.Uint64(padded))
	}
	return int64(binary.LittleEndian.Uint64(data))
} // }}}

func bytesToUint(data []byte) uint { // {{{
	if len(data) == 0 {
		return 0
	}
	if unsafe.Sizeof(uint(0)) == 4 {
		if len(data) < 4 {
			padded := make([]byte, 4)
			copy(padded, data)
			return uint(binary.LittleEndian.Uint32(padded))
		}
		return uint(binary.LittleEndian.Uint32(data))
	}
	if len(data) < 8 {
		padded := make([]byte, 8)
		copy(padded, data)
		return uint(binary.LittleEndian.Uint64(padded))
	}
	return uint(binary.LittleEndian.Uint64(data))
} // }}}

func bytesToUint16(data []byte) uint16 { // {{{
	if len(data) < 2 {
		padded := make([]byte, 2)
		copy(padded, data)
		return binary.LittleEndian.Uint16(padded)
	}
	return binary.LittleEndian.Uint16(data)
} // }}}

func bytesToUint32(data []byte) uint32 { // {{{
	if len(data) < 4 {
		padded := make([]byte, 4)
		copy(padded, data)
		return binary.LittleEndian.Uint32(padded)
	}
	return binary.LittleEndian.Uint32(data)
} // }}}

func bytesToUint64(data []byte) uint64 { // {{{
	if len(data) < 8 {
		padded := make([]byte, 8)
		copy(padded, data)
		return binary.LittleEndian.Uint64(padded)
	}
	return binary.LittleEndian.Uint64(data)
} // }}}

func bytesToFloat32(data []byte) float32 { // {{{
	if len(data) < 4 {
		padded := make([]byte, 4)
		copy(padded, data)
		return math.Float32frombits(binary.LittleEndian.Uint32(padded))
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
} // }}}

func bytesToFloat64(data []byte) float64 { // {{{
	if len(data) < 8 {
		padded := make([]byte, 8)
		copy(padded, data)
		return math.Float64frombits(binary.LittleEndian.Uint64(padded))
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(data))
} // }}}

func bytesToComplex64(data []byte) complex64 { // {{{
	if len(data) < 8 {
		// 填充不足的数据
		padded := make([]byte, 8)
		copy(padded, data)
		realPart := math.Float32frombits(binary.LittleEndian.Uint32(padded[0:4]))
		imagPart := math.Float32frombits(binary.LittleEndian.Uint32(padded[4:8]))
		return complex(realPart, imagPart)
	}
	realPart := math.Float32frombits(binary.LittleEndian.Uint32(data[0:4]))
	imagPart := math.Float32frombits(binary.LittleEndian.Uint32(data[4:8]))
	return complex(realPart, imagPart)
} // }}}

func bytesToComplex128(data []byte) complex128 { // {{{
	if len(data) < 16 {
		padded := make([]byte, 16)
		copy(padded, data)
		realPart := math.Float64frombits(binary.LittleEndian.Uint64(padded[0:8]))
		imagPart := math.Float64frombits(binary.LittleEndian.Uint64(padded[8:16]))
		return complex(realPart, imagPart)
	}
	realPart := math.Float64frombits(binary.LittleEndian.Uint64(data[0:8]))
	imagPart := math.Float64frombits(binary.LittleEndian.Uint64(data[8:16]))
	return complex(realPart, imagPart)
} // }}}

func bytesToTime(data []byte) time.Time { // {{{
	if len(data) < 8 {
		padded := make([]byte, 8)
		copy(padded, data)
		nano := int64(binary.LittleEndian.Uint64(padded))
		return time.Unix(0, nano)
	}
	nano := int64(binary.LittleEndian.Uint64(data))
	return time.Unix(0, nano)
} // }}}
//}}}
