package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/charmbracelet/huh/spinner"
	"github.com/fatih/color"
)

const maxConcurrentOperations = 5

type fileOperation struct {
	localPath string
	s3Key     string
	isUpdate  bool
}

// S3Client wraps S3 operations
type S3Client struct {
	client *s3.Client
	bucket string
}

// NewS3Client creates a new S3Client instance
func NewS3Client(ctx context.Context, opts UploadThingOpts) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(opts.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(opts.AccessKeyID, opts.SecretAccessKey, "")),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				if service == s3.ServiceID {
					return aws.Endpoint{URL: opts.Endpoint}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3Client{
		client: s3.NewFromConfig(cfg),
		bucket: opts.Bucket,
	}, nil
}

// Host uploads a file or directory to S3
func (s *S3Client) Host(ctx context.Context, sourcePath, key string, updateExisting bool) error {
	key = strings.ToUpper(key)
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", sourcePath, err)
	}

	if info.IsDir() {
		return s.uploadDirectory(ctx, sourcePath, key, updateExisting)
	}
	return s.uploadSingleFile(ctx, sourcePath, key, updateExisting)
}

func (s *S3Client) uploadDirectory(ctx context.Context, sourcePath, key string, updateExisting bool) error {
	var operations []fileOperation
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		s3Key := key + "/" + filepath.ToSlash(rel)
		if updateExisting {
			update, reason := s.shouldUpdateFile(ctx, s3Key, info.ModTime())
			LogInfo("%s: %s", filepath.Base(path), reason)
			if update {
				operations = append(operations, fileOperation{
					localPath: path,
					s3Key:     s3Key,
					isUpdate:  true,
				})
			}
		} else {
			LogPath("Upload", path, s3Key)
			operations = append(operations, fileOperation{
				localPath: path,
				s3Key:     s3Key,
				isUpdate:  false,
			})
		}
		return nil
	})
	if err != nil {
		return err
	}

	return s.processOperations(ctx, operations, s.uploadFile)
}

func (s *S3Client) uploadSingleFile(ctx context.Context, sourcePath, key string, updateExisting bool) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	var s3Key string
	if updateExisting {
		s3Key = key
		update, reason := s.shouldUpdateFile(ctx, s3Key, info.ModTime())
		LogInfo("%s: %s", filepath.Base(sourcePath), reason)
		if update {
			return s.uploadFile(ctx, sourcePath, s3Key)
		}
	} else {
		s3Key = key + "/" + filepath.Base(sourcePath)
		LogPath("Upload", sourcePath, s3Key)
		return s.uploadFile(ctx, sourcePath, s3Key)
	}
	return nil
}

// Sync downloads S3 objects matching a prefix to a local destination
func (s *S3Client) Sync(ctx context.Context, key, destPath string) error {
	key = strings.ToUpper(key)
	LogPath("Sync", key, destPath)
	out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("list objects: %w", err)
	}
	if len(out.Contents) == 0 {
		return fmt.Errorf("no objects found with prefix %q", key)
	}

	destInfo, _ := os.Stat(destPath)
	if len(out.Contents) == 1 {
		return s.downloadSingleObject(ctx, &out.Contents[0], destPath, destInfo)
	}
	return s.downloadMultipleObjects(ctx, out.Contents, destPath, destInfo)
}

func (s *S3Client) shouldDownloadFile(ctx context.Context, s3Key string, localPath string) (bool, string) {
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return true, color.YellowString("failed to check remote object, downloading")
	}

	localInfo, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, color.YellowString("local file does not exist, downloading")
		}
		return true, color.YellowString("failed to check local file, downloading")
	}

	// Compare sizes
	if localInfo.Size() != *head.ContentLength {
		return true, color.YellowString("file sizes differ, downloading")
	}

	// Compare modification times
	s3ModTime := *head.LastModified
	if s3ModTime.After(localInfo.ModTime()) {
		return true, color.YellowString("remote file is newer, downloading")
	}

	return false, color.GreenString("local file is up-to-date, skipping")
}

func (s *S3Client) downloadMultipleObjects(ctx context.Context, objects []types.Object, destPath string, destInfo os.FileInfo) error {
	if destInfo == nil {
		if err := os.MkdirAll(destPath, 0755); err != nil {
			return fmt.Errorf("create dest directory: %w", err)
		}
	} else if !destInfo.IsDir() {
		return fmt.Errorf("destination %q must be a directory for multiple objects", destPath)
	}

	var operations []fileOperation
	for _, obj := range objects {
		objKey := *obj.Key
		rel := strings.TrimPrefix(objKey, strings.ToUpper(SyncBucketRoot))
		rel = strings.TrimPrefix(rel, "/")
		rel = strings.SplitN(rel, "/", 2)[1]
		localPath := filepath.Join(destPath, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", localPath, err)
		}

		shouldDownload, reason := s.shouldDownloadFile(ctx, objKey, localPath)
		LogInfo("%s: %s", filepath.Base(localPath), reason)

		if shouldDownload {
			LogPath("Download", objKey, localPath)
			operations = append(operations, fileOperation{
				localPath: localPath,
				s3Key:     objKey,
			})
		}
	}

	if len(operations) == 0 {
		LogInfo("All files are up to date")
		return nil
	}

	return s.processOperations(ctx, operations, s.downloadFile)
}

func (s *S3Client) downloadSingleObject(ctx context.Context, obj *types.Object, destPath string, destInfo os.FileInfo) error {
	objKey := *obj.Key
	if destInfo != nil && destInfo.IsDir() {
		destPath = filepath.Join(destPath, filepath.Base(objKey))
	}

	shouldDownload, reason := s.shouldDownloadFile(ctx, objKey, destPath)
	LogInfo("%s: %s", filepath.Base(destPath), reason)

	if !shouldDownload {
		return nil
	}

	LogPath("Download", objKey, destPath)
	return s.downloadFile(ctx, objKey, destPath)
}

func (s *S3Client) shouldUpdateFile(ctx context.Context, s3Key string, localModTime time.Time) (bool, string) {
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return true, color.YellowString("object not found, uploading")
	}
	s3ModTime := *head.LastModified
	if localModTime.After(s3ModTime) {
		return true, color.YellowString("local file is newer, updating")
	}
	return false, color.GreenString("S3 object is up-to-date, skipping")
}

func (s *S3Client) uploadFile(ctx context.Context, localPath, s3Key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer file.Close()

	done := make(chan error, 1)
	cancelableCtx, cancelFunc := context.WithCancel(ctx)
	go func() {
		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s3Key),
			Body:   file,
		})
		cancelFunc()
		done <- err
	}()

	spinner.New().
		Type(GetRandomSpinner()).
		Title(color.CyanString(" Uploading %s...", filepath.Base(localPath))).
		Context(cancelableCtx).
		Run()
	if err := <-done; err != nil {
		return fmt.Errorf("put object %s: %w", s3Key, err)
	}
	return nil
}

func (s *S3Client) downloadFile(ctx context.Context, s3Key, localPath string) error {
	LogInfo("Downloading %s to %s", s3Key, localPath)
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("get object %s: %w", s3Key, err)
	}
	defer out.Body.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", localPath, err)
	}
	defer localFile.Close()

	done := make(chan error, 1)
	cancelableCtx, cancelFunc := context.WithCancel(ctx)
	go func() {
		_, err := io.Copy(localFile, out.Body)
		cancelFunc()
		done <- err
	}()

	spinner.New().
		Type(GetRandomSpinner()).
		Title(color.CyanString(" Downloading %s...", filepath.Base(s3Key))).
		Context(cancelableCtx).
		Run()
	if err := <-done; err != nil {
		return fmt.Errorf("copy to file %s: %w", localPath, err)
	}
	return nil
}

func (s *S3Client) processOperations(ctx context.Context, operations []fileOperation, processFunc func(context.Context, string, string) error) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentOperations)
	errChan := make(chan error, len(operations))

	for _, op := range operations {
		wg.Add(1)
		go func(op fileOperation) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			if err := processFunc(ctx, op.s3Key, op.localPath); err != nil {
				errChan <- fmt.Errorf("failed to process %s: %w", op.localPath, err)
			}
		}(op)
	}

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect any errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errs)
	}
	return nil
}