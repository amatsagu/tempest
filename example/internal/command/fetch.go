package command

import (
	tempest "github.com/amatsagu/tempest"
)

var Fetch tempest.Command = tempest.Command{
	Name:        "fetch",
	Description: "Group command - on it's own it does nothing.",
}
