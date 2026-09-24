package miniapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/docs"
)

func TestDocsServedWithoutSignature(t *testing.T) {
	handler := DocsHandler()

	for _, path := range []string{"/index.html", "/doc.json"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))

		if response.Code != http.StatusOK {
			t.Errorf("GET %s = %d, ожидалось 200", path, response.Code)
		}
	}
}

func TestEveryRouteIsDocumented(t *testing.T) {
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal([]byte(docs.SwaggerInfo.ReadDoc()), &spec); err != nil {
		t.Fatalf("описание API не разобрано: %v", err)
	}

	documented := make(map[string]bool)
	for path, operations := range spec.Paths {
		for method := range operations {
			documented[strings.ToUpper(method)+" "+path] = true
		}
	}

	routes := (&endpoints{}).routes()
	for route := range routes {
		if !documented[route] {
			t.Errorf("%s не описан: добавьте аннотации и выполните make swagger", route)
		}
	}
	for route := range documented {
		if _, ok := routes[route]; !ok {
			t.Errorf("описание обещает %s, которого в API нет", route)
		}
	}
}

func TestDocsNotServedByAPI(t *testing.T) {
	handler := NewHandler("токен", nil, nil, nil, nil, nil, nil)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/docs/index.html", nil))

	if response.Code == http.StatusOK {
		t.Error("описание API отдаётся с публичного адреса, а должно только с локального")
	}
}
