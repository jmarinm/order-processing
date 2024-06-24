require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go v1.54.5
	github.com/aws/aws-sdk-go-v2 v1.30.0
	github.com/aws/aws-sdk-go-v2/config v1.27.21
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.14.4
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.7.26
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.33.1
	github.com/aws/aws-sdk-go-v2/service/sqs v1.33.1
	github.com/go-chi/render v1.0.3
	github.com/google/uuid v1.6.0
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8

module hello-world

go 1.16
