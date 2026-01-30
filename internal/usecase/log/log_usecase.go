package log

import (
	"github.com/arsnazarenko/log-collector/internal/repo"
	"github.com/arsnazarenko/log-collector/internal/usecase"
)

var _ usecase.LogUsecase = (*LogUsecaseImpl)(nil)

type LogUsecaseImpl struct {
	logRepo repo.LogRepo
}
