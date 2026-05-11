package main

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	coreKeyHex = "687A4852416D736F356B496E62617857"
	metaKeyHex = "2331346C6A6B5F215C5D2630553C2728"
)

type decodeResult struct {
	Path   string
	Format string
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("AES 解密结果为空")
	}
	pad := int(data[len(data)-1])
	if pad <= 0 || pad > 16 {
		return nil, errors.New("AES 填充无效")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, errors.New("AES 填充校验失败")
		}
	}
	return data[:len(data)-pad], nil
}

func aesECBDecrypt(ciphertext []byte, keyHex string) ([]byte, error) {
	if len(ciphertext)%16 != 0 {
		return nil, errors.New("NCM 数据块长度异常")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("key hex decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES init: %w", err)
	}
	out := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += 16 {
		block.Decrypt(out[i:i+16], ciphertext[i:i+16])
	}
	return pkcs7Unpad(out)
}

func buildKeyBox(keyData []byte) [256]byte {
	var box [256]byte
	for i := range box {
		box[i] = byte(i)
	}
	var c, lastByte byte
	keyLen := len(keyData)
	keyOffset := 0
	for i := 0; i < 256; i++ {
		swap := box[i]
		c = swap + lastByte + keyData[keyOffset]
		keyOffset++
		if keyOffset >= keyLen {
			keyOffset = 0
		}
		box[i] = box[c]
		box[c] = swap
		lastByte = c
	}
	return box
}

func readLeU32(raw []byte, offset int) (uint32, int, error) {
	if offset+4 > len(raw) {
		return 0, offset, errors.New("NCM 数据结构损坏")
	}
	return binary.LittleEndian.Uint32(raw[offset : offset+4]), offset + 4, nil
}

func decodeNcmToTemp(src string) (*decodeResult, error) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	if len(raw) < 16 {
		return nil, errors.New("NCM 文件过小或已损坏")
	}
	if string(raw[:8]) != "CTENFDAM" {
		return nil, errors.New("不是有效的 .ncm 文件")
	}

	offset := 10

	keyLen, offset, err := readLeU32(raw, offset)
	if err != nil {
		return nil, err
	}
	keyBlock := make([]byte, keyLen)
	copy(keyBlock, raw[offset:offset+int(keyLen)])
	offset += int(keyLen)
	for i := range keyBlock {
		keyBlock[i] ^= 0x64
	}

	keyData, err := aesECBDecrypt(keyBlock, coreKeyHex)
	if err != nil {
		return nil, fmt.Errorf("key 解密失败: %w", err)
	}
	if len(keyData) <= 17 {
		return nil, errors.New("NCM key 解密结果异常")
	}
	keyData = keyData[17:]
	keyBox := buildKeyBox(keyData)

	metaLen, offset, err := readLeU32(raw, offset)
	if err != nil {
		return nil, err
	}
	formatHint := "audio"
	if metaLen > 0 && int(metaLen) <= len(raw)-offset {
		metaBlock := make([]byte, metaLen)
		copy(metaBlock, raw[offset:offset+int(metaLen)])
		for i := range metaBlock {
			metaBlock[i] ^= 0x63
		}
		if len(metaBlock) > 22 {
			decoded, err := base64.StdEncoding.DecodeString(string(metaBlock[22:]))
			if err == nil {
				plain, err := aesECBDecrypt(decoded, metaKeyHex)
				if err == nil && len(plain) > 6 {
					var meta map[string]interface{}
					if json.Unmarshal(plain[6:], &meta) == nil {
						if fmt, ok := meta["format"].(string); ok && fmt != "" {
							formatHint = strings.ToLower(strings.TrimSpace(fmt))
						}
					}
				}
			}
		}
	}
	offset += int(metaLen)

	// skip CRC32 (4 bytes) + padding (5 bytes)
	offset += 9

	imgSize, offset, err := readLeU32(raw, offset)
	if err != nil {
		return nil, err
	}
	offset += int(imgSize)

	if offset >= len(raw) {
		return nil, errors.New("NCM 音频数据为空")
	}

	audio := make([]byte, len(raw)-offset)
	copy(audio, raw[offset:])
	for i := range audio {
		j := byte(i + 1)
		k1 := keyBox[j]
		k2 := keyBox[k1+j]
		audio[i] ^= keyBox[k1+k2]
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("ncm_decoded_*.%s", formatHint))
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	if _, err := tmpFile.Write(audio); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmpFile.Close()

	return &decodeResult{Path: tmpFile.Name(), Format: formatHint}, nil
}
