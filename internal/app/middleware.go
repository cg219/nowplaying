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
                    if err := encode(w, 409, ResponseError{ Success: false, Messaage: "Username Taken", Code: CODE_USER_EXISTS }); err != nil {
                        return500(w)
                    }
                    return

                case AUTH_ERROR:
                    if err := encode(w, 404, ResponseError{ Success: false, Messaage: "Username/Password Incorrect", Code: AUTH_FAIL }); err != nil {
                        return500(w)
                    }
                    return

                case REDIRECT_ERROR:
                    s.log.Info("Redirect Error")
                    return

                case GOTO_NEXT_HANDLER_ERROR:
                    s.log.Info("Moving to next handler")
                    continue

                case INTERNAL_ERROR:
                    return500(w)
                    return
                }

                s.log.Error("Uncaught Error", "error", err)
            }
        }
    })
}


func (s *Server) RedirectAuthenticated(redirect string, onAuth bool) CandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) error {
        cookie, err := r.Cookie("nowplaying")
        if err != nil {
            s.log.Error("Cookie Retrieval", "cookie", "nowplaying", "method", "RedirectAuthenticated", "request", r, "error", err.Error())
            if !onAuth {
                http.Redirect(w, r, redirect, http.StatusSeeOther)
                return fmt.Errorf(REDIRECT_ERROR)
            } else {
                return nil
            }
        }

        value, err := base64.StdEncoding.DecodeString(cookie.Value)
        if err != nil {
            s.log.Error("Base64 Decoding", "cookie", cookie.Value, "method", "RedirectAuthenticated", "request", r, "error", err.Error())
            if !onAuth {
                http.Redirect(w, r, redirect, http.StatusSeeOther)
                return fmt.Errorf(REDIRECT_ERROR)
            } else {
                return nil
            }
        }

        var cookieValue webtoken.CookieAuthValue
        err = json.Unmarshal(value, &cookieValue)
        if err != nil {
            s.log.Error("Invalid Cookie Value", "cookie", cookie.Value, "method", "RedirectAuthenticated", "request", r, "error", err.Error())
            if !onAuth {
                http.Redirect(w, r, redirect, http.StatusSeeOther)
                return fmt.Errorf(REDIRECT_ERROR)
            } else {
                return nil
            }
        }

        ok, username, refresh := s.isAuthenticated(r.Context(), cookieValue.AccessToken, cookieValue.RefreshToken)
        if ok {
            s.authenticateRequest(r, username)

            if onAuth {
                http.Redirect(w, r, redirect, http.StatusSeeOther)
                return fmt.Errorf(GOTO_NEXT_HANDLER_ERROR)
            } else {
                return nil
            }
        }
        if !ok && refresh != nil {
            refresh(w)
            s.authenticateRequest(r, username)

            if onAuth {
                http.Redirect(w, r, redirect, http.StatusSeeOther)
                return fmt.Errorf(GOTO_NEXT_HANDLER_ERROR)
            } else {
                return nil
            }
        }

        if onAuth {
            return fmt.Errorf(GOTO_NEXT_HANDLER_ERROR)
        } else {
            http.Redirect(w, r, redirect, http.StatusSeeOther)
            return fmt.Errorf(REDIRECT_ERROR)
        }
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
