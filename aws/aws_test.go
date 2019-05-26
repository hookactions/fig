package aws

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/aws/aws-sdk-go-v2/service/ssm/ssmiface"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/viper"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/secretsmanageriface"
	"github.com/stretchr/testify/assert"
)

type mockSecretManagerClient struct {
	secretsmanageriface.SecretsManagerAPI

	checkInput  func(*secretsmanager.GetSecretValueInput)
	stringValue *string
	binaryValue []byte
}

func (m *mockSecretManagerClient) GetSecretValueRequest(in *secretsmanager.GetSecretValueInput) secretsmanager.GetSecretValueRequest {
	if m.checkInput != nil {
		m.checkInput(in)
	}

	req := &aws.Request{
		Data: &secretsmanager.GetSecretValueOutput{
			SecretString: m.stringValue,
			SecretBinary: m.binaryValue,
		},
		HTTPRequest: new(http.Request),
	}
	return secretsmanager.GetSecretValueRequest{Request: req, Input: in, Copy: m.GetSecretValueRequest}
}

type mockParameterStoreClient struct {
	ssmiface.SSMAPI

	checkInput  func(*ssm.GetParameterInput)
	stringValue *string
	binaryValue []byte
}

func (m *mockParameterStoreClient) GetParameterRequest(in *ssm.GetParameterInput) ssm.GetParameterRequest {
	if m.checkInput != nil {
		m.checkInput(in)
	}

	var value *string

	if m.stringValue != nil {
		value = m.stringValue
	} else if m.binaryValue != nil {
		value = aws.String(base64.StdEncoding.EncodeToString(m.binaryValue))
	}

	req := &aws.Request{
		Data: &ssm.GetParameterOutput{
			Parameter: &ssm.Parameter{
				Value: value,
			},
		},
		HTTPRequest: new(http.Request),
	}
	return ssm.GetParameterRequest{Request: req, Input: in, Copy: m.GetParameterRequest}
}

func TestFig_PreProcessConfigItems(t *testing.T) {
	t.Run("SecretsManager", func(t *testing.T) {
		manager := &mockSecretManagerClient{}

		fig := &Fig{
			DecryptParameterStoreValues: true,
			secretsManager:              manager,
		}

		t.Run("String", func(t *testing.T) {
			t.Run("Simple", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "sm://foo_bar")
				fig.viper = v

				manager.checkInput = func(input *secretsmanager.GetSecretValueInput) {
					assert.Equal(t, "foo_bar", *input.SecretId)
				}
				manager.stringValue = aws.String("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, "baz", v.Get("foo"))
			})

			// "complex" in the sense that this would break using strings.TrimPrefix(...)
			t.Run("Complex", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "sm://small_foo_bar")
				fig.viper = v

				manager.checkInput = func(input *secretsmanager.GetSecretValueInput) {
					assert.Equal(t, "small_foo_bar", *input.SecretId)
				}
				manager.stringValue = aws.String("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, "baz", v.Get("foo"))
			})
		})

		t.Run("Binary", func(t *testing.T) {
			t.Run("Simple", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "smb://foo_bar")
				fig.viper = v

				manager.checkInput = func(input *secretsmanager.GetSecretValueInput) {
					assert.Equal(t, "foo_bar", *input.SecretId)
				}
				manager.binaryValue = []byte("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, []byte("baz"), v.Get("foo"))
			})

			// "complex" in the sense that this would break using strings.TrimPrefix(...)
			t.Run("Complex", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "smb://smball_foo_bar")
				fig.viper = v

				manager.checkInput = func(input *secretsmanager.GetSecretValueInput) {
					assert.Equal(t, "smball_foo_bar", *input.SecretId)
				}
				manager.binaryValue = []byte("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, []byte("baz"), v.Get("foo"))
			})
		})
	})

	t.Run("ParameterStore", func(t *testing.T) {
		storeClient := &mockParameterStoreClient{}

		fig := &Fig{
			DecryptParameterStoreValues: true,
			parameterStore:              storeClient,
		}

		t.Run("String", func(t *testing.T) {
			t.Run("Simple", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "ssm://foo_bar")
				fig.viper = v

				storeClient.checkInput = func(input *ssm.GetParameterInput) {
					assert.Equal(t, "foo_bar", *input.Name)
					assert.True(t, *input.WithDecryption)
				}
				storeClient.stringValue = aws.String("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, "baz", v.Get("foo"))
			})

			// "complex" in the sense that this would break using strings.TrimPrefix(...)
			t.Run("Complex", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "ssm://ssmall_foo_bar")
				fig.viper = v

				storeClient.checkInput = func(input *ssm.GetParameterInput) {
					assert.Equal(t, "ssmall_foo_bar", *input.Name)
					assert.True(t, *input.WithDecryption)
				}
				storeClient.stringValue = aws.String("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, "baz", v.Get("foo"))
			})
		})

		storeClient.stringValue = nil

		t.Run("Binary", func(t *testing.T) {
			t.Run("Simple", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "ssmb64://foo_bar")
				fig.viper = v

				storeClient.checkInput = func(input *ssm.GetParameterInput) {
					assert.Equal(t, "foo_bar", *input.Name)
					assert.True(t, *input.WithDecryption)
				}
				storeClient.binaryValue = []byte("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, []byte("baz"), v.Get("foo"))
			})

			// "complex" in the sense that this would break using strings.TrimPrefix(...)
			t.Run("Complex", func(t *testing.T) {
				v := viper.New()
				v.Set("foo", "ssmb64://ssmb64_foo_bar")
				fig.viper = v

				storeClient.checkInput = func(input *ssm.GetParameterInput) {
					assert.Equal(t, "ssmb64_foo_bar", *input.Name)
					assert.True(t, *input.WithDecryption)
				}
				storeClient.binaryValue = []byte("baz")

				fig.PreProcessConfigItems(context.Background())

				assert.Equal(t, []byte("baz"), v.Get("foo"))
			})
		})
	})

	t.Run("Nested", func(t *testing.T) {
		manager := &mockSecretManagerClient{}

		fig := &Fig{
			DecryptParameterStoreValues: true,
			secretsManager:              manager,
		}

		v := viper.New()
		v.Set("foo.bar.baz", "sm://foo_bar")
		fig.viper = v
		manager.stringValue = aws.String("baz")

		fig.PreProcessConfigItems(context.Background())

		assert.Equal(t, "baz", v.Get("foo.bar.baz"))
	})
}
