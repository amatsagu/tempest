package tempest

// FindInteractiveComponent recursively scans the components array for the first interactive component that satisfies filter.
//
// Warning - This function can traverse arbitrarily deep nested component trees & uses frequent type assertions, reducing overall performance.
// Prefer checking individual components for interactions with smaller numbers of components.
func FindInteractiveComponent[CC AnyComponent, T InteractiveComponent](components []CC, filter func(T) bool) (T, bool) {
	var zero T

	// TODO: Can this use a type switch?
	for _, cmp := range components {
		switch cmp._kind() {
		case BUTTON_COMPONENT_TYPE,
			STRING_SELECT_COMPONENT_TYPE,
			TEXT_INPUT_COMPONENT_TYPE,
			USER_SELECT_COMPONENT_TYPE,
			ROLE_SELECT_COMPONENT_TYPE,
			MENTIONABLE_SELECT_COMPONENT_TYPE,
			CHANNEL_SELECT_COMPONENT_TYPE:
			if casted, ok := any(cmp).(T); ok && filter(casted) {
				return casted, true
			}
		case ACTION_ROW_COMPONENT_TYPE:
			if row, ok := any(cmp).(ActionRowComponent); ok {
				for _, icmp := range row.Components {
					if casted, ok := icmp.(T); ok && filter(casted) {
						return casted, true
					}
				}
			}
		case SECTION_COMPONENT_TYPE:
			if section, ok := any(cmp).(SectionComponent); ok {
				// Only button is InteractiveComponent for Accessory type.
				if section.Accessory._kind() == THUMBNAIL_COMPONENT_TYPE {
					continue
				}

				if casted, ok := section.Accessory.(T); ok && filter(casted) {
					return casted, true
				}
			}
		case CONTAINER_COMPONENT_TYPE:
			container, ok := any(cmp).(ContainerComponent)
			if !ok {
				continue
			}

			return FindInteractiveComponent(container.Components, filter)
		case LABEL_COMPONENT_TYPE:
			if label, ok := any(cmp).(LabelComponent); ok {
				if casted, ok := any(label.Component).(T); ok && filter(casted) {
					return casted, true
				}
			}
		}

	}

	return zero, false
}
