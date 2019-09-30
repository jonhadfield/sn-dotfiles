package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/jonhadfield/gosn"
	"github.com/jonhadfield/sn-cli/auth"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/ssh/terminal"
	"strings"
	"syscall"
)

const (
	MsgSessionRemovalSuccess = "session removed successfully"
	MsgSessionRemovalFailure = "failed to remove session"
)

func SessionExists(k keyring.Keyring) error {
	s, err := GetSessionFromKeyring(k)
	if err != nil {
		return err
	}
	if len(s) == 0 {
		return errors.New("session is empty")
	}
	return nil
}

func GetSessionFromKeyring(k keyring.Keyring) (s string, err error) {
	if k == nil {
		return keyring.Get(KeyringService, KeyringApplicationName)
	}
	return k.Get(KeyringService, KeyringApplicationName)
}

func AddSession(snServer, inKey string, k keyring.Keyring) (res string, err error) {
	// check if session exists in keyring
	var s string
	s, err = GetSessionFromKeyring(k)
	// only return an error if there's an issue accessing the keyring
	if err != nil && !strings.Contains(err.Error(), "secret not found in keyring") {
		return
	}

	if inKey == "." {
		var byteKey []byte
		fmt.Print("session key: ")
		byteKey, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return
		}
		inKey = string(byteKey)
		fmt.Println()
	}

	if s != "" {
		fmt.Print("replace existing session (y|n): ")
		var resp string
		_, err := fmt.Scanln(&resp)
		if err != nil || strings.ToLower(resp) != "y" {
			// do nothing
			return "", nil
		}
	}
	var session gosn.Session
	var email string
	session, email, err = GetSessionFromUser(snServer)
	if err != nil {
		return fmt.Sprint("failed to get session: ", err), err
	}

	rS := makeSessionString(email, session)
	if inKey != "" {
		key := []byte(inKey)
		rS = auth.Encrypt(key, makeSessionString(email, session))
	}
	err = writeSession(rS, k)
	if err != nil {
		return fmt.Sprint("failed to set session: ", err), err
	}
	return "session added successfully", err
}

func writeSession(s string, k keyring.Keyring) error {
	if k == nil {
		return keyring.Set(KeyringService, KeyringApplicationName, s)
	}
	return k.Set(KeyringService, KeyringApplicationName, s)
}

// RemoveSession removes the SN session from the keyring
func RemoveSession(k keyring.Keyring) string {
	var err error
	if err = SessionExists(k); err != nil {
		return fmt.Sprintf("%s: %s", MsgSessionRemovalFailure, err.Error())
	}
	if k == nil {
		err = keyring.Delete(KeyringService, KeyringApplicationName)
	} else {
		err = k.Delete(KeyringService, KeyringApplicationName)
	}
	if err != nil {
		return fmt.Sprintf("%s: %s", MsgSessionRemovalFailure, err.Error())
	}
	return MsgSessionRemovalSuccess
}

func makeSessionString(email string, session gosn.Session) string {
	return fmt.Sprintf("%s;%s;%s;%s;%s", email, session.Server, session.Token, session.Ak, session.Mk)
}

func SessionStatus(sKey string, k keyring.Keyring) (msg string, err error) {
	var s, keyringContent string
	keyringContent, err = k.Get(KeyringService, KeyringApplicationName)
	if keyringContent == "" {
		return "", errors.New("keyring is empty")
	}
	s, err = auth.GetSessionFromKeyring(sKey, k)
	if err != nil {
		if strings.Contains(err.Error(), "illegal base64") {
			err = errors.New("stored session is corrupt")
		}
		return
	}
	var email string
	email, _, err = ParseSessionString(s)
	if err != nil {
		msg = fmt.Sprint("failed to parse session: ", err)
		return
	}
	msg = fmt.Sprint("session found: ", email)
	return
}
