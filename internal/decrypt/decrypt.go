package decrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltStr       = "saltiest"
	ivStr         = "the iv: 16 bytes"
	fileHeader    = "V1MMWX"
	defaultXorKey = 0x66
)

func DecryptWxapkg(inputFile, appID string) ([]byte, error) {
	ciphertext, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	// 先检查是否已解密
	reader := bytes.NewReader(ciphertext)
	var firstMark byte
	binary.Read(reader, binary.BigEndian, &firstMark)
	var info1, indexInfoLength, bodyInfoLength uint32
	binary.Read(reader, binary.BigEndian, &info1)
	binary.Read(reader, binary.BigEndian, &indexInfoLength)
	binary.Read(reader, binary.BigEndian, &bodyInfoLength)
	var lastMark byte
	binary.Read(reader, binary.BigEndian, &lastMark)
	if firstMark == 0xBE && lastMark == 0xED {
		// 已解密直接返回
		return ciphertext, nil
	}

	if string(ciphertext[:len(fileHeader)]) != fileHeader {
		return nil, fmt.Errorf("无效的文件格式")
	}

	key := pbkdf2.Key([]byte(appID), []byte(saltStr), 1000, 32, sha1.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES密码块失败: %v", err)
	}

	iv := []byte(ivStr)
	mode := cipher.NewCBCDecrypter(block, iv)
	originData := make([]byte, 1024)
	mode.CryptBlocks(originData, ciphertext[6:1024+6])

	afData := make([]byte, len(ciphertext)-1024-6)
	var xorKey byte
	if len(appID) >= 2 {
		xorKey = appID[len(appID)-2]
	} else {
		xorKey = defaultXorKey
	}
	for i, b := range ciphertext[1024+6:] {
		afData[i] = b ^ xorKey
	}

	originData = append(originData[:1023], afData...)

	return originData, nil
}
