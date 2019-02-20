package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophish/gophish/config"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/stretchr/testify/suite"
)

var successHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
})

type MiddlewareSuite struct {
	suite.Suite
}

func (s *MiddlewareSuite) SetupSuite() {
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	err := models.Setup(conf)
	if err != nil {
		s.T().Fatalf("Failed creating database: %v", err)
	}
}

// MiddlewarePermissionTest maps an expected HTTP Method to an expected HTTP
// status code
type MiddlewarePermissionTest map[string]int

// TestEnforceViewOnly ensures that only users with the ModifyObjects
// permission have the ability to send non-GET requests.
func (s *MiddlewareSuite) TestEnforceViewOnly() {
	permissionTests := map[string]MiddlewarePermissionTest{
		models.RoleAdmin: MiddlewarePermissionTest{
			http.MethodGet:     http.StatusOK,
			http.MethodHead:    http.StatusOK,
			http.MethodOptions: http.StatusOK,
			http.MethodPost:    http.StatusOK,
			http.MethodPut:     http.StatusOK,
			http.MethodDelete:  http.StatusOK,
		},
		models.RoleUser: MiddlewarePermissionTest{
			http.MethodGet:     http.StatusOK,
			http.MethodHead:    http.StatusOK,
			http.MethodOptions: http.StatusOK,
			http.MethodPost:    http.StatusOK,
			http.MethodPut:     http.StatusOK,
			http.MethodDelete:  http.StatusOK,
		},
	}
	for r, checks := range permissionTests {
		role, err := models.GetRoleBySlug(r)
		s.Nil(err)

		for method, expected := range checks {
			req := httptest.NewRequest(method, "/", nil)
			response := httptest.NewRecorder()

			req = ctx.Set(req, "user", models.User{
				Role:   role,
				RoleID: role.ID,
			})

			EnforceViewOnly(successHandler).ServeHTTP(response, req)
			s.Equal(response.Code, expected)
		}
	}
}

func (s *MiddlewareSuite) TestRequirePermission() {
	middleware := RequirePermission(models.PermissionModifySystem)
	handler := middleware(successHandler)

	permissionTests := map[string]int{
		models.RoleUser:  http.StatusForbidden,
		models.RoleAdmin: http.StatusOK,
	}

	for role, expected := range permissionTests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		// Test that with the requested permission, the request succeeds
		role, err := models.GetRoleBySlug(role)
		s.Nil(err)
		req = ctx.Set(req, "user", models.User{
			Role:   role,
			RoleID: role.ID,
		})
		handler.ServeHTTP(response, req)
		s.Equal(response.Code, expected)
	}
}

func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}
