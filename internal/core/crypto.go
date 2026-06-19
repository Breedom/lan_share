package core

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	KeySize   = 32
	BlockSize = aes.BlockSize
)

type Crypto struct {
	key []byte
}

func NewCrypto(password string) *Crypto {
	key := deriveKey(password)
	return &Crypto{key: key}
}

func NewCryptoWithKey(key []byte) *Crypto {
	return &Crypto{key: key}
}

func deriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

func (c *Crypto) GenerateKeyPair() (publicKey, privateKey []byte, err error) {
	privateKey = make([]byte, 32)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, nil, err
	}

	publicKey = make([]byte, 32)
	copy(publicKey, privateKey)

	return publicKey, privateKey, nil
}

func (c *Crypto) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (c *Crypto) Decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (c *Crypto) EncryptFile(src io.Reader, dst io.Writer) error {
	buffer := make([]byte, 64*1024)

	for {
		n, err := src.Read(buffer)
		if n > 0 {
			encrypted, err := c.Encrypt(buffer[:n])
			if err != nil {
				return err
			}

			lenBuf := make([]byte, 4)
			binary.LittleEndian.PutUint32(lenBuf, uint32(len(encrypted)))

			if _, err := dst.Write(lenBuf); err != nil {
				return err
			}
			if _, err := dst.Write(encrypted); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Crypto) DecryptFile(src io.Reader, dst io.Writer) error {
	lenBuf := make([]byte, 4)

	for {
		_, err := io.ReadFull(src, lenBuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}

		dataLen := binary.LittleEndian.Uint32(lenBuf)
		data := make([]byte, dataLen)

		if _, err := io.ReadFull(src, data); err != nil {
			return err
		}

		decrypted, err := c.Decrypt(data)
		if err != nil {
			return err
		}

		if _, err := dst.Write(decrypted); err != nil {
			return err
		}
	}

	return nil
}

func (c *Crypto) Hash(data []byte) []byte {
	hash := sha256.Sum256(append(data, c.key...))
	return hash[:]
}

func (c *Crypto) Verify(data, expectedHash []byte) bool {
	actualHash := c.Hash(data)
	return bytes.Equal(actualHash, expectedHash)
}
