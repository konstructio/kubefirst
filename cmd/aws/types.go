package aws

import (
	"io"

	"github.com/konstructio/kubefirst/internal/common"
)

type Service struct {
	logger common.Logger
	writer io.Writer
}
