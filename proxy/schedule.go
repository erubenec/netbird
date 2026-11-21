package proxy

import (
	"github.com/netbirdio/netbird/shared/management/proto"
	"github.com/netbirdio/netbird/shared/schedule"
)

// scheduleFromProto converts a mapping's schedule. It returns nil only when the
// mapping carries none; a malformed schedule is kept so it evaluates as closed.
func scheduleFromProto(p *proto.Schedule) *schedule.Schedule {
	if p == nil {
		return nil
	}
	out := &schedule.Schedule{
		Timezone:  p.GetTimezone(),
		StartDate: p.GetStartDate(),
		EndDate:   p.GetEndDate(),
	}
	for _, w := range p.GetWindows() {
		out.Windows = append(out.Windows, schedule.Window{
			Days:  append([]string(nil), w.GetDays()...),
			Start: w.GetStart(),
			End:   w.GetEnd(),
		})
	}
	return out
}
