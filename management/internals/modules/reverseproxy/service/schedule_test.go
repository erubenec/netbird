package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/management/internals/modules/reverseproxy/proxy"
	"github.com/netbirdio/netbird/shared/management/http/api"
	"github.com/netbirdio/netbird/shared/schedule"
)

func officeSchedule() *schedule.Schedule {
	return &schedule.Schedule{
		Timezone:  "Europe/Madrid",
		StartDate: "2026-12-01",
		EndDate:   "2026-12-31",
		Windows: []schedule.Window{
			{Days: []string{"mon", "tue", "wed", "thu", "fri"}, Start: "08:00", End: "17:00"},
			{Days: []string{"sat"}, Start: "10:00", End: "14:00"},
		},
	}
}

func TestSchedule_APIRoundTrip(t *testing.T) {
	svc := validProxy()
	svc.Schedule = officeSchedule()

	resp := svc.ToAPIResponse()
	require.NotNil(t, resp.Schedule)

	req := &api.ServiceRequest{
		Name:     svc.Name,
		Domain:   svc.Domain,
		Enabled:  true,
		Targets:  &resp.Targets,
		Schedule: resp.Schedule,
	}
	restored := &Service{}
	require.NoError(t, restored.FromAPIRequest(req, "acc-1"))

	assert.Equal(t, svc.Schedule, restored.Schedule)
}

func TestSchedule_FromAPIRequest_AbsentClearsSchedule(t *testing.T) {
	svc := validProxy()
	svc.Schedule = officeSchedule()

	req := &api.ServiceRequest{Name: svc.Name, Domain: svc.Domain, Enabled: true}
	require.NoError(t, svc.FromAPIRequest(req, "acc-1"))

	assert.Nil(t, svc.Schedule, "a PUT without a schedule must remove it")
}

func TestSchedule_ToProtoMapping(t *testing.T) {
	svc := validProxy()
	svc.Schedule = officeSchedule()

	mapping := svc.ToProtoMapping(Create, "token", proxy.OIDCValidationConfig{})

	require.NotNil(t, mapping.Schedule)
	assert.Equal(t, "Europe/Madrid", mapping.Schedule.Timezone)
	assert.Equal(t, "2026-12-01", mapping.Schedule.StartDate)
	assert.Equal(t, "2026-12-31", mapping.Schedule.EndDate)
	require.Len(t, mapping.Schedule.Windows, 2)
	assert.Equal(t, []string{"mon", "tue", "wed", "thu", "fri"}, mapping.Schedule.Windows[0].Days)
	assert.Equal(t, "08:00", mapping.Schedule.Windows[0].Start)
	assert.Equal(t, "17:00", mapping.Schedule.Windows[0].End)
}

func TestSchedule_ToProtoMapping_NoneByDefault(t *testing.T) {
	mapping := validProxy().ToProtoMapping(Create, "token", proxy.OIDCValidationConfig{})
	assert.Nil(t, mapping.Schedule)
}

func TestSchedule_Validate(t *testing.T) {
	t.Run("valid HTTP service", func(t *testing.T) {
		svc := validProxy()
		svc.Schedule = officeSchedule()
		assert.NoError(t, svc.Validate())
	})
	t.Run("invalid timezone", func(t *testing.T) {
		svc := validProxy()
		svc.Schedule = &schedule.Schedule{Timezone: "Mars/Base"}
		assert.ErrorContains(t, svc.Validate(), "timezone")
	})
	t.Run("not supported for L4", func(t *testing.T) {
		svc := validProxy()
		svc.Mode = ModeTCP
		svc.ListenPort = 9000
		svc.Schedule = officeSchedule()
		assert.ErrorContains(t, svc.Validate(), "only supported for HTTP")
	})
}

func TestSchedule_Copy_IsIndependent(t *testing.T) {
	svc := validProxy()
	svc.Schedule = officeSchedule()

	cp := svc.Copy()
	cp.Schedule.Windows[0].Days[0] = "sun"

	assert.Equal(t, "mon", svc.Schedule.Windows[0].Days[0])
}
