package filestore

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/policy"
	"github.com/minio/minio-go/v7/pkg/set"
)

type BucketInitializer struct {
	client         *minio.Client
	bucketRegistry *BucketRegistry
}

func NewBucketInitializer(client *minio.Client, registry *BucketRegistry) *BucketInitializer {
	return &BucketInitializer{
		client:         client,
		bucketRegistry: registry,
	}
}

func (b *BucketInitializer) InitializeBuckets(ctx context.Context) error {
	// Iterate through all bucket types
	bucketTypes := []Bucket{FunctionsBucket, LogsBucket, QuarantineBucket, DrivesBucket}

	for _, bucketType := range bucketTypes {
		bucketName := b.bucketRegistry.Name(bucketType)

		// Create bucket if it doesn't exist
		exists, err := b.client.BucketExists(ctx, bucketName)
		if err != nil {
			return fmt.Errorf("checking bucket %s: %w", bucketName, err)
		}

		if !exists {
			err = b.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
			if err != nil {
				return fmt.Errorf("creating bucket %s: %w", bucketName, err)
			}
			log.Printf("Bucket '%s' created", bucketName)
		}

		// Configure public read access
		if b.bucketRegistry.IsPublic(bucketType) {
			err = b.setPublicReadPolicy(ctx, bucketName)
			if err != nil {
				return fmt.Errorf("setting public policy for %s: %w", bucketName, err)
			}
		} else {
			err = b.deletePublicReadPolicy(ctx, bucketName)
			if err != nil {
				// Log but don't fail if policy doesn't exist
				log.Printf("Warning: could not remove public policy from %s: %v", bucketName, err)
			}
		}

		// Configure lifecycle expiration
		retentionDays := b.bucketRegistry.RetentionDays(bucketType)
		if retentionDays != nil {
			err = b.setExpirationPolicy(ctx, bucketName, *retentionDays)
			if err != nil {
				return fmt.Errorf("setting expiration for %s: %w", bucketName, err)
			}
		} else {
			err = b.deleteExpirationPolicy(ctx, bucketName)
			if err != nil {
				log.Printf("Warning: could not remove expiration policy from %s: %v", bucketName, err)
			}
		}
	}

	return nil
}

func (b *BucketInitializer) setPublicReadPolicy(ctx context.Context, bucketName string) error {
	accessPolicy := policy.BucketAccessPolicy{
		Version: "2012-10-17",
		Statements: []policy.Statement{
			{
				Effect:    "Allow",
				Principal: policy.User{AWS: set.CreateStringSet("*")},
				Actions:   set.CreateStringSet("s3:GetObject"),
				Resources: set.CreateStringSet(fmt.Sprintf("arn:aws:s3:::%s/*", bucketName)),
			},
		},
	}

	policyBytes, err := json.Marshal(accessPolicy)
	if err != nil {
		return err
	}

	err = b.client.SetBucketPolicy(ctx, bucketName, string(policyBytes))
	if err != nil {
		return err
	}

	log.Printf("Public read policy set for bucket '%s'", bucketName)
	return nil
}

func (b *BucketInitializer) deletePublicReadPolicy(ctx context.Context, bucketName string) error {
	err := b.client.SetBucketPolicy(ctx, bucketName, "")
	if err != nil {
		return err
	}

	log.Printf("Public read policy removed for bucket '%s'", bucketName)
	return nil
}

func (b *BucketInitializer) setExpirationPolicy(ctx context.Context, bucketName string, days int) error {
	config := lifecycle.NewConfiguration()

	rule := lifecycle.Rule{
		ID:     fmt.Sprintf("expire-after-%d-days", days),
		Status: "Enabled",
		Expiration: lifecycle.Expiration{
			Days: lifecycle.ExpirationDays(days),
		},
		RuleFilter: lifecycle.Filter{
			Prefix: "",
		},
	}

	config.Rules = append(config.Rules, rule)

	// Set the lifecycle configuration
	err := b.client.SetBucketLifecycle(ctx, bucketName, config)
	if err != nil {
		return err
	}

	log.Printf("Lifecycle expiration set to %d day(s) for bucket '%s'", days, bucketName)
	return nil
}

func (b *BucketInitializer) deleteExpirationPolicy(ctx context.Context, bucketName string) error {
	err := b.client.SetBucketLifecycle(ctx, bucketName, nil)
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchLifecycleConfiguration" {
			log.Printf("No lifecycle configuration to delete for bucket '%s'", bucketName)
			return nil
		}
		return err
	}

	log.Printf("Lifecycle expiration removed for bucket '%s'", bucketName)
	return nil
}
