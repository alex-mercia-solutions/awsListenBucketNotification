package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/tidwall/gjson"
)

func awslisten() {
	queue := flag.String("q", "examplequeue", "The name of the queue")
	// can change wait time to determine how long to poll for new messages
	waitTime := flag.Int64("w", 10, "How long the queue waits for messages")
	flag.Parse()

	if *queue == "" {
		fmt.Println("You must supply a queue name (-q QUEUE")
		return
	}
	if *waitTime < 0 {
		*waitTime = 0
	}
	if *waitTime > 20 {
		*waitTime = 20
	}
	// Create a session that gets credential values from ~/.aws/credentials
	// and the default region from ~/.aws/config
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	result, err := GetQueueURL(sess, queue)
	if err != nil {
		fmt.Println("Got an error getting the queue URL:")
		fmt.Println(err)
		return
	}
	queueURL := result.QueueUrl
	//infite loop to check the queue for new messages
	for {
		msgs, err := GetLPMessages(sess, queueURL, waitTime)
		if err != nil {
			fmt.Println("Got an error receiving messages:")
			fmt.Println(err)
			return
		}
		fmt.Println("Message IDs:")
		for _, msg := range msgs {
			fmt.Println("    " + *msg.MessageId)
			// check if the body of the SQS message has put event in and run app
			if strings.Contains(*msg.Body, "ObjectCreated:Put") {
				if strings.Contains(*msg.Body, ".iso") {
					fmt.Println("Event detected: PUT")
					result := gjson.Get(*msg.Body, "Records.#.s3.object.key")
					fmt.Println(result.String())
					// your code goes here
					DeleteMessage(sess, *queueURL, msg.ReceiptHandle)
				}
			}
			if strings.Contains(*msg.Body, "ObjectCreated:CompleteMultipartUpload") {
				if strings.Contains(*msg.Body, ".iso") {
					fmt.Println("Event detected: PUT")
					result := gjson.Get(*msg.Body, "Records.#.s3.object.key")
					fmt.Println(result.String())
					// your code goes here
					DeleteMessage(sess, *queueURL, msg.ReceiptHandle)
				}
			}
		}
	}
}

func GetQueueURL(sess *session.Session, queue *string) (*sqs.GetQueueUrlOutput, error) {
	svc := sqs.New(sess)
	urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queue,
	})
	if err != nil {
		return nil, err
	}
	return urlResult, nil
}

func DeleteMessage(sess *session.Session, queueUrl string, messageHandle *string) error {
	sqsClient := sqs.New(sess)

	_, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: messageHandle,
	})
	fmt.Println("message deleted")
	return err
}

func GetLPMessages(sess *session.Session, queueURL *string, waitTime *int64) ([]*sqs.Message, error) {
	var msgs []*sqs.Message
	svc := sqs.New(sess)
	result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: queueURL,
		AttributeNames: aws.StringSlice([]string{
			"SentTimestamp",
		}),
		MaxNumberOfMessages: aws.Int64(1),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
		WaitTimeSeconds: waitTime,
	})
	if err != nil {
		return msgs, err
	}
	if len(result.Messages) == 0 {
		fmt.Println("Queue Empty")
	}
	return result.Messages, nil
}
