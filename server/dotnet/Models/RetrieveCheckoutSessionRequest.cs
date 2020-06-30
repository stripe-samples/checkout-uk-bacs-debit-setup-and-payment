using Newtonsoft.Json;

public class RetrieveCheckoutSessionRequest
{
    [JsonProperty("sessionId")]
    public string Session { get; set; }
}