package main

import (
	"fmt"
	"os"
	"sync"
)

// dvError 表示已确认的坏对象，不允许被重新写入。
const dvError = uint8(0x7F)

// KeyMap 持久化存储每个对象的 key 版本（1 字节/对象）。
//
// 字节格式：
//
//	0x00       = 从未写入
//	0x01–0x7E  = key 值 (1–126)，已写入版本
//	0x7F       = DV_ERROR，已确认坏对象
//	0x80–0xFF  = key | 0x80，写入进行中（busy 标志 = bit7）
type KeyMap struct {
	mu   sync.Mutex
	data []uint8
	path string
}

// NewKeyMap 打开或创建 path 处的 key map 文件。
// 如果文件不存在，创建并初始化为 objectCount 个零字节。
// 如果文件已存在，将其内容加载到内存。
func NewKeyMap(path string, objectCount int64) (*KeyMap, error) {
	km := &KeyMap{path: path}

	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read keymap file %s: %w", path, err)
		}
		// 文件不存在：创建并初始化为全零
		km.data = make([]uint8, objectCount)
		if err := os.WriteFile(path, km.data, 0644); err != nil {
			return nil, fmt.Errorf("failed to create keymap file %s: %w", path, err)
		}
	} else {
		// 文件已存在：校验长度后加载现有内容
		if int64(len(existing)) != objectCount {
			return nil, fmt.Errorf("keymap: file %s has %d entries, expected %d", path, len(existing), objectCount)
		}
		km.data = existing
	}

	return km, nil
}

// acquireBusy 以原子方式将对象标记为 busy（设置 bit7）。
// 返回旧的 key（清除 bit7 后的值）和 ok=true。
// 如果对象已处于 busy 状态（bit7 已置位），返回 0, false。
func (km *KeyMap) acquireBusy(objnum int64) (oldKey uint8, ok bool) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if objnum < 0 || objnum >= int64(len(km.data)) {
		return 0, false
	}
	current := km.data[objnum]
	if current&0x80 != 0 {
		// 已 busy
		return 0, false
	}
	if current == dvError {
		// DV_ERROR 对象不允许被重新写入
		return 0, false
	}
	km.data[objnum] = current | 0x80
	return current & 0x7F, true
}

// releaseBusy 释放对象的 busy 状态，将 key 写回 data[objnum]。
// 调用方负责决定传入新 key（成功时）还是旧 key（失败时恢复）。
func (km *KeyMap) releaseBusy(objnum int64, key uint8) {
	km.mu.Lock()
	defer km.mu.Unlock()
	if objnum < 0 || objnum >= int64(len(km.data)) {
		return
	}
	km.data[objnum] = key & 0x7F // 防御性掩码，确保 bit7 不被写入
}

// NextKey 返回下一个 key 值，循环：126 → 1。
func NextKey(oldKey uint8) uint8 {
	return (oldKey % 126) + 1
}

// Sync 将内存数据写回文件。
func (km *KeyMap) Sync() error {
	km.mu.Lock()
	defer km.mu.Unlock()
	if err := os.WriteFile(km.path, km.data, 0644); err != nil {
		return fmt.Errorf("failed to sync keymap to %s: %w", km.path, err)
	}
	return nil
}

// HasBusy 如果任意字节的 bit7 已置位则返回 true。
func (km *KeyMap) HasBusy() bool {
	km.mu.Lock()
	defer km.mu.Unlock()

	for _, b := range km.data {
		if b&0x80 != 0 {
			return true
		}
	}
	return false
}
