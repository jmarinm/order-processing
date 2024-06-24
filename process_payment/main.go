package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type TableBasics struct {
	DynamoDbClient *dynamodb.Client
	TableName      string
}

type ProcessPaymentRequest struct {
	OrderId string `json:"order_id"`
	Status  string `json:"status"`
}

type Payment struct {
	OrderId    string `dynamodbav:"orderId"`
	Status     string `dynamodbav:"status"`
	TotalPrice int64  `dynamodbav:"totalPrice"`
}

var queueUrl string
var basics TableBasics
var sqsClient *sqs.Client

func (payment Payment) String() string {
	paymentJson, _ := json.Marshal(payment)
	return string(paymentJson)
}

func (basics TableBasics) GetOrder(id string) (Payment, error) {

	payment := Payment{}

	orderId, err := attributevalue.Marshal(id)
	if err != nil {
		log.Fatalf("Got error marshalling key: %v", err)
		return payment, err
	}

	response, err := basics.DynamoDbClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		Key:       map[string]types.AttributeValue{"orderId": orderId},
		TableName: aws.String(basics.TableName),
	})

	if err != nil {
		log.Fatalf("Got error calling GetItem: %v", err)
		return payment, err
	}

	err = attributevalue.UnmarshalMap(response.Item, &payment)
	if err != nil {
		log.Fatalf("Got error unmarshalling item: %v", err)
		return payment, err
	}

	return payment, nil

}

func (basics TableBasics) ProcessPayment(paymentRequest ProcessPaymentRequest) (Payment, error) {

	var err error
	var response *dynamodb.UpdateItemOutput
	payment := Payment{}
	update := expression.Set(expression.Name("status"), expression.Value(paymentRequest.Status))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()

	if err != nil {
		log.Fatalf("Got error building expression: %v", err)
		return payment, err
	}

	orderId, err := attributevalue.Marshal(paymentRequest.OrderId)

	if err != nil {
		log.Fatalf("Got error marshalling key: %v", err)
		return payment, err
	}

	response, err = basics.DynamoDbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(basics.TableName),
		Key:                       map[string]types.AttributeValue{"orderId": orderId},
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		ReturnValues:              types.ReturnValueAllNew,
	})

	if err != nil {
		log.Fatalf("Got error calling UpdateItem: %v", err)
		return payment, err
	}

	err = attributevalue.UnmarshalMap(response.Attributes, &payment)
	if err != nil {
		log.Fatalf("Got error unmarshalling attributes: %v", err)
		return payment, err
	}

	return payment, nil

}

func sendToQueue(payment Payment) error {
	paymentEvent := struct {
		OrderId string
	}{OrderId: payment.OrderId}

	paymentEventJson, err := json.Marshal(paymentEvent)
	if err != nil {
		log.Fatalf("Got error marshalling payment event: %v",
			err)
		return err
	}

	_, err = sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		MessageBody: aws.String(string(paymentEventJson)),
		QueueUrl:    aws.String(queueUrl),
	})

	if err != nil {
		log.Fatalf("Error sending message to SQS: %v", err)
	}
	return err

}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	err := initAwsSession()
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error initializing AWS session",
			StatusCode: 500,
		}, err
	}

	processPaymentRequest := ProcessPaymentRequest{}
	err = json.Unmarshal([]byte(request.Body), &processPaymentRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error unmarshalling request",
			StatusCode: 500,
		}, err
	}

	// Validate the request body
	if processPaymentRequest.OrderId == "" || processPaymentRequest.Status == "" {
		return events.APIGatewayProxyResponse{
			Body:       "Missing required fields",
			StatusCode: 400,
		}, nil
	}

	payment, err := basics.GetOrder(processPaymentRequest.OrderId)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error getting order",
			StatusCode: 500,
		}, err
	}

	if payment.OrderId == "" {
		return events.APIGatewayProxyResponse{
			Body:       "Order not found",
			StatusCode: 404,
		}, nil
	}

	if payment.Status != "incomplete" {
		return events.APIGatewayProxyResponse{
			Body:       "Order already processed",
			StatusCode: 400,
		}, nil
	}

	updatedPayment, err := basics.ProcessPayment(processPaymentRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error processing payment",
			StatusCode: 500,
		}, err
	}

	err = sendToQueue(updatedPayment)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error sending payment to SQS",
			StatusCode: 500,
		}, err
	}

	responseBody, err := json.Marshal(updatedPayment)
	if err != nil {
		log.Fatalf("Got error marshalling response body: %v", err)
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
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
		TableName:      "payments",
	}

	sqsClient = sqs.NewFromConfig(cfg)

	var queueName string = "payments-queue"
	params := &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	}

	resp, err := sqsClient.GetQueueUrl(context.TODO(), params)
	if err != nil {
		log.Fatalf("Error getting queue URL: %v", err)
		return err
	}

	queueUrl = *resp.QueueUrl
	return err

}
func main() {
	lambda.Start(handler)
}
