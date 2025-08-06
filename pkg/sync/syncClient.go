package sync

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fsnotify/fsnotify"
	c "github.com/planetsp/k-drive/pkg/config"
	log "github.com/planetsp/k-drive/pkg/logging"
	s "github.com/planetsp/k-drive/pkg/models"
)

func StartSyncClient(syncInfoChannel chan *s.SyncInfo) {
	// Check if configuration is loaded
	if !c.IsConfigLoaded() {
		log.Error("Configuration not loaded, cannot start sync client")
		return
	}
	
	config := c.GetConfig()
	if config.WorkingDirectory == "" || config.BucketName == "" {
		log.Error("Invalid configuration: missing required fields")
		return
	}
	
	// Check if working directory exists
	if _, err := os.Stat(config.WorkingDirectory); os.IsNotExist(err) {
		log.Error("Working directory does not exist: %s", config.WorkingDirectory)
		log.Info("Please update your configuration with a valid directory path")
		return
	}
	
	client := CreateS3Client()
	if client == nil {
		log.Error("Failed to create S3 client, cannot start sync")
		return
	}
	
	log.Info("Starting sync client with working directory: %s", config.WorkingDirectory)
	log.Info("Syncing with S3 bucket: %s", config.BucketName)
	
	done := make(chan bool)
	go MonitorLocalFolderForChanges(client, syncInfoChannel)
	go MonitorCloudForChanges(client, syncInfoChannel)

	<-done
}

func CreateS3Client() *s3.Client {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Error("Failed to load AWS configuration: %v", err)
		log.Info("Ensure you have AWS credentials configured in ~/.aws/credentials or environment variables")
		return nil
	}

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)
	return client
}
func DownloadFileFromCloud(client *s3.Client, filename string, syncInfoChannel chan *s.SyncInfo) {
	// Send initial downloading status
	downloadingInfo := &s.SyncInfo{
		Filename:     filename,
		DateModified: time.Now(),
		Location:     s.Cloud,
		SyncStatus:   s.Downloading,
	}
	syncInfoChannel <- downloadingInfo

	result, err := client.GetObject(context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String(c.GetConfig().BucketName),
			Key:    aws.String(filename),
		})
	if err != nil {
		log.Error(err)
		return
	}

	defer result.Body.Close()
	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		log.Error(err)
		return
	}
	workingDirectory := c.GetConfig().WorkingDirectory
	err = ioutil.WriteFile(workingDirectory+filename, body, 0644)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("downloading %q from cloud", filename)

	// Send completion status
	syncedInfo := &s.SyncInfo{
		Filename:     filename,
		DateModified: *result.LastModified,
		Location:     s.Local,
		SyncStatus:   s.Synced,
	}
	syncInfoChannel <- syncedInfo
}

func UploadFileToCloud(client *s3.Client, filename string, syncInfoChannel chan *s.SyncInfo) {
	workingDirectory := c.GetConfig().WorkingDirectory

	f, err := os.Open(workingDirectory + filename)
	if err != nil {
		log.Error("failed to open file %q, %v", filename, err)
		return
	}
	defer f.Close()

	// get last modified time
	file, err := os.Stat(workingDirectory + filename)
	if err != nil {
		log.Error(err)
		return
	}

	// Send initial uploading status
	uploadingInfo := &s.SyncInfo{
		Filename:     filename,
		DateModified: file.ModTime(),
		Location:     s.Local,
		SyncStatus:   s.Uploading,
	}
	syncInfoChannel <- uploadingInfo

	log.Info("uploading %q to cloud", filename)
	_, err = client.PutObject(context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(c.GetConfig().BucketName),
			Key:    aws.String(filename),
			Body:   f,
		})

	if err != nil {
		log.Error("failed to upload file %q, %v", filename, err)
		return
	}

	// Send completion status
	syncedInfo := &s.SyncInfo{
		Filename:     filename,
		DateModified: file.ModTime(),
		Location:     s.Cloud,
		SyncStatus:   s.Synced,
	}
	syncInfoChannel <- syncedInfo
}
func GetSyncDiff(client *s3.Client, workingDirectory string) *s.SyncDiff {
	diff := s.SyncDiff{
		FilesNotAvailableInCloud: ListItemsInCloudNotAvailableLocally(client),
		FilesNotAvailableLocally: ListItemsInLocalDirNotAvailableInCloud(client),
	}
	return &diff
}
func ListItemsInCloud(client *s3.Client) map[string]bool {
	filenameSet := make(map[string]bool)
	log.Debug("Checking cloud")

	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(c.GetConfig().BucketName),
	})
	if err != nil {
		log.Error("Failed to list objects in cloud: %v", err)
		log.Info("Please ensure your AWS credentials and region are properly configured")
		return filenameSet // Return empty set on error
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
func ListItemsInCloudNotAvailableLocally(client *s3.Client) []s.SyncInfo {
	workingDirectory := c.GetConfig().WorkingDirectory

	cloudFilenames := ListItemsInCloud(client)
	localFilenames := ListItemsInLocalDir(workingDirectory)
	return ListDiffBetweenSets(cloudFilenames, localFilenames)
}

func ListItemsInLocalDirNotAvailableInCloud(client *s3.Client) []s.SyncInfo {
	workingDirectory := c.GetConfig().WorkingDirectory

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
func MonitorCloudForChanges(client *s3.Client, syncInfoChannel chan *s.SyncInfo) {
	// Test AWS connectivity first
	testInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.GetConfig().BucketName),
	}
	
	_, err := client.ListObjectsV2(context.TODO(), testInput)
	if err != nil {
		log.Error("Failed to connect to AWS S3: %v", err)
		log.Info("Cloud monitoring disabled. Please check your AWS configuration")
		return // Exit if we can't connect to AWS
	}

	log.Info("AWS S3 connectivity test successful")
	uptimeTicker := time.NewTicker(c.GetConfig().LocalDirectoryPollingFrequency * time.Second)
	defer uptimeTicker.Stop()
	
	for {
		select {
		case <-uptimeTicker.C:
			filesToBeSyncedToLocalDir := ListItemsInCloudNotAvailableLocally(client)
			for _, syncInfo := range filesToBeSyncedToLocalDir {
				go DownloadFileFromCloud(client, syncInfo.Filename, syncInfoChannel)
			}
		}
	}
}

func MonitorLocalFolderForChanges(client *s3.Client, syncInfoChannel chan *s.SyncInfo) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
	}
	workingDirectory := c.GetConfig().WorkingDirectory

	err = watcher.Add(workingDirectory)
	if err != nil {
		log.Error(err)
	}
	defer watcher.Close()
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Info("event:", event)
			filename := GetEventFilename(event.Name)
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Check if file is available in cloud at the time of the event
				availableInCloud := ListItemsInCloud(client)
				if _, ok := availableInCloud[filename]; !ok {
					go UploadFileToCloud(client, filename, syncInfoChannel)
					log.Info("modified file:", filename)
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

func GetEventFilename(eventName string) string {
	splitPathNameSlice := strings.Split(eventName, "/")
	filename := splitPathNameSlice[len(splitPathNameSlice)-1]
	return filename
}
