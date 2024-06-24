package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
)

type TableBasics struct {
	DynamoDbClient *dynamodb.Client
	TableName      string
}

type PaymentProcessedEvent struct {
	OrderId string
}

var basics TableBasics

func (basics TableBasics) UpdateOrder(paymentProcessedEvent PaymentProcessedEvent) error {
	var err error
	update := expression.Set(expression.Name("shippingReady"), expression.Value(true))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()

	if err != nil {
		log.Fatalf("Got error building expression: %s", err)
		return err
	}

	orderId, err := attributevalue.Marshal(paymentProcessedEvent.OrderId)

	if err != nil {
		log.Fatalf("Got error marshalling key: %v", err)
		return err
	}

	_, err = basics.DynamoDbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(basics.TableName),
		Key:                       map[string]types.AttributeValue{"orderId": orderId},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		log.Fatalf("Got error calling UpdateItem: %v", err)
		return err
	}

	return nil
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	err := initAwsSession()
	if err != nil {
		return err
	}

	for _, message := range sqsEvent.Records {
		paymentProcessedEvent := PaymentProcessedEvent{}
		err := json.Unmarshal([]byte(message.Body), &paymentProcessedEvent)
		if err != nil {
			return err
		}

		err = basics.UpdateOrder(paymentProcessedEvent)
		if err != nil {
			log.Fatalf("Error updating order: %v", err)
			return err
		}

	}
	return nil
}

func initAwsSession() error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		return err
	}
	basics = TableBasics{
		DynamoDbClient: dynamodb.NewFromConfig(cfg),
		TableName:      "orders",
	}

	return nil
}
func main() {
	lambda.Start(handler)
}
