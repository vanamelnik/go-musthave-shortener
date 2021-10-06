package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/context"
)

// CookieMdlw проверяет в http request наличие cookie с полями uuid и token и добавляет в контекст запроса поле "uuid".
// Если отсутствует поле uuid, пользователю присваивается уникальный идентификатор, которым помечаются все записи
// в хранилище, сделанные данным пользователем. Токен представляет собой uuid, симметрично хэшированный секретным ключом по алгоритму SHA256.
// При неверном токене создается новая кука.
func CookieMdlw(secret string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var id uuid.UUID
			h := hmac.New(sha256.New, []byte(secret))

			id, ok := analyseCookies(r, h)
			if !ok {
				var err error // определяем, чтобы избежать локального переопределения id
				id, err = newSession(w, h)
				if err != nil {
					http.Error(w, "Something went wrong: cannot generate uuid", http.StatusInternalServerError)

					return
				}
			}

			ctx := context.WithID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// analyseCookies проверяет наличие кук uuid и token, а также валидность их значений.
// значение ok == false указывает на необходимость создания новой сессии
func analyseCookies(r *http.Request, h hash.Hash) (uuid.UUID, bool) {
	// проверяем наличие куки uuid
	cookie, err := r.Cookie("uuid")
	if err != nil {
		return uuid.Nil, false
	}

	// проверяем валидность значения uuid
	id, err := uuid.Parse(cookie.Value)
	if err != nil {
		return uuid.Nil, false
	}

	// проверяем наличие куки token
	cookie, err = r.Cookie("token")
	if err != nil {
		return uuid.Nil, false
	}

	// раскодируем токен
	token, err := hex.DecodeString(cookie.Value)
	if err != nil {
		return uuid.Nil, false
	}

	// проверяем совпадение подписи id с переданным токеном
	h.Write([]byte(id.String()))
	if !hmac.Equal(h.Sum(nil), token) {
		h.Reset()
		return uuid.Nil, false
	}

	log.Printf("CookieMdlw: successfully authenticated: id=%s", id)

	return id, true
}

// newSession создает новые uuid и токен пользователя, сохраняет их в cookie.
// Возвращает ошибку в маловероятном случае сбоя генерации нового uuid
func newSession(w http.ResponseWriter, h hash.Hash) (uuid.UUID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Printf("CookieMdlw: cannot generate an uuid: %v", err)
		return uuid.Nil, err
	}

	h.Write([]byte(id.String()))
	token := hex.EncodeToString(h.Sum(nil))
	log.Printf("CookieMdlw: successfully created new session id=%s", id)

	http.SetCookie(w, &http.Cookie{Name: "uuid", Path: "/", Value: id.String()})
	http.SetCookie(w, &http.Cookie{Name: "token", Path: "/", Value: token})

	return id, nil
}
