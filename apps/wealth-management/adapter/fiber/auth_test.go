package fiber

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestWriteGuard(t *testing.T) {
	cases := []struct {
		name   string
		token  string
		host   string
		header string
		want   int
	}{
		{"loopback without token allows", "", "127.0.0.1", "", http.StatusOK},
		{"ipv6 loopback without token allows", "", "::1", "", http.StatusOK},
		{"tailnet without token refuses", "", "0.0.0.0", "", http.StatusForbidden},
		{"missing header", "secret", "0.0.0.0", "", http.StatusUnauthorized},
		{"wrong token", "secret", "0.0.0.0", "Bearer nope", http.StatusUnauthorized},
		{"right token", "secret", "0.0.0.0", "Bearer secret", http.StatusOK},
		{"token required even on loopback", "secret", "127.0.0.1", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/w", NewWriteGuard(tc.token, tc.host).Wrap(func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) }))
			request := httptest.NewRequest(http.MethodPost, "/w", nil)
			if tc.header != "" {
				request.Header.Set("Authorization", tc.header)
			}
			response, err := app.Test(request, -1)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, tc.want)
			}
		})
	}
}
