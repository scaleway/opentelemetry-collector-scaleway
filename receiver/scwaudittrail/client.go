package scwaudittrail

import (
	audit_trail "github.com/scaleway/scaleway-sdk-go/api/audit_trail/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	userAgent = "OpenTelemetry-Collector AuditTrail Receiver"
)

//go:generate mockgen -destination client_mock.go -package scwaudittrail . Client
type Client interface {
	ListEvents(*audit_trail.ListEventsRequest) (*audit_trail.ListEventsResponse, error)
}

type scwClient struct {
	adtAPI *audit_trail.API
}

func newScalewayClient(cfg *Config) (Client, error) {
	client, err := scw.NewClient(
		scw.WithAPIURL(cfg.APIURL),
		scw.WithDefaultOrganizationID(cfg.OrganizationID),
		scw.WithAuth(cfg.AccessKey, cfg.SecretKey),
		scw.WithDefaultRegion(scw.Region(cfg.Region)),
		scw.WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, err
	}

	return &scwClient{
		adtAPI: audit_trail.NewAPI(client),
	}, nil
}

func (c *scwClient) ListEvents(req *audit_trail.ListEventsRequest) (*audit_trail.ListEventsResponse, error) {
	return c.adtAPI.ListEvents(req)
}
