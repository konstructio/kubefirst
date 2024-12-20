package aws

import (
	"io"

	"github.com/konstructio/kubefirst/internal/common"
)

type AwsService struct {
	logger common.Logger
	writer io.Writer
}
