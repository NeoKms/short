package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedirectUsesLocationHeaderForRegularURL(t *testing.T) {
	target := "https://example.com/page#section"
	recorder := httptest.NewRecorder()

	redirect(recorder, httptest.NewRequest(http.MethodGet, "/Ab12Cd34", nil), target)

	response := recorder.Result()
	if response.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusFound)
	}
	if location := response.Header.Get("Location"); location != target {
		t.Errorf("Location = %q, want %q", location, target)
	}
}

func TestRedirectUsesHTMLForLongURL(t *testing.T) {
	target := "https://example.com/report#import=" + strings.Repeat("a", maxLocationHeaderURLLength)
	recorder := httptest.NewRecorder()

	redirect(recorder, httptest.NewRequest(http.MethodGet, "/Ab12Cd34", nil), target)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if location := response.Header.Get("Location"); location != "" {
		t.Errorf("Location = %q, want empty", location)
	}
	if body := recorder.Body.String(); !strings.Contains(body, target) {
		t.Error("response body does not contain redirect target")
	}
}

func TestRedirectEscapesLongURLInScript(t *testing.T) {
	target := "https://example.com/report#import=</script><script>alert(1)</script>" + strings.Repeat("a", maxLocationHeaderURLLength)
	recorder := httptest.NewRecorder()

	redirect(recorder, httptest.NewRequest(http.MethodGet, "/Ab12Cd34", nil), target)

	body := recorder.Body.String()
	if strings.Contains(body, "</script><script>alert(1)</script>") {
		t.Error("response body contains unescaped script markup")
	}
	if !strings.Contains(body, `\u003c/script\u003e`) {
		t.Error("response body does not contain JSON-escaped redirect target")
	}
}
