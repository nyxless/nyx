package x

import (
	"sync/atomic"
)

// 使用原子指针存储 map，支持并发安全地整体替换data
type FreeMap[K comparable, V any] struct {
	data atomic.Value // 存储 *map[K]V（map 的指针）
}

func NewFreeMap[K comparable, V any]() *FreeMap[K, V] {
	fm := &FreeMap[K, V]{}
	// 初始化为空 map 的指针
	emptyMap := make(map[K]V)
	fm.data.Store(&emptyMap)
	return fm
}

func NewAnyFreeMap() *FreeMap[string, any] {
	return NewFreeMap[string, any]()
}

// 原子替换整个map（注意：这里存储的是指针）
func (fm *FreeMap[K, V]) Update(newMap map[K]V) {
	fm.data.Store(&newMap)
}

// 基于当前状态安全更新
func (fm *FreeMap[K, V]) SafeUpdate(updater func(old map[K]V) map[K]V) {
	for {
		oldPtr := fm.data.Load().(*map[K]V)
		old := *oldPtr
		newMap := updater(old)

		// 使用 CompareAndSwap 确保在更新过程中没有被其他goroutine修改
		if fm.data.CompareAndSwap(oldPtr, &newMap) {
			return
		}
		// 如果CAS失败，重试
	}
}

// 读
func (fm *FreeMap[K, V]) Get(key K) (V, bool) {
	dataPtr := fm.data.Load().(*map[K]V)
	val, ok := (*dataPtr)[key]
	return val, ok
}

// 写
func (fm *FreeMap[K, V]) Set(key K, value V) {
	for {
		oldPtr := fm.data.Load().(*map[K]V)
		old := *oldPtr

		// 如果 key 已存在且值相同，避免不必要的复制
		if oldVal, exists := old[key]; exists && any(oldVal) == any(value) {
			return
		}

		// 创建新 map（复制所有数据）
		newMap := make(map[K]V, len(old)+1)
		for k, v := range old {
			newMap[k] = v
		}
		newMap[key] = value

		// 原子替换指针
		if fm.data.CompareAndSwap(oldPtr, &newMap) {
			return
		}
		// 如果CAS失败，重试
	}
}

// 删除
func (fm *FreeMap[K, V]) Delete(key K) {
	for {
		oldPtr := fm.data.Load().(*map[K]V)
		old := *oldPtr

		if _, exists := old[key]; !exists {
			return
		}

		// 创建新 map（排除要删除的 key）
		newMap := make(map[K]V, len(old))
		for k, v := range old {
			if k != key {
				newMap[k] = v
			}
		}

		// 原子替换指针
		if fm.data.CompareAndSwap(oldPtr, &newMap) {
			return
		}
		// 如果CAS失败，重试
	}
}

// 获取map长度
func (fm *FreeMap[K, V]) Len() int {
	dataPtr := fm.data.Load().(*map[K]V)
	return len(*dataPtr)
}

// 遍历键值对
func (fm *FreeMap[K, V]) Range(f func(key K, value V) bool) {
	dataPtr := fm.data.Load().(*map[K]V)
	data := *dataPtr
	for k, v := range data {
		if !f(k, v) {
			return
		}
	}
}

// 取单条样本数据
func (fm *FreeMap[K, V]) Sample() (K, V, bool) {
	dataPtr := fm.data.Load().(*map[K]V)
	data := *dataPtr
	for k, v := range data {
		return k, v, true
	}

	var zeroK K
	var zeroV V
	return zeroK, zeroV, false
}

// 获取当前map的副本
func (fm *FreeMap[K, V]) Copy() map[K]V {
	dataPtr := fm.data.Load().(*map[K]V)
	data := *dataPtr
	copyMap := make(map[K]V, len(data))
	for k, v := range data {
		copyMap[k] = v
	}
	return copyMap
}

// 原子比较并替换换
func (fm *FreeMap[K, V]) CompareAndSwap(oldMap, newMap map[K]V) bool {
	oldPtr := &oldMap
	newPtr := &newMap
	return fm.data.CompareAndSwap(oldPtr, newPtr)
}
