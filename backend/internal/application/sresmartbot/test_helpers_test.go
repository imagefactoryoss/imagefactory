package sresmartbot

import (
	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
)

type incidentFixture struct {
	Domain       string
	IncidentType string
	DisplayName  string
}

func (f *incidentFixture) toDomain() *domainsresmartbot.Incident {
	return &domainsresmartbot.Incident{
		ID:           uuid.New(),
		Domain:       f.Domain,
		IncidentType: f.IncidentType,
		DisplayName:  f.DisplayName,
	}
}
