package awsEnv

import (
	"context"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/ssmiface"
	"github.com/pkg/errors"
)

var (
	secretsManagerStringRe = regexp.MustCompile("^sm://")
	parameterStoreStringRe = regexp.MustCompile("^ssm://")
)

func checkPrefixAndStrip(re *regexp.Regexp, s string) (string, bool) {
	if re.MatchString(s) {
		return re.ReplaceAllString(s, ""), true
	}
	return s, false
}

type Fig struct {
	DecryptParameterStoreValues bool

	secretsManager secretsmanageriface.ClientAPI
	parameterStore ssmiface.ClientAPI
}

func New() (*Fig, error) {
	awsConfig, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "fig/awsEnv: error loading default aws config")
	}

	fig := &Fig{
		DecryptParameterStoreValues: true,

		secretsManager: secretsmanager.New(awsConfig),
		parameterStore: ssm.New(awsConfig),
	}

	return fig, nil
}

func (f *Fig) GetEnv(ctx context.Context, key string) string {
	value := os.Getenv(key)
	return f.processConfigItem(ctx, key, value)
}

func (f *Fig) processConfigItem(ctx context.Context, key string, value string) string {
	if v, ok := checkPrefixAndStrip(secretsManagerStringRe, value); ok {
		return f.LoadStringValueFromSecretsManager(ctx, v)
	} else if v, ok := checkPrefixAndStrip(parameterStoreStringRe, v); ok {
		return f.LoadStringValueFromParameterStore(ctx, v, f.DecryptParameterStoreValues)
	}
	return value
}

func (f *Fig) LoadStringValueFromSecretsManager(ctx context.Context, name string) string {
	resp, err := f.requestSecret(ctx, name)
	if err != nil {
		panic("fig/aws/LoadStringValueFromSecretsManager: error loading secret, " + err.Error())
	}

	return *resp.SecretString
}

func (f *Fig) requestSecret(ctx context.Context, name string) (*secretsmanager.GetSecretValueResponse, error) {
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

func (f *Fig) requestParameter(ctx context.Context, name string, decrypt bool) (*ssm.GetParameterResponse, error) {
	input := &ssm.GetParameterInput{Name: aws.String(name), WithDecryption: aws.Bool(decrypt)}
	return f.parameterStore.GetParameterRequest(input).Send(ctx)
}
