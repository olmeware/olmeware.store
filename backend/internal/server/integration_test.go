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
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/olmeware/backend/database"
	"github.com/olmeware/backend/internal/config"
	orderrepo "github.com/olmeware/backend/internal/orders"
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
	var userID string
	var orderIDs []string
	defer func() {
		for _, orderID := range orderIDs {
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
	orderID, _ := order["id"].(string)
	orderIDs = append(orderIDs, orderID)
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

	// Create a second cart and race two repository checkouts against it. Both
	// callers must resolve to the same order and reserve inventory only once.
	code, _ = call("POST", "/api/v1/cart/items", token, map[string]any{
		"variantId": variantID, "quantity": 1,
	})
	if code != http.StatusOK {
		t.Fatalf("add to second cart status = %d", code)
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	var cartID uuid.UUID
	if err := pool.QueryRow(ctx,
		`select id from carts where user_id=$1 and status='active'`, uid).Scan(&cartID); err != nil {
		t.Fatalf("load second cart: %v", err)
	}

	repo := orderrepo.NewRepo(pool)
	request := orderrepo.CreateOrderRequest{
		Email: email,
		Name:  "Integration Tester",
		ShippingAddress: orderrepo.Address{
			RecipientName: "Integration Tester", Line1: "1 Test St", City: "CDMX",
			State: "CDMX", PostalCode: "01000", CountryCode: "MX",
		},
	}
	type checkoutResult struct {
		order *orderrepo.Order
		err   error
	}
	results := make([]checkoutResult, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i].order, results[i].err = repo.CreateFromCart(ctx, cartID, &uid, request)
		}(i)
	}
	close(start)
	wg.Wait()
	for i, result := range results {
		if result.err != nil {
			t.Fatalf("concurrent checkout %d: %v", i, result.err)
		}
	}
	if results[0].order.ID != results[1].order.ID {
		t.Fatalf("concurrent checkouts created different orders: %s != %s",
			results[0].order.ID, results[1].order.ID)
	}
	concurrentOrderID := results[0].order.ID
	orderIDs = append(orderIDs, concurrentOrderID.String())

	var orderCount, reservationCount int
	if err := pool.QueryRow(ctx, `select count(*) from orders where cart_id=$1`, cartID).Scan(&orderCount); err != nil {
		t.Fatalf("count concurrent orders: %v", err)
	}
	if err := pool.QueryRow(ctx, `select count(*) from inventory_movements
		where reference_type='order' and reference_id=$1 and movement_type='reservation'`,
		concurrentOrderID).Scan(&reservationCount); err != nil {
		t.Fatalf("count concurrent reservations: %v", err)
	}
	if orderCount != 1 || reservationCount != 1 {
		t.Fatalf("concurrent checkout produced %d orders and %d reservations; want 1 and 1",
			orderCount, reservationCount)
	}
}
