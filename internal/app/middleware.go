package app

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    _ "net/http/pprof"
    "github.com/cg219/nowplaying/pkg/webtoken"
)

func (s *Server) handle(h ...CandlerFunc) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        for _, currentHandler := range h {
            if err := currentHandler(w, r); err != nil {
                switch err.Error() {
                case USERNAME_EXISTS_ERROR:
                    if err := encode[ResponseError](w, 409, ResponseError{ Success: false, Messaage: "Username Taken", Code: CODE_USER_EXISTS }); err != nil {
                        return500(w)
                    }
                    return

                case AUTH_ERROR:
                    if err := encode[ResponseError](w, 404, ResponseError{ Success: false, Messaage: "Username/Password Incorrect", Code: AUTH_FAIL }); err != nil {
                        return500(w)
                    }
                    return

                case REDIRECTED_ERROR:
                    return

                case INTERNAL_ERROR:
                    return500(w)
                    return
                }

                s.log.Error("Uncaught Error", err)
            }
        }
    })
}


func (s *Server) RedirectAuthenticated(redirect string, onAuth bool) CandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) error {
        cookie, err := r.Cookie("nowplaying")
        if err != nil {
            s.log.Error("Cookie Retrieval", "cookie", "nowplaying", "method", "UserOnly", "request", r, "error", err.Error())
            return nil
        }

        value, err := base64.StdEncoding.DecodeString(cookie.Value)
        if err != nil {
            s.log.Error("Base64 Decoding", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
            return nil
        }

        var cookieValue webtoken.CookieAuthValue
        err = json.Unmarshal(value, &cookieValue)
        if err != nil {
            s.log.Error("Invalid Cookie Value", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
            return nil
        }

        ok, username, refresh := s.isAuthenticated(r.Context(), cookieValue.AccessToken, cookieValue.RefreshToken)
        if ok {
            s.authenticateRequest(r, username)

            if onAuth {
                http.Redirect(w, r, redirect, http.StatusSeeOther)
                return fmt.Errorf(REDIRECTED_ERROR)
            }
        }
        if !ok && refresh != nil {
            refresh(w)
            s.authenticateRequest(r, username)

            if onAuth {
                http.Redirect(w, r, redirect, http.StatusSeeOther)
                return fmt.Errorf(REDIRECTED_ERROR)
            }
        }

        if !onAuth {
            http.Redirect(w, r, redirect, http.StatusSeeOther)
            return fmt.Errorf(REDIRECTED_ERROR)
        }
        return nil
    }
}

func (s *Server) UserOnly(w http.ResponseWriter, r *http.Request) error {
    cookie, err := r.Cookie("nowplaying")
    if err != nil {
        s.log.Error("Cookie Retrieval", "cookie", "nowplaying", "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    value, err := base64.StdEncoding.DecodeString(cookie.Value)
    if err != nil {
        s.log.Error("Base64 Decoding", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    var cookieValue webtoken.CookieAuthValue
    err = json.Unmarshal(value, &cookieValue)
    if err != nil {
        s.log.Error("Invalid Cookie Value", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    ok, username, refresh := s.isAuthenticated(r.Context(), cookieValue.AccessToken, cookieValue.RefreshToken)
    if ok {
        s.authenticateRequest(r, username)
        return nil
    }

    if !ok && refresh != nil {
        refresh(w)
        s.authenticateRequest(r, username)
        return nil
    }

    return fmt.Errorf(AUTH_ERROR)
}
