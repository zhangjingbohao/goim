package main

import (
	log "code.google.com/p/log4go"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/sha256"
	myrsa "github.com/Terry-Mao/goim/libs/crypto/rsa"
	"io/ioutil"
	"os"
)

var (
	RSAPri *rsa.PrivateKey
)

func InitRSA() (err error) {
	var (
		pri  []byte
		file *os.File
	)
	if file, err = os.Open(Conf.RSAPrivate); err != nil {
		log.Error("os.Open(\"%s\") error(%v)", Conf.RSAPrivate, err)
		return
	}
	if pri, err = ioutil.ReadAll(file); err != nil {
		log.Error("ioutil.ReadAll(\"%s\") error(%v)", Conf.RSAPrivate, err)
		return
	}
	log.Debug("private.pem : \n%s", string(pri))
	if RSAPri, err = myrsa.PrivateKey(pri); err != nil {
		log.Error("myrsa.PrivateKey() error(%v)", err)
	}
	return
}

type Cryptor interface {
	Exchange(*rsa.PrivateKey, []byte) (cipher.BlockMode, cipher.BlockMode, error) // use rsa exchange aes key&iv
	Encrypt(cipher.BlockMode, []byte) ([]byte, error)                             // aes encrypt
	Decrypt(cipher.BlockMode, []byte) ([]byte, error)                             // aes decrypt
}

type DefaultCryptor struct {
	dataLen int
	keyLen  int
}

func (c *DefaultCryptor) Exchange(pri *rsa.PrivateKey, cipherText []byte) (encryptor cipher.BlockMode, decryptor cipher.BlockMode, err error) {
	var (
		d     []byte
		key   []byte
		iv    []byte
		block cipher.Block
	)
	if d, err = rsa.DecryptOAEP(sha256.New(), nil, pri, cipherText, nil); err != nil {
		log.Error("rsa.DecryptOAEP() error(%v)", err)
		return
	}
	if len(d) != c.dataLen {
		log.Warn("handshake aes key size not valid: %d", len(d))
		err = ErrHandshake
		return
	}
	key = d[:c.keyLen]
	if block, err = aes.NewCipher(key); err != nil {
		log.Error("aes.NewCipher() error(%v)", err)
		return
	}
	iv = d[c.keyLen:]
	log.Debug("aes key: %x, iv: %x", key, iv)
	encryptor = cipher.NewCBCEncrypter(block, iv)
	decryptor = cipher.NewCBCDecrypter(block, iv)
	return
}

func (c *DefaultCryptor) Encrypt(encryptor cipher.BlockMode, msg []byte) (cipherText []byte, err error) {
	if msg != nil {
		// let caller do pkcs7 padding
		// msg = padding.PKCS7.Padding(msg, encryptor.BlockSize())
		if len(msg) < encryptor.BlockSize() || len(msg)%encryptor.BlockSize() != 0 {
			err = ErrInputTextSize
			return
		}
		cipherText = msg
		encryptor.CryptBlocks(cipherText, msg)
	}
	return
}

func (c *DefaultCryptor) Decrypt(decryptor cipher.BlockMode, cipherText []byte) (msg []byte, err error) {
	if decryptor != nil {
		if len(cipherText) < decryptor.BlockSize() || len(cipherText)%decryptor.BlockSize() != 0 {
			err = ErrInputTextSize
			return
		}
		msg = cipherText
		decryptor.CryptBlocks(msg, cipherText)
		// let caller do pkcs7 unpadding
		// msg, err = padding.PKCS7.Unpadding(msg, decryptor.BlockSize())
	}
	return
}