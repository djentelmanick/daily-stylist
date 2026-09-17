package s3

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.PhotoStorage = (*PhotoStorage)(nil)

type Config struct {
	Endpoint  string
	PublicURL string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type PhotoStorage struct {
	bucket  string
	client  *awss3.Client
	presign *awss3.PresignClient
}

// Клиента два, и различаются они адресом: в подпись входит имя хоста,
// а браузер приходит снаружи, не на localhost.
func NewPhotoStorage(cfg Config) *PhotoStorage {
	return &PhotoStorage{
		bucket:  cfg.Bucket,
		client:  newClient(cfg, cfg.Endpoint),
		presign: awss3.NewPresignClient(newClient(cfg, cfg.PublicURL)),
	}
}

func newClient(cfg Config, endpoint string) *awss3.Client {
	return awss3.New(awss3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		// Бакет в пути, а не поддоменом: ни у localhost, ни у адреса туннеля поддоменов нет.
		UsePathStyle: true,
		// Иначе SDK добавляет в подпись заголовки с контрольной суммой, которых
		// браузер не пришлёт, и подписанная ссылка перестанет работать.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
}

func (storage *PhotoStorage) UploadLink(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (string, error) {
	request, err := storage.presign.PresignPutObject(ctx, &awss3.PutObjectInput{
		Bucket:        aws.String(storage.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}, awss3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("ссылка на загрузку %q: %w", key, err)
	}
	return request.URL, nil
}

func (storage *PhotoStorage) DownloadLink(ctx context.Context, key string, ttl time.Duration) (string, error) {
	request, err := storage.presign.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(key),
	}, awss3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("ссылка на фотографию %q: %w", key, err)
	}
	return request.URL, nil
}

func (storage *PhotoStorage) Describe(ctx context.Context, key string) (service.PhotoInfo, error) {
	head, err := storage.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(key),
	})
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return service.PhotoInfo{}, service.ErrPhotoNotUploaded
	}
	if err != nil {
		return service.PhotoInfo{}, fmt.Errorf("чтение фотографии %q: %w", key, err)
	}
	return service.PhotoInfo{
		Size:        aws.ToInt64(head.ContentLength),
		ContentType: aws.ToString(head.ContentType),
	}, nil
}

// По одному, а не пачкой: пачками удаляют десятки тысяч, а здесь - фотографии
// нескольких вещей за раз.
func (storage *PhotoStorage) Delete(ctx context.Context, keys []string) error {
	var problems []error
	for _, key := range keys {
		_, err := storage.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
			Bucket: aws.String(storage.bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			problems = append(problems, fmt.Errorf("удаление фотографии %q: %w", key, err))
		}
	}
	return errors.Join(problems...)
}
