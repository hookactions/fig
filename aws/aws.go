package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go-v2/service/ssm/ssmiface"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var (
	secretsManagerStringRe = regexp.MustCompile("^sm://")
	secretsManagerBinaryRe = regexp.MustCompile("^smb://")

	parameterStoreStringRe = regexp.MustCompile("^ssm://")
	parameterStoreBinaryRe = regexp.MustCompile("^ssmb64://")
)

func checkPrefixAndStrip(re *regexp.Regexp, s string) (string, bool) {
	if re.MatchString(s) {
		return re.ReplaceAllString(s, ""), true
	}
	return s, false
}

type Fig struct {
	DecryptParameterStoreValues bool

	secretsManager secretsmanageriface.SecretsManagerAPI
	parameterStore ssmiface.SSMAPI
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

	fig := &Fig{
		DecryptParameterStoreValues: true,

		secretsManager: secretsmanager.New(awsConfig),
		parameterStore: ssm.New(awsConfig),
		viper:          v,
	}

	return fig, nil
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
		if v, ok := checkPrefixAndStrip(secretsManagerStringRe, v); ok {
			newValue := f.LoadStringValueFromSecretsManager(ctx, v)
			f.viper.Set(key, newValue)
		} else if v, ok := checkPrefixAndStrip(secretsManagerBinaryRe, v); ok {
			newValue := f.LoadBinaryValueFromSecretsManager(ctx, v)
			f.viper.Set(key, newValue)
		} else if v, ok := checkPrefixAndStrip(parameterStoreStringRe, v); ok {
			newValue := f.LoadStringValueFromParameterStore(ctx, v, f.DecryptParameterStoreValues)
			f.viper.Set(key, newValue)
		} else if v, ok := checkPrefixAndStrip(parameterStoreBinaryRe, v); ok {
			newValue := f.LoadBinaryValueFromParameterStore(ctx, v, f.DecryptParameterStoreValues)
			f.viper.Set(key, newValue)
		}
	}
}

func (f *Fig) LoadStringValueFromSecretsManager(ctx context.Context, name string) string {
	resp, err := f.requestSecret(ctx, name)
	if err != nil {
		panic("fig/aws/LoadStringValueFromSecretsManager: error loading secret, " + err.Error())
	}

	return *resp.SecretString
}

func (f *Fig) LoadBinaryValueFromSecretsManager(ctx context.Context, name string) []byte {
	resp, err := f.requestSecret(ctx, name)
	if err != nil {
		panic("fig/aws/LoadBinaryValueFromSecretsManager: error loading secret, " + err.Error())
	}

	return resp.SecretBinary
}

func (f *Fig) requestSecret(ctx context.Context, name string) (*secretsmanager.GetSecretValueOutput, error) {
	input := &secretsmanager.GetSecretValueInput{SecretId: aws.String(name)}
	return f.secretsManager.GetSecretValueRequest(input).Send(ctx)
}

func (f *Fig) LoadStringValueFromParameterStore(ctx context.Context, name string, decrypt bool) string {
	resp, err := f.requestParameter(ctx, name, decrypt)
	if err != nil {
		panic("fig/aws/LoadStringValueFromParameterStore: error loading value, " + err.Error())
	}

	return *resp.Parameter.Value
}

func (f *Fig) LoadBinaryValueFromParameterStore(ctx context.Context, name string, decrypt bool) []byte {
	resp, err := f.requestParameter(ctx, name, decrypt)
	if err != nil {
		panic("fig/aws/LoadBinaryValueFromParameterStore: error loading value, " + err.Error())
	}

	data, err := base64.StdEncoding.DecodeString(*resp.Parameter.Value)
	if err != nil {
		panic("fig/aws/LoadBinaryValueFromParameterStore: error decoding binary value, " + err.Error())
	}

	return data
}

func (f *Fig) requestParameter(ctx context.Context, name string, decrypt bool) (*ssm.GetParameterOutput, error) {
	input := &ssm.GetParameterInput{Name: aws.String(name), WithDecryption: aws.Bool(decrypt)}
	return f.parameterStore.GetParameterRequest(input).Send(ctx)
}
