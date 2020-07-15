using System.Collections.Generic;
using System.IO;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Options;
using Stripe;
using Stripe.Checkout;

using System;

// For more information on enabling MVC for empty projects, visit https://go.microsoft.com/fwlink/?LinkID=397860

namespace dotnet.Controllers
{
    public class PaymentsController : Controller
    {
        private readonly IOptions<StripeOptions> options;

        public PaymentsController(IOptions<StripeOptions> options)
        {
            this.options = options;
            StripeConfiguration.ApiKey = options.Value.SecretKey;
        }

        [HttpGet("config")]
        public ActionResult<ConfigResponse> GetConfig()
        {
            var service = new PriceService();
            var price = service.Get(this.options.Value.Price);

            return new ConfigResponse
            {
                PublishableKey = this.options.Value.PublishableKey,
                UnitAmount = price.UnitAmount,
                Currency = price.Currency,
            };
        }

        [HttpGet("checkout-session")]
        public ActionResult<Session> GetCheckoutSession([FromQuery(Name = "sessionId")] string sessionId)
        {
            var service = new SessionService();
            var session = service.Get(sessionId);

            return session;
        }

        [HttpPost("create-checkout-session")]
        public ActionResult<CreateCheckoutSessionResponse> CreateCheckoutSession([FromBody] CreateCheckoutSessionRequest req)
        {

            var options = new SessionCreateOptions
            {
                PaymentMethodTypes = new List<string>
                {
                    "bacs_debit",
                },
                LineItems = new List<SessionLineItemOptions>
                {
                    new SessionLineItemOptions
                    {
                        Price = this.options.Value.Price,
                        Quantity = 1,
                    },
                },
                Mode = "payment",
                PaymentIntentData = new SessionPaymentIntentDataOptions
                {
                    SetupFutureUsage = "off_session",
                },

                SuccessUrl = this.options.Value.Domain + "/success.html?session_id={CHECKOUT_SESSION_ID}",
                CancelUrl = this.options.Value.Domain + "/canceled.html",
            };

            var service = new SessionService();
            Session session = service.Create(options);

            return new CreateCheckoutSessionResponse
            {
                SessionId = session.Id,
            };
        }

        [HttpPost("webhook")]
        public async Task<IActionResult> Webhook()
        {
            var json = await new StreamReader(HttpContext.Request.Body).ReadToEndAsync();
            Event stripeEvent;
            try
            {
                stripeEvent = EventUtility.ConstructEvent(
                    json,
                    Request.Headers["Stripe-Signature"],
                    this.options.Value.WebhookSecret
                );
                Console.WriteLine($"Webhook notification with type: {stripeEvent.Type} found for {stripeEvent.Id}");
            }
            catch (Exception e)
            {
                Console.WriteLine($"Something failed {e}");
                return BadRequest();
            }

            switch (stripeEvent.Type)
            {
                case Events.CheckoutSessionCompleted:

                    System.Diagnostics.Debug.WriteLine("Checkout session completed");

                    break;
                case Events.CheckoutSessionAsyncPaymentSucceeded:

                    System.Diagnostics.Debug.WriteLine("Checkout session async payment succeeded");

                    break;
                case Events.CheckoutSessionAsyncPaymentFailed:

                    System.Diagnostics.Debug.WriteLine("Checkout session async payment failed");

                    break;
            }

            return Ok();
        }
    }
}
