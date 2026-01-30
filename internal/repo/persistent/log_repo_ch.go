package persistent

import "github.com/arsnazarenko/log-collector/internal/repo"

var _ repo.LogRepo = (*LogClickhouseRepo)(nil)

type LogClickhouseRepo struct{}
