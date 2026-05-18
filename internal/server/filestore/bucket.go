package filestore

type Bucket string

const (
	QuarantineBucket Bucket = "quarantine"
	ExtractedBucket  Bucket = "extracted"
	FunctionsBucket  Bucket = "functions"
	LogsBucket       Bucket = "logs"
	// Add more buckets as needed
)

type BucketConfig struct {
	Name          string
	IsPublic      bool
	RetentionDays *int
}

type BucketRegistry struct {
	configs map[Bucket]BucketConfig
}

func NewBucketRegistry() *BucketRegistry {
	return &BucketRegistry{
		configs: map[Bucket]BucketConfig{
			QuarantineBucket: {
				Name:          "oblak-quarantine",
				IsPublic:      false,
				RetentionDays: nil,
			},
			ExtractedBucket: {
				Name:          "oblak-extracted",
				IsPublic:      false,
				RetentionDays: nil,
			},
			FunctionsBucket: {
				Name:          "oblak-functions",
				IsPublic:      false,
				RetentionDays: nil,
			},
			LogsBucket: {
				Name:          "oblak-logs",
				IsPublic:      false,
				RetentionDays: ptr(30),
			},
		},
	}
}

func (r *BucketRegistry) Name(bucket Bucket) string {
	return r.configs[bucket].Name
}

func (r *BucketRegistry) IsPublic(bucket Bucket) bool {
	return r.configs[bucket].IsPublic
}

func (r *BucketRegistry) RetentionDays(bucket Bucket) *int {
	return r.configs[bucket].RetentionDays
}

func ptr(i int) *int {
	return &i
}
