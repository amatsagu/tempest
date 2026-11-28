package tempest

import "github.com/amatsagu/tempest/discord"

// Recursively scans components for the first interactive component satisfying filter.
func FindInteractiveComponent[CC discord.AnyComponent, T discord.InteractiveComponent](components []CC, filter func(T) bool) (component T, found bool) {
	for _, cmp := range components {
		switch c := any(cmp).(type) {
		case T:
			if filter(c) {
				return c, true
			}
		case discord.ActionRowComponent:
			for _, icmp := range c.Components {
				if casted, ok := icmp.(T); ok && filter(casted) {
					return casted, true
				}
			}
		case discord.SectionComponent:
			if _, isThumb := c.Accessory.(discord.ThumbnailComponent); isThumb || c.Accessory == nil {
				continue
			}

			if casted, ok := c.Accessory.(T); ok && filter(casted) {
				return casted, true
			}
		case discord.ContainerComponent:
			if found, ok := FindInteractiveComponent(c.Components, filter); ok {
				return found, true
			}
		case discord.LabelComponent:
			if casted, ok := c.Component.(T); ok && filter(casted) {
				return casted, true
			}
		}
	}

	return component, found
}
