package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fsnotify/fsnotify"
)

func HasLocalFileChanged() {

}

func HasRemoteFileChanged() {

}

func SyncBetweenLocalAndCloud() {

}

func CreateS3Client() *s3.Client {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)
	return client
}

func DownloadFileFromCloud(client *s3.Client, filename string) bool {
	client.GetObject(context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String("k-drive123"),
			Key:    aws.String(filename),
		})
	return false
}

func UploadFileToCloud(client *s3.Client, filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("failed to open file %q, %v", filename, err)
		return false
	}
	log.Printf("uploading %q to cloud", filename)
	client.PutObject(context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String("k-drive123"),
			Key:    aws.String(filename),
			Body:   f,
		})
	return true
}

func GetCloudFolderStatus(client *s3.Client) {
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String("k-drive123"),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("first page results:")
	for _, object := range output.Contents {
		log.Printf("key=%s size=%d", aws.ToString(object.Key), object.Size)
	}
}
func main() {
	log.Println("Starting k-drive")
	client := CreateS3Client()

	// DownloadFileFromCloud(client, filename)
	// UploadFileToCloud(client, filename)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					UploadFileToCloud(client, event.Name)
					log.Println("modified file:", event.Name)
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					UploadFileToCloud(client, event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}
	<-done

}
