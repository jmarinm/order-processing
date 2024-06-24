package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type TableBasics struct {
	DynamoDbClient *dynamodb.Client
	TableName      string
}

type Payment struct {
	OrderId    string `json:"order_id" dynamodbav:"orderId"`
	TotalPrice int64  `json:"total_price" dynamodbav:"totalPrice"`
	Status     string `json:"status" dynamodbav:"status"`
}

type OrderCreatedEvent struct {
	OrderId    string `json:"order_id"`
	TotalPrice int64  `json:"total_price"`
}

var basics TableBasics

func (basics TableBasics) AddPayment(orderCreatedEvent OrderCreatedEvent) error {

	payment := Payment{
		OrderId:    orderCreatedEvent.OrderId,
		TotalPrice: orderCreatedEvent.TotalPrice,
		Status:     "incomplete",
	}

	av, err := attributevalue.MarshalMap(payment)
	if err != nil {
		fmt.Println("Got error marshalling map: %v", err)
		return err
	}

	_, err = basics.DynamoDbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(basics.TableName),
	})
	if err != nil {
		fmt.Println("Got error calling PutItem: %v\n", err)
		return err
	}

	return nil
}
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, message := range sqsEvent.Records {
		orderCreatedEvent := OrderCreatedEvent{}
		err := json.Unmarshal([]byte(message.Body), &orderCreatedEvent)
		if err != nil {
			fmt.Println("Error unmarshalling order created event")
			return err
		}

		err = basics.AddPayment(orderCreatedEvent)
		if err != nil {
			fmt.Println("Error adding payment")
			return err
		}
	}

	return nil
}

func initSession() error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println("Error loading configuration")
		return err
	}

	basics = TableBasics{
		DynamoDbClient: dynamodb.NewFromConfig(cfg),
		TableName:      "payments",
	}

	return nil
}

func main() {
	err := initSession()
	if err != nil {
		fmt.Println("Error initializing session")
		return
	}
	lambda.Start(handler)
}
