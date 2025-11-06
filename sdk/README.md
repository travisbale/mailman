# Mailman SDK

Go SDK for interacting with the Mailman email service.

## Installation

```bash
go get github.com/travisbale/mailman/sdk
```

## Usage

### Creating a Client

```go
import "github.com/travisbale/mailman/sdk"

// Create a gRPC client
client, err := sdk.NewGRPCClient("localhost:50051")
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Sending a Single Email

```go
req := sdk.SendEmailRequest{
    TemplateID: "welcome_email",
    To:         "user@example.com",
    Variables: map[string]string{
        "UserName": "Alice",
        "AppName":  "MyApp",
    },
}

resp, err := client.SendEmail(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
```

### Sending a Batch of Emails

```go
req := sdk.SendEmailBatchRequest{
    Emails: []sdk.SendEmailRequest{
        {
            TemplateID: "welcome_email",
            To:         "user1@example.com",
            Variables:  map[string]string{"UserName": "Alice"},
        },
        {
            TemplateID: "welcome_email",
            To:         "user2@example.com",
            Variables:  map[string]string{"UserName": "Bob"},
        },
    },
}

resp, err := client.SendEmailBatch(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
```

### Scheduling an Email

```go
scheduledTime := time.Now().Add(1 * time.Hour)

req := sdk.SendEmailRequest{
    TemplateID:  "reminder_email",
    To:          "user@example.com",
    Variables:   map[string]string{"EventName": "Meeting"},
    ScheduledAt: &scheduledTime,
}

resp, err := client.SendEmail(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
```

### Setting Priority

```go
req := sdk.SendEmailRequest{
    TemplateID: "urgent_alert",
    To:         "admin@example.com",
    Variables:  map[string]string{"Message": "Critical issue"},
    Priority:   10, // Higher priority
}

resp, err := client.SendEmail(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
```

### Listing Available Templates

```go
resp, err := client.ListTemplates(context.Background())
if err != nil {
    log.Fatal(err)
}

for _, template := range resp.Templates {
    fmt.Printf("Template: %s (version %d)\n", template.ID, template.Version)
    fmt.Printf("  Subject: %s\n", template.Subject)
    fmt.Printf("  Required variables: %v\n", template.RequiredVariables)
}
```

### Getting Email Job Status

```go
req := sdk.GetEmailStatusRequest{
    JobID: "job-12345",
}

resp, err := client.GetEmailStatus(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", resp.Status)
fmt.Printf("Attempts: %d\n", resp.Attempts)
if resp.LastError != "" {
    fmt.Printf("Last error: %s\n", resp.LastError)
}
```

## Advanced Configuration

### Using Custom Dial Options

```go
import "google.golang.org/grpc/credentials"

// Use TLS credentials
creds, err := credentials.NewClientTLSFromFile("server.crt", "")
if err != nil {
    log.Fatal(err)
}

client, err := sdk.NewGRPCClient(
    "mailman.example.com:50051",
    sdk.WithDialOptions(grpc.WithTransportCredentials(creds)),
)
```

### Setting Custom Timeout

```go
client, err := sdk.NewGRPCClient(
    "localhost:50051",
    sdk.WithTimeout(60 * time.Second),
)
```

## Error Handling

The SDK returns typed errors from the gRPC layer. You can use `status.Code()` to check specific error codes:

```go
import "google.golang.org/grpc/status"
import "google.golang.org/grpc/codes"

resp, err := client.SendEmail(ctx, req)
if err != nil {
    if st, ok := status.FromError(err); ok {
        switch st.Code() {
        case codes.NotFound:
            fmt.Println("Template not found")
        case codes.InvalidArgument:
            fmt.Println("Invalid request:", st.Message())
        default:
            fmt.Println("Error:", err)
        }
    }
}
```

## Request Validation

All request types have a `Validate()` method that checks required fields:

```go
req := sdk.SendEmailRequest{
    // Missing required fields
}

if err := req.Validate(); err != nil {
    fmt.Println("Validation error:", err)
    // Output: template_id is required
}
```

The SDK automatically validates requests before sending to the server.
