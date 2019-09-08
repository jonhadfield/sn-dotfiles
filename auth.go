package sndotfiles

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/viper"
	keyring "github.com/zalando/go-keyring"
)

// GetCredentials is used to obtain the SN credentials via the CLI if not specified using envvars
func GetCredentials(inServer string) (email, password, apiServer, errMsg string) {
	switch {
	case viper.GetString("email") != "":
		email = viper.GetString("email")
	default:
		fmt.Print("email: ")
		_, err := fmt.Scanln(&email)
		if err != nil || len(strings.TrimSpace(email)) == 0 {
			errMsg = "email required"
			return
		}
	}

	if viper.GetString("password") != "" {
		password = viper.GetString("password")
	} else {
		fmt.Print("password: ")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err == nil {
			password = string(bytePassword)
		} else {
			errMsg = err.Error()
			return
		}
		if strings.TrimSpace(password) == "" {
			errMsg = "password not defined"
		}
	}

	switch {
	case inServer != "":
		apiServer = inServer
	case viper.GetString("server") != "":
		apiServer = viper.GetString("server")
	default:
		apiServer = SNServerURL
	}
	return email, password, apiServer, errMsg
}

func padToAESBlockSize(b []byte) []byte {
	n := aes.BlockSize - (len(b) % aes.BlockSize)
	pb := make([]byte, len(b)+n)
	copy(pb, b)
	copy(pb[len(b):], bytes.Repeat([]byte{byte(n)}, n))
	return pb
}

// encrypt string to base64 crypto using AES
func Encrypt(key []byte, text string) string {
	key = padToAESBlockSize(key)
	plaintext := []byte(text)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext)
}

// decrypt from base64 to decrypted string
func Decrypt(key []byte, cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)
	key = padToAESBlockSize(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext)
}

const Service = "StandardNotesCLI"

func GetSessionFromKeyring(key string) (session string, err error) {
	var rS string
	rS, err = keyring.Get(Service, KeyringApplicationName)
	if err != nil {
		return
	}
	// check if session is encrypted
	if len(strings.Split(rS, ";")) != 5 {
		if key == "" {
			fmt.Printf("encryption key: ")
			var byteKey []byte
			byteKey, err = terminal.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err == nil {
				key = string(byteKey)
			}
			if len(strings.TrimSpace(key)) == 0 {
				err = fmt.Errorf("password required")
			}
		}
		session = Decrypt([]byte(key), rS)
		if len(strings.Split(session, ";")) != 5 {
			err = fmt.Errorf("invalid session or wrong encryption key provided")
		}
	} else {
		session = rS
	}
	return session, err
}
