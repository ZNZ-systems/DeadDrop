package handlers

import (
	"net/http"
	"net/url"
)

const (
	flashCookieName     = "flash"
	flashTypeCookieName = "flash_type"
	flashTypeError      = "error"
	flashTypeSuccess    = "success"
)

func setFlashError(w http.ResponseWriter, message string, secure bool) {
	setFlash(w, message, flashTypeError, secure)
}

func setFlashSuccess(w http.ResponseWriter, message string, secure bool) {
	setFlash(w, message, flashTypeSuccess, secure)
}

func setFlash(w http.ResponseWriter, message, flashType string, secure bool) {
	if message == "" {
		return
	}
	if flashType != flashTypeSuccess {
		flashType = flashTypeError
	}

	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(message),
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     flashTypeCookieName,
		Value:    flashType,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func consumeFlash(w http.ResponseWriter, r *http.Request, secure bool) (string, string) {
	msgCookie, err := r.Cookie(flashCookieName)
	if err != nil || msgCookie.Value == "" {
		return "", ""
	}

	rawMessage, err := url.QueryUnescape(msgCookie.Value)
	if err != nil {
		rawMessage = msgCookie.Value
	}

	flashType := flashTypeError
	if typeCookie, err := r.Cookie(flashTypeCookieName); err == nil && typeCookie.Value == flashTypeSuccess {
		flashType = flashTypeSuccess
	}

	clearCookie(w, flashCookieName, secure)
	clearCookie(w, flashTypeCookieName, secure)

	return rawMessage, flashType
}

func clearCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
