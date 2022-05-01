package middleware

import (
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

// SubnetCheckerMdlw проверяет IP-адрес клиента и пропускает запрос только в случае, если
// он принадлежит доверенной подсети.
func SubnetCheckerMdlw(trustedSubnet string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trustedSubnet == "" {
				log.Println("subnetCheckerMdlw: trusted subnet is not defined")
				http.Error(w, "Forbidden", http.StatusForbidden)

				return

			}
			ipStr := r.Header.Get("X-Real-IP")
			if ipStr == "" {
				log.Println("subnetCheckerMdlw: no 'X-Real-IP' key in the header")
				http.Error(w, "Bad request", http.StatusBadRequest)

				return
			}
			ip := net.ParseIP(ipStr)
			_, subnet, err := net.ParseCIDR(trustedSubnet)
			if err != nil {
				log.Println("subnetCheckerMdlw: unreachable error: could not parse trusted_subnet: ", err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)

				return
			}
			if !subnet.Contains(ip) {
				log.Printf("subnetCheckerMdlw: ip address %s is not in the trusted subnet %s", ipStr, trustedSubnet)
				http.Error(w, "Forbidden", http.StatusForbidden)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
