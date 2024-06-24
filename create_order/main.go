package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/google/uuid"
)

type TableBasics struct {
	DynamoDbClient *dynamodb.Client
	TableName      string
}

type OrderRequest struct {
	UserId     string `json:"user_id"`
	Item       string `json:"item"`
	Quantity   int    `json:"quantity"`
	TotalPrice int64  `json:"total_price"`
}

type Order struct {
	OrderId       string `json:"order_id" dynamodbav:"orderId"`
	UserId        string `json:"user_id" dynamodbav:"userId"`
	Item          string `json:"item" dynamodbav:"item"`
	Quantity      int    `json:"quantity" dynamodbav:"quantity"`
	TotalPrice    int64  `json:"total_price" dynamodbav:"totalPrice"`
	ShippingReady bool   `json:"shipping_ready" dynamodbav:"shippingReady"`
}

type OrderCreatedEvent struct {
	OrderId    string `json:"order_id"`
	TotalPrice int64  `json:"total_price"`
}

func (order Order) String() string {
	orderJson, _ := json.Marshal(order)
	return string(orderJson)
}

var queueUrl string
var basics TableBasics
var sqsClient *sqs.Client

func (basics TableBasics) AddOrder(OrderRequest OrderRequest) (Order, error) {

	var order Order = Order{
		OrderId:       uuid.New().String(),
		UserId:        OrderRequest.UserId,
		Item:          OrderRequest.Item,
		Quantity:      OrderRequest.Quantity,
		TotalPrice:    OrderRequest.TotalPrice,
		ShippingReady: false,
	}

	item, err := attributevalue.MarshalMap(order)
	if err != nil {
		panic(err)
	}
	_, err = basics.DynamoDbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(basics.TableName), Item: item,
	})

	if err != nil {
		log.Printf("Couldn't add item to table. Here's why: %v\n", err)
	}
	return order, err
}

// Function to send event to SQS queue
func sendToQueue(order Order) error {
	orderEvent := OrderCreatedEvent{
		OrderId:    order.OrderId,
		TotalPrice: order.TotalPrice,
	}

	orderJson, _ := json.Marshal(orderEvent)
	_, err := sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		MessageBody: aws.String(string(orderJson)),
		QueueUrl:    aws.String(queueUrl),
	})

	if err != nil {
		log.Printf("Error sending message to SQS: %v\n", err)
	}

	return err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	err := initAwsSession()
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error initializing AWS session",
			StatusCode: 500,
		}, nil
	}

	//Unmarshal the request body into an OrderRequest struct
	order := OrderRequest{}
	err = json.Unmarshal([]byte(request.Body), &order)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error unmarshalling request",
			StatusCode: 500,
		}, nil
	}

	// Validate the request body
	if order.UserId == "" || order.Item == "" || order.Quantity == 0 || order.TotalPrice == 0 {
		return events.APIGatewayProxyResponse{
			Body:       "Missing required fields",
			StatusCode: 400,
		}, nil
	}

	// Add the order to the DynamoDB table
	orderCreated, err := basics.AddOrder(order)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error adding order to DynamoDB",
			StatusCode: 500,
		}, nil
	}

	// Send the order to the SQS queue
	err = sendToQueue(orderCreated)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error sending order to SQS",
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       orderCreated.String(),
		StatusCode: 200,
	}, nil
}

func initAwsSession() error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
		return err
	}

	basics = TableBasics{
		DynamoDbClient: dynamodb.NewFromConfig(cfg),
		TableName:      "orders",
	}

	sqsClient = sqs.NewFromConfig(cfg)

	var queueName string = "orders-queue"
	params := &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	}

	resp, err := sqsClient.GetQueueUrl(context.TODO(), params)
	if err != nil {
		log.Fatalf("Error getting queue URL: %v", err)
		return err
	}

	queueUrl = *resp.QueueUrl
	fmt.Printf("Queue URL: %s\n", queueUrl)

	return err
}

func main() {
	lambda.Start(handler)
}
