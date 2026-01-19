package projectrepo

import "go.openfort.xyz/shield/internal/core/domain/project"

type parser struct {
}

func newParser() *parser {
	return &parser{}
}

func (p *parser) toDomain(proj *Project) *project.Project {
	return &project.Project{
		ID:        proj.ID,
		Name:      proj.Name,
		APIKey:    proj.APIKey,
		APISecret: proj.APISecret,
		Enable2FA: proj.Enable2FA,
	}
}

func (p *parser) toDomainWithRateLimit(proj *ProjectWithRateLimit) *project.WithRateLimit {
	return &project.WithRateLimit{
		ID:             proj.ID,
		Name:           proj.Name,
		APIKey:         proj.APIKey,
		APISecret:      proj.APISecret,
		Enable2FA:      proj.Enable2FA,
		SMSRateLimit:   proj.SMSRequestsPerHour,
		EmailRateLimit: proj.EmailRequestsPerHour,
	}
}

func (p *parser) toDatabase(proj *project.Project) *Project {
	return &Project{
		ID:        proj.ID,
		Name:      proj.Name,
		APIKey:    proj.APIKey,
		APISecret: proj.APISecret,
		Enable2FA: proj.Enable2FA,
	}
}

func (p *parser) toDatabaseRateLimits(rateLimits *project.RateLimit) *RateLimit {
	return &RateLimit{
		ProjectID:            rateLimits.ProjectID,
		SMSRequestsPerHour:   rateLimits.SMSRequestsPerHour,
		EmailRequestsPerHour: rateLimits.EmailRequestsPerHour,
	}
}
