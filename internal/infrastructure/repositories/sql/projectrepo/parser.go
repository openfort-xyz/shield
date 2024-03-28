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
	}
}

func (p *parser) toDatabase(proj *project.Project) *Project {
	return &Project{
		ID:        proj.ID,
		Name:      proj.Name,
		APIKey:    proj.APIKey,
		APISecret: proj.APISecret,
	}
}

func (p *parser) toDomainAllowedOrigins(origins []AllowedOrigin) []string {
	var result []string
	for _, origin := range origins {
		result = append(result, origin.Origin)
	}
	return result
}
