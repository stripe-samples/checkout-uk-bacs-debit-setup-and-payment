using Newtonsoft.Json;
using Stripe;

public class CreateCheckoutSessionResponse
{
    [JsonProperty("sessionId")]
    public string SessionId { get; set; }
}