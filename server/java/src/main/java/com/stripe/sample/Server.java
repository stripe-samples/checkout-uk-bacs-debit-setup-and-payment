package com.stripe.sample;

import java.nio.file.Paths;

import java.util.HashMap;
import java.util.Map;

import static spark.Spark.get;
import static spark.Spark.post;
import static spark.Spark.port;
import static spark.Spark.staticFiles;

import com.google.gson.Gson;
import com.google.gson.annotations.SerializedName;

import com.stripe.Stripe;
import com.stripe.model.Event;
import com.stripe.model.checkout.Session;
import com.stripe.model.Price;
import com.stripe.exception.*;
import com.stripe.net.Webhook;
import com.stripe.param.checkout.SessionCreateParams;
import com.stripe.param.checkout.SessionCreateParams.LineItem;
import com.stripe.param.checkout.SessionCreateParams.PaymentMethodType;

import io.github.cdimascio.dotenv.Dotenv;

public class Server {
  private static Gson gson = new Gson();

  static class PostBody {
    @SerializedName("quantity")
    Long quantity;

    public Long getQuantity() {
      return quantity;
    }
  }

  public static void main(String[] args) {
    port(4242);

    Dotenv dotenv = Dotenv.load();

    Stripe.apiKey = dotenv.get("STRIPE_SECRET_KEY");

    staticFiles.externalLocation(
        Paths.get(Paths.get("").toAbsolutePath().toString(), dotenv.get("STATIC_DIR")).normalize().toString());

    get("/config", (request, response) -> {
      response.type("application/json");
      Price price = Price.retrieve(dotenv.get("PRICE"));

      Map<String, Object> responseData = new HashMap<>();
      responseData.put("publicKey", dotenv.get("STRIPE_PUBLISHABLE_KEY"));
      responseData.put("unitAmount", price.getUnitAmount());
      responseData.put("currency", price.getCurrency());

      return gson.toJson(responseData);
    });

    // Fetch the Checkout Session to display the JSON result on the success page
    get("/checkout-session", (request, response) -> {
      response.type("application/json");

      String sessionId = request.queryParams("sessionId");
      Session session = Session.retrieve(sessionId);

      return gson.toJson(session);
    });

    post("/create-checkout-session", (request, response) -> {
      response.type("application/json");
      PostBody postBody = gson.fromJson(request.body(), PostBody.class);

      String domainUrl = dotenv.get("DOMAIN");
      Long quantity = postBody.getQuantity();
      String price = dotenv.get("PRICE");

      // Create new Checkout Session for the order
      // Other optional params include:
      // [billing_address_collection] - to display billing address details on the page
      // [customer] - if you have an existing Stripe Customer ID
      // [customer_email] - lets you prefill the email input in the form
      // For full details see https://stripe.com/docs/api/checkout/sessions/create

      // ?session_id={CHECKOUT_SESSION_ID} means the redirect will have the session ID
      // set as a query param
      SessionCreateParams createParams =
        SessionCreateParams.builder()
        .addPaymentMethodType(SessionCreateParams.PaymentMethodType.BACS_DEBIT)
        .addLineItem(
            SessionCreateParams.LineItem.builder()
            .setPrice(price)
            .setQuantity(quantity)
            .build())
        .setMode(SessionCreateParams.Mode.PAYMENT)
        .setPaymentIntentData(
            SessionCreateParams.PaymentIntentData.builder()
            .setSetupFutureUsage(SessionCreateParams.PaymentIntentData.SetupFutureUsage.OFF_SESSION)
            .build())
        .setSuccessUrl(domainUrl + "/success.html?session_id={CHECKOUT_SESSION_ID}")
        .setCancelUrl(domainUrl + "/canceled.html")
        .build();

      Session session = Session.create(createParams);

      Map<String, Object> responseData = new HashMap<>();
      responseData.put("sessionId", session.getId());
      return gson.toJson(responseData);
    });

    post("/webhook", (request, response) -> {
      String payload = request.body();
      String sigHeader = request.headers("Stripe-Signature");
      String endpointSecret = dotenv.get("STRIPE_WEBHOOK_SECRET");

      Event event = null;

      try {
        event = Webhook.constructEvent(payload, sigHeader, endpointSecret);
      } catch (SignatureVerificationException e) {
        // Invalid signature
        response.status(400);
        return "";
      }

      switch (event.getType()) {
        case "checkout.session.completed":
          System.out.println("Checkout session succeeded");
          response.status(200);
          return "";
        case "checkout.session.async_payment_succeeded":
          System.out.println("Checkout session async payment succeeded");
          response.status(200);
          return "";
        case "checkout.session.async_payment_failed":
          System.out.println("Checkout session async payment failed");
          response.status(200);
          return "";
        default:
          response.status(200);
          return "";
      }
    });
  }
}
