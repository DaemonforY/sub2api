package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// s3ArchiveStore implements service.ArchiveObjectStore over an S3-compatible bucket (Tencent COS
// in production: endpoint https://cos.<region>.myqcloud.com, virtual-hosted style).
type s3ArchiveStore struct {
	client *s3.Client
	bucket string
}

var _ service.ArchiveObjectStore = (*s3ArchiveStore)(nil)

// NewS3ArchiveStore builds the archive store (no network call).
func NewS3ArchiveStore(ctx context.Context, cfg service.ArchiveStoreConfig) (service.ArchiveObjectStore, error) {
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("bucket is required")
	}
	client, err := newS3Client(ctx, s3ClientParams{
		Endpoint:        cfg.Endpoint,
		Region:          cfg.Region,
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
		ForcePathStyle:  cfg.ForcePathStyle,
	})
	if err != nil {
		return nil, err
	}
	return &s3ArchiveStore{client: client, bucket: cfg.Bucket}, nil
}

// ProvideArchiveObjectStoreFactory provides the factory the archive service rebuilds its store
// with whenever the settings change.
func ProvideArchiveObjectStoreFactory() service.ArchiveObjectStoreFactory {
	return NewS3ArchiveStore
}

// withContentMD5 adds a Content-MD5 header: COS (like S3 before flexible checksums) requires it on
// DeleteObjects and PutBucketLifecycleConfiguration, while SDK v2 only sends a CRC32 checksum.
func withContentMD5(o *s3.Options) {
	o.APIOptions = append(o.APIOptions, func(stack *smithymiddleware.Stack) error {
		if _, ok := stack.Build.Get("ContentChecksum"); ok {
			return nil
		}
		return smithyhttp.AddContentChecksumMiddleware(stack)
	})
}

func (s *s3ArchiveStore) PutObject(ctx context.Context, key string, body []byte, contentType, contentEncoding string, sse bool) error {
	in := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if contentEncoding != "" {
		in.ContentEncoding = aws.String(contentEncoding)
	}
	if sse {
		in.ServerSideEncryption = s3types.ServerSideEncryptionAes256
	}
	if _, err := s.client.PutObject(ctx, in); err != nil {
		return fmt.Errorf("PutObject %s: %w", key, err)
	}
	return nil
}

func (s *s3ArchiveStore) GetObject(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, 0, fmt.Errorf("GetObject %s: %w", key, err)
	}
	size := int64(-1)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, size, nil
}

func (s *s3ArchiveStore) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if len(keys) > 1000 {
		return errors.New("DeleteObjects takes at most 1000 keys")
	}
	ids := make([]s3types.ObjectIdentifier, 0, len(keys))
	for _, k := range keys {
		ids = append(ids, s3types.ObjectIdentifier{Key: aws.String(k)})
	}
	out, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &s3types.Delete{Objects: ids, Quiet: aws.Bool(true)},
	}, withContentMD5)
	if err != nil {
		return fmt.Errorf("DeleteObjects: %w", err)
	}
	if len(out.Errors) > 0 {
		first := out.Errors[0]
		return fmt.Errorf("DeleteObjects: %d keys failed, first %s: %s %s", len(out.Errors),
			aws.ToString(first.Key), aws.ToString(first.Code), aws.ToString(first.Message))
	}
	return nil
}

func (s *s3ArchiveStore) DeletePrefix(ctx context.Context, prefix string) (int, error) {
	if strings.TrimSpace(prefix) == "" {
		return 0, errors.New("refusing to delete an empty prefix")
	}
	deleted := 0
	var token *string
	for {
		out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
			MaxKeys:           aws.Int32(1000),
		})
		if err != nil {
			return deleted, fmt.Errorf("ListObjectsV2 %s: %w", prefix, err)
		}
		keys := make([]string, 0, len(out.Contents))
		for _, obj := range out.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
		if err := s.DeleteObjects(ctx, keys); err != nil {
			return deleted, err
		}
		deleted += len(keys)
		if !aws.ToBool(out.IsTruncated) || out.NextContinuationToken == nil {
			return deleted, nil
		}
		token = out.NextContinuationToken
	}
}

// ApplyLifecycle keeps the bucket's other rules and adds / replaces ours.
func (s *s3ArchiveStore) ApplyLifecycle(ctx context.Context, ruleID, prefix string, days int) error {
	var rules []s3types.LifecycleRule
	current, err := s.client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		var apiErr smithy.APIError
		if !errors.As(err, &apiErr) || apiErr.ErrorCode() != "NoSuchLifecycleConfiguration" {
			return fmt.Errorf("GetBucketLifecycleConfiguration: %w", err)
		}
	} else {
		for _, rule := range current.Rules {
			if aws.ToString(rule.ID) != ruleID {
				rules = append(rules, rule)
			}
		}
	}
	rules = append(rules, s3types.LifecycleRule{
		ID:         aws.String(ruleID),
		Status:     s3types.ExpirationStatusEnabled,
		Filter:     &s3types.LifecycleRuleFilter{Prefix: aws.String(prefix)},
		Expiration: &s3types.LifecycleExpiration{Days: aws.Int32(int32(days))},
	})
	_, err = s.client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket:                 aws.String(s.bucket),
		LifecycleConfiguration: &s3types.BucketLifecycleConfiguration{Rules: rules},
	}, withContentMD5)
	if err != nil {
		return fmt.Errorf("PutBucketLifecycleConfiguration: %w", err)
	}
	return nil
}
