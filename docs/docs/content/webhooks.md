# Webhooks

listmonk supports event webhooks that notify external services when specific events occur in the system. This enables real-time integrations with CRMs, analytics platforms, automation tools, and custom applications.

Webhooks are configured in **Settings -> Webhooks** in the admin UI.

## Configuration

Each webhook endpoint has the following settings:

| Setting | Description |
|:--------|:------------|
| **Name** | A unique identifier for the webhook (alphanumeric and dashes only) |
| **URL** | The HTTP(S) endpoint to POST events to |
| **Secret** | Optional HMAC-SHA256 signing secret for request verification |
| **Events** | List of event types to send to this endpoint |
| **Max Connections** | Maximum concurrent HTTP connections (1-100) |
| **Timeout** | HTTP request timeout (e.g., `5s`, `30s`, `1m`) |

## Events

The following event types are available:

### Subscriber Events

| Event | Triggered When |
|:------|:---------------|
| `subscriber.created` | A new subscriber is added |
| `subscriber.updated` | A subscriber's details are modified |
| `subscriber.confirmed` | A subscriber confirms their subscription (double opt-in) |
| `subscriber.unsubscribed` | A subscriber opts out |
| `subscriber.blocklisted` | A subscriber is added to the blocklist |
| `subscriber.deleted` | A subscriber is permanently deleted |

### Campaign Events

| Event | Triggered When |
|:------|:---------------|
| `campaign.started` | A campaign begins sending |
| `campaign.finished` | A campaign completes sending |

### Tracking Events

| Event | Triggered When |
|:------|:---------------|
| `tracking.email_open` | A subscriber opens an email |
| `tracking.link_click` | A subscriber clicks a link in an email |
| `tracking.bounce` | An email bounces (hard or soft) |

## Payload Format

Webhook requests are sent as HTTP POST with `Content-Type: application/json`. The payload structure is:

```json
{
  "event": "subscriber.created",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    // Event-specific data
  }
}
```

### Subscriber Event Payload

```json
{
  "event": "subscriber.created",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    "subscriber": {
      "id": 123,
      "uuid": "e44b4135-1e1d-40c5-8a30-0f9a886c2884",
      "email": "user@example.com",
      "name": "John Doe",
      "status": "enabled",
      "attribs": {
        "city": "New York",
        "plan": "premium"
      },
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    },
    "list_ids": [1, 2, 3]
  }
}
```

### Campaign Event Payload

```json
{
  "event": "campaign.started",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    "campaign": {
      "id": 45,
      "uuid": "2e7e4b51-f31b-418a-a120-e41800cb689f",
      "name": "January Newsletter",
      "subject": "Your January Update",
      "status": "running",
      "type": "regular",
      "tags": ["newsletter", "monthly"],
      "started_at": "2025-01-15T10:30:00Z"
    }
  }
}
```

### Tracking Event Payloads

**Email Open:**
```json
{
  "event": "tracking.email_open",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    "subscriber_uuid": "e44b4135-1e1d-40c5-8a30-0f9a886c2884",
    "campaign_uuid": "2e7e4b51-f31b-418a-a120-e41800cb689f",
    "user_agent": "Mozilla/5.0...",
    "ip_address": "203.0.113.50"
  }
}
```

**Link Click:**
```json
{
  "event": "tracking.link_click",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    "subscriber_uuid": "e44b4135-1e1d-40c5-8a30-0f9a886c2884",
    "campaign_uuid": "2e7e4b51-f31b-418a-a120-e41800cb689f",
    "link_url": "https://example.com/offer",
    "user_agent": "Mozilla/5.0...",
    "ip_address": "203.0.113.50"
  }
}
```

**Bounce:**
```json
{
  "event": "tracking.bounce",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    "subscriber_uuid": "e44b4135-1e1d-40c5-8a30-0f9a886c2884",
    "campaign_uuid": "2e7e4b51-f31b-418a-a120-e41800cb689f",
    "bounce_type": "hard",
    "bounce_message": "550 User not found"
  }
}
```

## Request Signature

When a **Secret** is configured, listmonk signs each request using HMAC-SHA256. The signature is included in the `X-Listmonk-Signature` header.

To verify the signature:

1. Compute HMAC-SHA256 of the raw request body using your secret key
2. Encode the result as hexadecimal
3. Compare with the `X-Listmonk-Signature` header value

### Verification Examples

**Go:**
```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func verifySignature(body []byte, secret, signature string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

**Python:**
```python
import hmac
import hashlib

def verify_signature(body: bytes, secret: str, signature: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        body,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)
```

**Node.js:**
```javascript
const crypto = require('crypto');

function verifySignature(body, secret, signature) {
    const expected = crypto
        .createHmac('sha256', secret)
        .update(body)
        .digest('hex');
    return crypto.timingSafeEqual(
        Buffer.from(expected),
        Buffer.from(signature)
    );
}
```

**PHP:**
```php
function verifySignature(string $body, string $secret, string $signature): bool {
    $expected = hash_hmac('sha256', $body, $secret);
    return hash_equals($expected, $signature);
}
```

## Response Handling

Your webhook endpoint should return an HTTP `2xx` status code to indicate successful receipt.

- **Success (2xx)**: The webhook delivery is marked as successful
- **Failure (4xx, 5xx, timeout)**: The delivery is marked as failed and logged

listmonk does not retry failed webhook deliveries. If reliability is critical, consider using a webhook proxy service that handles retries and queuing.

## Dashboard Metrics

Webhook delivery statistics are displayed on the Dashboard, showing:

- Total dispatched events
- Successful deliveries
- Failed deliveries
- Success rate per endpoint
- Last error message (if any)

## Best Practices

1. **Always verify signatures** in production to ensure requests are from listmonk
2. **Respond quickly** (within 5 seconds) to avoid timeouts. For long processing, queue the event and return immediately
3. **Use HTTPS** endpoints to encrypt webhook payloads in transit
4. **Handle duplicates** gracefully - the same event may occasionally be delivered more than once
5. **Monitor the dashboard** for failed deliveries to catch integration issues early

## Example: Simple Webhook Receiver

Here's a minimal webhook receiver in Node.js:

```javascript
const express = require('express');
const crypto = require('crypto');

const app = express();
const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET;

app.post('/webhook', express.raw({ type: 'application/json' }), (req, res) => {
    // Verify signature
    const signature = req.headers['x-listmonk-signature'];
    if (WEBHOOK_SECRET && signature) {
        const expected = crypto
            .createHmac('sha256', WEBHOOK_SECRET)
            .update(req.body)
            .digest('hex');
        if (!crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature))) {
            return res.status(401).send('Invalid signature');
        }
    }

    // Parse and handle the event
    const event = JSON.parse(req.body);
    console.log(`Received: ${event.event}`, event.data);

    // Process the event (e.g., sync to CRM, trigger automation)
    // ...

    res.status(200).send('OK');
});

app.listen(8888, () => console.log('Webhook receiver listening on port 8888'));
```

## Use Cases

- **CRM Integration**: Sync new subscribers to Salesforce, HubSpot, or Pipedrive
- **Analytics**: Track campaign performance in external analytics platforms
- **Automation**: Trigger workflows in Zapier, n8n, or custom automation systems
- **Notifications**: Send Slack/Discord alerts when campaigns complete
- **Data Warehousing**: Stream events to BigQuery, Snowflake, or data lakes
- **Compliance**: Log subscription changes for GDPR audit trails
