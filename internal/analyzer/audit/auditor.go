package audit

import "context"

type DependencyAuditor interface {
	Audit(ctx context.Context, dirPath string) error
}
