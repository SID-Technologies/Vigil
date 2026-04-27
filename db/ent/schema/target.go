package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Target is a user-configurable probe destination. Replaces the hardcoded
// DEFAULT_TARGETS list from the legacy Python tool. On first run, 13 builtin
// targets are seeded (is_builtin=true) covering Google/Cloudflare DNS,
// Teams/Zoom/Outlook over ICMP+TCP, and public STUN servers.
//
// User-added targets have is_builtin=false and can be freely deleted.
// Builtin targets can be disabled (enabled=false) but not deleted, so the
// "Reset to defaults" UI is meaningful.
type Target struct {
	ent.Schema
}

func (Target) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable().
			Comment("UUID set by application code at creation"),
		field.String("label").
			Unique().
			NotEmpty().
			MaxLen(64).
			Comment("Human-readable identifier, e.g. 'router_icmp', 'teams_tcp443'"),
		field.Enum("kind").
			Values("icmp", "tcp", "udp_dns", "udp_stun").
			Comment("Selects the Probe implementation"),
		field.String("host").
			NotEmpty().
			Comment("Hostname or IP address"),
		field.Int("port").
			Optional().
			Nillable().
			Comment("Required for tcp/udp_dns/udp_stun; ignored for icmp"),
		field.Bool("enabled").
			Default(true),
		field.Bool("is_builtin").
			Default(false).
			Immutable().
			Comment("True for the 13 seeded targets shipped with Vigil"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Target) Indexes() []ent.Index {
	return []ent.Index{
		// Hot query: monitor loop fetches enabled targets every cycle.
		index.Fields("enabled", "kind"),
	}
}
