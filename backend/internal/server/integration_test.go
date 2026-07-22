package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/olmeware/backend/database"
	"github.com/olmeware/backend/internal/config"
	"github.com/olmeware/backend/internal/server"
)

// TestStorefrontFlow exercises the public + authenticated happy path against the
// real database. It self-cleans and skips when no reachable DB is configured
// (e.g. in a sandbox without DATABASE_URL or network to Postgres).
func TestStorefrontFlow(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("database unreachable, skipping: %v", err)
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	ts := httptest.NewServer(server.New(cfg, pool).Handler())
	defer ts.Close()

	email := fmt.Sprintf("itest+%d@example.com", time.Now().UnixNano())
	var userID, orderID string
	defer func() {
		if orderID != "" {
			_, _ = pool.Exec(ctx, `update inventory i set reserved = reserved - oi.quantity
				from order_items oi where oi.order_id = $1 and oi.variant_id = i.variant_id`, orderID)
			_, _ = pool.Exec(ctx, `delete from inventory_movements where reference_type='order' and reference_id=$1`, orderID)
			_, _ = pool.Exec(ctx, `delete from orders where id=$1`, orderID)
		}
		_, _ = pool.Exec(ctx, `delete from users where email=$1`, email)
	}()

	client := ts.Client()
	call := func(method, path, token string, body any) (int, map[string]any) {
		var buf io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			buf = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, ts.URL+path, buf)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	// Health.
	if code, _ := call("GET", "/api/v1/health", "", nil); code != http.StatusOK {
		t.Fatalf("health status = %d", code)
	}

	// Register.
	code, reg := call("POST", "/api/v1/auth/register", "", map[string]string{
		"name": "Integration Tester", "email": email, "password": "supersecret",
	})
	if code != http.StatusCreated {
		t.Fatalf("register status = %d (%v)", code, reg)
	}
	token, _ := reg["accessToken"].(string)
	if user, ok := reg["user"].(map[string]any); ok {
		userID, _ = user["id"].(string)
	}
	if token == "" || userID == "" {
		t.Fatalf("register did not return token/user: %v", reg)
	}

	// Catalog: list + pick a product/variant.
	code, list := call("GET", "/api/v1/catalog/products?limit=1", "", nil)
	if code != http.StatusOK {
		t.Fatalf("catalog status = %d", code)
	}
	products, _ := list["products"].([]any)
	if len(products) == 0 {
		t.Fatal("catalog returned no products; is the DB seeded?")
	}
	slug := products[0].(map[string]any)["slug"].(string)

	_, detail := call("GET", "/api/v1/catalog/products/"+slug, "", nil)
	variants, _ := detail["variants"].([]any)
	if len(variants) == 0 {
		t.Fatalf("product %s has no variants", slug)
	}
	variantID := variants[0].(map[string]any)["id"].(string)

	// Cart: add item.
	code, cart := call("POST", "/api/v1/cart/items", token, map[string]any{
		"variantId": variantID, "quantity": 1,
	})
	if code != http.StatusOK {
		t.Fatalf("add to cart status = %d (%v)", code, cart)
	}
	if int(cart["itemCount"].(float64)) != 1 {
		t.Fatalf("expected itemCount 1, got %v", cart["itemCount"])
	}

	// Checkout.
	code, order := call("POST", "/api/v1/orders", token, map[string]any{
		"email": email, "name": "Integration Tester",
		"shippingAddress": map[string]string{
			"recipientName": "Integration Tester", "line1": "1 Test St",
			"city": "CDMX", "state": "CDMX", "postalCode": "01000",
		},
	})
	if code != http.StatusCreated {
		t.Fatalf("checkout status = %d (%v)", code, order)
	}
	orderID, _ = order["id"].(string)
	if order["status"] != "pending_payment" {
		t.Fatalf("order status = %v", order["status"])
	}

	// Orders list should now contain exactly this order.
	code, orders := call("GET", "/api/v1/orders", token, nil)
	if code != http.StatusOK {
		t.Fatalf("orders list status = %d", code)
	}
	if got := len(orders["orders"].([]any)); got != 1 {
		t.Fatalf("expected 1 order, got %d", got)
	}

	// Active cart should be empty (converted).
	_, cart2 := call("GET", "/api/v1/cart", token, nil)
	if int(cart2["itemCount"].(float64)) != 0 {
		t.Fatalf("cart should be empty after checkout, got %v", cart2["itemCount"])
	}
}
