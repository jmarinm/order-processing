
Welcome to this repository, where you can find an example of a AWS SAM (Simple Application Model) app. Which represents the creation of orders and processing of payments. With the purpose of using asynchronous communication and other technologies.

Technologies used:
- AWS SAM
- AWS Lambda
- AWS SQS
- AWS DynamoDB
- Golang

To test this application you must:

- Have AWS CLI installed.
- Have AWS SAM CLI Installed.
- Potentially have Go installed.

Start up:

- Clone the repo
- Go into the root folder.
- run: ```sam build```
- if doesn't work you must install go dependencies, go into each go module and run: ```go get -u```
- try running ```sam build``` again.
- When succesful run ```sam deploy```
- Wait until pops the changeset and type ```y```
- Wait until the process finishes.

Use:

- Final output of the startup should output both urls for both endpoints of this application.

POST /create-order

Expected body example: 

```
{
    "user_id": "45",
    "item":"Pocillo",
    "quantity": 52,
    "total_price": 4000
}
```

PUT /process-payment

Expected body example:

```
{
    "order_id":"3d3f8f47-edfa-4729-a0e1-cd7f10519020", //You probably want to use the id returned by the previous endpoint.
    "status":"complete"
}
```




