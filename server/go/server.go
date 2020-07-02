package main

import (
  "bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/price"
	"github.com/stripe/stripe-go/v71/checkout/session"
	"github.com/stripe/stripe-go/v71/webhook"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("godotenv.Load: %v", err)
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	http.Handle("/", http.FileServer(http.Dir(os.Getenv("STATIC_DIR"))))
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/create-checkout-session", handleCreateCheckoutSession)
	http.HandleFunc("/checkout-session", handleRetrieveCheckoutSession)
	http.HandleFunc("/webhook", handleWebhook)

	addr := "localhost:4242"
	log.Printf("Listening on %s ...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	price, err := price.Get(os.Getenv("PRICE"), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("price.Get: %v", err)
		return
	}

	writeJSON(w, struct {
		PublicKey string `json:"publicKey"`
		UnitAmount int64 `json:"unitAmount"`
		Currency stripe.Currency `json:"currency"`
	}{
		PublicKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		UnitAmount: price.UnitAmount,
		Currency: price.Currency,
	})
}

func handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Quantity *int64 `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

  params := &stripe.CheckoutSessionParams{
    SuccessURL: stripe.String(os.Getenv("DOMAIN") + "/success.html?session_id={CHECKOUT_SESSION_ID}"),
    CancelURL: stripe.String(os.Getenv("DOMAIN") + "/canceled.html"),

    PaymentMethodTypes: stripe.StringSlice([]string{
      "bacs_debit",
    }),
    PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
      SetupFutureUsage: stripe.String(string(stripe.PaymentIntentSetupFutureUsageOffSession)),
		},
    LineItems: []*stripe.CheckoutSessionLineItemParams{
      &stripe.CheckoutSessionLineItemParams{
        Price: stripe.String(os.Getenv("PRICE")),
        Quantity: req.Quantity,
      },
    },
    Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
  }
  s, _ := session.New(params)

	writeJSON(w, struct {
		Session string `json:"sessionId"`
	}{
		Session: s.ID,
	})
}

func handleRetrieveCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
    return
  }
  keys, ok := r.URL.Query()["sessionId"]

  if !ok || len(keys[0]) < 1 {
    log.Println("Url Param 'sessionId' is missing")
    return
  }

  sessionId := keys[0]
	
	session, err := session.Get(sessionId, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("session.Get: %v", err)
		return
	}

  writeJSON(w, session)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
  var buf bytes.Buffer
  if err := json.NewEncoder(&buf).Encode(v); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    log.Printf("json.NewEncoder.Encode: %v", err)
    return
  }
  w.Header().Set("Content-Type", "application/json")
  if _, err := io.Copy(w, &buf); err != nil {
    log.Printf("io.Copy: %v", err)
    return
  }
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("ioutil.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	if event.Type == "checkout.session.completed" {
		log.Printf("Checkout session completed")
			return
	}

	if event.Type == "checkout.session.async_payment_succeeded" {
		log.Printf("Checkout session async payment succeeded", err)
			return
	}

	if event.Type == "checkout.session.async_payment_failed" {
		log.Printf("Checkout session async payment failed", err)
			return
	}
}
