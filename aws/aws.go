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
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	secretsManagerStringPrefix = "sm://"
	secretsManagerBinaryPrefix = "smb://"

	parameterStoreStringPrefix = "ssm://"
	parameterStoreBinaryPrefix = "ssmb64://"
)

type Fig struct {
	DecryptParameterStoreValues bool

	secretsManager *secretsmanager.SecretsManager
	parameterStore *ssm.SSM
	viper          *viper.Viper
}

func New(v *viper.Viper) (*Fig, error) {
	if v == nil {
		v = viper.GetViper()
	}

	awsConfig, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "fig/aws: error loading default aws config")
	}

	return &Fig{
		DecryptParameterStoreValues: true,
		secretsManager:              secretsmanager.New(awsConfig),
		parameterStore:              ssm.New(awsConfig),
		viper:                       v,
	}, nil
}

func (f *Fig) PreProcessConfigItems(ctx context.Context) {
	for k, v := range f.viper.AllSettings() {
		f.preProcessConfigItem(ctx, k, v)
	}
}

func (f *Fig) preProcessConfigItem(ctx context.Context, key string, value interface{}) {
	if v, ok := value.(map[string]interface{}); ok {
		for k, v := range v {
			f.preProcessConfigItem(ctx, fmt.Sprintf("%s.%s", key, k), v)
		}
	} else if v, ok := value.(string); ok {
		if strings.HasPrefix(v, secretsManagerStringPrefix) {
			newValue := f.loadStringValueFromSecretsManager(ctx, v)
			f.viper.Set(key, newValue)
		} else if strings.HasPrefix(v, secretsManagerBinaryPrefix) {
			newValue := f.loadBinaryValueFromSecretsManager(ctx, v)
			f.viper.Set(key, newValue)
		} else if strings.HasPrefix(v, parameterStoreStringPrefix) {
			newValue := f.loadStringValueFromParameterStore(ctx, v, f.DecryptParameterStoreValues)
			f.viper.Set(key, newValue)
		} else if strings.HasPrefix(v, parameterStoreBinaryPrefix) {
			newValue := f.loadBinaryValueFromParameterStore(ctx, v, f.DecryptParameterStoreValues)
			f.viper.Set(key, newValue)
		}
	}
}

func (f *Fig) loadStringValueFromSecretsManager(ctx context.Context, name string) string {
	resp, err := f.requestSecret(ctx, name)
	if err != nil {
		panic("fig/aws/loadStringValueFromSecretsManager: error loading secret, " + err.Error())
	}

	return *resp.SecretString
}

func (f *Fig) loadBinaryValueFromSecretsManager(ctx context.Context, name string) []byte {
	resp, err := f.requestSecret(ctx, name)
	if err != nil {
		panic("fig/aws/loadBinaryValueFromSecretsManager: error loading secret, " + err.Error())
	}

	return resp.SecretBinary
}

func (f *Fig) requestSecret(ctx context.Context, name string) (*secretsmanager.GetSecretValueOutput, error) {
	input := &secretsmanager.GetSecretValueInput{SecretId: aws.String(name)}
	return f.secretsManager.GetSecretValueRequest(input).Send(ctx)
}

func (f *Fig) loadStringValueFromParameterStore(ctx context.Context, name string, decrypt bool) string {
	resp, err := f.requestParameter(ctx, name, decrypt)
	if err != nil {
		panic("fig/aws/loadStringValueFromParameterStore: error loading value, " + err.Error())
	}

	return *resp.Parameter.Value
}

func (f *Fig) loadBinaryValueFromParameterStore(ctx context.Context, name string, decrypt bool) []byte {
	resp, err := f.requestParameter(ctx, name, decrypt)
	if err != nil {
		panic("fig/aws/loadBinaryValueFromParameterStore: error loading value, " + err.Error())
	}

	data, err := base64.StdEncoding.DecodeString(*resp.Parameter.Value)
	if err != nil {
		panic("fig/aws/loadBinaryValueFromParameterStore: error decoding binary value, " + err.Error())
	}

	return data
}

func (f *Fig) requestParameter(ctx context.Context, name string, decrypt bool) (*ssm.GetParameterOutput, error) {
	input := &ssm.GetParameterInput{Name: aws.String(name), WithDecryption: aws.Bool(decrypt)}
	return f.parameterStore.GetParameterRequest(input).Send(ctx)
}
