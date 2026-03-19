package tabs

import (
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type StorageTab struct {
	Mode           string
	S3Bucket       string
	S3Region       string
	S3Key          string
	S3Secret       string
	AzureConn      string
	AzureContainer string
}

func NewStorageTab(cfg *config.Config) *StorageTab {
	return &StorageTab{
		Mode:           cfg.Storage.Mode,
		S3Bucket:       cfg.Storage.S3Bucket,
		S3Region:       cfg.Storage.S3Region,
		S3Key:          cfg.Storage.S3Key,
		S3Secret:       cfg.Storage.S3Secret,
		AzureConn:      cfg.Storage.AzureConn,
		AzureContainer: cfg.Storage.AzureContainer,
	}
}

func (t *StorageTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Storage Mode").
				Options(
					huh.NewOption("Local", "local"),
					huh.NewOption("S3-compatible", "s3"),
					huh.NewOption("Azure Blob Storage", "azure"),
				).
				Value(&t.Mode),
		),
		huh.NewGroup(
			huh.NewInput().Title("S3 Bucket").Value(&t.S3Bucket),
			huh.NewInput().Title("S3 Region").Value(&t.S3Region),
			huh.NewInput().Title("S3 Access Key").Value(&t.S3Key),
			huh.NewInput().
				Title("S3 Secret Key").
				EchoMode(huh.EchoModePassword).
				Value(&t.S3Secret),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Azure Connection String").
				EchoMode(huh.EchoModePassword).
				Value(&t.AzureConn),
			huh.NewInput().Title("Azure Container").Value(&t.AzureContainer),
		),
	)
}

func (t *StorageTab) Apply(cfg *config.Config) {
	cfg.Storage.Mode = t.Mode
	cfg.Storage.S3Bucket = t.S3Bucket
	cfg.Storage.S3Region = t.S3Region
	cfg.Storage.S3Key = t.S3Key
	cfg.Storage.S3Secret = t.S3Secret
	cfg.Storage.AzureConn = t.AzureConn
	cfg.Storage.AzureContainer = t.AzureContainer
}
