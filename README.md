# fig
![header](docs/header.png)
[![CircleCI](https://circleci.com/gh/hookactions/fig.svg?style=svg)](https://circleci.com/gh/hookactions/fig)

# Usage

## AWS

Pre-process config values by reading secrets from AWS.

```go
// main.go
package main

import (
	"context"
	"fmt"

	figAws "github.com/hookactions/fig/aws"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	fig, err := figAws.New(nil)
	if err != nil {
		panic(err)
	}

	fig.PreProcessConfigItems(context.Background())

	value := viper.GetString("my_var")
	fmt.Println(value)
}
```

```bash
echo "my_var: sm://foo" >> config.yaml
go run main.go
```

### Supported prefixes
- `sm://` – Get string value from [secrets manager](https://aws.amazon.com/secrets-manager/)
- `smb://` – Get binary value from [secrets manager](https://aws.amazon.com/secrets-manager/)
- `ssm://` – Get string value from [parameter store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html)
  - _Note_: decryption of the value is automatically requested.
- `ssmb64://` – Get base64 encoded binary value from [parameter store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html)
  - _Note_: decryption of the value is automatically requested.
