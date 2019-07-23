package deployment

import (
	"time"
)

func (m *Event) GetTimestampAsTime() time.Time {
	return time.Unix(m.GetTimestamp().GetSeconds(), int64(m.GetTimestamp().GetNanos()))
}
