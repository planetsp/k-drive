package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fsnotify/fsnotify"
)

type SyncDiff struct {
	FilesNotAvailableInCloud []string
	FilesNotAvailableLocally []string
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
	result, err := client.GetObject(context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String("k-drive123"),
			Key:    aws.String(filename),
		})
	if err != nil {
		log.Println(err)
	}

	defer result.Body.Close()
	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = ioutil.WriteFile(filename, body, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("downloading %q from cloud", filename)

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

func ListItemsInLocalDir(workingDirectory string) map[string]bool {
	filenameSet := make(map[string]bool)
	files, err := ioutil.ReadDir(workingDirectory)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("getting items available locally")

	for _, file := range files {
		if !file.IsDir() {
			filenameSet[file.Name()] = true
		}
	}
	return filenameSet
}

func ListItemsInCloud(client *s3.Client) map[string]bool {
	filenameSet := make(map[string]bool)
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String("k-drive123"),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("getting items available in cloud")
	for _, object := range output.Contents {
		filenameSet[*object.Key] = true
	}
	return filenameSet
}
func GetSyncDiff(client *s3.Client, workingDirectory string) *SyncDiff {
	diff := SyncDiff{
		FilesNotAvailableInCloud: ListItemsInCloudNotAvailableLocally(client, workingDirectory),
		FilesNotAvailableLocally: ListItemsInCloudNotAvailableLocally(client, workingDirectory),
	}
	return &diff
}
func ListItemsInCloudNotAvailableLocally(client *s3.Client, workingDirectory string) []string {
	cloudFilenames := ListItemsInCloud(client)
	localFilenames := ListItemsInLocalDir(workingDirectory)
	return ListDiffBetweenSets(cloudFilenames, localFilenames)
}

func ListItemsInLocalDirNotAvailableInCloud(client *s3.Client, workingDirectory string) []string {
	cloudFilenames := ListItemsInCloud(client)
	localFilenames := ListItemsInLocalDir(workingDirectory)
	return ListDiffBetweenSets(localFilenames, cloudFilenames)
}

func ListDiffBetweenSets(mapA map[string]bool, sliceB map[string]bool) []string {
	diff := []string{}
	for k := range mapA {
		exists := sliceB[k]
		if !exists {
			diff = append(diff, k)
		}
	}
	return diff
}

func MonitorCloudForChanges(client *s3.Client, workingDirectory string) {
	uptimeTicker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-uptimeTicker.C:
			log.Println("trying to go cloud")
			filesToBeSyncedToLocalDir := ListItemsInCloudNotAvailableLocally(client, workingDirectory)
			for _, filename := range filesToBeSyncedToLocalDir {
				DownloadFileFromCloud(client, filename)
			}
		}
	}
}

func MonitorLocalFolderForChanges(client *s3.Client, workingDirectory string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	err = watcher.Add(workingDirectory)
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	for {
		log.Println("trying to go local")

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

}

func main() {
	log.Println("Starting k-drive")
	client := CreateS3Client()

	workingDirectory := "."

	done := make(chan bool)
	go MonitorLocalFolderForChanges(client, workingDirectory)

	go MonitorCloudForChanges(client, workingDirectory)

	<-done

}
