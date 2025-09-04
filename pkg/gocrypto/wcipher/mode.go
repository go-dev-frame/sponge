package wcipher

import (
	"crypto/cipher"
)

// CipherMode provides a uniform interface to set the filling method for different operating modes.
type CipherMode interface {
	SetPadding(padding Padding) CipherMode
	Cipher(block cipher.Block, iv []byte) Cipher
}

type cipherMode struct {
	padding Padding
}

// SetPadding set padding
func (c *cipherMode) SetPadding(padding Padding) CipherMode {
	_ = padding
	return c
}

// Cipher mode cipher
func (c *cipherMode) Cipher(block cipher.Block, iv []byte) Cipher {
	_ = block
	_ = iv
	return nil
}

type ecbCipherModel cipherMode

// NewECBMode new ecb mode
func NewECBMode() CipherMode {
	return &ecbCipherModel{padding: NewPKCS57Padding()}
}

// SetPadding set ecb padding
func (ecb *ecbCipherModel) SetPadding(padding Padding) CipherMode {
	ecb.padding = padding
	return ecb
}

// Cipher ecb cipher
func (ecb *ecbCipherModel) Cipher(block cipher.Block, iv []byte) Cipher {
	_ = iv
	encrypter := NewECBEncrypt(block)
	decrypter := NewECBDecrypt(block)
	return NewBlockCipher(ecb.padding, encrypter, decrypter)
}

type cbcCipherModel cipherMode

// NewCBCMode new cbc mode
func NewCBCMode() CipherMode {
	return &cbcCipherModel{padding: NewPKCS57Padding()}
}

// SetPadding set cbc padding
func (cbc *cbcCipherModel) SetPadding(padding Padding) CipherMode {
	cbc.padding = padding
	return cbc
}

// Cipher cbc cipher
func (cbc *cbcCipherModel) Cipher(block cipher.Block, iv []byte) Cipher {
	encrypter := cipher.NewCBCEncrypter(block, iv)
	decrypter := cipher.NewCBCDecrypter(block, iv)
	return NewBlockCipher(cbc.padding, encrypter, decrypter)
}

type cfbCipherModel struct {
	cipherMode
}

// NewCFBMode new cfb mode
func NewCFBMode() CipherMode {
	return &cfbCipherModel{}
}

// Cipher cfb cipher
func (cfb *cfbCipherModel) Cipher(block cipher.Block, iv []byte) Cipher { //nolint
	// CFB is deprecated; prefer CTR (unauthenticated stream) or AEAD
	encrypter := cipher.NewCTR(block, iv)
	decrypter := cipher.NewCTR(block, iv)
	return NewStreamCipher(encrypter, decrypter)
}

type ofbCipherModel struct {
	cipherMode
}

// NewOFBMode new ofb mode
func NewOFBMode() CipherMode {
	return &ofbCipherModel{}
}

// Cipher ofb cipher
func (ofb *ofbCipherModel) Cipher(block cipher.Block, iv []byte) Cipher {
	// OFB is deprecated; prefer CTR (unauthenticated stream) or AEAD
	encrypter := cipher.NewCTR(block, iv)
	decrypter := cipher.NewCTR(block, iv)
	return NewStreamCipher(encrypter, decrypter)
}

type ctrCipherModel struct {
	cipherMode
}

// NewCTRMode new ctr mode
func NewCTRMode() CipherMode {
	return &ctrCipherModel{}
}

// Cipher ctr cipher
func (ctr *ctrCipherModel) Cipher(block cipher.Block, iv []byte) Cipher {
	encrypter := cipher.NewCTR(block, iv)
	decrypter := cipher.NewCTR(block, iv)
	return NewStreamCipher(encrypter, decrypter)
}
