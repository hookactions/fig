package awsEnv

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/ssmiface"
	"github.com/stretchr/testify/assert"
)

type mockSecretManagerClient struct {
	secretsmanageriface.ClientAPI

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
	ssmiface.ClientAPI

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

func TestFig_GetEnv(t *testing.T) {
	t.Run("NonPrefixedValues", func(t *testing.T) {
		fig := &Fig{}
		ctx := context.Background()

		require.NoError(t, os.Setenv("FOO_1", "bar"))
		require.NoError(t, os.Setenv("FOO_BAR_BAZ", "test"))

		defer os.Unsetenv("FOO_1")
		defer os.Unsetenv("FOO_BAR_BAZ")

		assert.Equal(t, "bar", fig.GetEnv(ctx, "FOO_1"))
		assert.Equal(t, "test", fig.GetEnv(ctx, "FOO_BAR_BAZ"))
	})

	t.Run("SecretsManager", func(t *testing.T) {
		manager := &mockSecretManagerClient{}

		fig := &Fig{
			DecryptParameterStoreValues: true,
			secretsManager:              manager,
		}
		ctx := context.Background()

		t.Run("String", func(t *testing.T) {
			t.Run("Simple", func(t *testing.T) {
				require.NoError(t, os.Setenv("foo", "sm://foo_bar"))
				defer os.Unsetenv("foo")

				manager.checkInput = func(input *secretsmanager.GetSecretValueInput) {
					assert.Equal(t, "foo_bar", *input.SecretId)
				}
				manager.stringValue = aws.String("baz")

				assert.Equal(t, "baz", fig.GetEnv(ctx, "foo"))
			})

			// "complex" in the sense that this would break using strings.TrimPrefix(...)
			t.Run("Complex", func(t *testing.T) {
				require.NoError(t, os.Setenv("foo", "sm://small_foo_bar"))
				defer os.Unsetenv("foo")

				manager.checkInput = func(input *secretsmanager.GetSecretValueInput) {
					assert.Equal(t, "small_foo_bar", *input.SecretId)
				}
				manager.stringValue = aws.String("baz")

				assert.Equal(t, "baz", fig.GetEnv(ctx, "foo"))
			})
		})
	})

	t.Run("ParameterStore", func(t *testing.T) {
		storeClient := &mockParameterStoreClient{}

		fig := &Fig{
			DecryptParameterStoreValues: true,
			parameterStore:              storeClient,
		}
		ctx := context.Background()

		t.Run("String", func(t *testing.T) {
			t.Run("Simple", func(t *testing.T) {
				require.NoError(t, os.Setenv("foo", "ssm://foo_bar"))
				defer os.Unsetenv("foo")

				storeClient.checkInput = func(input *ssm.GetParameterInput) {
					assert.Equal(t, "foo_bar", *input.Name)
					assert.True(t, *input.WithDecryption)
				}
				storeClient.stringValue = aws.String("baz")

				assert.Equal(t, "baz", fig.GetEnv(ctx, "foo"))
			})

			// "complex" in the sense that this would break using strings.TrimPrefix(...)
			t.Run("Complex", func(t *testing.T) {
				require.NoError(t, os.Setenv("foo", "ssm://ssmall_foo_bar"))
				defer os.Unsetenv("foo")

				storeClient.checkInput = func(input *ssm.GetParameterInput) {
					assert.Equal(t, "ssmall_foo_bar", *input.Name)
					assert.True(t, *input.WithDecryption)
				}
				storeClient.stringValue = aws.String("baz")

				assert.Equal(t, "baz", fig.GetEnv(ctx, "foo"))
			})
		})
	})
}
