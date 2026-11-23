// Package core holds the platform logic ported from the C# implementation:
// config storage, RDP file generation and the two Windows crypto calls.
package core

import (
	"crypto/cipher"
	"crypto/des"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Same key/iv as StringUtil.cs, so existing config.json passwords keep working.
var (
	desKey = []byte{226, 175, 129, 22, 187, 4, 143, 171}
	desIV  = []byte{181, 239, 77, 174, 238, 148, 206, 164}
)

func toUTF16LE(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return b
}

func fromUTF16LE(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[i*2]) | uint16(b[i*2+1])<<8
	}
	return string(utf16.Decode(u))
}

func pkcs7Pad(b []byte, blockSize int) []byte {
	n := blockSize - len(b)%blockSize
	for i := 0; i < n; i++ {
		b = append(b, byte(n))
	}
	return b
}

func pkcs7Unpad(b []byte, blockSize int) ([]byte, error) {
	n := int(b[len(b)-1])
	if n == 0 || n > blockSize || n > len(b) {
		return nil, errors.New("invalid PKCS7 padding")
	}
	return b[:len(b)-n], nil
}

// EncryptConfigPassword mirrors StringUtil.Encrypt (DES-CBC/PKCS7 over UTF-16LE, base64).
func EncryptConfigPassword(plain string) (string, error) {
	block, err := des.NewCipher(desKey)
	if err != nil {
		return "", err
	}
	in := pkcs7Pad(toUTF16LE(plain), block.BlockSize())
	out := make([]byte, len(in))
	cipher.NewCBCEncrypter(block, desIV).CryptBlocks(out, in)
	return base64.StdEncoding.EncodeToString(out), nil
}

// DecryptConfigPassword mirrors StringUtil.Decrypt.
func DecryptConfigPassword(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := des.NewCipher(desKey)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw)%block.BlockSize() != 0 {
		return "", errors.New("ciphertext is not a multiple of the block size")
	}
	out := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, desIV).CryptBlocks(out, raw)
	out, err = pkcs7Unpad(out, block.BlockSize())
	if err != nil {
		return "", err
	}
	return fromUTF16LE(out), nil
}

// EncryptMstscPassword mirrors Mstscpw.encryptpw: DPAPI CryptProtectData over
// UTF-16LE, returned as uppercase hex for the "password 51:b:" rdp key.
func EncryptMstscPassword(plain string) (string, error) {
	in := toUTF16LE(plain)
	if len(in) == 0 {
		return "", errors.New("empty password")
	}

	var dataIn, dataOut windows.DataBlob
	dataIn.Size = uint32(len(in))
	dataIn.Data = &in[0]

	desc, err := windows.UTF16PtrFromString("psw")
	if err != nil {
		return "", err
	}

	if err := windows.CryptProtectData(&dataIn, desc, nil, 0, nil,
		windows.CRYPTPROTECT_UI_FORBIDDEN, &dataOut); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(dataOut.Data)))

	return strings.ToUpper(hex.EncodeToString(unsafe.Slice(dataOut.Data, dataOut.Size))), nil
}

// decryptMstscPassword is what mstsc.exe does with the "password 51:b:" value:
// DPAPI CryptUnprotectData for the current user. Used by the tests to prove the
// blob we write really decrypts to the saved password.
func decryptMstscPassword(blobHex string) (string, error) {
	raw, err := hex.DecodeString(strings.ToLower(strings.TrimSpace(blobHex)))
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", errors.New("empty blob")
	}

	var dataIn, dataOut windows.DataBlob
	dataIn.Size = uint32(len(raw))
	dataIn.Data = &raw[0]

	if err := windows.CryptUnprotectData(&dataIn, nil, nil, 0, nil,
		windows.CRYPTPROTECT_UI_FORBIDDEN, &dataOut); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(dataOut.Data)))

	return fromUTF16LE(unsafe.Slice(dataOut.Data, dataOut.Size)), nil
}
