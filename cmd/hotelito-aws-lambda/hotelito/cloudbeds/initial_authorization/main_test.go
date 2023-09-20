package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandleInit(t *testing.T) {
	ctx := context.TODO()
	req := events.APIGatewayProxyRequest{
		// Fill in fields
	}
	expectedResponse := events.APIGatewayProxyResponse{
		// Fill in expected fields
	}

	response, err := handleRequest(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}
