package auto

import (
	"context"

	"github.com/hookactions/fig/aws"
)

func init() {
	fig, err := aws.New(nil)

	if err != nil {
		panic("fig/aws: unable to create new fig, " + err.Error())
	}

	fig.PreProcessConfigItems(context.Background())
}
