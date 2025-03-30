package ethereum

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		svc := New()
		if svc.rpcUrl != DefaultURL {
			t.Errorf("expected rpcUrl %q, got %q", DefaultURL, svc.rpcUrl)
		}

		if svc.httpClient == nil {
			t.Fatal("expected non-nil httpClient")
		}

		expectedTimeout := 5 * time.Second
		if svc.httpClient.Timeout != expectedTimeout {
			t.Errorf("expected timeout %v, got %v", expectedTimeout, svc.httpClient.Timeout)
		}
	})

	t.Run("custom URL", func(t *testing.T) {
		customURL := "http://custom-ethereum-node.com"

		svc := New(WithURL(customURL))
		if svc.rpcUrl != customURL {
			t.Errorf("expected rpcUrl %q, got %q", customURL, svc.rpcUrl)
		}
	})

	t.Run("custom Timeout", func(t *testing.T) {
		customTimeout := 10 * time.Second

		svc := New(WithTimeout(customTimeout))
		if svc.httpClient.Timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, svc.httpClient.Timeout)
		}
	})

	t.Run("custom URL and Timeout", func(t *testing.T) {
		var (
			customURL     = "http://another-custom-node.com"
			customTimeout = 15 * time.Second
		)

		svc := New(WithURL(customURL), WithTimeout(customTimeout))
		if svc.rpcUrl != customURL {
			t.Errorf("expected rpcUrl %q, got %q", customURL, svc.rpcUrl)
		}

		if svc.httpClient.Timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, svc.httpClient.Timeout)
		}
	})
}
