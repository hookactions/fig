package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/spf13/viper"
)

var manager *secretsmanager.SecretsManager
var paramStore *ssm.SSM

const (
	secretsManagerStringPrefix = "sm://"
	secretsManagerBinaryPrefix = "smb://"

	parameterStoreStringPrefix = "ssm://"
	parameterStoreBinaryPrefix = "ssmb64://"
)

func init() {
	PreprocessConfig()
}

func PreprocessConfig() {
	awsConfig, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("fig/aws: error loading default aws config, " + err.Error())
	}

	manager = secretsmanager.New(awsConfig)
	paramStore = ssm.New(awsConfig)

	ctx := context.Background()

	for k, v := range viper.AllSettings() {
		preprocessConfigItem(ctx, k, v)
	}
}

func preprocessConfigItem(ctx context.Context, key string, value interface{}) {
	if v, ok := value.(map[string]interface{}); ok {
		for k, v := range v {
			preprocessConfigItem(ctx, fmt.Sprintf("%s.%s", key, k), v)
		}
	} else if v, ok := value.(string); ok {
		if strings.HasPrefix(v, secretsManagerStringPrefix) {
			newValue := loadStringValueFromSecretsManager(ctx, v)
			viper.Set(key, newValue)
		} else if strings.HasPrefix(v, secretsManagerBinaryPrefix) {
			newValue := loadBinaryValueFromSecretsManager(ctx, v)
			viper.Set(key, newValue)
		} else if strings.HasPrefix(v, parameterStoreStringPrefix) {
			newValue := loadStringValueFromParameterStore(ctx, v, true)
			viper.Set(key, newValue)
		} else if strings.HasPrefix(v, parameterStoreBinaryPrefix) {
			newValue := loadBinaryValueFromParameterStore(ctx, v, true)
			viper.Set(key, newValue)
		}
	}
}

func loadStringValueFromSecretsManager(ctx context.Context, name string) string {
	resp, err := requestSecret(ctx, name)
	if err != nil {
		panic("fig/aws/loadStringValueFromSecretsManager: error loading secret, " + err.Error())
	}

	return *resp.SecretString
}

func loadBinaryValueFromSecretsManager(ctx context.Context, name string) []byte {
	resp, err := requestSecret(ctx, name)
	if err != nil {
		panic("fig/aws/loadBinaryValueFromSecretsManager: error loading secret, " + err.Error())
	}

	return resp.SecretBinary
}

func requestSecret(ctx context.Context, name string) (*secretsmanager.GetSecretValueOutput, error) {
	input := &secretsmanager.GetSecretValueInput{SecretId: aws.String(name)}
	return manager.GetSecretValueRequest(input).Send(ctx)
}

func loadStringValueFromParameterStore(ctx context.Context, name string, decrypt bool) string {
	resp, err := requestParameter(ctx, name, decrypt)
	if err != nil {
		panic("fig/aws/loadStringValueFromParameterStore: error loading value, " + err.Error())
	}

	return *resp.Parameter.Value
}

func loadBinaryValueFromParameterStore(ctx context.Context, name string, decrypt bool) []byte {
	resp, err := requestParameter(ctx, name, decrypt)
	if err != nil {
		panic("fig/aws/loadBinaryValueFromParameterStore: error loading value, " + err.Error())
	}

	data, err := base64.StdEncoding.DecodeString(*resp.Parameter.Value)
	if err != nil {
		panic("fig/aws/loadBinaryValueFromParameterStore: error decoding binary value, " + err.Error())
	}

	return data
}

func requestParameter(ctx context.Context, name string, decrypt bool) (*ssm.GetParameterOutput, error) {
	input := &ssm.GetParameterInput{Name: aws.String(name), WithDecryption: aws.Bool(decrypt)}
	return paramStore.GetParameterRequest(input).Send(ctx)
}
