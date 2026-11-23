package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет, что IP клиента входит в доверенную подсеть
func TrustedSubnetMiddleware(trustedSubnet string) (func(http.Handler) http.Handler, error) {
	var subnet *net.IPNet
	if trustedSubnet != "" {
		var err error
		_, subnet, err = net.ParseCIDR(trustedSubnet)
		if err != nil {
			return nil, fmt.Errorf("некорректный формат CIDR доверенной подсети %q: %w", trustedSubnet, err)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если подсеть не задана, пропускаем все запросы
			if subnet == nil {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := r.Header.Get("X-Real-IP")
			if clientIP == "" {
				log.Printf("Запрос без заголовка X-Real-IP от %s", r.RemoteAddr)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			// Парсим IP клиента
			ip := net.ParseIP(clientIP)
			if ip == nil {
				log.Printf("Невалидный IP в заголовке X-Real-IP: %s", clientIP)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Проверяем, входит ли IP в подсеть
			if !subnet.Contains(ip) {
				log.Printf("IP %s не входит в доверенную подсеть %s", clientIP, trustedSubnet)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// IP в доверенной подсети, пропускаем запрос
			next.ServeHTTP(w, r)
		})
	}, nil
}
