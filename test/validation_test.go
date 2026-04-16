package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/travisbale/mailman/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// rawGRPCClient returns a raw protobuf client that bypasses SDK validation.
func rawGRPCClient(t *testing.T) pb.MailmanServiceClient {
	t.Helper()

	conn, err := grpc.NewClient(
		grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return pb.NewMailmanServiceClient(conn)
}

func TestSendEmailMissingTemplateID(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	_, err := client.SendEmail(context.Background(), &pb.SendEmailRequest{
		TemplateId: "",
		To:         "user@example.com",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "template_id")
}

func TestSendEmailMissingRecipient(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	_, err := client.SendEmail(context.Background(), &pb.SendEmailRequest{
		TemplateId: "simple_template",
		To:         "",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "to")
}

func TestSendEmailBatchPartialFailure(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	_, err := client.SendEmailBatch(context.Background(), &pb.SendEmailBatchRequest{
		Emails: []*pb.SendEmailRequest{
			{
				TemplateId: "simple_template",
				To:         "valid@example.com",
				Variables:  map[string]string{"Name": "Alice"},
			},
			{
				TemplateId: "simple_template",
				To:         "", // Missing recipient
			},
		},
	})

	// Batch fails atomically - the invalid email causes the entire batch to fail
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSendEmailBatchEmpty(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	resp, err := client.SendEmailBatch(context.Background(), &pb.SendEmailBatchRequest{
		Emails: []*pb.SendEmailRequest{},
	})

	// An empty batch currently succeeds with an empty results list.
	// If this behavior changes to return an error, update this test.
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}
