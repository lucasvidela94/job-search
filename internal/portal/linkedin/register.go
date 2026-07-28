package linkedin

import "github.com/lucasvidela94/jobsearch/internal/portal"

func init() {
	portal.Register(New())
}
