package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddCaseTesting(t *testing.T) {
	cart := NewCart()
	if len(cart.Cases) != 0 {
		t.Fatal("expected empty cart")
	}

	blueLight := FixtureBeer("Labatt", "Blue Light", 12.0)
	cart.AddCase(FixtureCase(6, blueLight, 10.99))
	if len(cart.Cases) != 1 {
		t.Fatal("expected 1 case in cart")
	}
}

func TestAddCaseAssert(t *testing.T) {
	cart := NewCart()
	assert.Equal(t, 0, len(cart.Cases))

	blueLight := FixtureBeer("Labatt", "Blue Light", 12.0)
	cart.AddCase(FixtureCase(6, blueLight, 10.99))
	assert.Equal(t, 1, len(cart.Cases))
}

func TestSubtotal(t *testing.T) {
	cart := NewCart()
	assert.Equal(t, 0, len(cart.Cases))

	duvelHop := FixtureBeer("Duvel", "Tripel Hop", 11.0)
	cart.AddCase(FixtureCase(4, duvelHop, 14.99))
	blueLight := FixtureBeer("Labatt", "Blue Light", 12.0)
	cart.AddCase(FixtureCase(30, blueLight, 24.99))
	assert.Equal(t, 39.98, cart.Subtotal())
}

func TestSubtotalSuite(t *testing.T) {
	testCases := []struct {
		name     string
		cart     *Cart
		subtotal float64
	}{
		{
			name:     "Empty cart",
			cart:     &Cart{},
			subtotal: 0,
		},
		{
			name: "Party time",
			cart: &Cart{Cases: []*Case{
				FixtureCase(4, FixtureBeer("Duvel", "Tripel Hop", 11.0), 14.99),
				FixtureCase(30, FixtureBeer("Labatt", "Blue Light", 12.0), 24.99),
				FixtureCase(30, FixtureBeer("Labatt", "Blue Light", 12.0), 24.99),
			}},
			subtotal: 64.97,
		},
		{
			name: "Negative",
			cart: &Cart{Cases: []*Case{
				FixtureCase(4, FixtureBeer("Duvel", "Tripel Hop", 11.0), -14),
				FixtureCase(30, FixtureBeer("Labatt", "Blue Light", 12.0), 24),
			}},
			subtotal: 10.00,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.subtotal, tc.cart.Subtotal())
		})
	}
}

func TestProcessPayment(t *testing.T) {
	testCases := []struct {
		name          string
		handler       http.HandlerFunc
		expectedError error
		expectedBody  []byte
	}{
		{
			name: "OK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`OK`))
				w.WriteHeader(http.StatusOK)
			},
			expectedError: nil,
			expectedBody:  []byte(`OK`),
		},
		{
			name: "Internal server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: fmt.Errorf("payment server error: %d", http.StatusInternalServerError),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tc.handler))
			defer ts.Close()
			body, err := ProcessPayment(ts.URL, 21.11)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedBody, body)
		})
	}
}

func TestStartSubscriptionTimer(t *testing.T) {
	ctx := context.Background()
	cart1 := &Cart{Cases: []*Case{FixtureCase(4, FixtureBeer("Duvel", "Tripel Hop", 11.0), 14)}}
	cart2 := &Cart{Cases: []*Case{FixtureCase(30, FixtureBeer("Labatt", "Blue Light", 12.0), 24)}}
	subscription := &Subscription{
		cart:        cart1,
		interval:    time.Duration(1) * time.Second,
		messageChan: make(chan interface{}),
	}

	go subscription.startSubscriptionTimer(ctx)
	msg := <-subscription.messageChan
	order, ok := msg.(*Cart)
	if !ok {
		t.Fatal("received invalid message on message channel")
	}
	assert.Equal(t, cart1, order)

	subscription.SetCart(cart2)
	msg = <-subscription.messageChan
	order, ok = msg.(*Cart)
	if !ok {
		t.Fatal("received invalid message on message channel")
	}
	assert.Equal(t, cart2, order)
}

func TestStartOrderHandler(t *testing.T) {
	handler := &OrderHandler{
		messageChan: make(chan interface{}),
	}
	go handler.startOrderHandler(context.Background())
	assert.Equal(t, 0, len(handler.ProcessedOrders))

	handler.messageChan <- FixtureCart()
	handler.messageChan <- FixtureCart()
	handler.messageChan <- FixtureCase(30, FixtureBeer("Labatt", "Blue Light", 12.0), 24)
	assert.Equal(t, 2, len(handler.ProcessedOrders))
}
