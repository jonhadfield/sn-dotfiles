package sndotfiles

//// GetCredentials is used to obtain the SN credentials via the CLI if not specified using envvars
//func GetCredentials(inServer string) (email, password, apiServer, errMsg string) {
//	switch {
//	case viper.GetString("email") != "":
//		email = viper.GetString("email")
//	default:
//		fmt.Print("email: ")
//		_, err := fmt.Scanln(&email)
//		if err != nil || len(strings.TrimSpace(email)) == 0 {
//			errMsg = "email required"
//			return
//		}
//	}
//
//	if viper.GetString("password") != "" {
//		password = viper.GetString("password")
//	} else {
//		fmt.Print("password: ")
//		bytePassword, err := terminal.ReadPassword(syscall.Stdin)
//		fmt.Println()
//		if err == nil {
//			password = string(bytePassword)
//		} else {
//			errMsg = err.Error()
//			return
//		}
//		if strings.TrimSpace(password) == "" {
//			errMsg = "password not defined"
//		}
//	}
//
//	switch {
//	case inServer != "":
//		apiServer = inServer
//	case viper.GetString("server") != "":
//		apiServer = viper.GetString("server")
//	default:
//		apiServer = SNServerURL
//	}
//	return email, password, apiServer, errMsg
//}

//func padToAESBlockSize(b []byte) []byte {
//	n := aes.BlockSize - (len(b) % aes.BlockSize)
//	pb := make([]byte, len(b)+n)
//	copy(pb, b)
//	copy(pb[len(b):], bytes.Repeat([]byte{byte(n)}, n))
//	return pb
//}

// encrypt string to base64 crypto using AES
//func Encrypt(key []byte, text string) string {
//	key = padToAESBlockSize(key)
//	plaintext := []byte(text)
//	block, err := aes.NewCipher(key)
//	if err != nil {
//		panic(err)
//	}
//
//	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
//	iv := ciphertext[:aes.BlockSize]
//	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
//		panic(err)
//	}
//
//	stream := cipher.NewCFBEncrypter(block, iv)
//	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
//
//	// convert to base64
//	return base64.URLEncoding.EncodeToString(ciphertext)
//}

// decrypt from base64 to decrypted string
//func Decrypt(key []byte, cryptoText string) (pt string, err error) {
//	var ciphertext []byte
//	if ciphertext, err = base64.URLEncoding.DecodeString(cryptoText); err != nil {
//		return
//	}
//	key = padToAESBlockSize(key)
//	var block cipher.Block
//	if block, err = aes.NewCipher(key); err != nil {
//		return
//	}
//
//	if len(ciphertext) < aes.BlockSize {
//		return "", errors.New("ciphertext too short")
//	}
//	iv := ciphertext[:aes.BlockSize]
//	ciphertext = ciphertext[aes.BlockSize:]
//
//	stream := cipher.NewCFBDecrypter(block, iv)
//	stream.XORKeyStream(ciphertext, ciphertext)
//
//	pt = fmt.Sprintf("%s", ciphertext)
//	return
//}

//func GetSessionFromKeyring(key string) (session string, err error) {
//	var rS string
//	rS, err = keyring.Get(KeyringService, KeyringApplicationName)
//	if err != nil {
//		return
//	}
//	// check if Session is encrypted
//	if len(strings.Split(rS, ";")) != 5 {
//		if key == "" {
//			fmt.Printf("encryption key: ")
//			var byteKey []byte
//			byteKey, err = terminal.ReadPassword(syscall.Stdin)
//			fmt.Println()
//			if err == nil {
//				key = string(byteKey)
//			}
//			if len(strings.TrimSpace(key)) == 0 {
//				err = fmt.Errorf("key required")
//				return
//			}
//		}
//		if session, err = auth.Decrypt([]byte(key), rS); err != nil {
//			return
//		}
//		if len(strings.Split(session, ";")) != 5 {
//			err = fmt.Errorf("invalid session or wrong key provided")
//		}
//	} else {
//		session = rS
//	}
//	return session, err
//}
