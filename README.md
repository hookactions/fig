# fig
<p align="center">
  <img src="docs/header.png" />
</p>

# Usage

## AWS

Automatically pre-process config values, by reading secrets from AWS.

```go
// main.go
package main

import (
	"fmt"
	
	_ "github.com/hookactions/fig/aws"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	value := viper.GetString("my_var")
	fmt.Println(value)
}
```

```bash
MY_VAR=sm://my_var go run main.go
```

### Supported prefixes
- `sm://` – Get string value from [secrets manager](https://aws.amazon.com/secrets-manager/)
- `smb://` – Get binary value from [secrets manager](https://aws.amazon.com/secrets-manager/)
- `ssm://` – Get string value from [parameter store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html)
  - _Note_: decryption of the value is automatically requested.
- `ssmb64://` – Get base64 encoded binary value from [parameter store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html)
  - _Note_: decryption of the value is automatically requested.
