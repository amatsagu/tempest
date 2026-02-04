package tempest

// FindInteractiveComponent recursively scans the components array for the first interactive component that satisfies filter.
//
// Warning - This function can traverse arbitrarily deep nested component trees & uses frequent type assertions, reducing overall performance.
// Prefer checking individual components for interactions with smaller numbers of components.
func FindInteractiveComponent[CC AnyComponent, T InteractiveComponent](components []CC, filter func(T) bool) (component T, found bool) {
	for _, cmp := range components {
		switch c := any(cmp).(type) {
		case T:
			if filter(c) {
				return c, true
			}
		case ActionRowComponent:
			for _, icmp := range c.Components {
				if casted, ok := icmp.(T); ok && filter(casted) {
					return casted, true
				}
			}
		case SectionComponent:
			// Only valid interactive component inside section accessories are buttons
			if c.Accessory._kind() == THUMBNAIL_COMPONENT_TYPE {
				continue
			}

			if casted, ok := c.Accessory.(T); ok && filter(casted) {
				return casted, true
			}
		case ContainerComponent:
			return FindInteractiveComponent(c.Components, filter)
		case LabelComponent:
			if casted, ok := c.Component.(T); ok && filter(casted) {
				return casted, true
			}
		}
	}

	return component, found
}
