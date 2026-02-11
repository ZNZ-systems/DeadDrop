package handlers

import (
	"net/http"
	"time"

	"github.com/znz-systems/deaddrop/internal/auth"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

// AuthHandler handles HTTP requests for authentication routes.
type AuthHandler struct {
	auth   *auth.Service
	render *render.Renderer
}

// NewAuthHandler creates a new AuthHandler with the given auth service and renderer.
func NewAuthHandler(authService *auth.Service, renderer *render.Renderer) *AuthHandler {
	return &AuthHandler{
		auth:   authService,
		render: renderer,
	}
}

// ShowLogin renders the login page.
func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{}

	// Read flash message from cookie if present.
	if cookie, err := r.Cookie("flash"); err == nil {
		data["flash"] = cookie.Value
		// Clear the flash cookie.
		http.SetCookie(w, &http.Cookie{
			Name:     "flash",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
	}

	h.render.Render(w, r, "login.html", data)
}

// HandleLogin processes the login form submission.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setFlash(w, "Invalid form data.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	session, err := h.auth.Login(r.Context(), email, password)
	if err != nil {
		setFlash(w, "Invalid email or password.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ShowSignup renders the signup page.
func (h *AuthHandler) ShowSignup(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{}

	// Read flash message from cookie if present.
	if cookie, err := r.Cookie("flash"); err == nil {
		data["flash"] = cookie.Value
		// Clear the flash cookie.
		http.SetCookie(w, &http.Cookie{
			Name:     "flash",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
	}

	h.render.Render(w, r, "signup.html", data)
}

// HandleSignup processes the signup form submission.
func (h *AuthHandler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setFlash(w, "Invalid form data.")
		http.Redirect(w, r, "/signup", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if password != passwordConfirm {
		setFlash(w, "Passwords do not match.")
		http.Redirect(w, r, "/signup", http.StatusSeeOther)
		return
	}

	_, err := h.auth.Signup(r.Context(), email, password)
	if err != nil {
		setFlash(w, err.Error())
		http.Redirect(w, r, "/signup", http.StatusSeeOther)
		return
	}

	// Auto-login after successful signup.
	session, err := h.auth.Login(r.Context(), email, password)
	if err != nil {
		setFlash(w, "Account created. Please log in.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleLogout logs out the current user by deleting their session and clearing the cookie.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		_ = h.auth.Logout(r.Context(), cookie.Value)
	}

	// Clear the session cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// setFlash sets a flash message cookie for the next request.
func setFlash(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    message,
		Path:     "/",
		HttpOnly: true,
	})
}
