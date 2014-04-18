package encrypt

/*
Usage:

c := MyCrypto{key: "<your 32 bits encryption key>"}
plaintext := "hello world"
encrypted := c.encrypt(plaintext)
decrypted := c.decrypt(encrypted)
fmt.Println(encrypted)
fmt.Println(decrypted)

*/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
    "encoding/hex"
    "wapa/config"
    "os/user"
    "fmt"
)

type MyCrypto struct {
    key string
}
type CipherTextTooShortError struct {}

func NewCrypto() (*MyCrypto, error) {
    u, err := user.Current()
    if err != nil {
        return nil, err
    }
    myConfig, err := config.NewConfig(fmt.Sprintf("%s/.waparc", u.HomeDir))
    if err != nil {
        return nil, err
    }
    return &MyCrypto{key: myConfig.Encryption_key}, nil
}

func (err CipherTextTooShortError) Error() string {
    return "cipher text too short"
}

func (myCrypto *MyCrypto) getKey() []byte {
    return []byte(myCrypto.key)
}

func (myCrypto *MyCrypto) encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func (myCrypto *MyCrypto) decodeBase64(s string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (myCrypto *MyCrypto) hexEncode(src []byte) string {
    dst := make([]byte, hex.EncodedLen(len(src)))
    hex.Encode(dst, src)
    return string(dst)
}

func (myCrypto *MyCrypto) hexDecode(srcStr string) ([]byte, error) {
    src := []byte(srcStr)
    dst := make([]byte, hex.DecodedLen(len(src)))
    _, err := hex.Decode(dst, src)
    if err != nil {
        return nil, err
    }
    return dst, nil
}

func (myCrypto *MyCrypto) Encrypt(textStr string) (string, error) {
    key := myCrypto.getKey()
	text := []byte(textStr)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	b := myCrypto.encodeBase64(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return myCrypto.hexEncode(ciphertext), nil
}

func (myCrypto *MyCrypto) Decrypt(textStr string) (string, error) {
    key := myCrypto.getKey()
	text, err := myCrypto.hexDecode(textStr)
    if err != nil {
        return "", err
    }
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(text) < aes.BlockSize {
        return "", CipherTextTooShortError{}
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
    result, err := myCrypto.decodeBase64(string(text))
    if err != nil {
        return "", err
    }
	return string(result), nil
}
