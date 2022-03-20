package sync

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	log "github.com/planetsp/k-drive/pkg/logging"
	s "github.com/planetsp/k-drive/pkg/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fsnotify/fsnotify"
)

func StartSyncClient(syncInfoChannel chan *s.SyncInfo, workingDirectory string) {
	client := CreateS3Client()

	done := make(chan bool)
	go MonitorLocalFolderForChanges(client, workingDirectory, syncInfoChannel)

	go MonitorCloudForChanges(client, workingDirectory, syncInfoChannel)

	<-done
}

func CreateS3Client() *s3.Client {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Error(err)
	}

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)
	return client
}
func DownloadFileFromCloud(client *s3.Client, filename string, workingDirectory string) *s.SyncInfo {
	result, err := client.GetObject(context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String("k-drive123"),
			Key:    aws.String(filename),
		})
	if err != nil {
		log.Error(err)
	}

	defer result.Body.Close()
	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		log.Error(err)
	}

	err = ioutil.WriteFile(workingDirectory+filename, body, 0644)
	if err != nil {
		log.Error(err)
	}
	log.Info("downloading %q from cloud", filename)

	return &s.SyncInfo{
		Filename:     filename,
		DateModified: *result.LastModified,
		Location:     s.Cloud,
		SyncStatus:   s.Downloading,
	}
}

func UploadFileToCloud(client *s3.Client, filename string) *s.SyncInfo {
	f, err := os.Open(filename)
	if err != nil {
		log.Error("failed to open file %q, %v", filename, err)
		return nil
	}
	// get last modified time
	file, err := os.Stat(filename)

	if err != nil {
		log.Error(err)
	}
	log.Info("uploading %q to cloud", filename)
	client.PutObject(context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String("k-drive123"),
			Key:    aws.String(filename),
			Body:   f,
		})
	return &s.SyncInfo{
		Filename:     filename,
		DateModified: file.ModTime(),
		Location:     s.Local,
		SyncStatus:   s.Uploading,
	}
}
func GetSyncDiff(client *s3.Client, workingDirectory string) *s.SyncDiff {
	diff := s.SyncDiff{
		FilesNotAvailableInCloud: ListItemsInCloudNotAvailableLocally(client, workingDirectory),
		FilesNotAvailableLocally: ListItemsInLocalDirNotAvailableInCloud(client, workingDirectory),
	}
	return &diff
}
func ListItemsInCloud(client *s3.Client) map[string]bool {
	filenameSet := make(map[string]bool)
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String("k-drive123"),
	})
	if err != nil {
		log.Error(err)
	}
	for _, object := range output.Contents {
		filenameSet[*object.Key] = true
	}
	return filenameSet
}
func ListItemsInLocalDir(workingDirectory string) map[string]bool {
	filenameSet := make(map[string]bool)
	files, err := ioutil.ReadDir(workingDirectory)
	if err != nil {
		log.Error(err)
	}

	for _, file := range files {
		if !file.IsDir() {
			filenameSet[file.Name()] = true

		}
	}
	return filenameSet
}
func ListItemsInCloudNotAvailableLocally(client *s3.Client, workingDirectory string) []s.SyncInfo {
	cloudFilenames := ListItemsInCloud(client)
	localFilenames := ListItemsInLocalDir(workingDirectory)
	return ListDiffBetweenSets(cloudFilenames, localFilenames)
}

func ListItemsInLocalDirNotAvailableInCloud(client *s3.Client, workingDirectory string) []s.SyncInfo {
	cloudFilenames := ListItemsInCloud(client)
	localFilenames := ListItemsInLocalDir(workingDirectory)
	return ListDiffBetweenSets(localFilenames, cloudFilenames)
}

func ListDiffBetweenSets(mapA map[string]bool, sliceB map[string]bool) []s.SyncInfo {
	diff := []s.SyncInfo{}
	for k := range mapA {
		exists := sliceB[k]
		if !exists {
			diff = append(diff, s.SyncInfo{Filename: k, DateModified: time.Now()})
		}
	}
	return diff
}
func MonitorCloudForChanges(client *s3.Client, workingDirectory string, syncInfoChannel chan *s.SyncInfo) {

	uptimeTicker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-uptimeTicker.C:
			filesToBeSyncedToLocalDir := ListItemsInCloudNotAvailableLocally(client, workingDirectory)
			for _, syncInfo := range filesToBeSyncedToLocalDir {
				syncInfoChannel <- DownloadFileFromCloud(client, syncInfo.Filename, workingDirectory)
			}
		}
	}
}

func MonitorLocalFolderForChanges(client *s3.Client, workingDirectory string, syncInfoChannel chan *s.SyncInfo) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
	}
	err = watcher.Add(workingDirectory)
	if err != nil {
		log.Error(err)
	}
	defer watcher.Close()
	for {
		notCurrentlyAvailableInCloud := ListItemsInCloud(client)
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Info("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if notCurrentlyAvailableInCloud[event.Name] {
					syncInfoChannel <- UploadFileToCloud(client, event.Name)
					log.Info("modified file:", event.Name)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error("error:", err)
		}
	}

}
