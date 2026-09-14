package block

import "github.com/google/uuid"

type Requirement struct {
	ID     string
	Label  string
	Detail string
	Role   Role
}

type Check struct {
	Requirement
	Met     bool
	BlockID *uuid.UUID
}

var contentFloors = map[string][]Requirement{
	"character": {
		{
			ID:     "description",
			Label:  "Description",
			Detail: "Write the description this character is built on.",
			Role:   RoleDescription,
		},
		{
			ID:     "greetings",
			Label:  "Greeting",
			Detail: "Write at least one opening message.",
			Role:   RoleGreetings,
		},
	},
	"lorebook": {
		{
			ID:     "entries",
			Label:  "Entry",
			Detail: "Write at least one entry.",
			Role:   RoleLorebookEntries,
		},
	},
	"preset": {
		{
			ID:     "prompt_fragments",
			Label:  "Prompt fragment",
			Detail: "Write at least one prompt fragment.",
			Role:   RolePromptFragments,
		},
	},
	"theme": {
		{
			ID:     "palette",
			Label:  "Palette",
			Detail: "Add at least one colour to the palette.",
			Role:   RoleThemeTokens,
		},
	},
	"pack": {
		{
			ID:     "items",
			Label:  "Item",
			Detail: "Add at least one item to the pack.",
			Role:   RolePackItems,
		},
	},
	"extension": {
		{
			ID:     "archive",
			Label:  "Archive",
			Detail: "Upload the extension's archive.",
			Role:   RoleExtensionDetails,
		},
	},
}

func ContentFloor(kind string, blocks []Block) []Check {
	requirements := contentFloors[kind]
	checks := make([]Check, 0, len(requirements))
	for _, requirement := range requirements {
		check := Check{Requirement: requirement}
		for _, holder := range blocks {
			for _, element := range holder.Elements {
				if element.Role != requirement.Role {
					continue
				}
				written := element.Content != nil && !element.Content.Empty()
				if check.BlockID == nil || (written && !check.Met) {
					id := holder.ID
					check.BlockID = &id
				}
				check.Met = check.Met || written
			}
		}
		checks = append(checks, check)
	}
	return checks
}

func RequiredRoles(kind string) []Role {
	requirements := contentFloors[kind]
	roles := make([]Role, 0, len(requirements))
	for _, requirement := range requirements {
		roles = append(roles, requirement.Role)
	}
	return roles
}
